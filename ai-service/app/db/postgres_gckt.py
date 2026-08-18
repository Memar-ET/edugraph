"""
Postgres access layer for the EG-GCKT knowledge-tracing pipeline
(Milestones 1-4: event log, evidence log, model snapshots, skill states).

Mirrors postgres_gap.py's shape (module-per-domain, plain asyncpg over the
shared pool from app.db.postgres) rather than introducing a second ORM or
query-building layer.
"""

from __future__ import annotations

import json
from typing import Any, Optional

import asyncpg

from app.db.postgres import get_pool


async def fetch_learning_events_for_attempt(attempt_id: str) -> list[asyncpg.Record]:
    """Every students.learning_events row written by Go's
    RecordLearningEvents for this attempt -- one per graded answer, each
    already carrying its resolved skill_ids array. This is what
    kt_worker.py hands to the BKT/DINA engines (Milestones 2-3); neither
    engine re-derives skill_ids itself, they just consume what Go already
    resolved via the versioned Q-matrix (item_skill_mappings) or the
    questions.topic_id fallback."""
    pool = await get_pool()
    return await pool.fetch(
        """
        SELECT id, student_id, school_id, item_id, answer_id, occurred_at,
               skill_ids, response, correctness, response_time_secs, item_version
        FROM students.learning_events
        WHERE answer_id IN (
            SELECT id FROM assessment.student_answers WHERE attempt_id = $1
        )
        ORDER BY occurred_at
        """,
        attempt_id,
    )


async def get_active_model_snapshot(model_type: str, scope: Optional[str] = None) -> Optional[asyncpg.Record]:
    """The currently-active parameter set for one engine (spec section 19's
    candidate/validated/active/rejected lifecycle) -- e.g. BKT's population
    priors, scoped globally (scope=NULL) until a real per-subject refit
    (Milestone 9) creates a scoped one. Returns None if nothing has been
    seeded yet -- callers must fall back to hardcoded population defaults
    in that case, never fabricate a snapshot row on the fly."""
    pool = await get_pool()
    if scope is None:
        return await pool.fetchrow(
            """
            SELECT id, version, config, scope FROM modeling.model_snapshots
            WHERE model_type = $1 AND scope IS NULL AND status = 'active'
            ORDER BY version DESC LIMIT 1
            """,
            model_type,
        )
    return await pool.fetchrow(
        """
        SELECT id, version, config, scope FROM modeling.model_snapshots
        WHERE model_type = $1 AND scope = $2 AND status = 'active'
        ORDER BY version DESC LIMIT 1
        """,
        model_type,
        scope,
    )


async def insert_model_snapshot(
    model_type: str,
    config: dict[str, Any],
    *,
    scope: Optional[str] = None,
    status: str = "active",
    training_summary: Optional[dict[str, Any]] = None,
    notes: Optional[str] = None,
) -> str:
    """Inserts a new modeling.model_snapshots row and returns its id.
    Never mutates an existing snapshot in place -- a refit always creates a
    new version row (spec section 19's replay requirement) even when it's
    immediately marked 'active'. Callers seeding population-default
    parameters (Milestones 2/3) call this once with status='active' and
    scope=None; a nightly refit (Milestone 9) calls it with
    status='candidate' pending governance review."""
    pool = await get_pool()
    next_version = await pool.fetchval(
        """
        SELECT COALESCE(MAX(version), 0) + 1 FROM modeling.model_snapshots
        WHERE model_type = $1 AND scope IS NOT DISTINCT FROM $2
        """,
        model_type,
        scope,
    )
    row = await pool.fetchrow(
        """
        INSERT INTO modeling.model_snapshots (model_type, version, status, scope, config, training_summary, notes)
        VALUES ($1, $2, $3, $4, $5::jsonb, $6::jsonb, $7)
        RETURNING id
        """,
        model_type,
        next_version,
        status,
        scope,
        json.dumps(config),
        json.dumps(training_summary) if training_summary is not None else None,
        notes,
    )
    return str(row["id"])


async def insert_evidence(
    student_id: str,
    topic_id: str,
    *,
    estimate: Optional[float],
    uncertainty: Optional[float],
    sample_size: Optional[int],
    reliability: Optional[float],
    provenance: str,
    context: Optional[dict[str, Any]] = None,
    model_snapshot_id: Optional[str] = None,
    source_event_id: Optional[str] = None,
) -> Optional[str]:
    """Appends one Evidence = {estimate, uncertainty, recency, sample_size,
    reliability, provenance, context, model_version} object (spec section
    8.1) to modeling.evidence_log. Append-only -- never updated after
    insert. `recency` isn't a parameter here: it's the row's own
    created_at/default now(), which IS the recency timestamp fusion
    (Milestone 4) reads.

    ON CONFLICT DO NOTHING on (source_event_id, provenance) (V058) is
    what makes a dead-lettered/requeued knowledge-tracing job idempotent:
    without it, re-processing the same attempt_id (bkt/dina/irt always
    pass source_event_id = the originating learning_events.id, stable
    across re-processing) inserted duplicate evidence rows, which fusion
    then double-folded into skill_states -- inflating evidence_count/
    sample_size and artificially sharpening uncertainty. Returns None
    when the insert was skipped as a duplicate; no caller currently uses
    the return value, so this is a safe default rather than raising.
    graph_reasoning evidence (no source_event_id) is unaffected -- the
    unique index only covers source_event_id IS NOT NULL rows."""
    pool = await get_pool()
    row = await pool.fetchrow(
        """
        INSERT INTO modeling.evidence_log
            (student_id, topic_id, source_event_id, estimate, uncertainty, sample_size,
             reliability, provenance, context, model_snapshot_id)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10)
        ON CONFLICT (source_event_id, provenance) WHERE source_event_id IS NOT NULL
        DO NOTHING
        RETURNING id
        """,
        student_id,
        topic_id,
        source_event_id,
        estimate,
        uncertainty,
        sample_size,
        reliability,
        provenance,
        json.dumps(context) if context is not None else None,
        model_snapshot_id,
    )
    return str(row["id"]) if row is not None else None


async def fetch_unconsumed_evidence(student_id: str, topic_id: str) -> list[asyncpg.Record]:
    """Every modeling.evidence_log row for (student, topic) not yet folded
    into a skill_states estimate (consumed_by_fusion_at IS NULL) -- what
    fusion.py reads. Ordered oldest-first so a caller computing recency
    decay can weight later rows more heavily without re-sorting."""
    pool = await get_pool()
    return await pool.fetch(
        """
        SELECT id, estimate, uncertainty, recency, sample_size, reliability, provenance, context, created_at
        FROM modeling.evidence_log
        WHERE student_id = $1 AND topic_id = $2 AND consumed_by_fusion_at IS NULL
        ORDER BY created_at
        """,
        student_id,
        topic_id,
    )


async def mark_evidence_consumed(evidence_ids: list[str]) -> None:
    """Stamps consumed_by_fusion_at so a later fusion pass doesn't re-fuse
    the same evidence twice. Evidence rows are never deleted or updated
    otherwise -- this is the only mutation ever applied to
    modeling.evidence_log, and it's additive metadata, not a value
    change."""
    if not evidence_ids:
        return
    pool = await get_pool()
    await pool.execute(
        "UPDATE modeling.evidence_log SET consumed_by_fusion_at = now() WHERE id = ANY($1::uuid[])",
        evidence_ids,
    )


async def fetch_skill_state(student_id: str, topic_id: str) -> Optional[asyncpg.Record]:
    """The current students.skill_states row, if one exists yet -- used by
    fusion.py to compute trend/learning_velocity against the prior
    estimate. Returns None for a (student, topic) pair with no row at all,
    which is the correct cold-start signal (see V038's migration comment
    on why a row is never pre-created)."""
    pool = await get_pool()
    return await pool.fetchrow(
        """
        SELECT mastery_probability, computed_at, evidence_count
        FROM students.skill_states
        WHERE student_id = $1 AND topic_id = $2
        """,
        student_id,
        topic_id,
    )


async def upsert_skill_state(
    student_id: str,
    school_id: str,
    topic_id: str,
    *,
    mastery_probability: Optional[float],
    mastery_status: str,
    uncertainty: Optional[float],
    evidence_count: int,
    evidence_quality: Optional[float],
    recent_performance: Optional[float],
    trend: Optional[str],
    learning_velocity: Optional[float],
    forgetting_risk: Optional[float],
    last_seen: Any,
    next_item_success: Optional[float],
    diagnostic_provenance: Optional[dict[str, Any]],
    model_snapshot_id: Optional[str],
    structural_status: Optional[dict[str, Any]] = None,
) -> None:
    """Writes the fused EG-GCKT Student Knowledge State (spec section 6.4)
    for one (student, topic) pair. Upsert on the (student_id, topic_id)
    unique constraint (V038) -- the lazily-created-on-first-evidence row
    every subsequent fusion pass updates in place; skill_states is a
    current-state table, not an append-only log (that's what
    evidence_log/learning_events are for). structural_status (V045) is
    GCSF's "prerequisite satisfaction, blocked status, inconsistencies"
    output (spec section 8.3)."""
    pool = await get_pool()
    await pool.execute(
        """
        INSERT INTO students.skill_states
            (student_id, school_id, topic_id, mastery_probability, mastery_status, uncertainty,
             evidence_count, evidence_quality, recent_performance, trend, learning_velocity,
             forgetting_risk, last_seen, next_item_success, diagnostic_provenance, model_snapshot_id,
             structural_status, computed_at, neo4j_written)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15::jsonb, $16, $17::jsonb, now(), FALSE)
        ON CONFLICT (student_id, topic_id) DO UPDATE SET
            mastery_probability = EXCLUDED.mastery_probability,
            mastery_status = EXCLUDED.mastery_status,
            uncertainty = EXCLUDED.uncertainty,
            evidence_count = EXCLUDED.evidence_count,
            evidence_quality = EXCLUDED.evidence_quality,
            recent_performance = EXCLUDED.recent_performance,
            trend = EXCLUDED.trend,
            learning_velocity = EXCLUDED.learning_velocity,
            forgetting_risk = EXCLUDED.forgetting_risk,
            last_seen = EXCLUDED.last_seen,
            next_item_success = EXCLUDED.next_item_success,
            diagnostic_provenance = EXCLUDED.diagnostic_provenance,
            model_snapshot_id = EXCLUDED.model_snapshot_id,
            structural_status = EXCLUDED.structural_status,
            computed_at = now(),
            neo4j_written = FALSE
        """,
        student_id,
        school_id,
        topic_id,
        mastery_probability,
        mastery_status,
        uncertainty,
        evidence_count,
        evidence_quality,
        recent_performance,
        trend,
        learning_velocity,
        forgetting_risk,
        last_seen,
        next_item_success,
        json.dumps(diagnostic_provenance) if diagnostic_provenance is not None else None,
        model_snapshot_id,
        json.dumps(structural_status) if structural_status is not None else None,
    )


async def mark_skill_state_synced(student_id: str, topic_id: str) -> None:
    pool = await get_pool()
    await pool.execute(
        "UPDATE students.skill_states SET neo4j_written = TRUE WHERE student_id = $1 AND topic_id = $2",
        student_id,
        topic_id,
    )


async def fetch_skill_states_bulk(student_id: str, topic_ids: list[str]) -> dict[str, dict[str, Any]]:
    """Current students.skill_states rows for a set of topics, keyed by
    topic_id -- the EG-GCKT root-cause engine (Milestone 5) prefers this
    richer, uncertainty-aware estimate over the legacy marks-ratio
    (postgres_gap.fetch_topic_mastery) whenever a fused row exists, and
    falls back to the legacy ratio only for topics fusion hasn't touched
    yet (e.g. gap_analysis and knowledge_tracing can race on the same
    attempt, or a topic was never assessed via an item with a Q-matrix
    mapping)."""
    if not topic_ids:
        return {}
    pool = await get_pool()
    rows = await pool.fetch(
        """
        SELECT topic_id, mastery_probability, uncertainty, evidence_count, forgetting_risk
        FROM students.skill_states
        WHERE student_id = $1 AND topic_id = ANY($2::uuid[])
        """,
        student_id,
        topic_ids,
    )
    return {
        str(r["topic_id"]): {
            "mastery_probability": float(r["mastery_probability"]) if r["mastery_probability"] is not None else None,
            "uncertainty": float(r["uncertainty"]) if r["uncertainty"] is not None else None,
            "evidence_count": int(r["evidence_count"] or 0),
            "forgetting_risk": float(r["forgetting_risk"]) if r["forgetting_risk"] is not None else None,
        }
        for r in rows
    }


async def fetch_recovery_route_candidates(topic_id: str) -> list[dict[str, Any]]:
    """similar_to/alternative_to/related_to edges touching this topic, in
    EITHER direction -- these edges are stored as one directional row
    (see V033's migration comment) but are semantically symmetric-ish
    associations, so a route must be discoverable from whichever side the
    edge was authored from. 'lower_granularity' candidates (this topic's
    own subtopics) are appended separately since they're not prerequisite
    edges at all -- curriculum.topics.parent_topic_id."""
    pool = await get_pool()
    edge_rows = await pool.fetch(
        """
        SELECT prerequisite_id AS route_topic_id, edge_type, weight, confidence
        FROM curriculum.topic_prerequisites
        WHERE topic_id = $1 AND edge_type IN ('similar_to', 'alternative_to', 'related_to')
        UNION
        SELECT topic_id AS route_topic_id, edge_type, weight, confidence
        FROM curriculum.topic_prerequisites
        WHERE prerequisite_id = $1 AND edge_type IN ('similar_to', 'alternative_to', 'related_to')
        """,
        topic_id,
    )
    candidates = [
        {
            "route_topic_id": str(r["route_topic_id"]),
            "edge_type": r["edge_type"],
            "weight": float(r["weight"]) if r["weight"] is not None else 1.0,
            "confidence": float(r["confidence"]) if r["confidence"] is not None else None,
        }
        for r in edge_rows
    ]

    subtopic_rows = await pool.fetch("SELECT id FROM curriculum.topics WHERE parent_topic_id = $1", topic_id)
    for r in subtopic_rows:
        candidates.append(
            {"route_topic_id": str(r["id"]), "edge_type": "lower_granularity", "weight": 1.0, "confidence": None}
        )
    return candidates


async def fetch_recent_correctness(student_id: str, topic_id: str, limit: int = 5) -> list[bool]:
    """The student's most recent responses on this skill, newest first --
    what recovery.py's repeated-failure detector reads. Deliberately
    reads students.learning_events (raw responses), not skill_states or
    evidence_log, since "3 wrong in a row" is about actual response
    history, not a smoothed/fused estimate."""
    pool = await get_pool()
    rows = await pool.fetch(
        """
        SELECT correctness FROM students.learning_events
        WHERE student_id = $1 AND $2 = ANY(skill_ids) AND correctness IS NOT NULL
        ORDER BY occurred_at DESC
        LIMIT $3
        """,
        student_id,
        topic_id,
        limit,
    )
    return [bool(r["correctness"]) for r in rows]


async def fetch_tried_recovery_routes(student_id: str, blocked_topic_id: str) -> list[str]:
    """Every route_topic_id already attempted (any status) for this
    (student, blocked_topic) pair -- excluded from future route search so
    a failed route isn't retried, and repetition penalties (spec section
    13: "avoid cycling through ineffective interventions") have
    something to check against."""
    pool = await get_pool()
    rows = await pool.fetch(
        "SELECT route_topic_id FROM students.recovery_attempts WHERE student_id = $1 AND blocked_topic_id = $2",
        student_id,
        blocked_topic_id,
    )
    return [str(r["route_topic_id"]) for r in rows]


async def fetch_blocked_topics(student_id: str, topic_ids: list[str]) -> set[str]:
    """Which of these topics currently have an active recovery attempt --
    read by action_ranking.py so the next-best-action engine doesn't keep
    re-suggesting a topic the student is already blocked on (spec section
    13's repetition-penalty principle applied to the ranking engine, not
    just to recovery-route selection itself)."""
    if not topic_ids:
        return set()
    pool = await get_pool()
    rows = await pool.fetch(
        """
        SELECT DISTINCT blocked_topic_id FROM students.recovery_attempts
        WHERE student_id = $1 AND blocked_topic_id = ANY($2::uuid[]) AND status = 'active'
        """,
        student_id,
        topic_ids,
    )
    return {str(r["blocked_topic_id"]) for r in rows}


async def fetch_active_recovery_for_topic(student_id: str, blocked_topic_id: str) -> Optional[asyncpg.Record]:
    pool = await get_pool()
    return await pool.fetchrow(
        """
        SELECT id, route_topic_id, route_edge_type, triggered_at
        FROM students.recovery_attempts
        WHERE student_id = $1 AND blocked_topic_id = $2 AND status = 'active'
        ORDER BY triggered_at DESC LIMIT 1
        """,
        student_id,
        blocked_topic_id,
    )


async def fetch_active_recovery_attempts(student_id: str) -> list[asyncpg.Record]:
    """Every currently-active recovery attempt for a student -- checked
    after each new piece of evidence in case the route topic (not just
    the originally-blocked one) just improved."""
    pool = await get_pool()
    return await pool.fetch(
        """
        SELECT id, blocked_topic_id, route_topic_id, route_edge_type, triggered_at
        FROM students.recovery_attempts
        WHERE student_id = $1 AND status = 'active'
        """,
        student_id,
    )


async def insert_recovery_attempt(
    student_id: str,
    school_id: str,
    blocked_topic_id: str,
    route_topic_id: str,
    route_edge_type: str,
    trigger_reason: str,
) -> str:
    pool = await get_pool()
    row = await pool.fetchrow(
        """
        INSERT INTO students.recovery_attempts
            (student_id, school_id, blocked_topic_id, route_topic_id, route_edge_type, trigger_reason)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id
        """,
        student_id,
        school_id,
        blocked_topic_id,
        route_topic_id,
        route_edge_type,
        trigger_reason,
    )
    return str(row["id"])


async def update_recovery_status(attempt_id: str, status: str, notes: Optional[str] = None) -> None:
    """Records the outcome of a recovery route -- 'record which recovery
    route succeeded or failed for future personalization' (spec section
    13). Terminal statuses (anything but 'active') stamp resolved_at."""
    pool = await get_pool()
    await pool.execute(
        """
        UPDATE students.recovery_attempts
        SET status = $2, notes = COALESCE($3, notes),
            resolved_at = CASE WHEN $2 != 'active' THEN now() ELSE resolved_at END
        WHERE id = $1
        """,
        attempt_id,
        status,
        notes,
    )


async def fetch_school_id_for_student(student_id: str) -> Optional[str]:
    """skill_states.school_id is NOT NULL (matches every other students.*
    table's shape and the sync-outbox trigger's school_id extraction) --
    resolved here rather than threaded through every evidence-producing
    call site."""
    pool = await get_pool()
    row = await pool.fetchrow("SELECT school_id FROM students WHERE id = $1", student_id)
    return str(row["school_id"]) if row else None


async def fetch_skill_state_full(student_id: str, topic_id: str) -> Optional[asyncpg.Record]:
    """Every column of the current skill_states row -- what
    snapshot.take_snapshot copies. Distinct from fetch_skill_state (which
    only reads the three columns fusion.py needs for trend calculation)
    since a snapshot needs the whole row."""
    pool = await get_pool()
    return await pool.fetchrow(
        """
        SELECT school_id, mastery_probability, mastery_status, uncertainty,
               evidence_count, trend, model_snapshot_id
        FROM students.skill_states
        WHERE student_id = $1 AND topic_id = $2
        """,
        student_id,
        topic_id,
    )


async def fetch_evidence_range(student_id: str, topic_id: str) -> tuple[Optional[Any], Optional[Any]]:
    """(earliest, latest) created_at across every modeling.evidence_log
    row for (student, topic) -- the "source event range" a
    skill_state_snapshot is versioned against (spec section 18)."""
    pool = await get_pool()
    row = await pool.fetchrow(
        "SELECT min(created_at) AS earliest, max(created_at) AS latest FROM modeling.evidence_log WHERE student_id = $1 AND topic_id = $2",
        student_id,
        topic_id,
    )
    if row is None:
        return None, None
    return row["earliest"], row["latest"]


async def fetch_all_skill_state_pairs() -> list[tuple[str, str]]:
    """Every (student_id, topic_id) pair with a current skill_states row
    -- what the nightly snapshot pass (refit_worker.py) iterates. Bounded
    by this system's actual data volume (near-zero real interaction data
    at launch, per the implementation plan), so a full scan here is cheap
    today; a future high-volume deployment would need to batch/paginate
    this instead."""
    pool = await get_pool()
    rows = await pool.fetch("SELECT student_id, topic_id FROM students.skill_states")
    return [(str(r["student_id"]), str(r["topic_id"])) for r in rows]


async def insert_skill_state_snapshot(
    student_id: str,
    school_id: str,
    topic_id: str,
    *,
    mastery_probability: Optional[float],
    mastery_status: str,
    uncertainty: Optional[float],
    evidence_count: int,
    trend: Optional[str],
    model_snapshot_id: Optional[str],
    snapshot_reason: str,
    source_event_range_start: Any,
    source_event_range_end: Any,
) -> str:
    """Appends one point-in-time copy of a skill_states row -- an
    immutable historical record (spec: "support learner-state snapshots,"
    "version learner-state snapshots with source event range"),
    distinct from skill_states itself (upserted in place, current-value
    only)."""
    pool = await get_pool()
    row = await pool.fetchrow(
        """
        INSERT INTO students.skill_state_snapshots
            (student_id, school_id, topic_id, mastery_probability, mastery_status, uncertainty,
             evidence_count, trend, model_snapshot_id, snapshot_reason,
             source_event_range_start, source_event_range_end)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
        RETURNING id
        """,
        student_id,
        school_id,
        topic_id,
        mastery_probability,
        mastery_status,
        uncertainty,
        evidence_count,
        trend,
        model_snapshot_id,
        snapshot_reason,
        source_event_range_start,
        source_event_range_end,
    )
    return str(row["id"])
