-- Capability 2C: Student Answer Ingestion.
--
-- Lets both flows use INSERT ... ON CONFLICT (attempt_id, question_id) DO
-- UPDATE, so re-saving (a teacher fixing a typo and clicking "Save All"
-- again, or a retried request) is idempotent instead of creating duplicate
-- rows. No "one attempt per student per exam" constraint added here --
-- enforced in the Go service layer instead, so a future multi-attempt/
-- retake policy doesn't need another migration.

ALTER TABLE assessment.student_answers
    ADD CONSTRAINT student_answers_attempt_question_unique UNIQUE (attempt_id, question_id);
