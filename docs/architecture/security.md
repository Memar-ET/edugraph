# Security (checklist 11.1, 11.3)

Cross-referenced from [data-integrity.md](data-integrity.md) (10.1's
multi-tenancy findings) and [offline-sync.md](offline-sync.md) (the sync
device-auth fix) — this doc covers auth token storage and the completed
row-level enforcement audit specifically.

## 11.1 — Auth tokens moved off localStorage

`frontend/src/stores/auth.store.ts` used to persist both the access and
refresh JWTs to `localStorage` via Zustand's `persist` middleware — the
file's own prior comment already flagged this as a known gap ("A
production hardening pass should move the refresh token to an httpOnly
cookie... to reduce XSS exposure"). `localStorage` is readable by any
script running on the page, so a single XSS vulnerability anywhere in
the frontend (a dependency, a rendering bug, anything) could exfiltrate
both tokens and fully impersonate the user for the refresh token's
entire 7-day lifetime.

**Fixed**: both tokens are now set as `HttpOnly` cookies by the backend
(`pkg/middleware.SetAuthCookies`) and never appear in a JSON response
body at all (`dto.AuthResponse`'s token fields are `json:"-"`, a
defense-in-depth default that holds even if a future handler bug passes
the struct straight to `WriteJSON`). `HttpOnly` means JavaScript cannot
read the cookie's value under any circumstances, including via a
successful XSS injection — the browser sends it automatically on
matching requests and that's the only way it moves.

- **`SameSite=Lax`**, not `Strict` — `Strict` drops the cookie on a
  top-level GET navigation arriving from an external link (e.g. opening
  the app from an email link while already logged in), which would
  incorrectly bounce a legitimate session to `/login`. `Lax` still
  blocks the cross-site POST/fetch forgery this is actually defending
  against.
- **No separate CSRF token.** Every state-changing request here is
  `Content-Type: application/json`, which a cross-site `<form>` submission
  can't forge without triggering a CORS preflight — and the origin
  allowlist (`pkg/middleware.CORS`, `Access-Control-Allow-Credentials:
  true` only echoed for an allowlisted origin, never a wildcard) rejects
  that preflight from anywhere else.
- **`Secure` is environment-gated**, not hardcoded true: `cfg.AppEnv !=
  "development"`. A `Secure` cookie is silently dropped by the browser
  over plain HTTP, which local dev (`http://localhost`) still is — this
  must flip on wherever the app is actually served over HTTPS.
- **Legacy `Authorization: Bearer` header still accepted** as a fallback
  in `middleware.Authenticate` (cookie checked first) — kept for
  non-browser callers (scripts, future mobile clients, curl/Postman
  during development) that have no cookie jar. It carries none of
  `localStorage`'s exposure since this server never persists it; whatever
  the caller does with the raw token is on them.

**Also fixed in the same pass**: `AppHeader.tsx`/`AppShell.tsx`'s logout
handlers previously only cleared the frontend's local Zustand state
(`clearAuth()`) — before this change that was sufficient, since clearing
`localStorage` removed the only copy of the token that existed. With
`HttpOnly` cookies, client-side JS *cannot* clear them at all; only the
server can via a `Set-Cookie` with a negative `Max-Age`. Without also
calling the new `POST /api/v1/auth/logout` (see `endpoints.ts`'s
`logout()`), a user clicking "Logout" would have left a fully valid
session cookie behind, active until its natural expiry. Both handlers
now call the endpoint before clearing local state.

**Also found and fixed while wiring this up**: `AuthResponse.ExpiresIn`
was computed from the *refresh* token's expiry (~7 days) instead of the
*access* token's (~15 min) — a pre-existing bug, unrelated to cookies,
caught because setting the access cookie's own `Max-Age` needed the
correct value. `pkg/crypto.JWTSigner` gained an `AccessTTL()` accessor
(mirroring the existing `RefreshTTL()`) to fix it.

**Live-verified** against a real Postgres + Redis + Neo4j + the actual
`api` binary (not a script): register/login set both cookies with the
correct `HttpOnly`/`SameSite=Lax` attributes, `Secure` absent in dev
mode; `GET /auth/me` succeeds via cookie alone and is rejected with no
cookie; `POST /auth/refresh` rotates both cookies and the old refresh
token is rejected on reuse (single-use rotation, unchanged from before);
`POST /auth/logout` revokes the refresh token server-side (confirmed: a
subsequent refresh attempt with the logged-out cookie fails) and clears
both cookies; the CORS credentials header is present and scoped to the
allowlisted origin, never a wildcard.

**Known, accepted residual**: like any stateless-JWT scheme, a still-live
access token (up to 15 minutes) keeps working after logout — there's no
per-access-token revocation list, by design (that's the point of a
short-lived, stateless access token backed by a revocable long-lived
refresh token). This isn't a regression from this change; it's the same
tradeoff the architecture already made.

## 11.3 — Row-level enforcement audit (completed)

The 10.1 multi-tenancy audit already covered and fixed the concrete
gaps in this area — see [data-integrity.md](data-integrity.md)'s 10.1
section for the full writeup: unauthenticated sync push/pull, unscoped
student/teacher list/get, and the career-matches IDOR. This section
documents the additional sweep done specifically to close out 11.3 (the
checklist's own framing: "not just role-based UI hiding") across every
remaining repository, since 10.1 didn't claim to be exhaustive over the
whole codebase.

Audited: `assessment` (exams, questions, exam_attempts, student_answers,
exam_quality_reports), `curriculum` (upload_jobs, prerequisites,
versions, topic_list, graph), `students` domain reads (gap records,
study plans, subject profiles), `notification`, `storage`. Method: for
every handler taking an ID (path param or query string) that resolves to
a school/student/teacher-scoped row, checked whether the service/
repository layer verifies the caller may actually see that row, or only
checks role type.

**Finding: the entire teacher-facing exam-management surface had no
school-ownership check.** `GetExam`, `ValidateExam`, `UpdateExamScope`,
`PublishExam`, `UploadAnswerKey`, `BulkGradeExam`,
`ListQuestionsForGrading`, `ListExamInsights`, `GeneratePrintableExam`,
and `GenerateAnswerKeySheet` (`internal/assessment/service/*.go`) all
took only an `examID`, with `middleware.RequireRole(teacher,
school_admin)` as the *only* gate — any teacher or school_admin
nationwide could read, validate, publish, grade, or print any exam by
guessing/enumerating its UUID. This was a **documented, deliberate
convention at the time**, not an oversight: `ListQuestionsForGrading`'s
old comment read *"no per-teacher-school ownership check, matching the
same convention already used by GetExam/ValidateExam/PublishExam (role
gating alone...)"* — i.e. the gap was known and intentionally repeated
everywhere, exactly the "not just role-based UI hiding" pattern 11.3
asks about. The student-facing side already had the right pattern
(`verifyStudentAccess` in `exam_submit.go`, checking `student.SchoolID
== exam.SchoolID`); `GetExamQuality` (`exam_quality.go`) also already
had it inline. **Fixed**: a shared `verifyCallerOwnsExam` (mirroring
`GetExamQuality`'s existing check) applied uniformly to all ten
functions, backed by a new narrow `repo.ExamSchoolID` lookup that
doesn't touch any of their existing exam-fetch queries or DTOs.
Live-verified against a real stack (two schools, two teachers, one
exam): the owning teacher gets `200`, a teacher from the other school
gets `403 "exam belongs to a different school"` — confirmed on both
`GetExam` and `ListQuestionsForGrading` to prove the shared check
behaves identically across call sites, not just the one it was written
against.

**Finding: `POST /storage/presign-upload`/`presign-download` had no
role check at all**, only "logged in" — any authenticated account,
including a plain `student`, could generate a presigned S3 URL for any
key in any of the four configured buckets (`curriculum`, `exam`,
`reports`, `audit` — see `pkg/config.AWSConfig`), including *write*
access via a presigned PUT. This is real, functional AWS SDK code
(`internal/storage/repository/repository.go`), currently dormant only
because local dev has no AWS credentials configured and nothing in the
active curriculum/exam upload flow calls it (that flow uses
`PostgresStorage`, per `CLAUDE.md`'s dual-storage-adapter note) — not
because the code itself is a stub. **Fixed**: a per-bucket role
allowlist (`bucketRoles` in `internal/storage/service/service.go`) —
curriculum bucket restricted to curriculum_officer/ministry_admin (matching
`/curriculum/upload`'s own gate), exam to teacher/school_admin,
reports/audit to ministry-level roles only.

**Finding: `GET /jobs/{id}` had no ownership check**, while `GET /jobs`
(list) correctly used `ListMine`/`ListByCreator`. Any authenticated
user could read any job's `payload`/`result` by id. **Fixed**:
`service.Get` now checks `job.CreatedBy == callerUserID`, matching
`List`'s existing scoping. This `jobs` domain has zero frontend callers
today (confirmed by grep) — fixed anyway since the code is real and
reachable, not because it's currently exploited in practice.

**Found, deliberately not fixed — flagged for whoever picks this back
up**: `PATCH /jobs/{id}/status` has no role or ownership restriction
at all; any authenticated user can overwrite any job's status/result.
Its own doc comment says it's "called by workers... to report job
progress," but there's no service-account/internal-caller concept
anywhere in this codebase to restrict it to — every caller is a regular
end-user JWT. Properly fixing this needs a real design decision (a
service-role, an internal-only network boundary, or scoping to the
job's own creator) that doesn't exist as a pattern elsewhere in this
codebase yet, and this endpoint has zero current callers (same as the
presign endpoints, confirmed by grep) — inventing a new auth concept
for genuinely dead code isn't a proportionate fix for a checklist item
already this large. Do this before wiring any real feature up to
`jobs`, not after.

**Not audited further, deliberately**: `curriculum.*` endpoints
(`GET/POST /curriculum/jobs/{id}`, `/approve`, prerequisites, versions,
graph) have no per-uploader ownership check beyond
`RequireRole(curriculum_officer, ministry_admin)` — any curriculum
officer can view/approve/edit any other officer's pending submission.
This is **not a gap**: curriculum content is deliberately national and
shared (see [data-integrity.md](data-integrity.md)'s 10.1 section — no
`school_id` on any `curriculum.*` table, by design), and curriculum
review is meant to be a collaborative national workflow across a small
number of officers, not siloed per-uploader. `GET /curriculum/jobs`
(the dashboard list) *is* scoped to "my own jobs" — that's a UX choice
("show me my submissions first"), not a security boundary the
single-item `GET /jobs/{id}` needs to match.

`notification` (`List`/`MarkRead`) was already correctly scoped to
`middleware.UserID(ctx)` end to end — checked, no fix needed.
