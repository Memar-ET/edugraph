"""
Consumes CLO/topic embedding jobs off Redis (feature 1.1: generate and
store a vector embedding for every CLO/topic on curriculum approval).

Mirrors gap_worker.py: the Go backend LPUSHes a small JSON payload
({"kind": "topic", "id": <uuid>} or {"kind": "clo", "code": <code>}) onto
queue:embedding:generate from ApproveAndPromote (curriculum/service/
service.go), once per promoted topic/CLO. Same standalone asyncio BRPOP
consumer loop as the other workers, not Celery -- see app/db/redis.py.

The job payload deliberately carries only an id/code, not the text to
embed -- this worker re-fetches the current title/description from
Postgres (app/db/pgvector.py) so the embedding always reflects the latest
committed row rather than whatever text existed at enqueue time.

    python -m app.workers.embed_worker
"""

from __future__ import annotations

import asyncio
import json

import structlog

from app.db.pgvector import fetch_clo_text, fetch_topic_text, upsert_clo_embedding, upsert_topic_embedding
from app.db.redis import EMBEDDING_QUEUE, brpop_job
from app.utils.embeddings import get_embedding_provider

logger = structlog.get_logger()

_shutdown = asyncio.Event()


async def _process(payload: str) -> None:
    try:
        job = json.loads(payload)
    except ValueError:
        logger.warning("embed_worker.bad_payload", payload=payload)
        return

    kind = job.get("kind")
    provider = get_embedding_provider()

    if kind == "topic":
        topic_id = job.get("id")
        text = await fetch_topic_text(topic_id) if topic_id else None
        if text is None:
            logger.warning("embed_worker.topic_not_found", topic_id=topic_id)
            return
        embedding = await provider.embed(text)
        await upsert_topic_embedding(topic_id, embedding, provider.model_version)
        logger.info("embed_worker.topic_embedded", topic_id=topic_id, model=provider.model_version)
    elif kind == "clo":
        clo_code = job.get("code")
        text = await fetch_clo_text(clo_code) if clo_code else None
        if text is None:
            logger.warning("embed_worker.clo_not_found", clo_code=clo_code)
            return
        embedding = await provider.embed(text)
        await upsert_clo_embedding(clo_code, embedding, provider.model_version)
        logger.info("embed_worker.clo_embedded", clo_code=clo_code, model=provider.model_version)
    else:
        logger.warning("embed_worker.unknown_kind", kind=kind)


async def run_forever(poll_timeout: int = 5) -> None:
    logger.info("embed_worker.started", queue=EMBEDDING_QUEUE)
    while not _shutdown.is_set():
        try:
            payload = await brpop_job(EMBEDDING_QUEUE, timeout=poll_timeout)
        except Exception:  # noqa: BLE001 -- keep the loop alive on transient Redis errors
            logger.exception("embed_worker.redis_error")
            await asyncio.sleep(poll_timeout)
            continue

        if payload is None:
            continue  # BRPOP timed out with nothing queued -- loop and check shutdown

        try:
            await _process(payload)
        except Exception:  # noqa: BLE001 -- a single bad job must not kill the worker
            logger.exception("embed_worker.unhandled_job_error", payload=payload)


def request_shutdown() -> None:
    _shutdown.set()


if __name__ == "__main__":
    try:
        asyncio.run(run_forever())
    except KeyboardInterrupt:
        pass
