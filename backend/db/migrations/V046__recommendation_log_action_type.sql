-- EG-GCKT checklist section 14: next-best-action candidate action types
-- (practice/diagnostic/explanation/worked_example/alternative_
-- representation/prerequisite_review/spaced_review/teacher_escalation)
-- were defined as a CHECK constraint on students.recommendation_log
-- (V042) but nothing ever wrote anything other than the 'practice'
-- default -- the ranking engine never actually differentiated action
-- types. This also adds model_snapshot_id so a recommendation_policy
-- version (spec section 18: "version recommendation policy") can be
-- traced from a specific recommendation, mirroring how evidence_log
-- already references the snapshot that produced it.

ALTER TABLE students.recommendation_log
    ADD COLUMN model_snapshot_id UUID REFERENCES modeling.model_snapshots(id);
