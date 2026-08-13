# AI Models: Local vs. Cloud (checklist 8.1/8.2/8.3)

## 8.1 — Decision: build local-first by default

**Decided and implemented**: local (Ollama) is the primary, assumed AI
provider everywhere in this codebase; cloud (Gemini) is available behind
the same interface as a config-only swap, not the default. This matches
the PRD's core "School Box operates fully offline" requirement — the
whole point of the offline edge deployment falls apart if the platform's
AI features assume an internet connection by default.

Switching to cloud-first (`LLM_PROVIDER=cloud` in `.env`/environment) is
a one-value config change, not a rewrite — see `app/utils/llm_provider.py`.
Whichever provider ISN'T the configured primary is still tried as a
resilience fallback if the primary is unreachable, in both directions:
local-first configs still get a shot at Gemini if a key happens to be
set and Ollama is briefly down; cloud-first configs still fall back to
the offline-capable local model rather than going silent.

## 8.2 — Local embedding generator

Already done before this pass (Capability 2.1, `app/utils/embeddings.py`):
`fastembed` running `intfloat/multilingual-e5-large` (1024-dim) entirely
on CPU, no cloud API, no GPU. `EmbeddingProvider`/`FastEmbedProvider`
already followed the exact adapter pattern this session generalized to
text generation — this was the precedent `llm_provider.py`'s design
copies. There's no cloud embedding alternative anywhere in this codebase
(semantic matching only ever needed the one local model), so there was
nothing to make swappable here the way text generation needed.

## 8.3 — Local LLM (Ollama) fallback for gap analysis, study plans, and the tutor

Done, and generalized one step further than asked: rather than each of
the three named features (plus a fourth real call site found along the
way, the exam-to-CLO LLM matcher) hardcoding its own Gemini-then-Ollama
tiering logic — which is what this session originally built, three times
over, before this pass — all four now share one interface:
`app/utils/llm_provider.py`'s `LLMProvider` ABC
(`OllamaProvider`/`GeminiProvider`), `get_llm_provider()` /
`get_fallback_provider()` (config-driven selection), and
`generate_with_fallback()` (the actual try-primary-then-fallback call
every feature module uses).

**Why generalize instead of just flipping the tier order three times**:
the duplicated per-feature Gemini/Ollama HTTP logic was already showing
real drift (a genuine bug — an inverted language conditional in
study_plan's Ollama prompt selection — existed only because that logic
was hand-copied instead of shared). A single interface makes "local is
primary, cloud is a swap" a fact about the system once, not three facts
that can quietly disagree.

**What the interface does NOT abstract**: prompt content. Each feature
module still owns its own prompt wording, JSON response shape, and
parsing — `llm_provider.py`'s job is purely "send this text to a model,
get text back," matching whichever provider is configured. The one thing
it does expose to callers is `provider.supports_amharic` (`True` for
Gemini, `False` for the local models actually tested against this
deployment — see below), so a feature's prompt-building function can
vary bilingual-vs-English-only generically instead of every module
re-discovering the same Amharic limitation independently.

### Amharic

Verified directly, not assumed: `qwen2.5:7b-instruct` and `gemma2:9b`
(the two local models actually pulled and tested against this
deployment) both produce garbled, non-functional Amharic — real Ge'ez
characters, incoherent as language, with stray artifacts in unrelated
scripts. Confirmed with the user that Amharic isn't mandatory at this
stage. Every local-first call site therefore requests English-only from
Ollama regardless of what a caller originally wanted, rather than ship
broken text; the tutor additionally appends a short note when it had to
downgrade a live chat response from Amharic to English. Gemini's
Amharic output is unaffected — this is purely a local-model limitation,
not a design choice against Amharic itself. If a future local model with
genuine Amharic support is found, this is a one-flag change
(`OllamaProvider.supports_amharic = True`), not a multi-file rewrite.

### Verification

All four call sites re-verified against real data after the refactor
(not just import-checked):

- **Gap analysis**: re-ran the real `process_gap_job` against the
  already-ingested Grade 7 Biology data with `GEMINI_API_KEY` unset --
  `llm_model` correctly recorded `qwen2.5:7b-instruct-q4_K_M`, clean
  English summary.
- **AI Tutor**: English request and Amharic-requested-but-forced-English
  (with the offline-mode note) both re-verified; both-providers-down
  still correctly raises `TutorUnavailable`.
- **Study plan**: re-ran the real 4-day plan (the same multi-level
  prerequisite chain from checklist 4.1) -- all 4 days re-enriched
  correctly, topological order unaffected by the refactor.
- **Exam CLO matcher**: matched a real question ("What is biology and
  what does it study?") to the correct CLO among real candidates via
  the local provider.
- **Provider switching itself**: confirmed `LLM_PROVIDER=cloud` actually
  changes `get_llm_provider()` to return Gemini, and that with no
  `GEMINI_API_KEY` set, `generate_with_fallback` correctly falls through
  to the local model rather than failing -- the resilience path works in
  both configured directions, not just the default one.
