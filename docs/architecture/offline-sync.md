# Offline Sync (checklist 9.1, 9.4, 10.1)

Two halves, built in different sessions, never cross-verified against
each other until the 10.1 audit's live test: the **School Box side**
(local, this repo's `school-box/sync-agent`) and the **Central Cloud
side** (`backend/internal/sync`). They talk to each other over
`/api/v1/sync/push` and `/api/v1/sync/pull` — that's the only place they
actually meet.

## School Box side: transactional outbox + sync-agent

`sync.outbox` (`V029__sync_outbox.sql`) is a standard transactional
outbox: a trigger on every table a School Box writes locally
(`assessment.exams/questions/exam_attempts/student_answers`,
`students.gap_records/mastery_records/study_plans/exam_insights`)
appends a `to_jsonb()` snapshot of the row in the same transaction as the
domain write, so a crash between the two is impossible. This migration
runs against both Central Cloud and School Box Postgres (one shared
migration set) — on Central Cloud the triggers still fire and
`sync.outbox` still fills up, but nothing reads it there; only a School
Box's sync-agent drains it. Accepted simplification, not a bug.

`school-box/sync-agent` (a standalone Go binary, one per box, see its
own `Dockerfile` and `school-box/compose/docker-compose.yml`) runs one
push-then-pull cycle on a timer (`SYNC_INTERVAL_MINUTES`, default 360):

- **Push** (`internal/sync/queue.go`, `agent.go`): drains up to
  `SYNC_PUSH_BATCH_SIZE` unpushed outbox rows, POSTs them to
  `/api/v1/sync/push`, marks them pushed locally on success.
- **Pull** (`agent.go`, `crdt.go`): GETs `/api/v1/sync/pull` since its
  last successful pull, applies each change against
  `entityAllowlist` (`crdt.go`) — last-write-wins for mutable entities
  (exams, questions, gap records, study plans, exam insights),
  append-only set-union for immutable ones (exam attempts, student
  answers) — and logs anything it can't cleanly apply to
  `sync.conflicts` for human review rather than silently dropping either
  side.
- **Never blocks local operation**: both push and pull failures are
  logged and reflected in `/health/sync` (`internal/health`), never
  fatal — a School Box with no internet today just tries again next
  tick and keeps serving its LAN normally in the meantime.

See [`../../school-box/README.md`](../../school-box/README.md) for
deployment and [ai-models-local-vs-cloud.md](ai-models-local-vs-cloud.md)
for the parallel local-AI-inference story.

## Central Cloud side: `/sync/push` and `/sync/pull`

`backend/internal/sync` is a separate, simpler mechanism from
`sync.outbox` above — it uses its own `public.sync_logs` table
(`V009__sync_logs.sql`), not `sync.outbox`. `Push` (`service.go`)
inserts one `sync_logs` row per incoming change, status `pending`.
`cmd/sync-worker` polls and claims pending rows
(`SELECT ... FOR UPDATE SKIP LOCKED`, so multiple worker replicas never
double-process the same row) and marks them `applied` — but **does not
itself write the change into its target domain table**
(`students`, `assessment_results`, etc.); that dispatch was never
implemented (see `cmd/sync-worker/main.go`'s own doc comment). `Pull`
returns every `sync_logs` row with `status = 'applied'` for a school
since a given timestamp.

Practically: today, a pushed change is durably recorded and will show up
on a subsequent pull (verified live, see below), but nothing is actually
applying pushed changes into the real domain tables server-side yet.
That's real, scoped follow-up work — a per-`entity_type` dispatch table
in `sync-worker`, not part of what checklist 10.1 asked for.

### Device authentication (checklist 10.1, fixed 2026-08-13)

Until this fix, `/sync/push` and `/sync/pull` had **no authentication at
all** — the router comment literally said "no per-user auth." Any caller
on the internet could inject fabricated `sync_logs` rows tagged to any
`school_id`, or read back another school's pulled-change stream by
guessing a school UUID. `sync-agent/internal/sync/crdt.go`'s
`entityAllowlist` comment already flagged this as a known gap at the
time the sync-agent was built — it was never actually closed until this
audit.

Fixed with `sync.device_credentials` (`V030`): one row per School Box,
secret bcrypt-hashed (never plaintext), bound to exactly one
`school_id`. `internal/sync/handler/device_auth.go` validates
`X-Device-Id`/`X-Device-Secret` headers (not `Authorization: Bearer` —
a School Box is headless, no human logs in) and the service layer
cross-checks the credential's bound `school_id` against whatever the
request claims, so a *valid* credential for School A still can't
push/pull School B's data by changing a request field.

Provisioning is `backend/cmd/provision-school-box`, a CLI run once per
box before its first boot (no HTTP endpoint — a rare, manual,
operator-driven action, matching the existing "hand-edit
`.env.school-box`" model, see `school-box/README.md`'s Deployment
section). It prints the plaintext secret exactly once; only its hash is
stored.

Full audit writeup: [data-integrity.md](data-integrity.md).
