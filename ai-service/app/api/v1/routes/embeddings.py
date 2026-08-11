"""
POST /api/v1/embeddings/passages -- Capability 2.1's local embedding
endpoint, called synchronously by the Go API (pkg/ai/ai.go's
EmbedPassages) during curriculum approval to embed every newly-promoted
CLO's description (see internal/curriculum/service.go's
syncCLOEmbeddings). Exam question embedding, the other half of Capability
2.1, happens entirely on this side (exam_parser/vector_matcher.py) without
an HTTP round trip, since exam parsing already runs in this process --
this endpoint exists only because curriculum approval's Postgres
transaction runs in the Go backend, which has no local model runtime of
its own.

Trust model matches tutor.py: no auth here, the Go API is the only caller
and owns its own role checks (curriculum_officer/ministry_admin) before
ever reaching this service.
"""

from __future__ import annotations

from fastapi import APIRouter, HTTPException
from pydantic import BaseModel, Field

from app.core.config import settings
from app.utils import embeddings

router = APIRouter(prefix="/api/v1/embeddings", tags=["embeddings"])


class EmbedPassagesRequest(BaseModel):
    texts: list[str] = Field(min_length=1, max_length=500)


class EmbedPassagesResponse(BaseModel):
    embeddings: list[list[float]]
    model: str
    dim: int


@router.post("/passages", response_model=EmbedPassagesResponse)
async def embed_passages(req: EmbedPassagesRequest) -> EmbedPassagesResponse:
    if not embeddings.is_available():
        raise HTTPException(status_code=503, detail="local embedding model is unavailable")

    vectors = embeddings.embed_passages(req.texts)
    return EmbedPassagesResponse(embeddings=vectors, model=embeddings.model_name(), dim=settings.EMBEDDING_DIM)
