-- Adds 'llm_verified' for CLO matches made by the Gemini-backed matcher
-- (ai-service exam_parser/clo_matcher_llm.py) -- distinguishes a real LLM
-- judgment call from the plain keyword-overlap heuristic ('ai_draft').

ALTER TABLE assessment.questions DROP CONSTRAINT IF EXISTS questions_clo_align_method_check;
ALTER TABLE assessment.questions ADD CONSTRAINT questions_clo_align_method_check
    CHECK (clo_align_method IN ('embedding_auto', 'teacher_confirmed', 'ai_draft', 'llm_verified'));
