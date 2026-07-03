// CurriculumUnit nodes mirror backend.curriculum_units (id keeps both stores in sync).
// (:CurriculumUnit)-[:PREREQUISITE_OF]->(:CurriculumUnit) models the dependency graph
// used by the gap-analysis engine to find missing prerequisite knowledge.
// (:Subject)-[:HAS_UNIT]->(:CurriculumUnit) groups units under their subject.

CREATE INDEX unit_subject_lookup IF NOT EXISTS
FOR (u:CurriculumUnit) ON (u.subjectId);
