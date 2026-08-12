"""
Capability 3A: The Granular Gap Analysis Engine ("Diagnostic Doctor").

Runs when an exam attempt becomes fully graded (the Go side LPUSHes the
attempt id onto queue:gap:analyze from SubmitExam / BulkGradeExam):

  Pass 1 -- Question-Level Triage (the "what"): every question that lost
  marks is traced Question -> CLO -> Topic. That topic is the SYMPTOM.

  Pass 2 -- Root Cause Traversal (the "where"): walk backwards up the
  prerequisite graph from each symptom topic (Neo4j HAS_PREREQUISITE
  first, falling back to Postgres curriculum.topic_prerequisites -- see
  app/db/neo4j.py for the write path and current population status) and
  find the deepest prerequisite the student has EVIDENCE of being weak in
  (< WEAK_THRESHOLD mastery across their graded answer history /
  mastery_records). No evidence is not weakness: a prerequisite the
  student was never assessed on is never blamed.

  Pass 3 -- LLM Synthesis (the "why"): one call turns the symptom/root-cause
  pairs into per-gap explanations plus an exam-level narrative (English +
  Amharic) -- Gemini first, falling back to a local Ollama model when
  Gemini is unset/unreachable (see gap_analysis/llm.py), with a
  deterministic English fallback so exam_insights always gets a summary
  even when neither LLM tier is available.

Storage: gap_records (granular), exam_insights (per attempt),
subject_profiles (rolling subject health) + mastery_records refresh, all
in one transaction (postgres_gap.persist_analysis).
"""

from __future__ import annotations

import structlog

from app.db import neo4j as neo4j_db
from app.db import postgres_gap as db
from app.services.gap_analysis.llm import MAX_GAPS_TO_EXPLAIN, synthesize_insights

logger = structlog.get_logger()

# Below this mastery ratio (0..1) a prerequisite counts as a break in the
# chain. Matches the attempt-level pass threshold (50%).
WEAK_THRESHOLD = 0.5
MAX_PREREQ_DEPTH = 3


async def process_gap_job(attempt_id: str) -> None:
    attempt = await db.fetch_attempt(attempt_id)
    if attempt is None:
        logger.warning("gap_analysis.attempt_not_found", attempt_id=attempt_id)
        return
    if attempt["total_score"] is None:
        # Queued but no longer fully graded (e.g. a teacher re-opened a
        # grade between the push and now) -- it will be re-queued when
        # grading finalizes again.
        logger.warning("gap_analysis.attempt_not_finalized", attempt_id=attempt_id)
        return

    student_id = str(attempt["student_id"])
    logger.info(
        "gap_analysis.started",
        attempt_id=attempt_id,
        student_id=student_id,
        exam_id=str(attempt["exam_id"]),
    )

    missed = await db.fetch_missed_answers(attempt_id)

    # ── Pass 1: Question-Level Triage ────────────────────────────────
    clo_codes = sorted({r["clo_code"] for r in missed if r["clo_code"] and r["topic_id"] is None})
    clo_topics = await db.resolve_topics_for_clos(clo_codes)
    direct_topic_ids = sorted({str(r["topic_id"]) for r in missed if r["topic_id"] is not None})
    direct_topics = await db.fetch_topics(direct_topic_ids)

    gaps: list[dict] = []
    unmapped = 0
    for r in missed:
        if r["topic_id"] is not None:
            topic = direct_topics.get(str(r["topic_id"]))
            topic_id, topic_title = str(r["topic_id"]), topic["title_en"] if topic else "Unknown topic"
        elif r["clo_code"] in clo_topics:
            t = clo_topics[r["clo_code"]]
            topic_id, topic_title = str(t["topic_id"]), t["title_en"]
        else:
            # No CLO and no topic on the question -- nothing to trace.
            unmapped += 1
            continue

        possible = float(r["marks_possible"])
        awarded = float(r["marks_awarded"])
        gaps.append(
            {
                "question_id": str(r["question_id"]),
                "question_text": r["question_text"],
                "sequence_number": r["sequence_number"],
                "clo_code": r["clo_code"],
                "symptom_topic_id": topic_id,
                "symptom_topic_title": topic_title,
                "severity": round(1.0 - (awarded / possible), 4) if possible > 0 else 1.0,
            }
        )

    # ── Pass 2: Root Cause Traversal ─────────────────────────────────
    symptom_ids = sorted({g["symptom_topic_id"] for g in gaps})
    chains: dict[str, list[dict]] = {}
    for topic_id in symptom_ids:
        chain = await neo4j_db.fetch_prerequisite_chain(topic_id, MAX_PREREQ_DEPTH)
        if not chain:
            chain = await db.fetch_prerequisite_chain_pg(topic_id, MAX_PREREQ_DEPTH)
        chains[topic_id] = chain

    prereq_ids = sorted({c["id"] for chain in chains.values() for c in chain})
    mastery = await db.fetch_topic_mastery(student_id, symptom_ids + prereq_ids)

    root_causes: dict[str, dict] = {}
    for topic_id, chain in chains.items():
        weak = [c for c in chain if mastery.get(c["id"], 1.0) < WEAK_THRESHOLD]
        if weak:
            # The break in the chain: the deepest weak prerequisite (the
            # most foundational), lowest mastery as tie-breaker.
            root_causes[topic_id] = max(
                weak, key=lambda c: (c["depth"], -mastery.get(c["id"], 1.0))
            )

    for g in gaps:
        rc = root_causes.get(g["symptom_topic_id"])
        if rc:
            g["root_cause_topic_id"] = rc["id"]
            g["root_cause_topic_title"] = rc["title"]
            g["root_cause_grade"] = rc.get("grade_level")
            g["prerequisite_depth"] = rc["depth"]
        else:
            g["root_cause_topic_id"] = None
            g["prerequisite_depth"] = 0

    # ── Pass 3: LLM Synthesis ────────────────────────────────────────
    worst_first = sorted(range(len(gaps)), key=lambda i: -gaps[i]["severity"])
    to_explain = worst_first[:MAX_GAPS_TO_EXPLAIN]
    gap_contexts = [
        {
            "index": i,
            "question_text": gaps[i]["question_text"],
            "symptom_topic": gaps[i]["symptom_topic_title"],
            "root_cause_topic": gaps[i].get("root_cause_topic_title"),
            "root_cause_grade": gaps[i].get("root_cause_grade"),
            "severity": gaps[i]["severity"],
        }
        for i in to_explain
    ]

    explanations, summary, llm_model = await synthesize_insights(
        exam_title=attempt["exam_title"],
        exam_scope=attempt["exam_scope"],
        subject_code=attempt["subject_code"],
        grade_level=attempt["grade_level"],
        percentage=float(attempt["percentage"]) if attempt["percentage"] is not None else None,
        gap_contexts=gap_contexts,
    )
    if explanations:
        for i, text in explanations.items():
            if 0 <= i < len(gaps):
                gaps[i]["llm_explanation"] = text
    if summary is None:
        summary = _fallback_summary(attempt, gaps, unmapped)

    # ── Storage: granular + aggregate layers ─────────────────────────
    subject_agg = await db.compute_subject_aggregate(student_id, attempt["subject_code"])
    await db.persist_analysis(
        attempt=attempt,
        gaps=gaps,
        exam_summary=summary,
        llm_model=llm_model,
        mastery_updates=mastery,
        subject_agg=subject_agg,
    )

    logger.info(
        "gap_analysis.completed",
        attempt_id=attempt_id,
        gaps=len(gaps),
        root_causes=len(root_causes),
        unmapped_questions=unmapped,
        llm_used=llm_model is not None,
        subject_mastery_pct=subject_agg["mastery_pct"],
    )


def _fallback_summary(attempt, gaps: list[dict], unmapped: int) -> str:
    """Deterministic English summary when Gemini is unavailable -- the
    Exam Insight layer must never be empty for an analyzed attempt."""
    pct = round(float(attempt["percentage"]), 1) if attempt["percentage"] is not None else 0.0
    verdict = "passed" if attempt["passed"] else "did not pass"
    if not gaps and not unmapped:
        return f"Scored {pct}% and {verdict}. No knowledge gaps detected on this exam -- great work."

    topics = sorted(
        {(g["symptom_topic_title"], g["severity"]) for g in gaps}, key=lambda t: -t[1]
    )
    weak_list = ", ".join(t[0] for t in topics[:3]) or "unmapped topics"
    parts = [
        f"Scored {pct}% and {verdict}.",
        f"{len(gaps)} question(s) lost marks, concentrated in: {weak_list}.",
    ]
    root_titles = sorted({g["root_cause_topic_title"] for g in gaps if g.get("root_cause_topic_id")})
    if root_titles:
        parts.append(
            "Root-cause analysis points to earlier prerequisite gaps in: "
            + ", ".join(root_titles) + " -- review these first."
        )
    return " ".join(parts)
