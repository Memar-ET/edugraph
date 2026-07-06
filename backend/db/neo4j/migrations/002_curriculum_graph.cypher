// Replace CurriculumUnit with Topic
CREATE CONSTRAINT topic_id_unique IF NOT EXISTS
FOR (t:Topic) REQUIRE t.id IS UNIQUE;

CREATE CONSTRAINT subject_code_unique IF NOT EXISTS
FOR (s:Subject) REQUIRE s.code IS UNIQUE;

CREATE CONSTRAINT clo_code_unique IF NOT EXISTS
FOR (c:CLO) REQUIRE c.code IS UNIQUE;

// Indexes
CREATE INDEX topic_grade_subject_lookup IF NOT EXISTS
FOR (t:Topic) ON (t.gradeLevel, t.subjectCode);

CREATE INDEX clo_mandatory_lookup IF NOT EXISTS
FOR (c:CLO) ON (c.mandatory, c.gradeLevel);