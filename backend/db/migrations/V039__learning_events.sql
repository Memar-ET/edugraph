-- EG-GCKT Milestone 0: Student Event Model (spec section 14) -- an
-- append-only, immutable event log distinct from
-- assessment.student_answers on purpose.
--
-- student_answers is mutable (upserted on ON CONFLICT, time_spent_secs
-- COALESCEd, marks overwritten on regrade) while every downstream EG-GCKT
-- engine (BKT/DINA/fusion/replay) needs a write-once record it can trust
-- never changes under it. A view or trigger over student_answers can't
-- provide that immutability without becoming this table anyway, so the
-- two are kept physically separate and connected only by the answer_id
-- breadcrumb. skill_ids/attempt_number/response_time etc. are
-- deliberately NOT added to student_answers itself for the same reason:
-- that table is the grading system-of-record (owned by the assessment
-- domain's grading UI), this one is the ML event log (owned by the GCKT
-- foundation) -- different write cadences, different consumers.
--
-- Written synchronously by Go (Milestone 1) at the RecomputeAttemptTotals
-- seam -- the exact point an attempt becomes fully graded, which already
-- gates the existing queue:gap:analyze push -- not async off Redis,
-- because this table is the record every downstream engine depends on
-- for correctness and shouldn't need its own resync-repair mechanism the
-- way the best-effort Neo4j mirrors do.

CREATE TABLE students.learning_events (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id         UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    school_id          UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    item_id            UUID REFERENCES assessment.questions(id),
    answer_id          UUID REFERENCES assessment.student_answers(id) ON DELETE SET NULL,
    occurred_at        TIMESTAMPTZ NOT NULL,
    skill_ids          UUID[] NOT NULL DEFAULT '{}',
    response           JSONB,
    correctness        BOOLEAN,
    attempt_number     INT NOT NULL DEFAULT 1,
    response_time_secs INT,
    hint_count         INT NOT NULL DEFAULT 0,
    resource_context   TEXT,
    confidence_signal  NUMERIC CHECK (confidence_signal BETWEEN 0 AND 1),
    assessment_id      UUID REFERENCES assessment.exams(id),
    intervention_id    UUID,
    item_version       INT,
    model_snapshot_id  UUID REFERENCES modeling.model_snapshots(id),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learning_events_student ON students.learning_events(student_id, occurred_at DESC);
CREATE INDEX idx_learning_events_skill_ids ON students.learning_events USING GIN(skill_ids);

-- One event per underlying student_answers row -- a teacher regrade
-- (BulkGradeExam can be called more than once against the same attempt)
-- must UPDATE the existing event's correctness/skill_ids rather than
-- accumulate a duplicate "the student answered this again" event. The
-- student's response itself didn't change on a regrade, only its
-- grading outcome, so this is squarely "keep one row current," not a
-- violation of the append-only-event principle (which is about not
-- losing/mutating *response* history, not about re-grading events).
CREATE UNIQUE INDEX idx_learning_events_answer ON students.learning_events(answer_id) WHERE answer_id IS NOT NULL;
