"""
Postgres access layer for the curriculum-parsing worker.

Two tables matter here:
  - curriculum.upload_jobs  -- job metadata + status + the parsed_structure
                               JSONB column this worker fills in (see
                               migration V012).
  - app_storage.local_files -- where the raw uploaded bytes live in dev
                               mode (see pkg/storage/postgres.go on the Go
                               side). upload_jobs.file_s3_key holds the id
                               of the row in this table. Schema is
                               app_storage, not storage -- that name
                               collides with Supabase's own Storage-product
                               schema (see migration V027).
"""

from __future__ import annotations

import json
from typing import Any, Optional

import asyncpg

from app.core.config import settings

_pool: Optional[asyncpg.Pool] = None


async def get_pool() -> asyncpg.Pool:
    """Lazily creates (and reuses) a single connection pool for the process."""
    global _pool
    if _pool is None:
        _pool = await asyncpg.create_pool(
            dsn=settings.POSTGRES_DSN,
            min_size=1,
            max_size=max(5, settings.POSTGRES_MAX_CONNS // 4),
        )
    return _pool


async def close_pool() -> None:
    global _pool
    if _pool is not None:
        await _pool.close()
        _pool = None


async def fetch_job(job_id: str) -> Optional[asyncpg.Record]:
    """Fetch the upload_jobs row a queued job id points to."""
    pool = await get_pool()
    return await pool.fetchrow(
        """
        SELECT id, uploaded_by, subject_code, grade_level, academic_year,
               file_s3_key, file_name, file_size_bytes, status
        FROM curriculum.upload_jobs
        WHERE id = $1
        """,
        job_id,
    )


async def fetch_file_bytes(file_ref: str) -> bytes:
    """
    Download the raw file bytes for a job's file_s3_key.

    In dev mode, file_s3_key is actually the UUID of a row in
    app_storage.local_files (see pkg/storage/postgres.go's PostgresStorage).
    When the app moves to S3 in production, this function is the only
    place that needs to change -- swap it for an S3 GetObject call using
    the same file_ref as the S3 key.
    """
    pool = await get_pool()
    row = await pool.fetchrow(
        "SELECT file_data FROM app_storage.local_files WHERE id = $1",
        file_ref,
    )
    if row is None:
        raise FileNotFoundError(f"no app_storage.local_files row for id={file_ref}")
    return bytes(row["file_data"])


async def mark_parsing(job_id: str) -> None:
    """Flip the job to 'parsing' as soon as the worker picks it up."""
    pool = await get_pool()
    await pool.execute(
        """
        UPDATE curriculum.upload_jobs
        SET status = 'parsing', updated_at = now()
        WHERE id = $1
        """,
        job_id,
    )


async def save_parsed_structure(job_id: str, structure: dict) -> None:
    """Persist the extracted JSON tree and mark the job 'parsed'."""
    pool = await get_pool()
    await pool.execute(
        """
        UPDATE curriculum.upload_jobs
        SET parsed_structure = $2::jsonb,
            status = 'parsed',
            parse_error = NULL,
            updated_at = now()
        WHERE id = $1
        """,
        job_id,
        json.dumps(structure),
    )


async def mark_failed(job_id: str, error: str) -> None:
    """Mark a job 'failed' with the error message, e.g. on a parsing exception."""
    pool = await get_pool()
    await pool.execute(
        """
        UPDATE curriculum.upload_jobs
        SET status = 'failed', parse_error = $2, updated_at = now()
        WHERE id = $1
        """,
        job_id,
        error[:4000],  # keep it sane in case of a huge traceback
    )
