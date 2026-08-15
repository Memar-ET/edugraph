-- Fixes a real bug found while implementing checklist section 20
-- (recommendation outcome tracking): action_ranking.classify_action_types
-- (ai-service/app/services/study_plan/action_ranking.py) has assigned
-- 'teacher_escalation' as a candidate action_type since V046, but V042's
-- CHECK constraint on students.recommendation_log.action_type never
-- included that value -- any recommendation classified as a teacher
-- escalation fails INSERT with a constraint violation instead of being
-- logged. Postgres CHECK constraints can't be alter-in-place, so drop and
-- recreate with the missing value added.

ALTER TABLE students.recommendation_log
    DROP CONSTRAINT IF EXISTS recommendation_log_action_type_check;

ALTER TABLE students.recommendation_log
    ADD CONSTRAINT recommendation_log_action_type_check CHECK (action_type IN
        ('practice', 'diagnostic', 'explanation', 'worked_example',
         'alternative_representation', 'prerequisite_review', 'spaced_review',
         'teacher_escalation'));
