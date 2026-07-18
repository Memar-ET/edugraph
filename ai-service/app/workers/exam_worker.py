"""
Consumes exam-upload jobs off Redis and parses them (Capability 2A).

Mirrors curriculum_worker.py exactly: the Go backend LPUSHes plain exam-id
strings onto queue:exam:parse, so this is the same small standalone asyncio
BRPOP consumer loop, not a Celery task. Can run in-process (the default,
started from app/main.py's lifespan) or standalone:

    python -m app.workers.exam_worker
"""

from __future__ import annotations

import asyncio

import structlog

from app.db.redis import EXAM_PARSE_QUEUE, brpop_job
from app.services.exam_parser.service import process_exam_job

logger = structlog.get_logger()

_shutdown = asyncio.Event()


async def run_forever(poll_timeout: int = 5) -> None:
    logger.info("exam_worker.started", queue=EXAM_PARSE_QUEUE)
    while not _shutdown.is_set():
        try:
            exam_id = await brpop_job(EXAM_PARSE_QUEUE, timeout=poll_timeout)
        except Exception:  # noqa: BLE001 -- keep the loop alive on transient Redis errors
            logger.exception("exam_worker.redis_error")
            await asyncio.sleep(poll_timeout)
            continue

        if exam_id is None:
            continue  # BRPOP timed out with nothing queued -- loop and check shutdown

        try:
            await process_exam_job(exam_id)
        except Exception:  # noqa: BLE001 -- a single bad job must not kill the worker
            logger.exception("exam_worker.unhandled_job_error", exam_id=exam_id)


def request_shutdown() -> None:
    _shutdown.set()


if __name__ == "__main__":
    try:
        asyncio.run(run_forever())
    except KeyboardInterrupt:
        pass
