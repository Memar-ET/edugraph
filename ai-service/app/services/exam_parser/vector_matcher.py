"""
Semantic (vector-embedding-based) CLO matcher -- Capability 2.1 / PRD
Section 5.1.

This is the matcher the PRD originally specified (local embedding model +
pgvector cosine similarity at fixed 0.85/0.65 thresholds) -- previously
unimplemented; clo_matcher.py (keyword overlap) and clo_matcher_llm.py
(Gemini) were the only two matchers actually wired into
exam_parser/service.py, and embeddings.clo_embeddings/question_embeddings
were provisioned in the schema but never populated (see
db/migrations/V018__exam_clo_align_method.sql's comment explicitly calling
'embedding_auto' unused/"mislabeling" at the time).

Match priority in exam_parser/service.py is: this vector matcher first
(real semantic similarity, works offline once the model is cached, no
per-question cloud API call) -> clo_matcher_llm.py (Gemini, needs
GEMINI_API_KEY + internet) -> clo_matcher.py (keyword overlap, the
last-resort fallback that always works). This matcher itself falls back
silently (returns None) rather than raising when: the embedding model
failed to load, or the target subject has no embedded CLOs yet (curriculum
approved before Capability 2.1, or its embedding call failed -- see
curriculum/service.go's syncCLOEmbeddings).
"""

from __future__ import annotations

from typing import Optional

import structlog

from app.core.config import settings
from app.db import postgres_embeddings as db
from app.utils import embeddings

logger = structlog.get_logger()


async def match_clo_vector(question_text: str, subject_code: str) -> tuple[Optional[str], Optional[float], str]:
    """
    Returns (clo_code, score, classification) where classification is one
    of "auto_align" (score >= CLO_MATCH_AUTO_ALIGN_THRESHOLD, i.e. the PRD's
    0.85), "needs_review" (between the two thresholds), or "no_match"
    (below CLO_MATCH_NEEDS_REVIEW_THRESHOLD / no embeddings available for
    this subject / model unavailable). Only "auto_align" should be treated
    as an actual match by the caller -- "needs_review" and "no_match" both
    mean "let the next matcher in the chain try", the distinction is only
    for logging/future UI surfacing of near-miss matches.
    """
    if not embeddings.is_available():
        return None, None, "no_match"

    if not await db.has_embeddings_for_subject(subject_code):
        return None, None, "no_match"

    query_embedding = embeddings.embed_query(question_text)
    best = await db.find_best_matching_clo(subject_code, query_embedding)
    if best is None:
        return None, None, "no_match"

    score = float(best["score"])
    if score >= settings.CLO_MATCH_AUTO_ALIGN_THRESHOLD:
        return best["code"], score, "auto_align"
    if score >= settings.CLO_MATCH_NEEDS_REVIEW_THRESHOLD:
        logger.info("vector_matcher.needs_review", clo_code=best["code"], score=round(score, 4))
        return None, score, "needs_review"
    return None, score, "no_match"
