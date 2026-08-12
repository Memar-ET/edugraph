import urllib.parse

from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    # ── POSTGRES ─────────────────────────────
    POSTGRES_HOST: str = "postgres"
    POSTGRES_PORT: int = 5432
    POSTGRES_DB: str = "edugraph"
    POSTGRES_USER: str = "edugraph"
    POSTGRES_PASSWORD: str = "password"
    POSTGRES_SSLMODE: str = "disable"
    POSTGRES_MAX_CONNS: int = 80

    # ── NEO4J ────────────────────────────────
    NEO4J_URI: str = "bolt://neo4j:7687"
    NEO4J_USER: str = "neo4j"
    NEO4J_PASSWORD: str = "12345678"
    NEO4J_MAX_CONN_POOL_SIZE: int = 50

    # ── REDIS ────────────────────────────────
    REDIS_URL: str = "redis:6379"
    REDIS_PASSWORD: str = ""

    # ── APP ──────────────────────────────────
    APP_ENV: str = "development"
    LOG_LEVEL: str = "debug"
    PORT: int = 8080

    # ── AI SERVICE ───────────────────────────
    AI_SERVICE_URL: str = "ai-service:8000"
    OLLAMA_HOST: str = "ollama:11434"
    # Capability 3A offline fallback: tried when Gemini is unset/unreachable,
    # before falling back further to the deterministic English summary (see
    # gap_analysis/llm.py). Matches the model already referenced in
    # school-box/compose/docker-compose.yml's ollama service comment.
    OLLAMA_MODEL: str = "qwen2.5:7b-instruct-q4_K_M"
    EMBEDDING_MODEL: str = "intfloat/multilingual-e5-large"
    # Must match this model's real output width -- see
    # db/migrations/V028__fix_embedding_dimensions.sql. Checked defensively
    # in embeddings-dependent code paths so a future EMBEDDING_MODEL swap
    # to a different-width model fails loudly instead of silently
    # corrupting the pgvector columns.
    EMBEDDING_DIM: int = 1024
    # Capability 5.1 (exam-to-CLO semantic matching) thresholds, taken
    # directly from PRD Section 5.1: >=0.85 cosine similarity auto-aligns a
    # question to a CLO, 0.65-0.84 is flagged for teacher review, <0.65 is
    # treated as no match (falls through to the keyword/LLM matcher).
    CLO_MATCH_AUTO_ALIGN_THRESHOLD: float = 0.85
    CLO_MATCH_NEEDS_REVIEW_THRESHOLD: float = 0.65
    # Capability 2A: Gemini-backed CLO matcher (exam_parser/clo_matcher_llm.py).
    # Empty means "not configured" -- the parser falls back to the plain
    # keyword matcher, never a hard failure.
    GEMINI_API_KEY: str = ""

    @property
    def POSTGRES_DSN(self) -> str:
        """
        asyncpg connection string built from the individual settings above.
        User/password are percent-encoded -- Supabase pooler credentials
        commonly contain characters (@, ?, &) that would otherwise be
        misparsed as URL delimiters. asyncpg 0.29 understands `sslmode` as
        a DSN query param the same way libpq does.
        """
        user = urllib.parse.quote(self.POSTGRES_USER, safe="")
        password = urllib.parse.quote(self.POSTGRES_PASSWORD, safe="")
        return (
            f"postgresql://{user}:{password}"
            f"@{self.POSTGRES_HOST}:{self.POSTGRES_PORT}/{self.POSTGRES_DB}"
            f"?sslmode={self.POSTGRES_SSLMODE}"
        )


# Singleton instance (THIS is what your app imports)
settings = Settings()