# EduGraph AI — Service Level Objectives

These SLOs define the measurable targets for the production EduGraph stack.
They are the operationalised form of the quality bar in `docs/PRODUCTION_READINESS.md`.

All percentages are measured over a rolling 28-day window unless noted.

---

## 1. Availability

| Service | Target | Error budget (28 days) |
|---------|--------|------------------------|
| Cloud API (`api:8080`) | 99.5% | ~3.4 hours |
| AI service (`ai-service:8000`) | 99.0% | ~6.7 hours |
| School Box sync agent (per device) | 95.0% | ~33.6 hours |
| Neo4j (local Docker) | 99.0% | ~6.7 hours |

*Availability = fraction of 1-minute synthetic-probe checks that return HTTP 2xx on `/health`.*

---

## 2. Latency (cloud API)

| Endpoint class | p50 | p95 | p99 |
|----------------|-----|-----|-----|
| Auth (`/auth/*`) | ≤50 ms | ≤200 ms | ≤500 ms |
| Curriculum read endpoints | ≤100 ms | ≤400 ms | ≤1 s |
| Exam submit (triggers gap-analysis enqueue) | ≤200 ms | ≤800 ms | ≤2 s |
| Study plan read | ≤100 ms | ≤500 ms | ≤1 s |
| Tutor ask (Gemini call) | ≤2 s | ≤8 s | ≤15 s |
| School quality score (Redis cached) | ≤50 ms | ≤200 ms | ≤500 ms |

---

## 3. Background pipeline SLOs

| Pipeline | Definition | Target |
|----------|------------|--------|
| Curriculum parse | Time from Redis push to `status=parsed` | p95 ≤ 120 s |
| Gap analysis | Time from exam submission to `gap_records` inserted | p95 ≤ 60 s |
| Study plan generation | Time from gap analysis completion to plan available | p95 ≤ 30 s |
| EG-GCKT trace | Time from learning event to `skill_states` updated | p95 ≤ 90 s |
| Embedding generation | Time from approval to `clo_embeddings` upsert | p95 ≤ 180 s |

---

## 4. Error rate

| Scope | Target |
|-------|--------|
| Cloud API HTTP 5xx rate | ≤ 0.5% of all requests |
| Redis worker unhandled exception rate | ≤ 1% of all jobs |
| School Box push failure rate | ≤ 5% of sync cycles |

---

## 5. Data freshness (School Box)

| Metric | Target |
|--------|--------|
| Curriculum changes available on School Box | ≤ 5 minutes after cloud approval |
| Student attempt data available on cloud | ≤ 5 minutes after exam submit on School Box |
| Sync cycle interval | ≤ 60 seconds (configurable; default) |

---

## 6. Model governance

| Metric | Target |
|--------|--------|
| Refit candidate-to-review latency | Ministry admin reviews within 7 days |
| Parameter snapshot promotion reviewed before activation | 100% — no automatic activation |
| Evaluation harness run after each cohort | Within 14 days of cohort close |

---

## 7. Security SLOs

| Metric | Target |
|--------|--------|
| Mean time to revoke a compromised device credential | ≤ 1 hour |
| Mean time to rotate a compromised JWT keypair | ≤ 4 hours |
| Time to apply critical security patches | ≤ 48 hours from CVE disclosure |

---

## Measurement

- **API latency and error rate**: Prometheus histograms on `:9091/metrics`, visualised in Grafana.
- **Pipeline SLOs**: Postgres query over `updated_at - created_at` on the relevant job/record table.
- **Sync freshness**: Compare `sync.outbox.created_at` to `sync.outbox.pushed_at`; cloud-pull lag from `sync.applied_versions.applied_at` vs. cloud `synced_at`.
- **Availability probes**: External synthetic monitor (e.g. Grafana Cloud synthetic monitoring or a cron-based curl script) hitting `/health` every minute.

---

*Last updated: 2026-08-16. See `docs/PRODUCTION_READINESS.md` for the full readiness gate criteria.*
