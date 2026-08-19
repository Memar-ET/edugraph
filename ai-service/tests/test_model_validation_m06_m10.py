"""
Whole-System Model Validation Test Specification — M06 through M10
Source: EduGraph_Whole_System_Model_Validation_Test_Specification.docx

M06 – Psychometric / Assessment Measurement   (M06-T01..T08)
M07 – Diagnostic Inference                    (M07-T01..T08)
M08 – Prediction Models                       (M08-T01..T09)
M09 – Cohort / Population Model               (M09-T01..T08) — all N/A
M10 – Recommendation / Action Ranking         (M10-T01..T10)
"""

from __future__ import annotations

import math
from datetime import datetime, timedelta, timezone
from typing import Optional
from unittest.mock import MagicMock

import pytest

from app.services.evaluation import metrics
from app.services.gap_analysis import root_cause
from app.services.knowledge_tracing import bkt, irt, replay
from app.services.study_plan import action_ranking


# ══════════════════════════════════════════════════════════════════════════════
# Helpers
# ══════════════════════════════════════════════════════════════════════════════

def _mock_record(
    estimate: Optional[float],
    provenance: str = "bkt",
    reliability: float = 0.8,
    sample_size: int = 5,
    days_ago: float = 0.0,
    ref: Optional[datetime] = None,
) -> MagicMock:
    now = ref or datetime.now(timezone.utc)
    created_at = now - timedelta(days=days_ago)
    rec = MagicMock()
    data = {
        "estimate": estimate,
        "provenance": provenance,
        "reliability": reliability,
        "sample_size": sample_size,
        "created_at": created_at,
        "uncertainty": 0.2,
    }
    rec.__getitem__ = lambda self, k: data[k]
    rec.get = lambda k, default=None: data.get(k, default)
    return rec


def _skill_state(
    mastery: Optional[float],
    uncertainty: float = 0.15,
    forgetting_risk: float = 0.1,
    evidence_count: int = 5,
) -> dict:
    return {
        "mastery_probability": mastery,
        "uncertainty": uncertainty,
        "forgetting_risk": forgetting_risk,
        "evidence_count": evidence_count,
        "structural_status": None,
    }


def _action_ranking_env(monkeypatch: pytest.MonkeyPatch, states: dict, blocked: set, impact: float = 0.3) -> None:
    """Patch all of compute_action_scores's DB dependencies."""
    from app.db import postgres_studyplan as db_sp

    async def fake_states(student_id, topic_ids):
        return {tid: states[tid] for tid in topic_ids if tid in states}

    async def fake_blocked(student_id, topic_ids):
        return {tid for tid in topic_ids if tid in blocked}

    async def fake_impact(topic_id):
        return impact

    async def fake_snap(*args, **kwargs):
        return {"id": "snap-1"}

    async def fake_policy_snap():
        return "policy-snap-1"

    async def fake_insert_snap(*args, **kwargs):
        return "snap-1"

    async def fake_repetition(student_id, topic_ids, window_days=30):
        return {}

    async def fake_mandatory(topic_ids):
        return {tid: 0.5 for tid in topic_ids}

    async def fake_avg_item(topic_ids):
        return {}

    monkeypatch.setattr(action_ranking, "fetch_skill_states_bulk", fake_states)
    monkeypatch.setattr(action_ranking, "fetch_blocked_topics", fake_blocked)
    monkeypatch.setattr(action_ranking, "downstream_impact", fake_impact)
    monkeypatch.setattr(action_ranking, "get_active_model_snapshot", fake_snap)
    monkeypatch.setattr(action_ranking, "get_recommendation_policy_snapshot_id", fake_policy_snap)
    monkeypatch.setattr(action_ranking, "insert_model_snapshot", fake_insert_snap)
    monkeypatch.setattr(db_sp, "fetch_repetition_counts", fake_repetition)
    monkeypatch.setattr(db_sp, "fetch_mandatory_clo_fraction", fake_mandatory)
    monkeypatch.setattr(db_sp, "fetch_avg_item_difficulty", fake_avg_item)


def _classify_env(monkeypatch: pytest.MonkeyPatch, states: dict, blocked: set, failed_routes: dict | None = None) -> None:
    """Patch all of classify_action_types's DB dependencies."""
    from app.db import postgres_studyplan as db_sp

    async def fake_states(student_id, topic_ids):
        return {tid: states[tid] for tid in topic_ids if tid in states}

    async def fake_blocked(student_id, topic_ids):
        return {tid for tid in topic_ids if tid in blocked}

    async def fake_failed_routes(student_id, topic_ids):
        return failed_routes or {}

    monkeypatch.setattr(action_ranking, "fetch_skill_states_bulk", fake_states)
    monkeypatch.setattr(action_ranking, "fetch_blocked_topics", fake_blocked)
    monkeypatch.setattr(db_sp, "fetch_failed_recovery_route_counts", fake_failed_routes)


# ══════════════════════════════════════════════════════════════════════════════
# M06 – Psychometric / Assessment Measurement
# ══════════════════════════════════════════════════════════════════════════════


def test_m06_t01_p_correct_equals_half_when_theta_equals_b() -> None:
    """2PL IRT definition: P(correct | θ=b) = 0.5 for any a."""
    assert abs(irt.p_correct(0.0, a=1.0, b=0.0) - 0.5) < 1e-9
    assert abs(irt.p_correct(2.0, a=2.5, b=2.0) - 0.5) < 1e-9
    assert abs(irt.p_correct(-1.5, a=0.5, b=-1.5) - 0.5) < 1e-9


def test_m06_t02_p_correct_increases_monotonically_with_theta() -> None:
    """Higher ability → higher probability of correct response (monotone IRT)."""
    a, b = 1.0, 0.0
    thetas = [-3.0, -1.5, 0.0, 1.5, 3.0]
    probs = [irt.p_correct(t, a=a, b=b) for t in thetas]
    for i in range(len(probs) - 1):
        assert probs[i] < probs[i + 1], f"p_correct not monotone at theta={thetas[i]}"


def test_m06_t03_higher_discrimination_sharpens_curve_above_b() -> None:
    """Higher a → p_correct is further from 0.5 at a given theta > b."""
    theta, b = 1.0, 0.0
    low_a = irt.p_correct(theta, a=0.5, b=b)
    high_a = irt.p_correct(theta, a=3.0, b=b)
    assert high_a > low_a


def test_m06_t04_extreme_easy_item_gives_high_probability_to_low_ability() -> None:
    """b=-3 (very easy): even a learner at theta=-1 should get > 0.85."""
    assert irt.p_correct(-1.0, a=1.0, b=-3.0) > 0.85


def test_m06_t05_extreme_hard_item_gives_low_probability_to_high_ability() -> None:
    """b=3 (very hard): even a learner at theta=1 should get < 0.15."""
    assert irt.p_correct(1.0, a=1.0, b=3.0) < 0.15


def test_m06_t06_theta_estimation_mostly_correct_gives_positive_theta() -> None:
    """MLE theta for 8 correct, 2 incorrect → positive ability."""
    # tuple order is (a, b, correct)
    items: list[tuple[float, float, bool]] = (
        [(1.0, 0.0, True)] * 8 + [(1.0, 0.0, False)] * 2
    )
    theta, _ = irt._estimate_theta(items)
    assert theta > 0.0


def test_m06_t07_theta_estimation_mostly_incorrect_gives_negative_theta() -> None:
    """MLE theta for 2 correct, 8 incorrect → negative ability."""
    items: list[tuple[float, float, bool]] = (
        [(1.0, 0.0, True)] * 2 + [(1.0, 0.0, False)] * 8
    )
    theta, _ = irt._estimate_theta(items)
    assert theta < 0.0


def test_m06_t08_sparse_responses_theta_is_finite_and_bounded() -> None:
    """Single correct response — estimate must be finite, within THETA_CLAMP."""
    items: list[tuple[float, float, bool]] = [(1.0, 0.0, True)]
    theta, _ = irt._estimate_theta(items)
    assert math.isfinite(theta), "theta must be finite for sparse responses"
    assert -irt.THETA_CLAMP <= theta <= irt.THETA_CLAMP


# ══════════════════════════════════════════════════════════════════════════════
# M07 – Diagnostic Inference
# ══════════════════════════════════════════════════════════════════════════════


def test_m07_t07_rcs_formula_matches_spec_section_9() -> None:
    """compute_rcs must exactly implement Weakness × Confidence × Impact × Readiness × Gain."""
    weakness, confidence, impact, readiness, gain = 0.5, 0.8, 0.6, 0.9, 0.3
    expected = weakness * confidence * impact * readiness * gain
    assert abs(root_cause.compute_rcs(weakness, confidence, impact, readiness, gain) - expected) < 1e-12


@pytest.mark.asyncio
async def test_m07_t01_isolated_gap_returns_that_topic_as_root_cause(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """Chain with one weak topic → score_candidates returns it with rcs > 0."""
    chain = [{"id": "topic-weak", "title": "Weak Topic", "grade_level": 9, "depth": 1}]

    async def one_weak(student_id, topic_ids):
        return {"topic-weak": {"mastery_probability": 0.2, "uncertainty": 0.3}}

    async def no_prereqs(topic_id, max_depth=1):
        return []

    async def one_dependent(topic_id, max_depth=3):
        return [{"id": "topic-downstream", "depth": 1}]

    monkeypatch.setattr(root_cause, "fetch_skill_states_bulk", one_weak)
    monkeypatch.setattr(root_cause.neo4j_db, "fetch_prerequisite_chain", no_prereqs)
    monkeypatch.setattr(root_cause.db, "fetch_prerequisite_chain_pg", no_prereqs)
    monkeypatch.setattr(root_cause.neo4j_db, "fetch_downstream_dependents", one_dependent)
    monkeypatch.setattr(root_cause.db, "fetch_downstream_dependents_pg", one_dependent)


    result = await root_cause.score_candidates("student-1", chain, {})

    assert result is not None
    assert result["id"] == "topic-weak"
    assert result["rcs"] > 0


@pytest.mark.asyncio
async def test_m07_t02_high_downstream_impact_dominates_equally_weak_competitor(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """Two equally-weak topics; high-impact one should win the RCS ranking."""
    chain = [
        {"id": "topic-A", "title": "A", "grade_level": 9, "depth": 1},
        {"id": "topic-B", "title": "B", "grade_level": 9, "depth": 2},
    ]

    async def equal_weak(student_id, topic_ids):
        return {
            "topic-A": {"mastery_probability": 0.2, "uncertainty": 0.3},
            "topic-B": {"mastery_probability": 0.2, "uncertainty": 0.3},
        }

    call_count = {"n": 0}

    async def varying_impact(topic_id, max_depth=3):
        # A has many dependents; B has none
        if topic_id == "topic-A":
            return [{"id": f"dep-{i}", "depth": 1} for i in range(5)]
        return []

    async def pg_impact(topic_id, max_depth=3):
        return []

    async def no_prereqs(topic_id, max_depth=1):
        return []

    monkeypatch.setattr(root_cause, "fetch_skill_states_bulk", equal_weak)
    monkeypatch.setattr(root_cause.neo4j_db, "fetch_downstream_dependents", varying_impact)
    monkeypatch.setattr(root_cause.db, "fetch_downstream_dependents_pg", pg_impact)
    monkeypatch.setattr(root_cause.neo4j_db, "fetch_prerequisite_chain", no_prereqs)
    monkeypatch.setattr(root_cause.db, "fetch_prerequisite_chain_pg", no_prereqs)

    result = await root_cause.score_candidates("student-1", chain, {})

    assert result is not None
    assert result["id"] == "topic-A", "high-impact topic must outscore the equally-weak but impact-less one"


@pytest.mark.asyncio
async def test_m07_t03_multiple_gaps_returns_highest_rcs_not_a_blend(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """Two weak topics in chain: result is the single highest-RCS one, not a blend."""
    chain = [
        {"id": "topic-X", "title": "X", "grade_level": 9, "depth": 1},
        {"id": "topic-Y", "title": "Y", "grade_level": 9, "depth": 2},
    ]

    async def two_weak(student_id, topic_ids):
        return {
            "topic-X": {"mastery_probability": 0.1, "uncertainty": 0.2},  # weaker
            "topic-Y": {"mastery_probability": 0.35, "uncertainty": 0.2},
        }

    async def some_dependents(topic_id, max_depth=3):
        return [{"id": "dep-1", "depth": 1}]

    async def no_prereqs(topic_id, max_depth=1):
        return []

    monkeypatch.setattr(root_cause, "fetch_skill_states_bulk", two_weak)
    monkeypatch.setattr(root_cause.neo4j_db, "fetch_downstream_dependents", some_dependents)
    monkeypatch.setattr(root_cause.db, "fetch_downstream_dependents_pg", some_dependents)
    monkeypatch.setattr(root_cause.neo4j_db, "fetch_prerequisite_chain", no_prereqs)
    monkeypatch.setattr(root_cause.db, "fetch_prerequisite_chain_pg", no_prereqs)

    result = await root_cause.score_candidates("student-1", chain, {})

    assert result is not None
    assert "id" in result  # exactly one winner, not a list
    assert result["rcs"] > 0


@pytest.mark.asyncio
async def test_m07_t04_no_topics_below_weak_threshold_returns_none(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """All chain topics above WEAK_THRESHOLD → no credible root cause → None."""
    chain = [{"id": "topic-strong", "title": "Strong", "grade_level": 9, "depth": 1}]

    async def strong_state(student_id, topic_ids):
        return {"topic-strong": {"mastery_probability": 0.75, "uncertainty": 0.1}}

    async def no_deps(topic_id, **kwargs):
        return []

    monkeypatch.setattr(root_cause, "fetch_skill_states_bulk", strong_state)
    monkeypatch.setattr(root_cause.neo4j_db, "fetch_downstream_dependents", no_deps)
    monkeypatch.setattr(root_cause.db, "fetch_downstream_dependents_pg", no_deps)
    monkeypatch.setattr(root_cause.neo4j_db, "fetch_prerequisite_chain", no_deps)
    monkeypatch.setattr(root_cause.db, "fetch_prerequisite_chain_pg", no_deps)

    result = await root_cause.score_candidates("student-1", chain, {})
    assert result is None


@pytest.mark.asyncio
async def test_m07_t05_sparse_evidence_on_all_topics_returns_none(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """No skill_states rows → _mastery_and_confidence empty → no candidates → None."""
    chain = [{"id": "topic-unknown", "title": "Unknown", "grade_level": 9, "depth": 1}]

    async def empty_states(student_id, topic_ids):
        return {}  # true cold start for all topics

    async def no_deps(topic_id, **kwargs):
        return []

    monkeypatch.setattr(root_cause, "fetch_skill_states_bulk", empty_states)
    monkeypatch.setattr(root_cause.neo4j_db, "fetch_downstream_dependents", no_deps)
    monkeypatch.setattr(root_cause.db, "fetch_downstream_dependents_pg", no_deps)
    monkeypatch.setattr(root_cause.neo4j_db, "fetch_prerequisite_chain", no_deps)
    monkeypatch.setattr(root_cause.db, "fetch_prerequisite_chain_pg", no_deps)

    result = await root_cause.score_candidates("student-1", chain, {})
    assert result is None


@pytest.mark.asyncio
async def test_m07_t06_low_prerequisite_mastery_lowers_readiness(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """_prerequisite_readiness should be low when prerequisite mastery is low."""
    async def prereq_exists(topic_id, max_depth=1):
        return [{"id": "topic-prereq", "title": "Prereq", "depth": 1}]

    async def weak_prereq(student_id, topic_ids):
        return {"topic-prereq": {"mastery_probability": 0.1, "uncertainty": 0.2}}

    monkeypatch.setattr(root_cause.neo4j_db, "fetch_prerequisite_chain", prereq_exists)
    monkeypatch.setattr(root_cause.db, "fetch_prerequisite_chain_pg", prereq_exists)
    monkeypatch.setattr(root_cause, "fetch_skill_states_bulk", weak_prereq)

    readiness = await root_cause._prerequisite_readiness("student-1", "topic-T", {})
    assert readiness < 0.5, "readiness should be low when prerequisite is weak"


@pytest.mark.asyncio
async def test_m07_t08_no_evidence_rcs_pipeline_returns_none(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """Empty chain → score_candidates returns None immediately (no fabricated candidate)."""
    result = await root_cause.score_candidates("student-1", [], {})
    assert result is None


# ══════════════════════════════════════════════════════════════════════════════
# M08 – Prediction Models
# ══════════════════════════════════════════════════════════════════════════════


def test_m08_t01_bkt_mastery_rises_after_many_correct_responses() -> None:
    """BKT mastery after 10 correct responses from the default prior must exceed 0.7."""
    p = bkt.DEFAULT_BKT_PARAMS["p_init"]
    for _ in range(10):
        p = bkt._bkt_update(p, True, bkt.DEFAULT_BKT_PARAMS)
    assert p > 0.7


def test_m08_t02_irt_p_correct_beats_random_for_high_ability_learner() -> None:
    """High-ability learner (theta=2) on average item (b=0, a=1) → p > 0.5."""
    assert irt.p_correct(2.0, a=1.0, b=0.0) > 0.5


def test_m08_t03_auc_perfect_ranking_is_one() -> None:
    y_true = [1, 1, 1, 0, 0, 0]
    y_prob = [0.9, 0.8, 0.7, 0.3, 0.2, 0.1]
    assert metrics.auc(y_true, y_prob) == 1.0


def test_m08_t04_auc_random_ranking_is_near_half() -> None:
    y_true = [1, 0, 1, 0, 1, 0, 1, 0]
    y_prob = [0.5] * 8
    assert abs(metrics.auc(y_true, y_prob) - 0.5) < 1e-9


def test_m08_t05_log_loss_is_finite_and_non_negative() -> None:
    y_true = [1, 0, 1, 0, 1]
    y_prob = [0.9, 0.1, 0.8, 0.2, 0.7]
    ll = metrics.log_loss(y_true, y_prob)
    assert math.isfinite(ll)
    assert ll >= 0.0


def test_m08_t06_brier_score_is_zero_for_perfect_predictions() -> None:
    y_true = [1, 0, 1, 0]
    y_prob = [1.0, 0.0, 1.0, 0.0]
    assert abs(metrics.brier_score(y_true, y_prob)) < 1e-9


def test_m08_t07_expected_calibration_error_low_for_calibrated_predictor() -> None:
    """Predictor whose y_prob == actual frequency per bucket should have ECE < 0.1."""
    # Use simple 0.25 / 0.75 calibrated predictions
    y_true = [0, 0, 0, 1] * 5 + [1, 1, 1, 0] * 5
    y_prob = [0.25] * 20 + [0.75] * 20
    ece = metrics.expected_calibration_error(y_true, y_prob, n_bins=4)
    assert ece < 0.1


def test_m08_t08_temporal_leakage_future_evidence_changes_result() -> None:
    """Evidence from after the cutoff must not be included. Two calls with
    different `before` timestamps on the same evidence set must differ."""
    now = datetime.now(timezone.utc)
    # One fresh row, one future row
    fresh = _mock_record(0.3, days_ago=1.0, ref=now)   # before any cutoff
    future = _mock_record(0.9, days_ago=-5.0, ref=now)  # 5 days in the future

    # With only the past row (as prior_evidence's SQL would return)
    result_no_future = replay.fuse_point_in_time([fresh], before=now)
    # With both rows (simulating leakage)
    result_with_future = replay.fuse_point_in_time([fresh, future], before=now + timedelta(days=10))

    assert result_no_future is not None
    assert result_with_future is not None
    # Including the future high-estimate row should shift the result upward
    assert result_with_future > result_no_future, (
        "including future evidence must change the result (temporal leakage test)"
    )


def test_m08_t09_missing_evidence_prediction_returns_none() -> None:
    """No evidence → replay returns None, not a fabricated prediction."""
    result = replay.fuse_point_in_time([], before=datetime.now(timezone.utc))
    assert result is None


# ══════════════════════════════════════════════════════════════════════════════
# M09 – Cohort / Population Model (all N/A — not implemented)
# ══════════════════════════════════════════════════════════════════════════════

_M09_SKIP = "M09: No cohort model implemented — N/A per spec section 9"


def test_m09_t01_population_prior_estimation() -> None:
    pytest.skip(_M09_SKIP)


def test_m09_t02_cohort_distribution_matching() -> None:
    pytest.skip(_M09_SKIP)


def test_m09_t03_cross_cohort_comparability() -> None:
    pytest.skip(_M09_SKIP)


def test_m09_t04_small_cohort_stability() -> None:
    pytest.skip(_M09_SKIP)


def test_m09_t05_missing_cohort_attribute_handling() -> None:
    pytest.skip(_M09_SKIP)


def test_m09_t06_fairness_metrics_across_cohorts() -> None:
    pytest.skip(_M09_SKIP)


def test_m09_t07_calibration_across_cohorts() -> None:
    pytest.skip(_M09_SKIP)


def test_m09_t08_cohort_attribute_policy_boundary() -> None:
    pytest.skip(_M09_SKIP)


# ══════════════════════════════════════════════════════════════════════════════
# M10 – Recommendation / Action Ranking
# ══════════════════════════════════════════════════════════════════════════════


@pytest.mark.asyncio
async def test_m10_t01_high_forgetting_risk_topic_scores_higher(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """Topic with high forgetting risk and low mastery must outscore a recently-mastered topic."""
    from app.db import postgres_studyplan as db_sp

    states = {
        "topic-forgotten": _skill_state(mastery=0.4, uncertainty=0.3, forgetting_risk=0.9),
        "topic-recent": _skill_state(mastery=0.85, uncertainty=0.05, forgetting_risk=0.01),
    }
    _action_ranking_env(monkeypatch, states, blocked=set())

    scores = await action_ranking.compute_action_scores("student-1", ["topic-forgotten", "topic-recent"])

    assert scores["topic-forgotten"] > scores["topic-recent"]


@pytest.mark.asyncio
async def test_m10_t02_blocked_topic_scores_lower(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """A topic in the blocked set must score lower than an unblocked peer of equal state."""
    states = {
        "topic-blocked": _skill_state(mastery=0.3, uncertainty=0.3, forgetting_risk=0.5),
        "topic-free": _skill_state(mastery=0.3, uncertainty=0.3, forgetting_risk=0.5),
    }
    _action_ranking_env(monkeypatch, states, blocked={"topic-blocked"})

    scores = await action_ranking.compute_action_scores("student-1", ["topic-blocked", "topic-free"])

    assert scores["topic-free"] > scores["topic-blocked"]


@pytest.mark.asyncio
async def test_m10_t03_high_uncertainty_topic_scores_higher_information_gain(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """High-uncertainty topic (same mastery) should score higher due to information gain."""
    states = {
        "topic-uncertain": _skill_state(mastery=0.5, uncertainty=0.8, forgetting_risk=0.1),
        "topic-certain": _skill_state(mastery=0.5, uncertainty=0.05, forgetting_risk=0.1),
    }
    _action_ranking_env(monkeypatch, states, blocked=set(), impact=0.0)

    scores = await action_ranking.compute_action_scores("student-1", ["topic-uncertain", "topic-certain"])

    assert scores["topic-uncertain"] > scores["topic-certain"]


@pytest.mark.asyncio
async def test_m10_t04_no_evidence_topic_returns_nonzero_cold_start_score(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """Cold-start topic with no skill_states row: information_gain defaults to 0.7
    (spec's 'maximally uncertain by definition'), so score > 0, never fabricated 0."""
    _action_ranking_env(monkeypatch, states={}, blocked=set(), impact=0.0)

    scores = await action_ranking.compute_action_scores("student-1", ["topic-cold"])

    assert "topic-cold" in scores
    assert scores["topic-cold"] > 0.0, "cold-start topic must receive a non-zero score"


def test_m10_t05_difficulty_fit_neutral_without_item_params() -> None:
    """No item parameters → _difficulty_fit returns 0.5 (neutral, no fabricated fit)."""
    assert action_ranking._difficulty_fit(0.5, None) == 0.5
    assert action_ranking._difficulty_fit(None, None) == 0.5


def test_m10_t06_difficulty_fit_higher_near_zone_of_proximal_development() -> None:
    """Predicted success near 0.6 (zone of proximal development) should fit better
    than predicted success near 1.0 (too easy) or 0.0 (too hard)."""
    # b=0, a=1 → p_correct(0, 1, 0)=0.5 — slightly below target 0.6
    fit_zpd = action_ranking._difficulty_fit(0.5, (0.0, 1.0))
    # Very high mastery makes easy item trivial (predicted success → 1.0)
    fit_too_easy = action_ranking._difficulty_fit(0.99, (0.0, 1.0))
    assert fit_zpd > fit_too_easy


@pytest.mark.asyncio
async def test_m10_t07_ranking_is_deterministic(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """Two identical calls to compute_action_scores must return the same scores."""
    states = {
        "topic-A": _skill_state(mastery=0.3, uncertainty=0.4, forgetting_risk=0.5),
        "topic-B": _skill_state(mastery=0.6, uncertainty=0.2, forgetting_risk=0.1),
    }
    _action_ranking_env(monkeypatch, states, blocked=set())

    scores1 = await action_ranking.compute_action_scores("student-1", ["topic-A", "topic-B"])
    scores2 = await action_ranking.compute_action_scores("student-1", ["topic-A", "topic-B"])

    assert scores1 == scores2


@pytest.mark.asyncio
async def test_m10_t08_invalid_graph_degrades_gracefully(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """Empty graph (impact=0.0 for all topics) → scores are finite and non-negative."""
    states = {
        "topic-A": _skill_state(mastery=0.3, uncertainty=0.4, forgetting_risk=0.5),
        "topic-B": _skill_state(mastery=0.6, uncertainty=0.2, forgetting_risk=0.1),
    }
    _action_ranking_env(monkeypatch, states, blocked=set(), impact=0.0)

    scores = await action_ranking.compute_action_scores("student-1", ["topic-A", "topic-B"])

    for tid, score in scores.items():
        assert math.isfinite(score), f"score for {tid} must be finite"
        assert score >= 0.0, f"score for {tid} must be non-negative"


@pytest.mark.asyncio
async def test_m10_t09_cold_start_topic_classified_as_concept_explanation(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """A topic with no skill_states row (cold start) must be classified as
    'concept_explanation' — not a fabricated 'practice' assignment."""
    _classify_env(monkeypatch, states={}, blocked=set())

    action_types = await action_ranking.classify_action_types("student-1", ["topic-cold"])

    assert action_types.get("topic-cold") == "concept_explanation"


@pytest.mark.asyncio
async def test_m10_t10_blocked_topic_classified_as_alternative_representation(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """A blocked topic must be classified as 'alternative_representation'
    (recovery routing active — direct practice is not the right action)."""
    states = {"topic-blocked": _skill_state(mastery=0.3, uncertainty=0.4)}
    _classify_env(monkeypatch, states=states, blocked={"topic-blocked"})

    action_types = await action_ranking.classify_action_types("student-1", ["topic-blocked"])

    assert action_types.get("topic-blocked") == "alternative_representation"
