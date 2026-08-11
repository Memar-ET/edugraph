-- The exam parser now matches questions to CLOs by keyword overlap when a
-- document has no explicit CLO annotation (see ai-service exam_parser) --
-- not true embedding similarity, so 'embedding_auto' would be a
-- mislabeling. 'ai_draft' mirrors the value curriculum.topic_clo_mappings
-- .match_method already uses for the same "AI guessed, not yet confirmed"
-- concept (V011).

ALTER TABLE assessment.questions DROP CONSTRAINT IF EXISTS questions_clo_align_method_check;
ALTER TABLE assessment.questions ADD CONSTRAINT questions_clo_align_method_check
    CHECK (clo_align_method IN ('embedding_auto', 'teacher_confirmed', 'ai_draft'));
