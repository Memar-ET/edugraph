# Data Security & Privacy Policy

**Status**: internal technical/operational policy describing this platform's
actual current practice, verified against the running code and schema as
of 2026-08-13. **This is not a legally-vetted public-facing privacy
notice and not legal advice.** EduGraph AI processes personal data
belonging to K-12 students, most of them minors, across Ethiopian
schools — before any real deployment collects real students' data, this
document needs review by someone qualified on Ethiopian data-protection
law (and any applicable regional/international framework, depending on
where the platform is actually deployed). What follows is the honest
technical baseline that review would start from, not a substitute for
it.

## 1. What data this platform collects

Grounded in the actual schema (`backend/db/migrations/`), not aspirational:

| Category | Fields | Table(s) |
|---|---|---|
| Account identity | email, full name, phone (optional), role, hashed password | `public.users` |
| School/region affiliation | school_id, region_id | `public.users`, `public.students`, `public.teachers` |
| Student record | admission number, grade level | `public.students` |
| Academic performance | exam answers, scores, per-question marks, time spent | `assessment.student_answers`, `assessment.exam_attempts` |
| Diagnosed weaknesses | which topics/prerequisites a student struggles with, severity, whether it's a root cause | `students.gap_records` |
| AI-generated content about a student | narrative gap explanations, study plans, tutor conversation context | `students.exam_insights`, `students.study_plans`, generated per-request for the tutor (not persisted as chat history) |
| Career interest signals | per-subject grade averages used for career matching | computed on demand from `assessment.exam_attempts`, not separately stored beyond the match result (`career_matches_v010`) |

**Not collected**: no payment information, no biometric data, no location
tracking beyond the school/region a user is administratively assigned to,
no browsing/analytics tracking of any kind (no analytics SDK exists in
the frontend — checked, not assumed).

## 2. Where it's stored

- **Cloud deployment**: Postgres via a hosted Supabase project (see
  `CLAUDE.md`'s "Postgres is now Supabase" section) — Supabase-managed
  encryption at rest, connection requires `POSTGRES_SSLMODE=require`
  (enforced in config, not optional). Neo4j (the curriculum/progress
  knowledge graph) runs in a plain Docker container, **not currently on
  managed/encrypted infrastructure** — this is a real gap for a
  production cloud deployment, not yet addressed.
- **School Box (offline edge) deployment**: Postgres, Neo4j, and Redis
  all run locally on the school's own hardware (see
  [offline-sync.md](../architecture/offline-sync.md)) — student data
  physically stays on school premises except what the sync-agent
  explicitly pushes to Central Cloud, and even that requires the
  device-secret authentication added in the 10.1 audit (see
  [data-integrity.md](../architecture/data-integrity.md)).

## 3. Who can access what

Enforced server-side, not just hidden in the UI — see
[data-integrity.md](../architecture/data-integrity.md) (10.1) and
[security.md](security.md) (11.3) for the concrete audit and fixes this
claim is based on, not an assertion made without evidence:

- **Students** see only their own records (`/students/me/*` throughout).
- **Teachers/school_admins** see only their own school's students,
  exams, and grades — server-side, resolved from their own account, not
  a client-supplied school id (fixed 2026-08-13, was previously
  unscoped for the entire exam-management surface).
- **Regional admins** are scoped to schools within their own region.
- **Ministry admins** have national visibility — the one role for which
  this is by design, not a gap (matches the Ministry oversight role the
  PRD describes).
- **Curriculum officers** collaboratively manage the shared national
  curriculum (not school-scoped — curriculum content isn't
  student-specific personal data).

Passwords are bcrypt-hashed at cost 12 (`pkg/crypto`), never stored or
logged in plaintext. Session tokens are `HttpOnly` cookies as of the
11.1 fix (see [security.md](security.md)) — not readable by JavaScript,
reducing XSS exposure.

## 4. Third-party sharing

**AI providers.** Gap analysis, the AI tutor, and study-plan enrichment
call an LLM (`ai-service/app/utils/llm_provider.py`) — local (Ollama,
the default, see
[ai-models-local-vs-cloud.md](../architecture/ai-models-local-vs-cloud.md))
or Gemini (Google) as a fallback/alternate when `LLM_PROVIDER=cloud`.
**Checked directly against the actual prompts sent** (`gap_analysis/llm.py`,
`tutor/service.py`, `study_plan/llm.py`), not assumed: these prompts
include exam title, subject, grade level, score percentage, question
text, and topic names — **they do not include the student's name, email,
or any other direct identifier**. A determined party with access to
Gemini's request logs (Google, or anyone with access to Google's side)
could not connect a given prompt back to a specific named student from
the prompt content alone, though the *pattern* of grade-level/subject/
topic data is still personal academic information about a real (if
unnamed) child, and Google's own API terms and data-handling practices
apply to whatever is sent — this policy doesn't control that, only
what leaves this platform.

**No other third parties** currently receive student data. There is no
analytics vendor, no ad network, no data broker, no CRM integration.

**A School Box run fully local-only** (`LLM_PROVIDER=local`, no
`GEMINI_API_KEY` set — the recommended default, see
`.env.school-box.example`) sends **no student data to any third party
at all** for AI features; only the periodic sync to Central Cloud
leaves the building, over the device-authenticated `/sync/push`/`/pull`
endpoints.

## 5. Retention and deletion

**Honest gap, not yet implemented**: there is no automated retention
policy or scheduled purge job anywhere in this codebase. Data persists
indefinitely once written, with two exceptions added in the 10.3 audit
(see [data-integrity.md](../architecture/data-integrity.md)):
`students`/`teachers` support soft-delete (`deleted_at`), so a withdrawn
student's record can be excluded from active views without destroying
their historical academic data, and `assessment.answer_grade_history`
gives a real audit trail for grade changes. Neither of these is a
retention *limit* — nothing currently deletes old exam attempts, gap
records, or AI-generated content after any fixed period.

**Before a real deployment**, this needs an actual decision (likely
driven by whatever legal review this document flags as necessary in its
header): how long should a graduated or transferred student's data be
kept, and who is responsible for actually purging it. Recommending a
specific retention period here would be inventing a number, not
reporting on the system's actual behavior — this section is deliberately
descriptive of the gap, not prescriptive of an unvalidated policy.

## 6. Data subject rights (student/parent access, correction, deletion)

**Not yet built.** There is no self-service "download my data" or
"delete my account" flow for students or parents anywhere in the
frontend or API today. A school_admin can `DELETE /students/{id}`
(soft-delete, see above) and a `curriculum_officer`/`ministry_admin` can
edit most records through their respective role's endpoints, but there
is no student/parent-facing mechanism for exercising data rights
directly. This is a real, currently-open item — flagged here rather than
silently assumed to exist because a mature-sounding policy document
implies it.

## 7. Incident response

**Not yet built.** No breach-notification workflow, no incident runbook
specific to a data exposure, exists in `docs/runbooks/` today (checked,
not assumed empty). If this policy is adopted for a real deployment, a
runbook covering "what do we do if student data is exposed" belongs
alongside the existing operational runbooks and should name a specific
responsible party — this document doesn't invent one, since naming a
real accountable person/team isn't something to fabricate.

## 8. What's already a real, enforced control vs. what's aspirational

To make this document auditable against the code rather than a wish
list, every claim above is grounded in something checked during this
review (schema, code, or a live test run earlier in this session) except
where explicitly marked "not yet implemented" or "not yet built." If a
future reviewer finds a claim here that no longer matches the running
code, the code is authoritative — update this document, not the other
way around.
