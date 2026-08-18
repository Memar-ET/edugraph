-- V052: Exam autosave drafts + idempotency key on student_answers.
-- Autosave lets students recover from disconnects mid-exam.
-- The idempotency_key on student_answers prevents double-submission.
--
-- Answer options live as a JSONB column on assessment.questions
-- (QuestionOption{letter, text} -- there is no normalized
-- assessment.question_options table), so a draft answer is a plain
-- text response (an option letter or free text), mirroring the same
-- string-based shape assessment.student_answers.answer_text already
-- uses for the final submitted answer.

CREATE TABLE IF NOT EXISTS assessment.exam_draft_answers (
  id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  student_id         UUID        NOT NULL REFERENCES public.students(id)             ON DELETE CASCADE,
  exam_id            UUID        NOT NULL REFERENCES assessment.exams(id)            ON DELETE CASCADE,
  question_id        UUID        NOT NULL REFERENCES assessment.questions(id)        ON DELETE CASCADE,
  response           TEXT,
  saved_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (student_id, exam_id, question_id)
);

CREATE INDEX IF NOT EXISTS idx_exam_draft_answers_student_exam
  ON assessment.exam_draft_answers (student_id, exam_id);

COMMENT ON COLUMN assessment.exam_draft_answers.response IS
  'Raw student input for this question so far -- an MCQ option letter or free text, matching student_answers.answer_text.';

-- Idempotency key on student_answers: a client that retries a submit with
-- the same UUID gets back the original result without re-processing.
ALTER TABLE assessment.student_answers
  ADD COLUMN IF NOT EXISTS idempotency_key UUID;

-- student_answers has no exam_id column -- answers are scoped by
-- attempt_id (assessment.exam_attempts), which already identifies one
-- exam-taking session, so idempotency is keyed on that, not exam_id.
CREATE UNIQUE INDEX IF NOT EXISTS idx_student_answers_idempotency
  ON assessment.student_answers (student_id, attempt_id, idempotency_key)
  WHERE idempotency_key IS NOT NULL;
