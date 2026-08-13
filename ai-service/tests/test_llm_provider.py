"""Tests for app/utils/llm_provider.py's generate_with_fallback -- the
checklist 8.1/8.3 local-first-with-cloud-fallback logic every text-
generation call site (gap analysis, tutor, study plans, exam CLO
matching) depends on. Uses fake LLMProvider doubles rather than mocking
httpx, since what's under test is the fallback *sequencing*, not either
provider's actual HTTP call.
"""

from __future__ import annotations

from typing import Optional

import pytest

from app.utils import llm_provider
from app.utils.llm_provider import LLMProvider, generate_with_fallback


class FakeProvider(LLMProvider):
    """A provider double whose generate() is scripted per test rather
    than making a real request."""

    def __init__(self, name: str, supports_amharic: bool, response: Optional[str]):
        self.model_name = name
        self.supports_amharic = supports_amharic
        self._response = response
        self.calls: list[tuple[str, bool]] = []

    async def generate(self, prompt: str, *, json_mode: bool = False) -> Optional[str]:
        self.calls.append((prompt, json_mode))
        return self._response


def _patch_providers(monkeypatch: pytest.MonkeyPatch, primary: LLMProvider, fallback: LLMProvider) -> None:
    monkeypatch.setattr(llm_provider, "get_llm_provider", lambda: primary)
    monkeypatch.setattr(llm_provider, "get_fallback_provider", lambda: fallback)


@pytest.mark.asyncio
async def test_uses_primary_when_it_succeeds(monkeypatch: pytest.MonkeyPatch) -> None:
    primary = FakeProvider("primary", supports_amharic=False, response="primary said hi")
    fallback = FakeProvider("fallback", supports_amharic=True, response="fallback said hi")
    _patch_providers(monkeypatch, primary, fallback)

    text, provider = await generate_with_fallback(lambda p: f"prompt for {p.model_name}")

    assert text == "primary said hi"
    assert provider is primary
    assert len(primary.calls) == 1
    assert len(fallback.calls) == 0  # fallback must never be tried when primary succeeds


@pytest.mark.asyncio
async def test_falls_back_when_primary_returns_none(monkeypatch: pytest.MonkeyPatch) -> None:
    # None simulates every failure mode generate() collapses to: unset
    # config, network error, non-2xx -- see LLMProvider.generate's own
    # docstring on why this never raises.
    primary = FakeProvider("primary", supports_amharic=False, response=None)
    fallback = FakeProvider("fallback", supports_amharic=True, response="fallback saved the day")
    _patch_providers(monkeypatch, primary, fallback)

    text, provider = await generate_with_fallback(lambda p: f"prompt for {p.model_name}")

    assert text == "fallback saved the day"
    assert provider is fallback
    assert len(primary.calls) == 1
    assert len(fallback.calls) == 1


@pytest.mark.asyncio
async def test_returns_none_none_when_both_fail(monkeypatch: pytest.MonkeyPatch) -> None:
    primary = FakeProvider("primary", supports_amharic=False, response=None)
    fallback = FakeProvider("fallback", supports_amharic=True, response=None)
    _patch_providers(monkeypatch, primary, fallback)

    text, provider = await generate_with_fallback(lambda p: "prompt")

    assert text is None
    assert provider is None


@pytest.mark.asyncio
async def test_build_prompt_receives_the_provider_actually_being_called(monkeypatch: pytest.MonkeyPatch) -> None:
    # The whole reason build_prompt is a callback and not a plain string
    # (see the module's own docstring): callers vary the prompt by
    # provider.supports_amharic. This asserts the callback is invoked
    # once per provider actually tried, with THAT provider -- not the
    # primary's prompt reused for the fallback call.
    primary = FakeProvider("primary", supports_amharic=False, response=None)
    fallback = FakeProvider("fallback", supports_amharic=True, response="ok")
    _patch_providers(monkeypatch, primary, fallback)

    seen_providers: list[LLMProvider] = []

    def build_prompt(p: LLMProvider) -> str:
        seen_providers.append(p)
        return "bilingual prompt" if p.supports_amharic else "english-only prompt"

    await generate_with_fallback(build_prompt)

    assert seen_providers == [primary, fallback]
    assert primary.calls[0][0] == "english-only prompt"
    assert fallback.calls[0][0] == "bilingual prompt"


@pytest.mark.asyncio
async def test_json_mode_is_forwarded_to_the_provider(monkeypatch: pytest.MonkeyPatch) -> None:
    primary = FakeProvider("primary", supports_amharic=False, response="{}")
    fallback = FakeProvider("fallback", supports_amharic=True, response=None)
    _patch_providers(monkeypatch, primary, fallback)

    await generate_with_fallback(lambda p: "prompt", json_mode=True)

    assert primary.calls[0][1] is True
