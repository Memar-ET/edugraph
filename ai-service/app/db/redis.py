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
