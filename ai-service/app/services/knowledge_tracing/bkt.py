"""
Bayesian Knowledge Tracing (EG-GCKT Milestone 2, spec section 7.2).

Standard 4-parameter model per skill (topic): p_init (P(L0), prior
probability of already knowing the skill), p_transit (P(T), probability of
learning it after one practice opportunity), p_slip (P(S), probability of
an incorrect response despite knowing the skill), p_guess (P(G),
probability of a correct response despite not knowing it).

Cold start (spec section 15): this codebase has zero real student
interaction history, so there is nothing to fit p_slip/p_guess/p_transit
against yet. Runs entirely on population-default parameters (standard
literature defaults, not empirically fit) until a real nightly refit
(Milestone 9) has enough evidence per skill -- that refit, when it
exists, writes a new 'bkt_parameters' modeling.model_snapshots row rather
than mutating these defaults in place.

State is NOT cached in memory or in a dedicated "current P(L)" column --
BKT's posterior for (student, topic) is recomputed by replaying the prior
'bkt' evidence_log rows for that pair each time, which is what makes the
whole chain replayable from raw events plus versioned parameters (spec
section 19), at the cost of doing a little redundant arithmetic per call.
At this project's realistic data volume that tradeoff is a non-issue.
"""

from __future__ import annotations

import json
from typing import Any, Optional

import asyncpg
import structlog

from app.db.postgres import get_pool
from app.db.postgres_gckt import get_active_model_snapshot, insert_evidence, insert_model_snapshot

logger = structlog.get_logger()

# Standard literature defaults for an uncalibrated BKT model (e.g. Corbett
# & Anderson 1994's commonly-cited starting values) -- not fit to this
# project's data, because there isn't enough of it yet to fit anything.
DEFAULT_BKT_PARAMS: dict[str, float] = {
    "p_init": 0.3,
    "p_transit": 0.15,
    "p_slip": 0.1,
    "p_guess": 0.25,
}

# Minimum responses-per-skill before a real per-subject refit is even
# attempted (Milestone 9) -- fitting slip/guess/transit off fewer than
# this is more likely to overfit noise than recover a real parameter.
# Referenced here (not just in the refit job) so this module's docstring
# and the refit job's threshold can never silently drift apart.
MIN_OBSERVATIONS_FOR_REFIT = 30


async def _get_params(topic_id: Optional[str] = None) -> tuple[dict[str, float], Optional[str]]:
    """Returns (params, model_snapshot_id). Prefers a per-skill refit
    (Milestone 9's nightly job, scope=topic_id) a human has promoted to
    'active', then the global population-default snapshot (scope=NULL,
    seeded as a real 'active' snapshot on first use so every piece of
    evidence this module writes always has a real model_snapshot_id
    provenance pointer, never a null one standing in for "used the
    hardcoded defaults")."""
    if topic_id is not None:
        scoped = await get_active_model_snapshot("bkt_parameters", scope=topic_id)
        if scoped is not None:
            config = scoped["config"]
            params = json.loads(config) if isinstance(config, str) else config
            return params, str(scoped["id"])

    snap = await get_active_model_snapshot("bkt_parameters", scope=None)
    if snap is None:
        snapshot_id = await insert_model_snapshot(
            "bkt_parameters",
            DEFAULT_BKT_PARAMS,
            scope=None,
            status="active",
            notes="Population-default BKT parameters (uncalibrated) -- seeded on first use, no real fit exists yet.",
        )
        return dict(DEFAULT_BKT_PARAMS), snapshot_id
    config = snap["config"]
    params = json.loads(config) if isinstance(config, str) else config
    return params, str(snap["id"])


def _bkt_update(p_know: float, correct: bool, params: dict[str, float]) -> float:
    """One BKT forward-pass step: posterior-given-observation, then the
    transition to the next opportunity. Returns the updated P(L)."""
    slip, guess, transit = params["p_slip"], params["p_guess"], params["p_transit"]

    if correct:
        numerator = p_know * (1 - slip)
        denominator = numerator + (1 - p_know) * guess
    else:
        numerator = p_know * slip
        denominator = numerator + (1 - p_know) * (1 - guess)

    p_given_obs = numerator / denominator if denominator > 0 else p_know
    return p_given_obs + (1 - p_given_obs) * transit


async def _current_mastery(student_id: str, topic_id: str, params: dict[str, float]) -> tuple[float, int]:
    """Replays this (student, topic) pair's prior 'bkt' evidence to
    recover the last posterior P(L) and how many observations have fed it
    so far. No prior evidence => start from p_init with zero evidence,
    exactly the cold-start behavior spec section 15 calls for."""
    pool = await get_pool()
    row = await pool.fetchrow(
        """
        SELECT estimate, sample_size FROM modeling.evidence_log
        WHERE student_id = $1 AND topic_id = $2 AND provenance = 'bkt'
        ORDER BY created_at DESC LIMIT 1
        """,
        student_id,
        topic_id,
    )
    if row is None or row["estimate"] is None:
        return params["p_init"], 0
    return float(row["estimate"]), int(row["sample_size"] or 0)


def _uncertainty_for(evidence_count: int) -> float:
    """A simple, honestly-documented shrinking heuristic -- NOT a true HMM
    posterior variance. BKT's actual posterior uncertainty would require
    propagating a full belief distribution through the forward algorithm,
    which is more machinery than this system's current evidence volume
    justifies. This gives a monotonically-decreasing uncertainty that
    starts high (cold start) and never claims more precision than a
    handful of observations warrant, without pretending to be a rigorous
    variance estimate."""
    return max(0.05, 1.0 / (1.0 + evidence_count))


async def process_attempt(attempt_id: str, events: list[asyncpg.Record]) -> None:
    """Applies one BKT update per (skill, response) pair touched by this
    attempt's events, writing one modeling.evidence_log row (provenance=
    'bkt') per update. A single event can touch multiple skill_ids (an
    item can assess more than one topic); each gets its own independent
    BKT update -- this module treats every mapped skill as if the item
    were a pure single-skill item for that skill, which is the standard
    simplification single-skill BKT makes when handed a multi-skill Q-matrix
    row (DINA in this same package handles the multi-skill conjunction
    case properly; BKT's per-skill treatment here is the accepted
    complementary evidence source, not a duplicate of DINA's job)."""
    for event in events:
        skill_ids = event["skill_ids"] or []
        if not skill_ids:
            continue
        correct = event["correctness"]
        if correct is None:
            continue

        for topic_id in skill_ids:
            student_id = str(event["student_id"])
            topic_id = str(topic_id)

            params, snapshot_id = await _get_params(topic_id)
            p_know, evidence_count = await _current_mastery(student_id, topic_id, params)
            p_next = _bkt_update(p_know, bool(correct), params)
            evidence_count += 1

            await insert_evidence(
                student_id,
                topic_id,
                estimate=p_next,
                uncertainty=_uncertainty_for(evidence_count),
                sample_size=evidence_count,
                reliability=0.8,  # spec's "Model at a Glance" Authority column: BKT = primary temporal evidence
                provenance="bkt",
                context={"correct": bool(correct), "itemId": str(event["item_id"]) if event["item_id"] else None},
                model_snapshot_id=snapshot_id,
                source_event_id=str(event["id"]),
            )

    logger.info("bkt.process_attempt.done", attempt_id=attempt_id, events=len(events))
