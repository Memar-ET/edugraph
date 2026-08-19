"""
Point-in-time state reconstruction (EG-GCKT checklist sections 6/18/22:
"support state reconstruction/replay," "support historical state replay
where feasible").

Deliberately a SEPARATE, read-only reimplementation of fusion.py's
weighting formula -- not a call to fusion.fuse_skill_state, which
mutates production state (consumed_by_fusion_at, students.skill_states)
as a side effect that a replay must never trigger. A small amount of
duplicated math is the correct tradeoff against corrupting live learner
state while reconstructing a historical one.

Shared by app/services/evaluation/harness.py (prequential backtesting)
and app/services/knowledge_tracing/snapshot.py (periodic snapshot-taking)
so this logic exists in exactly one place rather than three.
"""

from __future__ import annotations

from datetime import datetime, timezone
from typing import Any, Optional

import asyncpg

from app.db.postgres import get_pool

# Mirrors fusion.py's DEFAULT_SOURCE_RELIABILITY / MIN_SAMPLE_FOR_FULL_WEIGHT
# / RECENCY_HALF_LIFE_DAYS -- kept as an independent copy (see module
# docstring on why replay can't just call fusion.fuse_skill_state).
SOURCE_RELIABILITY: dict[str, float] = {"bkt": 0.8, "dina": 0.8, "irt": 0.5, "graph_reasoning": 0.3}
FALLBACK_RELIABILITY = 0.3
MIN_SAMPLE_FOR_FULL_WEIGHT = 5
RECENCY_HALF_LIFE_DAYS = 30.0


async def prior_evidence(
    student_id: str, topic_id: str, before: Any, provenance: Optional[str] = None
) -> list[asyncpg.Record]:
    """Every modeling.evidence_log row for (student, topic) strictly
    before `before` -- the one non-negotiable rule for any replay/backtest
    (using evidence at-or-after the point being reconstructed would be
    circular)."""
    pool = await get_pool()
    if provenance is not None:
        return await pool.fetch(
            """
            SELECT estimate, uncertainty, sample_size, reliability, provenance, created_at
            FROM modeling.evidence_log
            WHERE student_id = $1 AND topic_id = $2 AND provenance = $3 AND created_at < $4
            ORDER BY created_at DESC
            """,
            student_id,
            topic_id,
            provenance,
            before,
        )
    return await pool.fetch(
        """
        SELECT estimate, uncertainty, sample_size, reliability, provenance, created_at
        FROM modeling.evidence_log
        WHERE student_id = $1 AND topic_id = $2 AND created_at < $3
        ORDER BY created_at DESC
        """,
        student_id,
        topic_id,
        before,
    )


def _recency_weight(created_at: datetime, before: Any) -> float:
    age_days = max(0.0, (before - created_at).total_seconds() / 86400.0)
    return 0.5 ** (age_days / RECENCY_HALF_LIFE_DAYS)


def fuse_point_in_time(evidence: list[asyncpg.Record], before: Any) -> Optional[float]:
    """Read-only reimplementation of fusion.py's weighted-average formula,
    restricted to evidence strictly before `before`."""
    estimated = [row for row in evidence if row["estimate"] is not None]
    if not estimated:
        return None

    total_weight = 0.0
    weighted_sum = 0.0
    for row in estimated:
        reliability = row["reliability"] if row["reliability"] is not None else SOURCE_RELIABILITY.get(row["provenance"], FALLBACK_RELIABILITY)
        sample_factor = min(1.0, (row["sample_size"] or 1) / MIN_SAMPLE_FOR_FULL_WEIGHT)
        weight = reliability * sample_factor * _recency_weight(row["created_at"], before)
        total_weight += weight
        weighted_sum += float(row["estimate"]) * weight

    return weighted_sum / total_weight if total_weight > 0 else None


async def reconstruct_mastery_as_of(student_id: str, topic_id: str, as_of: Optional[datetime] = None) -> Optional[float]:
    """The single-call convenience wrapper: what would GCSF have believed
    about (student, topic) at a given moment, using only evidence that
    existed by then. as_of defaults to now."""
    as_of = as_of or datetime.now(timezone.utc)
    evidence = await prior_evidence(student_id, topic_id, as_of)
    return fuse_point_in_time(evidence, as_of)
