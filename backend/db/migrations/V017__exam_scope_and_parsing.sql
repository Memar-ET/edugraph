-- Capability 2A: Exam Creation & Parsing.
--
-- assessment.exams gains an exam_scope (Unit Test / Midterm / Final Exam --
-- teachers don't pick this explicitly, it's derived from the exam title
-- alongside subject_code/grade_level, see backend/internal/assessment) and
-- a parse_error column mirroring curriculum.upload_jobs.parse_error. The
-- status CHECK is widened to cover the parsing lifecycle (pending/parsing/
-- failed) in front of the existing draft/validation_pending/published/closed
-- states, the same way curriculum.upload_jobs tracks parsing status
-- separately from the review/approve states.
--
-- assessment.questions gains part_label to capture "Part A"/"Part B"
-- style section headers found while parsing.

ALTER TABLE assessment.exams
    ADD COLUMN exam_scope TEXT NOT NULL DEFAULT 'unit_test'
        CHECK (exam_scope IN ('unit_test', 'midterm', 'final_exam')),
    ADD COLUMN parse_error TEXT;

ALTER TABLE assessment.exams ALTER COLUMN exam_scope DROP DEFAULT;

ALTER TABLE assessment.exams DROP CONSTRAINT IF EXISTS exams_status_check;
ALTER TABLE assessment.exams ADD CONSTRAINT exams_status_check
    CHECK (status IN ('pending', 'parsing', 'draft', 'validation_pending', 'published', 'closed', 'failed'));

ALTER TABLE assessment.exams ALTER COLUMN status SET DEFAULT 'pending';

ALTER TABLE assessment.questions ADD COLUMN part_label TEXT;
