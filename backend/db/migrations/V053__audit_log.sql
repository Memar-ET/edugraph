-- V053: Audit log table for privileged action tracking.
--
-- Records every HTTP request and explicit business-level privileged action
-- (curriculum approve, exam publish, model promote, misconception confirm).
-- Written fire-and-forget by the Go AuditLog middleware and AuditAction helper.
--
-- Design notes:
--  - actor_id is nullable so anonymous/pre-auth requests (e.g. a rate-limited
--    /auth/login attempt) are still recorded without a FK constraint failure.
--  - request_body stores only scrubbed JSON (passwords/tokens stripped) with
--    a 4 KB cap — never raw file bytes or multipart payloads.
--  - This table is append-only: no UPDATE or DELETE should ever be issued
--    against it. Retention/archival policy (e.g. partition by month, move to
--    S3 after 90 days) is an operational concern outside the migration itself.

CREATE TABLE IF NOT EXISTS public.audit_log (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id        UUID        REFERENCES public.users(id) ON DELETE SET NULL,
    actor_role      TEXT,
    action          TEXT        NOT NULL,       -- e.g. 'curriculum.approve', 'post:/auth/login'
    resource_type   TEXT        NOT NULL,       -- URL path or domain noun ('upload_job', 'exam', etc.)
    resource_id     TEXT,                       -- the specific resource UUID/code acted on
    request_id      TEXT,                       -- X-Request-ID for log correlation
    ip_address      TEXT,
    user_agent      TEXT,
    request_body    JSONB,                      -- scrubbed; NULL for GET/DELETE and file uploads
    response_status INT,
    duration_ms     INT,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Query pattern: all actions by a specific actor, newest first.
CREATE INDEX IF NOT EXISTS idx_audit_log_actor
    ON public.audit_log (actor_id, occurred_at DESC);

-- Query pattern: all occurrences of a specific privileged action.
CREATE INDEX IF NOT EXISTS idx_audit_log_action
    ON public.audit_log (action, occurred_at DESC);

-- Query pattern: time-range queries (e.g. activity in the last 24 h).
CREATE INDEX IF NOT EXISTS idx_audit_log_occurred
    ON public.audit_log (occurred_at DESC);

COMMENT ON TABLE public.audit_log IS
    'Append-only record of every HTTP request and privileged business action. '
    'Never UPDATE or DELETE rows. Managed by the Go AuditLog middleware and AuditAction helper.';
