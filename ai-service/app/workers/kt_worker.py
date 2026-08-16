"""
Consumes EG-GCKT knowledge-tracing jobs off Redis (Milestone 1).

Structurally identical to gap_worker.py's BRPOP consumer loop -- the Go
backend LPUSHes the same attempt-id string onto queue:gckt:trace (a
separate list from queue:gap:analyze, pushed at the same
RecomputeAttemptTotals seam) once students.learning_events rows exist for
the attempt. Can run in-process (the default, started from app/main.py's
lifespan) or standalone:

    python -m app.workers.kt_worker

Dispatches each attempt's events to the BKT (Milestone 2), DINA/G-DINA
(Milestone 3), and IRT (Milestone 8) engines, each of which appends
Evidence rows to modeling.evidence_log. Fusion (Milestone 4) then runs
per touched topic, followed by consistency checking (Milestone 5) and
blocked-learning/recovery detection (checklist section 13) -- this
worker's job ends at "the student's fused state was updated and checked,"
not just "evidence was produced."
"""

from __future__ import annotations

import asyncio

import structlog

from app.db.dead_letter import push_dead_letter
from app.db.postgres_gckt import fetch_learning_events_for_attempt
from app.db.redis import GCKT_TRACE_QUEUE, brpop_job
from app.services.gap_analysis import consistency, recovery, root_cause
from app.services.knowledge_tracing import bkt, dina, fusion, irt

logger = structlog.get_logger()

_shutdown = asyncio.Event()


async def process_knowledge_tracing_job(attempt_id: str) -> None:
    events = await fetch_learning_events_for_attempt(attempt_id)
    if not events:
        # Nothing to trace -- e.g. an attempt with zero mapped skills.
        # Not an error: a question with no topic_id and no Q-matrix row
        # produces an empty skill_ids array, which RecordLearningEvents
        # still writes as a row (skill_ids = '{}'), so this is genuinely
        # "no gradeable skill evidence," not a missed write.
        logger.info("kt_worker.no_events", attempt_id=attempt_id)
        return

    await bkt.process_attempt(attempt_id, events)
    await dina.process_attempt(attempt_id, events)
    await irt.process_attempt(attempt_id, events)

    # Fuse immediately after producing evidence, per-student/topic pair
    # touched by this attempt -- keeps skill_states current without
    # waiting for a separate scheduled fusion pass. A future nightly
    # batch (Milestone 9) re-fuses everything anyway as parameters get
    # refit, so an eager per-attempt fusion here is purely a freshness
    # optimization, not the only place fusion ever runs.
    topic_ids = {tid for e in events for tid in (e["skill_ids"] or [])}
    student_id = str(events[0]["student_id"])
    school_id = str(events[0]["school_id"])
    flagged_total = 0
    recovery_triggered = 0
    for topic_id in topic_ids:
        # Structural evidence (checklist section 9's "graph reasoning"
        # evidence source) before fusion, so it's available to be
        # weighed in alongside BKT/DINA/IRT for this pass.
        await root_cause.produce_graph_evidence(student_id, str(topic_id))
        await fusion.fuse_skill_state(student_id, str(topic_id))
        # Milestone 5's active verification loop: check right after this
        # topic's fused mastery is current, since the check depends on a
        # fresh skill_states row for it.
        flagged_total += await consistency.check_topic(student_id, str(topic_id))
        # Blocked-Learning & Recovery (checklist section 13): check for
        # repeated failure on this same topic right after its skill_state
        # (and the raw response history) are current.
        if await recovery.check_and_trigger(student_id, school_id, str(topic_id)):
            recovery_triggered += 1

    # Independent of which topics THIS attempt touched: any of the
    # student's already-active recovery routes may have just improved
    # (the student could be practicing the route topic in a different
    # exam entirely), so check all of them, not just topics from this job.
    recovery_resolved = await recovery.check_progress(student_id)

    logger.info(
        "kt_worker.processed",
        attempt_id=attempt_id,
        events=len(events),
        topics=len(topic_ids),
        consistency_flags=flagged_total,
        recovery_triggered=recovery_triggered,
        recovery_resolved=recovery_resolved,
    )


async def run_forever(poll_timeout: int = 5) -> None:
    logger.info("kt_worker.started", queue=GCKT_TRACE_QUEUE)
    while not _shutdown.is_set():
        try:
            attempt_id = await brpop_job(GCKT_TRACE_QUEUE, timeout=poll_timeout)
        except Exception:  # noqa: BLE001 -- keep the loop alive on transient Redis errors
            logger.exception("kt_worker.redis_error")
            await asyncio.sleep(poll_timeout)
            continue

        if attempt_id is None:
            continue  # BRPOP timed out with nothing queued -- loop and check shutdown

        try:
            await process_knowledge_tracing_job(attempt_id)
        except Exception as exc:  # noqa: BLE001 -- a single bad job must not kill the worker
            logger.exception("kt_worker.unhandled_job_error", attempt_id=attempt_id)
            await push_dead_letter(GCKT_TRACE_QUEUE, attempt_id, str(exc))


def request_shutdown() -> None:
    _shutdown.set()


if __name__ == "__main__":
    try:
        asyncio.run(run_forever())
    except KeyboardInterrupt:
        pass
