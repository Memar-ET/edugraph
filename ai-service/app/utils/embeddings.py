"""
Pluggable embedding provider for the CLO/topic embedding pipeline
(app/workers/embed_worker.py).

EmbeddingProvider mirrors the StorageProvider adapter pattern used for
Postgres/S3 dev storage (backend/pkg/storage/interface.go): every caller depends
only on this interface, so swapping embedding models is a one-file change here.

Supports:
- SentenceTransformerEmbeddingProvider (default, intfloat/multilingual-e5-large, 1024-dim)
- OllamaEmbeddingProvider (Ollama nomic-embed-text)
- StubEmbeddingProvider (deterministic hash placeholder for dev/testing)
"""

from __future__ import annotations

import hashlib
import json
import os
import struct
from abc import ABC, abstractmethod

import httpx
import structlog

from app.core.config import settings

logger = structlog.get_logger()

EMBEDDING_DIM = getattr(settings, "EMBEDDING_DIM", 1024)


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


class SentenceTransformerEmbeddingProvider(EmbeddingProvider):
    """1024-dim embeddings via local Hugging Face SentenceTransformer (intfloat/multilingual-e5-large).

    E5 models require prefixing 'passage: ' for documents or 'query: ' for queries.
    Falls back to StubEmbeddingProvider on error to ensure system stability.
    """

    def __init__(self, model_name: str | None = None) -> None:
        self.model_name = model_name or getattr(settings, "EMBEDDING_MODEL", "intfloat/multilingual-e5-large")
        self.model_version = self.model_name
        self._model = None
        self._fallback = StubEmbeddingProvider()

    def _load_model(self):
        if self._model is None:
            from sentence_transformers import SentenceTransformer
            logger.info("sentence_transformer.loading", model_name=self.model_name)
            self._model = SentenceTransformer(self.model_name)
        return self._model

    async def embed(self, text: str) -> list[float]:
        try:
            model = self._load_model()
            formatted_text = text if text.startswith(("passage:", "query:")) else f"passage: {text}"
            vec = model.encode(formatted_text, normalize_embeddings=True).tolist()
            if len(vec) != EMBEDDING_DIM:
                logger.warning(
                    "sentence_transformer.dim_mismatch",
                    expected=EMBEDDING_DIM,
                    got=len(vec),
                )
            return vec
        except Exception:
            logger.exception("sentence_transformer.failed_falling_back_to_stub")
            return await self._fallback.embed(text)


class OllamaEmbeddingProvider(EmbeddingProvider):
    """Embeddings via Ollama nomic-embed-text.

    Falls back to StubEmbeddingProvider on any connection error so the
    embed pipeline degrades gracefully when Ollama is offline.
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
      sentence_transformers / local / e5 – Hugging Face SentenceTransformer (default: intfloat/multilingual-e5-large)
      ollama                              – Ollama nomic-embed-text via OLLAMA_HOST
      stub                                – deterministic hash placeholder
    """
    global _provider
    if _provider is None:
        choice = os.getenv("EMBEDDING_PROVIDER", "sentence_transformers").lower()
        if choice in ("sentence_transformers", "local", "e5", "huggingface"):
            _provider = SentenceTransformerEmbeddingProvider()
        elif choice == "ollama":
            _provider = OllamaEmbeddingProvider()
        else:
            _provider = StubEmbeddingProvider()
    return _provider


def is_available() -> bool:
    """Return True if an embedding provider is configured and available."""
    try:
        p = get_embedding_provider()
        return p is not None
    except Exception:
        return False


def embed_query(text: str) -> list[float]:
    """Synchronously embed query text for vector search (e.g. in vector_matcher)."""
    p = get_embedding_provider()
    if isinstance(p, SentenceTransformerEmbeddingProvider):
        try:
            m = p._load_model()
            formatted = text if text.startswith(("passage:", "query:")) else f"query: {text}"
            return m.encode(formatted, normalize_embeddings=True).tolist()
        except Exception:
            logger.exception("embed_query.sentence_transformer_failed")
    
    # Fallback if async event loop or stub
    import asyncio
    try:
        loop = asyncio.get_event_loop()
        if loop.is_running():
            # Generate deterministic stub vector if loop is running synchronously
            seed = text.encode("utf-8")
            vec = []
            for i in range(EMBEDDING_DIM):
                digest = hashlib.sha256(seed + i.to_bytes(4, "big")).digest()
                n = struct.unpack(">Q", digest[:8])[0]
                vec.append((n / 2**64) * 2 - 1)
            norm = sum(x * x for x in vec) ** 0.5 or 1.0
            return [x / norm for x in vec]
        return loop.run_until_complete(p.embed(text))
    except Exception:
        seed = text.encode("utf-8")
        vec = []
        for i in range(EMBEDDING_DIM):
            digest = hashlib.sha256(seed + i.to_bytes(4, "big")).digest()
            n = struct.unpack(">Q", digest[:8])[0]
            vec.append((n / 2**64) * 2 - 1)
        norm = sum(x * x for x in vec) ** 0.5 or 1.0
        return [x / norm for x in vec]


def model_name() -> str:
    """Return the name/version of the current embedding provider model."""
    p = get_embedding_provider()
    return getattr(p, "model_version", "unknown")


