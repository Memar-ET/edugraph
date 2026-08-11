"""
Local (offline-capable) text embedding generation -- Capability 2.1.

Uses fastembed, an ONNX-runtime-based library that runs entirely on CPU
with no torch/GPU dependency and no cloud API call, matching the PRD's
"School Box operates fully offline" requirement -- unlike the rest of this
codebase's AI features (gap analysis, study plans, tutor, LLM-assisted CLO
matching in clo_matcher_llm.py), which all call the Gemini cloud API and
therefore stop working offline. This module is deliberately the one
AI-dependent path in the codebase that keeps working with zero internet
access, once the model weights are cached locally (see _MODEL_CACHE_DIR
below).

Model: settings.EMBEDDING_MODEL (default "intfloat/multilingual-e5-large",
1024-dim -- see db/migrations/V025__fix_embedding_dimension.sql for why
1024 and not the 768 originally in V011). Chosen for being genuinely
multilingual (curriculum content and student answers may be in English or
Amharic) rather than English-only, which ruled out fastembed's BAAI/bge-*
models for this use case.

e5-family models expect a "query: " or "passage: " prefix on every input
string -- this is a real accuracy requirement of the model, not a style
choice (see the model card). We treat CLO descriptions as the corpus being
searched ("passage") and exam question stems as what's searching it
("query"), which is the standard asymmetric-retrieval framing and the one
these models were tuned for.
"""

from __future__ import annotations

import time
from functools import lru_cache
from typing import Optional

import structlog

from app.core.config import settings

logger = structlog.get_logger()

_QUERY_PREFIX = "query: "
_PASSAGE_PREFIX = "passage: "


@lru_cache(maxsize=1)
def _get_model():
    """
    Lazily loads (and caches for the life of the process) the embedding
    model. Import is deferred to inside this function rather than at
    module load time so importing this module doesn't itself trigger a
    model download/load in contexts that never call embed_* (e.g. Alembic
    migrations, one-off scripts) -- the same lazy-singleton pattern
    app/db/neo4j.py and app/db/redis.py use for their clients.
    """
    from fastembed import TextEmbedding

    t0 = time.monotonic()
    model = TextEmbedding(model_name=settings.EMBEDDING_MODEL)
    logger.info(
        "embeddings.model_loaded",
        model=settings.EMBEDDING_MODEL,
        seconds=round(time.monotonic() - t0, 2),
    )
    return model


def embed_passages(texts: list[str]) -> list[list[float]]:
    """Embed a batch of "documents to be searched" -- CLO descriptions in
    this codebase. Order-preserving: result[i] corresponds to texts[i]."""
    if not texts:
        return []
    model = _get_model()
    prefixed = [_PASSAGE_PREFIX + t for t in texts]
    return [vec.tolist() for vec in model.embed(prefixed)]


def embed_query(text: str) -> list[float]:
    """Embed a single "search query" -- an exam question stem being
    matched against already-embedded CLOs."""
    model = _get_model()
    prefixed = [_QUERY_PREFIX + text]
    return next(iter(model.embed(prefixed))).tolist()


def model_name() -> str:
    """The currently configured model identifier, stored alongside every
    embedding as embeddings.*.model_ver so a future model change is
    traceable per row instead of assumed."""
    return settings.EMBEDDING_MODEL


def is_available() -> bool:
    """
    Best-effort check for whether local embeddings can actually be used
    right now, without raising -- callers (vector_matcher.py) use this to
    fall back to the keyword/LLM matcher rather than let a missing/corrupt
    model cache take down exam parsing entirely.
    """
    try:
        _get_model()
        return True
    except Exception:  # noqa: BLE001 -- any load failure means "unavailable"
        logger.exception("embeddings.model_unavailable")
        return False
