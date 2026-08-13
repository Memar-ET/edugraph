# Career Guidance (Capability 3D)

## Scope status: official, in-scope feature (confirmed)

Career Guidance is confirmed as an official, in-scope platform feature —
not out-of-scope work or unapproved scope creep, explicitly ratified as a
decision, same as the AI Tutor. This is backed by the PRD itself:
`edugraph-impl-plan.docx` specifies it as **Capability 3D — Career
Recommendation Engine**, phase-tracked alongside gap analysis, the study
planner, and the AI tutor (Phase 3, "Career recommendation" listed
explicitly in both the phase table and the timeline). An earlier review
claiming "career guidance isn't in the PRD I reviewed" was wrong — same
mistake as the AI Tutor's PRD-status error, from checking
`edugraph-architecture.docx` only.

## What was actually broken (checklist 7.1)

Deeper than the earlier description ("will fail with a 502 error if
clicked"). Investigating turned up two independent, stacked bugs:

1. **The Go repository read from dead tables.**
   `StudentSubjectAverages` queried `assessment_results`/`assessments`/
   `subjects` — the legacy public-schema tables from before the exam
   pipeline moved to `assessment.exam_attempts`/`assessment.exams`.
   Nothing writes to those legacy tables anymore (confirmed by grep), so
   this always returned an empty map for every real student, failing
   with "student has no graded assessments" before ever reaching the
   ai-service call.

2. **The tables it should have used don't exist under that name either.**
   `career_paths`/`career_matches` (unqualified, public schema) were
   themselves renamed — archived, not dropped — to
   `career_paths_v010`/`career_matches_v010` by
   `V011__updated_curriculum.sql`, to make room for a newer
   `careers.careers`/`careers.career_topic_requirements`/
   `careers.career_matches` schema (topic-level requirements with an
   `importance` weighting, much closer to the PRD's Neo4j-native design).
   The Go repository was never updated after that rename, so every query
   against `career_paths` failed outright with "relation does not
   exist" — a harder failure than a downstream 502, and the real reason
   nothing worked.

3. **The ai-service side didn't exist at all.** `ai-service/app/api/v1/
   routes/career.py` and `career_matcher/service.py` were both empty
   files, not registered in `main.py`. The Go backend's `MatchCareers()`
   call had nowhere to land.

4. **No UI ever called it.** `frontend/src/lib/api/endpoints.ts` already
   had `generateCareerMatches()`/`getCareerMatches()` defined, and
   `queryKeys.ts` already had a `careerMatches` key — but no page
   component called either function. The earlier claim that "the button
   is live in the UI today" doesn't hold for the code as it stands; the
   wiring existed, just unused.

### What was fixed

- `StudentSubjectAverages` now reads `assessment.exam_attempts`/
  `assessment.exams` (the tables the real exam-taking flow writes to),
  grouped by subject family (`curriculum.subjects.code` with trailing
  grade digits stripped, e.g. `"BIO7"` → `"BIO"` — a career applies
  across grades, not to one specific grade's subject code).
- The Go repository now targets `career_paths_v010`/`career_matches_v010`
  — the tables that actually hold live data — rather than the newer,
  completely unused `careers.*` schema. Migrating to that richer,
  topic-level schema is real future work (it also has no create/curation
  UI for `career_topic_requirements` yet), not done here; see the inline
  comments in `repository.go` and `career_matcher/service.py`.
- Built the missing ai-service side: `POST /api/v1/career/match` scores
  each career path as the mean of the student's grades across whichever
  required subjects they actually have exam data for. A subject with no
  data simply doesn't contribute (no evidence isn't scored as failure,
  same principle as gap-analysis's cold-start handling); a career with
  *no* matched subject data scores 0.0.
- Wired the already-defined frontend API functions into `CareerPathsPage.tsx`
  behind a student-only "My career matches" card with a generate/regenerate
  button — the feature is now actually reachable, not just fixed
  underneath an unused code path.

### Not done, left as documented future work

- Migrating to the `careers.careers`/`careers.career_topic_requirements`
  topic-level schema (would need a curation UI for linking careers to
  topics — none exists).
- The PRD's Neo4j-native readiness scoring
  (`(:Career)-[:REQUIRES]->(:Topic)`, `:MASTERED` edges) — no `(:Career)`
  nodes and no student-mastery graph edges exist in Neo4j today; this
  feature runs entirely on Postgres, matching how gap-analysis and the
  study planner also default to their Postgres path in practice.
- `career_paths_v010.embedding` (`vector(1024)`, from V007) — an unused
  design for semantic career matching. There's no existing "student
  profile" text or embedding anywhere in this system to compare it
  against, so it stays unpopulated.
- The Neo4j mirror in `SaveMatches` (`MATCH (s:Student {id: ...})`)
  silently no-ops today — no `(:Student)` nodes are ever created
  anywhere in the codebase, so the `MATCH` finds nothing and the
  `MERGE`/`SET` that follows has nothing to act on. Not fatal (Postgres
  is the system of record and is unaffected), but the "mirrors matches
  as Neo4j edges" comment above that function is currently aspirational.

### Verification

Seeded two real career paths (`Biologist` requiring `["BIO"]`,
`Software Engineer` requiring `["MATH", "PHY"]`) and gave a real test
student two real `BIO7` exam attempts (0% and 85%) against real Grade 7
Biology exams. Called the actual `POST /students/{id}/career/generate`
endpoint through the real Go API:

- `Biologist` scored `0.425` — exactly the average of the student's two
  BIO7 attempts.
- `Software Engineer` scored `0` — the student has no MATH/PHY exam data
  at all, correctly scored zero rather than erroring or being omitted.
- `GET /students/{id}/career/matches` returned the same persisted
  results on a separate call.
