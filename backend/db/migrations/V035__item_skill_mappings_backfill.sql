-- EG-GCKT Milestone 0: backfill assessment.item_skill_mappings from the
-- single-skill pointer every existing question already has
-- (questions.topic_id/clo_code/clo_align_score/clo_align_method).
--
-- Kept as its own migration, separate from V034's DDL, so a slow backfill
-- over a large questions table never blocks/couples with the table
-- creation -- matches this repo's existing "DDL first, data migration
-- second" pattern (V011 then V032).
--
-- Without this, every question authored before EG-GCKT shipped would
-- start with zero Q-matrix rows -- its own avoidable cold-start gap, since
-- the topic_id/clo_code pointer already tells us the one skill it's known
-- to assess.

INSERT INTO assessment.item_skill_mappings
    (question_id, topic_id, clo_code, relevance, generation_method, confirmed_at, version, is_current, created_at)
SELECT
    q.id,
    q.topic_id,
    q.clo_code,
    COALESCE(q.clo_align_score, 1.0),
    COALESCE(q.clo_align_method, 'manual'),
    CASE WHEN q.clo_align_method = 'teacher_confirmed' THEN q.created_at ELSE NULL END,
    1,
    TRUE,
    q.created_at
FROM assessment.questions q
WHERE q.topic_id IS NOT NULL
ON CONFLICT (question_id, topic_id, version) DO NOTHING;
