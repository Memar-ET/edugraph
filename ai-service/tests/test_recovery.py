"""Tests for app/services/gap_analysis/recovery.py -- checklist section
13 (Blocked-Learning & Recovery System): "unit-test recovery-mode
triggering and exit conditions."
"""

from __future__ import annotations

import pytest

from app.services.gap_analysis import recovery


def test_not_blocked_with_too_few_observations() -> None:
    assert recovery.is_blocked([False, False]) is False  # only 2, need 3


def test_blocked_after_enough_consecutive_failures() -> None:
    assert recovery.is_blocked([False, False, False]) is True


def test_not_blocked_if_most_recent_response_was_correct() -> None:
    # Newest-first: a correct response breaks the streak even if two
    # failures preceded it.
    assert recovery.is_blocked([True, False, False]) is False


def test_not_blocked_if_any_of_the_recent_window_was_correct() -> None:
    assert recovery.is_blocked([False, True, False]) is False


def test_blocked_uses_only_the_configured_window_size() -> None:
    # 3 consecutive failures followed by older successes still counts --
    # only the most recent CONSECUTIVE_FAILURES_FOR_BLOCK matter.
    long_history = [False, False, False, True, True, True]
    assert recovery.is_blocked(long_history) is True


def test_rank_routes_excludes_already_tried_topics() -> None:
    candidates = [
        {"route_topic_id": "a", "edge_type": "similar_to", "weight": 1.0, "confidence": 0.9},
        {"route_topic_id": "b", "edge_type": "similar_to", "weight": 1.0, "confidence": 0.9},
    ]
    ranked = recovery.rank_routes(candidates, tried={"a"})
    assert [c["route_topic_id"] for c in ranked] == ["b"]


def test_rank_routes_prefers_similar_to_over_related_to() -> None:
    candidates = [
        {"route_topic_id": "loose", "edge_type": "related_to", "weight": 1.0, "confidence": 0.99},
        {"route_topic_id": "close", "edge_type": "similar_to", "weight": 1.0, "confidence": 0.1},
    ]
    ranked = recovery.rank_routes(candidates, tried=set())
    # similar_to wins on edge-type priority even with lower confidence
    # than the related_to candidate.
    assert ranked[0]["route_topic_id"] == "close"


def test_rank_routes_prefers_alternative_to_over_lower_granularity() -> None:
    candidates = [
        {"route_topic_id": "subtopic", "edge_type": "lower_granularity", "weight": 1.0, "confidence": None},
        {"route_topic_id": "alt", "edge_type": "alternative_to", "weight": 1.0, "confidence": None},
    ]
    ranked = recovery.rank_routes(candidates, tried=set())
    assert ranked[0]["route_topic_id"] == "alt"


def test_rank_routes_breaks_ties_within_a_type_by_confidence() -> None:
    candidates = [
        {"route_topic_id": "weak", "edge_type": "similar_to", "weight": 1.0, "confidence": 0.3},
        {"route_topic_id": "strong", "edge_type": "similar_to", "weight": 1.0, "confidence": 0.9},
    ]
    ranked = recovery.rank_routes(candidates, tried=set())
    assert ranked[0]["route_topic_id"] == "strong"


def test_rank_routes_returns_empty_when_everything_has_been_tried() -> None:
    candidates = [{"route_topic_id": "a", "edge_type": "similar_to", "weight": 1.0, "confidence": 0.9}]
    assert recovery.rank_routes(candidates, tried={"a"}) == []


@pytest.mark.asyncio
async def test_check_and_trigger_does_nothing_when_already_in_recovery(monkeypatch: pytest.MonkeyPatch) -> None:
    async def fake_active(student_id, topic_id):
        return {"id": "existing-attempt"}

    monkeypatch.setattr(recovery.db, "fetch_active_recovery_for_topic", fake_active)

    result = await recovery.check_and_trigger("student-1", "school-1", "topic-1")
    assert result is None


@pytest.mark.asyncio
async def test_check_and_trigger_does_nothing_when_not_blocked(monkeypatch: pytest.MonkeyPatch) -> None:
    async def fake_active(student_id, topic_id):
        return None

    async def fake_recent(student_id, topic_id, limit):
        return [True, False, False]  # not 3 consecutive failures

    monkeypatch.setattr(recovery.db, "fetch_active_recovery_for_topic", fake_active)
    monkeypatch.setattr(recovery.db, "fetch_recent_correctness", fake_recent)

    result = await recovery.check_and_trigger("student-1", "school-1", "topic-1")
    assert result is None


@pytest.mark.asyncio
async def test_check_and_trigger_creates_an_attempt_when_blocked_with_a_route_available(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    async def fake_active(student_id, topic_id):
        return None

    async def fake_recent(student_id, topic_id, limit):
        return [False, False, False]

    async def fake_tried(student_id, topic_id):
        return []

    async def fake_candidates(topic_id):
        return [{"route_topic_id": "route-a", "edge_type": "similar_to", "weight": 1.0, "confidence": 0.8}]

    inserted = {}

    async def fake_insert(student_id, school_id, blocked_topic_id, route_topic_id, route_edge_type, trigger_reason):
        inserted.update(locals())
        return "new-attempt-id"

    monkeypatch.setattr(recovery.db, "fetch_active_recovery_for_topic", fake_active)
    monkeypatch.setattr(recovery.db, "fetch_recent_correctness", fake_recent)
    monkeypatch.setattr(recovery.db, "fetch_tried_recovery_routes", fake_tried)
    monkeypatch.setattr(recovery.db, "fetch_recovery_route_candidates", fake_candidates)
    monkeypatch.setattr(recovery.db, "insert_recovery_attempt", fake_insert)

    result = await recovery.check_and_trigger("student-1", "school-1", "topic-1")

    assert result is not None
    assert result["attempt_id"] == "new-attempt-id"
    assert result["route_topic_id"] == "route-a"
    assert inserted["route_topic_id"] == "route-a"
    assert inserted["route_edge_type"] == "similar_to"
