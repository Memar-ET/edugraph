"""
Read/write helpers for the embeddings.* schema (CLO/topic embeddings).

Kept separate from app/db/postgres.py, which is scoped to the curriculum
upload_jobs pipeline -- this module is the data layer for
app/workers/embed_worker.py instead.

asyncpg has no built-in pgvector codec, so vectors are passed as Postgres
vector-literal strings ("[0.1,-0.2,...]") and cast with `::vector` in SQL;
this is the standard asyncpg + pgvector approach and avoids adding a
dependency just for codec registration.
"""

from __future__ import annotations

from typing import Optional

from app.db.postgres import get_pool


async def fetch_topic_text(topic_id: str) -> Optional[str]:
    """Text to embed for a curriculum.topics row, or None if it doesn't exist."""
    pool = await get_pool()
    row = await pool.fetchrow(
        "SELECT title_en, description, key_concepts FROM curriculum.topics WHERE id = $1",
        topic_id,
    )
    if row is None:
        return None
    parts = [row["title_en"]]
    if row["description"]:
        parts.append(row["description"])
    if row["key_concepts"]:
        parts.append(", ".join(row["key_concepts"]))
    return " -- ".join(parts)


async def fetch_clo_text(clo_code: str) -> Optional[str]:
    """Text to embed for a curriculum.clos row, or None if it doesn't exist."""
    pool = await get_pool()
    row = await pool.fetchrow(
        "SELECT description_en, bloom_level FROM curriculum.clos WHERE code = $1",
        clo_code,
    )
    if row is None:
        return None
    if row["bloom_level"]:
        return f"{row['description_en']} (Bloom: {row['bloom_level']})"
    return row["description_en"]


def _to_vector_literal(embedding: list[float]) -> str:
    return "[" + ",".join(f"{v:.8f}" for v in embedding) + "]"


async def upsert_topic_embedding(topic_id: str, embedding: list[float], model_ver: str) -> None:
    pool = await get_pool()
    await pool.execute(
        """
        INSERT INTO embeddings.topic_embeddings (topic_id, embedding, model_ver)
        VALUES ($1, $2::vector, $3)
        ON CONFLICT (topic_id) DO UPDATE SET
            embedding  = EXCLUDED.embedding,
            model_ver  = EXCLUDED.model_ver,
            created_at = now()
        """,
        topic_id,
        _to_vector_literal(embedding),
        model_ver,
    )


async def upsert_clo_embedding(clo_code: str, embedding: list[float], model_ver: str) -> None:
    pool = await get_pool()
    await pool.execute(
        """
        INSERT INTO embeddings.clo_embeddings (clo_code, embedding, model_ver)
        VALUES ($1, $2::vector, $3)
        ON CONFLICT (clo_code) DO UPDATE SET
            embedding  = EXCLUDED.embedding,
            model_ver  = EXCLUDED.model_ver,
            created_at = now()
        """,
        clo_code,
        _to_vector_literal(embedding),
        model_ver,
    )
