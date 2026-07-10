# Step 4 — Finalization: PostgreSQL → Neo4j (Knowledge Graph)

Step 3 already did the Postgres half (writes `curriculum.subjects/units/topics/clos`
and marks the job `approved`, all in one transaction). This step adds the
"Magic Moment": after that Postgres transaction commits, the same
`ApproveAndPromote` call mirrors the approved hierarchy into Neo4j.

## What gets written to Neo4j

For every unit and topic just promoted to Postgres:

```cypher
MERGE (sub:Subject {code: $subjectCode})
SET sub.gradeLevel = $gradeLevel, sub.academicYear = $academicYear
MERGE (unit:Unit {id: $unitId})
SET unit.number = $number, unit.titleEn = $titleEn, unit.subjectCode = $subjectCode
MERGE (sub)-[:HAS_UNIT]->(unit)
```
```cypher
MATCH (unit:Unit {id: $unitId})
MERGE (topic:Topic {id: $topicId})
SET topic.titleEn = ..., topic.sequenceOrder = ..., topic.keyConcepts = ..., ...
MERGE (unit)-[:HAS_TOPIC]->(topic)
```

Exactly the node/relationship types you asked for: `:Subject`, `:Unit`,
`:Topic`, `:HAS_UNIT`, `:HAS_TOPIC`. Node ids are the same UUIDs/codes
already used in Postgres (`unit.id`, `topic.id`, `subject.code`) — no
separate id-mapping table needed, and it means re-running this (e.g. via
re-approving after an edit) safely updates the same nodes via `MERGE`
rather than creating duplicates. This mirrors the exact pattern already
used elsewhere in your codebase for Student/CareerPath nodes
(`internal/student/repository`, `internal/career/repository`), so it's
consistent with the rest of the app rather than a new convention.

CLO nodes aren't created here — your spec named `:Subject`/`:Unit`/`:Topic`
specifically for this step and separately called out CLOs/exam-alignment
as "Phase 2 (Exam Validation)", so I kept this step scoped to match.

## Where it runs, and what happens if Neo4j is down

This happens **synchronously inside the same approve request**, right
after the Postgres commit — matching "The Go Backend ... takes these new
PostgreSQL rows and writes them into Neo4j." It's deliberately *not* inside
the Postgres transaction, since Neo4j can't participate in a Postgres
transaction anyway, and the approval itself (the thing the officer is
actually doing) is already durably committed by that point.

So a Neo4j failure doesn't fail the whole request or roll back the
approval — the response comes back `200 OK` either way, with:

```json
{
  "jobId": "...", "status": "approved", "subjectCode": "BIO",
  "unitsPromoted": 6, "topicsPromoted": 18, "closPromoted": 18,
  "graphSynced": true
}
```
or, if Neo4j was unreachable:
```json
{
  "...": "... same promotion counts ...",
  "graphSynced": false,
  "graphSyncError": "sync unit 1 to neo4j: dial tcp ...: connection refused"
}
```

**How to retry a failed sync:** `curriculum.upload_jobs.neo4j_written`
(a column that already existed in your schema, previously unused) stays
`false` when the sync fails. Since every Postgres and Neo4j write here is
an upsert/MERGE, simply calling `POST .../approve` again — with no body,
or the same body — safely retries just the graph sync without duplicating
anything. I didn't build a separate background retry worker for this (your
spec said "the Go backend *or* the sync worker" — I went with the simpler
of the two), but the `neo4j_written` flag is exactly what a future cron/
worker would query (`WHERE status='approved' AND NOT neo4j_written`) if you
want one later — that's the same "unwritten outbox" pattern already used
elsewhere in your schema (`gap_records.neo4j_written`, etc.), so it'll fit
right in.

## Files changed

- `internal/curriculum/repository/repository.go` — `Repository` now holds
  a `neo4j neo4jdriver.DriverWithContext` (same type/import alias your
  student/career repos already use), collects the Postgres-generated
  unit/topic UUIDs while promoting, and calls the new
  `syncCurriculumGraph()` after the transaction commits.
- `internal/curriculum/dto/dto.go` — `ApproveResponse` gained
  `graphSynced`/`graphSyncError`.
- `cmd/api/main.go` — one-line change: `curriculumrepo.New(pgPool)` →
  `curriculumrepo.New(pgPool, neo4jDriver)`, same as the other repos that
  already take it.
- No new migration needed — `neo4j_written` already existed on
  `curriculum.upload_jobs`.

## How to check it worked

After approving a job, open Neo4j Browser (`http://localhost:7474`) and run:
```cypher
MATCH (s:Subject {code: "BIO"})-[:HAS_UNIT]->(u:Unit)-[:HAS_TOPIC]->(t:Topic)
RETURN s, u, t
```
You should see the subject fanning out into units, each fanning out into
its topics — visually, the graph you're describing.

## What I verified vs. couldn't

Same sandbox limitation as before: no live Neo4j/Postgres here to actually
run this against, and I still can't get a full `go build` running (pgx's
pinned version needs Go ≥1.25, and that toolchain isn't reachable from
this sandbox). I did syntax-check every file and matched the Neo4j
session/`MERGE` calling convention exactly against your existing, already-
working student and career repositories rather than inventing a new
pattern — but that's not a substitute for actually running it. Worth
watching the `api` container logs on first approve call.
