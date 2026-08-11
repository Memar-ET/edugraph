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
-- Both tables are expected to be empty at this point (nothing has ever
-- written to them -- see the PRD/status review that flagged this), so
-- this drops and recreates rather than attempting a data migration. If
-- that assumption is wrong in your environment, back up
-- embeddings.clo_embeddings/question_embeddings before running this.

DROP INDEX IF EXISTS embeddings.idx_clo_emb_hnsw;
DROP INDEX IF EXISTS embeddings.idx_q_emb_hnsw;

ALTER TABLE embeddings.clo_embeddings
    ALTER COLUMN embedding TYPE vector(1024);

ALTER TABLE embeddings.question_embeddings
    ALTER COLUMN embedding TYPE vector(1024);

CREATE INDEX idx_clo_emb_hnsw ON embeddings.clo_embeddings
    USING hnsw (embedding vector_cosine_ops);
CREATE INDEX idx_q_emb_hnsw ON embeddings.question_embeddings
    USING hnsw (embedding vector_cosine_ops);
