# ADR-006: Typed Prerequisite Edges (7 Types, Widened Unique Constraint)

Date: 2026-08-15  
Status: Accepted

## Context

The original `curriculum.topic_prerequisites` table had a single edge type (implicitly "requires") with a `UNIQUE(topic_id, prerequisite_id)` constraint. This was sufficient for basic prerequisite ordering but did not capture the semantics the EG-GCKT spec requires: the difference between a hard dependency, a soft recommendation, a conceptual similarity, or an alternative path.

The class heatmap's cross-grade alert (`[:HAS_PREREQUISITE*1..3]` Neo4j traversal) would silently treat `similar_to` edges as hard dependency chains once non-`requires` rows existed, producing false alerts.

## Decision

Migration V033 extends `curriculum.topic_prerequisites` with:

1. **`edge_type` column** — `TEXT DEFAULT 'requires' CHECK (edge_type IN ('requires', 'strongly_requires', 'recommended_before', 'related_to', 'similar_to', 'supports', 'alternative_to'))`. Existing rows backfilled to `'requires'`.

2. **Widened unique constraint** — `UNIQUE(topic_id, prerequisite_id)` → `UNIQUE(topic_id, prerequisite_id, edge_type)`. This allows a topic pair to have both a `requires` edge and a `similar_to` edge simultaneously (legitimate — e.g. "Fractions requires Whole Numbers AND is similar to Decimals").

3. **Cycle detection scoped to hard dependencies** — the recursive CTE in `backend/internal/curriculum/repository/prerequisites.go` adds `WHERE edge_type IN ('requires', 'strongly_requires')` on both legs. `similar_to` and `related_to` are legitimately symmetric and must not be cycle-blocked.

4. **Class heatmap Neo4j query fixed** — `class_heatmap.go`'s `[:HAS_PREREQUISITE*1..3]` walk is filtered to `WHERE ALL(r IN relationships(path) WHERE r.edgeType IN ['requires','strongly_requires'])`.

5. **`infer_method` and `confidence`** — human-authored provenance (`manual`/`explicit`/`moe_document`) is auto-validated; `ai_inferred` is left unconfirmed until a teacher/officer explicitly validates it via `PATCH /curriculum/topics/{id}/prerequisites/{prereqId}/validate`.

## Consequences

**Good:**
- EG-GCKT's recovery system (`recovery.py`) can route around a blocked skill via `similar_to`/`alternative_to` edges without confusing them with hard dependencies.
- Cycle detection remains correct and fast (only checks hard edges).
- The class heatmap correctly ignores similarity/support edges when alerting on upstream weaknesses.
- Human-authored vs. AI-inferred provenance is explicit and auditable.

**Bad:**
- Widening the unique constraint means the same topic pair can appear in multiple rows (one per edge type). Queries that assumed one row per pair need an explicit `WHERE edge_type = 'requires'` filter.
- The 7-value CHECK constraint must be kept in sync with the Python `_ROUTE_PRIORITY` dict in `recovery.py` and the Go DTO enum.
