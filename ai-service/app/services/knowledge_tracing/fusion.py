"""
Graph-Cognitive State Fusion Engine (GCSF) -- EG-GCKT Milestone 4, spec
section 8.

The central integration layer the spec calls "critical and well
motivated, but under-specified" -- this module is where that gets
formalized: combine every not-yet-fused Evidence object
(modeling.evidence_log) for a (student, topic) pair into one
uncertainty-aware students.skill_states row, with full provenance.

Per-source base reliability follows the spec's own "Model at a Glance"
Authority column (section 2): DINA/BKT are each listed as "Primary
[diagnostic|temporal] evidence", IRT as "Assessment evidence", graph
reasoning as "Structural evidence" -- read as a rough ranking, not exact
numbers, so the constants below are this module's own calibrated-by-eye
translation of the spec's authority language into weights, not a value
taken from the document itself.

Critical safeguard (spec section 8.3): graph-reasoning evidence is
deliberately given the lowest reliability weight of any source and is
never allowed to override observed evidence -- it nudges the fused
estimate, it doesn't set it (see root_cause.produce_graph_evidence, the
actual producer of provenance='graph_reasoning' rows -- called from
kt_worker.py right before this module runs). DKT/MIRT contribute nothing
(no code exists for either -- per the spec's own Phase 6 criterion,
deferred until real usage data justifies building them, see the
implementation plan) -- their absence from the weighted sum is itself
correct cold-start behavior, not a bug.
"""

from __future__ import annotations

import json
import math
from datetime import datetime, timezone
from typing import Any, Optional

import structlog

from app.db import neo4j as neo4j_db
from app.db import postgres_gap as pg_gap
from app.db.neo4j import sync_skill_state
from app.db.postgres_gckt import (
    fetch_active_recovery_for_topic,
    fetch_school_id_for_student,
    fetch_skill_state,
    fetch_skill_states_bulk,
    fetch_unconsumed_evidence,
    get_active_model_snapshot,
    insert_model_snapshot,
    mark_evidence_consumed,
    mark_skill_state_synced,
    upsert_skill_state,
)

logger = structlog.get_logger()

# Base reliability per evidence source, when the evidence row's own
# `reliability` field (set by the producing engine, e.g. bkt.py/dina.py)
# isn't already a confident, engine-specific number. In practice every
# producer in this codebase already stamps its own reliability at write
# time, so this is the fallback for any provenance a future engine adds
# without updating this table -- deliberately conservative (0.3) rather
# than silently trusting an unrecognized source at full weight.
DEFAULT_SOURCE_RELIABILITY: dict[str, float] = {
    "bkt": 0.8,
    "dina": 0.8,
    "irt": 0.5,
    "graph_reasoning": 0.3,
    "dkt": 0.4,
    "mirt": 0.5,
}
FALLBACK_RELIABILITY = 0.3

# Evidence below this sample size is down-weighted proportionally (spec
# section 8.2: "down-weight stale, low-sample, poorly calibrated, or
# low-quality evidence").
MIN_SAMPLE_FOR_FULL_WEIGHT = 5

# Recency half-life in days: evidence this old contributes half the
# weight of brand-new evidence of the same reliability/sample size.
RECENCY_HALF_LIFE_DAYS = 30.0

# mastery_probability thresholds for the four spec-defined mastery_status
# values (page 7's state-field table). Not specified numerically by the
# document itself -- this is a documented, defensible threshold choice,
# not a value taken from the spec.
STATUS_THRESHOLDS = (
    (0.4, "emerging"),
    (0.7, "proficient"),
)
TREND_EPSILON = 0.03


def _recency_weight(created_at: datetime) -> float:
    now = datetime.now(timezone.utc)
    age_days = max(0.0, (now - created_at).total_seconds() / 86400.0)
    return 0.5 ** (age_days / RECENCY_HALF_LIFE_DAYS)


def _source_weight(provenance: str, reliability: Optional[float], sample_size: Optional[int], created_at: datetime) -> float:
    # reliability comes straight off modeling.evidence_log's NUMERIC column,
    # so asyncpg hands back a decimal.Decimal here -- float() it before
    # mixing with the plain floats below (Decimal * float raises TypeError).
    base = float(reliability) if reliability is not None else DEFAULT_SOURCE_RELIABILITY.get(provenance, FALLBACK_RELIABILITY)
    sample_factor = min(1.0, (sample_size or 1) / MIN_SAMPLE_FOR_FULL_WEIGHT)
    return base * sample_factor * _recency_weight(created_at)


def _mastery_status(mastery_probability: Optional[float]) -> str:
    if mastery_probability is None:
        return "unknown"
    for threshold, status in STATUS_THRESHOLDS:
        if mastery_probability < threshold:
            return status
    return "mastered"


def _trend(current: float, previous: Optional[float]) -> Optional[str]:
    if previous is None:
        return None
    delta = current - previous
    if delta > TREND_EPSILON:
        return "improving"
    if delta < -TREND_EPSILON:
        return "declining"
    return "stable"


async def _get_fusion_policy_snapshot_id() -> Optional[str]:
    """Ensures a 'fusion_policy' modeling.model_snapshots row exists
    (global, scope=None), seeded once on first use with this module's
    weighting configuration -- the same governance pattern bkt.py/dina.py
    already use for their own parameters. Closes a real gap from the
    first EG-GCKT pass: this function's return value was previously
    always discarded and every skill_states row got model_snapshot_id=
    NULL, silently failing the spec's "every important state estimate
    has... model-version lineage" requirement (section 2) and "version
    fusion policy/model" (section 18)."""
    snap = await get_active_model_snapshot("fusion_policy", scope=None)
    if snap is not None:
        return str(snap["id"])
    config = {
        "defaultSourceReliability": DEFAULT_SOURCE_RELIABILITY,
        "fallbackReliability": FALLBACK_RELIABILITY,
        "minSampleForFullWeight": MIN_SAMPLE_FOR_FULL_WEIGHT,
        "recencyHalfLifeDays": RECENCY_HALF_LIFE_DAYS,
        "statusThresholds": list(STATUS_THRESHOLDS),
        "trendEpsilon": TREND_EPSILON,
    }
    return await insert_model_snapshot(
        "fusion_policy",
        config,
        scope=None,
        status="active",
        notes="Initial fusion weighting configuration -- hand-set, not fit to any outcome data (see module docstring).",
    )


async def _compute_structural_status(student_id: str, topic_id: str) -> dict[str, Any]:
    """GCSF's "structural status" output (spec section 8.3): prerequisite
    satisfaction, blocked status, inconsistencies. Reads the topic's own
    direct hard-dependency prerequisites (requires/strongly_requires) and
    whether an active recovery attempt exists for it (checklist section
    13) -- kept as a fusion-side computation, not root_cause.py's, since
    it belongs on every fused state, not just ones a gap-analysis pass
    happens to touch."""
    try:
        direct_prereqs = await neo4j_db.fetch_prerequisite_chain(topic_id, max_depth=1)
    except Exception:  # noqa: BLE001 -- graph-service outage (FI-008): degrade to the Postgres mirror, never fabricate structural status from a failed dependency
        logger.warning("fusion.neo4j_unavailable_falling_back", topic_id=topic_id, exc_info=True)
        direct_prereqs = []
    if not direct_prereqs:
        direct_prereqs = await pg_gap.fetch_prerequisite_chain_pg(topic_id, max_depth=1)

    prereqs_satisfied: Optional[bool] = None
    weak_count = 0
    if direct_prereqs:
        prereq_ids = [p["id"] for p in direct_prereqs]
        states = await fetch_skill_states_bulk(student_id, prereq_ids)
        known = [s["mastery_probability"] for s in states.values() if s["mastery_probability"] is not None]
        if known:
            weak_count = sum(1 for m in known if m < 0.5)
            prereqs_satisfied = weak_count == 0

    active_recovery = await fetch_active_recovery_for_topic(student_id, topic_id)

    return {
        "prerequisiteCount": len(direct_prereqs),
        "prerequisitesSatisfied": prereqs_satisfied,
        "weakPrerequisiteCount": weak_count,
        "blocked": active_recovery is not None,
        "activeRecoveryRouteTopicId": str(active_recovery["route_topic_id"]) if active_recovery else None,
    }


async def fuse_skill_state(
    student_id: str, topic_id: str, *, excluded_provenances: frozenset[str] = frozenset()
) -> None:
    """Fuses every not-yet-consumed evidence row for (student, topic) into
    one students.skill_states estimate. A no-op (does not create a row)
    when there is no unconsumed evidence -- cold start is "row doesn't
    exist," never a fabricated 0.0 (spec section 15 / Design Principles).

    excluded_provenances (EG-GCKT Milestone 10): drops evidence from the
    named sources before fusing -- the ablation switch
    app/services/evaluation/harness.py's baseline comparison uses to
    measure each engine's incremental contribution (spec section 17.1's
    "EG-GCKT without fusion"/single-engine-only baselines). Empty by
    default so normal production fusion is unaffected."""
    evidence = await fetch_unconsumed_evidence(student_id, topic_id)
    if excluded_provenances:
        evidence = [row for row in evidence if row["provenance"] not in excluded_provenances]
    if not evidence:
        return

    weights: list[float] = []
    for row in evidence:
        weights.append(_source_weight(row["provenance"], row["reliability"], row["sample_size"], row["created_at"]))

    estimated = [(row, w) for row, w in zip(evidence, weights) if row["estimate"] is not None]
    if not estimated:
        # Every unconsumed row carries no point estimate (shouldn't happen
        # for bkt/dina, but a future evidence source might legitimately
        # contribute only structural/contextual signal) -- mark consumed
        # so we don't spin on it forever, but don't fabricate a mastery
        # value from nothing.
        await mark_evidence_consumed([str(r["id"]) for r in evidence])
        return

    total_weight = sum(w for _, w in estimated)
    if total_weight <= 0:
        await mark_evidence_consumed([str(r["id"]) for r in evidence])
        return

    fused_mastery = sum(float(r["estimate"]) * w for r, w in estimated) / total_weight

    # Uncertainty: weighted average of each source's own uncertainty, PLUS
    # a disagreement term -- sources that disagree with each other about
    # the same skill should widen the fused uncertainty rather than let a
    # naive average quietly cancel the disagreement out (spec section 20:
    # "Model disagreement -> Evidence fusion with provenance rather than
    # arbitrary selection").
    known_uncertainties = [(float(r["uncertainty"]), w) for r, w in estimated if r["uncertainty"] is not None]
    base_uncertainty = (
        sum(u * w for u, w in known_uncertainties) / sum(w for _, w in known_uncertainties)
        if known_uncertainties
        else 0.5
    )
    disagreement = math.sqrt(
        sum(w * (float(r["estimate"]) - fused_mastery) ** 2 for r, w in estimated) / total_weight
    )
    fused_uncertainty = min(1.0, base_uncertainty + disagreement)

    evidence_quality = sum(w for _, w in estimated) / len(estimated) if estimated else None
    evidence_quality = min(1.0, evidence_quality) if evidence_quality is not None else None

    # asyncpg returns JSONB as text by default (see postgres_gckt.py) --
    # parse defensively rather than assuming a shape, since `context` is
    # producer-defined free-form JSON, not a fixed schema.
    def _was_correct(ctx: Any) -> Optional[bool]:
        try:
            parsed = json.loads(ctx) if isinstance(ctx, str) else ctx
            return bool(parsed.get("correct")) if isinstance(parsed, dict) and "correct" in parsed else None
        except (TypeError, ValueError):
            return None

    correctness_flags = [f for f in (_was_correct(r["context"]) for r in evidence[-5:]) if f is not None]
    recent_performance = (sum(1 for f in correctness_flags if f) / len(correctness_flags)) if correctness_flags else None

    prior = await fetch_skill_state(student_id, topic_id)
    previous_mastery = float(prior["mastery_probability"]) if prior and prior["mastery_probability"] is not None else None
    trend = _trend(fused_mastery, previous_mastery)
    cumulative_evidence_count = (prior["evidence_count"] if prior else 0) + len(evidence)

    learning_velocity = None
    if previous_mastery is not None and prior["computed_at"] is not None:
        days_elapsed = max(1.0 / 24, (datetime.now(timezone.utc) - prior["computed_at"]).total_seconds() / 86400.0)
        learning_velocity = (fused_mastery - previous_mastery) / days_elapsed

    last_seen = max(r["created_at"] for r in evidence)
    days_since_seen = max(0.0, (datetime.now(timezone.utc) - last_seen).total_seconds() / 86400.0)
    # Simple exponential forgetting-risk proxy: negligible right after new
    # evidence, rising with elapsed time. A real forgetting-curve model
    # (skill-specific decay rates) is future work; this is an honestly
    # documented heuristic, not a fitted psychological model.
    forgetting_risk = min(1.0, 1 - math.exp(-days_since_seen / 14.0))

    provenance_counts: dict[str, int] = {}
    for r in evidence:
        provenance_counts[r["provenance"]] = provenance_counts.get(r["provenance"], 0) + 1

    school_id = await fetch_school_id_for_student(student_id)
    if school_id is None:
        logger.warning("fusion.no_school_for_student", student_id=student_id)
        return

    fusion_policy_snapshot_id = await _get_fusion_policy_snapshot_id()
    structural_status = await _compute_structural_status(student_id, topic_id)

    await upsert_skill_state(
        student_id,
        school_id,
        topic_id,
        mastery_probability=fused_mastery,
        mastery_status=_mastery_status(fused_mastery),
        uncertainty=fused_uncertainty,
        evidence_count=cumulative_evidence_count,
        evidence_quality=evidence_quality,
        recent_performance=recent_performance,
        trend=trend,
        learning_velocity=learning_velocity,
        forgetting_risk=forgetting_risk,
        last_seen=last_seen,
        next_item_success=fused_mastery,  # simple proxy until IRT-based prediction (Milestone 8) supersedes it
        diagnostic_provenance={"sources": provenance_counts, "evidenceCount": len(evidence)},
        model_snapshot_id=fusion_policy_snapshot_id,
        structural_status=structural_status,
    )

    await mark_evidence_consumed([str(r["id"]) for r in evidence])

    synced = await sync_skill_state(
        student_id,
        topic_id,
        mastery_probability=fused_mastery,
        mastery_status=_mastery_status(fused_mastery),
        uncertainty=fused_uncertainty,
        trend=trend,
    )
    if synced:
        await mark_skill_state_synced(student_id, topic_id)

    logger.info(
        "fusion.fuse_skill_state.done",
        student_id=student_id,
        topic_id=topic_id,
        mastery=round(fused_mastery, 3),
        uncertainty=round(fused_uncertainty, 3),
        evidence_count=len(evidence),
    )
