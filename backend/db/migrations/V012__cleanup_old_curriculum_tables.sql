-- V012__cleanup_old_curriculum_tables.sql
-- Drops archived old curriculum tables after data migration to new schema
-- 
-- NOTE: This migration is intentionally separate from V011 to enable:
-- 1. Verification of data migration before cleanup
-- 2. Safe rollback if issues arise
-- 3. Zero-downtime deployments (old code can still read old tables during transition)
--
-- Prerequisites: V011 has renamed old tables to *_v010 archive versions.
-- All application code must be updated to use new curriculum.*, assessment.*,
-- and careers.* schema tables before running this migration.

-- Drop old archived flat curriculum tables (in correct dependency order)
DROP TABLE IF EXISTS career_matches_v010 CASCADE;
DROP TABLE IF EXISTS career_paths_v010 CASCADE;
DROP TABLE IF EXISTS assessment_results_v010 CASCADE;
DROP TABLE IF EXISTS assessment_questions_v010 CASCADE;
DROP TABLE IF EXISTS assessments_v010 CASCADE;
DROP TABLE IF EXISTS curriculum_units_v010 CASCADE;
DROP TABLE IF EXISTS subjects_v010 CASCADE;
