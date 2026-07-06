from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    # ── POSTGRES ─────────────────────────────
    POSTGRES_HOST: str = "postgress"
    POSTGRES_PORT: int = 5432
    POSTGRES_DB: str = "edugraph"
    POSTGRES_USER: str = "edugraph"
    POSTGRES_PASSWORD: str = "password"
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
    EMBEDDING_MODEL: str = "intfloat/multilingual-e5-large"


# Singleton instance (THIS is what your app imports)
settings = Settings()