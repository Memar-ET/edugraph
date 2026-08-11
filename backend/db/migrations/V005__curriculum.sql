CREATE TABLE subjects (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    code        TEXT NOT NULL UNIQUE,
    grade_level SMALLINT NOT NULL CHECK (grade_level BETWEEN 1 AND 12),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE curriculum_units (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_id   UUID NOT NULL REFERENCES subjects (id) ON DELETE CASCADE,
    title        TEXT NOT NULL,
    description  TEXT,
    order_index  INTEGER NOT NULL DEFAULT 0,
    embedding    vector(1024),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_curriculum_units_subject_id ON curriculum_units (subject_id);
CREATE INDEX idx_curriculum_units_embedding ON curriculum_units
    USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
