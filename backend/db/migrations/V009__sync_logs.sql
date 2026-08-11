CREATE TABLE sync_logs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id    TEXT NOT NULL,
    school_id    UUID NOT NULL REFERENCES schools (id) ON DELETE CASCADE,
    entity_type  TEXT NOT NULL,
    entity_id    UUID NOT NULL,
    operation    sync_operation NOT NULL,
    payload      JSONB NOT NULL,
    status       sync_status NOT NULL DEFAULT 'pending',
    error        TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    synced_at    TIMESTAMPTZ
);

CREATE INDEX idx_sync_logs_school_id_status ON sync_logs (school_id, status);
CREATE INDEX idx_sync_logs_device_id ON sync_logs (device_id);
