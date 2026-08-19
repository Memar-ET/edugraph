-- EG-GCKT checklist follow-up (section 10): the Root Cause Score was
-- being computed in ai-service's root_cause.py and then discarded --
-- only the WINNING topic_id survived into students.gap_records, with no
-- way to see why it won or which chain nodes were considered. This adds
-- storage for the score itself, its five component factors (Weakness,
-- EvidenceConfidence, DownstreamImpact, PrerequisiteReadiness,
-- InterventionGain), and the graph path from symptom to root cause --
-- "expose the evidence behind the root-cause score" / "expose the graph
-- path connecting root cause to downstream weaknesses."

ALTER TABLE students.gap_records
    ADD COLUMN rcs_score NUMERIC,
    ADD COLUMN rcs_factors JSONB,
    ADD COLUMN root_cause_path JSONB;
