// 002_curriculum_graph.cypher
// Complete curriculum knowledge graph schema
// Mirrors PostgreSQL curriculum.*, assessment.*, students.* schemas

// =====================================================
// 1. CLEANUP: Drop old constraints and indexes
// =====================================================
DROP CONSTRAINT unit_id_unique IF EXISTS;
DROP CONSTRAINT subject_id_unique IF EXISTS;
DROP CONSTRAINT career_id_unique IF EXISTS;
DROP CONSTRAINT student_id_unique IF EXISTS;
DROP INDEX unit_subject_lookup IF EXISTS;
DROP INDEX career_title_lookup IF EXISTS;

// =====================================================
// 2. UNIQUENESS CONSTRAINTS (Core Nodes)
// =====================================================

// Subject identified by code (e.g., PHY10, MATH10)
CREATE CONSTRAINT subject_code_unique IF NOT EXISTS
FOR (s:Subject) REQUIRE s.code IS UNIQUE;

// Unit identified by UUID
CREATE CONSTRAINT unit_id_unique IF NOT EXISTS
FOR (u:Unit) REQUIRE u.id IS UNIQUE;

// Topic identified by UUID (most important node)
CREATE CONSTRAINT topic_id_unique IF NOT EXISTS
FOR (t:Topic) REQUIRE t.id IS UNIQUE;

// CLO identified by official Ministry code
CREATE CONSTRAINT clo_code_unique IF NOT EXISTS
FOR (c:CLO) REQUIRE c.code IS UNIQUE;

// Question identified by UUID
CREATE CONSTRAINT question_id_unique IF NOT EXISTS
FOR (q:Question) REQUIRE q.id IS UNIQUE;

// Exam identified by UUID
CREATE CONSTRAINT exam_id_unique IF NOT EXISTS
FOR (e:Exam) REQUIRE e.id IS UNIQUE;

// Student identified by UUID
CREATE CONSTRAINT student_id_unique IF NOT EXISTS
FOR (s:Student) REQUIRE s.id IS UNIQUE;

// School identified by UUID
CREATE CONSTRAINT school_id_unique IF NOT EXISTS
FOR (s:School) REQUIRE s.id IS UNIQUE;

// Career identified by UUID
CREATE CONSTRAINT career_id_unique IF NOT EXISTS
FOR (c:Career) REQUIRE c.id IS UNIQUE;

// Region identified by UUID
CREATE CONSTRAINT region_id_unique IF NOT EXISTS
FOR (r:Region) REQUIRE r.id IS UNIQUE;

// =====================================================
// 3. PERFORMANCE INDEXES (Query Optimization)
// =====================================================

// Topic lookups
CREATE INDEX topic_grade_subject_lookup IF NOT EXISTS
FOR (t:Topic) ON (t.gradeLevel, t.subjectCode);

CREATE INDEX topic_bloom_lookup IF NOT EXISTS
FOR (t:Topic) ON (t.bloomLevel);

// CLO lookups
CREATE INDEX clo_grade_subject_lookup IF NOT EXISTS
FOR (c:CLO) ON (c.gradeLevel, c.subjectCode);

CREATE INDEX clo_mandatory_lookup IF NOT EXISTS
FOR (c:CLO) ON (c.mandatory, c.gradeLevel);

// Unit lookups
CREATE INDEX unit_subject_lookup IF NOT EXISTS
FOR (u:Unit) ON (u.subjectCode);

// Student lookups
CREATE INDEX student_school_lookup IF NOT EXISTS
FOR (s:Student) ON (s.schoolId);

CREATE INDEX student_grade_lookup IF NOT EXISTS
FOR (s:Student) ON (s.gradeLevel);

// School lookups
CREATE INDEX school_region_lookup IF NOT EXISTS
FOR (s:School) ON (s.regionId);

// Question lookups
CREATE INDEX question_topic_lookup IF NOT EXISTS
FOR (q:Question) ON (q.topicId);

CREATE INDEX question_clo_lookup IF NOT EXISTS
FOR (q:Question) ON (q.cloCode);

// Exam lookups
CREATE INDEX exam_subject_grade_lookup IF NOT EXISTS
FOR (e:Exam) ON (e.subjectCode, e.gradeLevel);

// Career lookups
CREATE INDEX career_sector_lookup IF NOT EXISTS
FOR (c:Career) ON (c.sector);

// Relationship indexes (for traversal optimization)
CREATE INDEX topic_prerequisite_grade IF NOT EXISTS
FOR (t:Topic) ON (t.gradeLevel);

// =====================================================
// 4. EXAMPLE RELATIONSHIP CREATION (Documentation)
// These are examples - actual relationships created by sync worker
// =====================================================

// Subject -> Unit -> Topic hierarchy
// (s:Subject)-[:HAS_UNIT]->(u:Unit)
// (u:Unit)-[:HAS_TOPIC]->(t:Topic)

// Topic -> CLO mapping
// (t:Topic)-[:MAPS_TO_CLO {alignmentScore, matchMethod, confirmedAt}]->(c:CLO)

// Topic prerequisites (cross-grade!)
// (t1:Topic)-[:PREREQUISITE_OF {weight, isCrossGrade, inferMethod}]->(t2:Topic)

// Question -> CLO assessment
// (q:Question)-[:ASSESSES {alignmentScore, alignMethod}]->(c:CLO)

// Question -> Exam membership
// (q:Question)-[:PART_OF]->(e:Exam)

// Student -> School enrollment
// (s:Student)-[:ATTENDS {enrolledAt}]->(sch:School)

// School -> Region membership
// (sch:School)-[:IN_REGION]->(r:Region)

// Student exam attempts
// (s:Student)-[:ATTEMPTED {totalScore, percentage, attemptDate}]->(e:Exam)

// Student answers
// (s:Student)-[:ANSWERED {marksAwarded, marksPossible, passed, timeSpentSecs}]->(q:Question)

// Student mastery/gaps
// (s:Student)-[:MASTERED {confidence, masteredAt}]->(t:Topic)
// (s:Student)-[:STRUGGLED_WITH {severity, isRootCause, prereqDepth, detectedAt}]->(t:Topic)

// Career requirements
// (c:Career)-[:REQUIRES {importance}]->(t:Topic)