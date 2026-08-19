-- EG-GCKT checklist section 9 (GCSF): "Produce structural status
-- including prerequisite satisfaction, blocked status, and
-- inconsistencies" was never actually stored anywhere -- the fusion
-- engine computed a fused mastery estimate but nothing captured whether
-- the topic's own prerequisites were satisfied or whether the student
-- was currently in recovery mode for it.

ALTER TABLE students.skill_states
    ADD COLUMN structural_status JSONB;

-- Also closes a related governance gap (spec section 18): fusion_policy
-- was never actually versioned as a modeling.model_snapshots row despite
-- the column existing (model_snapshot_id was always written as NULL).
-- No schema change needed for that -- model_snapshot_id already exists;
-- fusion.py just needs to start populating it, which is a code change,
-- not a migration.
