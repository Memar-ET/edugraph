"""
Pass 3: LLM synthesis (Capability 3A).

One request per analyzed attempt (not per gap -- cost stays O(exams), not
O(questions)): the symptom, root cause, and question text for every gap go
up together, and one JSON response comes back with a per-gap explanation
plus the exam-level narrative summary.

Three tiers, cheapest-quality-loss first: Gemini (best quality, needs
internet + an API key) -> Ollama (offline-capable local model, the School
Box's actual operating condition -- see PRD's "AI inference must work with
zero internet access") -> the caller's deterministic English summary
(service.py's _fallback_summary), which always produces *something*.
Same resilience contract at every tier: any failure (unset key, network
error, unparseable response) returns (None, None) and the caller moves to
the next tier -- an LLM hiccup must never fail the analysis.

Amharic: Gemini's prompt asks for English + Amharic (bilingual explanations
are the target design). Ollama gets an English-only prompt instead --
verified directly (not assumed) that qwen2.5:7b-instruct and gemma2:9b
both produce garbled, non-functional Amharic, and Amharic isn't mandatory
at this stage (confirmed), so the offline tier trades the bilingual
target for actually-usable English rather than ship broken text.
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
# Local inference on modest School Box hardware is slow -- a 7B model doing
# a multi-gap synthesis can genuinely take tens of seconds on CPU, so this
# tier gets a longer budget than the cloud call rather than reusing
# TIMEOUT_SECONDS and risking false negatives on a box that's just slow,
# not actually down.
OLLAMA_TIMEOUT_SECONDS = 90.0

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
    english_only: bool = False,
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

    if english_only:
        explanation_instruction = "Write each explanation in simple English only."
        summary_instruction = "Also write examSummary: 2-3 sentences (English only)"
        shape_hint = '{"gaps": [{"index": <int>, "explanation": "<en>"}], "examSummary": "<en>"}.'
    else:
        explanation_instruction = (
            "Write each explanation in English followed by the same message in Amharic."
        )
        summary_instruction = "Also write examSummary: 2-3 sentences (English, then Amharic)"
        shape_hint = '{"gaps": [{"index": <int>, "explanation": "<en + am>"}], "examSummary": "<en + am>"}.'

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


async def _synthesize_via_gemini(prompt: str) -> tuple[Optional[dict[int, str]], Optional[str]]:
    if not settings.GEMINI_API_KEY:
        return None, None
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
            return _parse_response(text)
    except Exception:  # noqa: BLE001 -- never let an LLM hiccup break analysis
        logger.exception("gap_analysis.gemini_request_failed")
        return None, None


def _ollama_url() -> str:
    host = settings.OLLAMA_HOST
    if "://" not in host:
        host = "http://" + host
    return host.rstrip("/") + "/api/generate"


async def _synthesize_via_ollama(prompt: str) -> tuple[Optional[dict[int, str]], Optional[str]]:
    """Offline-capable fallback -- see module docstring. Ollama's `format:
    "json"` constrains output to valid JSON the same way Gemini's
    responseMimeType does, so _parse_response works unchanged on either."""
    try:
        async with httpx.AsyncClient(timeout=OLLAMA_TIMEOUT_SECONDS) as client:
            resp = await client.post(
                _ollama_url(),
                json={
                    "model": settings.OLLAMA_MODEL,
                    "prompt": prompt,
                    "format": "json",
                    "stream": False,
                },
            )
            resp.raise_for_status()
            data = resp.json()
            return _parse_response(data["response"])
    except Exception:  # noqa: BLE001 -- never let an LLM hiccup break analysis
        logger.exception("gap_analysis.ollama_request_failed")
        return None, None


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
    -- model_used is None (not a fixed constant) when every tier failed, so
    the caller can honestly record which model actually produced the text
    (or that none did) instead of assuming Gemini.
    """
    if not gap_contexts:
        return None, None, None

    prompt = _build_prompt(exam_title, exam_scope, subject_code, grade_level, percentage, gap_contexts)

    explanations, summary = await _synthesize_via_gemini(prompt)
    if explanations or summary:
        return explanations, summary, GEMINI_MODEL

    # English-only prompt for Ollama -- see module docstring.
    ollama_prompt = _build_prompt(
        exam_title, exam_scope, subject_code, grade_level, percentage, gap_contexts, english_only=True
    )
    explanations, summary = await _synthesize_via_ollama(ollama_prompt)
    if explanations or summary:
        return explanations, summary, settings.OLLAMA_MODEL

    return None, None, None
