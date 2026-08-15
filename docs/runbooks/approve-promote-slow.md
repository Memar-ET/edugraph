# Runbook: ApproveAndPromote Slowness (Vite Proxy False-500)

**Applies to:** Approving a large curriculum job via the `JobReviewPage` UI and seeing an error, even though the data was written correctly  
**Status:** Known issue as of 2026-08-11, not yet fixed

---

## Symptoms

1. Curriculum officer approves a job in the UI
2. The browser shows an error (typically a blank or 500 response)
3. On closer inspection, the job **actually succeeded** — the topic/CLO counts in Postgres and Neo4j are correct
4. `docker compose logs api` shows the request completing (HTTP 200) ~3–5 minutes after the UI showed an error

---

## Root cause

`ApproveAndPromote` issues sequential per-row upserts to Supabase (topics, subtopics, CLOs, topic-CLO mappings) followed by `MERGE` calls to Neo4j. For a large subject (e.g. Biology G7: 95 topics + 138 CLO codes), this is on the order of 200–300 sequential round-trips.

When run against a local Postgres container, each round-trip takes <1ms, so the total is under a second. Against the Supabase pooler (`aws-0-eu-central-1.pooler.supabase.com`), each round-trip pays real network latency (~10–50ms), totalling 2–10 minutes for a full grade.

The Vite dev server proxy (`vite.config.ts`) has a shorter default timeout than the Go server. The proxy closes the connection and returns a 500 to the browser, while the Go handler continues running server-side to completion.

**The data is written correctly despite the UI-visible error.** This is a latency/timeout issue, not a data-correctness bug.

---

## Immediate workaround

After seeing the error, **do not re-approve the job**. Instead, verify the data was written:

```sql
-- Check that the job status flipped to 'approved':
SELECT id, status, approved_at FROM curriculum.upload_jobs WHERE id = '<job_id>';

-- Check topic counts:
SELECT COUNT(*) FROM curriculum.topics WHERE unit_id IN (
  SELECT id FROM curriculum.units WHERE subject_id IN (
    SELECT id FROM curriculum.subjects WHERE upload_job_id = '<job_id>'
  )
);

-- Check Neo4j mirror:
MATCH (s:Subject {uploadJobId: '<job_id>'})-[:HAS_UNIT]->(:Unit)-[:HAS_TOPIC]->(t:Topic)
RETURN count(t);
```

If the counts match what the extractor reported, the approval was successful. Refresh the `CurriculumDashboardPage` — the job should show status `approved`.

---

## For curriculum officers (non-technical)

If you approved a curriculum and saw an error:

1. **Wait 5 minutes**, then navigate to the Curriculum Dashboard.
2. If the job shows **Approved** status — the data was saved correctly; the error was a display glitch.
3. If the job still shows **Review** status after 5 minutes — contact the system administrator; do not re-submit.
4. **Never approve the same job twice** — the upsert-by-natural-key behavior means re-approving overwrites the existing data with identical content, which is harmless but wastes time.

---

## Permanent fix (not yet implemented)

The correct fix has two parts:

**1. Batch the inserts in `ApproveAndPromote`**

Replace per-row `INSERT ... VALUES (...)` loops with multi-row `INSERT ... VALUES (...), (...), (...)` batches. For 200 rows, this reduces round-trips from ~200 to ~1–5. Estimated effort: 2–3 hours. The function is in `backend/internal/curriculum/repository/repository.go`.

**2. Raise the Vite dev-server proxy timeout**

In `vite.config.ts`:

```ts
server: {
  proxy: {
    '/api': {
      target: process.env.API_PROXY_TARGET,
      timeout: 600_000,  // 10 minutes
      proxyTimeout: 600_000,
    }
  }
}
```

This at least ensures the UI reflects the true server response instead of a misleading 500. Note: this only helps in dev (Vite proxy); a production reverse proxy would need the same timeout increase.

---

## Affected subjects

All six Biology grades (G7–G12) were ingested via `cmd/ingest-biology` (which bypasses HTTP entirely and is not affected by this issue). Any new curriculum subject approved through the UI for a subject of similar size (>50 topics) will exhibit this slowness.

Small subjects (<20 topics) complete in under 30 seconds and are not affected.

---

## Monitoring

Check for slow approvals in the API logs:

```bash
docker compose logs api | grep "ApproveAndPromote\|POST /api/v1/curriculum/jobs" | tail -20
```

The log line includes the total duration. Anything over 30 seconds indicates the Supabase round-trip bottleneck is active.
