"""Tests for app/services/gap_analysis/llm.py -- Capability 3A's Pass 3
LLM synthesis. Covers _parse_response's real-world-malformed-JSON
handling directly, and synthesize_insights's provider-failure/malformed-
response resilience (an LLM hiccup must never crash gap analysis, see
the module's own docstring) via a monkeypatched generate_with_fallback
rather than a real LLM call.
"""

from __future__ import annotations

import json

import pytest

from app.services.gap_analysis import llm as gap_llm
from app.utils.llm_provider import LLMProvider


class _FakeProvider(LLMProvider):
    model_name = "fake-model"
    supports_amharic = False

    async def generate(self, prompt: str, *, json_mode: bool = False) -> str | None:
        raise NotImplementedError("not called directly in these tests")


def test_parse_response_extracts_explanations_and_summary() -> None:
    text = '{"gaps": [{"index": 1, "explanation": "You missed a step."}], "examSummary": "Overall good, watch fractions."}'

    explanations, summary = gap_llm._parse_response(text)

    assert explanations == {1: "You missed a step."}
    assert summary == "Overall good, watch fractions."


def test_parse_response_skips_gaps_with_missing_or_invalid_index() -> None:
    # A real LLM response occasionally drops/mistypes a field -- one bad
    # entry must not lose the others.
    text = """{"gaps": [
        {"index": 1, "explanation": "Good gap."},
        {"explanation": "Missing index entirely."},
        {"index": "not-a-number", "explanation": "Bad index type."},
        {"index": 2, "explanation": "   "}
    ], "examSummary": "Summary text."}"""

    explanations, summary = gap_llm._parse_response(text)

    assert explanations == {1: "Good gap."}
    assert summary == "Summary text."


def test_parse_response_returns_none_summary_when_blank_or_missing() -> None:
    explanations, summary = gap_llm._parse_response('{"gaps": [{"index": 1, "explanation": "ok"}]}')
    assert summary is None
    assert explanations == {1: "ok"}

    explanations, summary = gap_llm._parse_response('{"gaps": [{"index": 1, "explanation": "ok"}], "examSummary": "   "}')
    assert summary is None


def test_parse_response_returns_none_explanations_when_gaps_empty() -> None:
    explanations, summary = gap_llm._parse_response('{"gaps": [], "examSummary": "Just a summary."}')
    assert explanations is None
    assert summary == "Just a summary."


def test_parse_response_raises_on_invalid_json() -> None:
    # synthesize_insights is the one that must swallow this (see its own
    # try/except) -- _parse_response itself is expected to raise, that's
    # the contract the caller's try/except relies on.
    with pytest.raises(json.JSONDecodeError):
        gap_llm._parse_response("not json at all")


@pytest.mark.asyncio
async def test_synthesize_insights_returns_none_triple_with_no_gaps() -> None:
    explanations, summary, model = await gap_llm.synthesize_insights(
        exam_title="Test Exam", exam_scope="unit_test", subject_code="BIO", grade_level=9,
        percentage=80.0, gap_contexts=[],
    )
    assert (explanations, summary, model) == (None, None, None)


@pytest.mark.asyncio
async def test_synthesize_insights_returns_none_triple_when_all_providers_fail(monkeypatch: pytest.MonkeyPatch) -> None:
    async def fake_generate_with_fallback(build_prompt, *, json_mode=False):
        return None, None

    monkeypatch.setattr(gap_llm, "generate_with_fallback", fake_generate_with_fallback)

    explanations, summary, model = await gap_llm.synthesize_insights(
        exam_title="Test Exam", exam_scope="unit_test", subject_code="BIO", grade_level=9,
        percentage=40.0, gap_contexts=[{"index": 1, "question_text": "Q?", "symptom_topic": "Cells", "severity": 0.5}],
    )
    assert (explanations, summary, model) == (None, None, None)


@pytest.mark.asyncio
async def test_synthesize_insights_survives_malformed_llm_response(monkeypatch: pytest.MonkeyPatch) -> None:
    # A real, if rare, failure mode: the provider responds successfully
    # but the text isn't valid JSON (a chatty preamble, truncated output,
    # etc.) -- this must degrade gracefully, not crash the pipeline. See
    # synthesize_insights's own try/except around _parse_response.
    provider = _FakeProvider()

    async def fake_generate_with_fallback(build_prompt, *, json_mode=False):
        return "Sure, here's the analysis: <not actually json>", provider

    monkeypatch.setattr(gap_llm, "generate_with_fallback", fake_generate_with_fallback)

    explanations, summary, model = await gap_llm.synthesize_insights(
        exam_title="Test Exam", exam_scope="unit_test", subject_code="BIO", grade_level=9,
        percentage=40.0, gap_contexts=[{"index": 1, "question_text": "Q?", "symptom_topic": "Cells", "severity": 0.5}],
    )
    assert (explanations, summary, model) == (None, None, None)


@pytest.mark.asyncio
async def test_synthesize_insights_returns_parsed_result_on_success(monkeypatch: pytest.MonkeyPatch) -> None:
    provider = _FakeProvider()

    async def fake_generate_with_fallback(build_prompt, *, json_mode=False):
        return '{"gaps": [{"index": 1, "explanation": "You mixed up mitosis and meiosis."}], "examSummary": "Solid overall, review cell division."}', provider

    monkeypatch.setattr(gap_llm, "generate_with_fallback", fake_generate_with_fallback)

    explanations, summary, model = await gap_llm.synthesize_insights(
        exam_title="Test Exam", exam_scope="unit_test", subject_code="BIO", grade_level=9,
        percentage=60.0, gap_contexts=[{"index": 1, "question_text": "Q?", "symptom_topic": "Cell Division", "severity": 0.4}],
    )
    assert explanations == {1: "You mixed up mitosis and meiosis."}
    assert summary == "Solid overall, review cell division."
    assert model == "fake-model"
