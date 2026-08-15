"""National curriculum insights generator.

Accepts the ministry overview stats and returns a structured analysis with
an AI-generated headline, alerts, trend summary, and recommendations.
Falls back gracefully when no LLM is configured.
"""
from __future__ import annotations

import json
import logging

from app.utils.llm_provider import generate_with_fallback

logger = logging.getLogger(__name__)

_FALLBACK = {
    "headline": "National curriculum data received — configure an LLM provider for AI-generated insights.",
    "trend_summary": "",
    "alerts": [],
    "recommendations": [],
}

_PROMPT_TEMPLATE = """You are an expert educational analyst for the Ethiopian K-12 national curriculum system.

You have been given the following national statistics:
{stats_json}

Produce a concise, actionable national curriculum insight report in JSON format with exactly these fields:
{{
  "headline": "<one-sentence summary of the most important national finding>",
  "trend_summary": "<2-3 sentences describing overall trends, strengths, and systemic risks>",
  "alerts": [
    {{"severity": "critical|warning|info", "message": "<specific, actionable alert text>"}}
  ],
  "recommendations": [
    {{"priority": "high|medium|low", "text": "<specific recommendation for ministry officials>"}}
  ]
}}

Guidelines:
- Derive insights from the actual numbers provided — do not invent data not present.
- Alerts should flag ratios or patterns that indicate systemic risk (e.g. student/teacher ratio, school coverage gaps).
- Recommendations should be concrete policy actions a ministry official can act on.
- Return ONLY the JSON object, no markdown fences, no preamble.
"""


async def generate_national_insights(overview: dict) -> dict:
    stats_json = json.dumps(overview, indent=2)
    prompt = _PROMPT_TEMPLATE.format(stats_json=stats_json)

    def build_prompt(_unused):
        return prompt

    text, provider = await generate_with_fallback(build_prompt, json_mode=True)
    if text is None:
        logger.warning("insights.generate: no LLM provider available, returning fallback")
        return _FALLBACK

    try:
        parsed = json.loads(text)
    except json.JSONDecodeError:
        logger.warning("insights.generate: LLM returned non-JSON, returning fallback", extra={"raw": text[:200]})
        return _FALLBACK

    # Normalise to expected shape, tolerate partial responses.
    alerts = []
    for a in parsed.get("alerts") or []:
        if isinstance(a, dict) and a.get("message"):
            alerts.append({"severity": a.get("severity", "info"), "message": a["message"]})

    recs = []
    for rec in parsed.get("recommendations") or []:
        if isinstance(rec, dict) and rec.get("text"):
            recs.append({"priority": rec.get("priority", "medium"), "text": rec["text"]})

    return {
        "headline": parsed.get("headline") or _FALLBACK["headline"],
        "trend_summary": parsed.get("trend_summary") or "",
        "alerts": alerts,
        "recommendations": recs,
    }
