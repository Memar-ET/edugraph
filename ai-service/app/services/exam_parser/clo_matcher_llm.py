"""
LLM-backed CLO matcher (Capability 2A) -- an optional upgrade over
clo_matcher.py's keyword-overlap heuristic, used to align an exam question
with the correct Ministry of Education Curriculum Learning Outcome (CLO)
it actually assesses.

Tried first (service.py) when vector_matcher.py doesn't produce a
confident match; falls back to the plain keyword matcher whenever
neither LLM provider is available/reachable or the response can't be
parsed into a real CLO code -- this must never be a hard failure in the
parsing pipeline, same "provisional draft" philosophy as the rest of
this parser. Provider selection (local vs. cloud, checklist 8.1/8.3) is
app/utils/llm_provider.py's job, same as gap_analysis/tutor/study_plan;
no language variation needed here since a CLO code isn't natural-language
output.
"""

from __future__ import annotations

import json
from typing import Optional

from app.utils.llm_provider import generate_with_fallback


def _build_prompt(question_text: str, clos: list[tuple[str, str]]) -> str:
    candidates = "\n".join(f"{code}: {desc}" for code, desc in clos)
    return (
        "You are aligning a single exam question to the correct Ministry of "
        "Education Curriculum Learning Outcome (CLO) it assesses.\n\n"
        f'Question: "{question_text}"\n\n'
        f"Candidate CLOs:\n{candidates}\n\n"
        "Pick the single best-matching CLO code, or null if none of them "
        "genuinely fit. Respond with ONLY a JSON object of the exact shape "
        '{"cloCode": "<code or null>", "confidence": <0.0-1.0>}.'
    )


async def match_clo_llm(question_text: str, clos: list[tuple[str, str]]) -> tuple[Optional[str], Optional[float]]:
    """clos: list of (code, description_en). Returns (best_code, confidence)
    or (None, None) if no provider is available, the call fails, or it
    finds no genuine match among the candidates."""
    if not clos:
        return None, None

    prompt = _build_prompt(question_text, clos)
    text, _provider_used = await generate_with_fallback(lambda _provider: prompt, json_mode=True)
    if text is None:
        return None, None

    try:
        parsed = json.loads(text)
    except Exception:  # noqa: BLE001 -- a malformed response must never break parsing
        return None, None

    code = parsed.get("cloCode")
    valid_codes = {c for c, _ in clos}
    if not code or code not in valid_codes:
        return None, None

    try:
        confidence = float(parsed.get("confidence"))
    except (TypeError, ValueError):
        confidence = None
    return code, confidence
