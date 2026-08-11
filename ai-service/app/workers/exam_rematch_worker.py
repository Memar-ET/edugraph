"""
Capability 2D: consumes queue:exam:rematch and re-runs CLO matching for an
exam's already-extracted questions, without touching the source file.

Pushed by the Go API (assessment/service/exam_scope.go's
enqueueCLORematch) whenever a teacher's PATCH /api/v1/exams/:id/scope
changes subjectCode or gradeLevel -- every question's existing clo_code was
matched against the *old* subject's CLO set by exam_parser/service.py at
upload time and is now stale. This worker fixes that in place: same
matcher functions (clo_matcher_llm then clo_matcher, same "ai_draft"/
"llm_verified" method tags) as the original 2A parse, just re-run against
the corrected subject_code and reading question_text back out of Postgres
instead of re-extracting it from the PDF/DOCX.

Runs the same two ways as the other workers -- in-process background task
from app/main.py's lifespan for local/dev, or as its own process in
production via `python -m app.workers.exam_rematch_worker`.
"""

from __future__ import annotations

import asyncio

import structlog

from app.db import postgres_assessment as db
from app.db import postgres_embeddings
from app.db.redis import EXAM_REMATCH_QUEUE, brpop_job
from app.services.exam_parser import clo_matcher, clo_matcher_llm, vector_matcher
from app.utils import embeddings as embed_utils

logger = structlog.get_logger()

_shutdown = asyncio.Event()


async def process_rematch_job(exam_id: str) -> None:
    log = logger.bind(exam_id=exam_id)

    exam = await db.fetch_exam_for_rematch(exam_id)
    if exam is None:
        log.warning("exam_rematch.exam_not_found")
        return

    clo_rows = await db.fetch_clos_for_subject(exam["subject_code"])
    clo_pairs = [(r["code"], r["description_en"]) for r in clo_rows]

    questions = await db.fetch_questions_for_rematch(exam_id)
    if not questions:
        log.info("exam_rematch.no_questions")
        return

    rematched = 0
    for q in questions:
        code, score, method = None, None, None

        vec_code, vec_score, _classification = await vector_matcher.match_clo_vector(q["question_text"], exam["subject_code"])
        if vec_code:
            code, score, method = vec_code, vec_score, "embedding_auto"
            # Unlike the initial 2A parse (questions don't have ids yet at
            # that point), this question already has one -- worth
            # persisting the embedding for a future rematch/audit.
            if embed_utils.is_available():
                try:
                    await postgres_embeddings.upsert_question_embedding(str(q["id"]), embed_utils.embed_query(q["question_text"]), embed_utils.model_name())
                except Exception:  # noqa: BLE001 -- best-effort, matching already succeeded
                    log.exception("exam_rematch.embedding_store_failed", question_id=str(q["id"]))
        else:
            llm_code, llm_score = await clo_matcher_llm.match_clo_llm(q["question_text"], clo_pairs)
            if llm_code:
                code, score, method = llm_code, llm_score, "llm_verified"
            else:
                kw_code, kw_score = clo_matcher.match_clo(q["question_text"], clo_pairs)
                if kw_code:
                    code, score, method = kw_code, kw_score, "ai_draft"

        await db.update_question_clo_match(str(q["id"]), code, score, method)
        if code:
            rematched += 1

    log.info("exam_rematch.completed", question_count=len(questions), rematched=rematched, subject_code=exam["subject_code"])


async def run_forever(poll_timeout: int = 5) -> None:
    logger.info("exam_rematch_worker.started", queue=EXAM_REMATCH_QUEUE)
    while not _shutdown.is_set():
        try:
            exam_id = await brpop_job(EXAM_REMATCH_QUEUE, timeout=poll_timeout)
        except Exception:  # noqa: BLE001 -- keep the loop alive on transient Redis errors
            logger.exception("exam_rematch_worker.redis_error")
            await asyncio.sleep(poll_timeout)
            continue

        if exam_id is None:
            continue  # BRPOP timed out with nothing queued -- loop and check shutdown

        try:
            await process_rematch_job(exam_id)
        except Exception:  # noqa: BLE001 -- a single bad job must not kill the worker
            logger.exception("exam_rematch_worker.unhandled_job_error", exam_id=exam_id)


def request_shutdown() -> None:
    _shutdown.set()


if __name__ == "__main__":
    try:
        asyncio.run(run_forever())
    except KeyboardInterrupt:
        pass
