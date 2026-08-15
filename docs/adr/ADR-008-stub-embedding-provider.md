# ADR-008: StubEmbeddingProvider (Real Interface, Placeholder Math)

Date: 2026-08-10  
Status: Accepted

## Context

The PRD calls for semantic exam-compliance matching (PRD §5.1): given a CLO description, find the exam questions that best cover it using cosine similarity over embeddings. This requires a real embedding model.

Two candidates exist: Gemini embeddings API (cloud, pay-per-use) and a local model via Ollama (self-hosted, zero marginal cost). Neither was chosen yet because:

- The Gemini embeddings API pricing and rate limits for the expected curriculum corpus size had not been evaluated.
- The Ollama model selection for multilingual Ethiopian curriculum text (Amharic + English) had not been validated.
- The infrastructure for running Ollama reliably in production had not been designed.

## Decision

Define a pluggable `EmbeddingProvider` interface in `ai-service/app/utils/embeddings.py` that mirrors the `StorageProvider` adapter pattern from `ADR-001`. The production code calls the interface; the backing implementation is swapped by configuration.

`get_embedding_provider()` always returns `StubEmbeddingProvider` today. This provider:
- Returns a deterministic 768-dim vector derived from hashing the input text.
- Sets `model_ver = "stub-hash-v1"`.
- Writes real rows to `embeddings.clo_embeddings` and `embeddings.topic_embeddings` (HNSW cosine index).

**Cosine similarity between stub vectors is not semantically meaningful.** Code that consumes embeddings must check `model_ver` before trusting similarity scores.

The choice between Gemini embeddings API and a local Ollama model remains open. Both can be implemented by adding one new class that satisfies the interface and updating `get_embedding_provider()`.

## Consequences

**Good:**
- The full embedding pipeline (worker → upsert → HNSW index) runs and is tested without a real model.
- Switching to a real model is a one-function change once the model is chosen.
- No embedding API cost during development.
- The interface is explicit; no code accidentally depends on stub behaviour being "real."

**Bad:**
- `embeddings.clo_embeddings`, `embeddings.topic_embeddings`, and the HNSW indexes consume storage without providing any semantic value yet.
- Semantic exam-compliance matching is not functional until a real model is wired in.
- The `careers.embedding` and `curriculum.clos.embedding` `vector(1024)` columns (V005/V007) use a different dimension from the `embeddings.*` tables (`vector(768)`). This inconsistency must be resolved when the real model is chosen.
