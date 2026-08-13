"""
Postgres access layer for the study-plan generator (Capability 3B).

Reads the rich 3A output (students.gap_records with root causes and
Gemini explanations) and writes students.study_plans. Same
module-per-domain split as postgres_assessment.py / postgres_gap.py,
sharing the one process-wide pool.
"""

from __future__ import annotations

import json
from datetime import date
from typing import Optional

import asyncpg

from app.db.postgres import get_pool


async def fetch_gaps_for_plan(student_id: str, target_exam_id: Optional[str]) -> list[asyncpg.Record]:
    """
    Unresolved gap records for the student, newest detection first --
    scoped to the target exam's subject when one is given (a study plan
    for the Biology final shouldn't schedule Physics gaps), otherwise
    everything unresolved.
    """
    pool = await get_pool()
    if target_exam_id:
        return await pool.fetch(
            """
            SELECT g.id, g.topic_id, g.root_cause_topic_id, g.severity_score,
                   g.prerequisite_depth, g.llm_explanation, g.clo_code, g.detected_at
            FROM students.gap_records g
            JOIN assessment.exams ge ON ge.id = g.detected_in_exam
            WHERE g.student_id = $1 AND g.resolved_at IS NULL
              AND ge.subject_code = (SELECT subject_code FROM assessment.exams WHERE id = $2)
            ORDER BY g.detected_at DESC
            """,
            student_id,
            target_exam_id,
        )
    return await pool.fetch(
        """
        SELECT g.id, g.topic_id, g.root_cause_topic_id, g.severity_score,
               g.prerequisite_depth, g.llm_explanation, g.clo_code, g.detected_at
        FROM students.gap_records g
        WHERE g.student_id = $1 AND g.resolved_at IS NULL
        ORDER BY g.detected_at DESC
        """,
        student_id,
    )


async def fetch_topic_meta(topic_ids: list[str]) -> dict[str, asyncpg.Record]:
    """Everything the scheduler needs per topic: display title, effort
    estimate, key concepts, plus curriculum position for stable tie-break
    ordering inside the topological sort."""
    if not topic_ids:
        return {}
    pool = await get_pool()
    rows = await pool.fetch(
        """
        SELECT t.id, t.title_en, t.grade_level, t.estimated_hours, t.key_concepts,
               t.sequence_order, u.number AS unit_number, t.subject_code
        FROM curriculum.topics t
        JOIN curriculum.units u ON u.id = t.unit_id
        WHERE t.id = ANY($1::uuid[])
        """,
        topic_ids,
    )
    return {str(r["id"]): r for r in rows}


async def fetch_prereq_edges_pg(topic_ids: list[str]) -> list[tuple[str, str]]:
    """Postgres fallback for neo4j.fetch_prerequisite_edges_among -- same
    (topic_id, prerequisite_id) pairs from the system of record."""
    if not topic_ids:
        return []
    pool = await get_pool()
    rows = await pool.fetch(
        """
        SELECT topic_id, prerequisite_id
        FROM curriculum.topic_prerequisites
        WHERE topic_id = ANY($1::uuid[]) AND prerequisite_id = ANY($1::uuid[])
        """,
        topic_ids,
    )
    return [(str(r["topic_id"]), str(r["prerequisite_id"])) for r in rows]


async def fetch_target_exam_due_date(target_exam_id: str) -> Optional[date]:
    """For Step 5's "appears on your exam in N days" framing -- None if
    the exam has no due_date set (nullable, V011), which just means the
    enrichment prompt omits the countdown."""
    pool = await get_pool()
    row = await pool.fetchrow("SELECT due_date FROM assessment.exams WHERE id = $1", target_exam_id)
    return row["due_date"] if row else None


async def fetch_student_school(student_id: str) -> Optional[str]:
    pool = await get_pool()
    row = await pool.fetchrow("SELECT school_id FROM students WHERE id = $1", student_id)
    return str(row["school_id"]) if row else None


async def insert_study_plan(
    student_id: str,
    school_id: str,
    target_exam_id: Optional[str],
    plan_data: dict,
    total_days: int,
    total_hours: float,
    language: str,
) -> str:
    """
    Deactivates the student's previous active plan for the same target
    (IS NOT DISTINCT FROM handles the NULL "general plan" case) and
    inserts the new one -- regenerating is a replace, not a pile-up.
    """
    pool = await get_pool()
    async with pool.acquire() as conn:
        async with conn.transaction():
            await conn.execute(
                """
                UPDATE students.study_plans
                SET is_active = FALSE
                WHERE student_id = $1 AND target_exam_id IS NOT DISTINCT FROM $2 AND is_active
                """,
                student_id,
                target_exam_id,
            )
            row = await conn.fetchrow(
                """
                INSERT INTO students.study_plans
                    (student_id, school_id, target_exam_id, plan_data, total_days,
                     total_hours, language, expires_at, is_active)
                VALUES ($1, $2, $3, $4::jsonb, $5, $6, $7, now() + interval '30 days', TRUE)
                RETURNING id
                """,
                student_id,
                school_id,
                target_exam_id,
                json.dumps(plan_data),
                total_days,
                total_hours,
                language,
            )
    return str(row["id"])
