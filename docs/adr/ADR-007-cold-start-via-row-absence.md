# ADR-007: SKSG Cold Start via Row Absence (Never Fabricated 0.0)

Date: 2026-08-15  
Status: Accepted

## Context

The Student Knowledge State Graph (SKSG) requires an initial mastery estimate for every (student, topic) pair before any evidence has arrived. The naive approach is to pre-create rows with `mastery_probability = 0.0` or some population prior. The EG-GCKT specification §15 explicitly rejects this:

> "New skill / few observations → population priors + high uncertainty, never a fabricated point estimate. The system should say 'unknown' rather than manufacture confidence."

## Decision

**`students.skill_states` has no pre-created rows.** A row for `(student_id, topic_id)` only exists once the first evidence has arrived and `fuse_skill_state` has been called for that pair.

Design implications:

1. **`mastery_probability` is NULLABLE** — NULL means "no evidence yet," not "zero mastery." A query that returns NULL must be displayed as "Unknown" in the UI, not 0%.

2. **`fuse_skill_state` is a no-op when `fetch_unconsumed_evidence` returns an empty list** — no `upsert_skill_state` call is made. The SKSG row is never created for a new learner.

3. **Consistency checking short-circuits** on absent rows — `consistency.check_topic` returns 0 (no flags) when the dependent topic has no `skill_states` row. Absence of evidence is not treated as weakness.

4. **Snapshot suppression** — `snapshot.take_snapshot` returns `None` when the `skill_states` row doesn't exist or has `mastery_probability = NULL`. No fabricated historical state is written.

5. **Replay** — `replay.fuse_point_in_time([])` returns `None`. `reconstruct_mastery_as_of` returns `None` for a learner with no prior evidence at any timestamp.

This invariant is verified by the `COLD-001` test suite in `ai-service/tests/test_cold_start_cold001.py` (12 tests, all asserting `None` or no-write behaviour at every pipeline layer).

## Consequences

**Good:**
- No fabricated confidence. The system says "unknown" rather than presenting a meaningless 0% that looks like evidence of non-mastery.
- Downstream features (consistency, recovery, recommendations) handle the cold-start case gracefully without special-casing a fake 0.0 value.
- The invariant is explicitly tested at every layer of the pipeline.

**Bad:**
- Every query that joins `students.skill_states` must handle the case where the row does not exist (outer join, NULL check). Forgetting this produces silent result-set gaps.
- The frontend must display a distinct "No data yet" state rather than treating NULL as 0%.
- Root-cause analysis returns `None` for a learner who has never been assessed — the gap pipeline only produces results after at least one exam submission.
