"""
Postgres access layer for the AI tutor (Capability 3C).

Retrieval side of the Graph-RAG loop: candidate curriculum topics to
match the student's question against, and the student's own gap records
(3A output) so the answer can be tailored to their actual broken
prerequisites, not generic.
"""

from __future__ import annotations

from typing import Optional

import asyncpg

from app.db.postgres import get_pool


async def fetch_student(student_id: str) -> Optional[asyncpg.Record]:
    pool = await get_pool()
    return await pool.fetchrow(
        "SELECT id, school_id, grade_level FROM students WHERE id = $1", student_id
    )


async def fetch_candidate_topics(grade_level: Optional[int]) -> list[asyncpg.Record]:
    """
    Topics the question could be about. Grade-scoped to the student's
    grade AND below -- a Grade 10 student asking about a Grade 9
    prerequisite concept must still match it; later grades' material
    would only mislead the scorer.
    """
    pool = await get_pool()
    if grade_level is not None:
        return await pool.fetch(
            """
            SELECT id, title_en, description, key_concepts, grade_level, subject_code
            FROM curriculum.topics WHERE grade_level <= $1
            """,
            grade_level,
        )
    return await pool.fetch(
        "SELECT id, title_en, description, key_concepts, grade_level, subject_code FROM curriculum.topics"
    )


async def fetch_gap_context(student_id: str, topic_ids: list[str]) -> list[asyncpg.Record]:
    """
    The student's unresolved gaps touching the matched topics -- as
    symptom OR root cause (asking about the topic that broke something
    else matters just as much). Falls back to nothing cleanly; the tutor
    still answers, just without personalization.
    """
    pool = await get_pool()
    return await pool.fetch(
        """
        SELECT g.id, g.severity_score, g.llm_explanation, g.detected_at,
               st.title_en AS symptom_title,
               rt.title_en AS root_cause_title, rt.grade_level AS root_cause_grade
        FROM students.gap_records g
        JOIN curriculum.topics st ON st.id = g.topic_id
        LEFT JOIN curriculum.topics rt ON rt.id = g.root_cause_topic_id
        WHERE g.student_id = $1 AND g.resolved_at IS NULL
          AND (g.topic_id = ANY($2::uuid[]) OR g.root_cause_topic_id = ANY($2::uuid[]))
        ORDER BY g.severity_score DESC, g.detected_at DESC
        LIMIT 5
        """,
        student_id,
        topic_ids,
    )
