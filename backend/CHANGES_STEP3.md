# Step 3 — Human Review (Go API)

Three things, matching your spec exactly:

## 1. `GET /api/v1/curriculum/jobs/{id}` now returns the parsed tree

Previously this only returned `{jobId, status, fileName, error}` — no way
for the frontend to actually render the tree. `JobStatus` now includes:

```json
{
  "jobId": "...",
  "status": "parsed",
  "fileName": "grade11_biology.pdf",
  "subjectCode": "BIO",
  "gradeLevel": 11,
  "academicYear": "2026",
  "parsedStructure": { "units": [ { "number": 1, "titleEn": "...", "topics": [...] } ] },
  "approvedBy": null,
  "approvedAt": null,
  "error": null
}
```

**Access note:** I restricted this endpoint to `curriculum_officer`/`ministry_admin`
(it was open to any authenticated user before) — now that it returns the
full review payload, that seemed like the right default. Let me know if
other roles (e.g. `teacher`) need read access too.

## 2. `GET /api/v1/storage/files/{jobId}` — the dev-mode file proxy

Streams the original uploaded file straight from `storage.local_files`
(Postgres BYTEA) so the officer can view it next to the parsed tree. Same
role restriction as above. The handler doesn't know or care whether it's
talking to Postgres or S3 — it just calls `StorageProvider.Download()` —
so the "prod mode" half of your fork-in-the-road (presigned S3 URL) is a
future swap of *which provider* is wired in, not a change to this endpoint.

## 3. `POST /api/v1/curriculum/jobs/{id}/approve` — the actual promotion

This is the biggest piece. On approve:

1. **Locks the job row** (`SELECT ... FOR UPDATE`) and checks it's actually
   in `parsed` or `review` status — rejects with `409 Conflict` otherwise
   (e.g. someone double-clicking Approve, or approving a job that's still
   parsing).
2. **Accepts an optional edited tree.** Request body:
   ```json
   { "parsedStructure": { "units": [ ... ] } }
   ```
   If provided (the officer fixed something in the UI), that becomes both
   what gets promoted *and* the new value of `parsed_structure` — the
   stored record always matches what was actually approved. If the body is
   empty, the tree already sitting in `parsed_structure` from the AI step
   is approved as-is.

   **Important contract detail:** this must be the *complete* corrected
   tree, not a diff. Any unit/topic/CLO the frontend leaves out is treated
   as intentionally removed. Worth flagging clearly in the frontend so a
   partial edit (e.g. only resubmitting a fixed title) doesn't silently
   drop CLOs the AI already extracted.

3. **Promotes every unit → topic → CLO into the real tables** —
   `curriculum.subjects`, `.units`, `.topics`, `.clos`, `.topic_clo_mappings`
   — all in one transaction, so a bad row partway through rolls back
   everything rather than leaving a half-promoted job.
4. **Upserts, not inserts** — re-approving the same job (say, after another
   edit) updates the existing rows instead of duplicating them. This
   needed two new unique constraints that didn't exist before
   (`db/migrations/V016`): `units(subject_code, number)` and
   `topics(unit_id, sequence_order)`. **You'll need to run this migration**
   for approve to work.
5. **Marks the job `approved`**, with `approved_by`/`approved_at` set, and
   returns a summary:
   ```json
   { "jobId": "...", "status": "approved", "subjectCode": "BIO",
     "unitsPromoted": 6, "topicsPromoted": 18, "closPromoted": 18 }
   ```

### A couple of judgment calls worth knowing about

- **`curriculum.subjects.name_en`** has no better source than the subject
  code itself right now (e.g. `"BIO"` gets used as both `code` and
  `name_en`) — there's no "Biology" human-readable name anywhere in the
  upload flow yet. Officers can rename it later once a subjects-management
  endpoint exists; flagging so it doesn't look like a bug.
- **`clos.moe_version`** is required by the schema but nothing upstream
  produces a real MOE version string yet, so it's set to a placeholder
  (`"ai-draft-<academicYear>"`) at promotion time.
- **`topic_clo_mappings.match_method`** is set to `'human_confirmed'`
  for everything promoted here — reasonable, since promotion only happens
  after a human clicks Approve.

## What I could and couldn't verify

Syntax-checked every changed file. I could **not** get a full `go build`
running in this sandbox — `pgx/v5`'s current pinned version needs Go
≥1.25, and getting that toolchain (or even an older pgx compatible with
the installed Go 1.22) requires reaching `golang.org`/`gopkg.in`, which
aren't reachable from here. Same limitation as the very first backend
pass. I did manually verify the exact pgx v5 API surface used
(`pool.Begin`, `tx.QueryRow`/`Exec`, `tag.RowsAffected()`, `pgx.ErrNoRows`)
against well-established, stable API I've used many times — but that's
not a substitute for an actual compile. Please watch the `api` container's
logs on first build.

## Still open (not part of this step's spec, flagging for later)

- No endpoint to reject a job (`status = 'rejected'`) if the officer
  decides the whole upload is unusable rather than fixable.
- No endpoint to list *pending-review* jobs (`status IN ('parsed','review')`)
  for an officer's queue/dashboard — right now they'd need the job id
  from the upload response.
