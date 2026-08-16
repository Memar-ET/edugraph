# EduGraph AI — Production Readiness Checklist

This document is the gate criteria before any phase of EduGraph goes live
beyond the Supabase-backed local-dev stack. Each section maps to a
production risk category; every item must be in state **✅ Done** or
explicitly accepted as a known risk before a national deployment.

---

## 1. Security

| # | Item | State |
|---|------|-------|
| 1.1 | All demo/default passwords (`password123`, `admin`) removed from seed data and Docker images | ⬜ TODO |
| 1.2 | Secrets (API keys, DB passwords, private keys) moved to AWS Secrets Manager; no secrets in `.env` or version control | ⬜ TODO |
| 1.3 | RS256 JWT keypair rotated from dev auto-generated to production key; rotation runbook documented | ⬜ TODO |
| 1.4 | bcrypt cost ≥12 enforced for all password hashes (already in code; verify against prod DB) | ✅ Done |
| 1.5 | Magic-byte file validation in place for all upload endpoints | ✅ Done |
| 1.6 | Security headers middleware (`X-Frame-Options`, `X-Content-Type-Options`, `Strict-Transport-Security`, `Content-Security-Policy`) wired | ⬜ TODO |
| 1.7 | Rate limiting: unauthenticated endpoints ≤30 req/min, authenticated ≤300 req/min | ⬜ TODO |
| 1.8 | Audit log table (`public.audit_log`) populated for sensitive mutations (credential provisioning, role changes, exam approvals) | ⬜ TODO |
| 1.9 | IDOR checks on every endpoint that takes a foreign-user ID in the path (verified: ExplainPage, career matches; others audited) | ✅ Done (ExplainPage verified) |
| 1.10 | Supabase RLS policies enabled as a defence-in-depth layer for student PII tables | ⬜ TODO |
| 1.11 | Device credentials for School Box stored bcrypt-hashed; plaintext secrets never logged or stored | ✅ Done |
| 1.12 | TLS enforced end-to-end (cloud API + School Box sync agent, `POSTGRES_SSLMODE=require` on Supabase) | ✅ Done (Supabase) |

---

## 2. Authentication & Authorisation

| # | Item | State |
|---|------|-------|
| 2.1 | RS256 access token (15 min) + refresh rotation (7 day, single-use) working in production | ✅ Done |
| 2.2 | `RequireRole` middleware enforced on every handler route | ✅ Done |
| 2.3 | Refresh-token revocation on logout and on password reset | ✅ Done |
| 2.4 | School Box device auth (`X-Device-Id`/`X-Device-Secret`) enforces school_id ownership server-side | ✅ Done |
| 2.5 | Emergency credential revocation procedure documented (`revoked_at` column in `sync.device_credentials`) | ✅ Done |

---

## 3. Data Integrity & Migrations

| # | Item | State |
|---|------|-------|
| 3.1 | Flyway migrations applied sequentially (V001–V049) against Supabase; no gaps or out-of-order applies | ✅ Done |
| 3.2 | Never edit a merged migration — new migrations only | ✅ Done (enforced by convention; one exception: V013 `app_storage` rename, documented) |
| 3.3 | `ApproveAndPromote` bulk upsert batching to avoid 200 s timeouts on full-grade uploads | ⬜ TODO |
| 3.4 | Vite dev-proxy timeout raised or removed so a long approval doesn't surface as a false 500 | ⬜ TODO |
| 3.5 | School Box `sync.applied_versions` LWW logic tested against concurrent push+pull on same entity | ⬜ TODO |
| 3.6 | Nightly backup of Supabase project verified (Supabase built-in PITR enabled) | ⬜ TODO |

---

## 4. Observability

| # | Item | State |
|---|------|-------|
| 4.1 | OpenTelemetry traces exported to Jaeger (wired in `main.go`; verify in prod) | ✅ Done (wired) |
| 4.2 | Prometheus `/metrics` endpoint on `:9091` serving Go runtime + request latency histograms | ⬜ TODO |
| 4.3 | Grafana dashboard covering API p50/p95/p99, error rate, worker queue depth | ⬜ TODO |
| 4.4 | Structured JSON logs (zap) in production; no unstructured `fmt.Println` | ✅ Done |
| 4.5 | Health endpoint (`GET /health`) returns 200 with `{"status":"ok"}` from all services | ✅ Done (api + sync-agent) |
| 4.6 | Alerting rule: error rate >5% for >5 min pages on-call | ⬜ TODO |
| 4.7 | School Box sync-agent health endpoint (`GET :8081/health`) monitored by school IT | ✅ Done |

---

## 5. Performance

| # | Item | State |
|---|------|-------|
| 5.1 | `POSTGRES_MAX_CONNS` ≤15 (pooler budget); verified no connection exhaustion under normal load | ✅ Done |
| 5.2 | `ApproveAndPromote` batched to reduce per-row round-trips over Supabase | ⬜ TODO |
| 5.3 | School quality score Redis cache (1 h TTL) warm on first request | ✅ Done |
| 5.4 | pgvector HNSW indexes on `embeddings.clo_embeddings` / `topic_embeddings` (real model TBD) | ✅ Done (schema only; real model pending) |
| 5.5 | Load test: 200 concurrent students submitting exams without gap-analysis queue saturation | ⬜ TODO |
| 5.6 | ai-service workers isolated to separate containers in production (one worker = one process) | ⬜ TODO (currently all in-process) |

---

## 6. CI / CD

| # | Item | State |
|---|------|-------|
| 6.1 | `go build ./... && go vet ./...` runs clean on every PR | ✅ Done |
| 6.2 | `python -c "import app.main"` import-check in CI | ✅ Done |
| 6.3 | `npm run type-check && npm run lint && npm run build` clean | ✅ Done |
| 6.4 | 100 Python unit tests pass (`pytest`) | ✅ Done |
| 6.5 | Go integration tests against real Postgres (not mocks) | ⬜ TODO |
| 6.6 | Automated Playwright E2E test suite covering golden paths (upload → approve → exam → gap analysis) | ⬜ TODO |
| 6.7 | Docker image vulnerability scan (Trivy or Snyk) in CI | ⬜ TODO |
| 6.8 | Production image builds use pinned base image digests, not `:latest` tags | ⬜ TODO |
| 6.9 | Blue/green or canary deployment strategy defined for API rollouts | ⬜ TODO |

---

## 7. Reliability & Error Handling

| # | Item | State |
|---|------|-------|
| 7.1 | Redis worker dead-letter queue: after N failures, move job to `queue:dlq` and alert | ⬜ TODO |
| 7.2 | `queue:gckt:trace` kept separate from `queue:gap:analyze` so one BRPOP consumer cannot starve the other | ✅ Done |
| 7.3 | School Box push/pull cycles idempotent: retry on partial failure doesn't duplicate cloud data | ✅ Done |
| 7.4 | Sync conflict rows in `sync.conflicts` are reviewed by ministry_admin on a regular cadence | ⬜ TODO (UI not built) |
| 7.5 | Neo4j prerequisite resync endpoint (`POST /curriculum/prerequisites/resync`) scheduled as a weekly job | ⬜ TODO |
| 7.6 | Graceful shutdown: in-flight requests finish, workers drain before process exits | ✅ Done (Go context propagation + signal.NotifyContext) |

---

## 8. Data Governance & Privacy

| # | Item | State |
|---|------|-------|
| 8.1 | Student PII (answers, gap records, skill states) never logged to Jaeger spans or console | ✅ Done (by convention; enforce via log review) |
| 8.2 | `students.*` tables excluded from School Box cloud-pull (pull only cloud-authoritative types) | ✅ Done |
| 8.3 | Data retention policy defined for `students.learning_events` (append-only, never deleted) | ⬜ TODO |
| 8.4 | Right-to-erasure process documented for student data | ⬜ TODO |
| 8.5 | Ethiopian FDRE personal-data protection regulations compliance reviewed | ⬜ TODO |

---

## 9. Model Governance (EG-GCKT)

| # | Item | State |
|---|------|-------|
| 9.1 | No model snapshot is promoted automatically — always requires ministry_admin review | ✅ Done |
| 9.2 | Rejected/superseded snapshots kept in `modeling.model_snapshots` (never deleted) | ✅ Done |
| 9.3 | Real embedding model chosen and `StubEmbeddingProvider` replaced before semantic search goes live | ⬜ TODO |
| 9.4 | DKT/MIRT deferred until ≥1,000 real graded student responses per skill | ✅ Done (documented) |
| 9.5 | Evaluation harness (`python -m app.services.evaluation.harness`) run after first 30-day real-data cohort | ⬜ TODO |

---

## 10. Operational Runbooks

| # | Item | State |
|---|------|-------|
| 10.1 | School Box provisioning runbook (create device credential, flash SD card, configure `.env.school-box`) | ⬜ TODO |
| 10.2 | JWT keypair rotation runbook | ⬜ TODO |
| 10.3 | Database incident runbook (Supabase PITR restore procedure) | ⬜ TODO |
| 10.4 | Neo4j full resync runbook (MATCH (n) DETACH DELETE + re-ingest) | ✅ Done (used during Biology ingest) |
| 10.5 | ai-service worker dead-letter triage runbook | ⬜ TODO |

---

## 11. Feature Completeness Blockers

| # | Feature | State |
|---|---------|-------|
| 11.1 | Career matcher ai-service route wired (`career.py` registered in `main.py`, `service.py` implemented) | ⬜ TODO |
| 11.2 | Real embedding model wired (Gemini embeddings or Ollama) replacing `StubEmbeddingProvider` | ⬜ TODO |
| 11.3 | S3StorageProvider implemented for production file storage (currently Postgres BYTEA only) | ⬜ TODO |
| 11.4 | ExplainPage linked from class heatmap and exam-insight pages | ⬜ TODO |
| 11.5 | EG-GCKT end-to-end exercised against live Docker stack (currently compile/import-verified only) | ⬜ TODO |

---

## 12. Sign-off

Before national rollout, the following roles must sign off:

- [ ] Engineering lead — all ✅ items verified against production environment
- [ ] Security lead — sections 1–2 reviewed
- [ ] MoE data officer — sections 8–9 reviewed
- [ ] School IT representative — section 10 runbooks tested

---

*Last updated: 2026-08-16. See `docs/SLO.md` for measurable service-level objectives.*
