-- Phase 12: Report generation results table.
-- Reports are generated asynchronously by ai-service's report_worker.py
-- (triggered by queue:report:generate). The Go API creates a pending row
-- (POST /api/v1/reports/generate) and the worker updates it to done/failed.

CREATE TABLE IF NOT EXISTS public.report_results (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    report_type     TEXT        NOT NULL,   -- 'school_monthly' | 'national_heatmap' | 'clo_coverage'
    params          JSONB,                  -- input parameters (school_id, year, month, etc.)
    result          JSONB,                  -- structured report payload, written by the worker
    requester_id    UUID        REFERENCES public.users(id) ON DELETE SET NULL,
    status          TEXT        NOT NULL DEFAULT 'pending'
                                CHECK (status IN ('pending', 'running', 'done', 'failed')),
    error_text      TEXT,
    generated_at    TIMESTAMPTZ,            -- set when status → done
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_report_results_requester  ON public.report_results(requester_id, created_at DESC);
CREATE INDEX idx_report_results_type_status ON public.report_results(report_type, status, created_at DESC);
