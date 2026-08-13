# AI Tutor (Capability 3C)

## Correcting the record: this *is* in the PRD

An earlier review of this codebase concluded the AI Tutor "doesn't appear
anywhere in the PRD" and needed a scope decision (build it into the PRD,
or flag it as approved-but-undocumented scope creep). That conclusion was
wrong — it came from reviewing `edugraph-architecture.docx` only.
`edugraph-impl-plan.docx` documents it explicitly as **Capability 3C — AI
Tutor / Academic Assistant**, phase-tracked alongside gap analysis (3A)
and the study planner (3B), with its own dependency-chain entry ("AI
Tutor / Assistant — Explains topics, answers questions. Requires topic
graph + CLO descriptions + gap context").

The real gap is narrower and more specific than "undocumented": the PRD's
Capability 3C actually specifies **two** sub-features, and only one is
built.

## What's built: chat Q&A

`POST /api/v1/tutor/ask` (`ai-service/app/services/tutor/service.py`),
proxied through the Go backend's `internal/assessment` tutor endpoint
(auth/role checks happen there; the ai-service trusts the `studentId` it's
handed). This matches the PRD's described design closely:

- Retrieval runs over Postgres + Neo4j **before** any LLM call: keyword-match
  the question against curriculum topics at the student's grade and below,
  pull the student's own unresolved `gap_records` touching those topics
  (symptom or root cause), and walk the matched topic's prerequisite chain
  with the student's mastery on each link.
- All of that is injected as prompt context — "the student recently failed
  a question on this because they don't understand X, tailor the answer to
  bridge that specific gap" — the same curriculum-grounded, gap-aware
  design the PRD's Section 3.3 example describes.

## What's missing: practice-problem generation

The PRD also specifies `POST /ai/tutor/practice-problem` — given a
`topicId`/`cloCode`/`difficulty`/`language`, generate a practice problem,
store it for teacher review, then serve it to students. **None of this
exists.** There's no route, no service function, and no storage table for
it.

This is a real, larger gap than just "write the endpoint": the PRD's own
design for this feature depends on `(:KeyConcept)` graph nodes connected
via `HAS_CONCEPT`/`PREREQUISITE_OF` relationships that **don't exist in
the current Neo4j graph** either — the graph today only has
`Subject/Unit/Topic/CLO` nodes (confirmed directly: `MATCH (n) RETURN
labels(n)[0], count(n)` returns exactly those four labels, nothing else).
Building practice-problem generation as specified would mean extending
the curriculum-approval graph sync (`syncCurriculumGraph` in
`backend/internal/curriculum/repository/repository.go`) to also mirror
key concepts, before any generation logic could be written against them.

**Recommendation**: treat this as a real, scoped-out PRD feature for a
future session — not something to quietly drop from the spec, and not
something to build speculatively without a concrete need driving it.

## Offline capability (checklist 5.2)

Previously: no `GEMINI_API_KEY` meant every tutor request failed with
`TutorUnavailable` (HTTP 503) — the tutor was entirely unusable without
internet, contradicting the PRD's offline School Box requirement, same
gap gap-analysis and the study planner had.

Now: **Gemini → Ollama**, same two-tier pattern as
`gap_analysis/llm.py`, both verified end-to-end against real data (see
below). There's deliberately no third deterministic-text tier here (unlike
gap analysis's `_fallback_summary`) — an open-ended tutoring answer can't
be synthesized without an LLM, so `TutorUnavailable` is still raised, just
only once *both* tiers have failed.

**Amharic**: verified directly (not assumed) that both `qwen2.5:7b-instruct`
and `gemma2:9b` — the two models actually tested against this
deployment — produce garbled, non-functional Amharic (real Ge'ez
characters, incoherent as language, plus stray artifacts in unrelated
scripts). Rather than hand a student broken text in a live chat response,
the Ollama tier always answers in English regardless of the requested
language, appending a note that Amharic needs a cloud connection. Gemini's
Amharic output is unaffected — this only applies to the offline fallback
path.

### Verification

Ran the real `ask()` function against a real student and the actual
ingested Grade 7 Biology curriculum, with `GEMINI_API_KEY` unset:

- English request → fell through to Ollama, `model: qwen2.5:7b-instruct-q4_K_M`,
  real coherent answer grounded in the matched curriculum topics.
- Amharic request → also fell through to Ollama, correctly forced English
  instead of attempting (garbled) Amharic, with the offline-mode note
  appended.
- Both Gemini and Ollama unreachable → correctly raised `TutorUnavailable`.
