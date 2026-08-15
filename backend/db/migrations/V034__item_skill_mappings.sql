-- EG-GCKT Milestone 0: versioned Q-matrix (spec section 6.3).
--
-- assessment.questions.topic_id/clo_code/clo_align_score/clo_align_method
-- stays untouched -- gap analysis, idx_questions_topic_id, and the exam
-- pipeline all read that as the fast single-skill pointer, and it isn't
-- additive-safe to repurpose. item_skill_mappings is the new multi-skill,
-- versioned, psychometrics-bearing Q-matrix: relevance is the actual
-- Q-matrix entry (item j requires skill k with this relevance), while
-- difficulty/discrimination/qmatrix_confidence start NULL until a
-- calibration pass populates them (Milestone 8, cold-start per spec
-- section 15 -- never fabricated at creation time).
--
-- Versioned like curriculum.subjects (V025): a new version is always a
-- new row, never an in-place mutation, so a Q-matrix history can be
-- replayed later (spec section 19's governance requirement).

CREATE TABLE assessment.item_skill_mappings (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id         UUID NOT NULL REFERENCES assessment.questions(id) ON DELETE CASCADE,
    topic_id            UUID NOT NULL REFERENCES curriculum.topics(id) ON DELETE CASCADE,
    clo_code            TEXT REFERENCES curriculum.clos(code),
    relevance           NUMERIC NOT NULL CHECK (relevance BETWEEN 0 AND 1),
    cognitive_level     TEXT CHECK (cognitive_level IN
                            ('remember', 'understand', 'apply', 'analyze', 'evaluate', 'create')),
    difficulty          NUMERIC,        -- IRT b-parameter, NULL until calibrated (Milestone 8)
    discrimination      NUMERIC,        -- IRT a-parameter, NULL until calibrated
    qmatrix_confidence  NUMERIC CHECK (qmatrix_confidence BETWEEN 0 AND 1),
    generation_method   TEXT NOT NULL CHECK (generation_method IN
                            ('manual', 'ai_draft', 'embedding_auto', 'irt_calibrated', 'teacher_confirmed')),
    authored_by         UUID REFERENCES users(id),
    reviewed_by         UUID REFERENCES users(id),
    confirmed_at        TIMESTAMPTZ,
    version             INT NOT NULL DEFAULT 1,
    is_current          BOOLEAN NOT NULL DEFAULT TRUE,
    superseded_at        TIMESTAMPTZ,
    replaces_mapping_id  UUID REFERENCES assessment.item_skill_mappings(id),
    curriculum_version   TEXT,
    neo4j_written        BOOLEAN NOT NULL DEFAULT FALSE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (question_id, topic_id, version)
);

-- Exactly one current row per (item, skill) -- the invariant a calibration
-- batch job or a manual re-authoring relies on when it inserts a new
-- version and flips the old one's is_current off.
CREATE UNIQUE INDEX idx_item_skill_mappings_current
    ON assessment.item_skill_mappings(question_id, topic_id) WHERE is_current;

CREATE INDEX idx_item_skill_mappings_question ON assessment.item_skill_mappings(question_id) WHERE is_current;
CREATE INDEX idx_item_skill_mappings_topic ON assessment.item_skill_mappings(topic_id) WHERE is_current;
CREATE INDEX idx_item_skill_mappings_neo4j ON assessment.item_skill_mappings(neo4j_written) WHERE NOT neo4j_written;

-- Same append-only review-history shape as
-- curriculum.prerequisite_review_history (V033) -- see that migration's
-- comment for why this is a real FK per-entity-type table rather than a
-- single polymorphic one.
CREATE TABLE assessment.item_skill_mapping_review_history (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mapping_id   UUID NOT NULL REFERENCES assessment.item_skill_mappings(id) ON DELETE CASCADE,
    action       TEXT NOT NULL CHECK (action IN
                    ('created', 'validated', 'relevance_changed', 'calibrated', 'rejected', 'superseded')),
    previous_values JSONB,
    new_values      JSONB,
    reviewed_by     UUID REFERENCES users(id),
    reviewed_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    notes           TEXT
);

CREATE INDEX idx_item_skill_mapping_review_history_mapping ON assessment.item_skill_mapping_review_history(mapping_id);
