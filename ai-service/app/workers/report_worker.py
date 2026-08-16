"""
Report generation worker (Phase 12).

Consumes JSON job payloads from queue:report:generate. Each payload:
    {
        "report_id":    "<UUID of the public.report_results row>",
        "report_type":  "school_monthly" | "national_heatmap" | "clo_coverage",
        "params":       { ... type-specific params ... },
        "requester_id": "<UUID>"
    }

Three report types:
  school_monthly    – aggregated performance for one school in one month
  national_heatmap  – cross-region mastery snapshot for a given month
  clo_coverage      – CLO testing depth/gap report for one curriculum subject

Unlike the other BRPOP workers this one serialises heavy Postgres
aggregation queries. Reports are generated on-demand (not cached), and
the result is stored in public.report_results.result JSONB. The Go
read endpoint (GET /api/v1/reports/{id}) simply returns that row.
"""

from __future__ import annotations

import asyncio
import json

import structlog

from app.db.dead_letter import push_dead_letter
from app.db.postgres import get_pool
from app.db.redis import REPORT_QUEUE, brpop_job

logger = structlog.get_logger()

_shutdown = asyncio.Event()


# ─── DB helpers ────────────────────────────────────────────────────────────────

async def _set_status(report_id: str, status: str, result: dict | None = None, error: str | None = None) -> None:
    pool = await get_pool()
    if status == "done" and result is not None:
        await pool.execute(
            """
            UPDATE public.report_results
            SET status = 'done', result = $2::jsonb, generated_at = NOW(), updated_at = NOW()
            WHERE id = $1
            """,
            report_id,
            json.dumps(result),
        )
    elif status == "failed":
        await pool.execute(
            """
            UPDATE public.report_results
            SET status = 'failed', error_text = $2, updated_at = NOW()
            WHERE id = $1
            """,
            report_id,
            (error or "unknown error")[:4000],
        )
    else:
        await pool.execute(
            "UPDATE public.report_results SET status = $2, updated_at = NOW() WHERE id = $1",
            report_id,
            status,
        )


# ─── Report generators ─────────────────────────────────────────────────────────

async def generate_school_monthly(school_id: str, year: int, month: int) -> dict:
    pool = await get_pool()

    # Overall student count for the school
    total_students_row = await pool.fetchrow(
        "SELECT COUNT(*) AS n FROM public.students WHERE school_id = $1 AND deleted_at IS NULL",
        school_id,
    )
    total_students = int(total_students_row["n"]) if total_students_row else 0

    # Exams taken in the month
    exams_taken_row = await pool.fetchrow(
        """
        SELECT COUNT(DISTINCT e.id) AS n
        FROM assessment.exams e
        JOIN assessment.student_answers sa ON sa.exam_id = e.id
        JOIN public.students s ON s.id = sa.student_id
        WHERE s.school_id = $1
          AND EXTRACT(YEAR FROM sa.submitted_at) = $2
          AND EXTRACT(MONTH FROM sa.submitted_at) = $3
        """,
        school_id,
        year,
        month,
    )
    exams_taken = int(exams_taken_row["n"]) if exams_taken_row else 0

    # Average mastery by subject via gap_records
    subject_rows = await pool.fetch(
        """
        SELECT sub.subject_code, sub.subject_name,
               AVG(gr.mastery_level) AS avg_mastery,
               COUNT(DISTINCT gr.student_id) AS student_count
        FROM students.gap_records gr
        JOIN curriculum.topics t ON t.id = gr.topic_id
        JOIN curriculum.units u ON u.id = t.unit_id
        JOIN curriculum.subjects sub ON sub.code = u.subject_code
        JOIN public.students s ON s.id = gr.student_id
        WHERE s.school_id = $1
          AND EXTRACT(YEAR FROM gr.created_at) = $2
          AND EXTRACT(MONTH FROM gr.created_at) = $3
        GROUP BY sub.subject_code, sub.subject_name
        ORDER BY avg_mastery
        """,
        school_id,
        year,
        month,
    )
    avg_mastery_by_subject = [
        {
            "subject_code": r["subject_code"],
            "subject_name": r["subject_name"],
            "avg_mastery": round(float(r["avg_mastery"]), 4),
            "student_count": int(r["student_count"]),
        }
        for r in subject_rows
    ]

    # Top 5 weakest topics
    weak_rows = await pool.fetch(
        """
        SELECT t.id AS topic_id, t.title AS topic_title,
               AVG(gr.mastery_level) AS avg_mastery,
               COUNT(DISTINCT gr.student_id) AS affected_count
        FROM students.gap_records gr
        JOIN curriculum.topics t ON t.id = gr.topic_id
        JOIN public.students s ON s.id = gr.student_id
        WHERE s.school_id = $1
          AND EXTRACT(YEAR FROM gr.created_at) = $2
          AND EXTRACT(MONTH FROM gr.created_at) = $3
          AND gr.mastery_level < 0.5
        GROUP BY t.id, t.title
        ORDER BY avg_mastery
        LIMIT 5
        """,
        school_id,
        year,
        month,
    )
    top_weak_topics = [
        {
            "topic_id": str(r["topic_id"]),
            "topic_title": r["topic_title"],
            "avg_mastery": round(float(r["avg_mastery"]), 4),
            "affected_count": int(r["affected_count"]),
        }
        for r in weak_rows
    ]

    # Top 5 strongest topics
    strong_rows = await pool.fetch(
        """
        SELECT t.id AS topic_id, t.title AS topic_title,
               AVG(gr.mastery_level) AS avg_mastery,
               COUNT(DISTINCT gr.student_id) AS student_count
        FROM students.gap_records gr
        JOIN curriculum.topics t ON t.id = gr.topic_id
        JOIN public.students s ON s.id = gr.student_id
        WHERE s.school_id = $1
          AND EXTRACT(YEAR FROM gr.created_at) = $2
          AND EXTRACT(MONTH FROM gr.created_at) = $3
          AND gr.mastery_level >= 0.7
        GROUP BY t.id, t.title
        ORDER BY avg_mastery DESC
        LIMIT 5
        """,
        school_id,
        year,
        month,
    )
    top_strong_topics = [
        {
            "topic_id": str(r["topic_id"]),
            "topic_title": r["topic_title"],
            "avg_mastery": round(float(r["avg_mastery"]), 4),
            "student_count": int(r["student_count"]),
        }
        for r in strong_rows
    ]

    # CLO coverage: fraction of active CLOs that appeared in at least one exam
    clo_coverage_row = await pool.fetchrow(
        """
        SELECT
            COUNT(DISTINCT c.id) FILTER (WHERE ism.topic_id IS NOT NULL) AS tested,
            COUNT(DISTINCT c.id) AS total
        FROM curriculum.clos c
        LEFT JOIN assessment.item_skill_mappings ism ON ism.clo_code = c.code AND ism.is_current
        LEFT JOIN assessment.questions q ON q.id = ism.question_id
        LEFT JOIN assessment.exams e ON e.id = q.exam_id
        LEFT JOIN assessment.student_answers sa ON sa.exam_id = e.id
        LEFT JOIN public.students s ON s.id = sa.student_id
        WHERE (s.school_id = $1 OR s.school_id IS NULL)
        """,
        school_id,
    )
    clo_coverage_pct = (
        round(int(clo_coverage_row["tested"]) / max(int(clo_coverage_row["total"]), 1) * 100, 1)
        if clo_coverage_row
        else 0.0
    )

    return {
        "school_id": school_id,
        "year": year,
        "month": month,
        "total_students": total_students,
        "exams_taken": exams_taken,
        "avg_mastery_by_subject": avg_mastery_by_subject,
        "top_weak_topics": top_weak_topics,
        "top_strong_topics": top_strong_topics,
        "clo_coverage_pct": clo_coverage_pct,
    }


async def generate_national_heatmap(year: int, month: int) -> dict:
    pool = await get_pool()

    # National average mastery
    nat_row = await pool.fetchrow(
        """
        SELECT AVG(gr.mastery_level) AS national_avg
        FROM students.gap_records gr
        WHERE EXTRACT(YEAR FROM gr.created_at) = $1
          AND EXTRACT(MONTH FROM gr.created_at) = $2
        """,
        year,
        month,
    )
    national_avg = round(float(nat_row["national_avg"] or 0), 4)

    # Per-region, per-school mastery
    region_rows = await pool.fetch(
        """
        SELECT r.id AS region_id, r.name AS region_name,
               sc.id AS school_id, sc.name AS school_name,
               AVG(gr.mastery_level) AS avg_mastery
        FROM students.gap_records gr
        JOIN public.students st ON st.id = gr.student_id
        JOIN public.schools sc ON sc.id = st.school_id
        JOIN public.regions r ON r.id = sc.region_id
        WHERE EXTRACT(YEAR FROM gr.created_at) = $1
          AND EXTRACT(MONTH FROM gr.created_at) = $2
        GROUP BY r.id, r.name, sc.id, sc.name
        ORDER BY r.name, avg_mastery
        """,
        year,
        month,
    )

    # Group by region
    region_map: dict = {}
    for row in region_rows:
        rid = str(row["region_id"])
        if rid not in region_map:
            region_map[rid] = {"region_id": rid, "region_name": row["region_name"], "schools": []}
        region_map[rid]["schools"].append({
            "school_id": str(row["school_id"]),
            "school_name": row["school_name"],
            "avg_mastery": round(float(row["avg_mastery"]), 4),
        })

    # Weakest and strongest topics nationally
    weak_rows = await pool.fetch(
        """
        SELECT t.id AS topic_id, t.title AS topic_title,
               AVG(gr.mastery_level) AS avg_mastery,
               COUNT(DISTINCT gr.student_id) AS affected_count
        FROM students.gap_records gr
        JOIN curriculum.topics t ON t.id = gr.topic_id
        WHERE EXTRACT(YEAR FROM gr.created_at) = $1
          AND EXTRACT(MONTH FROM gr.created_at) = $2
          AND gr.mastery_level < 0.5
        GROUP BY t.id, t.title
        ORDER BY avg_mastery
        LIMIT 10
        """,
        year,
        month,
    )
    strong_rows = await pool.fetch(
        """
        SELECT t.id AS topic_id, t.title AS topic_title,
               AVG(gr.mastery_level) AS avg_mastery,
               COUNT(DISTINCT gr.student_id) AS student_count
        FROM students.gap_records gr
        JOIN curriculum.topics t ON t.id = gr.topic_id
        WHERE EXTRACT(YEAR FROM gr.created_at) = $1
          AND EXTRACT(MONTH FROM gr.created_at) = $2
          AND gr.mastery_level >= 0.7
        GROUP BY t.id, t.title
        ORDER BY avg_mastery DESC
        LIMIT 10
        """,
        year,
        month,
    )

    return {
        "year": year,
        "month": month,
        "national_avg_mastery": national_avg,
        "regions": list(region_map.values()),
        "weakest_topics": [
            {
                "topic_id": str(r["topic_id"]),
                "topic_title": r["topic_title"],
                "avg_mastery": round(float(r["avg_mastery"]), 4),
                "affected_count": int(r["affected_count"]),
            }
            for r in weak_rows
        ],
        "strongest_topics": [
            {
                "topic_id": str(r["topic_id"]),
                "topic_title": r["topic_title"],
                "avg_mastery": round(float(r["avg_mastery"]), 4),
                "student_count": int(r["student_count"]),
            }
            for r in strong_rows
        ],
    }


async def generate_clo_coverage(subject_code: str) -> dict:
    pool = await get_pool()

    rows = await pool.fetch(
        """
        SELECT
            c.code AS clo_code,
            c.description AS clo_description,
            COUNT(DISTINCT ism.question_id) FILTER (WHERE ism.question_id IS NOT NULL) AS tested_count,
            AVG(ism.qmatrix_confidence) FILTER (WHERE ism.question_id IS NOT NULL) AS avg_confidence,
            AVG(gr.mastery_level) AS avg_mastery
        FROM curriculum.clos c
        JOIN curriculum.subjects sub ON sub.code = c.subject_code
        LEFT JOIN assessment.item_skill_mappings ism
            ON ism.clo_code = c.code AND ism.is_current
        LEFT JOIN students.gap_records gr
            ON gr.topic_id IN (
                SELECT tcm.topic_id
                FROM curriculum.topic_clo_mappings tcm
                WHERE tcm.clo_code = c.code
            )
        WHERE sub.code = $1
        GROUP BY c.code, c.description
        ORDER BY tested_count, avg_mastery NULLS LAST
        """,
        subject_code,
    )

    clos = []
    for r in rows:
        avg_confidence = float(r["avg_confidence"]) if r["avg_confidence"] is not None else None
        avg_mastery = float(r["avg_mastery"]) if r["avg_mastery"] is not None else None
        gap_flag = (
            (r["tested_count"] == 0)
            or (avg_mastery is not None and avg_mastery < 0.5)
            or (avg_confidence is not None and avg_confidence < 0.5)
        )
        clos.append({
            "clo_code": r["clo_code"],
            "clo_description": (r["clo_description"] or "")[:200],
            "tested_count": int(r["tested_count"]),
            "avg_confidence": round(avg_confidence, 4) if avg_confidence is not None else None,
            "avg_mastery": round(avg_mastery, 4) if avg_mastery is not None else None,
            "gap_flag": gap_flag,
        })

    return {
        "subject_code": subject_code,
        "total_clos": len(clos),
        "flagged_count": sum(1 for c in clos if c["gap_flag"]),
        "clos": clos,
    }


# ─── Worker loop ───────────────────────────────────────────────────────────────

async def process_report_job(payload_str: str) -> None:
    try:
        payload = json.loads(payload_str)
    except json.JSONDecodeError:
        logger.error("report_worker.invalid_json", payload=payload_str[:64])
        return

    report_id = payload.get("report_id", "")
    report_type = payload.get("report_type", "")
    params = payload.get("params", {})

    if not report_id:
        logger.error("report_worker.missing_report_id")
        return

    await _set_status(report_id, "running")

    try:
        if report_type == "school_monthly":
            result = await generate_school_monthly(
                school_id=params["school_id"],
                year=int(params["year"]),
                month=int(params["month"]),
            )
        elif report_type == "national_heatmap":
            result = await generate_national_heatmap(
                year=int(params["year"]),
                month=int(params["month"]),
            )
        elif report_type == "clo_coverage":
            result = await generate_clo_coverage(subject_code=params["subject_code"])
        else:
            raise ValueError(f"unknown report_type: {report_type!r}")

        await _set_status(report_id, "done", result=result)
        logger.info("report_worker.done", report_id=report_id, report_type=report_type)

    except Exception as exc:
        err = str(exc)
        logger.exception("report_worker.failed", report_id=report_id, report_type=report_type)
        await _set_status(report_id, "failed", error=err)
        await push_dead_letter(REPORT_QUEUE, payload_str, err)


async def run_forever(poll_timeout: int = 5) -> None:
    logger.info("report_worker.started", queue=REPORT_QUEUE)
    while not _shutdown.is_set():
        try:
            payload_str = await brpop_job(REPORT_QUEUE, timeout=poll_timeout)
        except Exception:
            logger.exception("report_worker.redis_error")
            await asyncio.sleep(poll_timeout)
            continue

        if payload_str is None:
            continue

        try:
            await process_report_job(payload_str)
        except Exception:
            logger.exception("report_worker.unhandled_error", payload=payload_str[:64])


def request_shutdown() -> None:
    _shutdown.set()


if __name__ == "__main__":
    try:
        asyncio.run(run_forever())
    except KeyboardInterrupt:
        pass
