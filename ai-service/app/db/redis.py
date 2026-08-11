"""
Redis access for the AI service.

The Go backend pushes a plain job id string onto a Redis list with LPUSH
(see internal/curriculum/service/service.go):

    redis.LPush(ctx, "queue:curriculum:parse", jobID.String())

This is a raw list, not a Celery-formatted task message, so on this side
we consume it with a simple blocking BRPOP loop rather than routing it
through Celery -- see app/workers/curriculum_worker.py.
"""

from __future__ import annotations

from typing import Optional

import redis.asyncio as redis

from app.core.config import settings

CURRICULUM_PARSE_QUEUE = "queue:curriculum:parse"
EXAM_PARSE_QUEUE = "queue:exam:parse"
# Unlike the other two queues, this one's payload is a small JSON object
# ({"examId": ..., "fileRef": ...}), not a bare id string -- a separate
# answer-key upload needs both which exam to apply it to and where its
# file bytes live. See app/workers/answer_key_worker.py.
EXAM_ANSWERKEY_QUEUE = "queue:exam:answerkey"
# Capability 3A: attempt ids pushed by the Go side the moment an exam
# attempt becomes fully graded -- consumed by app/workers/gap_worker.py.
GAP_ANALYZE_QUEUE = "queue:gap:analyze"
# Capability 3B: JSON payload ({"studentId", "schoolId", "targetExamId",
# "language"}) pushed when a student requests a study plan -- consumed by
# app/workers/study_plan_worker.py.
STUDYPLAN_QUEUE = "queue:studyplan:generate"

_client: Optional[redis.Redis] = None


def get_redis() -> redis.Redis:
    global _client
    if _client is None:
        _client = redis.from_url(
            f"redis://{settings.REDIS_URL}"
            if not settings.REDIS_URL.startswith("redis://")
            else settings.REDIS_URL,
            password=settings.REDIS_PASSWORD or None,
            decode_responses=True,
        )
    return _client


async def close_redis() -> None:
    global _client
    if _client is not None:
        await _client.aclose()
        _client = None


async def brpop_job(queue: str = CURRICULUM_PARSE_QUEUE, timeout: int = 5) -> Optional[str]:
    """
    Block for up to `timeout` seconds waiting for a job id on `queue`.
    Returns None on timeout (so the caller's loop can check for shutdown).
    """
    client = get_redis()
    result = await client.brpop([queue], timeout=timeout)
    if result is None:
        return None
    _queue_name, job_id = result
    return job_id
