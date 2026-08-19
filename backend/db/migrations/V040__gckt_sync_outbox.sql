-- EG-GCKT Milestone 0: extend the existing School-Box offline outbox
-- (V029) to students.learning_events and students.skill_states.
--
-- Both tables are written locally at a school -- learning_events
-- synchronously by Go at exam-submit (which happens at schools, including
-- offline School Boxes), skill_states by an AI worker that also runs
-- locally, same as the existing gap-analysis/study-plan workers V029
-- already covers. Without these triggers, offline-first schools would
-- silently lose all EG-GCKT data on every sync -- the same class of gap
-- V029 was built to close for gap_records/mastery_records/study_plans.
--
-- item_skill_mappings/evidence_log/model_snapshots/*_review_history do
-- NOT get outbox triggers: they're ministry/AI-authored and flow
-- cloud->school like curriculum content already does (V029's documented
-- scope note), never written locally at a school.

CREATE TRIGGER trg_outbox_learning_events
    AFTER INSERT OR UPDATE OR DELETE ON students.learning_events
    FOR EACH ROW EXECUTE FUNCTION sync.record_outbox_change('students.learning_events');

CREATE TRIGGER trg_outbox_skill_states
    AFTER INSERT OR UPDATE OR DELETE ON students.skill_states
    FOR EACH ROW EXECUTE FUNCTION sync.record_outbox_change('students.skill_states');
