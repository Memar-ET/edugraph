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
  score every sufficiently-weak candidate in the chain with the EG-GCKT
  Root Cause Score (spec section 9, app/services/gap_analysis/
  root_cause.py) rather than simply picking the deepest weak node --
  RCS weighs weakness against evidence confidence, how many other topics
  depend on fixing this one, whether the candidate's OWN prerequisites are
  ready, and the resulting intervention gain, so a numerically "weaker"
  downstream symptom doesn't get blamed ahead of the upstream topic that
  actually explains it (see the module's docstring for the worked
  example). No evidence is not weakness: a prerequisite the student was
  never assessed on is never blamed.

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
from app.services.gap_analysis import misconception, root_cause
from app.services.gap_analysis.llm import MAX_GAPS_TO_EXPLAIN, synthesize_insights

logger = structlog.get_logger()

# The weak-mastery cutoff now lives in root_cause.WEAK_THRESHOLD (Milestone
# 5) -- kept as one shared value rather than a second copy that could
# silently drift from it.
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
                "answer_text": r["answer_text"],
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
        rc = await root_cause.score_candidates(student_id, chain, mastery)
        if rc:
            root_causes[topic_id] = rc

    for g in gaps:
        rc = root_causes.get(g["symptom_topic_id"])
        if rc:
            g["root_cause_topic_id"] = rc["id"]
            g["root_cause_topic_title"] = rc["title"]
            g["root_cause_grade"] = rc.get("grade_level")
            g["prerequisite_depth"] = rc["depth"]
            g["rcs_score"] = rc.get("rcs")
            g["rcs_factors"] = rc.get("factors")
            g["root_cause_path"] = rc.get("path")
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

    # ── Milestone 6: structured misconception hypotheses ─────────────
    # Runs after persist_analysis (not before) since it's a separate,
    # best-effort enrichment -- an LLM/parse failure here must never
    # affect gap_records/exam_insights, which are already durably written
    # by this point.
    misconceptions_written = 0
    verifications_checked = 0
    try:
        misconceptions_written = await misconception.generate_hypotheses(student_id, str(attempt["school_id"]), gaps)
        # Checked against EVERY answer on this attempt (not just missed
        # ones) -- a correctly-answered verification item is itself
        # evidence, weakening whatever hypothesis it was meant to test.
        all_answers = await db.fetch_all_attempt_answers(attempt_id)
        verifications_checked = await misconception.check_verification_responses(
            student_id, [{"question_id": str(a["question_id"]), "correct": a["correct"]} for a in all_answers]
        )
    except Exception:  # noqa: BLE001 -- misconception generation is enrichment, not core analysis
        logger.exception("gap_analysis.misconception_generation_failed", attempt_id=attempt_id)

    logger.info(
        "gap_analysis.completed",
        attempt_id=attempt_id,
        gaps=len(gaps),
        root_causes=len(root_causes),
        unmapped_questions=unmapped,
        llm_used=llm_model is not None,
        subject_mastery_pct=subject_agg["mastery_pct"],
        misconceptions_written=misconceptions_written,
        verifications_checked=verifications_checked,
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
