"""
Dead-letter queue persistence for failed AI worker jobs.

Every BRPOP worker catches exceptions and previously just logged them,
silently dropping the job. This module persists failures to
public.dead_letter_jobs so they can be inspected and retried.

Retry policy: up to 3 attempts, with a 5-minute minimum gap between
retries. refit_worker.run_once() calls requeue_retriable_jobs() nightly
to push eligible rows back onto their original queues.
"""

from __future__ import annotations

import structlog

from app.db.postgres import get_pool

logger = structlog.get_logger()

MAX_ATTEMPTS = 3
RETRY_INTERVAL_MINUTES = 5


async def push_dead_letter(
    queue_name: str,
    job_payload: str,
    error: str,
    attempt: int = 1,
) -> None:
    """Persist a failed job to public.dead_letter_jobs.

    Idempotent on the (queue_name, job_payload) pair: if the same payload
    has already been dead-lettered, increments attempt_count rather than
    inserting a duplicate row.
    """
    pool = await get_pool()
    try:
        await pool.execute(
            """
            INSERT INTO public.dead_letter_jobs
                (queue_name, job_payload, error_message, attempt_count, last_attempted_at)
            VALUES ($1, $2, $3, $4, NOW())
            ON CONFLICT (queue_name, job_payload)
            DO UPDATE SET
                error_message     = EXCLUDED.error_message,
                attempt_count     = dead_letter_jobs.attempt_count + 1,
                last_attempted_at = NOW()
            """,
            queue_name,
            job_payload,
            error[:4000],
            attempt,
        )
        logger.info(
            "dead_letter.pushed",
            queue=queue_name,
            payload_prefix=job_payload[:64],
            attempt=attempt,
        )
    except Exception:
        # Never let dead-letter persistence crash the worker loop.
        logger.exception("dead_letter.persist_failed", queue=queue_name)


async def fetch_retriable_jobs() -> list[dict]:
    """Return dead-letter rows eligible for retry.

    Eligible = attempt_count < MAX_ATTEMPTS AND last_attempted_at
    is older than RETRY_INTERVAL_MINUTES ago AND not yet resolved.
    """
    pool = await get_pool()
    rows = await pool.fetch(
        """
        SELECT id, queue_name, job_payload, attempt_count
        FROM public.dead_letter_jobs
        WHERE attempt_count < $1
          AND last_attempted_at < NOW() - ($2 * INTERVAL '1 minute')
          AND resolved_at IS NULL
        ORDER BY last_attempted_at
        LIMIT 100
        """,
        MAX_ATTEMPTS,
        RETRY_INTERVAL_MINUTES,
    )
    return [dict(r) for r in rows]


async def mark_resolved(job_id: str) -> None:
    pool = await get_pool()
    await pool.execute(
        "UPDATE public.dead_letter_jobs SET resolved_at = NOW() WHERE id = $1",
        job_id,
    )


async def requeue_retriable_jobs(redis_client) -> int:
    """Push retriable dead-letter jobs back onto their original queues.

    Returns the count of jobs re-queued. Called from refit_worker.run_once().
    """
    jobs = await fetch_retriable_jobs()
    requeued = 0
    for job in jobs:
        try:
            await redis_client.lpush(job["queue_name"], job["job_payload"])
            await mark_resolved(str(job["id"]))
            requeued += 1
            logger.info(
                "dead_letter.requeued",
                queue=job["queue_name"],
                attempt=job["attempt_count"] + 1,
            )
        except Exception:
            logger.exception("dead_letter.requeue_failed", job_id=str(job["id"]))
    return requeued
