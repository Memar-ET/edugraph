"""Tests for app/services/knowledge_tracing/dina.py's joint DINA
posterior update -- checklist requirement "unit-test learner-state
update calculations" applied to the conjunctive (AND) cognitive
diagnosis model.
"""

from __future__ import annotations

from app.services.knowledge_tracing.dina import _joint_dina_update


def test_single_skill_correct_response_increases_posterior() -> None:
    posterior = _joint_dina_update([0.3], True, slip=0.2, guess=0.2)
    assert posterior[0] > 0.3


def test_single_skill_incorrect_response_decreases_posterior() -> None:
    posterior = _joint_dina_update([0.3], False, slip=0.2, guess=0.2)
    assert posterior[0] < 0.3


def test_posterior_stays_in_unit_interval() -> None:
    for prior in (0.01, 0.3, 0.5, 0.7, 0.99):
        for correct in (True, False):
            posterior = _joint_dina_update([prior], correct, slip=0.2, guess=0.2)
            assert 0.0 <= posterior[0] <= 1.0


def test_conjunctive_item_requires_all_skills_for_full_credit_belief() -> None:
    # Two skills, symmetric priors: a correct response should raise BOTH
    # skills' posteriors roughly equally (the item's eta = AND of both).
    posterior = _joint_dina_update([0.5, 0.5], True, slip=0.2, guess=0.2)
    assert posterior[0] > 0.5
    assert posterior[1] > 0.5
    assert abs(posterior[0] - posterior[1]) < 1e-9  # symmetric priors -> symmetric posteriors


def test_asymmetric_priors_produce_asymmetric_posteriors() -> None:
    posterior = _joint_dina_update([0.9, 0.1], True, slip=0.2, guess=0.2)
    # The already-strong skill should end up more confidently mastered
    # than the weak one, even though both received the same observation.
    assert posterior[0] > posterior[1]


def test_extreme_slip_and_guess_still_move_posterior_in_correct_direction() -> None:
    # Even with noisy parameters, a correct response must not decrease
    # belief and an incorrect response must not increase it.
    posterior_correct = _joint_dina_update([0.5], True, slip=0.4, guess=0.4)
    posterior_incorrect = _joint_dina_update([0.5], False, slip=0.4, guess=0.4)
    assert posterior_correct[0] >= 0.5
    assert posterior_incorrect[0] <= 0.5
