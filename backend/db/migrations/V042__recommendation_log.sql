-- EG-GCKT Milestone 7: next-best-action repetition penalty (spec section
-- 13's "repetition penalty" ranking factor). Nothing in this codebase
-- previously recorded that a topic had already been recommended to a
-- student in a prior study plan -- each plan generation started from a
-- blank slate, so an unresolved gap could be re-suggested identically,
-- unchanged, forever. One row per (student, topic) per plan generation;
-- the next-best-action ranking (study_plan/service.py) reads how many
-- times a topic has been recommended without the underlying gap
-- resolving, and down-weights repeat offenders in favor of a different
-- approach (e.g. surfacing it as a candidate for recovery-mode/
-- alternative-representation instead of the same practice-question
-- recommendation again).

CREATE TABLE students.recommendation_log (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id    UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    school_id     UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    topic_id      UUID NOT NULL REFERENCES curriculum.topics(id) ON DELETE CASCADE,
    plan_id       UUID REFERENCES students.study_plans(id) ON DELETE SET NULL,
    action_type   TEXT NOT NULL DEFAULT 'practice' CHECK (action_type IN
                     ('practice', 'diagnostic', 'explanation', 'worked_example',
                      'alternative_representation', 'prerequisite_review', 'spaced_review')),
    recommended_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_recommendation_log_student_topic ON students.recommendation_log(student_id, topic_id, recommended_at DESC);

-- Generated locally by the study-plan worker (which can run at a School
-- Box), same offline-sync coverage as students.study_plans.
CREATE TRIGGER trg_outbox_recommendation_log
    AFTER INSERT OR UPDATE OR DELETE ON students.recommendation_log
    FOR EACH ROW EXECUTE FUNCTION sync.record_outbox_change('students.recommendation_log');
