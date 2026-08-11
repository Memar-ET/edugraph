CREATE TABLE career_paths (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title              TEXT NOT NULL,
    description        TEXT,
    required_subjects  JSONB NOT NULL DEFAULT '[]',
    embedding          vector(1024),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_career_paths_embedding ON career_paths
    USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

CREATE TABLE career_matches (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id      UUID NOT NULL REFERENCES students (id) ON DELETE CASCADE,
    career_path_id  UUID NOT NULL REFERENCES career_paths (id) ON DELETE CASCADE,
    match_score     NUMERIC(5, 4) NOT NULL,
    generated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (student_id, career_path_id)
);

CREATE INDEX idx_career_matches_student_id ON career_matches (student_id);
