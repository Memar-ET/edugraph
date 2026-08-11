"""
Pass 3: LLM synthesis with Gemini (Capability 3A).

One request per analyzed attempt (not per gap -- cost stays O(exams), not
O(questions)): the symptom, root cause, and question text for every gap go
up together, and one JSON response comes back with a per-gap explanation
plus the exam-level narrative summary.

Same resilience contract as exam_parser/clo_matcher_llm.py: when
GEMINI_API_KEY is unset, the request fails, or the response can't be
parsed, return (None, None) and let the caller fall back to a
deterministic summary -- an LLM hiccup must never fail the analysis.
"""

from __future__ import annotations

import json
from typing import Optional

import httpx
import structlog

from app.core.config import settings

logger = structlog.get_logger()

GEMINI_MODEL = "gemini-flash-latest"
GEMINI_URL = f"https://generativelanguage.googleapis.com/v1beta/models/{GEMINI_MODEL}:generateContent"
TIMEOUT_SECONDS = 30.0

# Cap how many gaps get an LLM explanation -- the worst ones matter most,
# and this bounds prompt size on a 50-question final where a student had
# a very bad day. Gaps beyond the cap keep llm_explanation NULL.
MAX_GAPS_TO_EXPLAIN = 10


async def synthesize_insights(
    exam_title: str,
    exam_scope: str,
    subject_code: str,
    grade_level: int,
    percentage: Optional[float],
    gap_contexts: list[dict],
) -> tuple[Optional[dict[int, str]], Optional[str]]:
    """
    gap_contexts: [{"index", "question_text", "symptom_topic",
    "root_cause_topic" (nullable), "root_cause_grade" (nullable),
    "severity"}]. Returns ({index: explanation}, exam_summary) or
    (None, None) on any failure.
    """
    if not settings.GEMINI_API_KEY or not gap_contexts:
        return None, None

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

    prompt = (
        "You are a diagnostic tutor for Ethiopian K-12 students. A student "
        f"took \"{exam_title}\" ({exam_scope.replace('_', ' ')}, {subject_code}, "
        f"Grade {grade_level}) and scored {round(percentage or 0, 1)}%.\n\n"
        "For each knowledge gap below, explain in ONE short, simple, "
        "encouraging sentence WHY the student likely missed that question -- "
        "connecting it to the root cause when one is given (e.g. \"You "
        "couldn't solve the force problem because you didn't add the vectors "
        "correctly first.\"). Write each explanation in English followed by "
        "the same message in Amharic.\n\n"
        + "\n".join(gap_lines)
        + "\n\nAlso write examSummary: 2-3 sentences (English, then Amharic) "
        "summarizing the whole exam -- what went well, the main weak area, "
        "and the single most important thing to review first.\n\n"
        "Respond with ONLY a JSON object of the exact shape "
        '{"gaps": [{"index": <int>, "explanation": "<en + am>"}], '
        '"examSummary": "<en + am>"}.'
    )

    try:
        async with httpx.AsyncClient(timeout=TIMEOUT_SECONDS) as client:
            resp = await client.post(
                GEMINI_URL,
                params={"key": settings.GEMINI_API_KEY},
                json={
                    "contents": [{"parts": [{"text": prompt}]}],
                    "generationConfig": {"responseMimeType": "application/json"},
                },
            )
            resp.raise_for_status()
            data = resp.json()
            text = data["candidates"][0]["content"]["parts"][0]["text"]
            parsed = json.loads(text)
    except Exception:  # noqa: BLE001 -- never let an LLM hiccup break analysis
        logger.exception("gap_analysis.llm_request_failed")
        return None, None

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
