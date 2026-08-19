-- Second occurrence of the same bug class fixed in V048: found during a
-- full checklist re-audit. action_ranking.classify_action_types
-- (ai-service/app/services/study_plan/action_ranking.py) assigns
-- 'concept_explanation' for cold-start topics (asserted by
-- test_classify_action_types_cold_start_gets_concept_explanation), but
-- V042's CHECK constraint only ever allowed 'explanation' -- a value
-- nothing in the codebase actually assigns. Every cold-start
-- recommendation (the common case for any new student/skill, evidence
-- count <= 1) would fail INSERT with a constraint violation. Widen the
-- constraint rather than rename the Python value, since
-- 'concept_explanation' is the more descriptive name and matches the
-- checklist's "concept explanations" language; keep 'explanation' too
-- since dropping an unused-but-harmless value has no benefit.

ALTER TABLE students.recommendation_log
    DROP CONSTRAINT IF EXISTS recommendation_log_action_type_check;

ALTER TABLE students.recommendation_log
    ADD CONSTRAINT recommendation_log_action_type_check CHECK (action_type IN
        ('practice', 'diagnostic', 'explanation', 'concept_explanation', 'worked_example',
         'alternative_representation', 'prerequisite_review', 'spaced_review',
         'teacher_escalation'));
