"""
Postgres access layer for the gap-analysis worker (Capability 3A).

Mirrors postgres_assessment.py's module-per-domain split: read side pulls
the graded attempt + its missed questions and resolves Question -> CLO ->
Topic; write side persists all three storage layers (gap_records,
exam_insights, subject_profiles) plus mastery_records in one transaction.

Prerequisite traversal: the primary walk is Neo4j (app/db/neo4j.py, per
the graph-first design); fetch_prerequisite_chain_pg here is the
system-of-record fallback over curriculum.topic_prerequisites. Both are
populated together by POST /curriculum/topics/:id/prerequisites (see
app/db/neo4j.py's docstring) -- as of this writing nobody has used that
endpoint yet, so both are empty in practice, but the write path exists.
Both callers treat "no chain" as normal, not an error, since a partially
(or not yet) populated graph is an expected ongoing state.
"""

from __future__ import annotations

import json
from typing import Any, Optional

import asyncpg

from app.db.postgres import get_pool


async def fetch_attempt(attempt_id: str) -> Optional[asyncpg.Record]:
    """The graded attempt joined to its exam -- everything the pipeline
    needs to know what was assessed and how it went overall."""
    pool = await get_pool()
    return await pool.fetchrow(
        """
        SELECT a.id, a.student_id, a.exam_id, a.school_id,
               a.total_score, a.percentage, a.passed,
               e.subject_code, e.grade_level, e.exam_scope, e.title AS exam_title
        FROM assessment.exam_attempts a
        JOIN assessment.exams e ON e.id = a.exam_id
        WHERE a.id = $1
        """,
        attempt_id,
    )


async def fetch_missed_answers(attempt_id: str) -> list[asyncpg.Record]:
    """Every graded answer on the attempt that lost marks (missed or
    partially right). Ungraded answers (marks_awarded IS NULL) are
    excluded -- the attempt shouldn't have been queued with any, but a
    re-grade can transiently unfinalize one. answer_text is included for
    the EG-GCKT misconception-hypothesis generator (Milestone 6), which
    needs to see what the student actually wrote, not just that they lost
    marks -- everything else here is unchanged from before that feature."""
    pool = await get_pool()
    return await pool.fetch(
        """
        SELECT sa.question_id, sa.marks_awarded, sa.marks_possible, sa.answer_text,
               q.sequence_number, q.question_text, q.question_type,
               q.clo_code, q.topic_id
        FROM assessment.student_answers sa
        JOIN assessment.questions q ON q.id = sa.question_id
        WHERE sa.attempt_id = $1
          AND sa.marks_awarded IS NOT NULL
          AND sa.marks_awarded < sa.marks_possible
        ORDER BY q.sequence_number
        """,
        attempt_id,
    )


async def resolve_topics_for_clos(clo_codes: list[str]) -> dict[str, asyncpg.Record]:
    """clo_code -> its best-aligned topic (highest alignment_score wins),
    for questions that carry a CLO but no direct topic_id."""
    if not clo_codes:
        return {}
    pool = await get_pool()
    rows = await pool.fetch(
        """
        SELECT DISTINCT ON (m.clo_code)
               m.clo_code, t.id AS topic_id, t.title_en, t.grade_level
        FROM curriculum.topic_clo_mappings m
        JOIN curriculum.topics t ON t.id = m.topic_id
        WHERE m.clo_code = ANY($1::text[])
        ORDER BY m.clo_code, m.alignment_score DESC NULLS LAST
        """,
        clo_codes,
    )
    return {r["clo_code"]: r for r in rows}


async def fetch_topics(topic_ids: list[str]) -> dict[str, asyncpg.Record]:
    if not topic_ids:
        return {}
    pool = await get_pool()
    rows = await pool.fetch(
        "SELECT id, title_en, grade_level FROM curriculum.topics WHERE id = ANY($1::uuid[])",
        topic_ids,
    )
    return {str(r["id"]): r for r in rows}


async def fetch_prerequisite_chain_pg(topic_id: str, max_depth: int = 3) -> list[dict]:
    """Recursive walk over curriculum.topic_prerequisites -- same result
    shape as neo4j.fetch_prerequisite_chain so the service can use either."""
    pool = await get_pool()
    rows = await pool.fetch(
        """
        WITH RECURSIVE chain AS (
            SELECT tp.prerequisite_id, 1 AS depth
            FROM curriculum.topic_prerequisites tp
            WHERE tp.topic_id = $1
            UNION
            SELECT tp.prerequisite_id, c.depth + 1
            FROM curriculum.topic_prerequisites tp
            JOIN chain c ON tp.topic_id = c.prerequisite_id
            WHERE c.depth < $2
        )
        SELECT t.id, t.title_en, t.grade_level, min(c.depth) AS depth
        FROM chain c
        JOIN curriculum.topics t ON t.id = c.prerequisite_id
        GROUP BY t.id, t.title_en, t.grade_level
        ORDER BY depth
        """,
        topic_id,
        max_depth,
    )
    return [
        {"id": str(r["id"]), "title": r["title_en"], "grade_level": r["grade_level"], "depth": r["depth"]}
        for r in rows
    ]


async def fetch_downstream_dependents_pg(topic_id: str, max_depth: int = 3) -> list[dict]:
    """Postgres fallback for the EG-GCKT root-cause engine's
    DownstreamImpact factor (spec section 9): every topic that has
    topic_id somewhere in ITS OWN upstream prerequisite chain -- the
    reverse direction of fetch_prerequisite_chain_pg. Restricted to
    edge_type IN ('requires', 'strongly_requires') for the same reason
    class_heatmap.py's Neo4j walk is (Milestone 0): only hard-dependency
    edges represent a real "fixing this unlocks that" chain, not
    similar_to/related_to associations."""
    pool = await get_pool()
    rows = await pool.fetch(
        """
        WITH RECURSIVE chain AS (
            SELECT tp.topic_id AS dependent_id, 1 AS depth
            FROM curriculum.topic_prerequisites tp
            WHERE tp.prerequisite_id = $1 AND tp.edge_type IN ('requires', 'strongly_requires')
            UNION
            SELECT tp.topic_id AS dependent_id, c.depth + 1
            FROM curriculum.topic_prerequisites tp
            JOIN chain c ON tp.prerequisite_id = c.dependent_id
            WHERE c.depth < $2 AND tp.edge_type IN ('requires', 'strongly_requires')
        )
        SELECT dependent_id AS id, min(depth) AS depth FROM chain GROUP BY dependent_id
        """,
        topic_id,
        max_depth,
    )
    return [{"id": str(r["id"]), "depth": r["depth"]} for r in rows]


async def fetch_topic_mastery(student_id: str, topic_ids: list[str]) -> dict[str, float]:
    """
    Historical mastery (0..1) per topic from ALL of the student's graded
    answers, resolving each question's topic directly (questions.topic_id)
    or via its CLO's best topic mapping. Topics with no answer history
    fall back to students.mastery_records.confidence when a row exists;
    topics absent from the result have no evidence at all -- callers must
    NOT treat "no evidence" as "failed".
    """
    if not topic_ids:
        return {}
    pool = await get_pool()
    rows = await pool.fetch(
        """
        SELECT x.topic_id, sum(x.marks_awarded) AS awarded, sum(x.marks_possible) AS possible
        FROM (
            SELECT COALESCE(q.topic_id, tcm.topic_id) AS topic_id,
                   sa.marks_awarded, sa.marks_possible
            FROM assessment.student_answers sa
            JOIN assessment.questions q ON q.id = sa.question_id
            LEFT JOIN LATERAL (
                SELECT m.topic_id
                FROM curriculum.topic_clo_mappings m
                WHERE m.clo_code = q.clo_code
                ORDER BY m.alignment_score DESC NULLS LAST
                LIMIT 1
            ) tcm ON q.topic_id IS NULL
            WHERE sa.student_id = $1 AND sa.marks_awarded IS NOT NULL
        ) x
        WHERE x.topic_id = ANY($2::uuid[])
        GROUP BY x.topic_id
        HAVING sum(x.marks_possible) > 0
        """,
        student_id,
        topic_ids,
    )
    mastery = {str(r["topic_id"]): float(r["awarded"]) / float(r["possible"]) for r in rows}

    missing = [t for t in topic_ids if t not in mastery]
    if missing:
        recorded = await pool.fetch(
            """
            SELECT topic_id, confidence FROM students.mastery_records
            WHERE student_id = $1 AND topic_id = ANY($2::uuid[]) AND confidence IS NOT NULL
            """,
            student_id,
            missing,
        )
        for r in recorded:
            mastery[str(r["topic_id"])] = float(r["confidence"])
    return mastery


async def compute_subject_aggregate(student_id: str, subject_code: str) -> dict[str, Any]:
    """
    The Subject Health Layer's inputs: overall mastery pct across every
    graded answer the student has in this subject (any exam scope), plus
    per-topic mastery for the weak-areas ranking.
    """
    pool = await get_pool()
    rows = await pool.fetch(
        """
        SELECT x.topic_id, t.title_en,
               sum(x.marks_awarded) AS awarded, sum(x.marks_possible) AS possible
        FROM (
            SELECT COALESCE(q.topic_id, tcm.topic_id) AS topic_id,
                   sa.marks_awarded, sa.marks_possible
            FROM assessment.student_answers sa
            JOIN assessment.questions q ON q.id = sa.question_id
            JOIN assessment.exams e ON e.id = q.exam_id
            LEFT JOIN LATERAL (
                SELECT m.topic_id
                FROM curriculum.topic_clo_mappings m
                WHERE m.clo_code = q.clo_code
                ORDER BY m.alignment_score DESC NULLS LAST
                LIMIT 1
            ) tcm ON q.topic_id IS NULL
            WHERE sa.student_id = $1 AND e.subject_code = $2
              AND sa.marks_awarded IS NOT NULL
        ) x
        LEFT JOIN curriculum.topics t ON t.id = x.topic_id
        GROUP BY x.topic_id, t.title_en
        """,
        student_id,
        subject_code,
    )
    total_awarded = sum(float(r["awarded"]) for r in rows)
    total_possible = sum(float(r["possible"]) for r in rows)
    mastery_pct = (total_awarded / total_possible * 100) if total_possible > 0 else 0.0

    topic_mastery = [
        {
            "topicId": str(r["topic_id"]),
            "title": r["title_en"],
            "masteryPct": round(float(r["awarded"]) / float(r["possible"]) * 100, 1),
        }
        for r in rows
        if r["topic_id"] is not None and float(r["possible"]) > 0
    ]
    topic_mastery.sort(key=lambda t: t["masteryPct"])
    return {"mastery_pct": round(mastery_pct, 1), "weak_areas": topic_mastery[:3]}


async def persist_analysis(
    attempt: asyncpg.Record,
    gaps: list[dict],
    exam_summary: str,
    llm_model: Optional[str],
    mastery_updates: dict[str, float],
    subject_agg: dict[str, Any],
) -> None:
    """
    Writes all three storage layers atomically. gap dicts carry:
    question_id, symptom_topic_id, clo_code, severity, root_cause_topic_id
    (nullable), prerequisite_depth, llm_explanation (nullable).
    The DELETE-then-insert on attempt_id makes re-analysis idempotent
    (same convention as save_exam_questions).
    """
    attempt_id = str(attempt["id"])
    student_id = str(attempt["student_id"])
    school_id = str(attempt["school_id"])
    pool = await get_pool()
    async with pool.acquire() as conn:
        async with conn.transaction():
            await conn.execute(
                "DELETE FROM students.gap_records WHERE attempt_id = $1", attempt_id
            )
            if gaps:
                await conn.executemany(
                    """
                    INSERT INTO students.gap_records
                        (student_id, school_id, topic_id, clo_code, severity_score,
                         is_root_cause, prerequisite_depth, detected_in_exam,
                         attempt_id, question_id, root_cause_topic_id, llm_explanation,
                         rcs_score, rcs_factors, root_cause_path)
                    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14::jsonb, $15::jsonb)
                    """,
                    [
                        (
                            student_id,
                            school_id,
                            g["symptom_topic_id"],
                            g.get("clo_code"),
                            g["severity"],
                            g.get("root_cause_topic_id") is not None,
                            g.get("prerequisite_depth", 0),
                            str(attempt["exam_id"]),
                            attempt_id,
                            g["question_id"],
                            g.get("root_cause_topic_id"),
                            g.get("llm_explanation"),
                            g.get("rcs_score"),
                            json.dumps(g["rcs_factors"]) if g.get("rcs_factors") is not None else None,
                            json.dumps(g["root_cause_path"]) if g.get("root_cause_path") is not None else None,
                        )
                        for g in gaps
                    ],
                )

            await conn.execute(
                """
                INSERT INTO students.exam_insights
                    (student_id, exam_id, attempt_id, school_id, total_score,
                     percentage, passed, gaps_found, llm_exam_summary, llm_model)
                VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
                ON CONFLICT (attempt_id) DO UPDATE SET
                    total_score = EXCLUDED.total_score,
                    percentage = EXCLUDED.percentage,
                    passed = EXCLUDED.passed,
                    gaps_found = EXCLUDED.gaps_found,
                    llm_exam_summary = EXCLUDED.llm_exam_summary,
                    llm_model = EXCLUDED.llm_model,
                    updated_at = now()
                """,
                student_id,
                str(attempt["exam_id"]),
                attempt_id,
                school_id,
                attempt["total_score"],
                attempt["percentage"],
                attempt["passed"],
                len(gaps),
                exam_summary,
                llm_model,
            )

            for topic_id, confidence in mastery_updates.items():
                await conn.execute(
                    """
                    INSERT INTO students.mastery_records
                        (student_id, school_id, topic_id, confidence, mastered_at)
                    VALUES ($1, $2, $3, $4, now())
                    ON CONFLICT (student_id, topic_id) DO UPDATE SET
                        confidence = EXCLUDED.confidence,
                        mastered_at = now()
                    """,
                    student_id,
                    school_id,
                    topic_id,
                    round(confidence, 4),
                )

            # exams_analyzed counts insights (including the one written
            # above, inside this same transaction) rather than attempts,
            # so it means "exams the engine has actually dissected".
            await conn.execute(
                """
                INSERT INTO students.subject_profiles
                    (student_id, school_id, subject_code, grade_level,
                     current_mastery_pct, top_weak_areas, exams_analyzed, last_updated)
                VALUES ($1, $2, $3, $4, $5, $6::jsonb,
                        (SELECT count(*) FROM students.exam_insights ei
                         JOIN assessment.exams e ON e.id = ei.exam_id
                         WHERE ei.student_id = $1 AND e.subject_code = $3),
                        now())
                ON CONFLICT (student_id, subject_code) DO UPDATE SET
                    grade_level = EXCLUDED.grade_level,
                    current_mastery_pct = EXCLUDED.current_mastery_pct,
                    top_weak_areas = EXCLUDED.top_weak_areas,
                    exams_analyzed = EXCLUDED.exams_analyzed,
                    last_updated = now()
                """,
                student_id,
                school_id,
                attempt["subject_code"],
                attempt["grade_level"],
                subject_agg["mastery_pct"],
                json.dumps(subject_agg["weak_areas"]),
            )


async def fetch_all_attempt_answers(attempt_id: str) -> list[asyncpg.Record]:
    """Every graded answer on the attempt -- not just missed ones, unlike
    fetch_missed_answers -- since a CORRECT answer to a misconception's
    verification item is exactly the "weakens this hypothesis" signal
    check_verification_responses needs to see."""
    pool = await get_pool()
    return await pool.fetch(
        """
        SELECT question_id, (marks_awarded >= marks_possible * 0.5) AS correct
        FROM assessment.student_answers
        WHERE attempt_id = $1 AND marks_awarded IS NOT NULL AND marks_possible > 0
        """,
        attempt_id,
    )


async def select_verification_item(topic_id: str, exclude_question_ids: list[str]) -> Optional[str]:
    """A question mapped to this topic the student hasn't already
    answered (so it's genuinely diagnostic, not something already known)
    -- 'use diagnostic items to actively verify important hypotheses'
    (spec section 12). Returns None (never fabricates one) when no such
    item exists yet."""
    pool = await get_pool()
    row = await pool.fetchrow(
        """
        SELECT m.question_id
        FROM assessment.item_skill_mappings m
        WHERE m.topic_id = $1 AND m.is_current AND NOT (m.question_id = ANY($2::uuid[]))
        ORDER BY m.relevance DESC
        LIMIT 1
        """,
        topic_id,
        exclude_question_ids,
    )
    return str(row["question_id"]) if row else None


async def fetch_candidate_misconception(student_id: str, topic_id: str) -> Optional[asyncpg.Record]:
    """The existing candidate hypothesis for this (student, topic), if
    one exists -- checked before creating a new one so a second repeated-
    failure pattern on the same topic accumulates evidence onto the
    existing hypothesis instead of spawning a duplicate."""
    pool = await get_pool()
    return await pool.fetchrow(
        """
        SELECT id, confidence, supporting_evidence, verification_item_id
        FROM students.misconception_hypotheses
        WHERE student_id = $1 AND topic_id = $2 AND status = 'candidate'
        ORDER BY created_at DESC LIMIT 1
        """,
        student_id,
        topic_id,
    )


async def append_misconception_evidence(hypothesis_id: str, new_answers: list[dict[str, Any]]) -> None:
    """Appends more wrong-answer evidence to an existing candidate
    hypothesis's supporting_evidence.answers array -- 'track evidence
    supporting or weakening a misconception hypothesis' (spec section
    12). Only touches 'candidate' rows; a confirmed/rejected hypothesis
    is a closed record a teacher has already decided on."""
    pool = await get_pool()
    await pool.execute(
        """
        UPDATE students.misconception_hypotheses
        SET supporting_evidence = jsonb_set(
            COALESCE(supporting_evidence, '{"answers": []}'::jsonb),
            '{answers}',
            COALESCE(supporting_evidence -> 'answers', '[]'::jsonb) || $2::jsonb
        )
        WHERE id = $1 AND status = 'candidate'
        """,
        hypothesis_id,
        json.dumps(new_answers),
    )


async def fetch_active_verification_items(student_id: str) -> list[asyncpg.Record]:
    """Every candidate hypothesis for this student that has a
    verification item assigned -- checked against each newly-graded
    answer (spec: "use diagnostic items to actively verify important
    hypotheses")."""
    pool = await get_pool()
    return await pool.fetch(
        """
        SELECT id, verification_item_id, confidence
        FROM students.misconception_hypotheses
        WHERE student_id = $1 AND status = 'candidate' AND verification_item_id IS NOT NULL
        """,
        student_id,
    )


async def adjust_misconception_confidence(hypothesis_id: str, delta: float, note: str) -> None:
    """Nudges a candidate hypothesis's confidence up (verification item
    also missed -- strengthens it) or down (verification item answered
    correctly -- weakens it), clamped to [0, 1]. 'Update misconception
    state after intervention outcomes' (spec section 12) -- this is the
    passive/observational version (a verification item just happened to
    be answered), not a teacher-initiated intervention outcome."""
    pool = await get_pool()
    await pool.execute(
        """
        UPDATE students.misconception_hypotheses
        SET confidence = GREATEST(0, LEAST(1, COALESCE(confidence, 0.5) + $2)),
            trigger_pattern = COALESCE(trigger_pattern, '') || CASE WHEN trigger_pattern IS NULL THEN '' ELSE ' | ' END || $3
        WHERE id = $1 AND status = 'candidate'
        """,
        hypothesis_id,
        delta,
        note,
    )


async def insert_misconception_hypothesis(
    *,
    student_id: str,
    school_id: str,
    topic_id: str,
    misconception_text: str,
    trigger_pattern: Optional[str],
    supporting_evidence: dict[str, Any],
    confidence: float,
    intervention_text: Optional[str],
    generated_by_model: Optional[str],
    verification_item_id: Optional[str] = None,
) -> str:
    """Persists one candidate misconception hypothesis (EG-GCKT Milestone
    6, spec section 11). Always inserted as status='candidate' -- this
    function never sets status='confirmed'; that only happens via a
    teacher reviewing it through the Go endpoint
    (assessment/handler/misconceptions.go)."""
    pool = await get_pool()
    row = await pool.fetchrow(
        """
        INSERT INTO students.misconception_hypotheses
            (student_id, school_id, topic_id, misconception_text, trigger_pattern,
             supporting_evidence, confidence, intervention_text, generated_by_model, verification_item_id)
        VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9, $10)
        RETURNING id
        """,
        student_id,
        school_id,
        topic_id,
        misconception_text,
        trigger_pattern,
        json.dumps(supporting_evidence),
        confidence,
        intervention_text,
        generated_by_model,
        verification_item_id,
    )
    return str(row["id"])
