-- Capability 3A: The Granular Gap Analysis Engine.
--
-- Three storage layers, written by the ai-service gap worker after an
-- exam attempt is fully graded (queue:gap:analyze):
--
--   1. Granular  -- students.gap_records (existing V011 table, extended
--      here): one row per missed/partially-missed question, tracing
--      Question -> CLO -> symptom Topic -> (prerequisite walk) -> root
--      cause Topic, with a Gemini-written explanation.
--   2. Per-exam  -- students.exam_insights (new): one narrative summary
--      per graded attempt, so a past exam reads as a story, not a number.
--   3. Per-subject -- students.subject_profiles (new): rolling "health
--      score" per (student, subject) aggregated across every analyzed
--      attempt, regardless of which exam the data came from.
--
-- gap_records extensions are additive (never DROP, per migration rules):
--   * attempt_id / question_id pin the record to the exact mistake.
--     question_id is ON DELETE SET NULL because a re-parse of an exam
--     DELETEs and reinserts its question rows (see ai-service
--     postgres_assessment.save_exam_questions) -- the gap record should
--     survive that with its topic/explanation intact.
--   * topic_id (existing) is the SYMPTOM topic; root_cause_topic_id is
--     where the prerequisite walk found the break in the chain. NULL
--     means "no upstream break found -- the symptom is its own root
--     cause" (also the correct degraded behavior while
--     curriculum.topic_prerequisites is still unpopulated).

ALTER TABLE students.gap_records
    ADD COLUMN attempt_id          UUID REFERENCES assessment.exam_attempts(id) ON DELETE CASCADE,
    ADD COLUMN question_id         UUID REFERENCES assessment.questions(id) ON DELETE SET NULL,
    ADD COLUMN root_cause_topic_id UUID REFERENCES curriculum.topics(id),
    ADD COLUMN llm_explanation     TEXT;

CREATE INDEX idx_gap_records_attempt_id ON students.gap_records(attempt_id);

-- One row per fully-graded attempt. UNIQUE(attempt_id) makes re-analysis
-- (a re-queued job, or a teacher amending a grade which re-finalizes the
-- attempt) an upsert instead of a duplicate.
CREATE TABLE students.exam_insights (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id       UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    exam_id          UUID NOT NULL REFERENCES assessment.exams(id) ON DELETE CASCADE,
    attempt_id       UUID NOT NULL UNIQUE REFERENCES assessment.exam_attempts(id) ON DELETE CASCADE,
    school_id        UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    total_score      NUMERIC,
    percentage       NUMERIC,
    passed           BOOLEAN,
    gaps_found       INT NOT NULL DEFAULT 0,
    -- Bilingual narrative (English + Amharic) from Gemini; a deterministic
    -- computed summary when GEMINI_API_KEY isn't configured -- never NULL
    -- for an analyzed attempt.
    llm_exam_summary TEXT,
    llm_model        TEXT,
    generated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_exam_insights_student_id ON students.exam_insights(student_id);
CREATE INDEX idx_exam_insights_exam_id ON students.exam_insights(exam_id);

-- Rolling subject "health": recomputed from ALL of the student's graded
-- answers in the subject every time any attempt in it is analyzed.
-- top_weak_areas: JSON array of the lowest-mastery topics, e.g.
--   [{"topicId": "...", "title": "Vector Addition", "masteryPct": 32.5}]
CREATE TABLE students.subject_profiles (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id          UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    school_id           UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    subject_code        TEXT NOT NULL REFERENCES curriculum.subjects(code),
    grade_level         INT NOT NULL,
    current_mastery_pct NUMERIC NOT NULL CHECK (current_mastery_pct BETWEEN 0 AND 100),
    top_weak_areas      JSONB NOT NULL DEFAULT '[]'::jsonb,
    exams_analyzed      INT NOT NULL DEFAULT 0,
    last_updated        TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (student_id, subject_code)
);

CREATE INDEX idx_subject_profiles_student_id ON students.subject_profiles(student_id);
