"""
Consumes answer-key upload jobs off Redis (Capability 2C).

Mirrors curriculum_worker.py/exam_worker.py's BRPOP consumer loop, with one
difference: the Go side pushes a small JSON payload
({"examId": ..., "fileRef": ...}) instead of a bare id string, since this
job needs both which exam to apply the key to and where the uploaded
file's bytes live -- brpop_job() itself doesn't care, it just returns
whatever string was pushed; this loop is the only place that parses it.
"""

from __future__ import annotations

import asyncio
import json

import structlog

from app.db.redis import EXAM_ANSWERKEY_QUEUE, brpop_job
from app.services.exam_parser.answer_key_job import process_answer_key_job

logger = structlog.get_logger()

_shutdown = asyncio.Event()


async def run_forever(poll_timeout: int = 5) -> None:
    logger.info("answer_key_worker.started", queue=EXAM_ANSWERKEY_QUEUE)
    while not _shutdown.is_set():
        try:
            payload = await brpop_job(EXAM_ANSWERKEY_QUEUE, timeout=poll_timeout)
        except Exception:  # noqa: BLE001 -- keep the loop alive on transient Redis errors
            logger.exception("answer_key_worker.redis_error")
            await asyncio.sleep(poll_timeout)
            continue

        if payload is None:
            continue

        try:
            data = json.loads(payload)
            await process_answer_key_job(data["examId"], data["fileRef"])
        except Exception:  # noqa: BLE001 -- a single bad job must not kill the worker
            logger.exception("answer_key_worker.unhandled_job_error", payload=payload)


def request_shutdown() -> None:
    _shutdown.set()


if __name__ == "__main__":
    try:
        asyncio.run(run_forever())
    except KeyboardInterrupt:
        pass
