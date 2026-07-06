// 005_school_region_graph.cypher
// School and regional organization graph
// Mirrors PostgreSQL schools.* and regions.* schemas

// =====================================================
// 1. SCHOOL/REGION CONSTRAINTS (already in 002)
// =====================================================
// School and Region constraints already created

// =====================================================
// 2. SCHOOL/REGION INDEXES
// =====================================================

// School type and connectivity
CREATE INDEX school_type_lookup IF NOT EXISTS
FOR (s:School) ON (s.schoolType, s.connectivity);

// Region code lookup
CREATE INDEX region_code_lookup IF NOT EXISTS
FOR (r:Region) ON (r.code);

// =====================================================
// 3. SCHOOL/REGION RELATIONSHIPS (Documentation)
// =====================================================

// Organizational hierarchy:
// (s:Student)-[:ATTENDS {enrolledAt}]->(sch:School)
// (sch:School)-[:IN_REGION]->(r:Region)

// School quality tracking:
// (sch:School)-[:HAS_QUALITY_SCORE {overallScore, academicYear, term}]->(qs:QualityScore)

// =====================================================
// 4. AGGREGATION QUERIES (Example)
// =====================================================

// Class-wide gap heatmap:
/*
MATCH (school:School {id: $schoolId})
      -[:ENROLLS]->(s:Student)
      -[r:STRUGGLED_WITH]->(t:Topic)
WHERE t.subjectCode = $subjectCode AND t.gradeLevel = $gradeLevel
RETURN t.title AS topic,
       COUNT(DISTINCT s) AS studentsStruggling,
       AVG(r.severity) AS avgSeverity
ORDER BY studentsStruggling DESC
LIMIT 15
*/

// Regional root cause analysis:
/*
MATCH (region:Region {id: $regionId})
      <-[:IN_REGION]-(school:School)
      -[:ENROLLS]->(s:Student)
      -[r:STRUGGLED_WITH {isRootCause: true}]->(t:Topic)
RETURN t.title AS rootCauseTopic,
       t.gradeLevel AS grade,
       COUNT(DISTINCT s) AS affectedStudents
ORDER BY affectedStudents DESC
LIMIT 20
*/