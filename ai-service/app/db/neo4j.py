"""
Neo4j access for the AI service (Capability 3A: root-cause traversal).

The Go side mirrors the curriculum as (:Subject)-[:HAS_UNIT]->(:Unit)
-[:HAS_TOPIC]->(:Topic) where Topic.id is the Postgres curriculum.topics
UUID (see backend/internal/curriculum/repository syncCurriculumGraph).
Prerequisite edges are (:Topic)-[:HAS_PREREQUISITE]->(:Topic), written by
POST /curriculum/topics/:id/prerequisites (curriculum/service/
prerequisites.go's AddTopicPrerequisite -> SyncPrerequisiteToNeo4j) any
time a curriculum officer links a topic to its prerequisite -- the Postgres
row and the graph edge are written together, best-effort on the graph side.
As of this writing nobody has actually used that endpoint yet (no seed
data, no frontend curation page), so in practice the graph is still empty
today -- but that's a data-population gap, not a missing mechanism.
Callers MUST still treat an empty result as normal (an unpopulated or
partially-populated graph is an expected, ongoing state, not an error) and
fall back to the Postgres recursive walk in postgres_gap.py. Neo4j being
down entirely is likewise non-fatal here: this service's system of record
is Postgres, the graph is a best-effort accelerator. Verified directly
(seeded a real multi-hop chain into both stores and exercised every
combination -- see the checklist 3.1 writeup) that fetch_prerequisite_chain
is genuinely hit and returns real hops when the graph has data.
"""

from __future__ import annotations

from typing import Optional

import structlog
from neo4j import AsyncDriver, AsyncGraphDatabase

from app.core.config import settings

logger = structlog.get_logger()

_driver: Optional[AsyncDriver] = None


def get_driver() -> AsyncDriver:
    global _driver
    if _driver is None:
        _driver = AsyncGraphDatabase.driver(
            settings.NEO4J_URI,
            auth=(settings.NEO4J_USER, settings.NEO4J_PASSWORD),
            max_connection_pool_size=settings.NEO4J_MAX_CONN_POOL_SIZE,
        )
    return _driver


async def close_neo4j() -> None:
    global _driver
    if _driver is not None:
        await _driver.close()
        _driver = None


async def fetch_prerequisite_chain(topic_id: str, max_depth: int = 3) -> list[dict]:
    """
    Walk backwards up the prerequisite graph from a symptom topic:
    every (:Topic) reachable via 1..max_depth HAS_PREREQUISITE hops,
    with the hop count so the caller can prefer the deepest broken link.

    Returns [] when the topic has no prerequisite edges (the normal case
    until the prerequisite graph is populated) AND on any Neo4j error --
    the caller falls back to Postgres either way.
    """
    # Topic property names (titleEn, gradeLevel) match what the Go side's
    # syncCurriculumGraph SETs on the mirrored nodes.
    query = (
        "MATCH path = (t:Topic {id: $topicId})-[:HAS_PREREQUISITE*1..%d]->(p:Topic) "
        "WITH p, min(length(path)) AS depth "
        "RETURN p.id AS id, p.titleEn AS title, p.gradeLevel AS gradeLevel, depth "
        "ORDER BY depth" % max_depth
    )
    try:
        driver = get_driver()
        records, _summary, _keys = await driver.execute_query(
            query, topicId=topic_id, database_="neo4j"
        )
        return [
            {"id": r["id"], "title": r["title"], "grade_level": r["gradeLevel"], "depth": r["depth"]}
            for r in records
        ]
    except Exception:  # noqa: BLE001 -- graph unavailability must never fail analysis
        logger.warning("neo4j.prerequisite_chain_unavailable", topic_id=topic_id)
        return []


async def fetch_prerequisite_edges_among(topic_ids: list[str]) -> list[tuple[str, str]]:
    """
    All direct HAS_PREREQUISITE edges whose BOTH endpoints are in
    topic_ids -- the edge set the study-plan generator (Capability 3B)
    topologically sorts. Returns (topic_id, prerequisite_id) pairs;
    [] on empty graph or Neo4j being down (callers fall back to the
    Postgres edge query, same contract as fetch_prerequisite_chain).
    """
    if not topic_ids:
        return []
    query = (
        "MATCH (t:Topic)-[:HAS_PREREQUISITE]->(p:Topic) "
        "WHERE t.id IN $ids AND p.id IN $ids "
        "RETURN t.id AS topicId, p.id AS prereqId"
    )
    try:
        driver = get_driver()
        records, _summary, _keys = await driver.execute_query(
            query, ids=topic_ids, database_="neo4j"
        )
        return [(r["topicId"], r["prereqId"]) for r in records]
    except Exception:  # noqa: BLE001
        logger.warning("neo4j.prerequisite_edges_unavailable", count=len(topic_ids))
        return []
