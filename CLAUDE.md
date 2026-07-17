## graphify

This project has a knowledge graph at graphify-out/ with god nodes, community structure, and cross-file relationships.

Rules:
- For codebase questions, first run `graphify query "<question>"` when graphify-out/graph.json exists. Use `graphify path "<A>" "<B>"` for relationships and `graphify explain "<concept>"` for focused concepts. These return a scoped subgraph, usually much smaller than GRAPH_REPORT.md or raw grep output.
- If graphify-out/wiki/index.md exists, use it for broad navigation instead of raw source browsing.
- Read graphify-out/GRAPH_REPORT.md only for broad architecture review or when query/path/explain do not surface enough context.
- After modifying code, run `graphify update .` to keep the graph current (AST-only, no API cost).

## Architecture

EduGraph AI is a curriculum-intelligence platform for Ethiopian K-12 education. **What actually runs today is a local-dev Docker Compose stack** — not the national-scale production system described in `EduGraph AI/edugraph-architecture.docx` (see "Planning docs vs. reality" below).

Services (`docker-compose.yml`):
- **`api`** — Go 1.22+ monolith (chi router, manual DI, no framework). Port 8080.
- **`ai-service`** — Python 3.12 FastAPI service. Port 8000. Runs a curriculum-parsing background worker in the same process (see below).
- **`frontend`** — React 18 + TypeScript + Vite, served by the Vite dev server in dev. Port 5173.
- **`postgres`** — `pgvector/pgvector:pg16`. System of record.
- **`neo4j`** — Curriculum knowledge graph (Subject → Unit → Topic).
- **`redis`** — Job queue between `api` and `ai-service` (curriculum parsing), plus refresh-token storage.
- **`ollama`**, **`jaeger`**, **`prometheus`**, **`grafana`** — present in compose but not load-bearing for the curriculum pipeline; optional/observability.

Backend structure: `internal/<domain>/{handler,service,repository,dto}` per domain (auth, curriculum, assessment, career, student, teacher, school, region, ministry, sync, notification, jobs, storage). `pkg/` holds cross-cutting infra (crypto, storage, middleware, database clients, config, errors, validator).

AI service: **not Celery** despite what `requirements.txt` and older docs imply — the curriculum worker is a plain `asyncio` loop doing Redis `BRPOP` on `queue:curriculum:parse`, started as a FastAPI lifespan background task (`app/main.py`). Rationale documented in `app/workers/curriculum_worker.py`: the Go side does a raw `LPUSH` of a job-id string, not a Celery-protocol message, so a Celery consumer doesn't apply.

## Key Components — The Curriculum Pipeline (Phase 1)

The core feature, defined in `EduGraph AI/Untitled document.docx` ("Improvised Implementation of Phase 1"), is a 4-step pipeline built around a **dual-storage adapter pattern** so the system runs locally without AWS and can swap to S3 later with zero handler/service changes:

```go
// backend/pkg/storage/interface.go
type StorageProvider interface {
    Upload(ctx, fileName, mimeType string, file io.Reader) (ref string, error)
    Download(ctx, ref string) (io.ReadCloser, error)
}
```
Dev implementation: `PostgresStorage` (`backend/pkg/storage/postgres.go`) — stores file bytes in `storage.local_files` (BYTEA), returns the row UUID as the reference. No `S3StorageProvider` exists yet (prod-mode is not implemented, by design).

1. **Upload** (`POST /api/v1/curriculum/upload`, `curriculum` handler/service) — role-gated to `curriculum_officer`/`ministry_admin`, magic-byte validated (not `Content-Type` header — see Critical Decisions), saved via `StorageProvider`, `curriculum.upload_jobs` row created, job id pushed to Redis.
2. **Parse** (`ai-service` curriculum worker) — `BRPOP`s the queue, fetches bytes via the same dual-storage abstraction (`app/db/postgres.py:fetch_file_bytes`), runs PyMuPDF-based extraction (`app/services/curriculum_parser/extractor.py` for PDF, `docx_extractor.py` for DOCX) using a TOC-first / font-heuristic-fallback strategy, writes `parsed_structure` JSONB and flips status to `parsed`.
3. **Human review** (`GET /api/v1/curriculum/jobs/{id}`, `GET /api/v1/storage/files/{jobId}` proxy, `POST /api/v1/curriculum/jobs/{id}/approve`) — frontend (`frontend/src/features/curriculum/JobReviewPage.tsx`) renders an editable unit/topic tree, lets the officer view the original file via a blob-fetch (not a plain `<a href>`, since the proxy endpoint needs a Bearer token a browser can't attach to a navigation), and submits edits or approves as-is.
4. **Finalize** (`ApproveAndPromote` in `backend/internal/curriculum/repository/repository.go`) — one Postgres transaction upserts `curriculum.subjects/units/topics/clos`, then (outside the transaction, best-effort) mirrors Subject→Unit→Topic into Neo4j via `MERGE`. CLO nodes are **not yet mirrored to Neo4j** — deferred to Phase 2 by design (matches `edugraph-impl-plan.docx` Capability 1C, not yet built).

## Important Files

| Area | Path |
|---|---|
| Backend entrypoint / DI wiring | `backend/cmd/api/main.go` |
| Route table + RBAC | `backend/cmd/api/router.go` |
| Storage adapter | `backend/pkg/storage/{interface,postgres}.go` |
| Curriculum domain (Steps 1/3/4) | `backend/internal/curriculum/{handler,service,repository}/*.go` |
| Auth (JWT RS256 + refresh rotation) | `backend/internal/auth/*`, `backend/pkg/crypto/crypto.go` |
| RBAC / error / response helpers | `backend/pkg/middleware/middleware.go`, `backend/pkg/errors/errors.go` |
| AI curriculum worker (Step 2) | `ai-service/app/workers/curriculum_worker.py`, `ai-service/app/services/curriculum_parser/{service,extractor,docx_extractor}.py` |
| Frontend API client (JWT refresh interceptor) | `frontend/src/lib/api/client.ts` |
| Frontend curriculum UI (Steps 1/3) | `frontend/src/features/curriculum/{UploadPage,JobReviewPage}.tsx` |
| Frontend routing/auth guards | `frontend/src/app/router.tsx`, `frontend/src/stores/auth.store.ts` |
| DB migrations | `backend/db/migrations/V{n}__{description}.sql` (Flyway) |
| Neo4j migrations | `backend/db/neo4j/migrations/{n}_{description}.cypher` |
| Local dev orchestration | `docker-compose.yml`, `.env` (from `.env.example`) |
| Planning docs (binary, need conversion to read) | `EduGraph AI/*.docx` |

## Database Structure

**Postgres** (verified live, not the aspirational 14-schema design in `edugraph-db-design.docx`) — actual schemas: `public` (users, schools, regions, students, teachers, jobs, notifications, refresh_tokens, sync_logs), `curriculum` (upload_jobs, subjects, units, topics, clos, topic_clo_mappings, topic_prerequisites), `storage` (local_files — BYTEA dev storage), `assessment`, `careers`, `embeddings`, `students`. No RLS, no per-schema partitioning, no pgvector usage yet despite the base image including it.

Key table: `curriculum.upload_jobs` — `status` progresses `pending → parsing → parsed → review → approved | rejected | failed`; `file_s3_key` holds the Postgres `storage.local_files` UUID in dev mode (name is legacy from the S3-first design); `parsed_structure` JSONB holds the AI-extracted tree; `neo4j_written` flags whether the graph mirror succeeded.

**Neo4j** — actual node/relationship types in use: `(:Subject)-[:HAS_UNIT]->(:Unit)-[:HAS_TOPIC]->(:Topic)`. CLO, prerequisite, and student/assessment graph structures described in the planning docs do not exist yet.

**Redis** — one queue in active use: `queue:curriculum:parse` (plain string job-id list, `LPUSH`/`BRPOP`). Also used for refresh-token storage (`internal/auth`).

Roles (Postgres `users.role`, enforced via Go `RequireRole` middleware — not Postgres RLS): `student`, `teacher`, `school_admin`, `regional_admin`, `ministry_admin`, `curriculum_officer`. Seed demo users exist per `V015__seed_demo_data.sql`, e.g. `curriculum.officer@edugraph.et` / `password123`.

## Development Rules

- **Local dev**: `docker compose up -d`. If containers already exist from a prior session, `docker compose up -d --force-recreate <service>` after editing `docker-compose.yml` — env var changes don't apply to a running container, only a recreate.
- **Migrations**: Flyway naming `V{n}__{description}.sql` in `backend/db/migrations/`, strictly sequential, never edit a merged migration — add a new one. Never `DROP` a column; additive only.
- **Auth**: RS256 JWT, 15-min access / 7-day refresh (single-use, rotated on `/auth/refresh`). Dev keypair auto-generated by `crypto.EnsureDevKeyPair` when `APP_ENV=development` — never commit real keys.
- **File uploads**: validate by magic bytes server-side, never trust `Content-Type`. See `sniffCurriculumMime` in `backend/internal/curriculum/handler/handler.go`.
- **Frontend**: `npm run type-check`, `npm run lint`, `npm run build` must all pass clean before considering frontend work done — `node_modules` and several config files (`tsconfig.node.json`, `.eslintrc.cjs`, `postcss.config.js`, `vite.config.ts`, etc.) have previously been found as **empty stub files** despite looking present; verify they have real content before assuming the scaffold works.
- **Don't build unused production infra.** Kong, Kubernetes, HashiCorp Vault, MFA, ElectricSQL, Istio — all described in `edugraph-architecture.docx` — do not exist in this repo and there's no cloud account to run them against. Treat that doc as a security/design checklist to mine for concrete hardening (e.g. bcrypt cost, magic-byte validation), not a build target, unless explicitly asked to provision real infra.
- **Reading the `EduGraph AI/*.docx` planning docs**: they're binary — use `python-docx` (`import docx; docx.Document(path)`) to extract text, or let `graphify` convert them (`graphifyy[office]` extra required — without it they're silently skipped).

## Critical Decisions

- **Dual-storage adapter over direct S3** (`Untitled document.docx`): deliberate choice to develop against Postgres BYTEA locally with a swappable `StorageProvider` interface, so production S3 support is a one-file addition later, not a rewrite.
- **BRPOP consumer, not Celery**, for the curriculum worker — the Go producer pushes a raw string via `LPUSH`, incompatible with Celery's message protocol; documented inline in `curriculum_worker.py`.
- **CLO Neo4j sync deferred to Phase 2** — `ApproveAndPromote` only mirrors Subject/Unit/Topic; this is intentional scope, not a bug (matches the commit history and `edugraph-impl-plan.docx`'s phase ordering).
- **bcrypt cost 12** (not the Go default of 10) and **magic-byte file validation** (not `Content-Type` header) were hardened deliberately against `edugraph-architecture.docx`'s stated minimums — both are real, currently-enforced controls, not aspirational.
- **Frontend was empty scaffolding until 2026-07-17** — full tooling was present (React, TanStack Router/Query, Zustand, Tailwind, Radix, etc.) but zero implementation existed; login/upload/review pages were built from scratch that session. If frontend files look suspiciously sparse, check file size before assuming they're implemented.
- **`docker-compose.yml`'s `VITE_API_URL`/`VITE_AI_URL` previously pointed at the AI service instead of the Go API** — fixed to a Vite dev-server proxy pattern (`API_PROXY_TARGET`, resolved per-environment in `vite.config.ts`) so the frontend always reaches the Go API regardless of whether it's running via `npm run dev` or inside Docker Compose.
