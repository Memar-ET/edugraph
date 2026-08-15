"""Tests for app/services/gap_analysis/misconception.py -- checklist
section 12: "prevent a misconception from being treated as confirmed
without sufficient evidence" / "track evidence supporting or weakening a
misconception hypothesis."
"""

from __future__ import annotations

import pytest

from app.services.gap_analysis import misconception


def test_parse_response_requires_has_misconception_true() -> None:
    assert misconception._parse_response('{"hasMisconception": false}') is None


def test_parse_response_requires_nonempty_misconception_text() -> None:
    assert misconception._parse_response('{"hasMisconception": true, "misconceptionText": ""}') is None


def test_parse_response_extracts_fields() -> None:
    text = (
        '{"hasMisconception": true, "misconceptionText": "confuses force with acceleration", '
        '"triggerPattern": "correct arithmetic, wrong concept", "confidence": 0.7, '
        '"interventionText": "review Newton\'s second law"}'
    )
    result = misconception._parse_response(text)
    assert result is not None
    assert result["misconception_text"] == "confuses force with acceleration"
    assert result["confidence"] == 0.7
    assert result["intervention_text"] == "review Newton's second law"


def test_parse_response_clamps_out_of_range_confidence() -> None:
    text = '{"hasMisconception": true, "misconceptionText": "x", "confidence": 5.0}'
    result = misconception._parse_response(text)
    assert result["confidence"] == 1.0


def test_parse_response_defaults_confidence_when_missing_or_invalid() -> None:
    text = '{"hasMisconception": true, "misconceptionText": "x", "confidence": "not a number"}'
    result = misconception._parse_response(text)
    assert result["confidence"] == 0.5


@pytest.mark.asyncio
async def test_check_verification_responses_weakens_on_correct_answer(monkeypatch: pytest.MonkeyPatch) -> None:
    async def fake_active(student_id):
        return [{"id": "hyp-1", "verification_item_id": "q-1", "confidence": 0.7}]

    adjustments = []

    async def fake_adjust(hypothesis_id, delta, note):
        adjustments.append((hypothesis_id, delta, note))

    monkeypatch.setattr(misconception.db, "fetch_active_verification_items", fake_active)
    monkeypatch.setattr(misconception.db, "adjust_misconception_confidence", fake_adjust)

    resolved = await misconception.check_verification_responses("student-1", [{"question_id": "q-1", "correct": True}])

    assert resolved == 1
    assert adjustments[0][0] == "hyp-1"
    assert adjustments[0][1] == misconception.VERIFICATION_CORRECT_DELTA
    assert adjustments[0][1] < 0  # correct answer must weaken, not strengthen


@pytest.mark.asyncio
async def test_check_verification_responses_strengthens_on_incorrect_answer(monkeypatch: pytest.MonkeyPatch) -> None:
    async def fake_active(student_id):
        return [{"id": "hyp-1", "verification_item_id": "q-1", "confidence": 0.5}]

    adjustments = []

    async def fake_adjust(hypothesis_id, delta, note):
        adjustments.append((hypothesis_id, delta, note))

    monkeypatch.setattr(misconception.db, "fetch_active_verification_items", fake_active)
    monkeypatch.setattr(misconception.db, "adjust_misconception_confidence", fake_adjust)

    resolved = await misconception.check_verification_responses("student-1", [{"question_id": "q-1", "correct": False}])

    assert resolved == 1
    assert adjustments[0][1] == misconception.VERIFICATION_INCORRECT_DELTA
    assert adjustments[0][1] > 0  # incorrect answer must strengthen


@pytest.mark.asyncio
async def test_check_verification_responses_ignores_unrelated_answers(monkeypatch: pytest.MonkeyPatch) -> None:
    async def fake_active(student_id):
        return [{"id": "hyp-1", "verification_item_id": "q-1", "confidence": 0.5}]

    async def fake_adjust(hypothesis_id, delta, note):
        pytest.fail("must not adjust a hypothesis whose verification item wasn't answered")

    monkeypatch.setattr(misconception.db, "fetch_active_verification_items", fake_active)
    monkeypatch.setattr(misconception.db, "adjust_misconception_confidence", fake_adjust)

    resolved = await misconception.check_verification_responses("student-1", [{"question_id": "q-2", "correct": True}])
    assert resolved == 0


@pytest.mark.asyncio
async def test_check_verification_responses_returns_zero_with_no_active_verifications(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    async def fake_active(student_id):
        return []

    monkeypatch.setattr(misconception.db, "fetch_active_verification_items", fake_active)
    resolved = await misconception.check_verification_responses("student-1", [{"question_id": "q-1", "correct": True}])
    assert resolved == 0
