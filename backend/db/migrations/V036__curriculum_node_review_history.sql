-- EG-GCKT Milestone 0: curriculum node review history (spec section 6.1's
-- "required graph controls": node provenance, validation status,
-- ownership/review responsibility).
--
-- curriculum.topics/clos/subjects only have a single updated_by/updated_at
-- "last editor" pair (V031) -- no history of prior edits. This table is
-- the closest thing to a generic append-only history for curriculum
-- nodes, but deliberately scoped to curriculum nodes only (not a
-- cross-schema polymorphic table): node_id is TEXT rather than UUID
-- specifically because curriculum.clos.code is a TEXT primary key, not a
-- UUID like topics/units/subjects -- a single UUID entity_id column
-- literally could not FK to it. Because node_id spans two different key
-- types, this table cannot carry a real FK the way
-- prerequisite_review_history/item_skill_mapping_review_history do; that
-- is the deliberate, documented tradeoff, not an oversight.

CREATE TABLE curriculum.node_review_history (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_type       TEXT NOT NULL CHECK (node_type IN ('subject', 'unit', 'topic', 'clo')),
    node_id         TEXT NOT NULL,
    action          TEXT NOT NULL CHECK (action IN ('created', 'updated', 'validated', 'retired', 'superseded')),
    previous_values JSONB,
    new_values      JSONB,
    reviewed_by     UUID REFERENCES users(id),
    reviewed_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    notes           TEXT
);

CREATE INDEX idx_node_review_history_node ON curriculum.node_review_history(node_type, node_id);
