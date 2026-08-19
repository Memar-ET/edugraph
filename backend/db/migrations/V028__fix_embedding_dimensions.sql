-- V011 sized embeddings.clo_embeddings/question_embeddings at vector(768)
-- with a comment claiming "multilingual-e5-large outputs 768". That's
-- incorrect: intfloat/multilingual-e5-large (the model
-- ai-service/app/core/config.py's EMBEDDING_MODEL setting already names)
-- outputs 1024-dimensional vectors -- 768 is what the *base* variant
-- (multilingual-e5-base) or bge-base outputs, not -large. Since
-- config.py's already-declared model choice is the source of truth here
-- (Capability 2.1's local embedding pipeline is built against it, see
-- ai-service/app/utils/embeddings.py), this migration corrects the column
-- width to match reality instead of picking a different, smaller model
-- to match the wrong width.
--
-- V025__embeddings_and_versioning.sql (merged from origin/develop after
-- this migration was first written) added embeddings.topic_embeddings
-- with the same vector(768) assumption, for the same reason -- it was
-- built against app/utils/embeddings.py's pre-fastembed stub provider
-- (768-dim placeholder), before this migration's fix and the real
-- fastembed/e5-large model existed on this branch. Fixed here rather
-- than in a separate migration since it's the identical bug.
--
-- All three tables were expected to be empty at this point, but in
-- practice clo_embeddings/topic_embeddings already hold real rows on
-- any environment where the Biology curriculum was ingested (2026-08-10)
-- -- written by the acknowledged-non-semantic StubEmbeddingProvider
-- (768-dim hash placeholder, model_ver='stub-hash-v1'; see CLAUDE.md
-- "CLO/Topic Embeddings"). A plain ALTER COLUMN TYPE vector(1024) fails
-- outright ("expected 1024 dimensions, not 768") since pgvector
-- validates existing row width on the cast -- it can't be resized in
-- place. Since this data is stub/non-semantic (cosine similarity over
-- it was never meaningful), the correct fix is to clear it, not
-- preserve it: TRUNCATE before widening the column, then let
-- embed_worker.py's normal re-embed-on-approve path repopulate real
-- 1024-dim vectors as curriculum jobs get re-approved/re-touched.
-- question_embeddings genuinely is still empty (never written to --
-- see CLAUDE.md Known Gaps), so its TRUNCATE is a no-op safety net,
-- not a behavior change.

DROP INDEX IF EXISTS embeddings.idx_clo_emb_hnsw;
DROP INDEX IF EXISTS embeddings.idx_q_emb_hnsw;
DROP INDEX IF EXISTS embeddings.idx_topic_emb_hnsw;

TRUNCATE TABLE embeddings.clo_embeddings;
TRUNCATE TABLE embeddings.question_embeddings;
TRUNCATE TABLE embeddings.topic_embeddings;

ALTER TABLE embeddings.clo_embeddings
    ALTER COLUMN embedding TYPE vector(1024);

ALTER TABLE embeddings.question_embeddings
    ALTER COLUMN embedding TYPE vector(1024);

ALTER TABLE embeddings.topic_embeddings
    ALTER COLUMN embedding TYPE vector(1024);

CREATE INDEX idx_clo_emb_hnsw ON embeddings.clo_embeddings
    USING hnsw (embedding vector_cosine_ops);
CREATE INDEX idx_q_emb_hnsw ON embeddings.question_embeddings
    USING hnsw (embedding vector_cosine_ops);
CREATE INDEX idx_topic_emb_hnsw ON embeddings.topic_embeddings
    USING hnsw (embedding vector_cosine_ops);
