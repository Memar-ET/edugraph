-- EG-GCKT Milestone 0: typed prerequisite edges (spec section 6.2).
--
-- curriculum.topic_prerequisites today has exactly one implicit edge
-- meaning ("topic requires prerequisite"). The EG-GCKT spec calls for
-- seven distinct edge types (requires / strongly_requires /
-- recommended_before / related_to / similar_to / supports /
-- alternative_to), each carrying confidence and evidence separate from
-- the existing "weight" column (weight = how essential the dependency
-- is; confidence = how sure we are the edge is correct at all -- an
-- ai_inferred edge can be weight=0.9, confidence=0.3).
--
-- Extending the existing table in place rather than forking a parallel
-- one: the cycle-detection CTE, Neo4j sync (SyncPrerequisiteToNeo4j),
-- and ListUnsyncedPrerequisites resync machinery all already work
-- against this table's shape, and CLAUDE.md's migration rule is
-- additive-only anyway.
--
-- Existing rows backfill edge_type='requires' -- semantically correct,
-- since every row inserted before this migration *is* a requires edge
-- (AddTopicPrerequisite has never written anything else).

ALTER TABLE curriculum.topic_prerequisites
    ADD COLUMN edge_type TEXT NOT NULL DEFAULT 'requires'
        CHECK (edge_type IN ('requires', 'strongly_requires', 'recommended_before',
                              'related_to', 'similar_to', 'supports', 'alternative_to')),
    ADD COLUMN confidence NUMERIC CHECK (confidence BETWEEN 0 AND 1),
    ADD COLUMN evidence TEXT,
    ADD COLUMN created_by UUID REFERENCES users(id),
    ADD COLUMN created_by_model TEXT,
    ADD COLUMN curriculum_version TEXT;

-- Widen the uniqueness constraint so e.g. "A requires B" and
-- "A similar_to B" can coexist without allowing a duplicate
-- "A requires B" row. Constraint replacement, not a column drop --
-- additive under CLAUDE.md's migration rule.
ALTER TABLE curriculum.topic_prerequisites
    DROP CONSTRAINT topic_prerequisites_topic_id_prerequisite_id_key,
    ADD CONSTRAINT topic_prerequisites_topic_id_prerequisite_id_edge_type_key
        UNIQUE (topic_id, prerequisite_id, edge_type);

-- Append-only review history, mirroring the one real audit-trail
-- precedent in this codebase (assessment.answer_grade_history, V031)
-- rather than the single-last-reviewer field topic_prerequisites had
-- until now. A real FK (topic_prerequisites.id is already a UUID
-- surrogate key), not a polymorphic entity_id -- see V036's
-- curriculum.node_review_history for why curriculum nodes (topics/clos)
-- can't use this same shape (clos.code is a TEXT primary key, not UUID).
CREATE TABLE curriculum.prerequisite_review_history (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prerequisite_edge_id UUID NOT NULL REFERENCES curriculum.topic_prerequisites(id) ON DELETE CASCADE,
    action              TEXT NOT NULL CHECK (action IN
                            ('created', 'validated', 'edge_type_changed', 'confidence_changed',
                             'rejected', 'superseded')),
    previous_values     JSONB,
    new_values          JSONB,
    reviewed_by         UUID REFERENCES users(id),
    reviewed_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    notes               TEXT
);

CREATE INDEX idx_prereq_review_history_edge ON curriculum.prerequisite_review_history(prerequisite_edge_id);

-- Force a resync: every already-synced Neo4j :HAS_PREREQUISITE
-- relationship predates the edgeType/confidence properties
-- SyncPrerequisiteToNeo4j now writes. class_heatmap.go's cross-grade walk
-- defensively treats a missing rel.edgeType as 'requires' either way, but
-- there's no reason to leave the graph mirror stale when the existing
-- resync endpoint (POST /curriculum/prerequisites/resync) can catch every
-- row up in one call -- the Biology dataset is only ~90 edges.
UPDATE curriculum.topic_prerequisites SET neo4j_written = false;
