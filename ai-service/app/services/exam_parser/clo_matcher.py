"""
Keyword-overlap CLO matcher (Capability 2A).

Used when a document has no explicit [CLO: ...] annotation per question
(see extractor.py's CLO_BRACKET_RE) -- deterministic, dependency-light word
overlap against curriculum.clos.description_en, same "provisional first
draft for human review" philosophy as the rest of this parser rather than
standing up embedding infrastructure for what's still explicitly the
parsing phase (2B/AI Validation is where a real embedding-based pass, using
the already-provisioned embeddings.clo_embeddings/question_embeddings
pgvector tables, belongs).
"""

from __future__ import annotations

import re
from typing import Optional

STOPWORDS = {
    "the", "a", "an", "is", "are", "was", "were", "of", "and", "to", "in", "on", "for",
    "which", "what", "that", "this", "these", "those", "with", "as", "by", "or", "be",
    "it", "its", "from", "at", "into", "how", "why", "when", "not", "but", "than",
    "each", "their", "you", "your", "can", "will", "would", "does", "do", "if",
}

MIN_CONFIDENCE = 0.2


def _normalize(word: str) -> str:
    # Crude singularization ("organelles" -> "organelle", "applications" ->
    # "application") so plural/singular forms in a question vs. a CLO
    # description still count as the same token -- without it, exact-match
    # overlap loses real signal on this kind of mismatch constantly.
    if len(word) > 4 and word.endswith("s") and not word.endswith("ss"):
        return word[:-1]
    return word


def _tokenize(text: str) -> set[str]:
    words = re.findall(r"[a-z]+", text.lower())
    return {_normalize(w) for w in words if w not in STOPWORDS and len(w) > 2}


def match_clo(question_text: str, clos: list[tuple[str, str]]) -> tuple[Optional[str], Optional[float]]:
    """clos: list of (code, description_en). Returns (best_code, score) or
    (None, None) if nothing clears MIN_CONFIDENCE."""
    q_tokens = _tokenize(question_text)
    if not q_tokens:
        return None, None

    best_code: Optional[str] = None
    best_score = 0.0
    for code, description in clos:
        clo_tokens = _tokenize(description)
        if not clo_tokens:
            continue
        overlap = q_tokens & clo_tokens
        score = len(overlap) / min(len(q_tokens), len(clo_tokens))
        if score > best_score:
            best_score, best_code = score, code

    if best_score < MIN_CONFIDENCE:
        return None, None
    return best_code, round(best_score, 3)
