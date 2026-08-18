"""
Shared LLM provider interface (checklist 8.1/8.2/8.3 -- AI Models: Local
vs. Cloud).

Decision (8.1): build for local by default. The PRD's whole "School Box
operates fully offline" pitch depends on AI inference not requiring
internet, so Ollama is the primary, assumed provider everywhere text
generation happens in this codebase -- gap-analysis explanations
(gap_analysis/llm.py), the AI tutor (tutor/service.py), study-plan day
goals (study_plan/llm.py), and the exam-to-CLO LLM matcher
(exam_parser/clo_matcher_llm.py). Cloud (Gemini) is available behind the
exact same interface, selected by one config value (LLM_PROVIDER), for
deployments that want it -- swapping providers is a config change, not a
rewrite, the same adapter pattern this codebase already uses for storage
(pkg/storage.StorageProvider) and embeddings (app/utils/embeddings.py's
EmbeddingProvider).

This mirrors the local-embedding-generator precedent almost exactly
(8.2 is already done: embeddings.py's FastEmbedProvider/
EmbeddingProvider pair was built before this file existed) -- this is
the same shape applied to text generation instead of embeddings.

Resilience: get_llm_provider() returns the *configured* provider, but
every call site here uses generate_with_fallback(), which also tries
whichever provider ISN'T configured if the primary one is unset/
unreachable. In the default local-first configuration that means an
unreachable Ollama still gets a shot at Gemini if a key happens to be
set (defense in depth, not the primary path); in a cloud-first
configuration, Ollama remains the offline safety net -- the box never
becomes fully mute just because one provider is down.

Amharic: verified directly (not assumed) that the local models actually
tested against this deployment (qwen2.5:7b-instruct, gemma2:9b) produce
garbled, non-functional Amharic. LLMProvider.supports_amharic tells
callers whether to ask for bilingual output or English-only, generically
-- see each feature module for how it's used; this file doesn't build
prompts itself; call sites decide what to write into a prompt, this
module's job is only "which model do I send it to."
"""

from __future__ import annotations

from abc import ABC, abstractmethod
from typing import Optional

import httpx
import structlog

from app.core.config import settings

logger = structlog.get_logger()

GEMINI_MODEL = "gemini-flash-latest"
GEMINI_URL = f"https://generativelanguage.googleapis.com/v1beta/models/{GEMINI_MODEL}:generateContent"
GEMINI_TIMEOUT_SECONDS = 30.0
# Local CPU inference is slow -- a 7B model doing a multi-item JSON
# synthesis can genuinely take tens of seconds on modest School Box
# hardware, so this tier gets a longer budget than the cloud call rather
# than risking false negatives on a box that's just slow, not down.
# Raised from 90s -> 150s after real tutor calls (a much longer,
# context-injected prompt than gap-analysis's shorter synthesis calls)
# measured taking >90s on CPU under realistic concurrent load (embedding
# backfill + several knowledge-tracing workers running at once) --
# pkg/ai/ai.go's client timeout must stay above this + GEMINI_TIMEOUT_SECONDS
# (the worst-case sequential local-then-fallback budget) or Go abandons
# the request while ai-service is still legitimately working.
OLLAMA_TIMEOUT_SECONDS = 150.0


class LLMProvider(ABC):
    model_name: str
    # Whether this provider can be trusted with Amharic output -- False
    # forces callers to request English-only regardless of what a user
    # asked for, rather than show broken text. See module docstring.
    supports_amharic: bool

    @abstractmethod
    async def generate(self, prompt: str, *, json_mode: bool = False) -> Optional[str]:
        """Returns the model's raw text response, or None on any failure
        (unset config, network error, non-2xx) -- callers decide what
        "no response" means for them (try the other provider, fall back
        to deterministic text, or surface an error), this never raises."""


class OllamaProvider(LLMProvider):
    model_name = settings.OLLAMA_MODEL
    supports_amharic = False

    def _url(self) -> str:
        host = settings.OLLAMA_HOST
        if "://" not in host:
            host = "http://" + host
        return host.rstrip("/") + "/api/generate"

    async def generate(self, prompt: str, *, json_mode: bool = False) -> Optional[str]:
        try:
            async with httpx.AsyncClient(timeout=OLLAMA_TIMEOUT_SECONDS) as client:
                payload = {"model": self.model_name, "prompt": prompt, "stream": False}
                if json_mode:
                    payload["format"] = "json"
                resp = await client.post(self._url(), json=payload)
                resp.raise_for_status()
                return resp.json()["response"]
        except Exception:  # noqa: BLE001 -- caller decides what a failed provider means
            logger.exception("llm_provider.ollama_request_failed")
            return None


class GeminiProvider(LLMProvider):
    model_name = GEMINI_MODEL
    supports_amharic = True

    async def generate(self, prompt: str, *, json_mode: bool = False) -> Optional[str]:
        if not settings.GEMINI_API_KEY:
            return None
        try:
            async with httpx.AsyncClient(timeout=GEMINI_TIMEOUT_SECONDS) as client:
                body: dict = {"contents": [{"parts": [{"text": prompt}]}]}
                if json_mode:
                    body["generationConfig"] = {"responseMimeType": "application/json"}
                resp = await client.post(GEMINI_URL, params={"key": settings.GEMINI_API_KEY}, json=body)
                resp.raise_for_status()
                data = resp.json()
                return data["candidates"][0]["content"]["parts"][0]["text"]
        except Exception:  # noqa: BLE001 -- caller decides what a failed provider means
            logger.exception("llm_provider.gemini_request_failed")
            return None


_LOCAL = OllamaProvider()
_CLOUD = GeminiProvider()


def get_llm_provider() -> LLMProvider:
    """The configured primary -- local unless LLM_PROVIDER=cloud."""
    return _CLOUD if settings.LLM_PROVIDER == "cloud" else _LOCAL


def get_fallback_provider() -> LLMProvider:
    """Whichever provider ISN'T primary -- see module docstring on why
    this is still tried rather than failing the moment the primary is
    unavailable."""
    return _LOCAL if settings.LLM_PROVIDER == "cloud" else _CLOUD


async def generate_with_fallback(
    build_prompt, *, json_mode: bool = False
) -> tuple[Optional[str], Optional[LLMProvider]]:
    """build_prompt(provider: LLMProvider) -> str -- a callback rather
    than a plain string because callers need to vary the prompt itself
    by provider.supports_amharic (bilingual vs. English-only), not just
    which model receives it. Returns (response_text, provider_used) --
    the provider object, not just its name, since some callers (the
    tutor) need supports_amharic too, not just model_name for storage.
    (None, None) if both providers failed."""
    primary = get_llm_provider()
    text = await primary.generate(build_prompt(primary), json_mode=json_mode)
    if text is not None:
        return text, primary

    fallback = get_fallback_provider()
    text = await fallback.generate(build_prompt(fallback), json_mode=json_mode)
    if text is not None:
        return text, fallback

    return None, None
