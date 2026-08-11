-- V014__add_curriculum_officer_role.sql
-- The curriculum upload workflow (see internal/curriculum) and the API
-- router restrict POST /api/v1/curriculum/upload to the "curriculum_officer"
-- and "ministry_admin" roles, but the original user_role enum (V001) only
-- defined: student, teacher, school_admin, regional_admin, ministry_admin.
-- This adds the missing role so accounts can actually be created with it.
--
-- Note: ALTER TYPE ... ADD VALUE must be the only statement running in its
-- transaction on PostgreSQL < 12; Flyway runs each migration file in its own
-- transaction by default, so this is safe as the sole statement here.
--
-- IMPORTANT: this file's *version number* must stay purely numeric (014, not
-- 014A/014_1/etc. mixed with letters in a way Flyway won't parse) — a
-- previous attempt named this "V014A__..." and Flyway silently skipped it
-- as an unrecognized migration, so the enum value was never actually added
-- and the next migration (seeding a curriculum_officer user) failed with
-- "invalid input value for enum user_role".

ALTER TYPE user_role ADD VALUE IF NOT EXISTS 'curriculum_officer';
