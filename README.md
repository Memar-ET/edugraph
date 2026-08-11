# EduGraph AI

**Curriculum-intelligence platform for Ethiopian K–12 education**

EduGraph turns Ministry curriculum documents and classroom exams into a connected knowledge graph, then uses that graph to diagnose *why* a student is struggling — tracing a missed exam question back through its learning outcome, to the topic it tests, to the earlier prerequisite that was never mastered — and to generate a personalized, prerequisite-ordered study plan and an AI tutor grounded in that student's own gaps.

> **Scope note.** What runs today is a **local-dev Docker Compose stack** (the system described here). The national-scale production architecture in `EduGraph AI/*.docx` (Kong, Kubernetes, Vault, S3, AuraDB, etc.) is a design target, not what's in this repo — see [Planning docs vs. reality](#planning-docs-vs-reality).

---

## What works today

### Phase 1 — Curriculum ingestion pipeline
A 4-step pipeline that ingests a curriculum document and promotes it into structured, graph-backed curriculum data:

1. **Upload** — a curriculum officer uploads a PDF/DOCX (role-gated, validated by magic bytes, stored via a swappable `StorageProvider`).
2. **Parse** — a Python worker extracts the Subject → Unit → Topic → CLO tree (PyMuPDF, TOC-first with a font-heuristic fallback).
3. **Human review** — the officer edits the extracted tree in the UI and approves.
4. **Finalize** — one Postgres transaction upserts subjects/units/topics/CLOs, then mirrors Subject→Unit→Topic into Neo4j.

### Phase 2 — Assessment & diagnostics

| Capability | What it does |
|---|---|
| **2A — Exam upload & parsing** | Teacher uploads an exam; a worker parses questions, MCQ options, marks, and aligns each question to a CLO (keyword heuristic, upgraded to a Gemini matcher when configured). |
| **2B — AI validation report** | Before publishing, generates a 5-part report: CLO coverage, topic coverage, Bloom balance, difficulty distribution, and prerequisite warnings. |
| **2C — Submission & grading** | Two flows: students submit digitally (MCQs auto-graded instantly); or a teacher bulk-enters paper results. Attempts finalize when every answer is graded. |
| **3A — Granular gap analysis** | The moment an attempt is fully graded, a 3-pass engine dissects it: **(1)** trace each missed question → CLO → symptom topic, **(2)** walk *backwards* up the prerequisite graph to find the root-cause topic the student never mastered, **(3)** one Gemini call writes a bilingual (English + Amharic) explanation per gap plus an exam narrative. Persisted in three layers: per-gap (`gap_records`), per-exam (`exam_insights`), and rolling per-subject health (`subject_profiles`). |
| **3B — Study plan generator** | Reads the rich gap records (root cause + explanation), **topologically sorts** the fix over the prerequisite graph (foundations before the topics built on them), and packs a day-by-day plan where every day carries the "why". |
| **3C — AI tutor (Graph-RAG)** | When a student asks a question, it checks *their* gap records first, then injects their specific broken prerequisites and mastery into a Gemini prompt so the answer bridges the actual gap rather than answering generically. |

The prerequisite graph that powers 3A/3B/3C is populated via `POST /api/v1/curriculum/topics/{id}/prerequisites` (cycle-checked, mirrored to Neo4j). The whole diagnostic chain activates automatically as soon as prerequisite edges exist — no code changes needed.

---

## Architecture

```
┌───────────┐    REST/JWT    ┌──────────────┐   dual-storage    ┌────────────┐
│ frontend  │ ─────────────► │   api (Go)   │ ────────────────► │  postgres  │  system of record
│ React/Vite│                │  chi router  │                   │  pgvector  │
└───────────┘                └──────┬───────┘                   └────────────┘
                                    │ LPUSH job ids                    ▲
                                    ▼                                  │ MERGE graph
                              ┌──────────┐   BRPOP    ┌─────────────┐  │
                              │  redis   │ ─────────► │ ai-service  │ ─┘
                              │  queues  │            │  (FastAPI)  │ ──► ┌────────┐
                              └──────────┘            │  + workers  │     │ neo4j  │ curriculum + prereq graph
                                                      └──────┬──────┘     └────────┘
                                                             │ HTTPS
                                                             ▼
                                                     Google Gemini (optional)
```

**Services** (`docker-compose.yml`):

| Service | Stack | Port | Role |
|---|---|---|---|
| `api` | Go 1.22 (chi, manual DI) | 8080 | Core REST API, auth, RBAC, orchestration |
| `ai-service` | Python 3.12 + FastAPI | 8000 | Parsing/analysis workers + tutor endpoint |
| `frontend` | React 18 + TS + Vite | 5173 | Web app |
| `postgres` | pgvector/pgvector:pg16 | 5432 | System of record |
| `neo4j` | Neo4j 5 (APOC, GDS) | 7474/7687 | Curriculum + prerequisite knowledge graph |
| `redis` | Redis 7 | 6379 | Job queues + refresh-token store |
| `ollama`, `jaeger`, `prometheus`, `grafana` | — | — | Present in compose; observability/optional |

**Backend** is a Go monolith organized as `internal/<domain>/{handler,service,repository,dto}` per domain (auth, curriculum, assessment, career, student, teacher, school, region, ministry, sync, notification, jobs, storage), with `pkg/` for cross-cutting infra (crypto, storage adapter, middleware, DB clients, the `ai` HTTP client, config, errors, validator).

**AI service** runs its queue consumers as FastAPI lifespan background tasks — plain `asyncio` `BRPOP` loops, **not Celery** (the Go side pushes raw id strings, incompatible with Celery's message protocol). Workers:

| Worker | Queue | Trigger |
|---|---|---|
| `curriculum_worker` | `queue:curriculum:parse` | Curriculum upload |
| `exam_worker` | `queue:exam:parse` | Exam upload (2A) |
| `answer_key_worker` | `queue:exam:answerkey` | Separate answer-key upload |
| `gap_worker` | `queue:gap:analyze` | Attempt fully graded (3A) |
| `study_plan_worker` | `queue:studyplan:generate` | Student requests a plan (3B) |

The tutor (3C) is a synchronous FastAPI route (`POST /api/v1/tutor/ask`), not a queue worker, because it's an interactive chat call.

---

## Key API routes

All under `/api/v1`, JWT-authenticated (RS256), role-gated via `RequireRole` middleware.

**Curriculum**
```
POST   /curriculum/upload                       curriculum_officer, ministry_admin
GET    /curriculum/jobs/{id}                     curriculum_officer, ministry_admin
POST   /curriculum/jobs/{id}/approve             curriculum_officer, ministry_admin
POST   /curriculum/topics/{id}/prerequisites     teacher, ministry_admin, curriculum_officer
GET    /curriculum/topics/{id}/prerequisites
```

**Exams (2A/2B/2C)**
```
POST   /exams/upload                             teacher, school_admin
POST   /exams/{id}/validate                      teacher, school_admin      (2B report)
POST   /exams/{id}/publish                       teacher, school_admin
POST   /exams/{id}/answer-key                    teacher, school_admin
GET    /exams/{id}/questions                     student                    (take exam)
POST   /exams/{id}/submit                        student                    (Flow 1)
POST   /exams/{id}/grades/bulk                   teacher, school_admin      (Flow 2)
GET    /exams/{id}/grading-questions             teacher, school_admin
```

**Diagnostics (3A/3B/3C)**
```
GET    /exams/{id}/my-insight                    student     (narrative + gap records)
GET    /exams/{id}/insights                      teacher, school_admin
GET    /students/me/subject-profiles             student     (subject health)
POST   /students/me/study-plans                  student     (queue 3B generation)
GET    /students/me/study-plans                  student
POST   /tutor/ask                                student     (Graph-RAG tutor)
```

---

## Data model

**Postgres** schemas actually in use: `public` (users, schools, regions, students, teachers, jobs, notifications, refresh_tokens), `curriculum` (upload_jobs, subjects, units, topics, clos, topic_clo_mappings, **topic_prerequisites**), `assessment` (exams, questions, exam_attempts, student_answers), `students` (**gap_records**, **exam_insights**, **subject_profiles**, mastery_records, study_plans), `storage` (local_files — dev BYTEA storage), plus `careers` and `embeddings` (defined, lightly used).

**Neo4j** node/relationship types in use:
```
(:Subject)-[:HAS_UNIT]->(:Unit)-[:HAS_TOPIC]->(:Topic)
(:Topic)-[:HAS_PREREQUISITE]->(:Topic)
```

**Migrations** — Flyway, `backend/db/migrations/V{n}__{description}.sql`, strictly sequential and additive (never edit a merged migration, never `DROP` a column). Current head: **V023** (`gap_analysis`).

---

## Quick start

### Prerequisites
- Docker + Docker Compose
- (For running services outside Docker) Go 1.22, Node.js 20, Python 3.12

### 1. Configure
```bash
cp .env.example .env       # set NEO4J_PASSWORD, and GEMINI_API_KEY if using LLM features
```
`GEMINI_API_KEY` is **optional**. Without it, the CLO matcher and gap analysis fall back to deterministic heuristics/summaries; the AI tutor (3C) returns HTTP 503 (a tutor that can't answer shouldn't pretend to).

### 2. Start the stack
```bash
docker compose up -d --build
```

| Service | URL |
|---|---|
| Backend API | http://localhost:8080 |
| Frontend | http://localhost:5173 |
| AI Service | http://localhost:8000 |
| Neo4j Browser | http://localhost:7474 |

> After editing `docker-compose.yml` env vars, running containers need `docker compose up -d --force-recreate <service>` — env changes don't apply to a running container.

### 3. Run migrations
```bash
docker compose run --rm flyway migrate
```

### 4. Demo users
Seeded by `V015__seed_demo_data.sql`, all with password `password123`:

| Email | Role |
|---|---|
| `curriculum.officer@edugraph.et` | curriculum_officer |
| `teacher@edugraph.et` | teacher |
| `student@edugraph.et` | student |

---

## End-to-end demo (diagnostics)

```bash
# 1. Teacher links a prerequisite so root-cause tracing has a graph to walk
TOKEN=$(curl -s -X POST localhost:8080/api/v1/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"teacher@edugraph.et","password":"password123"}' | jq -r .data.access_token)
curl -s -X POST localhost:8080/api/v1/curriculum/topics/<TOPIC>/prerequisites \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"prerequisiteTopicId":"<PREREQ_TOPIC>","weight":0.9}'

# 2. A graded attempt auto-queues gap analysis (or re-run it manually):
docker compose exec redis redis-cli LPUSH queue:gap:analyze <ATTEMPT_ID>

# 3. Student views the diagnosis, generates a study plan, asks the tutor
STOKEN=$(curl -s -X POST localhost:8080/api/v1/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"student@edugraph.et","password":"password123"}' | jq -r .data.access_token)
curl -s localhost:8080/api/v1/exams/<EXAM_ID>/my-insight -H "Authorization: Bearer $STOKEN"
curl -s -X POST localhost:8080/api/v1/students/me/study-plans -H "Authorization: Bearer $STOKEN" \
  -H 'Content-Type: application/json' -d '{"language":"en"}'
curl -s -X POST localhost:8080/api/v1/tutor/ask -H "Authorization: Bearer $STOKEN" \
  -H 'Content-Type: application/json' -d '{"question":"Why do enzymes speed up respiration?"}'
```

---

## Development notes

- **Backend**: `cd backend && go build ./... && go vet ./...`
- **AI service**: `cd ai-service && python -m compileall app`
- **Frontend**: `npm run type-check`, `npm run lint`, `npm run build` must all pass before frontend work is considered done. (Some tooling/config files have previously been found as empty stubs — verify real content before assuming the scaffold works.)
- **Keep the knowledge graph current**: after changing code, run `graphify update .` (AST-only, no API cost). Query it with `graphify query "<question>"` instead of grepping.

### Security controls (deliberately hardened, currently enforced)
- **RS256 JWT**, 15-min access / 7-day single-use refresh (rotated on `/auth/refresh`); dev keypair auto-generated when `APP_ENV=development` (never commit real keys).
- **bcrypt cost 12** (above the Go default of 10).
- **Magic-byte file validation** server-side (never trusts the `Content-Type` header).
- **RBAC** enforced in Go middleware per route.
- **LLM calls degrade gracefully** — a Gemini failure in a pipeline stage never fails the pipeline; it falls back to a deterministic result.

---

## Planning docs vs. reality

`EduGraph AI/*.docx` describe a national-scale production system (Kong gateway, Kubernetes, HashiCorp Vault, MFA, ElectricSQL, Istio, AWS RDS/S3/ElastiCache, Neo4j AuraDB). **None of that infrastructure exists in this repo and there's no cloud account behind it.** Treat those docs as a security/design checklist to mine for concrete hardening (e.g. bcrypt cost, magic-byte validation — both adopted), not as a build target. The `StorageProvider` interface is the deliberate seam: dev uses Postgres BYTEA, and an `S3StorageProvider` would be a one-file addition later with zero handler/service changes.

The docs are binary — read them with `python-docx` or let `graphify` convert them (needs the `graphifyy[office]` extra).

---

**AASTU Innovation Program · CONFIDENTIAL**
