"""
DINA / G-DINA cognitive diagnosis (EG-GCKT Milestone 3, spec section 7.1).

Binarized Q-matrix: an item "requires" a skill when
item_skill_mappings.relevance >= 0.5 (already resolved into
learning_events.skill_ids by Go's RecordLearningEvents, so this module
just consumes that array directly, same as bkt.py).

Cold start (spec section 15): slip=guess=0.2 population defaults per
item -- the standard psychometric default when nothing has been
empirically calibrated yet, seeded once as a global 'dina_parameters'
modeling.model_snapshots row. A real per-item slip/guess calibration
(EM fit against enough responses) is Milestone 9 batch-job work; this
module never fabricates per-item values in the meantime.

Joint-vs-independent modeling: true DINA needs a joint posterior over the
full 2^K attribute space for K skills. Doing that across an entire exam's
skill set (which can be 15-20+ distinct topics) is both computationally
wasteful and not what any individual item's response actually constrains.
Instead, each item induces a proper joint DINA update over only the small
set of skills THAT ITEM requires (almost always 1-3 in this dataset's
Q-matrix, so 2^k is trivially small) under the deterministic-input-noisy-
AND assumption (eta = AND of the item's required skills), and the
resulting marginal per-skill posteriors are written back as that skill's
new belief, becoming the prior the next item touching the same skill
updates from. This is a standard, principled way to run DINA sequentially
one item at a time without requiring a single ahead-of-time global
attribute space -- not an ad hoc shortcut, but it is a documented
simplification from "one joint posterior over every skill in the exam,"
which would be both intractable and unjustified at this system's item
counts.
"""

from __future__ import annotations

import itertools
import json
from collections import defaultdict
from typing import Optional

import asyncpg
import structlog

from app.db.postgres import get_pool
from app.db.postgres_gckt import get_active_model_snapshot, insert_evidence, insert_model_snapshot

logger = structlog.get_logger()

DEFAULT_DINA_PARAMS: dict[str, float] = {"slip": 0.2, "guess": 0.2}

# Population-level starting belief for "this skill is mastered" before any
# evidence exists -- kept distinct from (but numerically equal to) BKT's
# p_init; the two engines are independent evidence sources by design (spec
# section 8: fusion combines them, neither is authoritative alone) and
# should not import each other's internals to stay decoupled.
DEFAULT_PRIOR_MASTERY = 0.3

# Never expand a joint attribute space beyond this many skills per item --
# a real Q-matrix row this wide would be a data-quality problem to flag,
# not something to brute-force through 2^k enumeration.
MAX_JOINT_SKILLS = 20


async def _get_params(question_id: Optional[str] = None) -> tuple[dict[str, float], Optional[str]]:
    """Prefers a per-item refit (Milestone 9's nightly job, scope=
    question_id) a human has promoted to 'active', then the global
    population-default snapshot."""
    if question_id is not None:
        scoped = await get_active_model_snapshot("dina_parameters", scope=question_id)
        if scoped is not None:
            config = scoped["config"]
            params = json.loads(config) if isinstance(config, str) else config
            return params, str(scoped["id"])

    snap = await get_active_model_snapshot("dina_parameters", scope=None)
    if snap is None:
        snapshot_id = await insert_model_snapshot(
            "dina_parameters",
            DEFAULT_DINA_PARAMS,
            scope=None,
            status="active",
            notes="Population-default DINA slip/guess (uncalibrated) -- seeded on first use, no real per-item fit exists yet.",
        )
        return dict(DEFAULT_DINA_PARAMS), snapshot_id
    config = snap["config"]
    params = json.loads(config) if isinstance(config, str) else config
    return params, str(snap["id"])


async def _current_priors(student_id: str, topic_ids: list[str]) -> dict[str, float]:
    """Last DINA marginal posterior per skill, or the population prior if
    this student/skill pair has no DINA evidence yet."""
    pool = await get_pool()
    rows = await pool.fetch(
        """
        SELECT DISTINCT ON (topic_id) topic_id, estimate
        FROM modeling.evidence_log
        WHERE student_id = $1 AND topic_id = ANY($2::uuid[]) AND provenance = 'dina'
        ORDER BY topic_id, created_at DESC
        """,
        student_id,
        topic_ids,
    )
    priors = {str(r["topic_id"]): float(r["estimate"]) for r in rows if r["estimate"] is not None}
    for tid in topic_ids:
        priors.setdefault(tid, DEFAULT_PRIOR_MASTERY)
    return priors


def _joint_dina_update(priors: list[float], correct: bool, slip: float, guess: float) -> list[float]:
    """Exact joint posterior over the 2^k attribute patterns for the k
    skills one item requires (eta = AND of all of them, the DINA
    conjunctive assumption), then marginalized back to one P(mastered)
    per skill. k is the number of skills this single item maps to, not
    the exam's total skill count."""
    k = len(priors)
    weighted: list[tuple[tuple[int, ...], float]] = []
    for pattern in itertools.product((0, 1), repeat=k):
        p_pattern = 1.0
        for i, bit in enumerate(pattern):
            p_pattern *= priors[i] if bit == 1 else (1 - priors[i])
        eta = 1 if all(pattern) else 0
        p_correct_given_pattern = (1 - slip) if eta == 1 else guess
        likelihood = p_correct_given_pattern if correct else (1 - p_correct_given_pattern)
        weighted.append((pattern, p_pattern * likelihood))

    total = sum(w for _, w in weighted)
    if total <= 0:
        return priors  # degenerate (shouldn't happen with slip/guess in (0,1)) -- keep priors unchanged

    marginals = []
    for i in range(k):
        marginals.append(sum(w for pattern, w in weighted if pattern[i] == 1) / total)
    return marginals


async def process_attempt(attempt_id: str, events: list[asyncpg.Record]) -> None:
    """One joint DINA update per graded item (batch per attempt, not
    online per-response -- matches DINA's normal usage as a per-test-form
    diagnostic rather than a per-interaction tracker, which is BKT's job).
    Writes one modeling.evidence_log row (provenance='dina') per
    (skill, item) pair processed."""
    by_student: dict[str, list[asyncpg.Record]] = defaultdict(list)
    for event in events:
        if event["correctness"] is None or not (event["skill_ids"] or []):
            continue
        by_student[str(event["student_id"])].append(event)

    processed = 0
    for student_id, student_events in by_student.items():
        for event in student_events:
            topic_ids = [str(t) for t in event["skill_ids"]][:MAX_JOINT_SKILLS]
            if not topic_ids:
                continue

            item_id = str(event["item_id"]) if event["item_id"] else None
            params, snapshot_id = await _get_params(item_id)
            slip, guess = params["slip"], params["guess"]

            priors_by_topic = await _current_priors(student_id, topic_ids)
            prior_values = [priors_by_topic[t] for t in topic_ids]
            posterior_values = _joint_dina_update(prior_values, bool(event["correctness"]), slip, guess)

            # A DINA posterior naturally sharpens with evidence, so its own
            # spread from 0.5 is a reasonable uncertainty proxy -- distance
            # from "maximally uninformative" rather than BKT's
            # evidence-count heuristic, since DINA's belief IS the
            # posterior variance's proxy here, not a separate count.
            for topic_id, posterior in zip(topic_ids, posterior_values):
                uncertainty = max(0.05, 1.0 - 2 * abs(posterior - 0.5))
                await insert_evidence(
                    student_id,
                    topic_id,
                    estimate=posterior,
                    uncertainty=uncertainty,
                    sample_size=1,
                    reliability=0.8,  # spec's Authority column: DINA/G-DINA = primary diagnostic evidence
                    provenance="dina",
                    context={
                        "correct": bool(event["correctness"]),
                        "itemId": str(event["item_id"]) if event["item_id"] else None,
                        "jointSkillCount": len(topic_ids),
                    },
                    model_snapshot_id=snapshot_id,
                    source_event_id=str(event["id"]),
                )
                processed += 1

    logger.info("dina.process_attempt.done", attempt_id=attempt_id, evidence_written=processed)
