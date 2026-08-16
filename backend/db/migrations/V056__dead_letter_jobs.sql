-- Phase 7 reliability: dead-letter storage for failed AI worker jobs.
-- Every BRPOP worker (curriculum, exam, gap, kt, etc.) previously logged
-- and dropped failed jobs. This table persists them so they can be
-- inspected and retried (up to MAX_ATTEMPTS=3, see app/db/dead_letter.py).
-- refit_worker.run_once() re-queues eligible rows nightly.

CREATE TABLE IF NOT EXISTS public.dead_letter_jobs (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    queue_name          TEXT        NOT NULL,
    job_payload         TEXT        NOT NULL,
    error_message       TEXT,
    attempt_count       INT         NOT NULL DEFAULT 1,
    last_attempted_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at         TIMESTAMPTZ,            -- set when successfully re-queued or manually dismissed
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (queue_name, job_payload)            -- idempotent upsert in dead_letter.push_dead_letter
);

CREATE INDEX idx_dead_letter_retriable ON public.dead_letter_jobs(queue_name, last_attempted_at)
    WHERE resolved_at IS NULL;
