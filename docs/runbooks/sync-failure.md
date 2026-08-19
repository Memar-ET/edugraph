# Runbook: School-Box Sync Failure Recovery

**Applies to:** School-Box offline sync failures (outbox delivery failures, Neo4j mirror gaps)  
**Severity:** Medium — student data or curriculum updates may not reach the cloud until resolved

---

## 1. Detect

### 1a. Outbox delivery backlog (rows not synced)

Connect to Postgres and check:

```sql
-- Rows waiting to sync (not yet sent)
SELECT table_name, COUNT(*) AS pending
FROM sync.outbox
WHERE synced_at IS NULL
GROUP BY table_name
ORDER BY pending DESC;

-- Rows that failed to sync (error recorded)
SELECT table_name, error, COUNT(*)
FROM sync.outbox
WHERE error IS NOT NULL
GROUP BY table_name, error
ORDER BY COUNT(*) DESC;
```

If `pending` is growing and `synced_at` timestamps are not advancing, the sync worker has stalled.

### 1b. Neo4j mirror gaps (curriculum data)

```sql
-- Prerequisite edges that failed to mirror to Neo4j
SELECT COUNT(*) AS unsynced
FROM curriculum.topic_prerequisites
WHERE neo4j_written = false;

-- Subjects/topics where Neo4j mirror failed
SELECT id, code, name, neo4j_written
FROM curriculum.subjects
WHERE neo4j_written = false;
```

### 1c. sync_logs

```sql
SELECT device_id, direction, status, error_message, created_at
FROM public.sync_logs
WHERE status = 'error'
ORDER BY created_at DESC
LIMIT 50;
```

---

## 2. Diagnose

### 2a. Sync worker is down

```bash
docker compose ps ai-service api
docker compose logs api --tail=100
docker compose logs ai-service --tail=100
```

Look for panics, OOM kills, or Redis connection errors.

### 2b. Neo4j is unreachable

```bash
docker compose ps neo4j
docker compose logs neo4j --tail=50
# Test connectivity:
curl -u neo4j:${NEO4J_PASSWORD} http://localhost:7474/db/data/
```

If Neo4j is down, restart it:
```bash
docker compose restart neo4j
```

### 2c. School-Box device unreachable

Check whether the School-Box device can reach the cloud API:
```bash
# From the School-Box:
curl -I https://<api-host>/health
```

If the cloud API is unreachable, the School-Box will buffer in its local outbox until connectivity is restored. No action needed beyond restoring network access.

---

## 3. Recover

### 3a. Resume stalled sync worker

```bash
docker compose restart api
# Wait 30 s, then check outbox pending count again
```

### 3b. Resync prerequisite edges to Neo4j

If `curriculum.topic_prerequisites.neo4j_written = false` rows exist:

```bash
# Requires ministry_admin token
curl -X POST https://<api-host>/api/v1/curriculum/prerequisites/resync \
  -H "Authorization: Bearer <ministry_admin_token>"
```

Response:
```json
{ "data": { "synced": 90, "failed": 0 } }
```

If `failed > 0`, check Neo4j connectivity (step 2b) and retry.

### 3c. Full Neo4j rebuild

If Neo4j data is corrupted or the resync endpoint cannot recover all edges:

See `docs/runbooks/neo4j-rebuild.md`.

### 3d. Clear stuck outbox rows manually (last resort)

Only do this if you have verified the data was successfully received by the destination:

```sql
-- Mark specific stuck rows as synced (replace with real IDs)
UPDATE sync.outbox
SET synced_at = NOW(), error = NULL
WHERE id IN (<id1>, <id2>);
```

---

## 4. Escalation

- Neo4j data integrity issues → rebuild from Postgres (see neo4j-rebuild runbook)
- Persistent Supabase connectivity failures → check Supabase status page and pooler health
- School-Box offline for >7 days → manual data reconciliation may be required; contact the dev team
