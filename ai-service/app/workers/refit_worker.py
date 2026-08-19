"""
Nightly BKT/DINA/IRT parameter refit (EG-GCKT Milestone 9, spec section
19's versioning/governance requirement).

Python asyncio periodic loop -- the same "background task embedded in the
main process, no Celery/cron" shape this codebase already uses for Go's
qualityworker (backend/internal/assessment/qualityworker/worker.go),
translated to ai-service since the statistical fitting code lives here.
Every other ai-service worker is a Redis BRPOP consumer; this is
deliberately the first time-based one, because "refit whenever enough new
evidence has accumulated" doesn't map to a single triggering event the
way "an exam attempt was just graded" does.

Fitting method: coarse grid-search maximum-likelihood estimation, not
full Baum-Welch EM or gradient-based joint calibration. This is
real, principled MLE -- picking the parameter combination that maximizes
the likelihood of the observed responses -- just computationally coarser
than the most efficient variant. Given this system's realistic evidence
volume (documented during planning as ~zero real interaction data at
launch), a coarse grid is more than adequate and easier to verify
correct than a more elaborate iterative fitter would be.

Every refit writes a NEW modeling.model_snapshots row with
status='candidate' -- it NEVER mutates or activates a snapshot in place.
A human (ministry_admin/curriculum_officer) reviews and promotes
candidate -> active through the governance endpoints
(backend/internal/modeling/...) before a refit takes effect on the live
engines (bkt.py/dina.py/irt.py all check for an 'active' scoped snapshot
before falling back to defaults/CTT proxies).
"""

from __future__ import annotations

import asyncio
import json
import math
from collections import defaultdict
from typing import Optional

import structlog

from app.db import postgres_studyplan as studyplan_db
from app.db.dead_letter import requeue_retriable_jobs
from app.db.postgres import get_pool
from app.db.postgres_gckt import insert_model_snapshot
from app.db.redis import get_redis
from app.services.knowledge_tracing.snapshot import take_all_snapshots

logger = structlog.get_logger()

_shutdown = asyncio.Event()

INITIAL_DELAY_SECONDS = 90.0
DEFAULT_INTERVAL_SECONDS = 24 * 3600.0

MIN_OBSERVATIONS_FOR_REFIT = 30

# EG-GCKT checklist sections 2/14/20 (recommendation outcome / feedback
# loop): how long to wait after a recommendation before judging its
# outcome, and the minimum mastery_probability movement to call it
# "improved"/"worsened" rather than noise ("unchanged"). Both hand-set,
# not fit to any outcome data -- same honesty caveat as the ranking
# weights in action_ranking.py.
OUTCOME_EVALUATION_DELAY_DAYS = 7
OUTCOME_IMPROVEMENT_DELTA = 0.05
# A cold-start recommendation (no mastery_at_recommendation baseline) is
# judged against an absolute bar instead of a delta, since there's
# nothing to subtract from.
OUTCOME_COLD_START_IMPROVED_THRESHOLD = 0.5

SLIP_GUESS_GRID = [0.05, 0.10, 0.15, 0.20, 0.25, 0.30, 0.35]
TRANSIT_GRID = [0.05, 0.10, 0.15, 0.20, 0.25, 0.30, 0.40]
IRT_A_GRID = [0.3, 0.5, 0.8, 1.0, 1.3, 1.6, 2.0, 2.5]
IRT_B_GRID = [-2.0, -1.5, -1.0, -0.5, 0.0, 0.5, 1.0, 1.5, 2.0]

_EPS = 1e-9


def _clip_prob(p: float) -> float:
    return min(1.0 - _EPS, max(_EPS, p))


# ── BKT refit ──────────────────────────────────────────────────────────


async def _fetch_bkt_sequences(topic_id: str) -> dict[str, list[bool]]:
    """Every student's ordered correct/incorrect sequence on this skill,
    from the raw event log (not from BKT's own previously-computed
    evidence -- refitting against the engine's own output would be
    circular)."""
    pool = await get_pool()
    rows = await pool.fetch(
        """
        SELECT student_id, correctness, occurred_at
        FROM students.learning_events
        WHERE $1 = ANY(skill_ids) AND correctness IS NOT NULL
        ORDER BY student_id, occurred_at
        """,
        topic_id,
    )
    sequences: dict[str, list[bool]] = defaultdict(list)
    for r in rows:
        sequences[str(r["student_id"])].append(bool(r["correctness"]))
    return sequences


def _bkt_sequence_log_likelihood(seq: list[bool], p_init: float, p_slip: float, p_guess: float, p_transit: float) -> float:
    p_know = p_init
    ll = 0.0
    for correct in seq:
        p_correct = _clip_prob(p_know * (1 - p_slip) + (1 - p_know) * p_guess)
        ll += math.log(p_correct if correct else (1 - p_correct))

        if correct:
            num = p_know * (1 - p_slip)
            den = num + (1 - p_know) * p_guess
        else:
            num = p_know * p_slip
            den = num + (1 - p_know) * (1 - p_guess)
        p_given_obs = num / den if den > 0 else p_know
        p_know = p_given_obs + (1 - p_given_obs) * p_transit
    return ll


def _grid_search_bkt(sequences: dict[str, list[bool]]) -> tuple[dict[str, float], float]:
    firsts = [seq[0] for seq in sequences.values() if seq]
    p_init = min(0.95, max(0.05, sum(firsts) / len(firsts))) if firsts else 0.3

    best_ll = -math.inf
    best = {"p_init": p_init, "p_transit": 0.15, "p_slip": 0.1, "p_guess": 0.25}
    for slip in SLIP_GUESS_GRID:
        for guess in SLIP_GUESS_GRID:
            for transit in TRANSIT_GRID:
                total_ll = sum(
                    _bkt_sequence_log_likelihood(seq, p_init, slip, guess, transit) for seq in sequences.values()
                )
                if total_ll > best_ll:
                    best_ll = total_ll
                    best = {"p_init": p_init, "p_slip": slip, "p_guess": guess, "p_transit": transit}
    return best, best_ll


async def refit_bkt() -> int:
    pool = await get_pool()
    rows = await pool.fetch(
        """
        SELECT skill_id, count(*) AS n FROM (
            SELECT unnest(skill_ids) AS skill_id FROM students.learning_events WHERE correctness IS NOT NULL
        ) x GROUP BY skill_id HAVING count(*) >= $1
        """,
        MIN_OBSERVATIONS_FOR_REFIT,
    )

    written = 0
    for row in rows:
        topic_id = str(row["skill_id"])
        sequences = await _fetch_bkt_sequences(topic_id)
        if sum(len(s) for s in sequences.values()) < MIN_OBSERVATIONS_FOR_REFIT:
            continue
        params, log_likelihood = _grid_search_bkt(sequences)
        await insert_model_snapshot(
            "bkt_parameters",
            params,
            scope=topic_id,
            status="candidate",
            training_summary={
                "students": len(sequences),
                "observations": sum(len(s) for s in sequences.values()),
                "logLikelihood": round(log_likelihood, 2),
                "method": "grid_search_mle",
            },
            notes="EG-GCKT Milestone 9 nightly refit -- candidate, pending governance review.",
        )
        written += 1
    return written


# ── DINA refit ─────────────────────────────────────────────────────────


async def _fetch_dina_observations(question_id: str) -> list[tuple[bool, float]]:
    """(correct, currentMasteryEstimate) pairs for every response to this
    item -- masteryEstimate averaged across the item's mapped skills'
    CURRENT fused skill_states (a real, if approximate, stand-in for the
    "true attribute pattern" full joint DINA EM would otherwise need to
    estimate as a latent E-step; see module docstring on why a coarse,
    honest approximation is the right tradeoff here)."""
    pool = await get_pool()
    rows = await pool.fetch(
        """
        SELECT sa.marks_awarded, sa.marks_possible, le.student_id, le.skill_ids
        FROM assessment.student_answers sa
        JOIN students.learning_events le ON le.answer_id = sa.id
        WHERE sa.question_id = $1 AND sa.marks_awarded IS NOT NULL AND sa.marks_possible > 0
        """,
        question_id,
    )
    observations: list[tuple[bool, float]] = []
    for r in rows:
        skill_ids = [str(t) for t in (r["skill_ids"] or [])]
        if not skill_ids:
            continue
        states = await pool.fetch(
            "SELECT mastery_probability FROM students.skill_states WHERE student_id = $1 AND topic_id = ANY($2::uuid[])",
            r["student_id"],
            skill_ids,
        )
        values = [float(s["mastery_probability"]) for s in states if s["mastery_probability"] is not None]
        mastery = sum(values) / len(values) if values else 0.5
        correct = float(r["marks_awarded"]) / float(r["marks_possible"]) >= 0.5
        observations.append((correct, mastery))
    return observations


def _dina_log_likelihood(observations: list[tuple[bool, float]], slip: float, guess: float) -> float:
    ll = 0.0
    for correct, mastery in observations:
        p_correct = _clip_prob(mastery * (1 - slip) + (1 - mastery) * guess)
        ll += math.log(p_correct if correct else (1 - p_correct))
    return ll


def _grid_search_dina(observations: list[tuple[bool, float]]) -> tuple[dict[str, float], float]:
    best_ll = -math.inf
    best = {"slip": 0.2, "guess": 0.2}
    for slip in SLIP_GUESS_GRID:
        for guess in SLIP_GUESS_GRID:
            ll = _dina_log_likelihood(observations, slip, guess)
            if ll > best_ll:
                best_ll = ll
                best = {"slip": slip, "guess": guess}
    return best, best_ll


async def refit_dina() -> int:
    pool = await get_pool()
    rows = await pool.fetch(
        """
        SELECT question_id, count(*) AS n FROM assessment.student_answers
        WHERE marks_awarded IS NOT NULL AND marks_possible > 0
        GROUP BY question_id HAVING count(*) >= $1
        """,
        MIN_OBSERVATIONS_FOR_REFIT,
    )

    written = 0
    for row in rows:
        question_id = str(row["question_id"])
        observations = await _fetch_dina_observations(question_id)
        if len(observations) < MIN_OBSERVATIONS_FOR_REFIT:
            continue
        params, log_likelihood = _grid_search_dina(observations)
        await insert_model_snapshot(
            "dina_parameters",
            params,
            scope=question_id,
            status="candidate",
            training_summary={
                "observations": len(observations),
                "logLikelihood": round(log_likelihood, 2),
                "method": "grid_search_mle",
            },
            notes="EG-GCKT Milestone 9 nightly refit -- candidate, pending governance review.",
        )
        written += 1
    return written


# ── IRT refit ──────────────────────────────────────────────────────────


async def _fetch_irt_observations(question_id: str) -> list[tuple[float, bool]]:
    """(theta_proxy, correct) pairs. theta_proxy per student is their most
    recent 'irt' evidence context theta if one exists, else derived from
    the item's mapped skills' current fused mastery via the inverse of
    the same logistic squash irt.py uses to go the other direction."""
    pool = await get_pool()
    rows = await pool.fetch(
        """
        SELECT sa.marks_awarded, sa.marks_possible, le.student_id, le.skill_ids
        FROM assessment.student_answers sa
        JOIN students.learning_events le ON le.answer_id = sa.id
        WHERE sa.question_id = $1 AND sa.marks_awarded IS NOT NULL AND sa.marks_possible > 0
        """,
        question_id,
    )
    observations: list[tuple[float, bool]] = []
    for r in rows:
        theta_row = await pool.fetchrow(
            """
            SELECT context FROM modeling.evidence_log
            WHERE student_id = $1 AND provenance = 'irt' AND context IS NOT NULL
            ORDER BY created_at DESC LIMIT 1
            """,
            r["student_id"],
        )
        theta: Optional[float] = None
        if theta_row is not None:
            try:
                ctx = json.loads(theta_row["context"]) if isinstance(theta_row["context"], str) else theta_row["context"]
                theta = float(ctx.get("theta")) if isinstance(ctx, dict) and "theta" in ctx else None
            except (TypeError, ValueError):
                theta = None
        if theta is None:
            skill_ids = [str(t) for t in (r["skill_ids"] or [])]
            mastery = 0.5
            if skill_ids:
                states = await pool.fetch(
                    "SELECT mastery_probability FROM students.skill_states WHERE student_id = $1 AND topic_id = ANY($2::uuid[])",
                    r["student_id"],
                    skill_ids,
                )
                values = [float(s["mastery_probability"]) for s in states if s["mastery_probability"] is not None]
                mastery = sum(values) / len(values) if values else 0.5
            mastery = _clip_prob(mastery)
            theta = math.log(mastery / (1 - mastery))  # inverse of irt.py's sigmoid squash

        correct = float(r["marks_awarded"]) / float(r["marks_possible"]) >= 0.5
        observations.append((theta, correct))
    return observations


def _irt_log_likelihood(observations: list[tuple[float, bool]], a: float, b: float) -> float:
    ll = 0.0
    for theta, correct in observations:
        p = _clip_prob(1.0 / (1.0 + math.exp(-a * (theta - b))))
        ll += math.log(p if correct else (1 - p))
    return ll


def _grid_search_irt(observations: list[tuple[float, bool]]) -> tuple[dict[str, float], float]:
    best_ll = -math.inf
    best = {"a": 1.0, "b": 0.0}
    for a in IRT_A_GRID:
        for b in IRT_B_GRID:
            ll = _irt_log_likelihood(observations, a, b)
            if ll > best_ll:
                best_ll = ll
                best = {"a": a, "b": b}
    return best, best_ll


async def refit_irt() -> int:
    pool = await get_pool()
    rows = await pool.fetch(
        """
        SELECT question_id, count(*) AS n FROM assessment.student_answers
        WHERE marks_awarded IS NOT NULL AND marks_possible > 0
        GROUP BY question_id HAVING count(*) >= $1
        """,
        MIN_OBSERVATIONS_FOR_REFIT,
    )

    written = 0
    for row in rows:
        question_id = str(row["question_id"])
        observations = await _fetch_irt_observations(question_id)
        if len(observations) < MIN_OBSERVATIONS_FOR_REFIT:
            continue
        params, log_likelihood = _grid_search_irt(observations)
        await insert_model_snapshot(
            "irt_calibration",
            {**params, "n": len(observations)},
            scope=question_id,
            status="candidate",
            training_summary={
                "observations": len(observations),
                "logLikelihood": round(log_likelihood, 2),
                "method": "grid_search_mle",
            },
            notes="EG-GCKT Milestone 9 nightly refit -- candidate, pending governance review.",
        )
        written += 1
    return written


# ── Recommendation outcome evaluation ────────────────────────────────
# EG-GCKT checklist sections 2/14/20: "measure learning gain" / "capture
# action outcome for the feedback loop". study_plan/service.py captures a
# mastery/evidence baseline at recommendation time (V049); this closes
# the loop by comparing it against the CURRENT skill_states row once
# enough time has passed, and action_ranking.fetch_repetition_counts
# reads the result back in on the next plan generation (a demonstrably
# ineffective repeat is penalized harder than an ordinary repeat).


def _classify_outcome(
    mastery_before: Optional[float],
    evidence_before: int,
    mastery_now: Optional[float],
    evidence_now: int,
) -> str:
    """Pure classification, unit-tested directly. No new evidence at all
    since the recommendation means there's nothing to judge yet --
    that's 'insufficient_evidence', not 'unchanged' (unchanged implies we
    looked and it didn't move; this is "we couldn't look")."""
    if evidence_now <= evidence_before:
        return "insufficient_evidence"
    if mastery_now is None:
        return "insufficient_evidence"
    if mastery_before is None:
        return "improved" if mastery_now >= OUTCOME_COLD_START_IMPROVED_THRESHOLD else "unchanged"
    delta = mastery_now - mastery_before
    if delta >= OUTCOME_IMPROVEMENT_DELTA:
        return "improved"
    if delta <= -OUTCOME_IMPROVEMENT_DELTA:
        return "worsened"
    return "unchanged"


async def evaluate_recommendation_outcomes(older_than_days: int = OUTCOME_EVALUATION_DELAY_DAYS) -> int:
    pending = await studyplan_db.fetch_pending_recommendation_outcomes(older_than_days)
    if not pending:
        return 0
    updates = [
        (
            row["id"],
            _classify_outcome(row["mastery_before"], row["evidence_before"], row["mastery_now"], row["evidence_now"]),
            row["mastery_now"],
        )
        for row in pending
    ]
    await studyplan_db.apply_recommendation_outcomes(updates)
    return len(updates)


# ── Scheduling ─────────────────────────────────────────────────────────


async def run_once() -> None:
    bkt_written = await refit_bkt()
    dina_written = await refit_dina()
    irt_written = await refit_irt()

    # EG-GCKT checklist sections 6/18/22: nightly learner-state snapshots
    # -- taken in the same cycle as (but independent of) parameter
    # refitting, so "what did we believe before tonight's refit" is
    # always recoverable even before any candidate gets promoted.
    snapshots_written = await take_all_snapshots(reason="nightly")

    # EG-GCKT checklist sections 2/14/20: same nightly cycle, independent
    # of parameter refitting -- judges recommendations old enough to have
    # a fair outcome and feeds the result back into the next plan
    # generation's repetition penalty.
    outcomes_evaluated = await evaluate_recommendation_outcomes()

    dead_letter_requeued = await requeue_retriable_jobs(get_redis())

    logger.info(
        "refit_worker.completed",
        bkt_candidates=bkt_written,
        dina_candidates=dina_written,
        irt_candidates=irt_written,
        snapshots_written=snapshots_written,
        outcomes_evaluated=outcomes_evaluated,
        dead_letter_requeued=dead_letter_requeued,
    )


async def run_forever(initial_delay: float = INITIAL_DELAY_SECONDS, interval: float = DEFAULT_INTERVAL_SECONDS) -> None:
    logger.info("refit_worker.started", interval_seconds=interval)
    try:
        await asyncio.wait_for(_shutdown.wait(), timeout=initial_delay)
        return  # shutdown requested during the initial delay
    except asyncio.TimeoutError:
        pass

    while not _shutdown.is_set():
        try:
            await run_once()
        except Exception:  # noqa: BLE001 -- a bad refit cycle must not kill the worker; try again next interval
            logger.exception("refit_worker.cycle_failed")

        try:
            await asyncio.wait_for(_shutdown.wait(), timeout=interval)
        except asyncio.TimeoutError:
            continue


def request_shutdown() -> None:
    _shutdown.set()


if __name__ == "__main__":
    try:
        asyncio.run(run_once())
    except KeyboardInterrupt:
        pass
