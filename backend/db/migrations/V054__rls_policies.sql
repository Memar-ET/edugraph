-- V054: Row-Level Security (RLS) as defence-in-depth on sensitive student tables.
--
-- Context:
--   The Go API always connects via the Supabase service_role, which bypasses RLS
--   automatically — no policy change is needed for the app to keep working.
--   RLS here closes a different attack surface: direct PostgREST / Supabase JS
--   client calls from the browser using the anon or authenticated JWT (if anyone
--   ever exposes the Supabase URL + anon key from the frontend). Without RLS those
--   calls read any student's data unrestricted. With RLS enabled and no permissive
--   policy for anon/authenticated roles, those calls return empty result sets.
--
-- What we do NOT do here:
--   We do NOT add permissive policies for the service_role — Supabase service_role
--   bypasses RLS at the connection level (bypassrls=true on that role), so any
--   USING (true) policy for service_role is redundant noise.
--   We do NOT touch public.users or other non-PII tables — RLS there would add
--   complexity with no meaningful security gain given the app-level auth model.
--
-- Tables covered:
--   students.skill_states          — per-student KT mastery estimates
--   students.learning_events       — append-only raw response log
--   students.gap_records           — gap analysis output per attempt
--   assessment.student_answers     — per-question student responses

-- ── students.skill_states ────────────────────────────────────────────────────

ALTER TABLE students.skill_states ENABLE ROW LEVEL SECURITY;

-- Block all direct anon/authenticated access. The service_role bypasses this.
CREATE POLICY skill_states_deny_direct
    ON students.skill_states
    AS RESTRICTIVE
    TO PUBLIC
    USING (false);

-- ── students.learning_events ─────────────────────────────────────────────────

ALTER TABLE students.learning_events ENABLE ROW LEVEL SECURITY;

CREATE POLICY learning_events_deny_direct
    ON students.learning_events
    AS RESTRICTIVE
    TO PUBLIC
    USING (false);

-- ── students.gap_records ─────────────────────────────────────────────────────

ALTER TABLE students.gap_records ENABLE ROW LEVEL SECURITY;

CREATE POLICY gap_records_deny_direct
    ON students.gap_records
    AS RESTRICTIVE
    TO PUBLIC
    USING (false);

-- ── assessment.student_answers ───────────────────────────────────────────────

ALTER TABLE assessment.student_answers ENABLE ROW LEVEL SECURITY;

CREATE POLICY student_answers_deny_direct
    ON assessment.student_answers
    AS RESTRICTIVE
    TO PUBLIC
    USING (false);

-- ── audit_log (write-only for anon — defence-in-depth) ───────────────────────
-- The audit_log should not be readable by any role other than the service_role
-- or a privileged DB admin. Anon/authenticated roles get no access at all.

ALTER TABLE public.audit_log ENABLE ROW LEVEL SECURITY;

CREATE POLICY audit_log_deny_direct
    ON public.audit_log
    AS RESTRICTIVE
    TO PUBLIC
    USING (false);
