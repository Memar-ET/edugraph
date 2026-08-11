-- V015__seed_demo_data.sql
-- Seeds a demo region, school, and one user per role into the tables the
-- application actually queries (users/regions/schools/teachers/students,
-- see V002-V004). All demo accounts use the password "password123".
--
-- The previous version of this migration seeded a disconnected
-- "identity.users" / "regions.regions" / "schools.schools" schema that the
-- Go code never reads from — those tables have been removed (see V014).

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- 1. Region
INSERT INTO regions (id, name, code) VALUES
    ('b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a01', 'Addis Ababa', 'AA')
ON CONFLICT (code) DO NOTHING;

-- 2. School
INSERT INTO schools (id, region_id, name, code, address) VALUES
    ('c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a01',
     'b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a01',
     'AASTU Demonstration School',
     'MOE-AA-001',
     'Addis Ababa, Ethiopia')
ON CONFLICT (code) DO NOTHING;

-- 3. Users (one per role, password: "password123")
INSERT INTO users (id, email, password_hash, role, full_name, phone, is_active, region_id, school_id) VALUES
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'ministry.admin@edugraph.et',    crypt('password123', gen_salt('bf', 12)), 'ministry_admin',     'Dr. Abebe Kebede',   '+251911234567', true, NULL, NULL),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', 'curriculum.officer@edugraph.et', crypt('password123', gen_salt('bf', 12)), 'curriculum_officer', 'Selamawit Tesfaye',  '+251911234568', true, NULL, NULL),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13', 'regional.admin@edugraph.et',    crypt('password123', gen_salt('bf', 12)), 'regional_admin',     'Mohammed Ahmed',     '+251911234569', true, 'b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a01', NULL),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a14', 'school.admin@edugraph.et',      crypt('password123', gen_salt('bf', 12)), 'school_admin',       'Tigist Bekele',      '+251911234570', true, NULL, 'c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a01'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a15', 'teacher@edugraph.et',           crypt('password123', gen_salt('bf', 12)), 'teacher',            'Dawit Lemma',        '+251911234571', true, NULL, 'c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a01'),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a16', 'student@edugraph.et',           crypt('password123', gen_salt('bf', 12)), 'student',            'Hanna Solomon',      '+251911234572', true, NULL, 'c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a01')
ON CONFLICT (email) DO NOTHING;

-- 4. Teacher profile
INSERT INTO teachers (user_id, school_id, subject_specialty) VALUES
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a15', 'c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a01', 'Mathematics')
ON CONFLICT (user_id) DO NOTHING;

-- 5. Student profile
INSERT INTO students (user_id, school_id, admission_no, grade_level) VALUES
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a16', 'c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a01', 'STU-2026-001', 10)
ON CONFLICT (user_id) DO NOTHING;
