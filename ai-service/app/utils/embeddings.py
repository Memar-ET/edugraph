"""
Pluggable embedding provider for the CLO/topic embedding pipeline
(app/workers/embed_worker.py).

No real embedding model is wired up anywhere in this codebase yet -- that
decision (Gemini embeddings API vs. a local model via Ollama vs. something
else) is deliberately deferred. EmbeddingProvider mirrors the
StorageProvider adapter pattern used for Postgres/S3 dev storage
(backend/pkg/storage/interface.go): every caller depends only on this
interface, so wiring in a real model later is a one-file change here, not
a rewrite of the worker or the DB layer.

get_embedding_provider() currently always returns StubEmbeddingProvider,
a deterministic hash-based placeholder. It is NOT semantically meaningful
(cosine similarity between two stub vectors says nothing about whether the
underlying text is related) -- it only guarantees a stable 768-dim vector
per input, matching the embeddings.*_embeddings vector(768) columns, so
the write path (upsert) and read path (HNSW cosine search) can be built,
exercised, and tested end-to-end before a real model is chosen.
"""

from __future__ import annotations

import hashlib
import struct
from abc import ABC, abstractmethod

EMBEDDING_DIM = 768


class EmbeddingProvider(ABC):
    model_version: str

    @abstractmethod
    async def embed(self, text: str) -> list[float]:
        """Return an EMBEDDING_DIM-length vector for `text`."""


class StubEmbeddingProvider(EmbeddingProvider):
    model_version = "stub-hash-v1"

    async def embed(self, text: str) -> list[float]:
        seed = text.encode("utf-8")
        vec: list[float] = []
        for i in range(EMBEDDING_DIM):
            digest = hashlib.sha256(seed + i.to_bytes(4, "big")).digest()
            n = struct.unpack(">Q", digest[:8])[0]
            vec.append((n / 2**64) * 2 - 1)  # -> [-1, 1]
        norm = sum(x * x for x in vec) ** 0.5 or 1.0
        return [x / norm for x in vec]


_provider: EmbeddingProvider | None = None


def get_embedding_provider() -> EmbeddingProvider:
    global _provider
    if _provider is None:
        _provider = StubEmbeddingProvider()
    return _provider
