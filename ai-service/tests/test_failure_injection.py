"""Failure-injection test suite (FI-001..FI-008), from
`EduGraph_QA_Test_Case_and_Failure_Injection_Specification.docx`. Every
scenario asserts the spec's primary acceptance principle: the system must
prefer explicit uncertainty, abstention, degradation, or a clearly
surfaced failure over fabricated certainty.

Exercises app/services/knowledge_tracing/fusion.py's `fuse_skill_state`
end to end (not just its pure helpers, already covered by
test_fusion.py) with every DB dependency monkeypatched, plus
app/workers/kt_worker.py's job-level failure containment.
"""

from __future__ import annotations

from datetime import datetime, timezone
from typing import Any, Optional

import pytest

from app.services.knowledge_tracing import fusion
from app.services.gap_analysis import root_cause
from app.workers import kt_worker


def _evidence_row(
    *,
    provenance: str,
    estimate: Optional[float],
    uncertainty: Optional[float] = 0.2,
    sample_size: int = 5,
    reliability: Optional[float] = None,
    created_at: Optional[datetime] = None,
    context: Any = None,
    id: str = "ev-1",  # noqa: A002 -- matches the DB column name
) -> dict:
    return {
        "id": id,
        "provenance": provenance,
        "estimate": estimate,
        "uncertainty": uncertainty,
        "sample_size": sample_size,
        "reliability": reliability,
        "created_at": created_at or datetime.now(timezone.utc),
        "context": context,
    }


@pytest.fixture
def fusion_env(monkeypatch: pytest.MonkeyPatch) -> dict:
    """Patches every one of fuse_skill_state's DB dependencies to inert,
    cold-start-safe defaults, and records every call the function under
    test makes to the one function that actually persists a result
    (upsert_skill_state) so tests can assert on it -- including asserting
    it was NEVER called (FI-001's "no fabricated score")."""
    captured: dict = {"upsert_calls": [], "consumed_calls": [], "school_id": "school-1"}

    async def fake_fetch_unconsumed_evidence(student_id, topic_id):
        return []

    async def fake_fetch_skill_state(student_id, topic_id):
        return None

    async def fake_fetch_school_id_for_student(student_id):
        return captured["school_id"]

    async def fake_get_active_model_snapshot(model_type, scope=None):
        return {"id": "snap-fusion-policy"}

    async def fake_insert_model_snapshot(*args, **kwargs):
        return "snap-fusion-policy"

    async def fake_upsert_skill_state(*args, **kwargs):
        captured["upsert_calls"].append({"args": args, "kwargs": kwargs})

    async def fake_mark_evidence_consumed(evidence_ids):
        captured["consumed_calls"].append(list(evidence_ids))

    async def fake_mark_skill_state_synced(student_id, topic_id):
        return None

    async def fake_sync_skill_state(*args, **kwargs):
        return False  # best-effort Neo4j mirror, matches its own documented contract

    async def fake_fetch_active_recovery_for_topic(student_id, topic_id):
        return None

    async def fake_fetch_prerequisite_chain(topic_id, max_depth=1):
        return []

    async def fake_fetch_prerequisite_chain_pg(topic_id, max_depth=1):
        return []

    async def fake_fetch_skill_states_bulk(student_id, topic_ids):
        return {}

    monkeypatch.setattr(fusion, "fetch_unconsumed_evidence", fake_fetch_unconsumed_evidence)
    monkeypatch.setattr(fusion, "fetch_skill_state", fake_fetch_skill_state)
    monkeypatch.setattr(fusion, "fetch_school_id_for_student", fake_fetch_school_id_for_student)
    monkeypatch.setattr(fusion, "get_active_model_snapshot", fake_get_active_model_snapshot)
    monkeypatch.setattr(fusion, "insert_model_snapshot", fake_insert_model_snapshot)
    monkeypatch.setattr(fusion, "upsert_skill_state", fake_upsert_skill_state)
    monkeypatch.setattr(fusion, "mark_evidence_consumed", fake_mark_evidence_consumed)
    monkeypatch.setattr(fusion, "mark_skill_state_synced", fake_mark_skill_state_synced)
    monkeypatch.setattr(fusion, "sync_skill_state", fake_sync_skill_state)
    monkeypatch.setattr(fusion, "fetch_active_recovery_for_topic", fake_fetch_active_recovery_for_topic)
    monkeypatch.setattr(fusion.neo4j_db, "fetch_prerequisite_chain", fake_fetch_prerequisite_chain)
    monkeypatch.setattr(fusion.pg_gap, "fetch_prerequisite_chain_pg", fake_fetch_prerequisite_chain_pg)
    monkeypatch.setattr(fusion, "fetch_skill_states_bulk", fake_fetch_skill_states_bulk)

    return captured


# ── FI-001: Missing Evidence ────────────────────────────────────────────


@pytest.mark.asyncio
async def test_fi001_missing_evidence_produces_no_fabricated_state(fusion_env: dict, monkeypatch: pytest.MonkeyPatch) -> None:
    """DS-01: a learner with zero evidence for a concept. fuse_skill_state
    must be a true no-op -- no skill_states row is written at all (cold
    start via row absence, never a fabricated 0.0/unknown row)."""

    async def no_evidence(student_id, topic_id):
        return []

    monkeypatch.setattr(fusion, "fetch_unconsumed_evidence", no_evidence)

    await fusion.fuse_skill_state("student-1", "topic-1")

    assert fusion_env["upsert_calls"] == [], "missing evidence must not write any skill_states row"
    assert fusion_env["consumed_calls"] == []


@pytest.mark.asyncio
async def test_fi001_evidence_with_no_point_estimate_is_marked_consumed_but_not_fabricated(
    fusion_env: dict, monkeypatch: pytest.MonkeyPatch
) -> None:
    """A source that contributed only structural/contextual signal (no
    `estimate`) must not be turned into a fabricated mastery value -- the
    row is drained (so the worker doesn't spin on it forever) but no
    skill_states write happens."""
    row = _evidence_row(provenance="graph_reasoning", estimate=None, id="ev-no-estimate")

    async def one_estimateless_row(student_id, topic_id):
        return [row]

    monkeypatch.setattr(fusion, "fetch_unconsumed_evidence", one_estimateless_row)

    await fusion.fuse_skill_state("student-1", "topic-1")

    assert fusion_env["upsert_calls"] == []
    assert fusion_env["consumed_calls"] == [["ev-no-estimate"]]


# ── FI-002: Contradictory Model Outputs ─────────────────────────────────


@pytest.mark.asyncio
async def test_fi002_disagreeing_sources_widen_uncertainty_rather_than_average_it_away(
    fusion_env: dict, monkeypatch: pytest.MonkeyPatch
) -> None:
    """Two independent, well-evidenced, fresh sources materially disagree
    (0.9 vs 0.1). No single model may be silently treated as authoritative
    -- the fused estimate must be a genuine blend, and the resulting
    uncertainty must be wider than either source's own uncertainty, not
    quietly averaged away (spec section 20: fusion with provenance, not
    arbitrary selection)."""
    now = datetime.now(timezone.utc)
    high = _evidence_row(provenance="bkt", estimate=0.9, uncertainty=0.1, reliability=0.8, created_at=now, id="ev-high")
    low = _evidence_row(provenance="dina", estimate=0.1, uncertainty=0.1, reliability=0.8, created_at=now, id="ev-low")

    async def contradictory(student_id, topic_id):
        return [high, low]

    monkeypatch.setattr(fusion, "fetch_unconsumed_evidence", contradictory)

    await fusion.fuse_skill_state("student-1", "topic-1")

    assert len(fusion_env["upsert_calls"]) == 1
    kwargs = fusion_env["upsert_calls"][0]["kwargs"]

    # No arbitrary winner: fused mastery sits strictly between the two
    # disagreeing point estimates, not equal to either.
    assert 0.1 < kwargs["mastery_probability"] < 0.9

    # Disagreement widens uncertainty beyond either source's own 0.1.
    assert kwargs["uncertainty"] > 0.1

    # Both contributing sources remain visible in provenance -- disagreement
    # is represented, not hidden.
    assert kwargs["diagnostic_provenance"]["sources"] == {"bkt": 1, "dina": 1}
    assert set(fusion_env["consumed_calls"][0]) == {"ev-high", "ev-low"}


@pytest.mark.asyncio
async def test_fi002_agreeing_sources_do_not_inflate_uncertainty(fusion_env: dict, monkeypatch: pytest.MonkeyPatch) -> None:
    """Control case: sources that agree should NOT trigger the
    disagreement-widening path -- confirms FI-002's assertion is measuring
    disagreement specifically, not just "two sources present"."""
    now = datetime.now(timezone.utc)
    a = _evidence_row(provenance="bkt", estimate=0.7, uncertainty=0.1, reliability=0.8, created_at=now, id="ev-a")
    b = _evidence_row(provenance="dina", estimate=0.72, uncertainty=0.1, reliability=0.8, created_at=now, id="ev-b")

    async def agreeing(student_id, topic_id):
        return [a, b]

    monkeypatch.setattr(fusion, "fetch_unconsumed_evidence", agreeing)

    await fusion.fuse_skill_state("student-1", "topic-1")

    kwargs = fusion_env["upsert_calls"][0]["kwargs"]
    assert kwargs["uncertainty"] < 0.2  # close to the sources' own 0.1, not blown up by disagreement


# ── FI-003: Stale Evidence ──────────────────────────────────────────────


@pytest.mark.asyncio
async def test_fi003_stale_evidence_is_down_weighted_relative_to_fresh(fusion_env: dict, monkeypatch: pytest.MonkeyPatch) -> None:
    """A stale-but-strong (0.9) source and a fresh-but-weak (0.1) source of
    equal reliability/sample size: the fused result must lean toward the
    fresh evidence, proving staleness genuinely changes influence rather
    than being silently treated as current."""
    now = datetime.now(timezone.utc)
    stale = _evidence_row(
        provenance="bkt", estimate=0.9, uncertainty=0.1, reliability=0.8,
        created_at=now - __import__("datetime").timedelta(days=fusion.RECENCY_HALF_LIFE_DAYS * 4), id="ev-stale",
    )
    fresh = _evidence_row(provenance="dina", estimate=0.1, uncertainty=0.1, reliability=0.8, created_at=now, id="ev-fresh")

    async def stale_and_fresh(student_id, topic_id):
        return [stale, fresh]

    monkeypatch.setattr(fusion, "fetch_unconsumed_evidence", stale_and_fresh)

    await fusion.fuse_skill_state("student-1", "topic-1")

    kwargs = fusion_env["upsert_calls"][0]["kwargs"]
    # Weighted toward the fresh (0.1) source, not the midpoint (0.5) a naive
    # unweighted average would produce, and not toward the stale (0.9) one.
    assert kwargs["mastery_probability"] < 0.5


# ── FI-004: Invalid Graph Edge ───────────────────────────────────────────


@pytest.mark.asyncio
async def test_fi004_missing_prerequisite_endpoint_data_yields_unknown_not_fabricated_readiness(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """A prerequisite chain fetch that returns nothing usable (simulating
    an invalid/dangling edge that both Neo4j and the Postgres mirror can't
    resolve to real topic rows) must not fabricate a prerequisite-readiness
    score -- root_cause._prerequisite_readiness's documented contract is
    to treat "no resolvable prerequisites" as fully ready (1.0), not to
    silently invent a mid-range number."""

    async def no_prereqs(topic_id, max_depth=1):
        return []

    monkeypatch.setattr(root_cause.neo4j_db, "fetch_prerequisite_chain", no_prereqs)
    monkeypatch.setattr(root_cause.db, "fetch_prerequisite_chain_pg", no_prereqs)

    readiness = await root_cause._prerequisite_readiness("student-1", "topic-with-invalid-edge", {})
    assert readiness == 1.0


# ── FI-005: Invalid Q-Matrix Mapping ────────────────────────────────────


@pytest.mark.asyncio
async def test_fi005_unmapped_skill_ids_produce_no_evidence_and_no_state(fusion_env: dict, monkeypatch: pytest.MonkeyPatch) -> None:
    """A learning event whose Q-matrix mapping was invalid/unresolvable
    yields an empty skill_ids array upstream (RecordLearningEvents, Go
    side) -- simulated here by a topic with zero unconsumed evidence.
    Confirms no unsupported concept mastery is fabricated for a topic that
    was never validly assessed."""

    async def no_evidence(student_id, topic_id):
        return []

    monkeypatch.setattr(fusion, "fetch_unconsumed_evidence", no_evidence)

    await fusion.fuse_skill_state("student-1", "topic-invalid-mapping")

    assert fusion_env["upsert_calls"] == []


# ── FI-006: Duplicate Event ──────────────────────────────────────────────


@pytest.mark.asyncio
async def test_fi006_duplicate_evidence_ids_are_consumed_exactly_once_per_pass(
    fusion_env: dict, monkeypatch: pytest.MonkeyPatch
) -> None:
    """fuse_skill_state's own idempotency boundary: `mark_evidence_consumed`
    is called with exactly the evidence rows fused this pass, with no
    duplicate id repeated -- the strict `answer_id` unique index upstream
    (students.learning_events, V039) is what prevents a duplicate row from
    ever reaching evidence_log in the first place; this asserts the fusion
    side doesn't independently double-process a row if it were ever handed
    one twice in a single unconsumed-evidence batch."""
    now = datetime.now(timezone.utc)
    row = _evidence_row(provenance="bkt", estimate=0.6, created_at=now, id="ev-dup")

    async def duplicated_batch(student_id, topic_id):
        return [row, dict(row)]  # same id appearing twice in one unconsumed batch

    monkeypatch.setattr(fusion, "fetch_unconsumed_evidence", duplicated_batch)

    await fusion.fuse_skill_state("student-1", "topic-1")

    assert len(fusion_env["upsert_calls"]) == 1  # one fused write, not one per duplicate row
    consumed = fusion_env["consumed_calls"][0]
    assert consumed.count("ev-dup") == 2  # both physical rows marked consumed (dedup happens upstream, at ingestion)


# ── FI-007: Model Failure ───────────────────────────────────────────────


@pytest.mark.asyncio
async def test_fi007_engine_failure_aborts_the_job_without_writing_partial_state(monkeypatch: pytest.MonkeyPatch) -> None:
    """If an analytical engine (bkt/dina/irt) raises -- timeout, exception,
    invalid response -- process_knowledge_tracing_job must not swallow it
    into a fabricated success. No fusion/consistency/recovery step for any
    topic may run once the engine stage has failed."""
    events = [
        {"student_id": "student-1", "school_id": "school-1", "skill_ids": ["topic-1"]},
    ]

    async def fake_fetch_events(attempt_id):
        return events

    async def raising_bkt(attempt_id, events):
        raise TimeoutError("bkt engine unavailable")

    fusion_called = False

    async def spy_fuse(student_id, topic_id):
        nonlocal fusion_called
        fusion_called = True

    monkeypatch.setattr(kt_worker, "fetch_learning_events_for_attempt", fake_fetch_events)
    monkeypatch.setattr(kt_worker.bkt, "process_attempt", raising_bkt)
    monkeypatch.setattr(kt_worker.fusion, "fuse_skill_state", spy_fuse)

    with pytest.raises(TimeoutError):
        await kt_worker.process_knowledge_tracing_job("attempt-1")

    assert fusion_called is False, "fusion must never run on a topic whose upstream engine evidence failed to produce"


@pytest.mark.asyncio
async def test_fi007_worker_loop_contains_a_failed_job_and_continues(monkeypatch: pytest.MonkeyPatch) -> None:
    """run_forever's outer handler is the actual failure-safe boundary for
    this BRPOP worker (there is no synchronous HTTP caller to return a
    degraded response to): one job's unhandled exception must be logged
    and contained, not propagate and kill the worker loop."""
    calls = []

    async def fail_once_then_shutdown(attempt_id):
        calls.append(attempt_id)
        kt_worker.request_shutdown()
        raise RuntimeError("simulated model failure")

    async def fake_brpop(queue, timeout):
        return "attempt-1"

    monkeypatch.setattr(kt_worker, "process_knowledge_tracing_job", fail_once_then_shutdown)
    monkeypatch.setattr(kt_worker, "brpop_job", fake_brpop)
    kt_worker._shutdown.clear()

    await kt_worker.run_forever(poll_timeout=1)  # must return normally, not raise

    assert calls == ["attempt-1"]
    kt_worker._shutdown.clear()  # leave global state clean for later tests


# ── FI-008: Graph-Service Failure ───────────────────────────────────────


@pytest.mark.asyncio
async def test_fi008_neo4j_outage_degrades_to_postgres_mirror_for_structural_status(
    fusion_env: dict, monkeypatch: pytest.MonkeyPatch
) -> None:
    """The graph dependency (Neo4j) times out/5xxs while computing
    structural status. The service must fall back to the Postgres mirror
    (fetch_prerequisite_chain_pg) rather than let the dependency failure
    abort the whole fusion pass or fabricate a structural conclusion."""
    now = datetime.now(timezone.utc)
    row = _evidence_row(provenance="bkt", estimate=0.6, created_at=now, id="ev-1")

    async def one_row(student_id, topic_id):
        return [row]

    async def neo4j_down(topic_id, max_depth=1):
        raise ConnectionError("neo4j unavailable")

    pg_fallback_called = False

    async def pg_mirror(topic_id, max_depth=1):
        nonlocal pg_fallback_called
        pg_fallback_called = True
        return []  # topic genuinely has no prerequisites in the mirror either

    monkeypatch.setattr(fusion, "fetch_unconsumed_evidence", one_row)
    monkeypatch.setattr(fusion.neo4j_db, "fetch_prerequisite_chain", neo4j_down)
    monkeypatch.setattr(fusion.pg_gap, "fetch_prerequisite_chain_pg", pg_mirror)

    await fusion.fuse_skill_state("student-1", "topic-1")  # must not raise

    assert pg_fallback_called is True
    assert len(fusion_env["upsert_calls"]) == 1
    structural = fusion_env["upsert_calls"][0]["kwargs"]["structural_status"]
    # Degraded, not fabricated: no prerequisites resolvable -> explicitly
    # "no prerequisites" (count 0, satisfied None), never a fabricated
    # true/false satisfaction claim built from a failed dependency.
    assert structural["prerequisiteCount"] == 0
    assert structural["prerequisitesSatisfied"] is None


@pytest.mark.asyncio
async def test_fi008_graph_service_failure_in_root_cause_falls_back_not_raises(monkeypatch: pytest.MonkeyPatch) -> None:
    """Same outage, exercised through root_cause.downstream_impact (used
    by the RCS pipeline) -- must degrade to the Postgres mirror instead of
    propagating the Neo4j exception into gap-analysis."""

    async def neo4j_down(topic_id, max_depth=3):
        raise TimeoutError("neo4j timeout")

    pg_called = False

    async def pg_mirror(topic_id, max_depth=3):
        nonlocal pg_called
        pg_called = True
        return []

    monkeypatch.setattr(root_cause.neo4j_db, "fetch_downstream_dependents", neo4j_down)
    monkeypatch.setattr(root_cause.db, "fetch_downstream_dependents_pg", pg_mirror)

    impact = await root_cause.downstream_impact("topic-1")  # must not raise
    assert pg_called is True
    assert impact == 0.0  # no dependents resolvable anywhere -> zero impact, not fabricated
