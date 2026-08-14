"""
Capability 3C: The AI Tutor (Graph-RAG + LLM).

When a student asks a question, retrieval runs over BOTH stores before
any LLM sees anything:

  1. Match the question to curriculum topics (keyword overlap against
     title/description/key_concepts, student's grade and below).
  2. Check the student's gap_records FIRST: unresolved gaps where a
     matched topic is the symptom or the root cause, including 3A's
     stored explanations.
  3. Pull the matched topics' prerequisite chains from the graph (Neo4j,
     Postgres fallback -- same dual path as 3A/3B).

All of it is injected as context: "the student recently failed a question
on this because they don't understand X -- tailor the answer to bridge
that specific gap."

Provider selection (local vs. cloud, checklist 8.1/8.3) is entirely
app/utils/llm_provider.py's job, same as gap_analysis and study_plan.
Unlike those, there's no third, deterministic-text tier here: an
open-ended tutoring answer can't be synthesized without an LLM, so
TutorUnavailable (-> HTTP 503) is still raised, just only once BOTH
providers have failed -- a tutor that can't answer shouldn't pretend to.

Amharic: LLMProvider.supports_amharic tells this module whether to
request the actual requested language or force English -- qwen2.5/
gemma2 (the local models actually tested against this deployment) both
produce garbled, non-functional Amharic (verified directly, not
assumed). Rather than show a student broken Ge'ez-script text, a
non-Amharic-capable provider always answers in English regardless of
what was requested, and the response notes that Amharic needs a cloud
connection.
"""

from __future__ import annotations

import re

import structlog

from app.db import neo4j as neo4j_db
from app.db import postgres_gap as gap_db
from app.db import postgres_tutor as db
from app.utils.llm_provider import LLMProvider, generate_with_fallback

logger = structlog.get_logger()

MAX_TOPICS = 3
_AMHARIC_UNAVAILABLE_NOTE = (
    "\n\n(Offline mode: this answer was generated locally and is only "
    "available in English right now -- Amharic needs an internet connection.)"
)

_WORD = re.compile(r"[a-z]{3,}")
_STOPWORDS = frozenset(
    "the and for with that this from what why how are you can does when where "
    "which will would could should about into your very them they have has had "
    "not but its was were there here their then than also more most".split()
)


class TutorUnavailable(Exception):
    """Raised when both LLM providers (configured + fallback) are unset/unreachable."""


def _tokenize(text: str) -> set[str]:
    return {w for w in _WORD.findall(text.lower()) if w not in _STOPWORDS}


def _score_topic(question_tokens: set[str], topic) -> float:
    title_tokens = _tokenize(topic["title_en"] or "")
    concept_tokens = _tokenize(" ".join(topic["key_concepts"] or []))
    desc_tokens = _tokenize(topic["description"] or "")
    # Title/key-concept hits are far stronger signals than prose overlap.
    return (
        2.0 * len(question_tokens & title_tokens)
        + 1.5 * len(question_tokens & concept_tokens)
        + 0.5 * len(question_tokens & desc_tokens)
    )


def _lang_instruction(language: str) -> str:
    return {
        "am": "Answer in Amharic.",
        "en": "Answer in simple English.",
    }.get(language, "Answer in simple English, then repeat the key point in Amharic.")


def _build_prompt(grade_level, context_parts: list[str], question: str, lang_instruction: str) -> str:
    return (
        "You are a patient one-on-one tutor for an Ethiopian K-12 student"
        + (f" in Grade {grade_level}" if grade_level else "")
        + ".\n\n"
        + ("STUDENT CONTEXT (from their diagnostic records -- use it to tailor "
           "the answer and bridge their specific gaps, but never recite record "
           "ids or scores back at them):\n" + "\n".join(context_parts) + "\n\n"
           if context_parts else "")
        + f'STUDENT QUESTION: "{question}"\n\n'
        + "Teach, don't lecture: a short direct answer first, then the "
        "explanation. If the context shows a broken prerequisite, start from "
        f"that prerequisite and build up to their question. {lang_instruction}"
    )


async def ask(student_id: str, question: str, language: str = "en") -> dict:
    student = await db.fetch_student(student_id)
    grade_level = student["grade_level"] if student else None

    # ── Retrieval 1: question -> curriculum topics ───────────────────
    question_tokens = _tokenize(question)
    candidates = await db.fetch_candidate_topics(grade_level)
    scored = [(t, _score_topic(question_tokens, t)) for t in candidates]
    matched = [t for t, s in sorted(scored, key=lambda x: -x[1]) if s > 0][:MAX_TOPICS]
    matched_ids = [str(t["id"]) for t in matched]

    # ── Retrieval 2: the student's own gaps on those topics ──────────
    gaps = await db.fetch_gap_context(student_id, matched_ids) if matched_ids else []

    # ── Retrieval 3: prerequisite chains + the student's mastery ─────
    prereq_lines: list[str] = []
    if matched_ids:
        chain = await neo4j_db.fetch_prerequisite_chain(matched_ids[0])
        if not chain:
            chain = await gap_db.fetch_prerequisite_chain_pg(matched_ids[0])
        if chain:
            mastery = await gap_db.fetch_topic_mastery(student_id, [c["id"] for c in chain])
            for c in chain:
                m = mastery.get(c["id"])
                level = f"{round(m * 100)}% mastery" if m is not None else "not yet assessed"
                prereq_lines.append(
                    f'- "{c["title"]}" (Grade {c["grade_level"]}, depth {c["depth"]}): {level}'
                )

    # ── Context injection ────────────────────────────────────────────
    context_parts: list[str] = []
    if matched:
        context_parts.append(
            "The question appears to be about: "
            + "; ".join(f'"{t["title_en"]}" (Grade {t["grade_level"]}, {t["subject_code"]})' for t in matched)
            + "."
        )
    for g in gaps:
        note = (
            f'Note: this student recently lost marks on "{g["symptom_title"]}" '
            f"(severity {round(float(g['severity_score']) * 100)}%)"
        )
        if g["root_cause_title"]:
            note += (
                f' and root-cause analysis showed the real break is in the prerequisite '
                f'"{g["root_cause_title"]}" (Grade {g["root_cause_grade"]})'
            )
        note += "."
        if g["llm_explanation"]:
            note += f' Earlier diagnostic explanation: "{g["llm_explanation"]}"'
        context_parts.append(note)
    if prereq_lines:
        context_parts.append("Prerequisite chain for the main topic:\n" + "\n".join(prereq_lines))

    def build(provider: LLMProvider) -> str:
        effective_language = language if provider.supports_amharic else "en"
        return _build_prompt(grade_level, context_parts, question, _lang_instruction(effective_language))

    answer, provider_used = await generate_with_fallback(build)
    if answer is None or provider_used is None:
        raise TutorUnavailable("the tutoring model is unreachable")

    model_used = provider_used.model_name
    if not provider_used.supports_amharic and language != "en":
        answer += _AMHARIC_UNAVAILABLE_NOTE

    logger.info(
        "tutor.answered",
        student_id=student_id,
        matched_topics=len(matched),
        gaps_used=len(gaps),
        model=model_used,
    )
    return {
        "answer": answer,
        "relatedTopics": [
            {"topicId": str(t["id"]), "title": t["title_en"], "gradeLevel": t["grade_level"]}
            for t in matched
        ],
        "usedGaps": [
            {
                "gapId": str(g["id"]),
                "symptomTopic": g["symptom_title"],
                "rootCauseTopic": g["root_cause_title"],
                "severity": float(g["severity_score"]),
            }
            for g in gaps
        ],
        "model": model_used,
    }
