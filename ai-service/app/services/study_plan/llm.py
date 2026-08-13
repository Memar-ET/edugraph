"""
Step 5 (PRD Capability 3B): per-day LLM enrichment for the study plan.

The PRD names Qwen-7B specifically for this step ("the AI service (Qwen-7B)
generates a plain-language description for each day's study goal") --
this codebase's actual convention everywhere else (gap analysis, the
tutor, exam CLO matching) is Gemini-first with a local Ollama model as
the offline fallback, so that's what's implemented here too, for
consistency with the rest of the system rather than literally
Qwen-7B-only. Same two-tier resilience contract as gap_analysis/llm.py:
any failure returns (None, None) and the caller keeps the plan's existing
per-block "why" text, which was always there and never depended on this
enrichment step -- a missing/slow LLM must never block plan generation.

One call per plan (not per day), same O(plans)-not-O(days) reasoning as
gap_analysis's O(exams)-not-O(questions) batching.

Amharic: same finding as gap_analysis and the tutor -- qwen2.5/gemma2
both produce garbled Amharic, verified directly, and Amharic isn't
mandatory at this stage (confirmed). Gemini's prompt is always bilingual
(English + Amharic, same as gap_analysis's), matching the "generate both,
let the frontend pick" convention already used elsewhere in this
codebase; Ollama's is unconditionally English-only, regardless of the
plan's requested language, the same as gap_analysis's Ollama tier.
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
OLLAMA_TIMEOUT_SECONDS = 90.0


def _build_prompt(days: list[dict], exam_days_away: Optional[int], english_only: bool) -> str:
    day_lines = []
    for d in days:
        blocks = ", ".join(
            f'"{b["title"]}" ({b["hours"]}h{", root cause" if b["isRootCause"] else ""})'
            for b in d["blocks"]
        )
        day_lines.append(f'Day {d["day"]}: {blocks}')

    exam_line = (
        f"The plan is targeting an exam in {exam_days_away} day(s)."
        if exam_days_away is not None
        else "No specific exam deadline was given for this plan."
    )
    lang_instruction = (
        "Write each goal in simple English only."
        if english_only
        else "Write each goal in simple English, then the same sentence in Amharic."
    )

    return (
        "You are writing short daily study goals for an Ethiopian K-12 "
        "student's personalized study plan, built from their diagnosed "
        f"knowledge gaps and prerequisite chain. {exam_line}\n\n"
        + "\n".join(day_lines)
        + "\n\nFor each day, write ONE short, motivating sentence in the "
        'style of "Today: master Vector Addition -- this unlocks '
        "Newton's Second Law which appears on your exam in 12 days.\" "
        "Reference the exam countdown when relevant and explain briefly "
        f"why today's topic matters for what comes after it. {lang_instruction}\n\n"
        "Respond with ONLY a JSON object of the exact shape "
        '{"dayGoals": [{"day": <int>, "goal": "<text>"}]}.'
    )


def _parse_response(text: str) -> Optional[dict[int, str]]:
    parsed = json.loads(text)
    goals: dict[int, str] = {}
    for item in parsed.get("dayGoals") or []:
        try:
            day = int(item["day"])
        except (KeyError, TypeError, ValueError):
            continue
        if isinstance(item.get("goal"), str) and item["goal"].strip():
            goals[day] = item["goal"].strip()
    return goals or None


async def _enrich_via_gemini(prompt: str) -> Optional[dict[int, str]]:
    if not settings.GEMINI_API_KEY:
        return None
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
            return _parse_response(data["candidates"][0]["content"]["parts"][0]["text"])
    except Exception:  # noqa: BLE001 -- never let an LLM hiccup block plan generation
        logger.exception("study_plan.gemini_enrich_failed")
        return None


def _ollama_url() -> str:
    host = settings.OLLAMA_HOST
    if "://" not in host:
        host = "http://" + host
    return host.rstrip("/") + "/api/generate"


async def _enrich_via_ollama(prompt: str) -> Optional[dict[int, str]]:
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
            return _parse_response(resp.json()["response"])
    except Exception:  # noqa: BLE001 -- never let an LLM hiccup block plan generation
        logger.exception("study_plan.ollama_enrich_failed")
        return None


async def enrich_days(
    days: list[dict], exam_days_away: Optional[int]
) -> tuple[Optional[dict[int, str]], Optional[str]]:
    """Returns ({day_number: goal_text}, model_used), or (None, None) if
    both tiers failed -- the caller's per-block "why" text (already
    present regardless) is the only fallback needed, so there's no third
    tier here the way gap_analysis has _fallback_summary."""
    if not days:
        return None, None

    prompt = _build_prompt(days, exam_days_away, english_only=False)
    goals = await _enrich_via_gemini(prompt)
    if goals:
        return goals, GEMINI_MODEL

    # Unconditionally English-only for Ollama, same as gap_analysis --
    # qwen2.5/gemma2 can't do Amharic reliably (verified directly), so
    # this doesn't vary by the requested language the way the tutor's
    # does (a live chat response needing a language-aware fallback note
    # is a different situation from a stored plan's day-goal text).
    ollama_prompt = _build_prompt(days, exam_days_away, english_only=True)
    goals = await _enrich_via_ollama(ollama_prompt)
    if goals:
        return goals, settings.OLLAMA_MODEL

    return None, None
