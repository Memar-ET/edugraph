-- A dedicated table to hold raw files for local development.
-- In production, this table will be ignored, and we will use S3.
CREATE SCHEMA IF NOT EXISTS storage;

CREATE TABLE storage.local_files (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_name   TEXT NOT NULL,
    mime_type   TEXT NOT NULL,
    file_data   BYTEA NOT NULL,
    size_bytes  BIGINT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);