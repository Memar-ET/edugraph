-- V027__topic_hierarchy.sql
--
-- Adds Topic -> Subtopic nesting. curriculum.topics was previously a flat
-- list per unit; some source curricula (e.g. the Ethiopian MoE Biology
-- syllabus) structure content as numbered Topics with un-numbered
-- Subtopic bullets underneath each one. A subtopic is promoted as its own
-- topics row (so it's independently gap-analyzable/exam-alignable, same
-- as any other topic) with parent_topic_id pointing at its parent --
-- NULL for top-level topics, unchanged for all existing rows.

ALTER TABLE curriculum.topics
    ADD COLUMN parent_topic_id UUID REFERENCES curriculum.topics(id) ON DELETE CASCADE;

CREATE INDEX idx_topics_parent ON curriculum.topics(parent_topic_id) WHERE parent_topic_id IS NOT NULL;
