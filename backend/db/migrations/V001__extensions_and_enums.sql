-- Extensions
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "vector";

-- Shared enums
CREATE TYPE user_role AS ENUM (
    'student',
    'teacher',
    'school_admin',
    'regional_admin',
    'ministry_admin'
);

CREATE TYPE assessment_type AS ENUM ('quiz', 'exam', 'national');

CREATE TYPE sync_operation AS ENUM ('create', 'update', 'delete');

CREATE TYPE sync_status AS ENUM ('pending', 'applied', 'conflict', 'rejected');

CREATE TYPE job_status AS ENUM ('pending', 'running', 'completed', 'failed');
