-- EG-GCKT Milestone 0: versioning/provenance/governance backbone (spec
-- section 19). Every engine EG-GCKT adds (BKT, DINA/G-DINA, IRT, and
-- later MIRT/DKT if the data ever justifies them) needs its parameters
-- versioned with candidate/validated/active/rejected lifecycle and
-- rollback -- one generic schema for that, rather than one bespoke
-- versioning table per engine.

CREATE SCHEMA IF NOT EXISTS modeling;

-- One row per trained/configured artifact for any engine. status carries
-- the candidate -> validated -> active lifecycle spec section 19 calls
-- for; superseded_by gives rollback (the previously-active snapshot is
-- never deleted, just superseded). scope is free text (e.g. a subject
-- code) so a refit can be scoped per-subject without a schema change;
-- NULL scope = a global default (e.g. the BKT population-prior defaults
-- Milestone 2 ships with before any real refit has run).
CREATE TABLE modeling.model_snapshots (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    model_type        TEXT NOT NULL CHECK (model_type IN
                         ('prerequisite_graph', 'qmatrix', 'irt_calibration', 'mirt_calibration',
                          'dina_parameters', 'gdina_parameters', 'bkt_parameters', 'dkt_model',
                          'fusion_policy', 'recommendation_policy', 'student_state_snapshot')),
    version           INT NOT NULL,
    status            TEXT NOT NULL DEFAULT 'candidate'
                         CHECK (status IN ('candidate', 'validated', 'active', 'rejected', 'superseded')),
    scope             TEXT,
    config            JSONB,
    artifact_uri      TEXT,
    training_summary  JSONB,
    created_by        UUID REFERENCES users(id),
    validated_by      UUID REFERENCES users(id),
    validated_at      TIMESTAMPTZ,
    superseded_by     UUID REFERENCES modeling.model_snapshots(id),
    notes             TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (model_type, scope, version)
);

-- The query every engine actually runs: "give me the currently active
-- parameter set for this model type/scope."
CREATE INDEX idx_model_snapshots_active ON modeling.model_snapshots(model_type, scope) WHERE status = 'active';

-- Append-only. One row per Evidence = {estimate, uncertainty, recency,
-- sample_size, reliability, provenance, context, model_version} object
-- any analytical engine produces (spec section 8.1) -- never updated,
-- only inserted and later windowed/read by the fusion engine
-- (Milestone 4), which stamps consumed_by_fusion_at so it doesn't
-- re-fuse the same evidence twice. provenance is open TEXT ('bkt',
-- 'dina', 'irt', 'graph_reasoning', ...) rather than an enum so a future
-- engine (MIRT/DKT, if the data ever justifies building them) doesn't
-- need a migration to start writing evidence.
CREATE TABLE modeling.evidence_log (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id        UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    topic_id          UUID NOT NULL REFERENCES curriculum.topics(id) ON DELETE CASCADE,
    source_event_id   UUID,
    estimate          NUMERIC,
    uncertainty       NUMERIC,
    recency           TIMESTAMPTZ NOT NULL DEFAULT now(),
    sample_size       INT,
    reliability       NUMERIC CHECK (reliability BETWEEN 0 AND 1),
    provenance        TEXT NOT NULL,
    context           JSONB,
    model_snapshot_id UUID REFERENCES modeling.model_snapshots(id),
    consumed_by_fusion_at TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_evidence_log_student_topic ON modeling.evidence_log(student_id, topic_id, created_at DESC);
CREATE INDEX idx_evidence_log_unconsumed ON modeling.evidence_log(student_id, topic_id) WHERE consumed_by_fusion_at IS NULL;
