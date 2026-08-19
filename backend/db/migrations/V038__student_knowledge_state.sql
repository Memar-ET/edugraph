-- EG-GCKT Milestone 0: Student Knowledge State Graph (spec section 6.4 /
-- the state-field table on page 7) -- a dynamic per-student overlay on
-- curriculum.topics, not a duplicated graph.
--
-- Deliberately does not touch students.mastery_records: that table keeps
-- serving the existing gap-analysis flow (ai-service's
-- fetch_topic_mastery) completely unchanged. skill_states is additive and
-- parallel -- it's what Milestone 4's fusion engine writes to, richer
-- than a single scalar confidence.
--
-- Cold start via row absence, not zero defaults (spec section 15 /
-- Design Principles: "say unknown rather than manufacture confidence").
-- No row is pre-created for a (student, topic) pair combinatorially --
-- the first engine to produce evidence for that pair is what lazily
-- upserts a row, and even then mastery_probability stays NULL until an
-- actual estimate exists; mastery_status defaults to 'unknown' rather
-- than any numeric proxy for "no data yet."

CREATE TABLE students.skill_states (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id            UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    school_id             UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    topic_id              UUID NOT NULL REFERENCES curriculum.topics(id) ON DELETE CASCADE,
    mastery_probability   NUMERIC CHECK (mastery_probability BETWEEN 0 AND 1),
    mastery_status        TEXT NOT NULL DEFAULT 'unknown'
                             CHECK (mastery_status IN ('unknown', 'emerging', 'proficient', 'mastered')),
    uncertainty           NUMERIC CHECK (uncertainty BETWEEN 0 AND 1),
    evidence_count        INT NOT NULL DEFAULT 0,
    evidence_quality      NUMERIC CHECK (evidence_quality BETWEEN 0 AND 1),
    recent_performance    NUMERIC CHECK (recent_performance BETWEEN 0 AND 1),
    trend                 TEXT CHECK (trend IN ('improving', 'stable', 'declining')),
    learning_velocity     NUMERIC,
    forgetting_risk       NUMERIC CHECK (forgetting_risk BETWEEN 0 AND 1),
    last_seen             TIMESTAMPTZ,
    misconception_state   JSONB,
    next_item_success     NUMERIC CHECK (next_item_success BETWEEN 0 AND 1),
    diagnostic_provenance JSONB,
    model_snapshot_id     UUID REFERENCES modeling.model_snapshots(id),
    computed_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    neo4j_written         BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (student_id, topic_id)
);

CREATE INDEX idx_skill_states_student ON students.skill_states(student_id);
CREATE INDEX idx_skill_states_topic ON students.skill_states(topic_id);
CREATE INDEX idx_skill_states_neo4j ON students.skill_states(neo4j_written) WHERE NOT neo4j_written;
