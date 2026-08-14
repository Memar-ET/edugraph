-- Checklist 10.3: per-table soft-delete/audit-trail decisions.
-- Full per-table reasoning: docs/architecture/data-integrity.md.

-- ── Soft delete: students, teachers ─────────────────────────────
-- Delete() was a hard DELETE cascading through gap_records/
-- exam_attempts/student_answers/study_plans/career_matches -- a routine
-- withdrawal or transfer permanently destroyed a student's entire
-- academic history. deleted_at lets the record (and everything FK'd to
-- it) survive; List/GetByID now filter it out (see
-- student|teacher/repository.go).
ALTER TABLE students ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE teachers ADD COLUMN deleted_at TIMESTAMPTZ;

CREATE INDEX idx_students_active ON students(school_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_teachers_active ON teachers(school_id) WHERE deleted_at IS NULL;

-- ── Audit trail: regrades ────────────────────────────────────────
-- exam_attempts.graded_by/graded_at (V011) already covers first-grading.
-- Nothing recorded a SECOND grading of the same answer -- marks_awarded
-- was just silently overwritten, with no record of the previous value or
-- who changed it. This is exactly the "who changed a grade" concern
-- checklist 10.3 flags. One row per grading event (including the first,
-- for a complete history in one place); see
-- assessment/repository/*.go's grading write path for what inserts here.
CREATE TABLE assessment.answer_grade_history (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    answer_id       UUID NOT NULL REFERENCES assessment.student_answers(id) ON DELETE CASCADE,
    marks_awarded   NUMERIC,
    grading_notes   TEXT,
    graded_by       UUID NOT NULL REFERENCES users(id),
    graded_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_answer_grade_history_answer_id ON assessment.answer_grade_history(answer_id);

-- ── Audit trail: curriculum edits ────────────────────────────────
-- topics/clos already had updated_at but not who -- national curriculum
-- content should be attributable to a specific curriculum_officer/
-- ministry_admin. subjects had neither: it's upserted by re-approving
-- the same job's code (ON CONFLICT (code) DO UPDATE, see
-- ApproveAndPromote in curriculum/repository/repository.go) so it needed
-- both columns added, not just updated_by. Nullable throughout: existing
-- rows and system-driven writes (e.g. neo4j_written flips) have no
-- single human author.
ALTER TABLE curriculum.topics ADD COLUMN updated_by UUID REFERENCES users(id);
ALTER TABLE curriculum.clos ADD COLUMN updated_by UUID REFERENCES users(id);
ALTER TABLE curriculum.subjects
    ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN updated_by UUID REFERENCES users(id);
