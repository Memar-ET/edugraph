-- V011_updated_curriculum.sql
-- Replaces old flat curriculum tables with proper schema-based deep hierarchy

-- =====================================================
-- STEP 1: Archive old incorrect tables (backward-compatible)
-- =====================================================

-- Rename old tables to archive schema instead of dropping
ALTER TABLE IF EXISTS career_matches RENAME TO career_matches_v010;
ALTER TABLE IF EXISTS assessment_results RENAME TO assessment_results_v010;
ALTER TABLE IF EXISTS assessment_questions RENAME TO assessment_questions_v010;
ALTER TABLE IF EXISTS assessments RENAME TO assessments_v010;
ALTER TABLE IF EXISTS career_paths RENAME TO career_paths_v010;
ALTER TABLE IF EXISTS curriculum_units RENAME TO curriculum_units_v010;
ALTER TABLE IF EXISTS subjects RENAME TO subjects_v010;

-- =====================================================
-- STEP 2: Create curriculum schema
-- =====================================================

CREATE SCHEMA IF NOT EXISTS curriculum;

-- Upload jobs: tracks book parsing
CREATE TABLE curriculum.upload_jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    uploaded_by     UUID NOT NULL REFERENCES users(id),
    subject_code    TEXT NOT NULL,
    grade_level     INT NOT NULL CHECK (grade_level BETWEEN 1 AND 12),
    academic_year   TEXT NOT NULL,
    file_s3_key     TEXT NOT NULL,
    file_name       TEXT NOT NULL,
    file_size_bytes BIGINT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending' 
                    CHECK (status IN ('pending', 'parsing', 'parsed', 'review', 'approved', 'rejected', 'failed')),
    parse_error     TEXT,
    approved_by     UUID REFERENCES users(id),
    approved_at     TIMESTAMPTZ,
    neo4j_written   BOOLEAN DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Subjects: top-level container
CREATE TABLE curriculum.subjects (
    code            TEXT PRIMARY KEY,
    name_en         TEXT NOT NULL,
    name_am         TEXT,
    grade_level     INT NOT NULL CHECK (grade_level BETWEEN 1 AND 12),
    academic_year   TEXT NOT NULL,
    moe_code        TEXT,
    is_mandatory    BOOLEAN DEFAULT TRUE,
    upload_job_id   UUID REFERENCES curriculum.upload_jobs(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Units: chapters inside subjects
CREATE TABLE curriculum.units (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_code    TEXT NOT NULL REFERENCES curriculum.subjects(code) ON DELETE CASCADE,
    grade_level     INT NOT NULL,
    number          INT NOT NULL,
    title_en        TEXT NOT NULL,
    title_am        TEXT,
    neo4j_node_id   TEXT UNIQUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Topics: THE BRAIN - actual concepts taught
CREATE TABLE curriculum.topics (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    unit_id         UUID NOT NULL REFERENCES curriculum.units(id) ON DELETE CASCADE,
    subject_code    TEXT NOT NULL REFERENCES curriculum.subjects(code),
    grade_level     INT NOT NULL,
    sequence_order  INT NOT NULL,
    title_en        TEXT NOT NULL,
    title_am        TEXT,
    description     TEXT,
    estimated_hours NUMERIC DEFAULT 2.0,
    exam_weight     NUMERIC DEFAULT 1.0 CHECK (exam_weight BETWEEN 0 AND 5),
    bloom_level     TEXT CHECK (bloom_level IN ('remember', 'understand', 'apply', 'analyse', 'evaluate', 'create')),
    key_concepts    TEXT[] DEFAULT '{}',
    neo4j_node_id   TEXT UNIQUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_topics_subject_grade ON curriculum.topics(subject_code, grade_level);
CREATE INDEX idx_topics_unit_id ON curriculum.topics(unit_id);

-- Topic Prerequisites: cross-grade dependency graph
CREATE TABLE curriculum.topic_prerequisites (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    topic_id        UUID NOT NULL REFERENCES curriculum.topics(id) ON DELETE CASCADE,
    prerequisite_id UUID NOT NULL REFERENCES curriculum.topics(id) ON DELETE CASCADE,
    weight          NUMERIC DEFAULT 1.0 CHECK (weight BETWEEN 0 AND 1),
    is_cross_grade  BOOLEAN DEFAULT FALSE,
    infer_method    TEXT NOT NULL CHECK (infer_method IN ('explicit', 'ai_inferred', 'manual', 'moe_document')),
    reviewed_by     UUID REFERENCES users(id),
    confirmed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (topic_id, prerequisite_id)
);

-- CLOs: Curriculum Learning Outcomes
CREATE TABLE curriculum.clos (
    code            TEXT PRIMARY KEY,
    subject_code    TEXT NOT NULL REFERENCES curriculum.subjects(code),
    grade_level     INT NOT NULL,
    description_en  TEXT NOT NULL,
    description_am  TEXT,
    bloom_level     TEXT,
    is_mandatory    BOOLEAN DEFAULT TRUE,
    moe_version     TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Topic-CLO Mappings
CREATE TABLE curriculum.topic_clo_mappings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    topic_id        UUID NOT NULL REFERENCES curriculum.topics(id) ON DELETE CASCADE,
    clo_code        TEXT NOT NULL REFERENCES curriculum.clos(code) ON DELETE CASCADE,
    alignment_score NUMERIC CHECK (alignment_score BETWEEN 0 AND 1),
    match_method    TEXT NOT NULL CHECK (match_method IN ('embedding_auto', 'human_confirmed', 'ai_draft', 'manual')),
    reviewed_by     UUID REFERENCES users(id),
    confirmed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (topic_id, clo_code)
);

-- =====================================================
-- STEP 3: Create assessment schema
-- =====================================================

CREATE SCHEMA IF NOT EXISTS assessment;

-- Exams
CREATE TABLE assessment.exams (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_by           UUID NOT NULL REFERENCES users(id),
    school_id            UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    subject_code         TEXT NOT NULL REFERENCES curriculum.subjects(code),
    grade_level          INT NOT NULL CHECK (grade_level BETWEEN 1 AND 12),
    academic_year        TEXT NOT NULL,
    term                 INT CHECK (term BETWEEN 1 AND 4),
    title                TEXT NOT NULL,
    total_marks          INT NOT NULL,
    duration_mins        INT,
    due_date             DATE,
    status               TEXT NOT NULL DEFAULT 'draft' 
                         CHECK (status IN ('draft', 'validation_pending', 'published', 'closed')),
    validation_report_s3 TEXT,
    file_s3_key          TEXT,
    neo4j_node_id        TEXT UNIQUE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_assessments_school_id ON assessment.exams(school_id);
CREATE INDEX idx_assessments_subject_id ON assessment.exams(subject_code, grade_level);

-- Questions
CREATE TABLE assessment.questions (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_id           UUID NOT NULL REFERENCES assessment.exams(id) ON DELETE CASCADE,
    school_id         UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    sequence_number   INT NOT NULL,
    question_text     TEXT NOT NULL,
    question_type     TEXT NOT NULL CHECK (question_type IN ('mcq', 'short_answer', 'long_answer', 'essay', 'calculation')),
    marks             INT NOT NULL,
    difficulty_level  TEXT CHECK (difficulty_level IN ('easy', 'medium', 'hard')),
    topic_id          UUID REFERENCES curriculum.topics(id),
    clo_code          TEXT REFERENCES curriculum.clos(code),
    clo_align_score   NUMERIC,
    clo_align_method  TEXT CHECK (clo_align_method IN ('embedding_auto', 'teacher_confirmed')),
    answer_key        JSONB,
    neo4j_node_id     TEXT UNIQUE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_questions_exam_id ON assessment.questions(exam_id);
CREATE INDEX idx_questions_topic_id ON assessment.questions(topic_id);
CREATE INDEX idx_questions_clo_code ON assessment.questions(clo_code);

-- Exam Attempts
CREATE TABLE assessment.exam_attempts (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id    UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    exam_id       UUID NOT NULL REFERENCES assessment.exams(id) ON DELETE CASCADE,
    school_id     UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    attempt_number INT DEFAULT 1,
    started_at    TIMESTAMPTZ,
    submitted_at  TIMESTAMPTZ,
    total_score   NUMERIC,
    percentage    NUMERIC,
    passed        BOOLEAN,
    graded_at     TIMESTAMPTZ,
    graded_by     UUID REFERENCES users(id),
    is_offline    BOOLEAN DEFAULT FALSE,
    synced_at     TIMESTAMPTZ,
    neo4j_written BOOLEAN DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_exam_attempts_student_id ON assessment.exam_attempts(student_id);
CREATE INDEX idx_exam_attempts_exam_id ON assessment.exam_attempts(exam_id);

-- Student Answers (most heavily written table)
CREATE TABLE assessment.student_answers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    attempt_id      UUID NOT NULL REFERENCES assessment.exam_attempts(id) ON DELETE CASCADE,
    student_id      UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    question_id     UUID NOT NULL REFERENCES assessment.questions(id) ON DELETE CASCADE,
    school_id       UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    answer_text     TEXT,
    answer_data     JSONB,
    marks_awarded   NUMERIC,
    marks_possible  INT NOT NULL,
    passed          BOOLEAN,
    time_spent_secs INT,
    grading_notes   TEXT,
    neo4j_written   BOOLEAN DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_answers_attempt_id ON assessment.student_answers(attempt_id);
CREATE INDEX idx_answers_student_id ON assessment.student_answers(student_id);
CREATE INDEX idx_answers_question_id ON assessment.student_answers(question_id);
CREATE INDEX idx_answers_school_id_passed ON assessment.student_answers(school_id, passed);
CREATE INDEX idx_answers_neo4j ON assessment.student_answers(neo4j_written) WHERE NOT neo4j_written;

-- =====================================================
-- STEP 4: Create careers schema
-- =====================================================

CREATE SCHEMA IF NOT EXISTS careers;

-- Careers
CREATE TABLE careers.careers (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name_en            TEXT NOT NULL,
    name_am            TEXT,
    sector             TEXT NOT NULL,
    min_edu_level      TEXT NOT NULL,
    description        TEXT,
    moe_classification TEXT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Career-Topic Requirements
CREATE TABLE careers.career_topic_requirements (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    career_id   UUID NOT NULL REFERENCES careers.careers(id) ON DELETE CASCADE,
    topic_id    UUID NOT NULL REFERENCES curriculum.topics(id) ON DELETE CASCADE,
    importance  TEXT NOT NULL CHECK (importance IN ('critical', 'important', 'helpful')),
    neo4j_written BOOLEAN DEFAULT FALSE,
    UNIQUE (career_id, topic_id)
);

-- Career Matches (student recommendations)
CREATE TABLE careers.career_matches (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id      UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    career_path_id  UUID NOT NULL REFERENCES careers.careers(id) ON DELETE CASCADE,
    match_score     NUMERIC(5, 4) NOT NULL,
    generated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (student_id, career_path_id)
);

CREATE INDEX idx_career_matches_student_id ON careers.career_matches(student_id);

-- =====================================================
-- STEP 5: Create embeddings schema (pgvector)
-- =====================================================

CREATE SCHEMA IF NOT EXISTS embeddings;

-- CLO Embeddings (FIXED: 768 dimensions, not 1024!)
CREATE TABLE embeddings.clo_embeddings (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clo_code    TEXT UNIQUE NOT NULL REFERENCES curriculum.clos(code) ON DELETE CASCADE,
    embedding   vector(768) NOT NULL,  -- ← CRITICAL FIX: multilingual-e5-large outputs 768
    model_ver   TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Question Embeddings
CREATE TABLE embeddings.question_embeddings (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id UUID UNIQUE NOT NULL REFERENCES assessment.questions(id) ON DELETE CASCADE,
    embedding   vector(768) NOT NULL,  -- ← CRITICAL FIX
    model_ver   TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- HNSW indexes for fast similarity search
CREATE INDEX idx_clo_emb_hnsw ON embeddings.clo_embeddings 
    USING hnsw (embedding vector_cosine_ops);
CREATE INDEX idx_q_emb_hnsw ON embeddings.question_embeddings 
    USING hnsw (embedding vector_cosine_ops);

-- =====================================================
-- STEP 6: Create students schema additions
-- =====================================================

CREATE SCHEMA IF NOT EXISTS students;

-- Gap Records (output of gap analysis)
CREATE TABLE students.gap_records (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id          UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    school_id           UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    topic_id            UUID NOT NULL REFERENCES curriculum.topics(id) ON DELETE CASCADE,
    clo_code            TEXT REFERENCES curriculum.clos(code),
    severity_score      NUMERIC NOT NULL CHECK (severity_score BETWEEN 0 AND 1),
    is_root_cause       BOOLEAN DEFAULT FALSE,
    prerequisite_depth  INT DEFAULT 0,
    detected_in_exam    UUID REFERENCES assessment.exams(id),
    detected_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at         TIMESTAMPTZ,
    neo4j_written       BOOLEAN DEFAULT FALSE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_gap_records_student_id ON students.gap_records(student_id);
CREATE INDEX idx_gap_records_topic_id ON students.gap_records(topic_id);
CREATE INDEX idx_gap_records_neo4j ON students.gap_records(neo4j_written) WHERE NOT neo4j_written;

-- Mastery Records
CREATE TABLE students.mastery_records (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id    UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    school_id     UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    topic_id      UUID NOT NULL REFERENCES curriculum.topics(id) ON DELETE CASCADE,
    confidence    NUMERIC CHECK (confidence BETWEEN 0 AND 1),
    mastered_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    neo4j_written BOOLEAN DEFAULT FALSE,
    UNIQUE (student_id, topic_id)
);

-- Study Plans
CREATE TABLE students.study_plans (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id        UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    school_id         UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    target_exam_id    UUID REFERENCES assessment.exams(id),
    target_career_id  UUID REFERENCES careers.careers(id),
    plan_data         JSONB NOT NULL,
    total_days        INT NOT NULL,
    total_hours       NUMERIC NOT NULL,
    language          TEXT DEFAULT 'am',
    generated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at        TIMESTAMPTZ,
    is_active         BOOLEAN DEFAULT TRUE
);

CREATE INDEX idx_study_plans_student_id ON students.study_plans(student_id);
