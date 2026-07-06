-- This removes the failed migration record so Flyway can try again
DELETE FROM flyway_schema_history WHERE success = false;

-- V014__create_missing_schemas.sql
-- Creates missing schemas and tables without dropping existing ones.
-- Uses IF NOT EXISTS to ensure it is safe to run multiple times.

-- 1. Create Schemas
CREATE SCHEMA IF NOT EXISTS identity;
CREATE SCHEMA IF NOT EXISTS regions;
CREATE SCHEMA IF NOT EXISTS schools;
CREATE SCHEMA IF NOT EXISTS students;
CREATE SCHEMA IF NOT EXISTS assessment;
CREATE SCHEMA IF NOT EXISTS careers;
CREATE SCHEMA IF NOT EXISTS embeddings;
CREATE SCHEMA IF NOT EXISTS sync;
CREATE SCHEMA IF NOT EXISTS audit;
CREATE SCHEMA IF NOT EXISTS jobs;
CREATE SCHEMA IF NOT EXISTS ministry;
CREATE SCHEMA IF NOT EXISTS notifications;
CREATE SCHEMA IF NOT EXISTS config;

-- 2. Identity
CREATE TABLE IF NOT EXISTS identity.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT UNIQUE NOT NULL,
    phone TEXT UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('student', 'teacher', 'school_admin', 'regional_admin', 'ministry_admin', 'curriculum_officer', 'super_admin')),
    school_id UUID,
    region_id UUID,
    full_name TEXT NOT NULL,
    preferred_lang TEXT DEFAULT 'am' CHECK (preferred_lang IN ('am', 'en', 'om')),
    is_active BOOLEAN DEFAULT true NOT NULL,
    mfa_enabled BOOLEAN DEFAULT false NOT NULL,
    mfa_secret TEXT,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT now() NOT NULL
);

-- 3. Regions
CREATE TABLE IF NOT EXISTS regions.regions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    code TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- 4. Schools
CREATE TABLE IF NOT EXISTS schools.schools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    region_id UUID NOT NULL REFERENCES regions.regions(id),
    name TEXT NOT NULL,
    moe_school_code TEXT UNIQUE NOT NULL,
    school_type TEXT NOT NULL CHECK (school_type IN ('public', 'private', 'community')),
    connectivity TEXT DEFAULT 'none' CHECK (connectivity IN ('none', 'intermittent', 'reliable')),
    has_school_box BOOLEAN DEFAULT false,
    school_box_id TEXT UNIQUE,
    location_lat NUMERIC,
    location_lng NUMERIC,
    woreda TEXT,
    zone TEXT,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Safely add Foreign Keys to identity.users now that regions/schools exist
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_users_region') THEN
        ALTER TABLE identity.users ADD CONSTRAINT fk_users_region FOREIGN KEY (region_id) REFERENCES regions.regions(id) ON DELETE SET NULL;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_users_school') THEN
        ALTER TABLE identity.users ADD CONSTRAINT fk_users_school FOREIGN KEY (school_id) REFERENCES schools.schools(id) ON DELETE SET NULL;
    END IF;
END $$;

-- 5. Students
CREATE TABLE IF NOT EXISTS students.student_profiles (
    id UUID PRIMARY KEY REFERENCES identity.users(id) ON DELETE CASCADE,
    school_id UUID NOT NULL REFERENCES schools.schools(id),
    student_number TEXT UNIQUE NOT NULL,
    national_id TEXT,
    grade_level INT NOT NULL CHECK (grade_level BETWEEN 1 AND 12),
    section TEXT,
    date_of_birth DATE,
    gender TEXT,
    disability_flags TEXT[] DEFAULT '{}',
    parent_contact JSONB,
    career_interests TEXT[] DEFAULT '{}',
    daily_study_hrs NUMERIC DEFAULT 2.0,
    enrolled_at DATE NOT NULL,
    graduated_at DATE,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- 6. Assessment
CREATE TABLE IF NOT EXISTS assessment.exams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_by UUID NOT NULL REFERENCES identity.users(id),
    school_id UUID NOT NULL REFERENCES schools.schools(id) ON DELETE CASCADE,
    subject_code TEXT NOT NULL,
    grade_level INT NOT NULL CHECK (grade_level BETWEEN 1 AND 12),
    academic_year TEXT NOT NULL,
    term INT CHECK (term BETWEEN 1 AND 4),
    title TEXT NOT NULL,
    total_marks INT NOT NULL,
    duration_mins INT,
    due_date DATE,
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'validation_pending', 'published', 'closed')),
    validation_report_s3 TEXT,
    file_s3_key TEXT,
    neo4j_node_id TEXT UNIQUE,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS assessment.questions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_id UUID NOT NULL REFERENCES assessment.exams(id) ON DELETE CASCADE,
    school_id UUID NOT NULL REFERENCES schools.schools(id) ON DELETE CASCADE,
    sequence_number INT NOT NULL,
    question_text TEXT NOT NULL,
    question_type TEXT NOT NULL CHECK (question_type IN ('mcq', 'short_answer', 'long_answer', 'essay', 'calculation')),
    marks INT NOT NULL,
    difficulty_level TEXT CHECK (difficulty_level IN ('easy', 'medium', 'hard')),
    topic_id UUID,
    clo_code TEXT,
    clo_align_score NUMERIC,
    clo_align_method TEXT CHECK (clo_align_method IN ('embedding_auto', 'teacher_confirmed')),
    answer_key JSONB,
    neo4j_node_id TEXT UNIQUE,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS assessment.exam_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students.student_profiles(id) ON DELETE CASCADE,
    exam_id UUID NOT NULL REFERENCES assessment.exams(id) ON DELETE CASCADE,
    school_id UUID NOT NULL REFERENCES schools.schools(id) ON DELETE CASCADE,
    attempt_number INT DEFAULT 1,
    started_at TIMESTAMPTZ,
    submitted_at TIMESTAMPTZ,
    total_score NUMERIC,
    percentage NUMERIC,
    passed BOOLEAN,
    graded_at TIMESTAMPTZ,
    graded_by UUID REFERENCES identity.users(id),
    is_offline BOOLEAN DEFAULT false,
    synced_at TIMESTAMPTZ,
    neo4j_written BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS assessment.student_answers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    attempt_id UUID NOT NULL REFERENCES assessment.exam_attempts(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES students.student_profiles(id) ON DELETE CASCADE,
    question_id UUID NOT NULL REFERENCES assessment.questions(id) ON DELETE CASCADE,
    school_id UUID NOT NULL REFERENCES schools.schools(id) ON DELETE CASCADE,
    answer_text TEXT,
    answer_data JSONB,
    marks_awarded NUMERIC,
    marks_possible INT NOT NULL,
    passed BOOLEAN,
    time_spent_secs INT,
    grading_notes TEXT,
    neo4j_written BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

-- 7. Students Gap/Mastery/Study Plans
CREATE TABLE IF NOT EXISTS students.gap_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students.student_profiles(id) ON DELETE CASCADE,
    school_id UUID NOT NULL REFERENCES schools.schools(id),
    topic_id UUID NOT NULL,
    clo_code TEXT,
    severity_score NUMERIC NOT NULL CHECK (severity_score BETWEEN 0 AND 1),
    is_root_cause BOOLEAN DEFAULT false,
    prerequisite_depth INT DEFAULT 0,
    detected_in_exam UUID REFERENCES assessment.exams(id) ON DELETE SET NULL,
    detected_at TIMESTAMPTZ NOT NULL,
    resolved_at TIMESTAMPTZ,
    neo4j_written BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS students.mastery_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students.student_profiles(id) ON DELETE CASCADE,
    school_id UUID NOT NULL REFERENCES schools.schools(id),
    topic_id UUID NOT NULL,
    confidence NUMERIC CHECK (confidence BETWEEN 0 AND 1),
    mastered_at TIMESTAMPTZ NOT NULL,
    neo4j_written BOOLEAN DEFAULT false,
    UNIQUE(student_id, topic_id)
);

CREATE TABLE IF NOT EXISTS students.study_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL REFERENCES students.student_profiles(id) ON DELETE CASCADE,
    school_id UUID NOT NULL REFERENCES schools.schools(id),
    target_exam_id UUID REFERENCES assessment.exams(id) ON DELETE SET NULL,
    target_career_id UUID,
    plan_data JSONB NOT NULL,
    total_days INT NOT NULL,
    total_hours NUMERIC NOT NULL,
    language TEXT DEFAULT 'am',
    generated_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ,
    is_active BOOLEAN DEFAULT true
);