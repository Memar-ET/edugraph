-- V016__curriculum_unique_constraints.sql
-- The approval workflow (POST /api/v1/curriculum/jobs/{id}/approve) promotes
-- the AI-parsed structure into curriculum.units/topics via
-- INSERT ... ON CONFLICT ... DO UPDATE, so a job can be safely re-approved
-- (e.g. after the officer edits the tree and approves again) without
-- creating duplicate rows. That requires uniqueness on the natural key of
-- each table, which wasn't there before.

ALTER TABLE curriculum.units
    ADD CONSTRAINT uq_units_subject_number UNIQUE (subject_code, number);

ALTER TABLE curriculum.topics
    ADD CONSTRAINT uq_topics_unit_sequence UNIQUE (unit_id, sequence_order);
