"""Tests for app/services/evaluation/metrics.py -- checklist requirement
"define response-prediction metrics: AUC, log loss, and calibration-aware
metrics" / "measure calibration using Brier score, expected calibration
error." No sklearn in this repo, so these are independently verified
against hand-computed expectations rather than a reference library.
"""

from __future__ import annotations

import math

from app.services.evaluation import metrics


def test_auc_perfect_separation_is_one() -> None:
    y_true = [1, 1, 1, 0, 0, 0]
    y_prob = [0.9, 0.8, 0.7, 0.3, 0.2, 0.1]
    assert metrics.auc(y_true, y_prob) == 1.0


def test_auc_perfectly_wrong_ranking_is_zero() -> None:
    y_true = [1, 0, 1, 0]
    y_prob = [0.1, 0.9, 0.1, 0.9]
    assert metrics.auc(y_true, y_prob) == 0.0


def test_auc_random_ranking_is_near_one_half() -> None:
    y_true = [1, 0, 1, 0, 1, 0, 1, 0]
    y_prob = [0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5, 0.5]
    assert abs(metrics.auc(y_true, y_prob) - 0.5) < 1e-9


def test_auc_is_none_for_a_single_class() -> None:
    # Undefined without both classes present -- must not silently return
    # a fabricated 0.5.
    assert metrics.auc([1, 1, 1], [0.2, 0.5, 0.9]) is None
    assert metrics.auc([0, 0], [0.1, 0.9]) is None


def test_auc_handles_tied_scores_via_average_rank() -> None:
    y_true = [1, 0, 1, 0]
    y_prob = [0.5, 0.5, 0.5, 0.5]  # all tied -> should behave like random
    assert abs(metrics.auc(y_true, y_prob) - 0.5) < 1e-9


def test_log_loss_is_low_for_confident_correct_predictions() -> None:
    y_true = [1, 0]
    y_prob = [0.99, 0.01]
    assert metrics.log_loss(y_true, y_prob) < 0.02


def test_log_loss_is_high_for_confident_wrong_predictions() -> None:
    y_true = [1, 0]
    y_prob = [0.01, 0.99]
    assert metrics.log_loss(y_true, y_prob) > 4.0


def test_log_loss_clips_extreme_probabilities_to_avoid_infinity() -> None:
    # p=0 or p=1 would make log(0) = -inf without clipping.
    value = metrics.log_loss([1], [0.0])
    assert math.isfinite(value)


def test_brier_score_is_zero_for_perfect_predictions() -> None:
    assert metrics.brier_score([1, 0], [1.0, 0.0]) == 0.0


def test_brier_score_is_one_for_maximally_wrong_predictions() -> None:
    assert metrics.brier_score([1, 0], [0.0, 1.0]) == 1.0


def test_expected_calibration_error_is_zero_for_perfectly_calibrated_predictions() -> None:
    # Every bucket's predicted probability matches its actual outcome
    # rate exactly.
    y_true = [1.0, 0.0]
    y_prob = [1.0, 0.0]
    assert metrics.expected_calibration_error(y_true, y_prob) == 0.0


def test_expected_calibration_error_detects_systematic_overconfidence() -> None:
    # Predicts 0.9 for everything, but only half are actually correct --
    # badly miscalibrated, ECE should be large.
    y_true = [1, 0, 1, 0, 1, 0]
    y_prob = [0.9, 0.9, 0.9, 0.9, 0.9, 0.9]
    assert metrics.expected_calibration_error(y_true, y_prob) > 0.3
