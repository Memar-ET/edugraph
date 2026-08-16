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
import json
import os
import struct
from abc import ABC, abstractmethod

import httpx
import structlog

logger = structlog.get_logger()

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


class OllamaEmbeddingProvider(EmbeddingProvider):
    """768-dim embeddings via Ollama nomic-embed-text.

    Falls back to StubEmbeddingProvider on any connection error so the
    embed pipeline degrades gracefully when Ollama is offline, consistent
    with the codebase's broader "degrade, don't crash" pattern (see
    llm_provider.py's local/cloud fallback chain).
    """

    MODEL = "nomic-embed-text"
    model_version = "ollama-nomic-embed-text-v1"

    def __init__(self) -> None:
        self._base_url = os.getenv("OLLAMA_HOST", "http://ollama:11434").rstrip("/")
        self._fallback = StubEmbeddingProvider()

    async def embed(self, text: str) -> list[float]:
        try:
            async with httpx.AsyncClient(timeout=30.0) as client:
                resp = await client.post(
                    f"{self._base_url}/api/embeddings",
                    content=json.dumps({"model": self.MODEL, "prompt": text}),
                    headers={"Content-Type": "application/json"},
                )
                resp.raise_for_status()
                data = resp.json()
                vec: list[float] = data["embedding"]
                if len(vec) != EMBEDDING_DIM:
                    logger.warning(
                        "ollama_embedding.dim_mismatch",
                        expected=EMBEDDING_DIM,
                        got=len(vec),
                    )
                return vec
        except Exception:
            logger.warning("ollama_embedding.unavailable_falling_back_to_stub")
            return await self._fallback.embed(text)


_provider: EmbeddingProvider | None = None


def get_embedding_provider() -> EmbeddingProvider:
    """Return the configured provider.

    EMBEDDING_PROVIDER env var selects the implementation:
      stub   – deterministic hash placeholder (default, not semantically meaningful)
      ollama – Ollama nomic-embed-text via OLLAMA_HOST; falls back to stub on error
    """
    global _provider
    if _provider is None:
        choice = os.getenv("EMBEDDING_PROVIDER", "stub").lower()
        if choice == "ollama":
            _provider = OllamaEmbeddingProvider()
        else:
            _provider = StubEmbeddingProvider()
    return _provider
