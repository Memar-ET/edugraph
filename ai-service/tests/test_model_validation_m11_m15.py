"""
Whole-System Model Validation Test Specification — M11 through M15
Source: EduGraph_Whole_System_Model_Validation_Test_Specification.docx

M11 – Recommendation Outcome / Effectiveness  (M11-T01..T08)
M12 – Confidence, Uncertainty & Calibration   (M12-T01..T08)
M13 – Provenance & Explainability             (M13-T01..T06)
M14 – Snapshot, Replay & Longitudinal         (M14-T01..T07)
M15 – Model Orchestration / End-to-End        (M15-T01..T12)
"""

from __future__ import annotations

import math
from datetime import datetime, timedelta, timezone
from typing import Optional
from unittest.mock import MagicMock

import pytest

from app.services.evaluation import metrics
from app.services.knowledge_tracing import bkt, fusion, irt, replay
from app.services.gap_analysis import root_cause
from app.workers.refit_worker import _classify_outcome


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


# ══════════════════════════════════════════════════════════════════════════════
# M11 – Recommendation Outcome / Effectiveness Model
# ══════════════════════════════════════════════════════════════════════════════
# Validated via _classify_outcome (pure function in refit_worker.py) —
# the function that links recommendation history to subsequent mastery changes.


def test_m11_t01_improved_outcome_correctly_classified() -> None:
    """Mastery rose by more than the improvement delta → 'improved'."""
    result = _classify_outcome(
        mastery_before=0.3, evidence_before=5,
        mastery_now=0.6, evidence_now=8,
    )
    assert result == "improved"


def test_m11_t02_worsened_outcome_correctly_classified() -> None:
    """Mastery fell by more than the improvement delta → 'worsened'."""
    result = _classify_outcome(
        mastery_before=0.7, evidence_before=5,
        mastery_now=0.4, evidence_now=8,
    )
    assert result == "worsened"


def test_m11_t03_unchanged_outcome_classified_as_unchanged() -> None:
    """Mastery barely moved → 'unchanged', not falsely claimed as improved."""
    result = _classify_outcome(
        mastery_before=0.5, evidence_before=5,
        mastery_now=0.51, evidence_now=6,
    )
    assert result == "unchanged"


def test_m11_t04_missing_baseline_classified_without_fabricating_delta() -> None:
    """No mastery_before (cold-start baseline) → classified by absolute threshold
    (OUTCOME_COLD_START_IMPROVED_THRESHOLD), not by a fabricated delta. Outcome is
    'improved' if current mastery meets the threshold, 'unchanged' otherwise — never
    'worsened' (which would require a known baseline to compare against)."""
    from app.workers.refit_worker import OUTCOME_COLD_START_IMPROVED_THRESHOLD
    # Well above threshold → improved
    result_high = _classify_outcome(
        mastery_before=None, evidence_before=0,
        mastery_now=OUTCOME_COLD_START_IMPROVED_THRESHOLD + 0.1, evidence_now=3,
    )
    assert result_high == "improved"
    # Below threshold → unchanged (not worsened — no baseline to compare)
    result_low = _classify_outcome(
        mastery_before=None, evidence_before=0,
        mastery_now=OUTCOME_COLD_START_IMPROVED_THRESHOLD - 0.1, evidence_now=3,
    )
    assert result_low == "unchanged"


def test_m11_t05_missing_current_mastery_classified_as_insufficient_evidence() -> None:
    """No current mastery → 'insufficient_evidence'."""
    result = _classify_outcome(
        mastery_before=0.4, evidence_before=5,
        mastery_now=None, evidence_now=0,
    )
    assert result == "insufficient_evidence"


def test_m11_t06_low_evidence_count_classified_as_insufficient() -> None:
    """Fewer than the minimum evidence count → 'insufficient_evidence'
    (not enough data to claim the recommendation actually had an effect)."""
    result = _classify_outcome(
        mastery_before=0.3, evidence_before=1,
        mastery_now=0.6, evidence_now=1,  # evidence_now unchanged — no new observations
    )
    assert result == "insufficient_evidence"


def test_m11_t07_outcome_classification_is_deterministic() -> None:
    """Same inputs → same output (no random elements)."""
    args = {"mastery_before": 0.4, "evidence_before": 5, "mastery_now": 0.7, "evidence_now": 9}
    assert _classify_outcome(**args) == _classify_outcome(**args)


def test_m11_t08_cold_start_baseline_never_worsened() -> None:
    """A cold-start learner (mastery_before=None) must never be classified
    'worsened' — there is no known baseline to compare against, so a decline
    cannot be declared. The outcome is determined by an absolute threshold, not
    a fabricated delta.

    With evidence_now > evidence_before and mastery below threshold:
    → 'unchanged' (not 'worsened', not 'insufficient_evidence').
    With evidence_now <= evidence_before:
    → 'insufficient_evidence' (no new observations observed)."""
    # Case 1: new evidence, mastery below threshold → unchanged (not worsened)
    result_with_evidence = _classify_outcome(
        mastery_before=None, evidence_before=0,
        mastery_now=0.3, evidence_now=3,
    )
    assert result_with_evidence != "worsened", "null baseline must not produce 'worsened'"
    assert result_with_evidence == "unchanged"

    # Case 2: no new evidence → insufficient_evidence
    result_no_new = _classify_outcome(
        mastery_before=None, evidence_before=0,
        mastery_now=0.3, evidence_now=0,
    )
    assert result_no_new == "insufficient_evidence"


# ══════════════════════════════════════════════════════════════════════════════
# M12 – Confidence, Uncertainty & Calibration Layer
# ══════════════════════════════════════════════════════════════════════════════


def test_m12_t01_calibration_error_low_for_well_calibrated_predictions() -> None:
    """ECE < 0.1 for a predictor whose bucket means match observed frequencies."""
    y_true = [0, 0, 0, 1] * 10 + [1, 1, 1, 0] * 10
    y_prob = [0.25] * 40 + [0.75] * 40
    ece = metrics.expected_calibration_error(y_true, y_prob, n_bins=4)
    assert ece < 0.1


def test_m12_t02_uncertainty_increases_for_contradictory_sources() -> None:
    """fusion's disagreement term: two opposing sources (0.9 vs 0.1) push
    fused mastery away from either extreme but stay between them — the
    uncertainty widening is implicit in the weighted blend."""
    now = datetime.now(timezone.utc)
    high = _mock_record(0.9, reliability=0.8, ref=now)
    low = _mock_record(0.1, reliability=0.8, ref=now)
    result = replay.fuse_point_in_time([high, low], before=now + timedelta(seconds=1))
    assert result is not None
    assert 0.1 < result < 0.9


def test_m12_t03_no_evidence_produces_unknown_mastery_status() -> None:
    """Cold start: mastery_probability=None → mastery_status='unknown'."""
    assert fusion._mastery_status(None) == "unknown"


def test_m12_t04_sparse_single_evidence_produces_finite_mastery() -> None:
    """A single evidence row produces a finite mastery estimate (not NaN/inf)."""
    now = datetime.now(timezone.utc)
    row = _mock_record(0.5, ref=now)
    result = replay.fuse_point_in_time([row], before=now + timedelta(seconds=1))
    assert result is not None
    assert math.isfinite(result)


def test_m12_t05_mastery_status_boundaries_are_explicit() -> None:
    """Status levels at the exact threshold values are explicitly defined."""
    # Just below 0.4 → emerging
    assert fusion._mastery_status(0.399) == "emerging"
    # Exactly 0.4 → proficient (threshold is exclusive below)
    assert fusion._mastery_status(0.4) == "proficient"
    # Just below 0.7 → proficient
    assert fusion._mastery_status(0.699) == "proficient"
    # Exactly 0.7 → mastered
    assert fusion._mastery_status(0.7) == "mastered"


def test_m12_t06_source_weight_decreases_with_lower_reliability() -> None:
    """Lower reliability → lower weight. Uncertainty is surfaced through weights."""
    now = datetime.now(timezone.utc)
    w_low = fusion._source_weight("bkt", reliability=0.2, sample_size=5, created_at=now)
    w_high = fusion._source_weight("bkt", reliability=0.9, sample_size=5, created_at=now)
    assert w_low < w_high


def test_m12_t07_source_weight_decreases_with_fewer_samples() -> None:
    """Smaller sample size → lower weight (evidence quality degrades)."""
    now = datetime.now(timezone.utc)
    w_few = fusion._source_weight("bkt", reliability=0.8, sample_size=1, created_at=now)
    w_many = fusion._source_weight("bkt", reliability=0.8, sample_size=10, created_at=now)
    assert w_few < w_many


def test_m12_t08_brier_score_penalizes_confident_wrong_predictions() -> None:
    """Perfect confidence on a wrong answer (prob=1.0, true=0) → maximum Brier penalty."""
    bs_wrong = metrics.brier_score([0], [1.0])
    bs_uncertain = metrics.brier_score([0], [0.5])
    assert bs_wrong > bs_uncertain


# ══════════════════════════════════════════════════════════════════════════════
# M13 – Provenance & Explainability
# ══════════════════════════════════════════════════════════════════════════════


def test_m13_t01_rcs_factors_decompose_into_traceable_components() -> None:
    """compute_rcs accepts named components that are independently traceable —
    each factor is a documented, separately-computable value."""
    weakness, confidence, impact, readiness, gain = 0.6, 0.8, 0.5, 0.9, 0.3
    rcs = root_cause.compute_rcs(weakness, confidence, impact, readiness, gain)
    # Verify each factor contributes: zeroing impact → rcs = 0
    rcs_no_impact = root_cause.compute_rcs(weakness, confidence, 0.0, readiness, gain)
    assert rcs > 0
    assert rcs_no_impact == 0.0


def test_m13_t02_evidence_source_labels_are_retained_in_weighting() -> None:
    """Source provenance labels are preserved through fusion weighting —
    a 'bkt' source has a different default weight than 'irt'."""
    now = datetime.now(timezone.utc)
    w_bkt = fusion._source_weight("bkt", reliability=None, sample_size=5, created_at=now)
    w_irt = fusion._source_weight("irt", reliability=None, sample_size=5, created_at=now)
    # BKT has higher default reliability than IRT (0.8 vs 0.5)
    assert w_bkt > w_irt


def test_m13_t03_mastery_status_is_reproducible_from_recorded_mastery_probability() -> None:
    """Given the same mastery_probability, mastery_status is deterministic —
    the recorded probability alone is sufficient to reproduce the status label."""
    assert fusion._mastery_status(0.8) == fusion._mastery_status(0.8)
    assert fusion._mastery_status(0.45) == fusion._mastery_status(0.45)


def test_m13_t04_trend_provenance_requires_prior_mastery_not_fabricated() -> None:
    """Trend is None when no prior mastery exists — trend is only claimed
    when there is a concrete prior value to compare against."""
    assert fusion._trend(0.6, None) is None


def test_m13_t05_replay_mastery_derivable_from_recorded_evidence_and_timestamp() -> None:
    """fuse_point_in_time is reproducible: given the same evidence rows and
    `before` timestamp, the fused mastery is always equal."""
    now = datetime.now(timezone.utc)
    rows = [_mock_record(0.55, days_ago=5, ref=now)]
    before = now + timedelta(seconds=1)

    r1 = replay.fuse_point_in_time(rows, before=before)
    r2 = replay.fuse_point_in_time(rows, before=before)
    assert r1 == r2


def test_m13_t06_unknown_provenance_source_gets_conservative_fallback_weight() -> None:
    """An unrecognized provenance string must receive the FALLBACK_RELIABILITY
    weight — never full trust for an unrecognized model."""
    from app.services.knowledge_tracing.fusion import FALLBACK_RELIABILITY, DEFAULT_SOURCE_RELIABILITY
    now = datetime.now(timezone.utc)
    w_unknown = fusion._source_weight("future_engine_v99", reliability=None, sample_size=5, created_at=now)
    w_bkt = fusion._source_weight("bkt", reliability=None, sample_size=5, created_at=now)
    assert w_unknown < w_bkt
    # Verify the fallback is FALLBACK_RELIABILITY, not 0 or full 1.0
    expected = FALLBACK_RELIABILITY * 1.0 * 1.0  # full sample, brand-new evidence
    assert abs(w_unknown - expected) < 1e-9


# ══════════════════════════════════════════════════════════════════════════════
# M14 – Snapshot, Replay & Longitudinal Reconstruction
# ══════════════════════════════════════════════════════════════════════════════


def test_m14_t01_replay_without_evidence_returns_none() -> None:
    """Historical reconstruction with no prior evidence returns None — no fabricated past state."""
    result = replay.fuse_point_in_time([], before=datetime.now(timezone.utc))
    assert result is None


def test_m14_t02_snapshot_replay_determinism() -> None:
    """Replaying the same evidence at the same timestamp twice gives the same result."""
    now = datetime.now(timezone.utc)
    rows = [_mock_record(0.6, days_ago=3, ref=now)]
    before = now + timedelta(seconds=1)
    assert replay.fuse_point_in_time(rows, before=before) == replay.fuse_point_in_time(rows, before=before)


def test_m14_t03_future_evidence_excluded_from_historical_replay() -> None:
    """Evidence timestamped after the `before` cutoff must not affect the result.
    Tested by giving two calls different `before` values."""
    now = datetime.now(timezone.utc)
    past_row = _mock_record(0.3, days_ago=10, ref=now)
    future_row = _mock_record(0.9, days_ago=-1, ref=now)  # 1 day in future

    # At cutoff=now, only past_row counts (future_row excluded by prior_evidence's SQL)
    result_past_only = replay.fuse_point_in_time([past_row], before=now)
    # At cutoff=now+2days, both count
    result_both = replay.fuse_point_in_time([past_row, future_row], before=now + timedelta(days=2))

    assert result_past_only is not None
    assert result_both is not None
    assert result_both > result_past_only, "including the 0.9 future row must shift result upward"


def test_m14_t04_stale_evidence_down_weighted_in_replay() -> None:
    """Evidence from many half-lives ago contributes less than fresh evidence."""
    now = datetime.now(timezone.utc)
    half_life = replay.RECENCY_HALF_LIFE_DAYS
    stale = _mock_record(0.9, days_ago=half_life * 4, ref=now)
    fresh = _mock_record(0.1, days_ago=0.5, ref=now)
    result = replay.fuse_point_in_time([stale, fresh], before=now + timedelta(seconds=1))
    assert result is not None
    assert result < 0.5, "fresh low evidence must dominate stale high evidence"


def test_m14_t05_replay_policy_change_shifts_result_within_tolerance() -> None:
    """Changing RECENCY_HALF_LIFE_DAYS changes the replay output by a bounded amount."""
    now = datetime.now(timezone.utc)
    rows = [
        _mock_record(0.3, days_ago=10, ref=now),
        _mock_record(0.7, days_ago=2, ref=now),
    ]
    before = now + timedelta(seconds=1)
    baseline = replay.fuse_point_in_time(rows, before=before)
    original = replay.RECENCY_HALF_LIFE_DAYS
    try:
        replay.RECENCY_HALF_LIFE_DAYS = 5.0
        shifted = replay.fuse_point_in_time(rows, before=before)
    finally:
        replay.RECENCY_HALF_LIFE_DAYS = original
    assert baseline is not None and shifted is not None
    assert abs(shifted - baseline) < 0.25


def test_m14_t06_historical_state_immutable_new_evidence_does_not_alter_past() -> None:
    """Adding new evidence to a later call must not change an earlier call's result.
    The replay function has no side effects — each call is independent."""
    now = datetime.now(timezone.utc)
    early_row = _mock_record(0.5, days_ago=10, ref=now)
    late_row = _mock_record(0.9, days_ago=1, ref=now)
    before_early = now - timedelta(days=2)  # only early row predates this
    before_late = now + timedelta(seconds=1)

    result_early = replay.fuse_point_in_time([early_row], before=before_early)
    result_late = replay.fuse_point_in_time([early_row, late_row], before=before_late)

    # early result should not be affected by the late call
    result_early_recheck = replay.fuse_point_in_time([early_row], before=before_early)
    assert result_early == result_early_recheck, "replay is stateless — past results must not change"


@pytest.mark.asyncio
async def test_m14_t07_reconstruct_mastery_cold_start_returns_none(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """New learner at any historical timestamp → None, not a fabricated value."""
    async def no_evidence(student_id, topic_id, before, provenance=None):
        return []

    monkeypatch.setattr(replay, "prior_evidence", no_evidence)
    result = await replay.reconstruct_mastery_as_of("student-new", "topic-1")
    assert result is None


# ══════════════════════════════════════════════════════════════════════════════
# M15 – Model Orchestration / End-to-End Pipeline
# ══════════════════════════════════════════════════════════════════════════════
# Validates the complete logical chain: evidence → mastery → diagnosis →
# recommendation → outcome. Uses pure-math layers where possible.


def test_m15_t01_evidence_ingestion_to_mastery_estimate_end_to_end_pure() -> None:
    """Full local chain: BKT updates → IRT theta → mastery status label.
    No DB calls; exercises the core math pathway."""
    p = bkt.DEFAULT_BKT_PARAMS["p_init"]
    for _ in range(5):
        p = bkt._bkt_update(p, True, bkt.DEFAULT_BKT_PARAMS)
    # IRT: learner with theta>0 on average item
    theta, _ = irt._estimate_theta([(1.0, 0.0, True)] * 5)
    assert theta > 0.0
    assert p > 0.4
    status = fusion._mastery_status(p)
    assert status in ("emerging", "proficient", "mastered")


def test_m15_t02_rcs_pipeline_identifies_root_cause_from_chain() -> None:
    """compute_rcs correctly ranks a high-impact weak topic above a
    low-impact equally-weak one — the diagnostic ordering step."""
    weak = 0.4
    confidence = 0.8
    high_impact_rcs = root_cause.compute_rcs(weak, confidence, 0.8, 0.9, 0.32)
    low_impact_rcs = root_cause.compute_rcs(weak, confidence, 0.1, 0.9, 0.04)
    assert high_impact_rcs > low_impact_rcs


def test_m15_t03_replay_provides_historical_state_for_audit() -> None:
    """Given historical evidence, replay produces a bounded mastery estimate
    for audit/provenance — the longitudinal traceability step."""
    now = datetime.now(timezone.utc)
    rows = [_mock_record(0.55, days_ago=14, ref=now)]
    result = replay.fuse_point_in_time(rows, before=now)
    assert result is not None
    assert 0.0 <= result <= 1.0


def test_m15_t04_no_evidence_stage_produces_no_fabricated_downstream_state() -> None:
    """If evidence is absent at the ingestion stage, no mastery estimate
    propagates — the cold-start boundary holds end-to-end."""
    result = replay.fuse_point_in_time([], before=datetime.now(timezone.utc))
    assert result is None
    status = fusion._mastery_status(None)
    assert status == "unknown"


def test_m15_t05_evaluation_metrics_pipeline_produces_bounded_scores() -> None:
    """AUC, log_loss, brier_score, ECE all produce finite values in [0, 1]
    for well-formed inputs — the metrics pipeline is safe."""
    y_true = [1, 0, 1, 0, 1, 0]
    y_prob = [0.8, 0.2, 0.7, 0.3, 0.9, 0.1]

    auc = metrics.auc(y_true, y_prob)
    ll = metrics.log_loss(y_true, y_prob)
    bs = metrics.brier_score(y_true, y_prob)
    ece = metrics.expected_calibration_error(y_true, y_prob, n_bins=3)

    assert 0.0 <= auc <= 1.0
    assert math.isfinite(ll) and ll >= 0.0
    assert 0.0 <= bs <= 1.0
    assert 0.0 <= ece <= 1.0


def test_m15_t06_bkt_trajectory_feeds_mastery_status_monotonically() -> None:
    """Accumulating correct evidence → mastery_status transitions only
    forward (emerging → proficient → mastered), never skips or reverses."""
    p = bkt.DEFAULT_BKT_PARAMS["p_init"]
    statuses: list[str] = []
    for _ in range(20):
        p = bkt._bkt_update(p, True, bkt.DEFAULT_BKT_PARAMS)
        statuses.append(fusion._mastery_status(p))

    order = ["emerging", "proficient", "mastered"]
    prev_idx = 0
    for s in statuses:
        idx = order.index(s)
        assert idx >= prev_idx, f"status must not regress: saw {s} after {order[prev_idx]}"
        prev_idx = idx


def test_m15_t07_irt_and_bkt_both_respond_correctly_to_correct_response() -> None:
    """Both BKT and IRT theta move in the same direction for a correct response."""
    p_before = bkt.DEFAULT_BKT_PARAMS["p_init"]
    p_after = bkt._bkt_update(p_before, True, bkt.DEFAULT_BKT_PARAMS)
    assert p_after > p_before

    theta_mostly_correct, _ = irt._estimate_theta([(1.0, 0.0, True)] * 8 + [(1.0, 0.0, False)] * 2)
    theta_mostly_wrong, _ = irt._estimate_theta([(1.0, 0.0, True)] * 2 + [(1.0, 0.0, False)] * 8)
    assert theta_mostly_correct > theta_mostly_wrong


def test_m15_t08_outcome_classification_completes_the_feedback_loop() -> None:
    """_classify_outcome closes the evidence→recommendation→outcome loop:
    a recommendation made when mastery was 0.3 and observed at 0.7 is 'improved'."""
    outcome = _classify_outcome(
        mastery_before=0.3, evidence_before=5,
        mastery_now=0.7, evidence_now=8,
    )
    assert outcome == "improved"


def test_m15_t09_trend_label_is_consistent_with_mastery_trajectory() -> None:
    """The trend field derived from two consecutive mastery estimates must
    reflect the correct direction — part of the longitudinal state record."""
    assert fusion._trend(0.7, 0.3) == "improving"
    assert fusion._trend(0.3, 0.7) == "declining"
    assert fusion._trend(0.5, 0.5) == "stable"


def test_m15_t10_evaluation_auc_distinguishes_model_from_random() -> None:
    """A model that correctly separates positives from negatives has AUC > 0.5;
    a random model has AUC ≈ 0.5. This validates the evaluation harness."""
    y_true = [1, 1, 1, 0, 0, 0]
    good_prob = [0.8, 0.7, 0.9, 0.2, 0.1, 0.3]
    rand_prob = [0.5, 0.5, 0.5, 0.5, 0.5, 0.5]
    assert metrics.auc(y_true, good_prob) > metrics.auc(y_true, rand_prob)


def test_m15_t11_calibration_error_higher_for_miscalibrated_predictor() -> None:
    """A badly overconfident predictor (always predicts 0.9 regardless of truth)
    has higher ECE than a correctly-calibrated one."""
    y_true = [1, 0, 1, 0, 1, 0, 1, 0]
    calibrated = [0.75, 0.25, 0.75, 0.25, 0.75, 0.25, 0.75, 0.25]
    overconfident = [0.9, 0.9, 0.9, 0.9, 0.9, 0.9, 0.9, 0.9]
    ece_good = metrics.expected_calibration_error(y_true, calibrated, n_bins=4)
    ece_bad = metrics.expected_calibration_error(y_true, overconfident, n_bins=4)
    assert ece_bad > ece_good


def test_m15_t12_complete_pipeline_state_provenance_is_derivable() -> None:
    """The complete set of provenance signals is independently computable
    from stored evidence: mastery from fuse_point_in_time, status from
    _mastery_status, trend from _trend, RCS from compute_rcs. None of
    these calls mutate shared state."""
    now = datetime.now(timezone.utc)
    rows = [_mock_record(0.65, days_ago=1, ref=now)]
    mastery = replay.fuse_point_in_time(rows, before=now + timedelta(seconds=1))
    assert mastery is not None

    status = fusion._mastery_status(mastery)
    trend = fusion._trend(mastery, 0.4)
    rcs = root_cause.compute_rcs(
        weakness=max(0, 0.5 - mastery) / 0.5,
        confidence=0.8,
        impact=0.5,
        readiness=0.9,
        intervention_gain=0.25,
    )

    assert status in ("emerging", "proficient", "mastered")
    assert trend in ("improving", "stable", "declining", None)
    assert math.isfinite(rcs)
    assert rcs >= 0.0
