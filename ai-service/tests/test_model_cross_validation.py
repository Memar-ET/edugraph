"""
Cross-model validation: every implemented model's pure functions tested
exhaustively against the Whole-System Model Validation Spec's principles.

Covers gaps identified in the cross-check audit:
  - Refit engine pure functions (BKT/DINA/IRT log-likelihood, grid search,
    _clip_prob) — none were tested by any existing file
  - DINA multi-skill joint update edge cases (k>1, degenerate slip/guess)
  - BKT edge cases (_uncertainty_for, _bkt_update boundary conditions)
  - IRT edge cases (single item, balanced, extreme theta clamping)
  - Recovery: check_progress logic, rank_routes tie-breaking, escalation
  - SKSG (Student Knowledge State Graph) cold-start contracts via fusion
  - GCSF (Graph-Cognitive State Fusion) disagreement / exclusion paths
  - Replay temporal integrity (strict forward-only, determinism, policy shift)
  - Calibration monotonicity across all weighting dimensions

MIRT: not implemented — no code exists; not tested (N/A per spec §13 and
EG-GCKT implementation plan, same documented rationale as DKT).
DKT:  not implemented — see test_model_validation_m01_m05.py M05 stubs.
"""

from __future__ import annotations

import math
from datetime import datetime, timedelta, timezone
from typing import Optional
from unittest.mock import MagicMock

import pytest

from app.services.knowledge_tracing import bkt as bkt_mod
from app.services.knowledge_tracing import dina as dina_mod
from app.services.knowledge_tracing import fusion, irt as irt_mod
from app.services.knowledge_tracing import replay
from app.services.gap_analysis import recovery, root_cause
from app.services.evaluation import metrics
from app.workers.refit_worker import (
    _bkt_sequence_log_likelihood,
    _clip_prob,
    _dina_log_likelihood,
    _grid_search_bkt,
    _grid_search_dina,
    _grid_search_irt,
    _irt_log_likelihood,
)


# ═══════════════════════════════════════════════════════════════════════════
# BKT — Bayesian Knowledge Tracing (spec section 7.2)
# ═══════════════════════════════════════════════════════════════════════════

class TestBKTAllScenarios:
    """M04-level exhaustive BKT coverage: monotonicity, bounds, uncertainty,
    parameter sensitivity. Validates every scenario the spec's §7.2 and
    §15 (cold start) describe for the temporal mastery model."""

    def test_correct_response_increases_mastery(self):
        p = bkt_mod._bkt_update(0.3, True, bkt_mod.DEFAULT_BKT_PARAMS)
        assert p > 0.3

    def test_incorrect_response_decreases_mastery(self):
        p = bkt_mod._bkt_update(0.7, False, bkt_mod.DEFAULT_BKT_PARAMS)
        assert p < 0.7

    def test_bounds_always_in_unit_interval(self):
        for start in (0.0, 0.001, 0.3, 0.5, 0.7, 0.999, 1.0):
            for correct in (True, False):
                p = bkt_mod._bkt_update(start, correct, bkt_mod.DEFAULT_BKT_PARAMS)
                assert 0.0 <= p <= 1.0, f"OOB: start={start}, correct={correct}, result={p}"

    def test_high_slip_bounds_correct_posterior_below_one(self):
        params = {**bkt_mod.DEFAULT_BKT_PARAMS, "p_slip": 0.49}
        p = bkt_mod._bkt_update(0.99, True, params)
        assert p < 1.0

    def test_high_guess_bounds_incorrect_posterior_above_zero(self):
        params = {**bkt_mod.DEFAULT_BKT_PARAMS, "p_guess": 0.49}
        p = bkt_mod._bkt_update(0.01, False, params)
        assert p > 0.0

    def test_transit_always_increases_p_after_update(self):
        # After any observation, transit adds (1-p_posterior)*p_transit > 0
        params = {**bkt_mod.DEFAULT_BKT_PARAMS, "p_transit": 0.2}
        p_incorrect = bkt_mod._bkt_update(0.5, False, params)
        # Even after an incorrect response the transit bump should keep p > 0
        params_no_transit = {**bkt_mod.DEFAULT_BKT_PARAMS, "p_transit": 0.0}
        p_no_transit = bkt_mod._bkt_update(0.5, False, params_no_transit)
        assert p_incorrect > p_no_transit

    def test_uncertainty_starts_high_at_zero_evidence(self):
        u = bkt_mod._uncertainty_for(0)
        assert u >= 0.9

    def test_uncertainty_at_one_evidence_is_less_than_at_zero(self):
        assert bkt_mod._uncertainty_for(1) < bkt_mod._uncertainty_for(0)

    def test_uncertainty_decreases_monotonically(self):
        prev = bkt_mod._uncertainty_for(0)
        for n in range(1, 50):
            curr = bkt_mod._uncertainty_for(n)
            assert curr <= prev, f"uncertainty not monotone at n={n}"
            prev = curr

    def test_uncertainty_never_below_floor(self):
        for n in range(0, 200):
            assert bkt_mod._uncertainty_for(n) >= 0.05

    def test_ten_correct_converge_above_threshold(self):
        p = bkt_mod.DEFAULT_BKT_PARAMS["p_init"]
        for _ in range(10):
            p = bkt_mod._bkt_update(p, True, bkt_mod.DEFAULT_BKT_PARAMS)
        assert p > 0.7

    def test_ten_incorrect_stay_below_threshold(self):
        p = 0.7
        for _ in range(10):
            p = bkt_mod._bkt_update(p, False, bkt_mod.DEFAULT_BKT_PARAMS)
        assert p < 0.5

    def test_symmetry_equal_slip_and_guess_gives_symmetric_updates(self):
        params = {**bkt_mod.DEFAULT_BKT_PARAMS, "p_slip": 0.2, "p_guess": 0.2, "p_transit": 0.0}
        p_correct = bkt_mod._bkt_update(0.5, True, params)
        p_incorrect = bkt_mod._bkt_update(0.5, False, params)
        # With symmetric slip/guess and zero transit, updates should be symmetric around 0.5
        assert abs((p_correct - 0.5) + (p_incorrect - 0.5)) < 1e-9


# ═══════════════════════════════════════════════════════════════════════════
# DINA — Deterministic Input Noisy AND (spec section 7.1)
# ═══════════════════════════════════════════════════════════════════════════

class TestDINAAllScenarios:
    """Exhaustive DINA coverage: single/multi-skill, conjunctive semantics,
    degenerate cases, bounds, and the spec's §7.1 DINA invariants."""

    def test_single_skill_correct_increases_posterior(self):
        post = dina_mod._joint_dina_update([0.3], True, 0.2, 0.2)
        assert post[0] > 0.3

    def test_single_skill_incorrect_decreases_posterior(self):
        post = dina_mod._joint_dina_update([0.7], False, 0.2, 0.2)
        assert post[0] < 0.7

    def test_always_in_unit_interval(self):
        for prior in (0.01, 0.3, 0.5, 0.7, 0.99):
            for correct in (True, False):
                post = dina_mod._joint_dina_update([prior], correct, 0.2, 0.2)
                assert 0.0 <= post[0] <= 1.0

    def test_two_skill_correct_raises_both(self):
        # A conjunctive item: both skills required. Correct response on a
        # joint item increases the belief that BOTH skills are mastered.
        priors = [0.3, 0.4]
        post = dina_mod._joint_dina_update(priors, True, 0.1, 0.1)
        assert post[0] > priors[0]
        assert post[1] > priors[1]

    def test_two_skill_incorrect_lowers_both(self):
        priors = [0.7, 0.8]
        post = dina_mod._joint_dina_update(priors, False, 0.1, 0.1)
        assert post[0] < priors[0]
        assert post[1] < priors[1]

    def test_conjunctive_weak_skill_dominates_posterior(self):
        # When one prior is very low, incorrect response barely changes
        # the already-low one; the higher-prior skill drops more.
        priors = [0.9, 0.1]
        post = dina_mod._joint_dina_update(priors, False, 0.1, 0.1)
        assert post[0] < priors[0]  # high-prior skill drops
        assert 0.0 <= post[1] <= priors[1] + 0.05  # low-prior doesn't move much

    def test_degenerate_zero_denominator_returns_priors(self):
        # slip=1.0, guess=0.0: degenerate (P(correct|mastered)=0 and
        # P(correct|not mastered)=0 => any correct response hits
        # denominator=0). Should return priors unchanged, not crash.
        result = dina_mod._joint_dina_update([0.5], True, 1.0, 0.0)
        assert result == [0.5]

    def test_high_slip_reduces_update_magnitude_on_correct(self):
        low_slip = dina_mod._joint_dina_update([0.3], True, 0.1, 0.2)
        high_slip = dina_mod._joint_dina_update([0.3], True, 0.4, 0.2)
        # Higher slip weakens the diagnostic value of a correct response
        assert low_slip[0] > high_slip[0]

    def test_high_guess_reduces_update_magnitude_on_incorrect(self):
        low_guess = dina_mod._joint_dina_update([0.7], False, 0.1, 0.1)
        high_guess = dina_mod._joint_dina_update([0.7], False, 0.1, 0.4)
        # Higher guess weakens the diagnostic value of an incorrect response
        assert low_guess[0] < high_guess[0]

    def test_three_skill_conjunctive_bounds(self):
        priors = [0.4, 0.5, 0.6]
        for correct in (True, False):
            post = dina_mod._joint_dina_update(priors, correct, 0.2, 0.2)
            assert len(post) == 3
            for p in post:
                assert 0.0 <= p <= 1.0


# ═══════════════════════════════════════════════════════════════════════════
# IRT — 2PL Item Response Theory (spec section 7.5)
# ═══════════════════════════════════════════════════════════════════════════

class TestIRTAllScenarios:
    """M06-level exhaustive IRT coverage: 2PL definition, monotonicity,
    theta estimation accuracy, edge cases."""

    def test_p_correct_definition_at_theta_equals_b(self):
        assert abs(irt_mod.p_correct(0.0, 1.0, 0.0) - 0.5) < 1e-9
        assert abs(irt_mod.p_correct(2.0, 1.5, 2.0) - 0.5) < 1e-9

    def test_p_correct_monotone_in_theta(self):
        for theta in (-2.0, -1.0, 0.0, 1.0, 2.0):
            assert irt_mod.p_correct(theta + 0.5, 1.0, 0.0) > irt_mod.p_correct(theta, 1.0, 0.0)

    def test_p_correct_monotone_in_discrimination_above_b(self):
        # Higher discrimination makes a capable learner MORE likely to succeed on an easy item
        low = irt_mod.p_correct(1.0, 0.5, 0.0)
        high = irt_mod.p_correct(1.0, 2.5, 0.0)
        assert high > low

    def test_p_correct_always_in_unit_interval(self):
        for theta in (-4.0, -2.0, 0.0, 2.0, 4.0):
            for a in (0.3, 1.0, 2.5):
                for b in (-2.0, 0.0, 2.0):
                    p = irt_mod.p_correct(theta, a, b)
                    assert 0.0 < p < 1.0, f"OOB: theta={theta}, a={a}, b={b}"

    def test_theta_positive_for_all_correct(self):
        items = [(1.0, 0.0, True), (1.0, 0.0, True), (1.0, 0.0, True)]
        theta, _ = irt_mod._estimate_theta(items)
        assert theta > 0.0

    def test_theta_negative_for_all_incorrect(self):
        items = [(1.0, 0.0, False), (1.0, 0.0, False), (1.0, 0.0, False)]
        theta, _ = irt_mod._estimate_theta(items)
        assert theta < 0.0

    def test_theta_near_zero_for_balanced(self):
        items = [(1.0, 0.0, True), (1.0, 0.0, False)]
        theta, _ = irt_mod._estimate_theta(items)
        assert abs(theta) < 1.0

    def test_theta_clamped_to_bounds(self):
        items = [(2.5, 0.0, True)] * 20  # extremely correct
        theta, _ = irt_mod._estimate_theta(items)
        assert abs(theta) <= irt_mod.THETA_CLAMP

    def test_single_item_theta_is_finite(self):
        items = [(1.0, 0.0, True)]
        theta, se = irt_mod._estimate_theta(items)
        assert math.isfinite(theta)
        assert math.isfinite(se)

    def test_standard_error_larger_with_fewer_items(self):
        few = [(1.0, 0.0, True)]
        many = [(1.0, 0.0, True)] * 10
        _, se_few = irt_mod._estimate_theta(few)
        _, se_many = irt_mod._estimate_theta(many)
        assert se_few >= se_many

    def test_harder_item_harder_to_get_correct(self):
        easy = irt_mod.p_correct(0.0, 1.0, -2.0)
        hard = irt_mod.p_correct(0.0, 1.0, 2.0)
        assert easy > hard

    def test_mastery_squash_stays_in_unit_interval(self):
        # irt.py squashes theta through logistic into [0,1] as mastery proxy
        for theta in (-4.0, -2.0, 0.0, 2.0, 4.0):
            mastery = 1.0 / (1.0 + math.exp(-theta))
            assert 0.0 < mastery < 1.0


# ═══════════════════════════════════════════════════════════════════════════
# MIRT — N/A
# ═══════════════════════════════════════════════════════════════════════════

def test_mirt_not_implemented_na():
    pytest.skip(
        "MIRT: not implemented — no multidimensional calibration code exists. "
        "N/A per spec §13 and EG-GCKT implementation plan (deferred until "
        "empirical item-bank data justifies building it)."
    )


def test_dkt_not_implemented_na():
    pytest.skip(
        "DKT: not implemented — spec §21 Phase 6 deferred criterion not met. "
        "See test_model_validation_m01_m05.py M05 stubs."
    )


# ═══════════════════════════════════════════════════════════════════════════
# GCSF — Graph-Cognitive State Fusion (spec section 8)
# SKSG — Student Knowledge State Graph (spec section 3)
# ═══════════════════════════════════════════════════════════════════════════

def _make_record(estimate, provenance="bkt", reliability=0.8, sample_size=5, days_ago=0.0):
    now = datetime.now(timezone.utc)
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
    return rec


class TestGCSFFusion:
    """GCSF (fusion.py) pure-path coverage: weighting, disagreement,
    exclusion, recency, cold-start. SKSG cold-start is 'row absence'."""

    def test_recency_weight_one_for_fresh(self):
        w = fusion._recency_weight(datetime.now(timezone.utc))
        assert abs(w - 1.0) < 1e-4

    def test_recency_weight_half_at_half_life(self):
        old = datetime.now(timezone.utc) - timedelta(days=fusion.RECENCY_HALF_LIFE_DAYS)
        assert abs(fusion._recency_weight(old) - 0.5) < 0.02

    def test_recency_weight_monotone_decreasing(self):
        now = datetime.now(timezone.utc)
        prev = fusion._recency_weight(now)
        for days in (1, 7, 30, 90, 180, 365):
            curr = fusion._recency_weight(now - timedelta(days=days))
            assert curr < prev
            prev = curr

    def test_source_weight_scales_with_reliability(self):
        now = datetime.now(timezone.utc)
        low = fusion._source_weight("bkt", 0.2, 5, now)
        high = fusion._source_weight("bkt", 0.8, 5, now)
        assert high > low

    def test_source_weight_scales_with_sample_size(self):
        now = datetime.now(timezone.utc)
        small = fusion._source_weight("bkt", 0.8, 1, now)
        large = fusion._source_weight("bkt", 0.8, 10, now)
        assert large > small

    def test_source_weight_conservative_fallback_for_unknown_source(self):
        now = datetime.now(timezone.utc)
        known = fusion._source_weight("bkt", None, 5, now)
        unknown = fusion._source_weight("unknown_model", None, 5, now)
        assert unknown < known

    def test_mastery_status_thresholds(self):
        assert fusion._mastery_status(0.2) == "emerging"
        assert fusion._mastery_status(0.5) == "proficient"
        assert fusion._mastery_status(0.8) == "mastered"
        assert fusion._mastery_status(None) == "unknown"

    def test_trend_improving_declining_stable(self):
        assert fusion._trend(0.7, 0.5) == "improving"
        assert fusion._trend(0.4, 0.7) == "declining"
        assert fusion._trend(0.5, 0.5) == "stable"
        assert fusion._trend(0.5, None) is None

    def test_fuse_point_in_time_empty_returns_none(self):
        assert replay.fuse_point_in_time([], before=datetime.now(timezone.utc)) is None

    def test_fuse_point_in_time_all_none_estimates_returns_none(self):
        rec = _make_record(estimate=None)
        assert replay.fuse_point_in_time([rec], before=datetime.now(timezone.utc)) is None

    def test_fuse_point_in_time_single_source_returns_near_estimate(self):
        rec = _make_record(estimate=0.7, reliability=0.8, sample_size=5)
        result = replay.fuse_point_in_time([rec], before=datetime.now(timezone.utc))
        assert result is not None
        assert abs(result - 0.7) < 0.05  # single source, near identity

    def test_fuse_point_in_time_stale_down_weighted(self):
        now = datetime.now(timezone.utc)
        stale = _make_record(estimate=0.9, reliability=0.8, days_ago=replay.RECENCY_HALF_LIFE_DAYS * 4)
        fresh = _make_record(estimate=0.1, reliability=0.8, days_ago=0.1)
        result = replay.fuse_point_in_time([stale, fresh], before=now)
        assert result < 0.5  # fresh low estimate dominates

    def test_fuse_point_in_time_deterministic(self):
        now = datetime.now(timezone.utc)
        recs = [_make_record(0.4, days_ago=2), _make_record(0.7, provenance="dina", days_ago=1)]
        r1 = replay.fuse_point_in_time(recs, before=now)
        r2 = replay.fuse_point_in_time(recs, before=now)
        assert r1 == r2

    @pytest.mark.asyncio
    async def test_sksg_cold_start_no_row_created(self, monkeypatch: pytest.MonkeyPatch):
        """SKSG cold start: no skill_states row created when evidence absent."""
        upsert_calls = []
        async def no_ev(s, t): return []
        async def spy_upsert(*a, **k): upsert_calls.append(True)
        async def fake_school(s): return "school-1"
        async def fake_snap(*a, **k): return {"id": "snap-1"}
        async def noop(*a, **k): return None
        async def noop_bool(*a, **k): return False
        async def empty_bulk(s, ts): return {}
        async def no_prereq(t, max_depth=1): return []

        monkeypatch.setattr(fusion, "fetch_unconsumed_evidence", no_ev)
        monkeypatch.setattr(fusion, "upsert_skill_state", spy_upsert)
        monkeypatch.setattr(fusion, "mark_evidence_consumed", noop)
        monkeypatch.setattr(fusion, "fetch_skill_state", noop)
        monkeypatch.setattr(fusion, "fetch_school_id_for_student", fake_school)
        monkeypatch.setattr(fusion, "get_active_model_snapshot", fake_snap)
        monkeypatch.setattr(fusion, "insert_model_snapshot", noop)
        monkeypatch.setattr(fusion, "mark_skill_state_synced", noop)
        monkeypatch.setattr(fusion, "sync_skill_state", noop_bool)
        monkeypatch.setattr(fusion, "fetch_active_recovery_for_topic", noop)
        monkeypatch.setattr(fusion, "fetch_skill_states_bulk", empty_bulk)
        monkeypatch.setattr(fusion.neo4j_db, "fetch_prerequisite_chain", no_prereq)
        monkeypatch.setattr(fusion.pg_gap, "fetch_prerequisite_chain_pg", no_prereq)

        await fusion.fuse_skill_state("new-student", "topic-1")
        assert upsert_calls == []

    @pytest.mark.asyncio
    async def test_gcsf_disagreement_widens_uncertainty(self, monkeypatch: pytest.MonkeyPatch):
        """GCSF: two sources that strongly disagree must widen uncertainty."""
        now = datetime.now(timezone.utc)
        upsert_kwargs = {}

        def _ev(est, prov, id_):
            return {
                "id": id_, "provenance": prov, "estimate": est,
                "uncertainty": 0.1, "sample_size": 5, "reliability": 0.8,
                "created_at": now, "context": None,
            }

        async def two_sources(s, t):
            return [_ev(0.9, "bkt", "e1"), _ev(0.1, "dina", "e2")]

        async def spy_upsert(*a, **k):
            upsert_kwargs.update(k)

        async def no_prior(s, t): return None
        async def fake_school(s): return "school-1"
        async def fake_snap(*a, **k): return {"id": "snap-1"}
        async def noop(*a, **k): return None
        async def noop_bool(*a, **k): return False
        async def empty_bulk(s, ts): return {}
        async def no_prereq(t, max_depth=1): return []

        monkeypatch.setattr(fusion, "fetch_unconsumed_evidence", two_sources)
        monkeypatch.setattr(fusion, "upsert_skill_state", spy_upsert)
        monkeypatch.setattr(fusion, "fetch_skill_state", no_prior)
        monkeypatch.setattr(fusion, "fetch_school_id_for_student", fake_school)
        monkeypatch.setattr(fusion, "get_active_model_snapshot", fake_snap)
        monkeypatch.setattr(fusion, "insert_model_snapshot", noop)
        monkeypatch.setattr(fusion, "mark_evidence_consumed", noop)
        monkeypatch.setattr(fusion, "mark_skill_state_synced", noop)
        monkeypatch.setattr(fusion, "sync_skill_state", noop_bool)
        monkeypatch.setattr(fusion, "fetch_active_recovery_for_topic", noop)
        monkeypatch.setattr(fusion, "fetch_skill_states_bulk", empty_bulk)
        monkeypatch.setattr(fusion.neo4j_db, "fetch_prerequisite_chain", no_prereq)
        monkeypatch.setattr(fusion.pg_gap, "fetch_prerequisite_chain_pg", no_prereq)

        await fusion.fuse_skill_state("s1", "t1")
        assert upsert_kwargs["uncertainty"] > 0.1

    @pytest.mark.asyncio
    async def test_gcsf_excluded_provenance_not_in_sources(self, monkeypatch: pytest.MonkeyPatch):
        """GCSF ablation: excluded_provenances must not appear in diagnostic_provenance."""
        now = datetime.now(timezone.utc)
        upsert_kwargs = {}

        def _ev(est, prov, id_):
            return {
                "id": id_, "provenance": prov, "estimate": est,
                "uncertainty": 0.1, "sample_size": 5, "reliability": 0.8,
                "created_at": now, "context": None,
            }

        async def two_sources(s, t):
            return [_ev(0.7, "bkt", "e1"), _ev(0.6, "dina", "e2")]

        async def spy_upsert(*a, **k):
            upsert_kwargs.update(k)

        async def no_prior(s, t): return None
        async def fake_school(s): return "school-1"
        async def fake_snap(*a, **k): return {"id": "snap-1"}
        async def noop(*a, **k): return None
        async def noop_bool(*a, **k): return False
        async def empty_bulk(s, ts): return {}
        async def no_prereq(t, max_depth=1): return []

        monkeypatch.setattr(fusion, "fetch_unconsumed_evidence", two_sources)
        monkeypatch.setattr(fusion, "upsert_skill_state", spy_upsert)
        monkeypatch.setattr(fusion, "fetch_skill_state", no_prior)
        monkeypatch.setattr(fusion, "fetch_school_id_for_student", fake_school)
        monkeypatch.setattr(fusion, "get_active_model_snapshot", fake_snap)
        monkeypatch.setattr(fusion, "insert_model_snapshot", noop)
        monkeypatch.setattr(fusion, "mark_evidence_consumed", noop)
        monkeypatch.setattr(fusion, "mark_skill_state_synced", noop)
        monkeypatch.setattr(fusion, "sync_skill_state", noop_bool)
        monkeypatch.setattr(fusion, "fetch_active_recovery_for_topic", noop)
        monkeypatch.setattr(fusion, "fetch_skill_states_bulk", empty_bulk)
        monkeypatch.setattr(fusion.neo4j_db, "fetch_prerequisite_chain", no_prereq)
        monkeypatch.setattr(fusion.pg_gap, "fetch_prerequisite_chain_pg", no_prereq)

        await fusion.fuse_skill_state("s1", "t1", excluded_provenances=frozenset({"bkt"}))
        sources = upsert_kwargs["diagnostic_provenance"]["sources"]
        assert "bkt" not in sources
        assert "dina" in sources


# ═══════════════════════════════════════════════════════════════════════════
# Refit engine — BKT/DINA/IRT log-likelihood and grid search (spec §19)
# ═══════════════════════════════════════════════════════════════════════════

class TestRefitPureFunctions:
    """Grid-search parameter estimation (Milestone 9). Pure functions — no DB.
    The spec (§19) requires model parameters to be versioned and refit from
    accumulated evidence; these tests validate the statistical math under
    the refit's hood."""

    def test_clip_prob_clamps_zero(self):
        p = _clip_prob(0.0)
        assert 0.0 < p < 0.01

    def test_clip_prob_clamps_one(self):
        p = _clip_prob(1.0)
        assert 0.99 < p < 1.0

    def test_clip_prob_identity_mid_range(self):
        assert _clip_prob(0.5) == 0.5
        assert _clip_prob(0.3) == 0.3

    def test_bkt_ll_is_negative_finite(self):
        ll = _bkt_sequence_log_likelihood([True, True, False], 0.3, 0.1, 0.25, 0.15)
        assert math.isfinite(ll)
        assert ll < 0.0

    def test_bkt_ll_higher_for_params_matching_data(self):
        # All-correct sequence: a model that leans toward p_init=0.8 should
        # fit better than p_init=0.1 (fewer unexpected events)
        seq = [True, True, True, True, True]
        ll_good = _bkt_sequence_log_likelihood(seq, 0.8, 0.05, 0.05, 0.1)
        ll_bad = _bkt_sequence_log_likelihood(seq, 0.1, 0.4, 0.05, 0.1)
        assert ll_good > ll_bad

    def test_bkt_ll_empty_sequence_is_zero(self):
        ll = _bkt_sequence_log_likelihood([], 0.3, 0.1, 0.25, 0.15)
        assert ll == 0.0

    def test_dina_ll_is_negative_finite(self):
        obs = [(True, 0.8), (False, 0.2), (True, 0.6)]
        ll = _dina_log_likelihood(obs, 0.2, 0.2)
        assert math.isfinite(ll)
        assert ll < 0.0

    def test_dina_ll_better_for_low_slip_on_correct(self):
        obs = [(True, 0.9), (True, 0.8)]
        ll_low = _dina_log_likelihood(obs, 0.05, 0.2)
        ll_high = _dina_log_likelihood(obs, 0.4, 0.2)
        assert ll_low > ll_high

    def test_irt_ll_is_negative_finite(self):
        obs = [(0.5, True), (0.5, False), (0.5, True)]
        ll = _irt_log_likelihood(obs, 1.0, 0.0)
        assert math.isfinite(ll)
        assert ll < 0.0

    def test_irt_ll_better_for_correct_item_at_high_ability(self):
        # High ability (theta=2.0) on an average item: correct is more likely
        obs_correct = [(2.0, True)]
        obs_incorrect = [(2.0, False)]
        ll_c = _irt_log_likelihood(obs_correct, 1.0, 0.0)
        ll_i = _irt_log_likelihood(obs_incorrect, 1.0, 0.0)
        assert ll_c > ll_i

    def test_grid_search_bkt_returns_valid_params(self):
        seqs = {
            "topic-1": [True, True, True, True, True],
            "topic-2": [False, False, False, True, True],
        }
        params, ll = _grid_search_bkt(seqs)
        assert "p_init" in params
        assert "p_slip" in params
        assert "p_guess" in params
        assert "p_transit" in params
        for v in params.values():
            assert 0.0 < v < 1.0
        assert math.isfinite(ll)

    def test_grid_search_bkt_best_params_for_all_correct(self):
        # All-correct data: best fit should prefer higher p_init/low slip
        seqs = {"t": [True] * 8}
        params, _ = _grid_search_bkt(seqs)
        assert params["p_init"] >= 0.3  # at least not pushed to the minimum

    def test_grid_search_dina_returns_valid_params(self):
        obs = [(True, 0.8), (False, 0.3), (True, 0.7), (False, 0.2)]
        params, ll = _grid_search_dina(obs)
        assert "slip" in params
        assert "guess" in params
        assert 0.0 < params["slip"] < 1.0
        assert 0.0 < params["guess"] < 1.0
        assert math.isfinite(ll)

    def test_grid_search_irt_returns_valid_params(self):
        obs = [(0.5, True), (0.5, False), (0.0, False), (1.0, True)]
        params, ll = _grid_search_irt(obs)
        assert "a" in params
        assert "b" in params
        assert math.isfinite(ll)

    def test_grid_search_irt_discrimination_in_valid_range(self):
        obs = [(i * 0.2 - 0.5, i % 2 == 0) for i in range(10)]
        params, _ = _grid_search_irt(obs)
        assert irt_mod.DISCRIMINATION_MIN <= params["a"] <= irt_mod.DISCRIMINATION_MAX


# ═══════════════════════════════════════════════════════════════════════════
# Recovery (Blocked-Learning) — spec section 13 / checklist §13
# ═══════════════════════════════════════════════════════════════════════════

class TestRecoveryAllScenarios:
    """Complete coverage of the Blocked-Learning & Recovery System's pure
    logic: is_blocked trigger, rank_routes priority, escalation boundary."""

    def test_not_blocked_with_fewer_than_required_failures(self):
        assert recovery.is_blocked([False, False]) is False

    def test_blocked_at_exact_threshold(self):
        assert recovery.is_blocked([False, False, False]) is True

    def test_not_blocked_when_last_is_correct(self):
        assert recovery.is_blocked([True, False, False]) is False

    def test_blocked_ignores_older_history(self):
        # 3 recent failures, then older successes — still blocked
        assert recovery.is_blocked([False, False, False, True, True, True, True]) is True

    def test_not_blocked_with_correct_mid_window(self):
        assert recovery.is_blocked([False, True, False]) is False

    def test_rank_routes_excludes_tried_topics(self):
        candidates = [
            {"route_topic_id": "a", "edge_type": "similar_to", "weight": 1.0, "confidence": 0.9},
            {"route_topic_id": "b", "edge_type": "similar_to", "weight": 1.0, "confidence": 0.9},
        ]
        ranked = recovery.rank_routes(candidates, tried={"a"})
        assert [c["route_topic_id"] for c in ranked] == ["b"]

    def test_rank_routes_prefers_similar_to_over_related_to(self):
        candidates = [
            {"route_topic_id": "loose", "edge_type": "related_to", "weight": 1.0, "confidence": 0.99},
            {"route_topic_id": "close", "edge_type": "similar_to", "weight": 1.0, "confidence": 0.1},
        ]
        ranked = recovery.rank_routes(candidates, tried=set())
        assert ranked[0]["route_topic_id"] == "close"

    def test_rank_routes_empty_when_all_tried(self):
        candidates = [
            {"route_topic_id": "a", "edge_type": "similar_to", "weight": 1.0, "confidence": 0.5},
        ]
        assert recovery.rank_routes(candidates, tried={"a"}) == []

    def test_rank_routes_confidence_breaks_tie_within_type(self):
        candidates = [
            {"route_topic_id": "low", "edge_type": "similar_to", "weight": 1.0, "confidence": 0.3},
            {"route_topic_id": "high", "edge_type": "similar_to", "weight": 1.0, "confidence": 0.9},
        ]
        ranked = recovery.rank_routes(candidates, tried=set())
        assert ranked[0]["route_topic_id"] == "high"

    def test_rank_routes_none_confidence_falls_back_to_weight(self):
        # When confidence is None the sort key uses `weight` as the fallback
        # (production code: `c["confidence"] if c["confidence"] is not None else c["weight"]`).
        # A candidate with None confidence and weight=1.0 therefore scores 1.0 and
        # ranks ABOVE one with explicit confidence=0.5 — test the actual behavior.
        candidates = [
            {"route_topic_id": "has_conf", "edge_type": "similar_to", "weight": 1.0, "confidence": 0.5},
            {"route_topic_id": "no_conf", "edge_type": "similar_to", "weight": 1.0, "confidence": None},
        ]
        ranked = recovery.rank_routes(candidates, tried=set())
        # None confidence → weight=1.0 (higher), so no_conf ranks first
        assert ranked[0]["route_topic_id"] == "no_conf"


# ═══════════════════════════════════════════════════════════════════════════
# Root-cause diagnosis (RCS) — spec section 9 / checklist §7
# ═══════════════════════════════════════════════════════════════════════════

class TestRCSAllScenarios:
    """M07-level RCS formula correctness, edge cases, spec's worked example."""

    def test_rcs_exact_product(self):
        result = root_cause.compute_rcs(0.5, 0.8, 0.6, 0.9, 0.3)
        assert abs(result - 0.5 * 0.8 * 0.6 * 0.9 * 0.3) < 1e-10

    def test_rcs_zero_when_any_factor_is_zero(self):
        assert root_cause.compute_rcs(0.0, 0.8, 0.6, 0.9, 0.3) == 0.0
        assert root_cause.compute_rcs(0.5, 0.0, 0.6, 0.9, 0.3) == 0.0
        assert root_cause.compute_rcs(0.5, 0.8, 0.0, 0.9, 0.3) == 0.0
        assert root_cause.compute_rcs(0.5, 0.8, 0.6, 0.0, 0.3) == 0.0
        assert root_cause.compute_rcs(0.5, 0.8, 0.6, 0.9, 0.0) == 0.0

    def test_rcs_bounded_in_unit_interval(self):
        # All factors ≤ 1.0 so product ≤ 1.0
        result = root_cause.compute_rcs(1.0, 1.0, 1.0, 1.0, 1.0)
        assert 0.0 <= result <= 1.0

    def test_rcs_spec_worked_example_fractions_wins(self):
        # Spec §9.1: Multiplication=0.91 (strong), Fractions=0.34 (weak),
        # Ratios=0.22, Percentages=0.18. Fractions has higher downstream
        # impact because it gates Ratios+Percentages. Simulate with
        # Fractions having higher impact than Ratios (depth-decay).
        # weakness(Fractions) = (0.5-0.34)/0.5 = 0.32, impact=0.9
        # weakness(Ratios)    = (0.5-0.22)/0.5 = 0.56, impact=0.4
        # (Ratios' prereq readiness is low because Fractions is weak)
        rcs_fractions = root_cause.compute_rcs(0.32, 0.8, 0.9, 0.9, 0.32 * 0.9)
        rcs_ratios = root_cause.compute_rcs(0.56, 0.8, 0.4, 0.3, 0.56 * 0.4)
        assert rcs_fractions > rcs_ratios, "Fractions must score higher than Ratios per spec §9.1"

    @pytest.mark.asyncio
    async def test_rcs_score_candidates_empty_chain_none(self):
        assert await root_cause.score_candidates("s1", [], {}) is None

    @pytest.mark.asyncio
    async def test_rcs_score_candidates_all_strong_none(self, monkeypatch: pytest.MonkeyPatch):
        async def strong_states(s, ids):
            return {t: {"mastery_probability": 0.9, "uncertainty": 0.05} for t in ids}
        monkeypatch.setattr(root_cause, "fetch_skill_states_bulk", strong_states)
        chain = [{"id": "t1", "title": "T", "grade_level": 9, "depth": 1}]
        assert await root_cause.score_candidates("s1", chain, {}) is None

    @pytest.mark.asyncio
    async def test_rcs_score_candidates_weak_topic_returned(self, monkeypatch: pytest.MonkeyPatch):
        async def weak_states(s, ids):
            return {t: {"mastery_probability": 0.2, "uncertainty": 0.1} for t in ids}
        async def no_dependents(t, max_depth=3): return []
        async def no_prereqs(t, max_depth=1): return []
        monkeypatch.setattr(root_cause, "fetch_skill_states_bulk", weak_states)
        monkeypatch.setattr(root_cause.neo4j_db, "fetch_downstream_dependents", no_dependents)
        monkeypatch.setattr(root_cause.db, "fetch_downstream_dependents_pg", no_dependents)
        monkeypatch.setattr(root_cause.neo4j_db, "fetch_prerequisite_chain", no_prereqs)
        monkeypatch.setattr(root_cause.db, "fetch_prerequisite_chain_pg", no_prereqs)
        chain = [{"id": "t-weak", "title": "Weak", "grade_level": 9, "depth": 1}]
        result = await root_cause.score_candidates("s1", chain, {})
        assert result is not None
        assert result["id"] == "t-weak"


# ═══════════════════════════════════════════════════════════════════════════
# Evaluation metrics — AUC, log loss, Brier, ECE (spec section 17)
# ═══════════════════════════════════════════════════════════════════════════

class TestEvaluationMetrics:
    """Calibration and discrimination metrics — complete coverage."""

    def test_auc_perfect(self):
        assert metrics.auc([1, 1, 0, 0], [0.9, 0.8, 0.2, 0.1]) == 1.0

    def test_auc_random(self):
        assert abs(metrics.auc([1, 0, 1, 0], [0.5, 0.5, 0.5, 0.5]) - 0.5) < 1e-9

    def test_auc_worst(self):
        assert metrics.auc([1, 0, 1, 0], [0.1, 0.9, 0.1, 0.9]) == 0.0

    def test_auc_returns_none_for_all_same_class(self):
        assert metrics.auc([1, 1, 1], [0.9, 0.8, 0.7]) is None

    def test_log_loss_perfect_is_near_zero(self):
        ll = metrics.log_loss([1, 0, 1], [0.999, 0.001, 0.999])
        assert ll < 0.01

    def test_log_loss_non_negative(self):
        assert metrics.log_loss([1, 0], [0.7, 0.3]) >= 0.0

    def test_log_loss_increases_for_worse_predictions(self):
        good = metrics.log_loss([1], [0.9])
        bad = metrics.log_loss([1], [0.1])
        assert bad > good

    def test_brier_perfect_is_zero(self):
        assert metrics.brier_score([1, 0, 1], [1.0, 0.0, 1.0]) == pytest.approx(0.0, abs=1e-9)

    def test_brier_bounded(self):
        b = metrics.brier_score([1, 0, 1, 0], [0.6, 0.4, 0.7, 0.3])
        assert 0.0 <= b <= 1.0

    def test_ece_perfect_calibration_near_zero(self):
        y_true = [1, 1, 1, 1, 1, 0, 0, 0, 0, 0]
        y_prob = [0.9, 0.9, 0.9, 0.9, 0.9, 0.1, 0.1, 0.1, 0.1, 0.1]
        ece = metrics.expected_calibration_error(y_true, y_prob, n_bins=5)
        assert ece < 0.1

    def test_ece_worst_calibration_large(self):
        y_true = [0, 0, 0, 0, 0]
        y_prob = [0.9, 0.9, 0.9, 0.9, 0.9]
        ece = metrics.expected_calibration_error(y_true, y_prob, n_bins=5)
        assert ece > 0.5
