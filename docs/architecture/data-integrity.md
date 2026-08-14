# Database & Data Integrity (checklist 10.1–10.3)

## 10.1 — Multi-tenancy audit

Systematic pass over every migration (V001–V029 at the time of this
audit) plus the query layer that reads/writes them, not just a schema
read-through — the checklist explicitly called out that schema coverage
alone doesn't prove enforcement.

**Schema layer: correctly scoped.** Every table holding school/student-
generated data carries `school_id`: `students`, `teachers`,
`assessment.exams/questions/exam_attempts/student_answers`,
`students.gap_records/mastery_records/study_plans`,
`schools.quality_scores`. Curriculum tables (`curriculum.*`) correctly
have *no* `school_id` — curriculum is deliberately national/shared
content, not tenant data, and scoping it would be wrong, not missing.
`careers.career_matches`/`career_matches_v010` don't carry a direct
`school_id` (only `student_id`, joinable to `students.school_id`) — a
minor inconsistency versus the denormalized pattern everywhere else, not
a leak on its own.

**Query layer: three real, exploitable gaps found and fixed.**

1. **`POST/GET /api/v1/sync/push` and `/sync/pull` had no authentication
   at all.** The router comment literally said "no per-user auth." Any
   caller on the internet could inject fabricated `sync_logs` rows
   tagged to any `school_id`, or read back another school's entire
   pulled-change stream by guessing a school UUID. This is the cloud
   endpoint the School Box sync-agent talks to (`internal/sync/*` —
   note this is a separate, older mechanism from the School Box's own
   local `sync.outbox`, see [offline-sync.md](offline-sync.md); the two
   were never cross-verified against each other until this audit's live
   test). **Fixed**: `sync.device_credentials` (V030) — one row per
   School Box, secret bcrypt-hashed, bound to exactly one `school_id`.
   `internal/sync/handler/device_auth.go` validates two headers
   (`X-Device-Id`/`X-Device-Secret`, not `Authorization: Bearer` — a
   School Box is headless, this is a different scheme from the human
   JWT flow) and the service layer cross-checks the credential's bound
   `school_id` against whatever the request claims, so a valid device
   credential for School A still can't push/pull School B's data by
   changing a request field. Provisioning is a new CLI,
   `backend/cmd/provision-school-box` (no HTTP endpoint — this is a
   rare, manual, operator-driven action, matching the existing
   "hand-edit `.env.school-box`" provisioning model, not something that
   needed its own admin UI yet). Live-verified against a real Postgres:
   wrong secret rejected, unknown device rejected, correct secret
   resolves the right `school_id`, matching-school push/pull succeed,
   and — the actual vulnerability — a *valid* credential attempting to
   push or pull a *different* `school_id` is rejected with Forbidden.

2. **`GET /students`, `/students/{id}`, and identically `/teachers`,
   had zero role gate and a client-supplied `school_id`/`id` with no
   ownership check.** Any authenticated account — including a bare
   `student` role — could list any school's full roster or fetch any
   student's record by ID, country-wide. Verified against the frontend's
   actual usage (`GradeExamPage.tsx`, `SchoolAdminDashboardPage.tsx`)
   that both real callers already only ever pass their own
   `user.school_id`, so this was pure attack surface, not something
   legitimate traffic depended on. **Fixed**: `student|teacher/service.go`
   now resolve the caller's own school/region server-side
   (`repository.CallerScope`, mirroring the pattern
   `assessment/service/school_quality.go`'s
   `GetSchoolQualityScores`/`class_heatmap.go` already established) —
   `ministry_admin` unrestricted, `regional_admin` forced to their own
   region (any narrower `school_id` they pass is ANDed with the region
   filter, so a mismatched school just matches zero rows rather than
   being trusted), `school_admin`/`teacher` forced to their own school
   regardless of what the request asked for, everything else `Forbidden`.
   The router additionally excludes a bare `student` role from these
   routes entirely — there's no legitimate reason for one to browse the
   roster.

3. **`POST/GET /students/{studentID}/career/generate|matches` took the
   student id from the URL, not the caller's identity** — an IDOR any
   authenticated account could use to read or trigger AI-generated
   career-match generation for someone else's academic performance data.
   The frontend only ever called it with your own id, same story as
   finding 2. **Fixed**: moved to `/students/me/career/*`
   (`RequireRole(student)`), resolving the student via
   `career/repository.go`'s new `StudentIDByUserID` from the JWT
   subject — the same shape as `study_plan.go`'s established `/me`
   pattern, which this domain just hadn't followed.

**Light-touch, deliberately not fixed further**: `GET /schools`,
`/schools/{id}`, `/regions`, `/regions/{id}` stay open to any
authenticated role — school/region name/address/code is low-sensitivity
directory data, not PII, unlike a student roster. The one gap actually
closed there: `school.List`'s `region_id` is now forced to the caller's
own region for `regional_admin` (same reasoning as finding 2), so a
regional admin can't browse another region's school directory even
though the data itself is low-stakes.

**Out of scope for this audit, flagged for Section 11**: full row-level
RBAC enforcement across every repository (checklist 11.3) is a larger,
separate pass — this audit fixed the specific, concrete leaks it found in
scope for multi-tenancy, not every repository method in the codebase.

## 10.2 — Curriculum versioning schema

Cross-referenced with checklist 1.3 — already resolved before this audit
(V025, 2026-08-10): `curriculum.subjects` gained
`version`/`is_current`/`superseded_at`/`previous_version_code`, a
revision is promoted under a new subject `code` through the normal
upload/approve pipeline and explicitly linked via
`POST /curriculum/subjects/{code}/supersede`. Verified still correct as
part of this audit; no changes needed. See `CLAUDE.md`'s "What's Built —
CLO/Topic Embeddings, Officer Dashboard, Curriculum Versioning" section
for the full original writeup.

## 10.3 — Audit trail / soft-delete decisions

Per-table decision, implemented in V031, not just documented:

| Table(s) | Decision | Why |
|---|---|---|
| `students`, `teachers` | **Soft-delete** (`deleted_at`) | `Delete()` was a hard `DELETE` cascading through `gap_records`/`exam_attempts`/`student_answers`/`study_plans`/`career_matches` — a routine withdrawal, transfer, or admin mistake permanently destroyed a student's entire academic history. `GetByID`/`List`/`Update` now all filter `deleted_at IS NULL`; a second `Delete()` on an already-deleted row correctly reports not-found rather than silently no-op'ing. |
| `assessment.student_answers` grading | **Append-only history** (`assessment.answer_grade_history`) | `exam_attempts.graded_by`/`graded_at` (V011) already covered *first* grading; nothing recorded a regrade — `marks_awarded` was silently overwritten with no record of the previous value or who changed it a second time. This is exactly the "who changed a grade" accountability gap the checklist flags. One row per grading *event* (including the first, not just regrades) so the full history lives in one place. Deliberately scoped to `BulkGradeExam` (a teacher actually typing in marks) only — the self-submit MCQ auto-grading path passes `gradedBy=nil` and creates no history row, since that path is deterministic (derived straight from the answer key) and has no "who decided this" question to answer. |
| `curriculum.subjects/topics/clos` | **`updated_by`** (+ `updated_at` on `subjects`, which had neither) | National curriculum content should be attributable to a specific `curriculum_officer`/`ministry_admin`. Wired into `ApproveAndPromote`'s existing upsert paths (subject/topic/CLO), which already had the approving `userID` in scope — re-approving the same subject code, topic, or CLO is a real edit (`ON CONFLICT DO UPDATE`), not just a first-time insert. |
| Everything else (quality scores, embeddings, sync/notifications, `careers.*`) | **No change** | Computed/recomputed on read or inherently transient/system-driven — no user-facing edit history would ever be lost. |

Live-verified against a real Postgres (all 31 migrations, not just
V031 in isolation): soft-delete hides a row from `GetByID`/`List` while
the row itself survives with `deleted_at` set (confirmed via direct SQL,
not just application-level absence); `answer_grade_history` correctly
records exactly the two teacher-driven grading events in a
self-submit-then-regrade-then-regrade-again sequence and none from the
self-submit, with `graded_by` correctly attributed.
