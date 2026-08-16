-- V052: Exam autosave drafts + idempotency key on student_answers.
-- Autosave lets students recover from disconnects mid-exam.
-- The idempotency_key on student_answers prevents double-submission.

CREATE TABLE IF NOT EXISTS assessment.exam_draft_answers (
  id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  student_id         UUID        NOT NULL REFERENCES public.students(id)             ON DELETE CASCADE,
  exam_id            UUID        NOT NULL REFERENCES assessment.exams(id)            ON DELETE CASCADE,
  question_id        UUID        NOT NULL REFERENCES assessment.questions(id)        ON DELETE CASCADE,
  selected_option_id UUID                 REFERENCES assessment.question_options(id) ON DELETE SET NULL,
  text_answer        TEXT,
  saved_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (student_id, exam_id, question_id)
);

CREATE INDEX IF NOT EXISTS idx_exam_draft_answers_student_exam
  ON assessment.exam_draft_answers (student_id, exam_id);

-- Idempotency key on student_answers: a client that retries a submit with
-- the same UUID gets back the original result without re-processing.
ALTER TABLE assessment.student_answers
  ADD COLUMN IF NOT EXISTS idempotency_key UUID;

CREATE UNIQUE INDEX IF NOT EXISTS idx_student_answers_idempotency
  ON assessment.student_answers (student_id, exam_id, idempotency_key)
  WHERE idempotency_key IS NOT NULL;
