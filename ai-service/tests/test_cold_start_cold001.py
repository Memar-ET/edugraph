"""
COLD-001: End-to-End Cold Start Test
From EduGraph_QA_Test_Case_and_Failure_Injection_Specification.docx.

A brand-new learner with zero prior evidence should exhibit these
properties throughout the entire GCKT pipeline:

  1. fuse_skill_state is a no-op: no students.skill_states row is
     created (cold start via row absence, never a fabricated 0.0).
  2. check_topic returns 0: consistency cannot flag anything without
     sufficient evidence on either side.
  3. downstream_impact / _prerequisite_readiness gracefully handle a
     learner who has no skill_states rows at all.
  4. produce_graph_evidence returns None when no prerequisite mastery
     exists to infer from -- no synthetic history is fabricated.
  5. take_snapshot returns None: no snapshot is created when mastery is
     NULL (cold-start snapshot suppression).
  6. reconstruct_mastery_as_of returns None for any historical timestamp:
     no fabricated historical mastery.

COLD-001 acceptance criterion (spec section 15 + Design Principles):
  "New learner → population priors + high uncertainty, never a fabricated
   point estimate. The system says 'unknown' rather than manufacture
   confidence."
"""

from __future__ import annotations

import pytest

from app.services.gap_analysis import consistency, root_cause
from app.services.knowledge_tracing import fusion, replay, snapshot


# ── COLD-001: Fusion layer ─────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_cold001_new_learner_fusion_is_noop(monkeypatch: pytest.MonkeyPatch) -> None:
    """fuse_skill_state must write nothing when no evidence exists for the
    (student, topic) pair -- cold start via row absence, never a fabricated
    skill_states row."""
    upsert_calls: list = []

    async def no_evidence(student_id, topic_id):
        return []

    async def spy_upsert(*args, **kwargs):
        upsert_calls.append(True)

    monkeypatch.setattr(fusion, "fetch_unconsumed_evidence", no_evidence)
    monkeypatch.setattr(fusion, "upsert_skill_state", spy_upsert)

    await fusion.fuse_skill_state("new-student", "topic-1")

    assert upsert_calls == [], "no skill_states row must be written for a learner with zero evidence"


# ── COLD-001: Consistency layer ────────────────────────────────────────────


@pytest.mark.asyncio
async def test_cold001_consistency_check_on_new_learner_returns_zero(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """check_topic on a topic whose student has no skill_states rows at all
    must return 0 -- there is nothing to be inconsistent about."""

    async def empty_skill_states(student_id, topic_ids):
        return {}  # no rows -- true cold start

    async def fake_prereq_chain(topic_id, max_depth=1):
        return [{"id": "topic-prereq", "title": "Prereq", "depth": 1}]

    monkeypatch.setattr(consistency, "fetch_skill_states_bulk", empty_skill_states)
    monkeypatch.setattr(consistency.neo4j_db, "fetch_prerequisite_chain", fake_prereq_chain)
    monkeypatch.setattr(consistency.db, "fetch_prerequisite_chain_pg", fake_prereq_chain)

    flagged = await consistency.check_topic("new-student", "topic-C")
    assert flagged == 0, "consistency check on a new learner must return 0 (nothing evidenced)"


@pytest.mark.asyncio
async def test_cold001_consistency_check_when_dependent_has_no_state_returns_zero(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """When the topic under check has no skill_states row, check_topic must
    short-circuit at the dependent-state lookup and return 0 -- not try
    to fetch prerequisites or fabricate a conclusion."""
    prereq_fetches: list = []

    async def no_state(student_id, topic_ids):
        return {}

    async def spy_prereq(topic_id, max_depth=1):
        prereq_fetches.append(topic_id)
        return []

    monkeypatch.setattr(consistency, "fetch_skill_states_bulk", no_state)
    monkeypatch.setattr(consistency.neo4j_db, "fetch_prerequisite_chain", spy_prereq)
    monkeypatch.setattr(consistency.db, "fetch_prerequisite_chain_pg", spy_prereq)

    flagged = await consistency.check_topic("new-student", "topic-C")
    assert flagged == 0
    # Should short-circuit before even fetching prerequisites
    assert prereq_fetches == [], "no prerequisite lookup should occur when the dependent has no state"


# ── COLD-001: Root-cause layer ─────────────────────────────────────────────


@pytest.mark.asyncio
async def test_cold001_downstream_impact_with_empty_graph_returns_zero(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """downstream_impact for a topic with no downstream dependents in either
    store returns 0.0 -- no fabricated impact score."""

    async def no_dependents(topic_id, max_depth=3):
        return []

    monkeypatch.setattr(root_cause.neo4j_db, "fetch_downstream_dependents", no_dependents)
    monkeypatch.setattr(root_cause.db, "fetch_downstream_dependents_pg", no_dependents)

    impact = await root_cause.downstream_impact("topic-leaf")
    assert impact == 0.0


@pytest.mark.asyncio
async def test_cold001_prerequisite_readiness_with_no_prereq_returns_one(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """_prerequisite_readiness for a foundational topic (no prerequisites)
    returns 1.0 -- it is fully 'ready' because there's nothing upstream
    gating it. Not fabricated confidence: no prerequisites = no constraint."""

    async def no_prereqs(topic_id, max_depth=1):
        return []

    monkeypatch.setattr(root_cause.neo4j_db, "fetch_prerequisite_chain", no_prereqs)
    monkeypatch.setattr(root_cause.db, "fetch_prerequisite_chain_pg", no_prereqs)

    readiness = await root_cause._prerequisite_readiness("new-student", "topic-foundational", {})
    assert readiness == 1.0, "topic with no prerequisites must have readiness=1.0"


@pytest.mark.asyncio
async def test_cold001_produce_graph_evidence_without_prereq_mastery_returns_none(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """produce_graph_evidence cannot infer anything if none of the
    prerequisite topics have mastery estimates yet -- it must return None,
    not insert a fabricated evidence row."""
    insert_calls: list = []

    async def prereqs_exist(topic_id, max_depth=1):
        # Prerequisites exist in the graph
        return [{"id": "topic-prereq", "title": "Prereq", "depth": 1}]

    async def no_mastery(student_id, topic_ids):
        return {}  # No skill_states rows for any of them

    async def spy_insert(*args, **kwargs):
        insert_calls.append(True)
        return "ev-fake"

    monkeypatch.setattr(root_cause.neo4j_db, "fetch_prerequisite_chain", prereqs_exist)
    monkeypatch.setattr(root_cause.db, "fetch_prerequisite_chain_pg", prereqs_exist)
    monkeypatch.setattr(root_cause, "fetch_skill_states_bulk", no_mastery)
    monkeypatch.setattr(root_cause, "insert_evidence", spy_insert)

    result = await root_cause.produce_graph_evidence("new-student", "topic-1")
    assert result is None, "no prerequisite mastery data → produce_graph_evidence must return None"
    assert insert_calls == [], "no evidence row must be inserted when there's nothing to infer from"


@pytest.mark.asyncio
async def test_cold001_score_candidates_with_no_chain_returns_none(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """score_candidates given an empty prerequisite chain returns None --
    no root cause can be identified without a chain to walk."""
    result = await root_cause.score_candidates("new-student", [], {})
    assert result is None


@pytest.mark.asyncio
async def test_cold001_score_candidates_with_no_weak_evidence_returns_none(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """score_candidates given a chain where every topic is either above
    WEAK_THRESHOLD or has no evidence must return None -- nothing is a
    credible root cause."""
    chain = [{"id": "topic-A", "title": "A", "grade_level": 9, "depth": 1}]

    async def no_states(student_id, topic_ids):
        return {}  # no evidence for any topic in the chain

    monkeypatch.setattr(root_cause, "fetch_skill_states_bulk", no_states)

    result = await root_cause.score_candidates("new-student", chain, {})
    assert result is None, "no evidence → no credible root cause → must return None"


# ── COLD-001: Snapshot layer ───────────────────────────────────────────────


@pytest.mark.asyncio
async def test_cold001_snapshot_of_new_learner_is_suppressed(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """take_snapshot on a (student, topic) pair with no skill_states row
    must return None -- cold-start snapshots are never fabricated."""
    from app.services.knowledge_tracing import snapshot

    async def no_state(student_id, topic_id):
        return None  # no skill_states row

    monkeypatch.setattr(snapshot, "fetch_skill_state_full", no_state)

    result = await snapshot.take_snapshot("new-student", "topic-1")
    assert result is None, "snapshot must return None when no skill_states row exists (cold start)"


@pytest.mark.asyncio
async def test_cold001_snapshot_of_learner_with_null_mastery_is_suppressed(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """take_snapshot on a row that exists but has mastery_probability=NULL
    must also return None -- NULL mastery is the 'cold start via row absence'
    invariant; no snapshot of an incomplete state is written."""
    from app.services.knowledge_tracing import snapshot

    async def null_mastery_state(student_id, topic_id):
        return {"mastery_probability": None, "school_id": "school-1", "mastery_status": "unknown",
                "uncertainty": None, "evidence_count": 0, "trend": None,
                "model_snapshot_id": None, "computed_at": None}

    monkeypatch.setattr(snapshot, "fetch_skill_state_full", null_mastery_state)

    result = await snapshot.take_snapshot("new-student", "topic-1")
    assert result is None


# ── COLD-001: Replay layer ─────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_cold001_reconstruct_mastery_for_new_learner_returns_none(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """For a learner who has never had any evidence, reconstruct_mastery_as_of
    at any timestamp must return None -- no historical mastery is fabricated."""

    async def no_evidence(student_id, topic_id, before, provenance=None):
        return []

    monkeypatch.setattr(replay, "prior_evidence", no_evidence)

    from datetime import datetime, timezone
    result = await replay.reconstruct_mastery_as_of("new-student", "topic-1")
    assert result is None, "cold-start learner → no historical mastery → None at every timestamp"


def test_cold001_fuse_point_in_time_with_empty_evidence_returns_none() -> None:
    """The pure function: empty evidence list → None, not 0.0."""
    from datetime import datetime, timezone
    result = replay.fuse_point_in_time([], before=datetime.now(timezone.utc))
    assert result is None
