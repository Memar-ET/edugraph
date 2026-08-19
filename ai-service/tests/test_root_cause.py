"""Tests for app/services/gap_analysis/root_cause.py -- checklist
requirement "unit-test root-cause scoring." The integration-style test
below reproduces the spec's own worked example (section 9.1) end to end
through score_candidates(), with the DB-backed helpers monkeypatched so
it runs without Postgres/Neo4j.
"""

from __future__ import annotations

import pytest

from app.services.gap_analysis import root_cause


def test_compute_rcs_is_the_product_of_all_five_factors() -> None:
    assert root_cause.compute_rcs(0.4, 0.8, 0.3, 0.9, 0.13) == pytest.approx(0.4 * 0.8 * 0.3 * 0.9 * 0.13)


def test_compute_rcs_is_zero_when_any_factor_is_zero() -> None:
    assert root_cause.compute_rcs(0.0, 0.8, 0.3, 0.9, 0.13) == 0.0
    assert root_cause.compute_rcs(0.4, 0.8, 0.0, 0.9, 0.13) == 0.0  # no downstream impact -> no RCS


@pytest.mark.asyncio
async def test_score_candidates_returns_none_for_empty_chain() -> None:
    assert await root_cause.score_candidates("student-1", [], {}) is None


@pytest.mark.asyncio
async def test_score_candidates_returns_none_when_nothing_in_chain_is_weak(monkeypatch: pytest.MonkeyPatch) -> None:
    async def fake_states(student_id, topic_ids):
        return {tid: {"mastery_probability": 0.95, "uncertainty": 0.05} for tid in topic_ids}

    monkeypatch.setattr(root_cause, "fetch_skill_states_bulk", fake_states)

    chain = [{"id": "M", "title": "Multiplication", "grade_level": 7, "depth": 1}]
    assert await root_cause.score_candidates("student-1", chain, {}) is None


@pytest.mark.asyncio
async def test_score_candidates_reproduces_the_spec_fractions_worked_example(monkeypatch: pytest.MonkeyPatch) -> None:
    # Spec section 9.1: Multiplication=0.91, Fractions=0.34, Ratios=0.22,
    # Percentages=0.18, chain Multiplication -> Fractions -> Ratios ->
    # Percentages. EG-GCKT must pick Fractions as the root cause, not the
    # numerically weaker Ratios/Percentages -- because Ratios/Percentages'
    # own prerequisite readiness is gated by Fractions itself still being
    # weak, while Fractions' prerequisite (Multiplication) is genuinely
    # ready and Fractions sits upstream of two weaknesses.
    mastery = {
        "multiplication": 0.91,
        "fractions": 0.34,
        "ratios": 0.22,
        "percentages": 0.18,
    }

    async def fake_states(student_id, topic_ids):
        return {tid: {"mastery_probability": mastery[tid], "uncertainty": 0.1} for tid in topic_ids if tid in mastery}

    async def fake_downstream_dependents(topic_id, max_depth=3):
        # Percentages depends on Ratios depends on Fractions depends on Multiplication.
        graph = {
            "multiplication": [{"id": "fractions", "depth": 1}, {"id": "ratios", "depth": 2}, {"id": "percentages", "depth": 3}],
            "fractions": [{"id": "ratios", "depth": 1}, {"id": "percentages", "depth": 2}],
            "ratios": [{"id": "percentages", "depth": 1}],
            "percentages": [],
        }
        return graph.get(topic_id, [])

    async def fake_prerequisite_chain(topic_id, max_depth=3):
        chain_map = {
            "percentages": [{"id": "ratios", "title": "Ratios", "grade_level": 9, "depth": 1}],
            "ratios": [{"id": "fractions", "title": "Fractions", "grade_level": 8, "depth": 1}],
            "fractions": [{"id": "multiplication", "title": "Multiplication", "grade_level": 7, "depth": 1}],
            "multiplication": [],
        }
        return chain_map.get(topic_id, [])

    monkeypatch.setattr(root_cause, "fetch_skill_states_bulk", fake_states)
    monkeypatch.setattr(root_cause.neo4j_db, "fetch_downstream_dependents", fake_downstream_dependents)
    monkeypatch.setattr(root_cause.neo4j_db, "fetch_prerequisite_chain", fake_prerequisite_chain)

    # Percentages is the symptom; its ancestor chain is Ratios -> Fractions -> Multiplication.
    chain = [
        {"id": "ratios", "title": "Ratios", "grade_level": 9, "depth": 1},
        {"id": "fractions", "title": "Fractions", "grade_level": 8, "depth": 2},
        {"id": "multiplication", "title": "Multiplication", "grade_level": 7, "depth": 3},
    ]

    result = await root_cause.score_candidates("student-1", chain, {})

    assert result is not None
    assert result["id"] == "fractions", (
        f"expected Fractions to win as the root cause, got {result['id']!r} with factors {result.get('factors')}"
    )
    # The path from symptom to root cause must include everything at or
    # above Fractions' depth (Ratios, depth 1, then Fractions, depth 2) --
    # not the whole chain (Multiplication, depth 3, is beyond the winner).
    assert [p["id"] for p in result["path"]] == ["ratios", "fractions"]
