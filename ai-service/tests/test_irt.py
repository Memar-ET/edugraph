"""Tests for app/services/knowledge_tracing/irt.py's 2PL IRT math --
checklist requirement "unit-test learner-state update calculations"
applied to Newton-Raphson ability (theta) estimation.
"""

from __future__ import annotations

from app.services.knowledge_tracing.irt import THETA_CLAMP, _estimate_theta, p_correct


def test_p_correct_at_theta_equals_b_is_one_half() -> None:
    # 2PL definition: when ability exactly matches item difficulty, the
    # probability of a correct response is exactly 0.5, regardless of a.
    assert abs(p_correct(0.0, a=1.0, b=0.0) - 0.5) < 1e-9
    assert abs(p_correct(1.5, a=2.0, b=1.5) - 0.5) < 1e-9


def test_p_correct_increases_with_theta() -> None:
    low = p_correct(-1.0, a=1.0, b=0.0)
    high = p_correct(1.0, a=1.0, b=0.0)
    assert high > low


def test_higher_discrimination_sharpens_the_curve_away_from_b() -> None:
    low_a = p_correct(1.0, a=0.5, b=0.0)
    high_a = p_correct(1.0, a=3.0, b=0.0)
    assert high_a > low_a  # steeper item response function above b


def test_theta_estimate_is_positive_when_mostly_correct() -> None:
    items = [(1.0, 0.0, True), (1.0, 0.0, True), (1.0, 0.5, True)]
    theta, se = _estimate_theta(items)
    assert theta > 0
    assert se > 0


def test_theta_estimate_is_negative_when_mostly_incorrect() -> None:
    items = [(1.0, 0.0, False), (1.0, 0.0, False), (1.0, -0.5, False)]
    theta, se = _estimate_theta(items)
    assert theta < 0


def test_theta_estimate_clamps_at_bounds_for_degenerate_all_correct() -> None:
    # All-correct is a classic MLE degeneracy for IRT -- the true MLE is
    # +infinity, so clamping to THETA_CLAMP (with a correspondingly large
    # standard error) is the correct, expected behavior, not a bug.
    theta, se = _estimate_theta([(1.0, 0.0, True)] * 5)
    assert theta == THETA_CLAMP
    assert se > 1.0  # high uncertainty signals the degenerate estimate


def test_theta_estimate_is_neutral_for_balanced_responses() -> None:
    items = [(1.0, 0.0, True), (1.0, 0.0, False)]
    theta, _se = _estimate_theta(items)
    assert abs(theta) < 0.5
