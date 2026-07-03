CREATE TABLE regions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    code        TEXT NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE schools (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    region_id   UUID NOT NULL REFERENCES regions (id) ON DELETE RESTRICT,
    name        TEXT NOT NULL,
    code        TEXT NOT NULL UNIQUE,
    address     TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_schools_region_id ON schools (region_id);

-- Users belong to a region and/or school depending on role
ALTER TABLE users
    ADD COLUMN region_id UUID REFERENCES regions (id) ON DELETE SET NULL,
    ADD COLUMN school_id UUID REFERENCES schools (id) ON DELETE SET NULL;

CREATE INDEX idx_users_school_id ON users (school_id);
CREATE INDEX idx_users_region_id ON users (region_id);
