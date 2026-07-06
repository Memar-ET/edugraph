-- V015__seed_demo_data.sql
-- Seeds the database with initial regions, schools, and demo users.
-- Uses pgcrypto to generate valid bcrypt hashes for 'password123'.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- 1. Seed Region
INSERT INTO regions.regions (id, name, code) VALUES
('b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a01', 'Addis Ababa', 'AA')
ON CONFLICT (code) DO NOTHING;

-- 2. Seed School
INSERT INTO schools.schools (id, region_id, name, moe_school_code, school_type, connectivity, has_school_box) VALUES
('c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a01', 'b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a01', 'AASTU Demonstration School', 'MOE-AA-001', 'public', 'reliable', false)
ON CONFLICT (moe_school_code) DO NOTHING;

-- 3. Seed Users
INSERT INTO identity.users (id, email, password_hash, role, full_name, phone, is_active, mfa_enabled, preferred_lang, school_id, region_id) VALUES
('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'ministry.admin@edugraph.et', crypt('password123', gen_salt('bf', 12)), 'ministry_admin', 'Dr. Abebe Kebede', '+251911234567', true, false, 'am', NULL, NULL),
('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', 'curriculum.officer@edugraph.et', crypt('password123', gen_salt('bf', 12)), 'curriculum_officer', 'Selamawit Tesfaye', '+251911234568', true, false, 'am', NULL, NULL),
('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13', 'regional.admin@edugraph.et', crypt('password123', gen_salt('bf', 12)), 'regional_admin', 'Mohammed Ahmed', '+251911234569', true, false, 'am', NULL, 'b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a01'),
('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a14', 'school.admin@edugraph.et', crypt('password123', gen_salt('bf', 12)), 'school_admin', 'Tigist Bekele', '+251911234570', true, false, 'am', 'c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a01', NULL),
('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a15', 'teacher@edugraph.et', crypt('password123', gen_salt('bf', 12)), 'teacher', 'Dawit Lemma', '+251911234571', true, false, 'am', 'c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a01', NULL),
('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a16', 'student@edugraph.et', crypt('password123', gen_salt('bf', 12)), 'student', 'Hanna Solomon', '+251911234572', true, false, 'am', 'c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a01', NULL)
ON CONFLICT (email) DO NOTHING;

-- 4. Seed Student Profile for the student user
INSERT INTO students.student_profiles (id, school_id, student_number, grade_level, section, enrolled_at, daily_study_hrs) VALUES
('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a16', 'c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a01', 'STU-2026-001', 10, 'A', '2026-09-01', 2.0)
ON CONFLICT (id) DO NOTHING;