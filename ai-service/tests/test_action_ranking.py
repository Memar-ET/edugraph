"""Tests for app/services/study_plan/action_ranking.py -- checklist
section 14: "unit-test next-best-action ranking," score difficulty fit,
and distinct action-type classification.
"""

from __future__ import annotations

import pytest

from app.services.study_plan import action_ranking


def test_difficulty_fit_is_neutral_with_no_calibrated_item() -> None:
    assert action_ranking._difficulty_fit(0.5, None) == 0.5


def test_difficulty_fit_is_neutral_with_no_mastery_estimate() -> None:
    assert action_ranking._difficulty_fit(None, (1.0, 0.0)) == 0.5


def test_difficulty_fit_peaks_when_predicted_success_hits_target() -> None:
    # At theta=b (mastery=0.5 under the sigmoid squash), a=1, b=0 -- the
    # 2PL predicted success is exactly 0.5, which is BELOW the 0.6 target,
    # so fit should be less than 1.0 but still reasonably high.
    fit_at_target = action_ranking._difficulty_fit(0.5, (0.0, 1.0))
    fit_far_off = action_ranking._difficulty_fit(0.99, (0.0, 1.0))  # predicted success near 1.0, far from 0.6 target
    assert fit_at_target > fit_far_off


def test_difficulty_fit_stays_in_unit_interval() -> None:
    for mastery in (0.01, 0.3, 0.5, 0.7, 0.99):
        for item in (None, (0.0, 1.0), (-2.0, 2.0), (2.0, 0.5)):
            fit = action_ranking._difficulty_fit(mastery, item)
            assert 0.0 <= fit <= 1.0


@pytest.mark.asyncio
async def test_classify_action_types_escalates_after_repeated_failed_routes(monkeypatch: pytest.MonkeyPatch) -> None:
    async def fake_states(student_id, topic_ids):
        return {}

    async def fake_blocked(student_id, topic_ids):
        return set()

    async def fake_failed_routes(student_id, topic_ids):
        return {"topic-1": action_ranking.TEACHER_ESCALATION_FAILED_ROUTES}

    monkeypatch.setattr(action_ranking, "fetch_skill_states_bulk", fake_states)
    monkeypatch.setattr(action_ranking, "fetch_blocked_topics", fake_blocked)
    monkeypatch.setattr(action_ranking.db, "fetch_failed_recovery_route_counts", fake_failed_routes)

    result = await action_ranking.classify_action_types("student-1", ["topic-1"])
    assert result["topic-1"] == "teacher_escalation"


@pytest.mark.asyncio
async def test_classify_action_types_routes_blocked_topics_to_alternative_representation(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    async def fake_states(student_id, topic_ids):
        return {}

    async def fake_blocked(student_id, topic_ids):
        return {"topic-1"}

    async def fake_failed_routes(student_id, topic_ids):
        return {}

    monkeypatch.setattr(action_ranking, "fetch_skill_states_bulk", fake_states)
    monkeypatch.setattr(action_ranking, "fetch_blocked_topics", fake_blocked)
    monkeypatch.setattr(action_ranking.db, "fetch_failed_recovery_route_counts", fake_failed_routes)

    result = await action_ranking.classify_action_types("student-1", ["topic-1"])
    assert result["topic-1"] == "alternative_representation"


@pytest.mark.asyncio
async def test_classify_action_types_cold_start_gets_concept_explanation(monkeypatch: pytest.MonkeyPatch) -> None:
    async def fake_states(student_id, topic_ids):
        return {"topic-1": {"evidence_count": 0, "mastery_probability": None, "uncertainty": None, "forgetting_risk": None}}

    async def fake_blocked(student_id, topic_ids):
        return set()

    async def fake_failed_routes(student_id, topic_ids):
        return {}

    monkeypatch.setattr(action_ranking, "fetch_skill_states_bulk", fake_states)
    monkeypatch.setattr(action_ranking, "fetch_blocked_topics", fake_blocked)
    monkeypatch.setattr(action_ranking.db, "fetch_failed_recovery_route_counts", fake_failed_routes)

    result = await action_ranking.classify_action_types("student-1", ["topic-1"])
    assert result["topic-1"] == "concept_explanation"


@pytest.mark.asyncio
async def test_classify_action_types_root_cause_gets_prerequisite_review(monkeypatch: pytest.MonkeyPatch) -> None:
    async def fake_states(student_id, topic_ids):
        return {"topic-1": {"evidence_count": 5, "mastery_probability": 0.3, "uncertainty": 0.2, "forgetting_risk": 0.1}}

    async def fake_blocked(student_id, topic_ids):
        return set()

    async def fake_failed_routes(student_id, topic_ids):
        return {}

    monkeypatch.setattr(action_ranking, "fetch_skill_states_bulk", fake_states)
    monkeypatch.setattr(action_ranking, "fetch_blocked_topics", fake_blocked)
    monkeypatch.setattr(action_ranking.db, "fetch_failed_recovery_route_counts", fake_failed_routes)

    result = await action_ranking.classify_action_types("student-1", ["topic-1"], is_root_cause={"topic-1": True})
    assert result["topic-1"] == "prerequisite_review"


@pytest.mark.asyncio
async def test_classify_action_types_defaults_to_practice(monkeypatch: pytest.MonkeyPatch) -> None:
    async def fake_states(student_id, topic_ids):
        return {"topic-1": {"evidence_count": 5, "mastery_probability": 0.3, "uncertainty": 0.1, "forgetting_risk": 0.1}}

    async def fake_blocked(student_id, topic_ids):
        return set()

    async def fake_failed_routes(student_id, topic_ids):
        return {}

    monkeypatch.setattr(action_ranking, "fetch_skill_states_bulk", fake_states)
    monkeypatch.setattr(action_ranking, "fetch_blocked_topics", fake_blocked)
    monkeypatch.setattr(action_ranking.db, "fetch_failed_recovery_route_counts", fake_failed_routes)

    result = await action_ranking.classify_action_types("student-1", ["topic-1"])
    assert result["topic-1"] == "practice"
