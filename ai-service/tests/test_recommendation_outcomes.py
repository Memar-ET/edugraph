"""Tests for app/workers/refit_worker.py's recommendation-outcome
evaluation -- checklist sections 2/14/20: "measure learning gain" /
"capture action outcome for the feedback loop"."""

from __future__ import annotations

import pytest

from app.workers import refit_worker


def test_no_new_evidence_is_insufficient_evidence() -> None:
    assert refit_worker._classify_outcome(0.3, 2, 0.3, 2) == "insufficient_evidence"


def test_no_current_estimate_is_insufficient_evidence() -> None:
    # evidence_now increased but fusion hasn't produced an estimate yet --
    # shouldn't normally happen once evidence_count > 0, but must not be
    # misclassified as a real outcome.
    assert refit_worker._classify_outcome(0.3, 2, None, 3) == "insufficient_evidence"


def test_cold_start_baseline_above_threshold_is_improved() -> None:
    assert refit_worker._classify_outcome(None, 0, 0.6, 1) == "improved"


def test_cold_start_baseline_below_threshold_is_unchanged() -> None:
    assert refit_worker._classify_outcome(None, 0, 0.4, 1) == "unchanged"


def test_mastery_increase_above_delta_is_improved() -> None:
    assert refit_worker._classify_outcome(0.3, 2, 0.4, 4) == "improved"


def test_mastery_decrease_below_delta_is_worsened() -> None:
    assert refit_worker._classify_outcome(0.5, 2, 0.4, 4) == "worsened"


def test_small_movement_within_delta_is_unchanged() -> None:
    assert refit_worker._classify_outcome(0.5, 2, 0.52, 4) == "unchanged"


def test_movement_at_delta_boundary_counts() -> None:
    delta = refit_worker.OUTCOME_IMPROVEMENT_DELTA
    assert refit_worker._classify_outcome(0.5, 2, 0.5 + delta, 4) == "improved"
    assert refit_worker._classify_outcome(0.5 + delta, 2, 0.5, 4) == "worsened"


@pytest.mark.asyncio
async def test_evaluate_recommendation_outcomes_applies_classified_updates(monkeypatch: pytest.MonkeyPatch) -> None:
    pending = [
        {"id": "rec-1", "mastery_before": 0.3, "evidence_before": 2, "mastery_now": 0.5, "evidence_now": 4},
        {"id": "rec-2", "mastery_before": 0.3, "evidence_before": 2, "mastery_now": 0.3, "evidence_now": 2},
    ]
    applied: list = []

    async def fake_fetch(older_than_days: int) -> list[dict]:
        assert older_than_days == refit_worker.OUTCOME_EVALUATION_DELAY_DAYS
        return pending

    async def fake_apply(updates: list) -> None:
        applied.extend(updates)

    monkeypatch.setattr(refit_worker.studyplan_db, "fetch_pending_recommendation_outcomes", fake_fetch)
    monkeypatch.setattr(refit_worker.studyplan_db, "apply_recommendation_outcomes", fake_apply)

    count = await refit_worker.evaluate_recommendation_outcomes()

    assert count == 2
    assert ("rec-1", "improved", 0.5) in applied
    assert ("rec-2", "insufficient_evidence", 0.3) in applied


@pytest.mark.asyncio
async def test_evaluate_recommendation_outcomes_no_pending_rows(monkeypatch: pytest.MonkeyPatch) -> None:
    async def fake_fetch(older_than_days: int) -> list[dict]:
        return []

    monkeypatch.setattr(refit_worker.studyplan_db, "fetch_pending_recommendation_outcomes", fake_fetch)
    count = await refit_worker.evaluate_recommendation_outcomes()
    assert count == 0
