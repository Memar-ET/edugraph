"""
Prerequisite consistency checking / active verification (EG-GCKT
Milestone 5, spec section 10).

    "When the graph says A -> B -> C but the estimated learner state
    suggests A is weak while B and C are strong, the system should flag a
    consistency issue rather than force the state into graph conformity."

Read against this schema's edge direction (topic_id "requires"
prerequisite_id, so a chain walked via fetch_prerequisite_chain runs from
a topic to its most foundational ancestor), the spec's "A -> B -> C" is a
topic and its prerequisite: A is the foundational one, and the pattern to
detect is "a topic (D) shows strong mastery while one of its OWN DIRECT
prerequisites (W) shows weak mastery, and both estimates are well
evidenced" -- the graph claims D depends on W, but the observed state
doesn't support that dependency having actually mattered for this
student.

Scope decision (documented in the implementation plan): steps 1-2
(detect, classify sparse-evidence-vs-real) and steps 5-6 (record the
outcome) are implemented here, running right after fusion produces a new
skill_states estimate (see kt_worker.py). Steps 3-4 (select a
high-information diagnostic item, deliver it, observe the response) are
NOT automated in this pass -- a full closed-loop adaptive-item-delivery
pipeline is a separate, large UI/testing project. Instead, a confirmed
(non-sparse-evidence) inconsistency lowers the involved prerequisite
edge's confidence (the column Milestone 0 added) and logs the reason to
curriculum.prerequisite_review_history -- surfaced to a human (teacher/
curriculum officer) through the existing prerequisite validation UI. This
is the real "graph becomes an active diagnostic instrument" loop the spec
describes, just teacher-mediated rather than fully automated for v1.
"""

from __future__ import annotations

import structlog

from app.db import neo4j as neo4j_db
from app.db import postgres_gap as db
from app.db.postgres import get_pool
from app.db.postgres_gckt import fetch_skill_states_bulk

logger = structlog.get_logger()

STRONG_THRESHOLD = 0.7
WEAK_THRESHOLD = 0.5
# Below this evidence_count, or above MAX_UNCERTAINTY_FOR_FLAG, on EITHER
# side of the pair -- treat as sparse-evidence-explainable (spec step 2),
# not a real inconsistency, and don't touch the edge's confidence.
MIN_EVIDENCE_FOR_FLAG = 3
MAX_UNCERTAINTY_FOR_FLAG = 0.4
CONFIDENCE_PENALTY = 0.15


def _is_sparse(state: dict) -> bool:
    return state["evidence_count"] < MIN_EVIDENCE_FOR_FLAG or (state["uncertainty"] or 1.0) > MAX_UNCERTAINTY_FOR_FLAG


async def _penalize_edge_confidence(topic_id: str, prereq_id: str, dependent_mastery: float, prereq_mastery: float) -> None:
    """Steps 5-6: record the result as evidence for future inference.
    Lowers the edge's confidence (floored at 0) and logs an append-only
    review-history row explaining why -- never deletes or auto-rejects
    the edge itself, since a single student's unusual path is evidence to
    weigh, not proof the edge is wrong (spec's critical safeguard,
    section 8.3: observed evidence informs, it doesn't get silently
    overridden by -- or silently override -- the graph)."""
    pool = await get_pool()
    row = await pool.fetchrow(
        """
        UPDATE curriculum.topic_prerequisites
        SET confidence = GREATEST(0, COALESCE(confidence, 0.5) - $3), neo4j_written = FALSE
        WHERE topic_id = $1 AND prerequisite_id = $2 AND edge_type IN ('requires', 'strongly_requires')
        RETURNING id, confidence
        """,
        topic_id,
        prereq_id,
        CONFIDENCE_PENALTY,
    )
    if row is None:
        return
    await pool.execute(
        """
        INSERT INTO curriculum.prerequisite_review_history (prerequisite_edge_id, action, new_values, notes)
        VALUES ($1, 'confidence_changed', jsonb_build_object('confidence', $2::float8),
                $3)
        """,
        row["id"],
        float(row["confidence"]),
        f"EG-GCKT consistency check: dependent topic mastery={dependent_mastery:.2f} while "
        f"prerequisite mastery={prereq_mastery:.2f}, both well-evidenced -- confidence lowered "
        "pending human review.",
    )


async def check_topic(student_id: str, topic_id: str) -> int:
    """Checks whether `topic_id`, given it now shows strong fused mastery,
    has any direct prerequisite the student is evidenced-weak in. Returns
    how many edges were flagged (penalized) -- 0 is the normal case."""
    states = await fetch_skill_states_bulk(student_id, [topic_id])
    dependent_state = states.get(topic_id)
    if not dependent_state or dependent_state["mastery_probability"] is None:
        return 0
    if dependent_state["mastery_probability"] < STRONG_THRESHOLD:
        return 0  # this topic isn't strong -- nothing to check
    if _is_sparse(dependent_state):
        return 0  # not enough evidence on the dependent side to trust the "strong" reading

    try:
        direct_prereqs = await neo4j_db.fetch_prerequisite_chain(topic_id, max_depth=1)
    except Exception:  # noqa: BLE001 -- graph-service outage (FI-008): degrade to the Postgres mirror rather than let a dependency failure abort the check
        logger.warning("consistency.neo4j_unavailable_falling_back", topic_id=topic_id, exc_info=True)
        direct_prereqs = []
    if not direct_prereqs:
        direct_prereqs = await db.fetch_prerequisite_chain_pg(topic_id, max_depth=1)
    if not direct_prereqs:
        return 0

    prereq_ids = [p["id"] for p in direct_prereqs]
    prereq_states = await fetch_skill_states_bulk(student_id, prereq_ids)

    flagged = 0
    for prereq_id in prereq_ids:
        state = prereq_states.get(prereq_id)
        if not state or state["mastery_probability"] is None:
            continue  # no evidence on the prerequisite -- "no evidence is not weakness"
        if state["mastery_probability"] >= WEAK_THRESHOLD:
            continue  # prerequisite isn't weak -- consistent, nothing to flag
        if _is_sparse(state):
            continue  # sparse-evidence-explainable (spec step 2) -- not a real inconsistency

        await _penalize_edge_confidence(
            topic_id, prereq_id, dependent_state["mastery_probability"], state["mastery_probability"]
        )
        flagged += 1
        logger.info(
            "consistency.flagged",
            student_id=student_id,
            dependent_topic_id=topic_id,
            prerequisite_topic_id=prereq_id,
            dependent_mastery=round(dependent_state["mastery_probability"], 3),
            prerequisite_mastery=round(state["mastery_probability"], 3),
        )

    return flagged
