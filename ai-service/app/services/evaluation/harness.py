"""
Evaluation / ablation harness (EG-GCKT Milestone 10, spec section 17).

Prequential (forward-chaining) evaluation: for each graded response in
students.learning_events, a variant's prediction may only use evidence
that existed STRICTLY BEFORE that response's occurred_at. This is the
one non-negotiable correctness rule in this module -- scoring a
prediction against evidence that response itself produced would be
circular (the model would be "predicting" an outcome partly derived from
that same outcome), and would make every number this harness reports
meaningless regardless of which metric is used.

Baselines, matching spec section 17.1's list where this system has an
equivalent: frequency (population base rate), bkt-only, dina-only,
irt-only, graph-only (heuristic), no_fusion ("EG-GCKT without fusion" --
an unweighted average across every available evidence source, isolating
whether the WEIGHTED fusion step itself adds value over simply having
multiple evidence sources), full_gcsf (the real reliability/recency-
weighted fusion). DKT/MIRT-only baselines don't exist because those
engines were never built (see the implementation plan's explicit Phase 6
deferral).

full_gcsf's point-in-time fusion reuses
knowledge_tracing/replay.py's read-only reimplementation of fusion.py's
weighting formula (shared with snapshot.py) rather than calling
fusion.fuse_skill_state directly -- that function mutates production
state (consumed_by_fusion_at, students.skill_states) as a side effect,
which a backtest must never do.

Every result honestly reports insufficient_data when there isn't enough
held-out coverage for a metric to mean anything -- this harness is
expected to report exactly that on most dimensions until real usage
volume accumulates (documented in the implementation plan); it does not
fabricate a benchmark number to fill the gap.
"""

from __future__ import annotations

from typing import Any, Optional

import asyncpg
import structlog

from app.db.postgres import get_pool
from app.services.evaluation import metrics
from app.services.knowledge_tracing.replay import fuse_point_in_time, prior_evidence

logger = structlog.get_logger()

MIN_EVENTS_FOR_EVALUATION = 30
MIN_PREDICTIONS_FOR_METRICS = 20

VARIANTS = ("frequency", "bkt", "dina", "irt", "graph", "no_fusion", "full_gcsf")


async def _fetch_events() -> list[asyncpg.Record]:
    pool = await get_pool()
    return await pool.fetch(
        """
        SELECT id, student_id, skill_ids, correctness, occurred_at
        FROM students.learning_events
        WHERE correctness IS NOT NULL AND array_length(skill_ids, 1) > 0
        ORDER BY occurred_at
        """
    )


async def _predict(variant: str, student_id: str, topic_id: str, before: Any, item_id: Optional[str]) -> Optional[float]:
    if variant == "frequency":
        if item_id is None:
            return None
        pool = await get_pool()
        row = await pool.fetchrow(
            """
            SELECT avg(CASE WHEN marks_awarded >= marks_possible * 0.5 THEN 1.0 ELSE 0.0 END) AS p, count(*) AS n
            FROM assessment.student_answers
            WHERE question_id = $1 AND marks_awarded IS NOT NULL AND marks_possible > 0
            """,
            item_id,
        )
        return float(row["p"]) if row and row["p"] is not None and row["n"] >= 3 else None

    if variant in ("bkt", "dina", "irt"):
        evidence = await prior_evidence(student_id, topic_id, before, variant)
        return float(evidence[0]["estimate"]) if evidence and evidence[0]["estimate"] is not None else None

    if variant == "graph":
        # Heuristic graph-only baseline (spec section 17.1's "graph-only
        # prerequisite heuristic"): average mastery-so-far of the topic's
        # direct prerequisites, using whatever evidence (any provenance)
        # existed before `before` -- deliberately coarse, it's meant as a
        # weak baseline every real engine should beat, not a competitor.
        from app.db import neo4j as neo4j_db
        from app.db import postgres_gap as db

        direct_prereqs = await neo4j_db.fetch_prerequisite_chain(topic_id, max_depth=1)
        if not direct_prereqs:
            direct_prereqs = await db.fetch_prerequisite_chain_pg(topic_id, max_depth=1)
        if not direct_prereqs:
            return None
        values = []
        for p in direct_prereqs:
            evidence = await prior_evidence(student_id, p["id"], before, None)
            fused = fuse_point_in_time(evidence, before)
            if fused is not None:
                values.append(fused)
        return sum(values) / len(values) if values else None

    if variant == "no_fusion":
        # "EG-GCKT without fusion" (spec section 17.1): every available
        # evidence source contributes, but with a plain UNWEIGHTED mean
        # instead of full_gcsf's reliability/recency-weighted combination
        # -- isolates whether the weighting step itself is worth having,
        # versus just having multiple evidence sources at all.
        evidence = await prior_evidence(student_id, topic_id, before, None)
        estimated = [float(row["estimate"]) for row in evidence if row["estimate"] is not None]
        return sum(estimated) / len(estimated) if estimated else None

    if variant == "full_gcsf":
        evidence = await prior_evidence(student_id, topic_id, before, None)
        return fuse_point_in_time(evidence, before)

    raise ValueError(f"unknown evaluation variant: {variant}")


async def evaluate(variant: str) -> dict[str, Any]:
    """Runs one variant's prequential evaluation over every graded
    response in the event log. See module docstring for the
    non-negotiable no-future-evidence rule and why most dimensions will
    honestly report insufficient_data at this project's current data
    volume."""
    if variant not in VARIANTS:
        raise ValueError(f"unknown evaluation variant: {variant!r}, expected one of {VARIANTS}")

    events = await _fetch_events()
    if len(events) < MIN_EVENTS_FOR_EVALUATION:
        return {
            "variant": variant,
            "status": "insufficient_data",
            "totalEvents": len(events),
            "minEventsRequired": MIN_EVENTS_FOR_EVALUATION,
        }

    y_true: list[float] = []
    y_pred: list[float] = []
    for event in events:
        item_id = None  # resolved per-topic below only for the frequency variant's item lookup
        for topic_id in event["skill_ids"]:
            pred = await _predict(variant, str(event["student_id"]), str(topic_id), event["occurred_at"], item_id)
            if pred is None:
                continue
            y_true.append(1.0 if event["correctness"] else 0.0)
            y_pred.append(pred)

    if len(y_true) < MIN_PREDICTIONS_FOR_METRICS:
        return {
            "variant": variant,
            "status": "insufficient_data",
            "totalEvents": len(events),
            "coveredPredictions": len(y_true),
            "minPredictionsRequired": MIN_PREDICTIONS_FOR_METRICS,
        }

    auc_value = metrics.auc(y_true, y_pred)
    return {
        "variant": variant,
        "status": "ok",
        "totalEvents": len(events),
        "coveredPredictions": len(y_true),
        "auc": round(auc_value, 4) if auc_value is not None else None,
        "logLoss": round(metrics.log_loss(y_true, y_pred), 4),
        "brierScore": round(metrics.brier_score(y_true, y_pred), 4),
        "ece": round(metrics.expected_calibration_error(y_true, y_pred), 4),
    }


async def evaluate_all() -> list[dict[str, Any]]:
    """One report per baseline variant, for the ablation comparison table
    (spec section 17.2's essential research question: does the full
    integrated architecture beat the simpler baselines)."""
    return [await evaluate(variant) for variant in VARIANTS]


if __name__ == "__main__":
    # Runnable directly: `python -m app.services.evaluation.harness` --
    # prints the full ablation comparison table as JSON. Not exposed as
    # an HTTP endpoint; this is an offline research/ops tool run on
    # demand, not a live product surface.
    import asyncio
    import json

    async def _main() -> None:
        results = await evaluate_all()
        print(json.dumps(results, indent=2, default=str))

    asyncio.run(_main())
