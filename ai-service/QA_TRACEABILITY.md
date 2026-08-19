# QA-001: Formal Verification Traceability Register
# EduGraph EG-GCKT — QA Test Case & Failure Injection Specification

Generated: 2026-08-15  
Source specs:
- `EduGraph_QA_Test_Case_and_Failure_Injection_Specification.docx` (FI-001..FI-008, CONS-001, REPLAY-001, COLD-001)
- `EduGraph_Whole_System_Model_Validation_Test_Specification.docx` (M01..M15)

Test runner: `python -m pytest tests/ -q` (398 passed, 28 skipped, 0 failures)

---

## Test Case → Implementation Map

| Spec ID | Title | Test File | Key Test Functions | Production Code Exercised |
|---|---|---|---|---|
| FI-001 | Missing Evidence | `test_failure_injection.py` | `test_fi001_missing_evidence_produces_no_fabricated_state`, `test_fi001_evidence_with_no_point_estimate_is_marked_consumed_but_not_fabricated` | `fusion.fuse_skill_state` (no-op on empty evidence; drain-but-don't-write for estimate=None rows) |
| FI-002 | Contradictory Model Outputs | `test_failure_injection.py` | `test_fi002_disagreeing_sources_widen_uncertainty_rather_than_average_it_away`, `test_fi002_agreeing_sources_do_not_inflate_uncertainty` | `fusion.fuse_skill_state` (disagreement term in `fused_uncertainty`) |
| FI-003 | Stale Evidence | `test_failure_injection.py` | `test_fi003_stale_evidence_is_down_weighted_relative_to_fresh` | `fusion._recency_weight`, `fusion._source_weight` |
| FI-004 | Invalid Graph Edge | `test_failure_injection.py` | `test_fi004_missing_prerequisite_endpoint_data_yields_unknown_not_fabricated_readiness` | `root_cause._prerequisite_readiness` (1.0 when no resolvable prerequisites) |
| FI-005 | Invalid Q-Matrix Mapping | `test_failure_injection.py` | `test_fi005_unmapped_skill_ids_produce_no_evidence_and_no_state` | `fusion.fuse_skill_state` (no state written when no evidence exists for a topic) |
| FI-006 | Duplicate Event | `test_failure_injection.py` | `test_fi006_duplicate_evidence_ids_are_consumed_exactly_once_per_pass` | `fusion.fuse_skill_state` + `mark_evidence_consumed` |
| FI-007 | Model Failure | `test_failure_injection.py` | `test_fi007_engine_failure_aborts_the_job_without_writing_partial_state`, `test_fi007_worker_loop_contains_a_failed_job_and_continues` | `kt_worker.process_knowledge_tracing_job`, `kt_worker.run_forever` |
| FI-008 | Graph-Service Failure | `test_failure_injection.py` | `test_fi008_neo4j_outage_degrades_to_postgres_mirror_for_structural_status`, `test_fi008_graph_service_failure_in_root_cause_falls_back_not_raises` | `fusion._compute_structural_status`, `root_cause.downstream_impact` (try/except + PG fallback) |
| CONS-001 | A→B→C Consistency | `test_consistency_cons001.py` | `test_cons001_strong_topic_with_evidenced_weak_prerequisite_is_flagged`, `test_cons001_mastery_state_is_not_forced_down_after_inconsistency`, `test_cons001_sparse_evidence_on_*_suppresses_flagging` (×3), `test_cons001_no_evidence_on_prerequisite_produces_no_flag`, `test_cons001_consistent_chain_produces_no_flag` | `consistency.check_topic`, `consistency._is_sparse`, `consistency._penalize_edge_confidence` |
| REPLAY-001 | Replay Tolerance | `test_replay_replay001.py` | `test_replay001_evidence_strictly_before_cutoff_is_used`, `test_replay001_no_evidence_before_cutoff_returns_none`, `test_replay001_determinism_same_inputs_same_output`, `test_replay001_stale_evidence_is_down_weighted`, `test_replay001_policy_weight_change_shifts_estimate_within_tolerance`, `test_replay001_reconstruct_mastery_as_of_*` (×2) | `replay.fuse_point_in_time`, `replay.reconstruct_mastery_as_of`, `replay.prior_evidence` |
| COLD-001 | End-to-End Cold Start | `test_cold_start_cold001.py` | `test_cold001_new_learner_fusion_is_noop`, `test_cold001_consistency_check_on_new_learner_returns_zero`, `test_cold001_*` (×11 total) | `fusion.fuse_skill_state`, `consistency.check_topic`, `root_cause.downstream_impact`, `root_cause._prerequisite_readiness`, `root_cause.produce_graph_evidence`, `root_cause.score_candidates`, `snapshot.take_snapshot`, `replay.reconstruct_mastery_as_of`, `replay.fuse_point_in_time` |
| M01 | Educational Graph Validation | `test_model_validation_m01_m05.py` | `test_m01_t01..t05` (5 tests) | `root_cause.compute_rcs`, `root_cause.downstream_impact`, `root_cause.score_candidates`, `root_cause.produce_graph_evidence` |
| M02 | Evidence Ingestion | `test_model_validation_m01_m05.py` | `test_m02_t01..t06` (6 tests) | `fusion.fuse_skill_state` (valid/duplicate/estimateless/stale/schema/multi-topic) |
| M03 | Q-Matrix / Skill Mapping | `test_model_validation_m01_m05.py` | `test_m03_t01..t07` (7 tests) | `fusion.fuse_skill_state` (1:1, 1:many, low-confidence, missing, invalid, provenance, many:1) |
| M04 | Learner-State Inference | `test_model_validation_m01_m05.py` | `test_m04_t01..t09` (9 tests) | `bkt._bkt_update`, `fusion.fuse_skill_state`, `fusion._mastery_status`, `fusion._trend` |
| M05 | Sequential Knowledge Tracing (DKT) | `test_model_validation_m01_m05.py` | 10 tests all `pytest.skip` — DKT not implemented, N/A per spec §5 | N/A |
| M06 | Psychometric Model | `test_model_validation_m06_m10.py` | `test_m06_t01..t08` (8 tests) | `irt.p_correct`, `irt._estimate_theta` |
| M07 | Diagnostic Inference | `test_model_validation_m06_m10.py` | `test_m07_t01..t08` (8 tests) | `root_cause.compute_rcs`, `root_cause.score_candidates`, `root_cause._prerequisite_readiness` |
| M08 | Prediction & Evaluation | `test_model_validation_m06_m10.py` | `test_m08_t01..t09` (9 tests) | `bkt._bkt_update`, `irt._estimate_theta`, `metrics.auc`, `metrics.log_loss`, `metrics.brier_score`, `metrics.expected_calibration_error` |
| M09 | Cohort-Level Model | `test_model_validation_m06_m10.py` | 8 tests all `pytest.skip` — no cohort model, N/A per spec §9 | N/A |
| M10 | Recommendation Ranking | `test_model_validation_m06_m10.py` | `test_m10_t01..t10` (10 tests) | `action_ranking.compute_action_scores`, `action_ranking.classify_action_types`, `action_ranking._difficulty_fit` |
| M11 | Recommendation Outcomes | `test_model_validation_m11_m15.py` | `test_m11_t01..t08` (8 tests) | `refit_worker._classify_outcome` — includes cold-start-never-worsened invariant |
| M12 | Calibration & Uncertainty | `test_model_validation_m11_m15.py` | `test_m12_t01..t08` (8 tests) | `fusion.fuse_skill_state` (uncertainty widening), `fusion._mastery_status`, `fusion._source_weight`, `metrics.brier_score`, `metrics.expected_calibration_error` |
| M13 | Provenance & Explainability | `test_model_validation_m11_m15.py` | `test_m13_t01..t06` (6 tests) | `root_cause.compute_rcs`, `fusion._source_weight`, `fusion._mastery_status`, `fusion._trend`, `replay.fuse_point_in_time` |
| M14 | Snapshot, Replay & Longitudinal | `test_model_validation_m11_m15.py` | `test_m14_t01..t07` (7 tests) | `replay.fuse_point_in_time`, `replay.reconstruct_mastery_as_of`, `replay.prior_evidence` |
| M15 | Model Orchestration / End-to-End | `test_model_validation_m11_m15.py` | `test_m15_t01..t12` (12 tests) | Full pipeline: `bkt._bkt_update` → `irt._estimate_theta` → `fusion._mastery_status` → `root_cause.compute_rcs` → `replay.fuse_point_in_time` → `metrics.*` → `refit_worker._classify_outcome` |
| Cross-model audit | BKT/DINA/IRT/GCSF/SKSG/Refit/Recovery/RCS/Metrics exhaustive | `test_model_cross_validation.py` | 94 tests (92 pass + 2 skip for MIRT/DKT N/A) — gap-filling cross-check covering every previously-untested pure function across all model modules | `bkt._bkt_update`, `bkt._uncertainty_for`, `dina._joint_dina_update` (1/2/3-skill, degenerate), `irt.p_correct`, `irt._estimate_theta` (all edge cases), `fusion._recency_weight`, `fusion._source_weight`, `fusion._mastery_status`, `fusion._trend`, `refit_worker._clip_prob`, `refit_worker._bkt_sequence_log_likelihood`, `refit_worker._dina_log_likelihood`, `refit_worker._irt_log_likelihood`, `refit_worker._grid_search_bkt`, `refit_worker._grid_search_dina`, `refit_worker._grid_search_irt`, `recovery.is_blocked`, `recovery.rank_routes`, `root_cause.compute_rcs`, `root_cause.score_candidates`, `metrics.auc`, `metrics.log_loss`, `metrics.brier_score`, `metrics.expected_calibration_error` |

---

## Acceptance Principle Verification

The spec's primary acceptance principle:
> **"The system must prefer explicit uncertainty, abstention, degradation, or a clearly surfaced failure over fabricated certainty."**

| Principle | Verified By |
|---|---|
| No fabricated mastery when evidence is absent | FI-001, FI-005, COLD-001 (`upsert_calls == []`) |
| Disagreement widens uncertainty, not averaged away | FI-002 (`uncertainty > 0.1` for disagreeing sources) |
| Stale evidence is down-weighted | FI-003, REPLAY-001 (`result < 0.5` when fresh evidence is weak) |
| Invalid/missing graph data → documented default, not fabricated score | FI-004 (`readiness == 1.0`), FI-008 (`prerequisitesSatisfied is None`) |
| Graph-service outage degrades to PG mirror | FI-008 (`pg_fallback_called is True`) |
| Engine failure aborts job; worker loop contains exception | FI-007 (`fusion_called is False`; `run_forever` returns normally) |
| Inconsistency detection never mutates learner state | CONS-001 (`upsert_called is False`) |
| Sparse evidence suppresses false-positive consistency flags | CONS-001 (3 sparse-evidence guard tests) |
| Replay is strictly forward-only (no future-evidence leakage) | REPLAY-001 (`created_at < before` strict boundary) |
| Replay is deterministic | REPLAY-001 (`result1 == result2`) |
| Cold start → no fabricated historical mastery | COLD-001 (`result is None` at every layer) |
| Snapshot suppressed when mastery is NULL | COLD-001 (`take_snapshot` returns None for null/absent rows) |

---

## Production Code Hardened by This QA Pass

The following production files were modified to satisfy FI-008's graceful-degradation requirement — the tests exercise real production behaviour, not mocked error paths:

| File | Change |
|---|---|
| `ai-service/app/services/knowledge_tracing/fusion.py` | `_compute_structural_status`: Neo4j call wrapped in `try/except`; falls back to `pg_gap.fetch_prerequisite_chain_pg` |
| `ai-service/app/services/gap_analysis/consistency.py` | `check_topic`: Neo4j call wrapped in `try/except`; falls back to `db.fetch_prerequisite_chain_pg` |
| `ai-service/app/services/gap_analysis/root_cause.py` | All 3 Neo4j calls (`downstream_impact`, `_prerequisite_readiness`, `produce_graph_evidence`) wrapped; each falls back to its PG mirror |
| `ai-service/app/services/study_plan/service.py` | `_topological_order`: `neo4j_db.fetch_prerequisite_edges_among` wrapped; falls back to `db.fetch_prereq_edges_pg` |

---

## Known Limitations (Documented, Not Suppressed)

- **FI-006 dedup semantics**: The test asserts both physical duplicate rows are marked consumed (deduplication is upstream, at ingestion via the `answer_id` UNIQUE index on `students.learning_events`). The fusion engine itself does not deduplicate — it processes whatever `fetch_unconsumed_evidence` returns.
- **REPLAY-001 policy tolerance**: The `test_replay001_policy_weight_change_shifts_estimate_within_tolerance` bound (0.20 absolute) was chosen for evidence 5–10 days old against a half-life change from 30→5 days. It is not a universal tolerance; evidence closer to the cutoff will show less sensitivity.
- **COLD-001 does not exercise a live Docker stack**: All tests use monkeypatched DB calls. End-to-end Docker/Playwright verification of the cold-start path is the remaining unverified item for this test case (consistent with the broader EG-GCKT known gap documented in CLAUDE.md).
- **QA-001 is a documentation artifact**: No automated test runs this file. It is a point-in-time snapshot (2026-08-15) of which spec requirements have been verified and how. It should be updated whenever new test cases are added or production code is changed.
