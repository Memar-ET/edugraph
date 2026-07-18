-- Capability 4B (Exam Quality Scoring) + 4C (School Quality Scoring).

-- 4B: recalibrated difficulty lives NEXT TO the stated one -- the
-- teacher/parser-stated difficulty_level is never overwritten, the
-- performance-derived value is what future exams should consult.
ALTER TABLE assessment.questions ADD COLUMN calibrated_difficulty TEXT;

-- 4B: one persisted quality report per exam, recomputed on demand.
-- Scalar columns are what 4C aggregates (avg discrimination across a
-- school's exams); the full per-question breakdown stays in JSONB.
CREATE TABLE assessment.exam_quality_reports (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_id            UUID NOT NULL UNIQUE REFERENCES assessment.exams(id) ON DELETE CASCADE,
    school_id          UUID NOT NULL REFERENCES public.schools(id) ON DELETE CASCADE,
    attempts_analyzed  INT NOT NULL DEFAULT 0,
    -- Mean discrimination index across scorable questions, -1..1.
    -- NULL when no question had enough graded answers to compute one.
    discrimination_avg NUMERIC,
    -- 0..100 composite for this exam (see service.computeExamQuality).
    quality_score      NUMERIC,
    report             JSONB NOT NULL,
    computed_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_exam_quality_reports_school ON assessment.exam_quality_reports(school_id);

-- 4C: per-school/subject/grade composite health scores.
CREATE SCHEMA IF NOT EXISTS schools;

CREATE TABLE schools.quality_scores (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    school_id             UUID NOT NULL REFERENCES public.schools(id) ON DELETE CASCADE,
    subject_code          TEXT NOT NULL REFERENCES curriculum.subjects(code),
    grade_level           INT NOT NULL CHECK (grade_level BETWEEN 1 AND 12),
    -- Component metrics, all 0..100 (see the 0.30/0.25/0.25/0.20
    -- weighted formula in the impl plan / service.computeSchoolQuality).
    clo_coverage_pct      NUMERIC NOT NULL,
    student_mastery_pct   NUMERIC NOT NULL,
    exam_quality_avg      NUMERIC NOT NULL,
    curriculum_compliance NUMERIC NOT NULL,
    composite_score       NUMERIC NOT NULL,
    flagged_for_review    BOOLEAN NOT NULL DEFAULT false,
    computed_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (school_id, subject_code, grade_level)
);

CREATE INDEX idx_quality_scores_school ON schools.quality_scores(school_id);
