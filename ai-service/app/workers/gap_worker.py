"""
Consumes gap-analysis jobs off Redis (Capability 3A).

Mirrors exam_worker.py exactly: the Go backend LPUSHes plain attempt-id
strings onto queue:gap:analyze whenever an exam attempt becomes fully
graded (service/exam_submit.go), so this is the same small standalone
asyncio BRPOP consumer loop, not a Celery task. Can run in-process (the
default, started from app/main.py's lifespan) or standalone:

    python -m app.workers.gap_worker
"""

from __future__ import annotations

import asyncio

import structlog

from app.core.config import settings
from app.db.dead_letter import push_dead_letter
from app.db.redis import GAP_ANALYZE_QUEUE, brpop_job
from app.services.gap_analysis.service import process_gap_job

logger = structlog.get_logger()

_shutdown = asyncio.Event()


async def _consume_one(poll_timeout: int) -> None:
    try:
        attempt_id = await brpop_job(GAP_ANALYZE_QUEUE, timeout=poll_timeout)
    except Exception:  # noqa: BLE001 -- keep the loop alive on transient Redis errors
        logger.exception("gap_worker.redis_error")
        await asyncio.sleep(poll_timeout)
        return

    if attempt_id is None:
        return  # BRPOP timed out with nothing queued -- loop and check shutdown

    try:
        await process_gap_job(attempt_id)
    except Exception as exc:  # noqa: BLE001 -- a single bad job must not kill the worker
        logger.exception("gap_worker.unhandled_job_error", attempt_id=attempt_id)
        await push_dead_letter(GAP_ANALYZE_QUEUE, attempt_id, str(exc))


async def _consumer_loop(poll_timeout: int) -> None:
    while not _shutdown.is_set():
        await _consume_one(poll_timeout)


async def run_forever(poll_timeout: int = 5) -> None:
    concurrency = max(1, settings.GAP_WORKER_CONCURRENCY)
    logger.info("gap_worker.started", queue=GAP_ANALYZE_QUEUE, concurrency=concurrency)
    await asyncio.gather(*(_consumer_loop(poll_timeout) for _ in range(concurrency)))


def request_shutdown() -> None:
    _shutdown.set()


if __name__ == "__main__":
    try:
        asyncio.run(run_forever())
    except KeyboardInterrupt:
        pass
