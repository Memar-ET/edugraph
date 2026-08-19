-- EG-GCKT checklist sections 2/14/20: nothing tracked whether a
-- recommended action actually improved mastery -- study_plan/service.py
-- generated recommendations, action_ranking.py down-weighted repeats via
-- fetch_repetition_counts, but no outcome was ever measured or fed back.
-- This closes the loop: mastery_probability/evidence_count are captured
-- at recommendation time (mastery_at_recommendation may be NULL --
-- cold-start honesty, same as everywhere else in this schema), and a
-- nightly worker (refit_worker.evaluate_recommendation_outcomes)
-- classifies the outcome once enough time and evidence has passed.

ALTER TABLE students.recommendation_log
    ADD COLUMN mastery_at_recommendation NUMERIC(5, 4),
    ADD COLUMN evidence_count_at_recommendation INT NOT NULL DEFAULT 0,
    ADD COLUMN outcome_status TEXT NOT NULL DEFAULT 'pending' CHECK (outcome_status IN
        ('pending', 'improved', 'unchanged', 'worsened', 'insufficient_evidence')),
    ADD COLUMN mastery_at_evaluation NUMERIC(5, 4),
    ADD COLUMN outcome_evaluated_at TIMESTAMPTZ;

-- Backs the nightly sweep's "find pending rows old enough to evaluate" scan.
CREATE INDEX idx_recommendation_log_pending_outcome
    ON students.recommendation_log (recommended_at)
    WHERE outcome_status = 'pending';
