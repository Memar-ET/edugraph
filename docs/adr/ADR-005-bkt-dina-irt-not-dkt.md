# ADR-005: BKT/DINA/IRT for EG-GCKT; DKT and MIRT Explicitly Deferred

Date: 2026-08-15  
Status: Accepted

## Context

The EG-GCKT specification (`EduGraph_EG-GCKT_Professional_Model_Specification.docx.pdf`) defines five analytical engines for learner knowledge tracing: DINA/G-DINA, BKT, DKT (Deep Knowledge Tracing), MIRT (Multidimensional Item Response Theory), and 2PL IRT. The spec's own §21 roadmap defers DKT and full MIRT calibration to Phase 6 under the criterion "where justified by data."

At the time of implementation (2026-08-15), the system has effectively zero real student interaction data:
- The demo seed data has 0 exam answers.
- The only bulk data generator is a disposable k6 load-test fixture producing degenerate single-question answers.
- There is no longitudinal response sequence long enough to train a sequence model.

The spec's own Design Principles state: *"The system should be able to say 'unknown' or 'uncertain' rather than manufacture confidence."*

## Decision

**Implemented:**
- **BKT** — 4-parameter Bayesian Knowledge Tracing. Online, per-response forward-pass updates using standard literature defaults (`p_init=0.3, p_transit=0.15, p_slip=0.1, p_guess=0.25`) stored as an active `model_snapshots` row. Real math, honestly-labeled low-confidence inputs.
- **DINA** — Deterministic Input Noisy AND. Joint posterior over 2^k attribute patterns. Conjunctive multi-skill model. Population defaults (`slip=0.2, guess=0.2`).
- **IRT (2PL)** — Real logistic ability estimation via Newton-Raphson MLE. Provisional item parameters seeded from existing CTT heuristics (`exam_quality.go`); labeled as provisional.
- **GCSF fusion** — Weighted evidence fusion (BKT 0.8, DINA 0.8, IRT 0.5, graph_reasoning 0.3). Disagreement widens uncertainty.

**Explicitly deferred:**
- **DKT** — Requires longitudinal response sequences. The system doesn't have the data. Building a DKT model would produce fabricated results with no statistical basis, violating the spec's own core principle. `model_snapshots.model_type` reserves `dkt_model` for when this criterion is met.
- **True MIRT calibration** — Requires a well-calibrated multidimensional item bank across a large population. Same rationale. Reserves `mirt_calibration`.

**Not silently skipped:** Both deferrals are documented in CLAUDE.md, in the QA traceability register, and in the test suite as explicit `pytest.skip(...)` stubs (M05 and M09 in `test_model_validation_m01_m05.py` and `test_model_validation_m06_m10.py`).

## Consequences

**Good:**
- BKT/DINA/IRT produce real, statistically honest estimates with honest uncertainty on the data available.
- Cold start is handled correctly (population priors + high uncertainty) rather than fabricating point estimates.
- The deferred engines have reserved `model_type` values, so activating them later is a new implementation file + new snapshot row, not a schema change.

**Bad:**
- BKT/DINA/IRT all run on population default parameters (not empirically fitted) until enough data accumulates for the nightly refit to trigger.
- IRT discrimination and difficulty are seeded from CTT proxies, not real calibration. Labeled as provisional.
- DKT is the most powerful sequence model for mastery trajectory prediction; its absence means the system cannot detect forgetting curves or learning velocity changes as well as it eventually could.
