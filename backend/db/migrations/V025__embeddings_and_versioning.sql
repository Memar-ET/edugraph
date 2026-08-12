-- V025__embeddings_and_versioning.sql
--
-- Two additive changes, both scoped to the CLO/topic part of the
-- curriculum domain:
--
-- 1. embeddings.topic_embeddings -- the topic-side counterpart to
--    embeddings.clo_embeddings (V011), which existed but was never
--    written to (see ai-service/app/workers/embed_worker.py).
--
-- 2. curriculum.subjects versioning columns -- today ApproveAndPromote
--    upserts subjects/units/topics/clos keyed on natural keys (code,
--    (unit_id, sequence_order)), so re-approving a revised curriculum
--    under the SAME subject code silently overwrites the content behind
--    existing topic_id/clo_code values that students' gap_records,
--    mastery_records, and exam questions already point to. A mid-year
--    Ministry revision must instead be promoted under a NEW subject code
--    (the normal upload/approve pipeline, unchanged) and then explicitly
--    linked as superseding the old one via these columns -- see
--    Repository.MarkAsRevision / POST /curriculum/subjects/{code}/supersede.
--    The old subject's rows (and everything hanging off them) are never
--    mutated or deleted, so historical references stay accurate forever.

CREATE TABLE embeddings.topic_embeddings (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    topic_id    UUID UNIQUE NOT NULL REFERENCES curriculum.topics(id) ON DELETE CASCADE,
    embedding   vector(768) NOT NULL,
    model_ver   TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_topic_emb_hnsw ON embeddings.topic_embeddings
    USING hnsw (embedding vector_cosine_ops);

ALTER TABLE curriculum.subjects
    ADD COLUMN version               INT NOT NULL DEFAULT 1,
    ADD COLUMN is_current             BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN superseded_at          TIMESTAMPTZ,
    ADD COLUMN previous_version_code  TEXT REFERENCES curriculum.subjects(code);

-- Fast "current curriculum for this subject" lookups (gap analysis, study
-- plans, and any future semantic-match reads should filter on this).
CREATE INDEX idx_subjects_current ON curriculum.subjects(is_current) WHERE is_current;
