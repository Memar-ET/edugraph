# EduGraph API Reference

**Base URL:** `http://localhost:8080` (local dev) / `https://<cloud-api-host>` (production)  
**Version:** v1 — all paths are prefixed `/api/v1/`  
**Auth:** RS256 JWT, `Authorization: Bearer <access_token>` header. Access tokens expire in 15 minutes; use `/auth/refresh` to rotate.

---

## Authentication

### `POST /api/v1/auth/register`
Create a new user account.

**Body:**
```json
{ "email": "user@school.et", "password": "...", "role": "student", "schoolId": "uuid" }
```
Valid roles: `student`, `teacher`, `school_admin`, `regional_admin`, `ministry_admin`, `curriculum_officer`.

**Response `201`:**
```json
{ "data": { "access_token": "...", "refresh_token": "..." } }
```

---

### `POST /api/v1/auth/login`
**Body:** `{ "email": "...", "password": "..." }`  
**Response `200`:** `{ "data": { "access_token": "...", "refresh_token": "..." } }`

---

### `POST /api/v1/auth/refresh`
Rotate a refresh token (single-use; the old token is invalidated on use).

**Body:** `{ "refresh_token": "..." }`  
**Response `200`:** `{ "data": { "access_token": "...", "refresh_token": "..." } }`

---

### `POST /api/v1/auth/logout`
Invalidate the current refresh token. No body required.  
**Response `204`**

---

### `GET /api/v1/auth/me`
Returns the current user's profile.  
**Roles:** any authenticated  
**Response `200`:** `{ "data": { "id": "...", "email": "...", "role": "...", "schoolId": "..." } }`

---

## Sync (School-Box)

School-Box devices authenticate with a device secret rather than a JWT.

### `POST /api/v1/sync/push`
Push local outbox rows from a School-Box to the cloud.  
**Auth:** `X-Device-Secret: <secret>` header  
**Body:** Array of outbox rows  

### `GET /api/v1/sync/pull`
Pull cloud changes for this School-Box.  
**Auth:** `X-Device-Secret: <secret>` header

---

## Regions

**Roles:** All authenticated users may read; `ministry_admin` only for create/update/delete.

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/regions` | List all regions |
| `GET` | `/api/v1/regions/{id}` | Get one region |
| `POST` | `/api/v1/regions` | Create region (ministry_admin) |
| `PATCH` | `/api/v1/regions/{id}` | Update region (ministry_admin) |
| `DELETE` | `/api/v1/regions/{id}` | Delete region (ministry_admin) |

---

## Schools

**Roles:** Read: any authenticated; Create/Delete: `ministry_admin` or `regional_admin`; Update: `regional_admin` or `school_admin`.

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/schools` | List schools (scoped to caller's region for regional_admin) |
| `GET` | `/api/v1/schools/{id}` | Get one school |
| `POST` | `/api/v1/schools` | Create school |
| `PATCH` | `/api/v1/schools/{id}` | Update school |
| `DELETE` | `/api/v1/schools/{id}` | Delete school |
| `GET` | `/api/v1/schools/{id}/quality-scores` | Composite quality score per subject+grade (school_admin, regional_admin, ministry_admin) |

**Quality score response:**
```json
{
  "data": {
    "schoolId": "uuid",
    "scores": [
      { "subjectCode": "BIO-G9", "gradeLevel": 9, "cloScore": 0.72, "masteryScore": 0.68, "qualityScore": 0.80, "complianceScore": 0.90, "composite": 0.77 }
    ],
    "cachedAt": "2026-08-15T00:00:00Z"
  }
}
```
Scores are cached in Redis for 1 hour.

---

## Students

**Roles:** List/Get: `teacher`, `school_admin`, `regional_admin`, `ministry_admin`; Create/Update: `school_admin`, `teacher`; Delete: `school_admin`.  
Results are scoped server-side to the caller's school or region.

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/students` | List students |
| `GET` | `/api/v1/students/{id}` | Get one student |
| `POST` | `/api/v1/students` | Create student |
| `PATCH` | `/api/v1/students/{id}` | Update student |
| `DELETE` | `/api/v1/students/{id}` | Delete student |
| `GET` | `/api/v1/students/me/subject-profiles` | Student's own subject health profiles (student only) |
| `POST` | `/api/v1/students/me/study-plans` | Generate study plan from gap records (student only) |
| `GET` | `/api/v1/students/me/study-plans` | List student's study plans (student only) |
| `POST` | `/api/v1/students/me/career/generate` | Trigger career match generation (student only, **currently broken — ai-service route unwired**) |
| `GET` | `/api/v1/students/me/career/matches` | Get career matches (student only) |
| `GET` | `/api/v1/students/{id}/topics/{topicId}/explain` | EG-GCKT five-part explanation of a student's mastery state. IDOR-safe: student may only view their own; teacher/school_admin may view same-school students; ministry_admin unrestricted. |
| `GET` | `/api/v1/students/{id}/topics/{topicId}/state-snapshots` | Historical skill-state snapshots (same authorization as Explain) |

**Subject profile response:**
```json
{
  "data": [
    { "subjectCode": "BIO-G9", "masteryPercent": 0.65, "weakTopics": ["Photosynthesis", "Cell Division"], "strongTopics": ["Genetics"] }
  ]
}
```

**Study plan body:**
```json
{ "subjectCode": "BIO-G9", "targetDays": 30 }
```

**Explain response:**
```json
{
  "data": {
    "currentState": { "masteryProbability": 0.42, "uncertainty": 0.18, "trend": "improving" },
    "evidence": [ { "provenance": "bkt", "estimate": 0.40, "reliability": 0.9 } ],
    "structuralContext": { "prerequisitesSatisfied": true, "weakPrerequisiteCount": 1 },
    "confidence": "medium",
    "recommendation": { "action": "practice", "reason": "..." }
  }
}
```

---

## Teachers

**Roles:** List/Get: `teacher`, `school_admin`, `regional_admin`, `ministry_admin`; Create/Update/Delete: `school_admin`.

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/teachers` | List teachers |
| `GET` | `/api/v1/teachers/{id}` | Get one teacher |
| `POST` | `/api/v1/teachers` | Create teacher |
| `PATCH` | `/api/v1/teachers/{id}` | Update teacher |
| `DELETE` | `/api/v1/teachers/{id}` | Delete teacher |
| `GET` | `/api/v1/teachers/me/class-heatmap` | Class-wide gap heatmap for the caller's class (teacher, school_admin) |

**Heatmap response:**
```json
{
  "data": {
    "topics": [
      { "topicId": "uuid", "topicName": "Cell Division", "severity": 0.78, "studentCount": 12, "isRootCause": true, "crossGradeAlert": true }
    ]
  }
}
```
`crossGradeAlert: true` means >40% of the class struggles with this topic and it is a prerequisite for topics in another grade.

---

## Ministry / Regional Analytics

**Roles:** `ministry_admin`, `regional_admin`; `CurriculumInsights` is `ministry_admin` only.

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/ministry/overview` | National overview (total students, schools, regions, avg mastery) |
| `GET` | `/api/v1/ministry/regions/{regionID}/stats` | Stats for one region |
| `GET` | `/api/v1/ministry/regions/{regionID}/underperforming` | Ranked underperforming schools with weak-topic breakdown (6.1) |
| `POST` | `/api/v1/ministry/curriculum-insights` | Generate AI curriculum insight narrative (ministry_admin, 6.2) |

**Underperforming schools response:**
```json
{
  "data": {
    "regionId": "uuid",
    "schools": [
      {
        "schoolId": "uuid",
        "schoolName": "...",
        "averageMastery": 0.41,
        "weakestTopics": [
          { "topicId": "uuid", "topicName": "Photosynthesis", "avgMastery": 0.28, "affectedStudents": 34 }
        ]
      }
    ]
  }
}
```

**Curriculum insights body:**
```json
{ "subjectCode": "BIO-G9", "gradeLevel": 9 }
```
**Response:** `{ "data": { "insight": "Narrative text...", "generatedAt": "2026-08-15T...", "modelUsed": "gemini-..." } }`  
Degrades gracefully: if no Gemini key is configured, returns `{ "data": { "insight": null, "error": "llm_unavailable" } }` (not a 503).

---

## Curriculum

**Roles:** Most write endpoints: `curriculum_officer`, `ministry_admin`. Read endpoints: any authenticated.

### Upload & Parse Pipeline

| Method | Path | Roles | Description |
|---|---|---|---|
| `POST` | `/api/v1/curriculum/upload` | curriculum_officer, ministry_admin | Upload PDF or DOCX curriculum file. Magic-byte validated. Starts parse job asynchronously. |
| `GET` | `/api/v1/curriculum/jobs` | curriculum_officer, ministry_admin | List caller's upload jobs (paginated) |
| `GET` | `/api/v1/curriculum/jobs/{id}` | curriculum_officer, ministry_admin | Get one job with `parsed_structure` |
| `POST` | `/api/v1/curriculum/jobs/{id}/approve` | curriculum_officer, ministry_admin | Approve extracted content, promote to curriculum tables, mirror to Neo4j |

**Upload:** `multipart/form-data` with fields `file`, `subjectCode`, `gradeLevel`, `academicYear`.

**Job status lifecycle:** `pending → parsing → parsed → review → approved | rejected | failed`

**Approve body:** The `parsed_structure` JSON (possibly edited) from the job, same shape as returned by `GET /jobs/{id}`.

> **Warning:** Approving a large subject (>50 topics) may take 2–5 minutes over Supabase. The Vite dev proxy may surface this as a 500. See `docs/runbooks/approve-promote-slow.md`.

### Subjects & Versioning

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/curriculum/subjects` | List all approved subjects (ministry_admin, curriculum_officer) |
| `GET` | `/api/v1/curriculum/subjects/{code}/topics` | Flat topic list for a subject (backs the prerequisites UI topic picker) |
| `GET` | `/api/v1/curriculum/subjects/{code}/versions` | Version lineage for a subject code (oldest-first) |
| `POST` | `/api/v1/curriculum/subjects/{code}/supersede` | Link a new subject as superseding an old one (curriculum_officer, ministry_admin) |

**Supersede body:** `{ "previousCode": "BIO-G9-2025" }`

### Prerequisite Graph

| Method | Path | Roles | Description |
|---|---|---|---|
| `POST` | `/api/v1/curriculum/topics/{id}/prerequisites` | ministry_admin, teacher, curriculum_officer | Add a prerequisite edge |
| `GET` | `/api/v1/curriculum/topics/{id}/prerequisites` | any authenticated | List prerequisite edges for a topic |
| `PATCH` | `/api/v1/curriculum/topics/{id}/prerequisites/{prereqId}/validate` | ministry_admin, teacher, curriculum_officer | Confirm an `ai_inferred` edge |
| `GET` | `/api/v1/curriculum/topics/{id}/prerequisites/{prereqId}/history` | any authenticated | Append-only review history for one edge |
| `POST` | `/api/v1/curriculum/prerequisites/resync` | ministry_admin | Bulk re-mirror all `neo4j_written=false` rows to Neo4j |

**Add prerequisite body:**
```json
{
  "prerequisiteTopicId": "uuid",
  "edgeType": "requires",
  "weight": 1.0,
  "confidence": 0.9,
  "evidence": "MoE Biology syllabus page 12",
  "inferMethod": "moe_document"
}
```
Valid `edgeType` values: `requires`, `strongly_requires`, `recommended_before`, `related_to`, `similar_to`, `supports`, `alternative_to`.  
Valid `inferMethod` values: `manual`, `explicit`, `moe_document` (auto-validated), `ai_inferred` (left unvalidated until PATCH /validate).

### Knowledge Graph Visualization

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/curriculum/subjects/{code}/graph` | Neo4j subgraph for a subject (Subject→Unit→Topic[→Subtopic][→CLO] + HAS_PREREQUISITE edges) |

**Query params:** `includeClos=true` (default false — CLO count is several times the topic count for large subjects).

### Q-Matrix & Quality

| Method | Path | Roles | Description |
|---|---|---|---|
| `POST` | `/api/v1/questions/{id}/skill-mappings` | ministry_admin, teacher, curriculum_officer | Add a versioned skill mapping for a question |
| `GET` | `/api/v1/questions/{id}/skill-mappings` | any authenticated | List skill mappings |
| `POST` | `/api/v1/questions/skill-mappings/resync` | ministry_admin | Resync backfilled mappings |
| `GET` | `/api/v1/curriculum/subjects/{code}/qmatrix-quality` | curriculum_officer, ministry_admin | Report: missing/ambiguous/low-confidence Q-matrix mappings |
| `GET` | `/api/v1/curriculum/subjects/{code}/prerequisite-quality` | curriculum_officer, ministry_admin | Report: orphaned topics, low-confidence edges |

### Units (direct CRUD, not part of upload pipeline)

| Method | Path | Roles | Description |
|---|---|---|---|
| `POST` | `/api/v1/curriculum/units` | ministry_admin, teacher | Create a unit |
| `PATCH` | `/api/v1/curriculum/units/{id}` | ministry_admin, teacher | Update a unit |
| `DELETE` | `/api/v1/curriculum/units/{id}` | ministry_admin | Delete a unit |

### Storage (file proxy)

| Method | Path | Roles | Description |
|---|---|---|---|
| `GET` | `/api/v1/storage/files/{jobId}` | curriculum_officer, ministry_admin | Stream the original uploaded file (dev: from Postgres BYTEA; prod: would be S3) |
| `POST` | `/api/v1/storage/presign-upload` | any authenticated | Get a presigned upload URL |
| `POST` | `/api/v1/storage/presign-download` | any authenticated | Get a presigned download URL |

---

## Exams

### Upload & Validate

| Method | Path | Roles | Description |
|---|---|---|---|
| `POST` | `/api/v1/exams/upload` | teacher, school_admin | Upload exam PDF/DOCX. Triggers AI parsing asynchronously. |
| `GET` | `/api/v1/exams/{id}` | teacher, school_admin | Get exam with extracted questions |
| `POST` | `/api/v1/exams/{id}/validate` | teacher, school_admin | Trigger AI validation report (CLO coverage, question quality) |
| `PATCH` | `/api/v1/exams/{id}/scope` | teacher, school_admin | Fix subject/grade/exam-type/unit-range without re-uploading |
| `POST` | `/api/v1/exams/{id}/publish` | teacher, school_admin | Publish exam (makes it visible to students) |
| `POST` | `/api/v1/exams/{id}/answer-key` | teacher, school_admin | Upload answer key; triggers AI CLO alignment |

### Student Submission

| Method | Path | Roles | Description |
|---|---|---|---|
| `GET` | `/api/v1/exams/{id}/questions` | student | List questions for a published exam |
| `POST` | `/api/v1/exams/{id}/submit` | student | Submit answers. Triggers gap analysis + EG-GCKT trace asynchronously. |

**Submit body:**
```json
{
  "answers": [
    { "questionId": "uuid", "selectedOptionId": "uuid", "timeSpentSecs": 45 }
  ]
}
```

### Grading

| Method | Path | Roles | Description |
|---|---|---|---|
| `GET` | `/api/v1/exams/{id}/grading-questions` | teacher, school_admin | Questions with student answer summaries for manual grading |
| `POST` | `/api/v1/exams/{id}/grades/bulk` | teacher, school_admin | Bulk grade student answers |

### Insights

| Method | Path | Roles | Description |
|---|---|---|---|
| `GET` | `/api/v1/exams/{id}/my-insight` | student | Gap-analysis result for the current student (async — may return 404 while worker runs) |
| `GET` | `/api/v1/exams/{id}/insights` | teacher, school_admin | All students' insights for an exam |
| `GET` | `/api/v1/exams/{id}/quality` | teacher, school_admin | Exam quality report (discrimination, calibration, timing, CLO coverage). Recomputed on each read. |

### Print

| Method | Path | Roles | Description |
|---|---|---|---|
| `GET` | `/api/v1/exams/{id}/print` | teacher, school_admin | Printable exam sheet (HTML) |
| `GET` | `/api/v1/exams/{id}/print/answer-key` | teacher, school_admin | Printable answer key (teacher only — never reachable by students) |

---

## EG-GCKT: Misconceptions

| Method | Path | Roles | Description |
|---|---|---|---|
| `GET` | `/api/v1/misconceptions` | teacher, school_admin | List candidate misconception hypotheses for the caller's school |
| `PATCH` | `/api/v1/misconceptions/{id}/review` | teacher, school_admin | Confirm or reject a hypothesis |

**Review body:** `{ "action": "confirm" | "reject", "notes": "..." }`

---

## EG-GCKT: Model Governance

| Method | Path | Roles | Description |
|---|---|---|---|
| `GET` | `/api/v1/model-snapshots/candidates` | ministry_admin, curriculum_officer | List candidate model snapshots awaiting review |
| `POST` | `/api/v1/model-snapshots/{id}/promote` | ministry_admin | Promote snapshot to active (supersedes prior active snapshot) |
| `POST` | `/api/v1/model-snapshots/{id}/reject` | ministry_admin | Reject snapshot |

Nightly refit (`refit_worker.py`) creates `candidate` snapshots automatically. They are never auto-promoted.

---

## Tutor (Graph-RAG)

| Method | Path | Roles | Description |
|---|---|---|---|
| `POST` | `/api/v1/tutor/ask` | student | Ask the AI tutor a curriculum question. Requires Gemini API key — hard 503 if unconfigured. |

**Body:** `{ "question": "What is photosynthesis?" }`  
**Response:** `{ "data": { "answer": "...", "referencedTopics": ["uuid"], "sourcedFrom": ["gap_records", "prerequisites"] } }`

---

## Career Paths

| Method | Path | Roles | Description |
|---|---|---|---|
| `GET` | `/api/v1/career/paths` | any authenticated | List available career paths |
| `POST` | `/api/v1/career/paths` | ministry_admin | Create a career path |

> **Note:** `POST /api/v1/students/me/career/generate` is **currently broken** — the ai-service career route is not wired. It returns 404 from the ai-service. See Known Gaps in `CLAUDE.md`.

---

## Notifications

| Method | Path | Roles | Description |
|---|---|---|---|
| `GET` | `/api/v1/notifications` | any authenticated | List notifications for the current user |
| `PATCH` | `/api/v1/notifications/{id}/read` | any authenticated | Mark a notification as read |
| `POST` | `/api/v1/notifications` | school_admin, regional_admin, ministry_admin | Create a manual notification |

Notifications are also inserted automatically when a school's `flagged_for_review` transitions from false → true.

---

## Jobs

General-purpose job queue table (internal use, not the curriculum upload pipeline).

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/jobs` | Create a job |
| `GET` | `/api/v1/jobs` | List jobs |
| `GET` | `/api/v1/jobs/{id}` | Get one job |
| `PATCH` | `/api/v1/jobs/{id}/status` | Update job status |

---

## Health Check

`GET /health` — no auth required  
**Response `200`:** `{ "status": "ok" }`

---

## Error Format

All error responses follow:
```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "topic not found"
  }
}
```

Common codes: `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `VALIDATION_ERROR`, `INTERNAL_ERROR`, `CONFLICT`.

---

## AI Service Direct Endpoints

The ai-service (`localhost:8000`) exposes only **one** real HTTP route. All other ai-service capabilities are triggered asynchronously via Redis queues pushed by the Go API.

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/tutor/ask` | Graph-RAG tutor (proxied through Go → ai-service) |

The following ai-service routes exist as **empty stubs** and are not registered in `main.py`:  
`/api/v1/routes/career.py`, `/api/v1/routes/policy.py`, `/api/v1/routes/plan.py`

---

## Redis Queue Summary

The Go API pushes job IDs (or JSON payloads) to these queues; the ai-service `BRPOP`s them:

| Queue | Trigger | Consumer |
|---|---|---|
| `queue:curriculum:parse` | `POST /curriculum/upload` | `curriculum_worker.py` |
| `queue:exam:parse` | `POST /exams/upload` | `exam_worker.py` |
| `queue:exam:answerkey` | `POST /exams/{id}/answer-key` | `answer_key_worker.py` |
| `queue:gap:analyze` | `POST /exams/{id}/submit` (after grading) | `gap_worker.py` |
| `queue:studyplan:generate` | `POST /students/me/study-plans` | `study_plan_worker.py` |
| `queue:gckt:trace` | `POST /exams/{id}/submit` (alongside gap) | `kt_worker.py` |
| `queue:embedding:generate` | `POST /curriculum/jobs/{id}/approve` | `embed_worker.py` |
