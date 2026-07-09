import asyncio
from contextlib import asynccontextmanager

from fastapi import FastAPI

from app.core.logging import configure_logging
from app.db.postgres import close_pool
from app.db.redis import close_redis
from app.workers.curriculum_worker import request_shutdown, run_forever

logger = configure_logging()


@asynccontextmanager
async def lifespan(app: FastAPI):
    # Run the curriculum-parsing queue consumer as a background task
    # alongside the API. In production you'd more likely run
    # `python -m app.workers.curriculum_worker` as its own container/process
    # (see that module's docstring) so parsing load doesn't compete with
    # request handling, but a single background task keeps local dev/
    # docker-compose simple since only one ai-service container is defined.
    worker_task = asyncio.create_task(run_forever())
    logger.info("ai_service.startup", worker="curriculum_worker")

    yield

    request_shutdown()
    worker_task.cancel()
    try:
        await worker_task
    except asyncio.CancelledError:
        pass
    await close_pool()
    await close_redis()
    logger.info("ai_service.shutdown")


app = FastAPI(lifespan=lifespan)


@app.get("/")
def health():
    return {"status": "ok"}
