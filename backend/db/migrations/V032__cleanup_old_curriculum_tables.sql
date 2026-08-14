-- Originally V012__cleanup_old_curriculum_tables.sql -- renumbered to
-- V032 because it collided with the already-existing, already-applied
-- V012__add_parsed_structure_to_upload_jobs.sql (two files can't share
-- a Flyway version). This migration had never successfully applied
-- anywhere under either number -- confirmed by the collision error
-- itself, Flyway refuses to run when two migrations claim the same
-- version -- so renumbering it is the same exception CLAUDE.md already
-- documents for V013 ("never edit a merged migration" applies to
-- migrations that have actually run somewhere; this one hadn't).
--
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
--
-- career_matches_v010/career_paths_v010 are deliberately NOT dropped
-- here, unlike the original version of this migration: that precondition
-- was never actually met for career matching -- internal/career/repository
-- /repository.go still queries those two tables by name today, on
-- purpose (see its own doc comment), because the newer
-- careers.careers/careers.career_matches schema is a structurally
-- different topic-requirement-based matching model with no curation UI
-- yet, and migrating to it was explicitly scoped out as separate,
-- larger future work. Dropping these out from under that repository is
-- what caused every career-matching query to fail outright with
-- "relation does not exist" when this migration first ran (under its
-- original V012 number, before the collision was found).
DROP TABLE IF EXISTS assessment_results_v010 CASCADE;
DROP TABLE IF EXISTS assessment_questions_v010 CASCADE;
DROP TABLE IF EXISTS assessments_v010 CASCADE;
DROP TABLE IF EXISTS curriculum_units_v010 CASCADE;
DROP TABLE IF EXISTS subjects_v010 CASCADE;
