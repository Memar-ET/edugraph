import asyncio
from contextlib import asynccontextmanager

from fastapi import FastAPI

from app.api.v1.routes import career as career_routes
from app.api.v1.routes import tutor as tutor_routes
from app.core.logging import configure_logging
from app.db.neo4j import close_neo4j
from app.db.postgres import close_pool
from app.db.redis import close_redis
from app.workers import answer_key_worker, curriculum_worker, embed_worker, exam_worker, gap_worker, study_plan_worker

logger = configure_logging()

WORKERS = (curriculum_worker, exam_worker, answer_key_worker, gap_worker, study_plan_worker, embed_worker)


@asynccontextmanager
async def lifespan(app: FastAPI):
    # Run the curriculum-, exam-, and answer-key-parsing queue consumers as
    # background tasks alongside the API. In production you'd more likely
    # run each as its own container/process (see each worker module's
    # docstring) so parsing load doesn't compete with request handling, but
    # background tasks keep local dev/docker-compose simple since only one
    # ai-service container is defined.
    tasks = [asyncio.create_task(w.run_forever()) for w in WORKERS]
    logger.info("ai_service.startup", workers=[w.__name__ for w in WORKERS])

    yield

    for w in WORKERS:
        w.request_shutdown()
    for task in tasks:
        task.cancel()
        try:
            await task
        except asyncio.CancelledError:
            pass
    await close_pool()
    await close_redis()
    await close_neo4j()
    logger.info("ai_service.shutdown")


app = FastAPI(lifespan=lifespan)

app.include_router(tutor_routes.router)
app.include_router(career_routes.router)


@app.get("/")
def health():
    return {"status": "ok"}
