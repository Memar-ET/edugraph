"""
Pass 3: LLM synthesis (Capability 3A).

One request per analyzed attempt (not per gap -- cost stays O(exams), not
O(questions)): the symptom, root cause, and question text for every gap go
up together, and one JSON response comes back with a per-gap explanation
plus the exam-level narrative summary.

Provider selection (local vs. cloud, checklist 8.1/8.3) is entirely
app/utils/llm_provider.py's job -- this module just builds the prompt
(varying it by provider.supports_amharic) and parses the response. If
neither provider produces anything, the caller falls back further to a
deterministic English summary (service.py's _fallback_summary), which
always produces *something* -- an LLM hiccup, or simply not having any
LLM configured, must never fail the analysis.
"""

from __future__ import annotations

import json
from typing import Optional

import structlog

from app.utils.llm_provider import LLMProvider, generate_with_fallback

logger = structlog.get_logger()

# Cap how many gaps get an LLM explanation -- the worst ones matter most,
# and this bounds prompt size on a 50-question final where a student had
# a very bad day. Gaps beyond the cap keep llm_explanation NULL.
MAX_GAPS_TO_EXPLAIN = 10


def _build_prompt(
    exam_title: str,
    exam_scope: str,
    subject_code: str,
    grade_level: int,
    percentage: Optional[float],
    gap_contexts: list[dict],
    provider: LLMProvider,
) -> str:
    gap_lines = []
    for g in gap_contexts:
        line = (
            f'Gap {g["index"]}: Question "{g["question_text"][:300]}" '
            f'(marks lost: {round(g["severity"] * 100)}%) tests the topic '
            f'"{g["symptom_topic"]}".'
        )
        if g.get("root_cause_topic"):
            line += (
                f' Root-cause analysis shows the student is weak in the prerequisite '
                f'"{g["root_cause_topic"]}" (Grade {g.get("root_cause_grade", "?")}).'
            )
        else:
            line += " No broken prerequisite was found; the gap is in this topic itself."
        gap_lines.append(line)

    if provider.supports_amharic:
        explanation_instruction = (
            "Write each explanation in English followed by the same message in Amharic."
        )
        summary_instruction = "Also write examSummary: 2-3 sentences (English, then Amharic)"
        shape_hint = '{"gaps": [{"index": <int>, "explanation": "<en + am>"}], "examSummary": "<en + am>"}.'
    else:
        explanation_instruction = "Write each explanation in simple English only."
        summary_instruction = "Also write examSummary: 2-3 sentences (English only)"
        shape_hint = '{"gaps": [{"index": <int>, "explanation": "<en>"}], "examSummary": "<en>"}.'

    return (
        "You are a diagnostic tutor for Ethiopian K-12 students. A student "
        f"took \"{exam_title}\" ({exam_scope.replace('_', ' ')}, {subject_code}, "
        f"Grade {grade_level}) and scored {round(percentage or 0, 1)}%.\n\n"
        "For each knowledge gap below, explain in ONE short, simple, "
        "encouraging sentence WHY the student likely missed that question -- "
        "connecting it to the root cause when one is given (e.g. \"You "
        "couldn't solve the force problem because you didn't add the vectors "
        f"correctly first.\"). {explanation_instruction}\n\n"
        + "\n".join(gap_lines)
        + f"\n\n{summary_instruction} "
        "summarizing the whole exam -- what went well, the main weak area, "
        "and the single most important thing to review first.\n\n"
        "Respond with ONLY a JSON object of the exact shape " + shape_hint
    )


def _parse_response(text: str) -> tuple[Optional[dict[int, str]], Optional[str]]:
    parsed = json.loads(text)

    explanations: dict[int, str] = {}
    for item in parsed.get("gaps") or []:
        try:
            idx = int(item["index"])
        except (KeyError, TypeError, ValueError):
            continue
        if isinstance(item.get("explanation"), str) and item["explanation"].strip():
            explanations[idx] = item["explanation"].strip()

    summary = parsed.get("examSummary")
    if not isinstance(summary, str) or not summary.strip():
        summary = None
    return (explanations or None), summary


async def synthesize_insights(
    exam_title: str,
    exam_scope: str,
    subject_code: str,
    grade_level: int,
    percentage: Optional[float],
    gap_contexts: list[dict],
) -> tuple[Optional[dict[int, str]], Optional[str], Optional[str]]:
    """
    gap_contexts: [{"index", "question_text", "symptom_topic",
    "root_cause_topic" (nullable), "root_cause_grade" (nullable),
    "severity"}]. Returns ({index: explanation}, exam_summary, model_used)
    -- model_used is None (not a fixed constant) when every provider
    failed, so the caller can honestly record which model actually
    produced the text (or that none did) instead of assuming one.
    """
    if not gap_contexts:
        return None, None, None

    def build(provider: LLMProvider) -> str:
        return _build_prompt(exam_title, exam_scope, subject_code, grade_level, percentage, gap_contexts, provider)

    text, provider_used = await generate_with_fallback(build, json_mode=True)
    if text is None or provider_used is None:
        return None, None, None

    try:
        explanations, summary = _parse_response(text)
    except Exception:  # noqa: BLE001 -- a malformed response must never fail the analysis
        logger.exception("gap_analysis.response_parse_failed", model=provider_used.model_name)
        return None, None, None
    if not explanations and not summary:
        return None, None, None
    return explanations, summary, provider_used.model_name
