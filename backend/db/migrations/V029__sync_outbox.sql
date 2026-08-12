-- School Box offline sync -- transactional outbox (checklist 9.1/9.4).
--
-- The School Box's sync-agent (school-box/sync-agent) needs a way to know
-- "this row changed locally, it needs to reach the cloud" without polling
-- every domain table for an updated_at column most of them don't have.
-- Standard transactional-outbox pattern: a single trigger function,
-- attached to every table a School Box writes to locally, appends a
-- to_jsonb() snapshot of the row to sync.outbox in the same transaction
-- as the actual write, so a crash between the domain write and the
-- outbox write is impossible.
--
-- This migration runs against both Central Cloud and School Box Postgres
-- (one shared migration set, per CLAUDE.md) -- on Central Cloud the
-- triggers still fire and sync.outbox still fills up, but nothing reads
-- it there; only a School Box runs the sync-agent that drains it. That's
-- an accepted simplification for this MVP, not a bug.
--
-- Table scope: the tables actually written locally at a school --
-- assessment.exams/questions (teacher uploads/corrections),
-- assessment.exam_attempts/student_answers (exam-taking), and the
-- students.* tables the AI service's local workers write during gap
-- analysis / study planning. Curriculum data flows the other direction
-- (ministry -> schools) and isn't in scope here.

CREATE SCHEMA IF NOT EXISTS sync;

CREATE TABLE sync.outbox (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    entity_type  TEXT NOT NULL,
    entity_id    UUID NOT NULL,
    school_id    UUID,
    operation    TEXT NOT NULL CHECK (operation IN ('create', 'update', 'delete')),
    payload      JSONB NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    pushed_at    TIMESTAMPTZ
);

-- The sync-agent's main query: oldest unpushed rows first.
CREATE INDEX idx_sync_outbox_unpushed ON sync.outbox (created_at) WHERE pushed_at IS NULL;
CREATE INDEX idx_sync_outbox_entity ON sync.outbox (entity_type, entity_id);

-- One row per device (a School Box has exactly one sync-agent, but this
-- stays keyed by device_id rather than a singleton so a device rebuild
-- doesn't need a migration to re-seed it).
CREATE TABLE sync.device_state (
    device_id      TEXT PRIMARY KEY,
    last_pushed_at TIMESTAMPTZ,
    last_pulled_at TIMESTAMPTZ,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- What entity_id + synced_at a device has actually applied, per pulled
-- entity -- lets the agent tell "stale duplicate pull" apart from "real
-- update" without re-deriving it from the domain tables. Only used for
-- LWW-mode entities (see sync-agent's crdt.go); append-only entities
-- (exam_attempts, student_answers) rely on the primary key instead.
CREATE TABLE sync.applied_versions (
    entity_type TEXT NOT NULL,
    entity_id   UUID NOT NULL,
    applied_at  TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (entity_type, entity_id)
);

-- Logged, not blocking: a local edit was still pending in the outbox
-- when a remote change for the same entity arrived. The agent applies
-- the remote change anyway (last-write-wins, matching the platform's
-- original CRDT design intent) and logs it here for human review rather
-- than silently dropping either side.
CREATE TABLE sync.conflicts (
    id               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    entity_type      TEXT NOT NULL,
    entity_id        UUID NOT NULL,
    reason           TEXT NOT NULL,
    incoming_payload JSONB NOT NULL,
    local_payload    JSONB,
    detected_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved         BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE OR REPLACE FUNCTION sync.record_outbox_change() RETURNS TRIGGER AS $$
DECLARE
    v_op      TEXT;
    v_row     RECORD;
    v_payload JSONB;
BEGIN
    IF TG_OP = 'DELETE' THEN
        v_op := 'delete';
        v_row := OLD;
    ELSIF TG_OP = 'UPDATE' THEN
        v_op := 'update';
        v_row := NEW;
    ELSE
        v_op := 'create';
        v_row := NEW;
    END IF;

    v_payload := to_jsonb(v_row);

    INSERT INTO sync.outbox (entity_type, entity_id, school_id, operation, payload)
    VALUES (TG_ARGV[0], v_row.id, (v_payload ->> 'school_id')::UUID, v_op, v_payload);

    RETURN NULL; -- AFTER trigger; return value is ignored either way.
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_outbox_exams
    AFTER INSERT OR UPDATE OR DELETE ON assessment.exams
    FOR EACH ROW EXECUTE FUNCTION sync.record_outbox_change('assessment.exams');

CREATE TRIGGER trg_outbox_questions
    AFTER INSERT OR UPDATE OR DELETE ON assessment.questions
    FOR EACH ROW EXECUTE FUNCTION sync.record_outbox_change('assessment.questions');

CREATE TRIGGER trg_outbox_exam_attempts
    AFTER INSERT OR UPDATE OR DELETE ON assessment.exam_attempts
    FOR EACH ROW EXECUTE FUNCTION sync.record_outbox_change('assessment.exam_attempts');

CREATE TRIGGER trg_outbox_student_answers
    AFTER INSERT OR UPDATE OR DELETE ON assessment.student_answers
    FOR EACH ROW EXECUTE FUNCTION sync.record_outbox_change('assessment.student_answers');

CREATE TRIGGER trg_outbox_gap_records
    AFTER INSERT OR UPDATE OR DELETE ON students.gap_records
    FOR EACH ROW EXECUTE FUNCTION sync.record_outbox_change('students.gap_records');

CREATE TRIGGER trg_outbox_mastery_records
    AFTER INSERT OR UPDATE OR DELETE ON students.mastery_records
    FOR EACH ROW EXECUTE FUNCTION sync.record_outbox_change('students.mastery_records');

CREATE TRIGGER trg_outbox_study_plans
    AFTER INSERT OR UPDATE OR DELETE ON students.study_plans
    FOR EACH ROW EXECUTE FUNCTION sync.record_outbox_change('students.study_plans');

CREATE TRIGGER trg_outbox_exam_insights
    AFTER INSERT OR UPDATE OR DELETE ON students.exam_insights
    FOR EACH ROW EXECUTE FUNCTION sync.record_outbox_change('students.exam_insights');
