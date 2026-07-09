"""
Consumes curriculum-upload jobs off Redis and parses them.

The Go backend pushes plain job-id strings onto a Redis list with LPUSH
(queue:curriculum:parse) rather than a Celery-protocol message, so this
is a small standalone asyncio consumer loop (BRPOP) instead of a Celery
task -- see app/db/redis.py for why. It can run two ways:

  1. In-process, as a background task started from app/main.py's FastAPI
     startup event (the default for local/dev docker-compose, since only
     one ai-service container is defined).
  2. As its own process for production, so parsing load is isolated from
     the API and can be scaled independently:

        python -m app.workers.curriculum_worker

Both paths call the same run_forever() loop below.
"""

from __future__ import annotations

import asyncio

import structlog

from app.db.redis import CURRICULUM_PARSE_QUEUE, brpop_job
from app.services.curriculum_parser.service import process_job

logger = structlog.get_logger()

_shutdown = asyncio.Event()


async def run_forever(poll_timeout: int = 5) -> None:
    logger.info("curriculum_worker.started", queue=CURRICULUM_PARSE_QUEUE)
    while not _shutdown.is_set():
        try:
            job_id = await brpop_job(CURRICULUM_PARSE_QUEUE, timeout=poll_timeout)
        except Exception:  # noqa: BLE001 -- keep the loop alive on transient Redis errors
            logger.exception("curriculum_worker.redis_error")
            await asyncio.sleep(poll_timeout)
            continue

        if job_id is None:
            continue  # BRPOP timed out with nothing queued -- loop and check shutdown

        try:
            await process_job(job_id)
        except Exception:  # noqa: BLE001 -- a single bad job must not kill the worker
            logger.exception("curriculum_worker.unhandled_job_error", job_id=job_id)


def request_shutdown() -> None:
    _shutdown.set()


if __name__ == "__main__":
    try:
        asyncio.run(run_forever())
    except KeyboardInterrupt:
        pass
