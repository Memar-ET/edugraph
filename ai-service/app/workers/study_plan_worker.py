"""
Consumes study-plan generation jobs off Redis (Capability 3B).

Same BRPOP loop as the other workers; like the answer-key queue the
payload is a small JSON object, not a bare id -- {"studentId",
"schoolId", "targetExamId", "language"} -- pushed by the Go side's
POST /students/me/study-plans. Standalone:

    python -m app.workers.study_plan_worker
"""

from __future__ import annotations

import asyncio
import json

import structlog

from app.db.redis import STUDYPLAN_QUEUE, brpop_job
from app.services.study_plan.service import process_study_plan_job

logger = structlog.get_logger()

_shutdown = asyncio.Event()


async def run_forever(poll_timeout: int = 5) -> None:
    logger.info("study_plan_worker.started", queue=STUDYPLAN_QUEUE)
    while not _shutdown.is_set():
        try:
            raw = await brpop_job(STUDYPLAN_QUEUE, timeout=poll_timeout)
        except Exception:  # noqa: BLE001 -- keep the loop alive on transient Redis errors
            logger.exception("study_plan_worker.redis_error")
            await asyncio.sleep(poll_timeout)
            continue

        if raw is None:
            continue

        try:
            payload = json.loads(raw)
        except json.JSONDecodeError:
            logger.error("study_plan_worker.bad_payload", raw=raw[:200])
            continue

        try:
            await process_study_plan_job(payload)
        except Exception:  # noqa: BLE001 -- a single bad job must not kill the worker
            logger.exception("study_plan_worker.unhandled_job_error", payload=payload)


def request_shutdown() -> None:
    _shutdown.set()


if __name__ == "__main__":
    try:
        asyncio.run(run_forever())
    except KeyboardInterrupt:
        pass
