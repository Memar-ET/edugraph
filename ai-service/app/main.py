from contextlib import asynccontextmanager
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from fastapi.middleware.trustedhost import TrustedHostMiddleware

from app.core.config import settings
from app.core.logging import configure_logging
from app.api.v1.routes import exam, gap, plan, career, tutor, policy, health
from app.db.neo4j import get_neo4j_driver
from app.db.redis import get_redis_client
from app.db.postgres import get_pg_pool

configure_logging()


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Startup and shutdown."""
    # Startup
    app.state.neo4j = await get_neo4j_driver()
    app.state.redis = await get_redis_client()
    app.state.pg    = await get_pg_pool()
    yield
    # Shutdown
    await app.state.neo4j.close()
    await app.state.redis.aclose()
    await app.state.pg.close()


app = FastAPI(
    title="EduGraph AI Service",
    version="0.1.0",
    lifespan=lifespan,
    docs_url="/docs" if settings.APP_ENV != "production" else None,
    redoc_url=None,
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=settings.CORS_ORIGINS,
    allow_methods=["POST", "GET"],
    allow_headers=["Authorization", "Content-Type"],
)
app.add_middleware(TrustedHostMiddleware, allowed_hosts=settings.ALLOWED_HOSTS)

# Routers
app.include_router(health.router, prefix="/health", tags=["health"])
app.include_router(exam.router,   prefix="/api/v1/exam",   tags=["exam"])
app.include_router(gap.router,    prefix="/api/v1/gap",    tags=["gap"])
app.include_router(plan.router,   prefix="/api/v1/plan",   tags=["plan"])
app.include_router(career.router, prefix="/api/v1/career", tags=["career"])
app.include_router(tutor.router,  prefix="/api/v1/tutor",  tags=["tutor"])
app.include_router(policy.router, prefix="/api/v1/policy", tags=["policy"])
