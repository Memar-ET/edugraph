import pytest
from unittest.mock import MagicMock, patch
from app.utils.embeddings import (
    EMBEDDING_DIM,
    EmbeddingProvider,
    StubEmbeddingProvider,
    SentenceTransformerEmbeddingProvider,
    get_embedding_provider,
)


@pytest.mark.asyncio
async def test_stub_embedding_provider_dimension():
    provider = StubEmbeddingProvider()
    vec = await provider.embed("test sentence")
    assert len(vec) == EMBEDDING_DIM
    assert provider.model_version == "stub-hash-v1"


@pytest.mark.asyncio
async def test_sentence_transformer_provider_mock():
    mock_model = MagicMock()
    # Mock returning a 1024-element vector
    mock_model.encode.return_value = MagicMock(tolist=lambda: [0.1] * 1024)

    with patch("sentence_transformers.SentenceTransformer", return_value=mock_model):
        provider = SentenceTransformerEmbeddingProvider(model_name="intfloat/multilingual-e5-large")
        vec = await provider.embed("Sample CLO text")
        
        assert len(vec) == 1024
        assert provider.model_version == "intfloat/multilingual-e5-large"
        # Check that 'passage: ' prefix was added to non-prefixed text
        mock_model.encode.assert_called_once_with("passage: Sample CLO text", normalize_embeddings=True)


@pytest.mark.asyncio
async def test_sentence_transformer_provider_fallback_on_error():
    with patch("sentence_transformers.SentenceTransformer", side_effect=RuntimeError("Model load failed")):
        provider = SentenceTransformerEmbeddingProvider(model_name="invalid-model")
        vec = await provider.embed("Sample text")
        # Should gracefully fall back to stub vector of length 1024
        assert len(vec) == 1024


def test_get_embedding_provider_selection(monkeypatch):
    import app.utils.embeddings as emb_module

    # Reset global singleton
    emb_module._provider = None
    monkeypatch.setenv("EMBEDDING_PROVIDER", "sentence_transformers")
    p1 = get_embedding_provider()
    assert isinstance(p1, SentenceTransformerEmbeddingProvider)

    emb_module._provider = None
    monkeypatch.setenv("EMBEDDING_PROVIDER", "stub")
    p2 = get_embedding_provider()
    assert isinstance(p2, StubEmbeddingProvider)
