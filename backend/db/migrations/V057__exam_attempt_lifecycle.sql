-- V057: Real exam attempt lifecycle (production-hardening pass).
--
-- Until now, assessment.exam_attempts rows were created at SUBMISSION
-- time (see the old SubmitExam), not when the student actually started --
-- there was no live "in progress" session at all, which is what made a
-- server-authoritative timer, persistent per-student randomization, and
-- resume-after-refresh impossible: there was nothing to persist those
-- into. This migration adds the columns/tables a real attempt lifecycle
-- needs. attempt_number already existed (V011, DEFAULT 1, never actually
-- incremented since attempts were 1-per-student-per-exam by construction
-- until now) -- it starts being computed for real once /start enforces
-- assessment.exams.attempt_limit (V051, also previously unread by any
-- Go code).
--
-- status uses row-presence-with-value rather than row-absence for
-- "not started" (unlike students.skill_states' cold-start-via-absence
-- convention) because an exam_attempts row is the resume/randomization
-- anchor itself -- it must exist once a student starts, by definition.

ALTER TABLE assessment.exam_attempts
  ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'in_progress'
    CHECK (status IN ('in_progress','submitted','auto_submitted','expired','cancelled','invalidated')),
  ADD COLUMN IF NOT EXISTS expires_at         TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS time_limit_minutes INT,
  ADD COLUMN IF NOT EXISTS submission_reason  TEXT
    CHECK (submission_reason IN ('student_submit','time_expired','teacher_override','system')),
  ADD COLUMN IF NOT EXISTS last_activity_at   TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS idempotency_key    UUID;

-- Backfill existing rows (all created at submit-time under the old flow)
-- to a consistent terminal state so the new status CHECK/default don't
-- misrepresent already-graded history as still in_progress.
UPDATE assessment.exam_attempts
SET status = 'submitted', submission_reason = 'student_submit'
WHERE submitted_at IS NOT NULL;

-- One idempotency key may only ever back one attempt -- lets SubmitExam
-- (and the new StartAttempt) catch a retried request via a unique-
-- violation and return the existing attempt instead of erroring or
-- silently creating a second one. Partial (WHERE NOT NULL) since old
-- backfilled rows have no key.
CREATE UNIQUE INDEX IF NOT EXISTS idx_exam_attempts_idempotency
  ON assessment.exam_attempts (idempotency_key) WHERE idempotency_key IS NOT NULL;

-- The real fix for the double-submission race: FindAttempt-then-
-- CreateAttempt was two round-trips with no DB-level guarantee between
-- them, so two concurrent requests could both observe "no attempt yet"
-- and both insert. A unique constraint makes the second insert fail
-- atomically instead, which StartAttempt/SubmitExam now catch and
-- resolve by returning the existing attempt.
CREATE UNIQUE INDEX IF NOT EXISTS idx_exam_attempts_student_exam_number
  ON assessment.exam_attempts (student_id, exam_id, attempt_number);

-- Persists each student's exact exam form once generated at StartAttempt
-- time, so refreshing the browser never re-randomizes it. option_order
-- is an array of option letters in the order they should be displayed
-- (e.g. ["C","A","D","B"]) -- the letter itself stays the stable answer
-- identifier grading compares against (assessment.questions.answer_key),
-- only its on-screen position is shuffled, so no change to the options
-- JSONB shape or the grading comparison was needed for this.
CREATE TABLE IF NOT EXISTS assessment.attempt_questions (
  id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  attempt_id    UUID        NOT NULL REFERENCES assessment.exam_attempts(id) ON DELETE CASCADE,
  question_id   UUID        NOT NULL REFERENCES assessment.questions(id)     ON DELETE CASCADE,
  display_order INT         NOT NULL,
  option_order  JSONB,
  UNIQUE (attempt_id, question_id),
  UNIQUE (attempt_id, display_order)
);

CREATE INDEX IF NOT EXISTS idx_attempt_questions_attempt ON assessment.attempt_questions(attempt_id);

-- Re-key drafts to the attempt they were saved under. Nullable/backfilled
-- rather than a hard NOT NULL: existing draft rows predate any attempt
-- (drafts were student+exam scoped, attempts didn't exist until submit),
-- so there is no attempt to backfill them onto. New autosave writes
-- always carry the current in_progress attempt_id going forward.
ALTER TABLE assessment.exam_draft_answers
  ADD COLUMN IF NOT EXISTS attempt_id UUID REFERENCES assessment.exam_attempts(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_exam_draft_answers_attempt ON assessment.exam_draft_answers(attempt_id);

-- Exam-integrity signal stream (tab visibility, fullscreen, connection
-- loss/restore) -- deliberately a dedicated table rather than reusing
-- public.audit_log: that table has no attempt-scoped index/FK and its
-- one-fire-and-forget-goroutine-per-row design (pkg/middleware/audit.go)
-- isn't shaped for a higher-frequency per-question event stream. These
-- are signals for troubleshooting/integrity review, not proof of
-- misconduct -- see the handler-level framing.
CREATE TABLE IF NOT EXISTS assessment.exam_integrity_events (
  id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  attempt_id       UUID        NOT NULL REFERENCES assessment.exam_attempts(id) ON DELETE CASCADE,
  event_type       TEXT        NOT NULL CHECK (event_type IN (
                      'tab_hidden','tab_visible','fullscreen_entered','fullscreen_exited',
                      'connection_lost','connection_restored'
                    )),
  occurred_at      TIMESTAMPTZ NOT NULL,
  sequence_number  INT         NOT NULL,
  metadata         JSONB,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (attempt_id, sequence_number)
);

CREATE INDEX IF NOT EXISTS idx_exam_integrity_events_attempt ON assessment.exam_integrity_events(attempt_id);
