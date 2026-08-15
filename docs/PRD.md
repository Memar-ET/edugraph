# EduGraph AI — Product Requirements Document

**Date:** 2026-08-15  
**Status:** Living document — reflects what is actually built and shipped today.  
**Scope:** Local-dev Docker Compose stack + Supabase (hosted Postgres). This is NOT the national-scale production deployment described in the planning docs.

---

## 1. Product Overview

EduGraph AI is a curriculum-intelligence platform for Ethiopian K-12 education. It connects the MoE curriculum graph (subjects → units → topics → CLOs with typed prerequisite edges) to student assessment data and AI-driven learning analytics.

**What it does today:**

1. **Curriculum ingestion** — curriculum officers upload PDF/DOCX files; an AI parser extracts structured units/topics/CLOs; an officer reviews and approves; the knowledge graph is mirrored to Neo4j.
2. **Exam management** — teachers upload and parse exams; students take them online; grading is automated (MCQ) or teacher-reviewed (open); per-question CLO alignment is tracked.
3. **Gap analysis** — after every submission, a 3-pass AI pipeline identifies weak prerequisite topics and generates a bilingual (EN/AM) insight narrative.
4. **Study plans** — topologically sorted, day-packed study plans generated from a student's gap records and prerequisite graph.
5. **Graph-RAG Tutor** — a student asks a question; the system matches it to relevant topics, injects the student's own gap records and prerequisite chain into the Gemini prompt, and returns a contextual answer.
6. **EG-GCKT Knowledge Tracing** — BKT/DINA/IRT analytical engines fused into a Student Knowledge State Graph (SKSG); nightly parameter refit; misconception modeling; next-best-action ranking; explainability endpoint.
7. **Class heatmap** — teachers see a Neo4j-backed heatmap of topics their class is struggling with, with cross-grade prerequisite alerts.
8. **School quality scores** — nightly composite score (CLO coverage, mastery, exam quality, compliance); Redis-cached 1 h.

**What it is NOT:**

- Not a national-scale deployment (no Kong, Kubernetes, Vault, or MFA)
- Not a closed-loop adaptive testing system (humans review AI recommendations)
- Not real-time (all AI pipelines are async via Redis queues)

---

## 2. User Roles

Six roles enforced by Go `RequireRole` middleware (Postgres `users.role` enum, not Postgres RLS).

### 2.1 `student`

| Capability | Endpoint |
|---|---|
| View own exam list and take exams | `GET /exams/{id}/questions`, `POST /exams/{id}/submit` |
| View gap-analysis insight for own submissions | `GET /exams/{id}/my-insight` |
| View own subject profiles | `GET /students/me/subject-profiles` |
| Generate and list own study plans | `POST /students/me/study-plans`, `GET /students/me/study-plans` |
| Ask the Graph-RAG tutor | `POST /tutor/ask` |
| View own career matches | `GET /students/me/career/matches` ⚠️ broken |
| View own skill-state explanation | `GET /students/{id}/topics/{topicId}/explain` |
| View own skill-state snapshots | `GET /students/{id}/topics/{topicId}/state-snapshots` |

### 2.2 `teacher`

| Capability | Endpoint |
|---|---|
| Upload and manage exams | `POST /exams/upload`, `GET /exams/{id}`, `POST /exams/{id}/validate`, `PATCH /exams/{id}/scope`, `POST /exams/{id}/publish` |
| Upload answer keys | `POST /exams/{id}/answer-key` |
| Bulk grade exams | `POST /exams/{id}/grades/bulk` |
| View all student insights for an exam | `GET /exams/{id}/insights` |
| View exam quality report | `GET /exams/{id}/quality` |
| Print exam / answer key | `GET /exams/{id}/print`, `GET /exams/{id}/print/answer-key` |
| View class-wide gap heatmap | `GET /teachers/me/class-heatmap` |
| Add / validate prerequisite links | `POST /curriculum/topics/{id}/prerequisites`, `PATCH /curriculum/topics/{id}/prerequisites/{prereqId}/validate` |
| Review misconception hypotheses | `GET /misconceptions/`, `PATCH /misconceptions/{id}/review` |
| List / get students and teachers | standard CRUD (same-school scoped) |

### 2.3 `school_admin`

All teacher capabilities, plus:

| Capability | Endpoint |
|---|---|
| Create / update / delete students and teachers | student/teacher CRUD |
| View school quality scores | `GET /schools/{id}/quality-scores` |
| Create notifications | `POST /notifications/` |

### 2.4 `regional_admin`

| Capability | Endpoint |
|---|---|
| View regional overview and stats | `GET /ministry/overview`, `GET /ministry/regions/{regionID}/stats` |
| Create / delete schools | `POST /schools/`, `DELETE /schools/{id}` |
| View school quality scores | `GET /schools/{id}/quality-scores` |
| List students and teachers (region-scoped server-side) | read-only |

### 2.5 `ministry_admin`

All regional_admin capabilities, plus:

| Capability | Endpoint |
|---|---|
| Manage regions (CRUD) | `/regions/` |
| Manage all schools | `/schools/` full CRUD |
| Upload and approve curriculum | `/curriculum/upload`, `/curriculum/jobs/{id}/approve` |
| Resync prerequisites to Neo4j | `POST /curriculum/prerequisites/resync` |
| Link curriculum versions (supersede) | `POST /curriculum/subjects/{code}/supersede` |
| View all subjects | `GET /curriculum/subjects` |
| Promote / reject model snapshots | `POST /model-snapshots/{id}/promote`, `POST /model-snapshots/{id}/reject` |
| Create career paths | `POST /career/paths` |

### 2.6 `curriculum_officer`

| Capability | Endpoint |
|---|---|
| Upload curriculum files | `POST /curriculum/upload` |
| View own upload job history | `GET /curriculum/jobs` |
| Review and approve parsed curriculum | `GET /curriculum/jobs/{id}`, `POST /curriculum/jobs/{id}/approve` |
| Add prerequisite links | `POST /curriculum/topics/{id}/prerequisites` |
| View curriculum graph | `GET /curriculum/subjects/{code}/graph` |
| View Q-matrix and prerequisite quality reports | `GET /curriculum/subjects/{code}/qmatrix-quality`, `.../prerequisite-quality` |
| Link curriculum versions | `POST /curriculum/subjects/{code}/supersede` |

---

## 3. Core Features

### 3.1 Curriculum Pipeline

**Flow:** Upload → AI parse → human review → approve → Neo4j mirror

1. Officer uploads a PDF or DOCX via `POST /curriculum/upload`. The Go handler validates by magic bytes (not `Content-Type`), saves via the `StorageProvider` (Postgres BYTEA in dev), creates an `upload_jobs` row, and pushes the job id to `queue:curriculum:parse`.
2. The ai-service `curriculum_worker` BRPOPs the queue, fetches the file, runs PyMuPDF (PDF) or python-docx (DOCX) extraction with a TOC-first / font-heuristic fallback. Supports the "Unified ID Convention" format (`G{grade}.U{unit}.T{topic}`) used by the real MoE Biology curriculum.
3. The officer reviews the extracted unit/topic/CLO tree in `JobReviewPage.tsx` and can edit before approving.
4. `ApproveAndPromote` upserts `curriculum.subjects/units/topics/clos` in one Postgres transaction, then (best-effort) mirrors to Neo4j.
5. Curriculum versioning: a mid-year revision is promoted as a new subject code and linked via `POST /curriculum/subjects/{newCode}/supersede`.

**Known issue:** `ApproveAndPromote` takes ~198 s for a full Biology grade over Supabase. The Vite dev proxy times out and returns a false 500; the backend write completes correctly. Fix pending (batch inserts).

### 3.2 Exam Management

Teachers upload exams; the ai-service `exam_worker` extracts questions. A separate `answer_key_worker` aligns answer keys to CLOs. Students submit via `POST /exams/{id}/submit` with per-answer `timeSpentSecs`. MCQ grading is automatic; open-answer bulk grading is teacher-driven. Submissions push `queue:gap:analyze` and `queue:gckt:trace`.

### 3.3 Gap Analysis Engine

Three-pass async pipeline triggered on each exam submission:

1. **Triage** — identify weak topics (mastery < threshold).
2. **Root-cause walk** — traverse the prerequisite graph via RCS scoring (`Weakness × EvidenceConfidence × DownstreamImpact × PrerequisiteReadiness × InterventionGain`).
3. **LLM narrative** — one Gemini call per attempt, bilingual EN/AM summary.

Output: `students.gap_records`, `students.exam_insights`, `students.subject_profiles`.

**Read endpoints:** `GET /exams/{id}/my-insight`, `GET /exams/{id}/insights`, `GET /students/me/subject-profiles`.

### 3.4 Study Plan Generator

`POST /students/me/study-plans` triggers `queue:studyplan:generate`. The ai-service `study_plan_worker` runs:

1. Kahn's algorithm topological sort over prerequisite edges (Neo4j, Postgres fallback).
2. Root-cause-before-symptom ordering using RCS scores.
3. **6-factor next-best-action ranking**: expected learning gain, information gain (high-uncertainty topics), prerequisite impact, difficulty fit (IRT `p_correct` in 0.5–0.7 ZPD band), forgetting/recency need, repetition penalty.
4. Greedy day-packing by estimated hours.

Output: `students.study_plans` with day-by-day blocks and explanatory "why" derived from gap root causes.

### 3.5 Graph-RAG Tutor

`POST /tutor/ask` — the only ai-service capability exposed as a real HTTP route. The ai-service `tutor/service.py` keyword-matches the question to topics, injects the student's gap records and prerequisite chain + mastery into the Gemini prompt, and returns a structured answer.

**Requirement:** a `GEMINI_API_KEY` must be configured. Without it the endpoint returns HTTP 503.

### 3.6 EG-GCKT Knowledge Tracing

Full EG-GCKT implementation (12 milestones, 2026-08-15):

- **BKT** — 4-parameter Bayesian Knowledge Tracing, online per-response forward-pass.
- **DINA** — DINA joint posterior over 2^k attribute patterns, conjunctive multi-skill model.
- **IRT** — 2PL logistic ability estimation via Newton-Raphson MLE.
- **GCSF fusion** — weighted average of BKT/DINA/IRT/graph-reasoning evidence; disagreement widens uncertainty; recency-decay weighting.
- **SKSG** — `students.skill_states`, cold start via row absence (never a fabricated 0.0).
- **Nightly refit** — `refit_worker.py` asyncio periodic loop; BKT/DINA/IRT grid-search MLE; writes candidate `model_snapshots`, never auto-activates.
- **Misconception modeling** — candidate hypotheses from LLM, teacher confirm/reject.
- **Blocked-learning recovery** — 3 consecutive failures trigger routing via `similar_to`/`alternative_to` prerequisite edges.
- **Consistency checking** — detects strong-topic/weak-prerequisite patterns, penalizes edge confidence.
- **Explainability** — `GET /students/{id}/topics/{topicId}/explain` returns 5-part explanation (current state, evidence, structural context, confidence, recommendation).
- **Model governance** — ministry_admin reviews candidate snapshots at `GET /model-snapshots/candidates`.

**DKT and MIRT are deliberately not implemented** — the spec's Phase 6 criterion requires real longitudinal data that doesn't exist yet.

### 3.7 Class Heatmap

`GET /teachers/me/class-heatmap` — Neo4j query over `(:School)-[:ENROLLS]->(:Student)-[:STRUGGLED_WITH]->(:Topic)`. Cross-grade alert fires when >40% of a class struggles with a topic that has downstream dependents via `HAS_PREREQUISITE*1..3` (filtered to `requires`/`strongly_requires` edges).

### 3.8 School Quality Score

`GET /schools/{id}/quality-scores` — composite of CLO coverage, mastery, exam quality, compliance. Redis-cached 1 h (`school_quality:{schoolId}`). Nightly recompute via Go `qualityworker` goroutine (90 s initial delay + 24 h ticker). Written to `schools.quality_scores` schema (distinct from `public.schools` table).

### 3.9 Curriculum Graph Visualization

`GET /curriculum/subjects/{code}/graph?includeClos=true` — queries Neo4j directly for Subject→Unit→Topic[→Subtopic][→CLO] subtree plus every `HAS_PREREQUISITE` edge, including cross-grade external nodes. Frontend at `/curriculum/subjects/{code}/graph` renders with `reactflow` + `dagre`. "Show subtopics" toggle (default off) keeps initial view readable.

---

## 4. Known Gaps / Not Built

| Feature | Status | Notes |
|---|---|---|
| Career matching | ⚠️ Broken | `ai-service/app/api/v1/routes/career.py` never registered in `main.py`; `career_matcher/service.py` is a 0-line stub. Calling `/students/me/career/generate` returns HTTP 404 from ai-service. |
| DKT (Deep Knowledge Tracing) | Deferred | Spec §21 Phase 6 — requires longitudinal data that doesn't exist. `model_snapshots.model_type` reserves `dkt_model`. |
| MIRT calibration | Deferred | Same rationale. Reserves `mirt_calibration`. |
| Closed-loop adaptive item delivery | Deferred | Consistency detection exists; fully automated adaptive testing is out of scope. |
| Real embedding model | ⚠️ Stub only | `StubEmbeddingProvider` writes deterministic hash-based 768-dim vectors; cosine similarity is not semantically meaningful. Gemini vs. Ollama undecided. |
| `ApproveAndPromote` latency | ⚠️ Known bug | ~198 s for a full grade over Supabase; Vite proxy times out → false 500. Backend write is correct. Fix: batch inserts (not yet done). |
| Empty ai-service stubs | ⚠️ Dead code | `app/api/v1/routes/{policy,plan,career,exam,gap}.py`, `app/services/{policy_insight,gap_engine,study_planner}/service.py`, `app/workers/report_worker.py` — all 0 lines. |

---

## 5. Data Model Summary

### Postgres Schemas (live on Supabase)

| Schema | Key Tables |
|---|---|
| `public` | `users`, `schools`, `regions`, `students`, `teachers`, `jobs`, `notifications`, `refresh_tokens`, `sync_logs` |
| `curriculum` | `upload_jobs`, `subjects`, `units`, `topics`, `clos`, `topic_clo_mappings`, `topic_prerequisites` (typed edges V033), `prerequisite_review_history`, `node_review_history` |
| `app_storage` | `local_files` (BYTEA dev storage — named `app_storage` to avoid collision with Supabase's `storage` schema) |
| `assessment` | `exams`, `questions`, `question_options`, `student_answers`, `exam_quality_reports`, `item_skill_mappings` (versioned Q-matrix V034) |
| `careers` | `career_paths`, `career_matches` |
| `embeddings` | `clo_embeddings`, `topic_embeddings` (stub provider), `question_embeddings` (unused) |
| `students` | `gap_records`, `exam_insights`, `subject_profiles`, `study_plans`, `skill_states` (SKSG), `learning_events`, `misconception_hypotheses`, `recommendation_log`, `recovery_attempts`, `skill_state_snapshots` |
| `schools` | `quality_scores` (separate schema from `public.schools`) |
| `modeling` | `model_snapshots`, `evidence_log` |

### Neo4j Node/Relationship Types

```
(:Subject)-[:HAS_UNIT]->(:Unit)-[:HAS_TOPIC]->(:Topic)
(:Topic)-[:HAS_SUBTOPIC]->(:Topic)
(:Topic)-[:HAS_PREREQUISITE {edgeType, confidence}]->(:Topic)
(:Topic)-[:HAS_CLO]->(:CLO)
(:School)-[:ENROLLS]->(:Student)-[:STRUGGLED_WITH {severity, isRootCause}]->(:Topic)
(:Teacher)-[:TEACHES_AT]->(:School)
```

---

## 6. API Surface (summary)

Full documentation: `docs/api.md`.

| Domain | Base Path |
|---|---|
| Auth | `/api/v1/auth` |
| Curriculum | `/api/v1/curriculum` |
| Exams | `/api/v1/exams` |
| Students | `/api/v1/students` |
| Teachers | `/api/v1/teachers` |
| Schools | `/api/v1/schools` |
| Regions | `/api/v1/regions` |
| Ministry | `/api/v1/ministry` |
| Tutor | `/api/v1/tutor` |
| Career | `/api/v1/career` |
| Questions (Q-matrix) | `/api/v1/questions` |
| Misconceptions | `/api/v1/misconceptions` |
| Model Snapshots | `/api/v1/model-snapshots` |
| Notifications | `/api/v1/notifications` |
| Storage | `/api/v1/storage` |
| Sync (School-Box) | `/api/v1/sync` |
