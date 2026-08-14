-- Checklist 10.1 finding: POST/GET /api/v1/sync/push and /sync/pull (the
-- cloud endpoints the School Box sync-agent talks to, internal/sync/*)
-- were completely unauthenticated -- any caller on the internet could
-- inject fabricated sync_logs rows tagged to any school_id, or read back
-- another school's entire pulled-change stream by guessing a school
-- UUID. This table backs a device-secret check added at the same time
-- (see internal/sync/handler/device_auth.go): one row per School Box,
-- secret stored bcrypt-hashed (same as users.password_hash, see
-- pkg/crypto.HashPassword) never plaintext, bound to exactly one
-- school_id so a leaked/stolen credential for School A can't be pointed
-- at School B's data by just changing the request's school_id field --
-- the handler cross-checks the two.
CREATE SCHEMA IF NOT EXISTS sync;

CREATE TABLE sync.device_credentials (
    device_id     TEXT PRIMARY KEY, -- matches SCHOOL_BOX_ID in .env.school-box
    school_id     UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    secret_hash   TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at    TIMESTAMPTZ,
    last_seen_at  TIMESTAMPTZ
);

CREATE INDEX idx_device_credentials_school_id ON sync.device_credentials(school_id);
