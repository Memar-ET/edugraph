CREATE TABLE students (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL UNIQUE REFERENCES users (id) ON DELETE CASCADE,
    school_id     UUID NOT NULL REFERENCES schools (id) ON DELETE RESTRICT,
    admission_no  TEXT NOT NULL,
    grade_level   SMALLINT NOT NULL CHECK (grade_level BETWEEN 1 AND 12),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (school_id, admission_no)
);

CREATE INDEX idx_students_school_id ON students (school_id);

CREATE TABLE teachers (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID NOT NULL UNIQUE REFERENCES users (id) ON DELETE CASCADE,
    school_id           UUID NOT NULL REFERENCES schools (id) ON DELETE RESTRICT,
    subject_specialty   TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_teachers_school_id ON teachers (school_id);
