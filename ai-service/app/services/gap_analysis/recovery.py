"""
Blocked-Learning & Recovery System (EG-GCKT checklist section 13).

Detects repeated failure on the same skill, routes the student to an
alternative representation via the typed similar_to/alternative_to/
related_to prerequisite edges (Milestone 0) plus the topic's own
subtopics as a "lower-granularity" fallback, and tracks whether each
route actually worked -- closing a checklist section that had zero
implementation after the first EG-GCKT pass (the typed edges existed,
nothing consumed them for recovery routing).
"""

from __future__ import annotations

from typing import Any, Optional

import structlog

from app.db import postgres_gckt as db
from app.db.postgres_gckt import fetch_skill_states_bulk

logger = structlog.get_logger()

# "Repeated failure" (spec section 13) -- this many consecutive incorrect
# responses on the same skill, newest-first, triggers recovery mode.
CONSECUTIVE_FAILURES_FOR_BLOCK = 3

# A route topic reaching this mastery, with enough evidence to trust it,
# counts as "evidence of readiness" to return to the original target.
READY_THRESHOLD = 0.6
MIN_EVIDENCE_FOR_VERDICT = 3

# Route-type preference order, closest conceptual match first: an
# analogous representation before a looser association, with
# lower-granularity (subtopics) and related_to as later fallbacks --
# matches the spec's own ordering ("Search similarity graph... Search
# alternative_to... Search related concepts... Use simpler/lower-
# granularity supporting concepts when required").
_ROUTE_PRIORITY = {"similar_to": 0, "alternative_to": 1, "lower_granularity": 2, "related_to": 3}


def is_blocked(recent_correctness: list[bool]) -> bool:
    """recent_correctness is newest-first. True when the most recent
    CONSECUTIVE_FAILURES_FOR_BLOCK responses are all incorrect."""
    if len(recent_correctness) < CONSECUTIVE_FAILURES_FOR_BLOCK:
        return False
    return not any(recent_correctness[:CONSECUTIVE_FAILURES_FOR_BLOCK])


def rank_routes(candidates: list[dict[str, Any]], tried: set[str]) -> list[dict[str, Any]]:
    """Excludes already-tried routes (spec: "apply repetition penalties
    to avoid cycling through ineffective interventions") and orders the
    rest by route-type priority, then by edge weight/confidence."""
    available = [c for c in candidates if c["route_topic_id"] not in tried]
    available.sort(
        key=lambda c: (_ROUTE_PRIORITY.get(c["edge_type"], 9), -(c["confidence"] if c["confidence"] is not None else c["weight"]))
    )
    return available


async def check_and_trigger(student_id: str, school_id: str, topic_id: str) -> Optional[dict[str, Any]]:
    """Called once per topic touched by a graded attempt, after fusion.
    Returns the newly-created recovery attempt's info, or None -- the
    normal case, since most topics never hit repeated failure. Never
    triggers a second concurrent recovery attempt for the same
    (student, topic) pair."""
    existing = await db.fetch_active_recovery_for_topic(student_id, topic_id)
    if existing is not None:
        return None

    recent = await db.fetch_recent_correctness(student_id, topic_id, limit=CONSECUTIVE_FAILURES_FOR_BLOCK)
    if not is_blocked(recent):
        return None

    tried = set(await db.fetch_tried_recovery_routes(student_id, topic_id))
    candidates = await db.fetch_recovery_route_candidates(topic_id)
    ranked = rank_routes(candidates, tried)
    if not ranked:
        logger.info("recovery.no_route_available", student_id=student_id, topic_id=topic_id)
        return None

    chosen = ranked[0]
    attempt_id = await db.insert_recovery_attempt(
        student_id,
        school_id,
        topic_id,
        chosen["route_topic_id"],
        chosen["edge_type"],
        trigger_reason=f"{CONSECUTIVE_FAILURES_FOR_BLOCK} consecutive incorrect responses",
    )
    logger.info(
        "recovery.triggered",
        student_id=student_id,
        blocked_topic_id=topic_id,
        route_topic_id=chosen["route_topic_id"],
        route_edge_type=chosen["edge_type"],
    )
    return {"attempt_id": attempt_id, **chosen}


async def check_progress(student_id: str) -> int:
    """Checks every active recovery attempt for a student against the
    ROUTE topic's current fused state (not the originally-blocked one --
    that's what the student is actively practicing during recovery).
    Strong + well-evidenced mastery on the route marks
    'ready_to_retarget' (spec: "return to the original learning target
    after evidence of readiness"); enough evidence without improvement
    marks 'failed' so the next check_and_trigger call tries a different
    route instead of repeating this one. Returns how many attempts were
    resolved this call -- most calls resolve zero, which is normal."""
    attempts = await db.fetch_active_recovery_attempts(student_id)
    resolved = 0
    for a in attempts:
        route_id = str(a["route_topic_id"])
        state = (await fetch_skill_states_bulk(student_id, [route_id])).get(route_id)
        if not state or state["mastery_probability"] is None:
            continue

        mastery = state["mastery_probability"]
        evidence_count = state["evidence_count"]
        if mastery >= READY_THRESHOLD and evidence_count >= MIN_EVIDENCE_FOR_VERDICT:
            await db.update_recovery_status(
                str(a["id"]),
                "ready_to_retarget",
                notes=f"Route topic reached {mastery:.2f} mastery ({evidence_count} observations) -- ready to retry the original target.",
            )
            resolved += 1
        elif evidence_count >= MIN_EVIDENCE_FOR_VERDICT and mastery < READY_THRESHOLD:
            await db.update_recovery_status(
                str(a["id"]),
                "failed",
                notes=f"Route topic still at {mastery:.2f} mastery after {evidence_count} observations -- trying a different route next time.",
            )
            resolved += 1

    return resolved
