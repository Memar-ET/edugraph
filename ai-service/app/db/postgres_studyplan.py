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
from app.db.postgres_gckt import fetch_skill_states_bulk


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


# Per-recommendation weight fed into the repetition-penalty sum below,
# keyed by outcome_status -- this is what makes outcome tracking (V049)
# an actual FEEDBACK loop rather than a write-only log: a repeat that
# demonstrably worsened mastery penalizes more than a plain repeat, one
# that already improved things penalizes less (repetition is fine if it's
# working), and an outcome we haven't evaluated yet (pending) or couldn't
# (insufficient_evidence) counts as an ordinary, un-adjusted repeat.
_OUTCOME_REPETITION_WEIGHT = {
    "worsened": 2.0,
    "unchanged": 1.5,
    "improved": 0.5,
    "pending": 1.0,
    "insufficient_evidence": 1.0,
}


async def fetch_repetition_counts(student_id: str, topic_ids: list[str], window_days: int = 30) -> dict[str, float]:
    """Weighted count of how often each topic has been recommended to this
    student in the last `window_days` -- EG-GCKT Milestone 7's
    repetition-penalty ranking factor (spec section 13), now weighted by
    each past recommendation's measured outcome (checklist sections
    2/14/20) rather than a flat per-occurrence count."""
    if not topic_ids:
        return {}
    pool = await get_pool()
    rows = await pool.fetch(
        """
        SELECT topic_id, outcome_status, count(*) AS n
        FROM students.recommendation_log
        WHERE student_id = $1 AND topic_id = ANY($2::uuid[])
          AND recommended_at > now() - ($3 || ' days')::interval
        GROUP BY topic_id, outcome_status
        """,
        student_id,
        topic_ids,
        str(window_days),
    )
    weighted: dict[str, float] = {}
    for r in rows:
        tid = str(r["topic_id"])
        weight = _OUTCOME_REPETITION_WEIGHT.get(r["outcome_status"], 1.0)
        weighted[tid] = weighted.get(tid, 0.0) + weight * int(r["n"])
    return weighted


async def fetch_failed_recovery_route_counts(student_id: str, topic_ids: list[str]) -> dict[str, int]:
    """How many recovery routes have already failed for each blocked
    topic -- action_ranking.classify_action_types escalates to a human
    ('teacher_escalation') once this crosses a threshold rather than
    trying yet another automated route indefinitely."""
    if not topic_ids:
        return {}
    pool = await get_pool()
    rows = await pool.fetch(
        """
        SELECT blocked_topic_id, count(*) AS n
        FROM students.recovery_attempts
        WHERE student_id = $1 AND blocked_topic_id = ANY($2::uuid[]) AND status = 'failed'
        GROUP BY blocked_topic_id
        """,
        student_id,
        topic_ids,
    )
    return {str(r["blocked_topic_id"]): int(r["n"]) for r in rows}


async def fetch_avg_item_difficulty(topic_ids: list[str]) -> dict[str, tuple[float, float]]:
    """Mean (difficulty, discrimination) across a topic's CALIBRATED
    items only (item_skill_mappings rows with non-NULL difficulty/
    discrimination -- Milestone 8's IRT engine, or a promoted Milestone 9
    refit). A topic with no calibrated items is absent from the result;
    the caller applies a neutral default rather than pretending to know
    an uncalibrated item's difficulty."""
    if not topic_ids:
        return {}
    pool = await get_pool()
    rows = await pool.fetch(
        """
        SELECT topic_id, avg(difficulty) AS avg_difficulty, avg(discrimination) AS avg_discrimination
        FROM assessment.item_skill_mappings
        WHERE topic_id = ANY($1::uuid[]) AND is_current AND difficulty IS NOT NULL AND discrimination IS NOT NULL
        GROUP BY topic_id
        """,
        topic_ids,
    )
    return {str(r["topic_id"]): (float(r["avg_difficulty"]), float(r["avg_discrimination"])) for r in rows}


async def fetch_mandatory_clo_fraction(topic_ids: list[str]) -> dict[str, float]:
    """Fraction of each topic's mapped CLOs that are curriculum.clos.
    is_mandatory -- EG-GCKT Milestone 7's curriculum-priority ranking
    factor. A topic with no CLO mappings at all is absent from the
    result; the caller applies a neutral default."""
    if not topic_ids:
        return {}
    pool = await get_pool()
    rows = await pool.fetch(
        """
        SELECT m.topic_id,
               avg(CASE WHEN c.is_mandatory THEN 1.0 ELSE 0.0 END) AS mandatory_fraction
        FROM curriculum.topic_clo_mappings m
        JOIN curriculum.clos c ON c.code = m.clo_code
        WHERE m.topic_id = ANY($1::uuid[])
        GROUP BY m.topic_id
        """,
        topic_ids,
    )
    return {str(r["topic_id"]): float(r["mandatory_fraction"]) for r in rows}


async def log_recommendations(
    student_id: str,
    school_id: str,
    topic_ids: list[str],
    plan_id: Optional[str],
    action_types: Optional[dict[str, str]] = None,
    model_snapshot_id: Optional[str] = None,
) -> None:
    """Records that this plan generation recommended each of these topics
    -- what fetch_repetition_counts reads on the NEXT generation. Called
    after insert_study_plan so plan_id is available; plan_id is nullable
    (FK ON DELETE SET NULL) so a later plan deletion never orphans this
    history. action_types (EG-GCKT checklist section 14) is per-topic --
    a topic serving as a recovery route gets 'alternative_representation',
    a cold-start topic gets 'concept_explanation', etc (see
    action_ranking.classify_action_types); defaults to 'practice' for any
    topic not present in the map.

    Also captures each topic's CURRENT mastery_probability/evidence_count
    as the recommendation-time baseline (checklist sections 2/14/20:
    "measure learning gain" / "capture action outcome") -- mastery_at_
    recommendation stays NULL for a genuinely cold-start topic (no
    fabricated baseline), which evaluate_recommendation_outcomes
    (refit_worker.py) treats specially."""
    if not topic_ids:
        return
    action_types = action_types or {}
    states = await fetch_skill_states_bulk(student_id, topic_ids)
    pool = await get_pool()
    await pool.executemany(
        """
        INSERT INTO students.recommendation_log
            (student_id, school_id, topic_id, plan_id, action_type, model_snapshot_id,
             mastery_at_recommendation, evidence_count_at_recommendation)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        """,
        [
            (
                student_id,
                school_id,
                tid,
                plan_id,
                action_types.get(tid, "practice"),
                model_snapshot_id,
                states.get(tid, {}).get("mastery_probability"),
                states.get(tid, {}).get("evidence_count", 0),
            )
            for tid in topic_ids
        ],
    )


async def fetch_pending_recommendation_outcomes(older_than_days: int) -> list[dict]:
    """recommendation_log rows still 'pending' whose recommended_at is old
    enough to fairly judge -- too soon and a student simply hasn't had a
    chance to act on it yet. Joins the CURRENT students.skill_states row
    so refit_worker.evaluate_recommendation_outcomes doesn't need an N+1
    lookup per row."""
    pool = await get_pool()
    rows = await pool.fetch(
        """
        SELECT rl.id, rl.mastery_at_recommendation, rl.evidence_count_at_recommendation,
               ss.mastery_probability AS mastery_now, ss.evidence_count AS evidence_now
        FROM students.recommendation_log rl
        LEFT JOIN students.skill_states ss
               ON ss.student_id = rl.student_id AND ss.topic_id = rl.topic_id
        WHERE rl.outcome_status = 'pending'
          AND rl.recommended_at <= now() - ($1 || ' days')::interval
        """,
        str(older_than_days),
    )
    return [
        {
            "id": str(r["id"]),
            "mastery_before": float(r["mastery_at_recommendation"]) if r["mastery_at_recommendation"] is not None else None,
            "evidence_before": int(r["evidence_count_at_recommendation"] or 0),
            "mastery_now": float(r["mastery_now"]) if r["mastery_now"] is not None else None,
            "evidence_now": int(r["evidence_now"] or 0),
        }
        for r in rows
    ]


async def apply_recommendation_outcomes(updates: list[tuple[str, str, Optional[float]]]) -> None:
    """updates: (recommendation_log.id, outcome_status, mastery_at_evaluation) tuples."""
    if not updates:
        return
    pool = await get_pool()
    await pool.executemany(
        """
        UPDATE students.recommendation_log
        SET outcome_status = $2, mastery_at_evaluation = $3, outcome_evaluated_at = now()
        WHERE id = $1
        """,
        updates,
    )


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
