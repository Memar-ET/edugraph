-- V060: Fix a real pre-existing dimension mismatch.
--
-- embeddings.topic_embeddings/clo_embeddings/question_embeddings were
-- created as vector(768) (V011/V025), but Settings.EMBEDDING_MODEL has
-- long been configured to intfloat/multilingual-e5-large, a 1024-dim
-- model (Settings.EMBEDDING_DIM=1024, app/utils/embeddings.py), and
-- SentenceTransformerEmbeddingProvider was already implemented to serve
-- it. Every insert into these three tables from the real provider would
-- fail outright -- pgvector rejects a dimension mismatch. This had never
-- surfaced because get_embedding_provider() defaulted to
-- StubEmbeddingProvider until now (see CLAUDE.md's "CLO/Topic
-- Embeddings" section), and because the Biology curriculum was ingested
-- via the one-off ingest-biology importer, which calls
-- Repository.ApproveAndPromote directly in-process and never pushes
-- queue:embedding:generate (only Service.Approve's HTTP path does) --
-- so these tables had zero rows for the real curriculum, real or stub,
-- until this pass.
--
-- No data migration needed: all existing rows (if any, from earlier
-- stub-embedded demo content) are 768-dim and incompatible with a
-- 1024-dim column regardless, so they're dropped along with the column
-- change rather than coerced.

DROP INDEX IF EXISTS embeddings.idx_topic_emb_hnsw;
DROP INDEX IF EXISTS embeddings.idx_clo_emb_hnsw;
DROP INDEX IF EXISTS embeddings.idx_q_emb_hnsw;

TRUNCATE embeddings.topic_embeddings, embeddings.clo_embeddings, embeddings.question_embeddings;

ALTER TABLE embeddings.topic_embeddings ALTER COLUMN embedding TYPE vector(1024);
ALTER TABLE embeddings.clo_embeddings ALTER COLUMN embedding TYPE vector(1024);
ALTER TABLE embeddings.question_embeddings ALTER COLUMN embedding TYPE vector(1024);

CREATE INDEX idx_topic_emb_hnsw ON embeddings.topic_embeddings USING hnsw (embedding vector_cosine_ops);
CREATE INDEX idx_clo_emb_hnsw ON embeddings.clo_embeddings USING hnsw (embedding vector_cosine_ops);
CREATE INDEX idx_q_emb_hnsw ON embeddings.question_embeddings USING hnsw (embedding vector_cosine_ops);
