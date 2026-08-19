"""Tests for app/services/knowledge_tracing/bkt.py's pure BKT math --
checklist requirement "unit-test learner-state update calculations" /
"unit-test uncertainty propagation."
"""

from __future__ import annotations

from app.services.knowledge_tracing.bkt import DEFAULT_BKT_PARAMS, _bkt_update, _uncertainty_for


def test_correct_response_increases_mastery() -> None:
    p_next = _bkt_update(0.3, True, DEFAULT_BKT_PARAMS)
    assert p_next > 0.3


def test_incorrect_response_decreases_mastery() -> None:
    p_next = _bkt_update(0.3, False, DEFAULT_BKT_PARAMS)
    assert p_next < 0.3


def test_mastery_stays_in_unit_interval_across_many_updates() -> None:
    p = 0.3
    for correct in [True, True, False, True, False, False, True, True, True, False]:
        p = _bkt_update(p, correct, DEFAULT_BKT_PARAMS)
        assert 0.0 <= p <= 1.0


def test_repeated_correct_responses_converge_toward_mastery() -> None:
    p = DEFAULT_BKT_PARAMS["p_init"]
    for _ in range(10):
        p = _bkt_update(p, True, DEFAULT_BKT_PARAMS)
    assert p > 0.9


def test_repeated_incorrect_responses_stay_low_but_never_hit_zero() -> None:
    # A learner can never be updated to *exactly* zero mastery -- BKT's
    # transition step (p_transit) always adds some probability of having
    # learned, and slip/guess keep the posterior from collapsing to a
    # hard 0/1. This is a deliberate property of the model, not a bug.
    p = DEFAULT_BKT_PARAMS["p_init"]
    for _ in range(10):
        p = _bkt_update(p, False, DEFAULT_BKT_PARAMS)
    assert 0.0 < p < 0.3


def test_slip_and_guess_bound_the_immediate_posterior_update() -> None:
    # With a high slip and high guess, one response barely moves belief
    # -- a sanity check that the parameters are actually being used.
    noisy_params = {"p_init": 0.3, "p_transit": 0.0, "p_slip": 0.45, "p_guess": 0.45}
    p_correct = _bkt_update(0.5, True, noisy_params)
    p_incorrect = _bkt_update(0.5, False, noisy_params)
    assert abs(p_correct - 0.5) < 0.2
    assert abs(p_incorrect - 0.5) < 0.2


def test_uncertainty_decreases_monotonically_with_evidence_count() -> None:
    values = [_uncertainty_for(n) for n in (0, 1, 3, 10, 50)]
    assert values == sorted(values, reverse=True)
    assert values[-1] >= 0.05  # floor, never claims perfect certainty
