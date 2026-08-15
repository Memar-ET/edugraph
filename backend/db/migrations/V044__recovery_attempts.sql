-- EG-GCKT checklist section 13: Blocked-Learning & Recovery System --
-- entirely missing from the first implementation pass. Detects repeated
-- failure on the same topic, routes the student to an alternative
-- representation via the typed similar_to/alternative_to/related_to
-- edges Milestone 0 already added (until now, unused by anything), and
-- tracks whether each recovery route actually worked.

CREATE TABLE students.recovery_attempts (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id       UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    school_id        UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    blocked_topic_id UUID NOT NULL REFERENCES curriculum.topics(id) ON DELETE CASCADE,
    route_topic_id   UUID NOT NULL REFERENCES curriculum.topics(id) ON DELETE CASCADE,
    route_edge_type  TEXT NOT NULL CHECK (route_edge_type IN
                        ('similar_to', 'alternative_to', 'related_to', 'lower_granularity')),
    trigger_reason   TEXT NOT NULL,
    status           TEXT NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active', 'ready_to_retarget', 'failed', 'returned_to_target')),
    triggered_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at      TIMESTAMPTZ,
    notes            TEXT
);

CREATE INDEX idx_recovery_attempts_student_blocked ON students.recovery_attempts(student_id, blocked_topic_id);
CREATE INDEX idx_recovery_attempts_active ON students.recovery_attempts(student_id) WHERE status = 'active';

-- Generated locally by the ai-service worker (School Box), same offline
-- sync coverage as the other EG-GCKT students.* tables.
CREATE TRIGGER trg_outbox_recovery_attempts
    AFTER INSERT OR UPDATE OR DELETE ON students.recovery_attempts
    FOR EACH ROW EXECUTE FUNCTION sync.record_outbox_change('students.recovery_attempts');
