-- A dedicated table to hold raw files for local development.
-- In production, this table will be ignored, and we will use S3.
--
-- Schema is app_storage, not storage: on Supabase (see .env/CLAUDE.md),
-- "storage" is a platform-owned schema (Supabase's own Storage product --
-- buckets/objects/etc, present in every project before this migration
-- ever runs) and the `postgres` role has USAGE but not CREATE there, so
-- `CREATE TABLE storage.local_files` fails with permission denied.
CREATE SCHEMA IF NOT EXISTS app_storage;

CREATE TABLE app_storage.local_files (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_name   TEXT NOT NULL,
    mime_type   TEXT NOT NULL,
    file_data   BYTEA NOT NULL,
    size_bytes  BIGINT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);