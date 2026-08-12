# Gap Analysis Job (Capability 3A)

Runs when an exam attempt becomes fully graded: the Go backend `LPUSH`es
the attempt id onto `queue:gap:analyze` (`exam_submit.go` /
`BulkGradeExam`), and `ai-service/app/workers/gap_worker.py` — a plain
`asyncio` `BRPOP` loop, not Celery, same convention as every other worker
in this service — picks it up and calls
`app/services/gap_analysis/service.py::process_gap_job`.

## What actually runs today (superseding the PRD's description)

The PRD describes this as calibration-verification-driven. The pipeline
that is actually implemented is a different, 3-pass design:

1. **Question-Level Triage** ("what") — every question the student lost
   marks on is traced `Question -> CLO -> Topic` (direct `topic_id` first,
   falling back to the CLO's best-aligned topic). That topic is the
   *symptom*.
2. **Root Cause Traversal** ("where") — walk backwards up the prerequisite
   graph from each symptom topic and find the deepest prerequisite the
   student has *evidence* of being weak in. Graph-first: Neo4j
   `HAS_PREREQUISITE` edges are tried before falling back to Postgres
   `curriculum.topic_prerequisites` (see "Prerequisite data" below).
3. **LLM Synthesis** ("why") — one Gemini call turns symptom/root-cause
   pairs into per-gap explanations plus a bilingual (EN/AM) exam-level
   narrative, with a deterministic English fallback (`_fallback_summary`)
   so `exam_insights` is never empty even with no `GEMINI_API_KEY`.

Calibration (severity scoring, weak-threshold comparison) is folded into
Pass 2, not a separate pass. This is a reasonable evolution of the PRD's
original design, not a bug — but the PRD document itself still describes
the old shape and should be updated to match this.

Storage, one transaction (`postgres_gap.persist_analysis`):
`students.gap_records` (granular, per missed question),
`students.exam_insights` (one row per attempt), `students.subject_profiles`
(rolling subject health), plus a `students.mastery_records` refresh.

## Prerequisite data: mechanism vs. population

**The mechanism is complete and verified working** (see "How this was
verified" below): a curriculum officer (or `ministry_admin`) calls
`POST /api/v1/curriculum/topics/:id/prerequisites`
(`curriculum/service/prerequisites.go::AddTopicPrerequisite`), which in
one call:

- Cycle-checks the new edge against existing `curriculum.topic_prerequisites`
  rows (rejects anything that would make the topological sort the study
  planner (3B) depends on unsolvable).
- Upserts the Postgres row, `infer_method` in
  `('explicit', 'ai_inferred', 'manual', 'moe_document')`, `reviewed_by`/
  `confirmed_at` set — this **is** the validated-vs-inferred distinction
  checklist item 1.4 asked for; it already exists, just not as a separate
  boolean column.
- Best-effort mirrors the same edge into Neo4j as
  `(:Topic)-[:HAS_PREREQUISITE]->(:Topic)` (`SyncPrerequisiteToNeo4j`),
  reporting `graphSynced: false` in the response (not failing the request)
  if the mirror write fails — same contract as curriculum approval's
  Subject/Unit/Topic sync.

**What's actually missing is population, not mechanism**: there is no
seed data anywhere in the migration set, and no frontend page exists for
a curriculum officer to actually call this endpoint (`grep -r
"prerequisite" frontend/src` turns up only type definitions and an
unrelated exam-review reference — no curation UI). So today, with a fresh
`docker compose up`, the prerequisite graph is empty in both stores, and
every gap gets `root_cause_topic_id = NULL`, `is_root_cause = false`,
`prerequisite_depth = 0` — not because anything is broken, but because
nobody has entered the data yet. This is checklist item 1.6 (seed a real
prerequisite graph for at least one subject) and the missing piece of
1.5/1.4's UI surface — both are unblocked by the API existing, but the
API alone doesn't get you a populated graph.

## Cold-start handling ("never tested" prerequisites)

`fetch_topic_mastery` in `postgres_gap.py` computes mastery from the
student's actual graded-answer history; a topic with **zero** answer
history and no `mastery_records` row is simply **absent** from the
returned dict. The root-cause selection reads it as
`mastery.get(topic_id, 1.0)` — i.e. **no evidence defaults to full
mastery**, so an untested prerequisite can never be flagged as the
weak link. This directly implements "no evidence is not weakness"
(the docstring's stated design intent) and was empirically confirmed
(see below): a prerequisite with zero answer history was correctly
skipped even though it was the *shallower* (closer) topic in the chain,
in favor of a deeper topic that actually had evidence of weakness.

This solves half of the PRD's original cold-start concern (never
falsely blame an untested topic) but **not** the other half: there is
still no deliberate mechanism to *probe* old prerequisite topics on
current-year exams, so if a foundational gap is never touched by any
exam, the engine's most valuable output — an actual confirmed root
cause — simply never has a chance to fire, even once the graph is
populated. That remains open; a probing mechanism (e.g. periodically
including a low-weight prerequisite-topic question on unrelated exams)
is a separate, larger feature, not implemented here.

## How this was verified

Reading the code isn't sufficient to confirm "runs on real graph data,
not falling back silently" — both `fetch_prerequisite_chain` (Neo4j) and
`fetch_prerequisite_chain_pg` (Postgres) needed to be exercised for real.
Verification seeded a genuine 2-hop chain (`Quadratic Equations` requires
`Linear Equations` requires `Basic Arithmetic`) into both a live Neo4j
instance and Postgres, with a student who: missed an exam question on the
symptom topic, had a documented weak (30%) score on the deepest
prerequisite, and had **zero** answer history on the middle prerequisite
(the cold-start case), then ran the real `process_gap_job` against five
scenarios:

| Scenario | Result |
|---|---|
| Neo4j + Postgres both populated | Root cause found at depth 2 via Neo4j |
| Neo4j populated, Postgres empty | Root cause found at depth 2 (confirms Neo4j path alone works) |
| Neo4j empty, Postgres populated | Root cause found at depth 2 (confirms the Postgres fallback alone works) |
| Neo4j unreachable (bad auth), Postgres populated | Logs `neo4j.prerequisite_chain_unavailable`, falls back to Postgres, finds root cause |
| Both empty (today's actual shipped state) | `root_causes=0`, no error, no log line — silent by design |

In every case the untested middle topic was correctly never blamed. This
confirms the traversal mechanism itself is correct and will work the
moment real prerequisite data exists; the gap is purely that it doesn't
exist yet in any running environment.

## Debugging a gap-analysis run

```bash
# Tail the worker's structured logs for a specific attempt
docker compose logs ai-service | grep <attempt_id>

# Inspect what got written
psql -c "SELECT * FROM students.gap_records WHERE attempt_id = '<attempt_id>'"
psql -c "SELECT * FROM students.exam_insights WHERE attempt_id = '<attempt_id>'"

# Re-run analysis for an attempt directly (bypasses the queue)
python -c "
import asyncio
from app.services.gap_analysis.service import process_gap_job
asyncio.run(process_gap_job('<attempt_id>'))
"
```

`persist_analysis` deletes and reinserts `gap_records` by `attempt_id`,
so re-running for the same attempt is always safe.

If `root_causes` is unexpectedly `0` for an attempt where you'd expect a
prerequisite chain to exist, check `curriculum.topic_prerequisites` and
the Neo4j `HAS_PREREQUISITE` edges for the symptom topic — as of this
writing that's the expected state for every topic, since no prerequisite
data has been seeded anywhere yet.
