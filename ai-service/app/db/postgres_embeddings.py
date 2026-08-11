"""
Postgres access for embeddings.clo_embeddings / embeddings.question_embeddings
(pgvector) -- Capability 2.1's semantic exam-to-CLO matching.

Vectors are passed as pgvector text literals ("[0.1,0.2,...]") cast to
::vector in SQL rather than via the pgvector Python package's asyncpg
codec registration -- this avoids changing get_pool()'s shared connection
init behavior in app/db/postgres.py just for this one feature. See
backend/internal/curriculum/repository/embeddings.go's vectorLiteral for
the equivalent on the Go side.
"""

from __future__ import annotations

from typing import Optional

import asyncpg

from app.db.postgres import get_pool


def _vector_literal(embedding: list[float]) -> str:
    return "[" + ",".join(repr(float(x)) for x in embedding) + "]"


async def has_embeddings_for_subject(subject_code: str) -> bool:
    """
    Cheap existence check the matcher uses to decide whether it's even
    worth doing a vector search for this subject -- a subject whose
    curriculum was approved before Capability 2.1 shipped (or whose
    ai-service embedding call failed at approval time, see
    curriculum/service.go's syncCLOEmbeddings) has zero rows here, and a
    similarity search against zero CLOs would just waste a round trip
    before falling through to the keyword/LLM matcher anyway.
    """
    pool = await get_pool()
    return await pool.fetchval(
        """
        SELECT EXISTS(
            SELECT 1 FROM embeddings.clo_embeddings ce
            JOIN curriculum.clos c ON c.code = ce.clo_code
            WHERE c.subject_code = $1
        )
        """,
        subject_code,
    )


async def find_best_matching_clo(subject_code: str, embedding: list[float]) -> Optional[asyncpg.Record]:
    """
    Nearest CLO (by cosine similarity) within `subject_code` to the given
    embedding -- the core of exam-to-curriculum vector matching. pgvector's
    <=> operator returns cosine *distance* (0 = identical, 2 = opposite);
    we return 1 - distance as `score` so callers compare against the PRD's
    similarity thresholds (>=0.85 / 0.65-0.84 / <0.65) directly.
    """
    pool = await get_pool()
    return await pool.fetchrow(
        """
        SELECT c.code, 1 - (ce.embedding <=> $2::vector) AS score
        FROM embeddings.clo_embeddings ce
        JOIN curriculum.clos c ON c.code = ce.clo_code
        WHERE c.subject_code = $1
        ORDER BY ce.embedding <=> $2::vector ASC
        LIMIT 1
        """,
        subject_code,
        _vector_literal(embedding),
    )


async def upsert_question_embedding(question_id: str, embedding: list[float], model_ver: str) -> None:
    """Stores a question's embedding for audit/debugging and so a future
    re-matching pass (e.g. after a curriculum revision) doesn't need to
    recompute it. Not currently read back by anything -- the match itself
    happens in the same request via find_best_matching_clo."""
    pool = await get_pool()
    await pool.execute(
        """
        INSERT INTO embeddings.question_embeddings (question_id, embedding, model_ver)
        VALUES ($1, $2::vector, $3)
        ON CONFLICT (question_id) DO UPDATE SET
            embedding = EXCLUDED.embedding,
            model_ver = EXCLUDED.model_ver,
            created_at = now()
        """,
        question_id,
        _vector_literal(embedding),
        model_ver,
    )
