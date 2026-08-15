# ADR-002: Plain Redis LPUSH/BRPOP Instead of Celery

Date: 2026-07-17  
Status: Accepted

## Context

The ai-service needs to receive async work (curriculum parsing, exam parsing, gap analysis, etc.) from the Go API. The standard Python async worker framework is Celery, which uses a Redis or RabbitMQ broker and provides task routing, retries, result backends, and monitoring.

However, the Go API producer pushes jobs using a raw Redis `LPUSH` of a plain string (the job id). Celery expects its message-protocol envelope: a JSON blob with `task`, `id`, `args`, `kwargs`, `retries`, `eta`, and other fields. A Celery consumer receiving a raw string would fail to deserialise it.

## Decision

Each ai-service worker is a plain Python asyncio coroutine that does:

```python
while True:
    _, raw = await redis.brpop(QUEUE_NAME)
    job_id = raw.decode()
    await process(job_id)
```

No Celery dependency. The Go producer does:

```go
redis.LPush(ctx, "queue:curriculum:parse", jobID)
```

This is the simplest protocol that works across both languages without adding a serialisation dependency on either side. The rationale is documented inline in `ai-service/app/workers/curriculum_worker.py`.

The one exception to the "BRPOP on a queue" pattern is `refit_worker.py` (EG-GCKT, 2026-08-15), which is a time-based asyncio periodic loop (no queue) because "refit when enough evidence accumulates" has no single triggering event.

## Consequences

**Good:**
- Zero marshalling overhead — job ids are short strings.
- No Celery process management complexity (beat scheduler, flower monitoring, etc.).
- Works identically in local dev (single Docker Compose service) and any multi-process deployment.
- Adding a new queue is a one-liner on both sides.

**Bad:**
- No built-in retry logic — a failed job is lost unless the worker explicitly re-enqueues it or writes a failure record.
- No task result backend — callers cannot await a result; they must poll Postgres for status changes.
- No dead-letter queue — stuck or repeatedly-failing jobs need manual intervention.
- `requirements.txt` previously listed Celery (legacy from an earlier design); the actual imports were never there.
