# ADR-004: Keep Neo4j as Local Docker (Not AuraDB)

Date: 2026-08-10  
Status: Accepted

## Context

When Postgres was migrated to hosted Supabase, the question arose whether to also migrate Neo4j to a hosted graph database (Supabase has no native graph offering; the natural choice would be Neo4j AuraDB).

## Decision

Neo4j remains a local Docker container (`neo4j` service in `docker-compose.yml`). It was deliberately not migrated to a hosted graph DB.

Reasons:

1. **Data rebuild is cheap.** The Neo4j graph is a derived view of Postgres data — if it's lost or corrupted, it can be fully rebuilt by re-running `syncCurriculumGraph` and the prerequisite resync endpoint. No unique data lives only in Neo4j.

2. **Connection latency.** Neo4j traversal queries (prerequisite chain walks, class heatmap cross-grade alerts, knowledge graph visualization) make many small round-trips. Adding cloud network latency to each hop would make these features noticeably slower.

3. **Cost.** AuraDB pricing at the required instance size would be significant for a project with no cloud budget.

4. **Operational simplicity.** Having one external dependency (Supabase) instead of two keeps the local dev setup simpler.

## Consequences

**Good:**
- Zero latency Neo4j queries in local dev.
- No additional cloud cost.
- Data can always be rebuilt from Postgres if Neo4j is wiped.

**Bad:**
- Neo4j data does not survive container destruction without a volume mount (already configured in `docker-compose.yml` as `neo4j_data`).
- Multi-developer teams sharing a cloud Postgres cannot share a Neo4j instance — each developer runs their own local Neo4j.
- The first-time graph build from a fresh container requires running the ingest-biology command and prerequisite resync (see `docs/runbooks/neo4j-rebuild.md`).
