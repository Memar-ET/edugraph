-- V026__prerequisite_validation_and_sync.sql
--
-- curriculum.topic_prerequisites was the one best-effort-synced-to-Neo4j
-- table in this schema without a way to track whether that sync actually
-- succeeded -- every other one does (upload_jobs, assessment.exam_attempts,
-- students.gap_records, students.mastery_records, assessment.student_answers,
-- careers.career_topic_requirements all have neo4j_written). A failed or
-- stale sync here had no way to be detected or retried; see
-- Repository.ListUnsyncedPrerequisites / Service.ResyncPrerequisitesToNeo4j.
--
-- infer_method (V011) already distinguishes provenance and
-- confirmed_at/reviewed_by (V011) already distinguish "validated" from
-- "not yet reviewed" -- but the only write path (AddTopicPrerequisite)
-- always confirmed a link immediately on creation, so "inferred, pending
-- validation" was never a reachable state. No schema change needed for
-- that half of this feature; see repository/service/handler/prerequisites.go.

ALTER TABLE curriculum.topic_prerequisites
    ADD COLUMN neo4j_written BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX idx_topic_prereq_neo4j ON curriculum.topic_prerequisites(neo4j_written) WHERE NOT neo4j_written;
