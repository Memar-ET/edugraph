"""
Gemini-backed CLO matcher (Capability 2A) -- an optional upgrade over
clo_matcher.py's keyword-overlap heuristic, used to align an exam question
with the correct Ministry of Education Curriculum Learning Outcome (CLO)
it actually assesses.

Tried first (service.py) when GEMINI_API_KEY is configured; falls back to
the plain keyword matcher whenever the key is unset, the request times
out, or the response can't be parsed into a real CLO code -- this must
never be a hard failure in the parsing pipeline, same "provisional draft"
philosophy as the rest of this parser.
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
TIMEOUT_SECONDS = 15.0


async def match_clo_llm(question_text: str, clos: list[tuple[str, str]]) -> tuple[Optional[str], Optional[float]]:
    """clos: list of (code, description_en). Returns (best_code, confidence)
    or (None, None) if Gemini isn't configured, the call fails, or it finds
    no genuine match among the candidates."""
    if not settings.GEMINI_API_KEY or not clos:
        return None, None

    candidates = "\n".join(f"{code}: {desc}" for code, desc in clos)
    prompt = (
        "You are aligning a single exam question to the correct Ministry of "
        "Education Curriculum Learning Outcome (CLO) it assesses.\n\n"
        f'Question: "{question_text}"\n\n'
        f"Candidate CLOs:\n{candidates}\n\n"
        "Pick the single best-matching CLO code, or null if none of them "
        "genuinely fit. Respond with ONLY a JSON object of the exact shape "
        '{"cloCode": "<code or null>", "confidence": <0.0-1.0>}.'
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
    except Exception:  # noqa: BLE001 -- never let an LLM hiccup break parsing
        logger.exception("clo_matcher_llm.request_failed")
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
