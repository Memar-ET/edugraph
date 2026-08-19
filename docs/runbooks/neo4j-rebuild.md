# Runbook: Neo4j Full Rebuild from Postgres

**Applies to:** Neo4j graph database corruption, accidental data loss, or drift from Postgres source of truth  
**Estimated time:** 5–20 minutes depending on dataset size  
**Data loss risk:** Neo4j is a mirror — all authoritative data lives in Postgres. A full rebuild from Postgres is safe.

---

## When to use this runbook

- Neo4j data is corrupted or nodes are missing
- `neo4j_written` resync endpoint (`POST /curriculum/prerequisites/resync`) reports persistent failures
- You ran a destructive Cypher query by mistake
- Setting up a fresh School-Box or development environment
- After a Neo4j container volume was deleted

---

## 1. Confirm Postgres is the source of truth

Before wiping Neo4j, verify Postgres has the expected data:

```sql
SELECT COUNT(*) FROM curriculum.subjects WHERE neo4j_written = true;
SELECT COUNT(*) FROM curriculum.topics;
SELECT COUNT(*) FROM curriculum.topic_prerequisites WHERE neo4j_written = true;
```

If Postgres counts look correct, proceed.

---

## 2. Wipe Neo4j

> **Warning:** This deletes all Neo4j data. Only proceed if you have confirmed Postgres is intact.

```bash
# Open a Cypher shell:
docker compose exec neo4j cypher-shell -u neo4j -p "${NEO4J_PASSWORD}"

# Inside the shell — delete everything:
MATCH (n) DETACH DELETE n;
# Expected output: 0 rows available after 0 ms (or a deletion count)

# Exit:
:quit
```

---

## 3. Reset `neo4j_written` flags in Postgres

```sql
UPDATE curriculum.subjects SET neo4j_written = false;
UPDATE curriculum.upload_jobs SET neo4j_written = false;
UPDATE curriculum.topic_prerequisites SET neo4j_written = false;
```

---

## 4. Trigger Neo4j rebuild via resync endpoints

### 4a. Resync curriculum graph (subjects, units, topics, CLOs)

Re-approve every currently-approved curriculum job to trigger `syncCurriculumGraph`:

```bash
# Ministry admin only — list approved jobs:
curl https://<api-host>/api/v1/curriculum/jobs?status=approved \
  -H "Authorization: Bearer <ministry_admin_token>"

# For each job_id, re-trigger the approve flow is NOT the right approach.
# Instead, use the bulk resync endpoint:
```

> **Note:** As of 2026-08-15, there is no single `/curriculum/resync-all` endpoint. The rebuild path for curriculum nodes is to use the `ingest-biology` binary (for the real MoE curriculum) or to re-approve each job via the UI.

For the Biology curriculum (the only live content as of 2026-08-15):

```bash
# From the repo root, run the ingest binary directly against local Neo4j:
cd backend
go run ./cmd/ingest-biology/... \
  --neo4j-uri bolt://localhost:7687 \
  --neo4j-user neo4j \
  --neo4j-password "${NEO4J_PASSWORD}" \
  --input ./cmd/ingest-biology/biology_unified.json
```

### 4b. Resync prerequisite edges

```bash
curl -X POST https://<api-host>/api/v1/curriculum/prerequisites/resync \
  -H "Authorization: Bearer <ministry_admin_token>"
```

Expected response:
```json
{ "data": { "synced": 90, "failed": 0 } }
```

### 4c. Resync student progress (STRUGGLED_WITH edges)

The `sync` outbox worker replicates `STRUGGLED_WITH` edges via the outbox mechanism. To force a replay:

```sql
-- Reset outbox rows for student progress (so they re-sync):
UPDATE sync.outbox
SET synced_at = NULL, error = NULL
WHERE table_name IN ('students.gap_records', 'students.skill_states')
  AND synced_at IS NOT NULL;
```

Then restart the sync worker to drain the queue:

```bash
docker compose restart api
```

---

## 5. Verify rebuild

```bash
docker compose exec neo4j cypher-shell -u neo4j -p "${NEO4J_PASSWORD}" \
  "MATCH (n) RETURN labels(n)[0] AS label, count(*) AS count ORDER BY count DESC;"
```

Expected counts for the live Biology G7–G12 dataset:

| Label | Expected |
|---|---|
| Subject | 6 |
| Unit | 33 |
| Topic | 554 (109 top-level + 445 subtopics) |
| CLO | 806 |

Relationship counts:

```bash
docker compose exec neo4j cypher-shell -u neo4j -p "${NEO4J_PASSWORD}" \
  "MATCH ()-[r]->() RETURN type(r) AS rel, count(*) AS count ORDER BY count DESC;"
```

| Relationship | Expected |
|---|---|
| HAS_TOPIC | 554 |
| HAS_SUBTOPIC | 445 |
| HAS_CLO | 4593 |
| HAS_PREREQUISITE | 90 |

If counts match, the rebuild is complete.

---

## 6. Update `neo4j_written` flags in Postgres

After a successful rebuild, mark all rows as synced:

```sql
UPDATE curriculum.subjects SET neo4j_written = true
WHERE id IN (SELECT DISTINCT subject_id FROM curriculum.upload_jobs WHERE status = 'approved');

UPDATE curriculum.topic_prerequisites SET neo4j_written = true;
```

---

## Notes

- The Neo4j rebuild is safe to run multiple times — `MERGE` in Go's `syncCurriculumGraph` is idempotent.
- Neo4j is a mirror, not the source of truth. If Postgres and Neo4j disagree, Postgres wins.
- For a School-Box that has been offline and needs to resync, see `docs/runbooks/sync-failure.md`.
