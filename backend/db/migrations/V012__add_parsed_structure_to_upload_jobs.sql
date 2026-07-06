-- V012__add_parsed_structure_to_upload_jobs.sql
-- Adds a column to hold the AI-parsed curriculum structure (units, topics, etc.)
-- before human review and approval.

ALTER TABLE curriculum.upload_jobs
    ADD COLUMN parsed_structure JSONB;

COMMENT ON COLUMN curriculum.upload_jobs.parsed_structure IS
    'AI-extracted curriculum hierarchy (units, topics, subtopics) from the uploaded book. 
     Populated by the AI parsing service. Null until parsing completes. 
     Curriculum officer reviews this structure before approval.';