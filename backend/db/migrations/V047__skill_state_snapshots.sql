-- EG-GCKT checklist sections 6/18/22: "support learner-state snapshots,"
-- "support historical state comparison," "version learner-state
-- snapshots with source event range," "provide historical state/replay
-- service." students.skill_states is a current-state-only table
-- (upserted in place); this is the append-only historical record next
-- to it -- point-in-time copies, not a live-queryable table, so a
-- historical comparison never changes retroactively.

CREATE TABLE students.skill_state_snapshots (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id           UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    school_id            UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    topic_id             UUID NOT NULL REFERENCES curriculum.topics(id) ON DELETE CASCADE,
    mastery_probability  NUMERIC CHECK (mastery_probability BETWEEN 0 AND 1),
    mastery_status       TEXT NOT NULL,
    uncertainty          NUMERIC CHECK (uncertainty BETWEEN 0 AND 1),
    evidence_count       INT NOT NULL,
    trend                TEXT,
    model_snapshot_id    UUID REFERENCES modeling.model_snapshots(id),
    snapshot_reason      TEXT NOT NULL CHECK (snapshot_reason IN ('nightly', 'manual', 'pre_refit')),
    source_event_range_start TIMESTAMPTZ,
    source_event_range_end   TIMESTAMPTZ,
    taken_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_skill_state_snapshots_lookup ON students.skill_state_snapshots(student_id, topic_id, taken_at DESC);

-- Generated locally by ai-service's nightly refit_worker.py (which can
-- run at a School Box), same offline-sync coverage as the other EG-GCKT
-- students.* tables.
CREATE TRIGGER trg_outbox_skill_state_snapshots
    AFTER INSERT OR UPDATE OR DELETE ON students.skill_state_snapshots
    FOR EACH ROW EXECUTE FUNCTION sync.record_outbox_change('students.skill_state_snapshots');
