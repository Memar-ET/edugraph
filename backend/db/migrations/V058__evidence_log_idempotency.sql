-- V058: modeling.evidence_log idempotency (EG-GCKT production-hardening
-- pass). kt_worker.py's pipeline (ai-service) was not idempotent: if a
-- knowledge-tracing job for one attempt_id failed partway through and
-- got dead-lettered/requeued (see app/db/dead_letter.py,
-- refit_worker.run_once), re-processing the same attempt_id re-derived
-- and re-inserted duplicate evidence_log rows for bkt/dina/irt (each
-- always passes source_event_id = the originating learning_events.id,
-- which is itself stable across re-processing since
-- RecordLearningEvents's own ON CONFLICT (answer_id) DO UPDATE never
-- changes a row's id). Those duplicates got folded into skill_states a
-- second time by fusion.fuse_skill_state, inflating evidence_count/
-- sample_size and artificially sharpening uncertainty.
--
-- source_event_id is nullable -- root_cause.produce_graph_evidence
-- (provenance='graph_reasoning') legitimately calls insert_evidence
-- without one, since that evidence isn't tied to one specific answer
-- event and can recur multiple times for the same (student, topic) with
-- no natural per-event dedup key. The partial index below only
-- constrains the source_event_id-bearing rows (bkt/dina/irt), which is
-- exactly where the duplication risk was.
CREATE UNIQUE INDEX IF NOT EXISTS idx_evidence_log_source_provenance
  ON modeling.evidence_log (source_event_id, provenance)
  WHERE source_event_id IS NOT NULL;
