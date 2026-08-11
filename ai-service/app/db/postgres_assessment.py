"""
Postgres access layer for the exam-parsing worker (Capability 2A).

Mirrors app/db/postgres.py's curriculum functions but targets
assessment.exams / assessment.questions instead of
curriculum.upload_jobs -- kept in a separate module so the two domains'
queries don't get tangled together, while still sharing the one process-wide
connection pool via get_pool()/fetch_file_bytes() from postgres.py.
"""

from __future__ import annotations

import json
from decimal import Decimal
from typing import Optional

import asyncpg

from app.db.postgres import get_pool

SUPPORTED_EXAM_TYPES = ("mcq", "short_answer", "long_answer", "essay", "calculation")


async def fetch_exam(exam_id: str) -> Optional[asyncpg.Record]:
    """Fetch the assessment.exams row a queued exam id points to."""
    pool = await get_pool()
    return await pool.fetchrow(
        """
        SELECT id, school_id, subject_code, grade_level, exam_scope,
               file_s3_key, status
        FROM assessment.exams
        WHERE id = $1
        """,
        exam_id,
    )


async def mark_exam_parsing(exam_id: str) -> None:
    """Flip the exam to 'parsing' as soon as the worker picks it up."""
    pool = await get_pool()
    await pool.execute(
        "UPDATE assessment.exams SET status = 'parsing', updated_at = now() WHERE id = $1",
        exam_id,
    )


async def fetch_valid_clo_codes(subject_code: str) -> set[str]:
    """
    curriculum.clos.code is the exam's clo_code FK target -- an exam's CLO
    bracket annotation can reference a code that's a typo, or that belongs
    to curriculum not yet uploaded/approved, so callers must filter against
    this before inserting rather than let the FK constraint fail the whole
    batch.
    """
    pool = await get_pool()
    rows = await pool.fetch("SELECT code FROM curriculum.clos WHERE subject_code = $1", subject_code)
    return {r["code"] for r in rows}


async def fetch_clos_for_subject(subject_code: str) -> list[asyncpg.Record]:
    """CLOs (code + description_en) for the keyword-based matcher in
    clo_matcher.py, used when a document has no explicit CLO annotation."""
    pool = await get_pool()
    return await pool.fetch(
        "SELECT code, description_en FROM curriculum.clos WHERE subject_code = $1",
        subject_code,
    )


async def match_subject_code(subject_hint: str, grade_level: int) -> Optional[str]:
    """
    Mirrors the Go side's repository.MatchSubjectFromTitle -- finds the
    curriculum.subjects row for gradeLevel whose name_en or code appears in
    subject_hint (e.g. "Biology"). Needed on this side too because the
    metadata table (extracted here, not at Go upload time) is the more
    reliable source once it's available.
    """
    pool = await get_pool()
    rows = await pool.fetch(
        "SELECT code, name_en FROM curriculum.subjects WHERE grade_level = $1", grade_level
    )
    lower_hint = subject_hint.lower()
    for row in rows:
        if row["name_en"].lower() in lower_hint or row["code"].lower() in lower_hint:
            return row["code"]
    return None


async def update_exam_metadata(
    exam_id: str,
    subject_code: str,
    grade_level: int,
    exam_scope: str,
    total_marks: int,
    unit_numbers: Optional[list[int]] = None,
) -> None:
    """Corrects the exam row with metadata-table-derived values once
    they're fully resolved (see metadata.py) -- the title-derived values
    from Go's upload-time validation are a first-pass fallback, this is the
    authoritative source when the document actually has the table.
    unit_numbers uses COALESCE: when the table's "Exam Type" cell doesn't
    mention a unit range, keep whatever Go's title-derived fallback already
    wrote rather than clobber it with NULL."""
    pool = await get_pool()
    await pool.execute(
        """
        UPDATE assessment.exams
        SET subject_code = $2, grade_level = $3, exam_scope = $4, total_marks = $5,
            unit_numbers = COALESCE($6::int[], unit_numbers), updated_at = now()
        WHERE id = $1
        """,
        exam_id, subject_code, grade_level, exam_scope, total_marks, unit_numbers,
    )


def _clo_align_method(q: dict) -> Optional[str]:
    """
    Distinguishes how clo_code was determined: an explicit [CLO: ...]
    annotation in the source document (no score -- the exam author already
    specified it, closer to a confirmed link), the Gemini-backed matcher
    ("llm_verified", set explicitly by service.py), the plain keyword
    matcher ("ai_draft", also set explicitly), or -- for older callers that
    don't set cloAlignMethod -- inferred from whether a score is present.
    """
    if not q.get("cloCode"):
        return None
    if q.get("cloAlignMethod"):
        return q["cloAlignMethod"]
    return "ai_draft" if q.get("cloAlignScore") is not None else "teacher_confirmed"


async def save_exam_questions(exam_id: str, school_id: str, questions: list[dict]) -> None:
    """
    Replaces this exam's questions with the freshly parsed set and marks the
    exam 'draft' (ready for a human/2B validation pass, mirroring
    curriculum's parsed->review handoff). The DELETE makes a retry after a
    partial failure idempotent instead of leaving stale duplicate rows.
    """
    pool = await get_pool()
    async with pool.acquire() as conn:
        async with conn.transaction():
            await conn.execute("DELETE FROM assessment.questions WHERE exam_id = $1", exam_id)
            if questions:
                await conn.executemany(
                    """
                    INSERT INTO assessment.questions
                        (exam_id, school_id, sequence_number, question_text, question_type, marks,
                         part_label, clo_code, clo_align_score, clo_align_method, answer_key, options)
                    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb, $12::jsonb)
                    """,
                    [
                        (
                            exam_id,
                            school_id,
                            q["sequenceNumber"],
                            q["questionText"],
                            q["questionType"],
                            q["marks"],
                            q.get("partLabel"),
                            q.get("cloCode"),
                            # str() first -- Decimal(float) preserves the
                            # source float's binary imprecision (e.g.
                            # 0.7139999...), Decimal(str(float)) doesn't.
                            Decimal(str(q["cloAlignScore"])) if q.get("cloAlignScore") is not None else None,
                            _clo_align_method(q),
                            json.dumps(q["answerKey"]) if q.get("answerKey") else None,
                            json.dumps(q["options"]) if q.get("options") else None,
                        )
                        for q in questions
                    ],
                )
            await conn.execute(
                """
                UPDATE assessment.exams
                SET status = 'draft', parse_error = NULL, updated_at = now()
                WHERE id = $1
                """,
                exam_id,
            )


async def fetch_exam_school(exam_id: str) -> Optional[str]:
    """Just school_id -- used by the answer-key upload path to validate
    ownership without pulling the full exam row."""
    pool = await get_pool()
    row = await pool.fetchrow("SELECT school_id FROM assessment.exams WHERE id = $1", exam_id)
    return row["school_id"] if row else None


async def apply_answer_key(exam_id: str, answer_key_map: dict[int, str]) -> int:
    """
    Applies a separately-uploaded answer key to an exam's already-parsed
    questions, matched by sequence_number -- only mcq questions are
    touched (same MCQ-only scope as the embedded-table path in
    exam_parser/service.py). Returns how many question rows were actually
    matched and updated, so the caller can tell the teacher if the answer
    key didn't line up with the exam (e.g. wrong exam, or the exam was
    re-parsed and sequence numbers shifted).
    """
    if not answer_key_map:
        return 0
    pool = await get_pool()
    matched = 0
    async with pool.acquire() as conn:
        async with conn.transaction():
            for seq, letter in answer_key_map.items():
                tag = await conn.execute(
                    """
                    UPDATE assessment.questions
                    SET answer_key = $3::jsonb
                    WHERE exam_id = $1 AND sequence_number = $2 AND question_type = 'mcq'
                    """,
                    exam_id,
                    seq,
                    json.dumps({"correctOption": letter}),
                )
                if tag.endswith(" 1"):
                    matched += 1
    return matched


async def mark_exam_failed(exam_id: str, error: str) -> None:
    """Mark an exam 'failed' with the error message, e.g. on a parsing exception."""
    pool = await get_pool()
    await pool.execute(
        "UPDATE assessment.exams SET status = 'failed', parse_error = $2, updated_at = now() WHERE id = $1",
        exam_id,
        error[:4000],  # keep it sane in case of a huge traceback
    )
