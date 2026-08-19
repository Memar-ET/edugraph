-- V051: Formal exam lifecycle state machine (Phase 2, production roadmap).
-- Adds lifecycle_status plus scheduling/snapshot/policy columns.
-- Existing "approved" exams are backfilled to "published"; everything else
-- becomes "draft" so existing code that checks status='published' is unaffected.

ALTER TABLE assessment.exams
  ADD COLUMN IF NOT EXISTS lifecycle_status TEXT NOT NULL DEFAULT 'draft'
    CHECK (lifecycle_status IN (
      'draft','parsing','validation','teacher_review',
      'approved','scheduled','published','closed','archived'
    )),
  ADD COLUMN IF NOT EXISTS published_at             TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS scheduled_at             TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS closed_at                TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS published_question_snapshot JSONB,
  ADD COLUMN IF NOT EXISTS published_qmatrix_version   TEXT,
  ADD COLUMN IF NOT EXISTS curriculum_version_code     TEXT,
  ADD COLUMN IF NOT EXISTS attempt_limit            INT  NOT NULL DEFAULT 1,
  ADD COLUMN IF NOT EXISTS time_limit_minutes       INT,
  ADD COLUMN IF NOT EXISTS eligibility_rule         JSONB;

-- Backfill: align lifecycle_status with the existing status column.
UPDATE assessment.exams
SET lifecycle_status = CASE
  WHEN status = 'approved' OR status = 'published' THEN 'published'
  WHEN status = 'parsing'                           THEN 'parsing'
  WHEN status = 'validated'                         THEN 'teacher_review'
  ELSE 'draft'
END;
