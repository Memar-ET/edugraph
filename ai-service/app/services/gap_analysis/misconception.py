"""
Structured misconception modeling (EG-GCKT Milestone 6, spec section 11).

The spec is explicit that misconceptions should be represented as
structured hypotheses linked to evidence and a verification item, not
just mentioned as free-text indicators (which is all the original
gap-analysis LLM synthesis produced -- prose inside
exam_insights.llm_exam_summary, never a queryable, reviewable record).

Trigger: within one attempt, when >= MIN_PATTERN_SIZE questions mapped to
the SAME symptom topic were all missed, that's a structured pattern
(repeated failure on one topic, not one random slip) worth hypothesizing
about -- the module asks the LLM to propose a specific misconception
given the actual wrong-answer text, not just "the student is weak here"
(which gap_records/root_cause.py already capture without needing an LLM).
Every hypothesis is inserted as status='candidate' -- nothing here ever
auto-confirms a misconception; a teacher reviews and confirms/rejects it
via the Go endpoint (assessment/handler/misconceptions.go).
"""

from __future__ import annotations

import json
from collections import defaultdict
from typing import Optional

import structlog

from app.db import postgres_gap as db
from app.utils.llm_provider import LLMProvider, generate_with_fallback

logger = structlog.get_logger()

MIN_PATTERN_SIZE = 2
MAX_TOPICS_PER_ATTEMPT = 5  # bound worst-case LLM calls on a very bad exam


def _build_prompt(topic_title: str, wrong_answers: list[dict], provider: LLMProvider) -> str:
    lines = [
        f'Q: "{a["question_text"][:300]}" -> student answered: "{(a["answer_text"] or "").strip()[:200] or "(no answer)"}"'
        for a in wrong_answers
    ]
    language_instruction = (
        "Write misconceptionText and interventionText in English only."
        if not provider.supports_amharic
        else "Write misconceptionText and interventionText in English only (Amharic is not needed here)."
    )
    return (
        "You are an expert diagnostic tutor for Ethiopian K-12 education. A student "
        f'missed {len(wrong_answers)} questions in a row on the topic "{topic_title}":\n\n'
        + "\n".join(lines)
        + "\n\nIf these wrong answers share a common, SPECIFIC conceptual misunderstanding "
        "(not just 'doesn't know the topic'), describe it. If there is no clear shared "
        "misconception (e.g. the answers look like random guesses or unrelated mistakes), "
        "say so explicitly.\n\n"
        f"{language_instruction}\n\n"
        "Respond with ONLY a JSON object of the exact shape: "
        '{"hasMisconception": <bool>, "misconceptionText": "<specific claim, or empty string>", '
        '"triggerPattern": "<short description of the shared wrong-answer pattern>", '
        '"confidence": <0..1>, "interventionText": "<one short suggested fix, or empty string>"}.'
    )


def _parse_response(text: str) -> Optional[dict]:
    parsed = json.loads(text)
    if not parsed.get("hasMisconception"):
        return None
    misconception_text = parsed.get("misconceptionText")
    if not isinstance(misconception_text, str) or not misconception_text.strip():
        return None
    confidence = parsed.get("confidence")
    try:
        confidence = max(0.0, min(1.0, float(confidence)))
    except (TypeError, ValueError):
        confidence = 0.5
    return {
        "misconception_text": misconception_text.strip(),
        "trigger_pattern": (parsed.get("triggerPattern") or "").strip() or None,
        "confidence": confidence,
        "intervention_text": (parsed.get("interventionText") or "").strip() or None,
    }


async def generate_hypotheses(student_id: str, school_id: str, gaps: list[dict]) -> int:
    """gaps: the same list service.py's Pass 1/2 builds (each carrying
    symptom_topic_id, symptom_topic_title, question_text, answer_text).
    Groups by symptom topic, and for every topic with a structured
    multi-question failure pattern:
    - if a 'candidate' hypothesis already exists for this (student,
      topic), appends the new wrong answers to its supporting_evidence
      instead of asking the LLM again (spec: "track evidence supporting
      or weakening a misconception hypothesis") -- a topic re-triggering
      this pattern across multiple exams shouldn't spawn duplicate
      hypotheses.
    - otherwise asks the LLM for a new candidate misconception, assigns a
      verification item (a not-yet-answered question mapped to the
      topic), and persists it.
    Returns how many hypotheses were created or updated -- purely
    informational for the caller's log line, callers must not treat 0 as
    an error (most attempts won't have a repeated pattern on any single
    topic, which is the normal case)."""
    by_topic: dict[str, dict] = defaultdict(lambda: {"title": None, "answers": []})
    for g in gaps:
        topic_id = g.get("symptom_topic_id")
        if not topic_id:
            continue
        entry = by_topic[topic_id]
        entry["title"] = g.get("symptom_topic_title") or entry["title"]
        entry["answers"].append(
            {
                "question_id": g.get("question_id"),
                "question_text": g.get("question_text"),
                "answer_text": g.get("answer_text"),
            }
        )

    qualifying = [(tid, e) for tid, e in by_topic.items() if len(e["answers"]) >= MIN_PATTERN_SIZE]
    qualifying.sort(key=lambda item: -len(item[1]["answers"]))
    qualifying = qualifying[:MAX_TOPICS_PER_ATTEMPT]

    written = 0
    for topic_id, entry in qualifying:
        existing = await db.fetch_candidate_misconception(student_id, topic_id)
        if existing is not None:
            await db.append_misconception_evidence(str(existing["id"]), entry["answers"])
            written += 1
            continue

        def build(provider: LLMProvider, _title=entry["title"], _answers=entry["answers"]) -> str:
            return _build_prompt(_title or "this topic", _answers, provider)

        text, provider_used = await generate_with_fallback(build, json_mode=True)
        if text is None:
            continue
        try:
            hypothesis = _parse_response(text)
        except Exception:  # noqa: BLE001 -- a malformed response must never fail gap analysis
            logger.exception("misconception.response_parse_failed", topic_id=topic_id)
            continue
        if hypothesis is None:
            continue

        exclude_ids = [a["question_id"] for a in entry["answers"] if a.get("question_id")]
        verification_item_id = await db.select_verification_item(topic_id, exclude_ids)

        await db.insert_misconception_hypothesis(
            student_id=student_id,
            school_id=school_id,
            topic_id=topic_id,
            misconception_text=hypothesis["misconception_text"],
            trigger_pattern=hypothesis["trigger_pattern"],
            supporting_evidence={"answers": entry["answers"]},
            confidence=hypothesis["confidence"],
            intervention_text=hypothesis["intervention_text"],
            generated_by_model=provider_used.model_name if provider_used else None,
            verification_item_id=verification_item_id,
        )
        written += 1

    return written


# Confidence nudges when a verification item is answered -- small,
# bounded moves, never enough on their own to flip a candidate to
# confirmed (that still requires a teacher, per spec: "prevent a
# misconception from being treated as confirmed without sufficient
# evidence").
VERIFICATION_CORRECT_DELTA = -0.15  # weakens the hypothesis
VERIFICATION_INCORRECT_DELTA = 0.1  # strengthens the hypothesis


async def check_verification_responses(student_id: str, answers: list[dict]) -> int:
    """answers: [{"question_id", "correct"}] for every answer on a just-
    graded attempt (not just missed ones -- a CORRECT answer to a
    verification item is exactly the "weakens" signal). Checked against
    every candidate hypothesis with a verification item assigned;
    returns how many hypotheses were adjusted."""
    active = await db.fetch_active_verification_items(student_id)
    if not active:
        return 0
    by_question: dict[str, list] = defaultdict(list)
    for h in active:
        by_question[str(h["verification_item_id"])].append(h)

    adjusted = 0
    for a in answers:
        question_id = str(a.get("question_id")) if a.get("question_id") else None
        if question_id not in by_question:
            continue
        correct = a.get("correct")
        if correct is None:
            continue
        delta = VERIFICATION_CORRECT_DELTA if correct else VERIFICATION_INCORRECT_DELTA
        note = (
            "Verification item answered correctly -- weakens this hypothesis."
            if correct
            else "Verification item also answered incorrectly -- strengthens this hypothesis."
        )
        for h in by_question[question_id]:
            await db.adjust_misconception_confidence(str(h["id"]), delta, note)
            adjusted += 1

    return adjusted
