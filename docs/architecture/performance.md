# Performance Verification (checklist 13.1, 13.2)

## 13.1 — Gap-analysis speed vs. the PRD's target

No gap-analysis-specific latency target exists anywhere in the PRD docs
(checked `graphify-out/converted/*.md` directly, not assumed). The
closest stated proxy is `edugraph-architecture.docx`'s SLO table: **"AI
job completion (study plan): 95% of jobs complete in < 30 seconds."**
Gap analysis and study plan are architecturally identical — both one
LLM call via `generate_with_fallback`, dispatched off a Redis-queued
job — so this benchmark uses that same 30s/P95 bar, explicitly as a
proxy, not a number the PRD states for this specific feature.

**Tool**: `ai-service/scripts/benchmark_gap_analysis.py`, calling the
real `synthesize_insights` against the real local Ollama
(`qwen2.5:7b-instruct-q4_K_M`), not a mock — the correctness tests
(`tests/test_gap_analysis_llm.py`) already cover the mocked path.

**Result** (5 runs, 10-gap payload — `MAX_GAPS_TO_EXPLAIN`'s ceiling,
the representative worst case):

```
P50: 7.65s   P95: 12.31s   min: 7.26s   max: 12.31s
PASS -- P95 (12.31s) is within the 30s proxy target.
```

**Caveat**: run on a development machine, not School Box reference
hardware (`school-box/README.md`'s assumed 4–8 core/16GB baseline).
Real School Box CPUs are likely to be slower than this dev machine, so
this P95 should be treated as a best case, not a guarantee — re-run this
script on actual target hardware before trusting it for a real
deployment.

## 13.2 — Exam-submission load test vs. the PRD's concurrency target

**What the PRD actually asks for** (checked directly, not assumed):
`edugraph-impl-plan.docx` states a load-test target of **50,000
concurrent students submitting exams, asserting P99 < 400ms**, and
`edugraph-architecture.docx`'s capacity table adds a **200,000-user
spike scenario** for national results day. There is no "150" anywhere
in the PRD — that number doesn't appear in any of the four converted
planning documents.

**What this repo actually is**: per `CLAUDE.md`'s explicit instruction
("don't build unused production infra"), the Kong/Kubernetes/multi-AZ/
horizontal-autoscaling architecture the PRD's 50k/200k targets assume
does not exist here — what exists is **one Go binary**
(`cmd/api/main.go`), no load balancer, no horizontal scaling, backed by
a Supabase connection pooler capped at `POSTGRES_MAX_CONNS=15` in the
real cloud deployment. Load-testing toward 50,000 concurrent requests
against that would only measure how fast a single process falls over on
one machine — not anything meaningful about the architecture the PRD
describes, and not something this repo was ever built to sustain. This
section tests **current reality** and reports the gap honestly instead.

**Tooling**: `k6/exam-submit-load-test.js` (real login → list questions
→ submit, against the real running `api` binary — not mocked at any
layer) driven by `k6/run-load-test.sh` at increasing concurrency, using
synthetic student accounts from `backend/cmd/seed-load-test` (one
bcrypt hash computed once and reused across every seeded account — real
accounts never share a password, this is a load-test-fixture-only
shortcut to keep seeding itself from becoming the bottleneck).

**Results** (local Postgres, not the Supabase pooler — see caveat
below; `http_req_duration` aggregated across all three steps):

| Concurrent students | avg | P90 | P95 | Failures |
|---|---|---|---|---|
| 10 | 77ms | 223ms | 223ms | 0% |
| 25 | 114ms | 329ms | 350ms | 0% |
| 50 | 228ms | 662ms | 675ms | 0% |
| 100 | 458ms | 1,328ms | 1,331ms | 0% |
| 200 | 931ms | 2,664ms | 2,704ms | 0% |
| 400 | 2.07s | 6.00s | 6.01s | 0% |
| 800 | 3.81s | 11.06s | 11.07s | 0% |

**Zero failures at every level tested, up to 800 concurrent** — the
exam-submission path itself doesn't break under this load, it degrades
smoothly. A per-step breakdown at 200 concurrent (`step` tags in the k6
script) shows exactly why:

| Step | avg | P95 |
|---|---|---|
| `login` (bcrypt password verification, cost 12) | 2,481ms | 2,711ms |
| `list_questions` | 227ms | 676ms |
| `submit` (the actual thing being load-tested) | **18ms** | **26ms** |

**The bottleneck is login, not exam submission.** bcrypt cost 12 is a
deliberate security choice (see `CLAUDE.md`'s Critical Decisions — hardened
against `edugraph-architecture.docx`'s stated minimum) and is
CPU-bound and inherently serial per request; a single process on one
machine has a hard ceiling on concurrent bcrypt computations regardless
of how fast everything downstream of auth is. The actual submission
write path (the thing the PRD's target is about) stays under 30ms P95
even at 200 concurrent — it's `login`'s cost that dominates the
end-to-end number.

**At 1,200 requested VUs, the test's own client-side k6 process could
only sustain ~800 concurrent connections on this machine** (`vus_max`
reported 1200 but `vus` topped out around 800) — this is a load-*generator*
limit (single-machine k6, not distributed), not confirmed evidence of a
server-side ceiling above 800. The real server-side ceiling is higher
than what was measured here; finding it precisely needs a distributed
load generator, out of scope for this pass.

**Gap vs. the PRD's 50,000/P99<400ms target, and why it exists**: at
just 100 concurrent logins, P95 is already ~1.3s — several times over
the PRD's 400ms bar, well before reaching even 1% of the stated 50,000
target. This isn't a code defect to fix; it's the direct, expected
consequence of the architecture decision already documented in
`CLAUDE.md`: this repo is a single-instance system, and the PRD's
targets assume the horizontal-scaling infrastructure (Kong, Kubernetes
autoscaling, multiple API pods behind a load balancer) that was
deliberately not built. Closing this gap for real would need, in order
of impact: (1) running multiple `api` instances behind a load balancer
— the `login` bottleneck is CPU-bound per-request work that parallelizes
trivially across processes/cores, unlike a shared-state bottleneck; (2)
a connection-pooler sized for real concurrency, not the current
cloud deployment's 15-connection Supabase cap; (3) session-affinity-free
horizontal scaling, which the codebase's stateless-JWT-cookie auth (see
[security.md](security.md)) already supports without further redesign
— the sessions aren't sticky to one instance today. None of this is
built in this pass — documenting the path, not walking it, matches the
scope actually asked for here.

**Caveat**: this ran against a local, dedicated Postgres container, not
the real cloud deployment's Supabase pooler (`POSTGRES_MAX_CONNS=15`,
plus real network latency to the pooler). A cloud-deployed single
instance would likely show a *lower* ceiling than measured here, not a
higher one.
