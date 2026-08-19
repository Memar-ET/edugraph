"""
Learner-state snapshots (EG-GCKT checklist sections 6/18/22: "support
learner-state snapshots," "support historical state comparison,"
"version learner-state snapshots with source event range").

students.skill_states is a current-value-only table (upserted in place
by fusion.py) -- there is no way to ask "what did we believe about this
student's mastery of this topic three weeks ago" from it alone. This
module takes immutable point-in-time copies into
students.skill_state_snapshots, each stamped with the evidence_log time
range that contributed to it, so a later historical comparison never
changes retroactively even as fusion keeps updating skill_states.

replay.reconstruct_mastery_as_of (a separate module) answers "what would
GCSF believe as of an ARBITRARY timestamp" by replaying evidence_log --
useful for a timestamp no snapshot happens to cover. This module is the
cheaper, coarser-grained complement: periodic (nightly) or on-demand
snapshots of skill_states AS IT ACTUALLY WAS, not a reconstruction.
"""

from __future__ import annotations

import structlog

from app.db.postgres_gckt import (
    fetch_all_skill_state_pairs,
    fetch_evidence_range,
    fetch_skill_state_full,
    insert_skill_state_snapshot,
)

logger = structlog.get_logger()


async def take_snapshot(student_id: str, topic_id: str, reason: str = "nightly") -> str | None:
    """Copies the CURRENT skill_states row into an immutable snapshot.
    Returns the new snapshot's id, or None when there's nothing to
    snapshot yet (no skill_states row, or mastery_probability still NULL
    -- cold start, per spec section 15, is never given a fabricated
    snapshot)."""
    state = await fetch_skill_state_full(student_id, topic_id)
    if state is None or state["mastery_probability"] is None:
        return None

    range_start, range_end = await fetch_evidence_range(student_id, topic_id)
    return await insert_skill_state_snapshot(
        student_id,
        str(state["school_id"]),
        topic_id,
        mastery_probability=float(state["mastery_probability"]),
        mastery_status=state["mastery_status"],
        uncertainty=float(state["uncertainty"]) if state["uncertainty"] is not None else None,
        evidence_count=state["evidence_count"],
        trend=state["trend"],
        model_snapshot_id=str(state["model_snapshot_id"]) if state["model_snapshot_id"] else None,
        snapshot_reason=reason,
        source_event_range_start=range_start,
        source_event_range_end=range_end,
    )


async def take_all_snapshots(reason: str = "nightly") -> int:
    """Snapshots every (student, topic) pair with a current skill_states
    row -- called from refit_worker.py's nightly cycle. Returns how many
    snapshots were actually written (pairs with no mastery estimate yet
    are silently skipped, not an error)."""
    pairs = await fetch_all_skill_state_pairs()
    written = 0
    for student_id, topic_id in pairs:
        if await take_snapshot(student_id, topic_id, reason):
            written += 1
    logger.info("snapshot.take_all_snapshots.done", pairs=len(pairs), written=written, reason=reason)
    return written
