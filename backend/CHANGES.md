# Changes — Storage / Login / Curriculum Upload

This pass focused on the three features you asked for. Most of the scaffolding
already existed in the zip, but three things were silently broken and would
have failed the moment you tried to actually log in and upload a file. Fixed
all of them; details below.

## 1. Storage interface — already correct, no changes

`pkg/storage/interface.go` defines `StorageProvider` (Upload/Download), and
`pkg/storage/postgres.go` implements it against `storage.local_files`
(BYTEA column) for local dev. `cmd/api/main.go` wires the Postgres
implementation in:

```go
storageProvider := storagepkg.NewPostgresStorage(pgPool)
curriculumService := curriculumsvc.New(curriculumRepo, storageProvider, redisClient)
```

To go to production, swap that one line for an `S3StorageProvider` that
satisfies the same interface — nothing else in the app needs to change.
That provider isn't written yet since you said not to worry about S3 for now.

## 2. Login system — fixed a role gap that blocked the upload flow

The auth code itself (`internal/auth/*`) was already solid: bcrypt password
hashing, RS256-signed access/refresh tokens, refresh-token rotation via
Postgres, dev keypair auto-generation. Nothing was wrong there.

The bug was in the database migrations:

- **Duplicate Flyway versions.** The migrations folder had both a real file
  and an empty placeholder file for versions 11–15
  (e.g. `V011__updated_curriculum.sql` *and* `V011__placeholder.sql`).
  Flyway refuses to run when two migrations share a version number, so
  `docker compose up` would have failed outright on the `flyway` service.
  **Fix:** deleted the six empty placeholders and one redundant duplicate
  (`V017` was an exact re-run of `V013`).

- **Seed data pointed at tables the app never queries.** `V014` created a
  parallel `identity.users` / `regions.regions` / `schools.schools` schema,
  and `V015` seeded demo users into *that* schema. The Go code (and the
  original `V002`/`V003` migrations) only ever read/write the plain
  `users`, `regions`, and `schools` tables. So the seeded demo accounts
  existed in the database but `POST /api/v1/auth/login` could never find
  them.
  **Fix:** rewrote `V014` to just add the missing `curriculum_officer` role
  to the `user_role` enum, and rewrote `V015` to seed one demo user per
  role into the real tables. Also fixed `db/scripts/seed-users.go`, which
  had the same wrong-table bug.

- **`curriculum_officer` wasn't a valid role at all.** The `user_role` enum
  only had `student/teacher/school_admin/regional_admin/ministry_admin`,
  but the router requires `curriculum_officer` (or `ministry_admin`) to hit
  the upload endpoint, and the implementation plan calls for a curriculum
  officer role. Added it via the `V014` migration above and to the
  registration validator (`internal/auth/dto/dto.go`).

**Demo accounts seeded (password `password123` for all):**

| Email | Role |
|---|---|
| ministry.admin@edugraph.et | ministry_admin |
| curriculum.officer@edugraph.et | curriculum_officer |
| regional.admin@edugraph.et | regional_admin |
| school.admin@edugraph.et | school_admin |
| teacher@edugraph.et | teacher |
| student@edugraph.et | student |

Login: `POST /api/v1/auth/login` with `{"email": "...", "password": "password123"}`
→ returns `{ access_token, refresh_token, expires_in, user }`.

## 3. Document upload — fixed two bugs that made it fail on every request

`internal/curriculum/handler/handler.go`'s `Upload` handler had:

1. **Wrong context key.** It read `r.Context().Value("userID")` (a plain
   string), but the auth middleware stores the user id under a typed key
   (`contextkeys.UserIDKey`, via `middleware.UserID(ctx)`). The plain-string
   lookup always returned nothing, so every upload request — even from a
   perfectly valid, authenticated curriculum officer — got a `401 user not
   authenticated`.
2. **Reading the body twice.** It called `r.ParseMultipartForm(...)` (which
   consumes the entire request body) and then tried to
   `json.NewDecoder(r.Body).Decode(&req)` on the same, now-empty body. That
   decode would always fail, so the request never got past parsing.

**Fix:** rewrote the handler to read `subjectCode` / `gradeLevel` /
`academicYear` from the parsed multipart form via `r.FormValue(...)` (there's
no JSON body in a multipart upload), validate them with the existing
`pkg/validator`, and use `middleware.UserID(ctx)` for the authenticated user.
Also switched it to the same `middleware.WriteJSON`/`WriteError` response
envelope the rest of the app uses, instead of hand-rolled `http.Error` JSON.

**How to call it:**

```
POST /api/v1/curriculum/upload
Authorization: Bearer <access_token>   (role must be curriculum_officer or ministry_admin)
Content-Type: multipart/form-data

  file:          <the PDF or DOCX>
  subjectCode:   e.g. "MATH-G10"
  gradeLevel:    e.g. "10"
  academicYear:  e.g. "2026"
```

Response `202 Accepted`:
```json
{ "success": true, "data": { "jobId": "...", "status": "pending", "message": "File uploaded successfully. Parsing queued." } }
```

Behind the scenes: the file bytes go to `storage.local_files` via the
`StorageProvider` (dev mode), a row is created in `curriculum.upload_jobs`,
and the job id is pushed onto the Redis list `queue:curriculum:parse` for
the AI service to pick up (that consumer isn't in this zip — only the
`backend` service was uploaded, no `ai-service` — so the job will sit at
`status: pending` until something pops that queue).

Check status any time with:
```
GET /api/v1/curriculum/jobs/{id}
Authorization: Bearer <access_token>
```

## Everything else

Confirmed nothing else in the codebase referenced the removed
`identity.*` schema. All edited Go files pass a `gofmt` syntax check
(full `go build` wasn't possible in this sandbox — no network access to
the Go module proxy — so please run `docker compose up --build` and watch
the `api` and `flyway` logs on first boot).
