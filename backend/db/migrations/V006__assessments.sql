CREATE TABLE assessments (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_id   UUID NOT NULL REFERENCES subjects (id) ON DELETE RESTRICT,
    school_id    UUID REFERENCES schools (id) ON DELETE CASCADE,
    created_by   UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    title        TEXT NOT NULL,
    type         assessment_type NOT NULL DEFAULT 'quiz',
    total_marks  INTEGER NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_assessments_school_id ON assessments (school_id);
CREATE INDEX idx_assessments_subject_id ON assessments (subject_id);

CREATE TABLE assessment_questions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assessment_id   UUID NOT NULL REFERENCES assessments (id) ON DELETE CASCADE,
    question_text   TEXT NOT NULL,
    options         JSONB,
    correct_answer  TEXT NOT NULL,
    marks           INTEGER NOT NULL DEFAULT 1,
    order_index     INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_assessment_questions_assessment_id ON assessment_questions (assessment_id);

CREATE TABLE assessment_results (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assessment_id  UUID NOT NULL REFERENCES assessments (id) ON DELETE CASCADE,
    student_id     UUID NOT NULL REFERENCES students (id) ON DELETE CASCADE,
    answers        JSONB,
    score          NUMERIC(6, 2),
    submitted_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    graded_at      TIMESTAMPTZ,
    UNIQUE (assessment_id, student_id)
);

CREATE INDEX idx_assessment_results_student_id ON assessment_results (student_id);
