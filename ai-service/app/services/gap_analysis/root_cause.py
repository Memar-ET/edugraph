"""
Prerequisite-aware root-cause diagnosis (EG-GCKT Milestone 5, spec section
9). Replaces gap_analysis's original Pass 2 logic ("pick the deepest weak
prerequisite in the chain") with the spec's Root Cause Score:

    RCS(k) = Weakness(k) x EvidenceConfidence(k) x DownstreamImpact(k)
             x PrerequisiteReadiness(k) x InterventionGain(k)

This reproduces the spec's own worked example (section 9.1): given
Multiplication=0.91, Fractions=0.34, Ratios=0.22, Percentages=0.18 along
the chain Multiplication -> Fractions -> Ratios -> Percentages, RCS
correctly favors Fractions over the numerically-weaker Ratios/Percentages
-- their PrerequisiteReadiness is gated by Fractions itself still being
weak, so an intervention on Percentages before fixing Fractions would be
ineffective and scores low, despite Percentages having lower raw mastery.
"""

from __future__ import annotations

from typing import Optional

import structlog

from app.db import neo4j as neo4j_db
from app.db import postgres_gap as db
from app.db.postgres_gckt import fetch_skill_states_bulk, insert_evidence

logger = structlog.get_logger()

# Graph-reasoning evidence (produce_graph_evidence, below) is always
# treated as a weak, indirect signal -- never as confident as a direct
# response observation -- regardless of how much prerequisite mastery
# data backs it.
GRAPH_EVIDENCE_UNCERTAINTY = 0.6

# Matches gap_analysis.service's existing WEAK_THRESHOLD -- kept as one
# shared value rather than a second, slightly different magic number.
WEAK_THRESHOLD = 0.5

# Used only when a topic has legacy ratio-based mastery but no fused
# skill_states row yet -- a raw marks ratio carries no uncertainty
# estimate of its own, so this is a fixed, documented stand-in rather
# than a fabricated precise number.
DEFAULT_EVIDENCE_CONFIDENCE = 0.5


async def _mastery_and_confidence(
    student_id: str, topic_ids: list[str], legacy_mastery: dict[str, float]
) -> dict[str, dict]:
    """Per-topic {mastery, confidence}, preferring students.skill_states
    (real fused mastery_probability + uncertainty, Milestone 4) and
    falling back to the legacy marks-ratio (gap_analysis's original
    fetch_topic_mastery) for topics fusion hasn't touched yet -- e.g.
    gap_analysis and knowledge_tracing can race on the same attempt, or a
    topic was only ever assessed via an item with no Q-matrix mapping.
    Topics with no evidence in EITHER source are simply absent from the
    result -- "no evidence is not weakness," never fabricated."""
    fused = await fetch_skill_states_bulk(student_id, topic_ids)
    out: dict[str, dict] = {}
    for tid in topic_ids:
        row = fused.get(tid)
        if row and row["mastery_probability"] is not None:
            uncertainty = row["uncertainty"] if row["uncertainty"] is not None else 0.5
            out[tid] = {"mastery": row["mastery_probability"], "confidence": 1.0 - uncertainty}
        elif tid in legacy_mastery:
            out[tid] = {"mastery": legacy_mastery[tid], "confidence": DEFAULT_EVIDENCE_CONFIDENCE}
    return out


async def downstream_impact(topic_id: str, max_depth: int = 3) -> float:
    """Normalized DownstreamImpact(k): a depth-decayed count of topics
    that depend on k (direct + transitive, requires/strongly_requires
    edges only), squashed into [0, 1) via raw/(1+raw) so it never needs an
    arbitrary hard cap on "how many dependents is a lot"."""
    try:
        dependents = await neo4j_db.fetch_downstream_dependents(topic_id, max_depth)
    except Exception:  # noqa: BLE001 -- graph-service outage (FI-008): degrade to the Postgres mirror
        logger.warning("root_cause.neo4j_unavailable_falling_back", topic_id=topic_id, exc_info=True)
        dependents = []
    if not dependents:
        dependents = await db.fetch_downstream_dependents_pg(topic_id, max_depth)
    raw = sum(0.5 ** (d["depth"] - 1) for d in dependents)
    return raw / (1.0 + raw)


def compute_rcs(weakness: float, confidence: float, impact: float, readiness: float, intervention_gain: float) -> float:
    """The spec's Root Cause Score (section 9), factored out as a pure
    function so it's independently unit-testable against the spec's own
    Fractions worked example without needing a database."""
    return weakness * confidence * impact * readiness * intervention_gain


async def _prerequisite_readiness(student_id: str, topic_id: str, legacy_mastery: dict[str, float]) -> float:
    """Mean mastery of topic_id's OWN direct prerequisites. A topic with
    no prerequisites of its own (a true foundational topic) is treated as
    fully "ready" (1.0) -- there's nothing further upstream gating an
    intervention there. Unknown (no evidence on any direct prerequisite)
    defaults to 0.5 -- neither penalizing nor rewarding a candidate purely
    for lacking prerequisite evidence."""
    try:
        own_prereqs = await neo4j_db.fetch_prerequisite_chain(topic_id, max_depth=1)
    except Exception:  # noqa: BLE001 -- graph-service outage (FI-008): degrade to the Postgres mirror
        logger.warning("root_cause.neo4j_unavailable_falling_back", topic_id=topic_id, exc_info=True)
        own_prereqs = []
    if not own_prereqs:
        own_prereqs = await db.fetch_prerequisite_chain_pg(topic_id, max_depth=1)
    if not own_prereqs:
        return 1.0

    prereq_ids = [p["id"] for p in own_prereqs]
    estimates = await _mastery_and_confidence(student_id, prereq_ids, legacy_mastery)
    values = [estimates[p]["mastery"] for p in prereq_ids if p in estimates]
    return (sum(values) / len(values)) if values else 0.5


async def score_candidates(
    student_id: str,
    chain: list[dict],
    legacy_mastery: dict[str, float],
) -> Optional[dict]:
    """Given one symptom topic's prerequisite chain (as returned by
    fetch_prerequisite_chain[_pg]: [{id, title, grade_level, depth}, ...]),
    scores every sufficiently-weak candidate in it and returns the
    highest-RCS one -- or None if nothing in the chain has evidence of
    being weak enough to be a credible root cause."""
    if not chain:
        return None

    all_ids = [c["id"] for c in chain]
    estimates = await _mastery_and_confidence(student_id, all_ids, legacy_mastery)

    candidates = [c for c in chain if c["id"] in estimates and estimates[c["id"]]["mastery"] < WEAK_THRESHOLD]
    if not candidates:
        return None

    scored = []
    for c in candidates:
        tid = c["id"]
        mastery = estimates[tid]["mastery"]
        confidence = estimates[tid]["confidence"]
        weakness = max(0.0, WEAK_THRESHOLD - mastery) / WEAK_THRESHOLD

        impact = await downstream_impact(tid)
        readiness = await _prerequisite_readiness(student_id, tid, legacy_mastery)
        intervention_gain = impact * weakness

        rcs = compute_rcs(weakness, confidence, impact, readiness, intervention_gain)
        scored.append(
            {
                **c,
                "mastery": mastery,
                "rcs": rcs,
                "factors": {
                    "weakness": weakness,
                    "evidenceConfidence": confidence,
                    "downstreamImpact": impact,
                    "prerequisiteReadiness": readiness,
                    "interventionGain": intervention_gain,
                },
            }
        )

    scored.sort(key=lambda s: -s["rcs"])
    winner = scored[0]
    # The path from the symptom down to the chosen root cause: every
    # candidate node at or above the winner's depth (spec section 10's
    # "expose the graph path connecting root cause to downstream
    # weaknesses") -- shallow (closer to the symptom) first.
    winner["path"] = sorted(
        [c for c in chain if c["depth"] <= winner["depth"]], key=lambda c: c["depth"]
    )
    return winner


async def produce_graph_evidence(student_id: str, topic_id: str) -> Optional[str]:
    """The GCSF evidence source the spec calls "graph reasoning ->
    prerequisite reachability, dependency influence, structural
    consistency -> structural evidence and diagnostic signals" (Model at
    a Glance table). Estimates this topic's likely mastery from its
    direct hard-dependency prerequisites' CURRENT fused mastery -- a
    weak, indirect inference (hence GRAPH_EVIDENCE_UNCERTAINTY and
    fusion.py's low DEFAULT_SOURCE_RELIABILITY["graph_reasoning"] weight;
    the critical safeguard from spec section 8.3 is that this can only
    ever nudge the fused estimate, never override direct evidence).
    Called once per topic right before fusion (see kt_worker.py) so
    there's always at least a structural prior available to fuse
    alongside BKT/DINA/IRT, not just an unused reliability constant with
    nothing ever produced under that provenance. Returns the inserted
    evidence_log row id, or None when there's no prerequisite or no
    evidence on any of them yet (nothing to infer from)."""
    try:
        own_prereqs = await neo4j_db.fetch_prerequisite_chain(topic_id, max_depth=1)
    except Exception:  # noqa: BLE001 -- graph-service outage (FI-008): degrade to the Postgres mirror
        logger.warning("root_cause.neo4j_unavailable_falling_back", topic_id=topic_id, exc_info=True)
        own_prereqs = []
    if not own_prereqs:
        own_prereqs = await db.fetch_prerequisite_chain_pg(topic_id, max_depth=1)
    if not own_prereqs:
        return None

    prereq_ids = [p["id"] for p in own_prereqs]
    states = await fetch_skill_states_bulk(student_id, prereq_ids)
    values = [s["mastery_probability"] for s in states.values() if s["mastery_probability"] is not None]
    if not values:
        return None

    estimate = sum(values) / len(values)
    return await insert_evidence(
        student_id,
        topic_id,
        estimate=estimate,
        uncertainty=GRAPH_EVIDENCE_UNCERTAINTY,
        sample_size=len(values),
        reliability=None,  # let fusion.py's DEFAULT_SOURCE_RELIABILITY["graph_reasoning"] apply
        provenance="graph_reasoning",
        context={"prerequisiteCount": len(values), "method": "mean_direct_prerequisite_mastery"},
    )
