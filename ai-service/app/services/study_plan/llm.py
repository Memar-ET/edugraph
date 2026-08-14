"""
Step 5 (PRD Capability 3B): per-day LLM enrichment for the study plan.

The PRD names Qwen-7B specifically for this step ("the AI service (Qwen-7B)
generates a plain-language description for each day's study goal") --
matches this codebase's checklist-8.1 decision to build local-first by
default (app/utils/llm_provider.py), so no adaptation needed here; Gemini
remains available as the config-swappable cloud alternative, same as
gap_analysis and the tutor.

Same resilience contract as gap_analysis/llm.py: any failure returns
(None, None) and the caller keeps the plan's existing per-block "why"
text, which was always there and never depended on this enrichment step
-- a missing/slow LLM must never block plan generation. There's no third,
deterministic-text tier here the way gap_analysis has _fallback_summary,
because the per-block "why" text already covers that role.

One call per plan (not per day), same O(plans)-not-O(days) reasoning as
gap_analysis's O(exams)-not-O(questions) batching.

Amharic: same finding as gap_analysis and the tutor -- the local models
tested against this deployment produce garbled Amharic, verified
directly, and Amharic isn't mandatory at this stage (confirmed). Prompt
varies by provider.supports_amharic, same mechanism as the other two
modules.
"""

from __future__ import annotations

import json
from typing import Optional

from app.utils.llm_provider import LLMProvider, generate_with_fallback


def _build_prompt(days: list[dict], exam_days_away: Optional[int], provider: LLMProvider) -> str:
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
        "Write each goal in simple English, then the same sentence in Amharic."
        if provider.supports_amharic
        else "Write each goal in simple English only."
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


async def enrich_days(
    days: list[dict], exam_days_away: Optional[int]
) -> tuple[Optional[dict[int, str]], Optional[str]]:
    """Returns ({day_number: goal_text}, model_used), or (None, None) if
    both providers failed -- the caller's per-block "why" text (already
    present regardless) is the only fallback needed."""
    if not days:
        return None, None

    def build(provider: LLMProvider) -> str:
        return _build_prompt(days, exam_days_away, provider)

    text, provider_used = await generate_with_fallback(build, json_mode=True)
    if text is None or provider_used is None:
        return None, None

    try:
        goals = _parse_response(text)
    except Exception:  # noqa: BLE001 -- a malformed response must never block plan generation
        return None, None
    if not goals:
        return None, None
    return goals, provider_used.model_name
