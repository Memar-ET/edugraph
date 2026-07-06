// 004_assessment_graph.cypher
// Assessment and student performance graph
// Mirrors PostgreSQL assessment.* and students.* schemas

// =====================================================
// 1. ASSESSMENT CONSTRAINTS (already in 002)
// =====================================================
// Question and Exam constraints already created

// =====================================================
// 2. ASSESSMENT INDEXES
// =====================================================

// Exam attempts by student
CREATE INDEX attempt_student_lookup IF NOT EXISTS
FOR (a:ExamAttempt) ON (a.studentId);

// Answer performance tracking
CREATE INDEX answer_passed_lookup IF NOT EXISTS
FOR (a:Answer) ON (a.passed);

// =====================================================
// 3. ASSESSMENT RELATIONSHIPS (Documentation)
// =====================================================

// Exam structure:
// (e:Exam)-[:FOR_SUBJECT]->(s:Subject)
// (q:Question)-[:PART_OF]->(e:Exam)
// (q:Question)-[:ASSESSES {alignmentScore}]->(c:CLO)

// Student performance:
// (s:Student)-[:ATTEMPTED {totalScore, percentage, attemptDate}]->(e:Exam)
// (s:Student)-[:ANSWERED {marksAwarded, passed, timeSpentSecs}]->(q:Question)

// Learning outcomes:
// (s:Student)-[:MASTERED {confidence, masteredAt}]->(t:Topic)
// (s:Student)-[:STRUGGLED_WITH {severity, isRootCause, prereqDepth}]->(t:Topic)

// =====================================================
// 4. GAP ANALYSIS TRAVERSAL (Example)
// =====================================================

// Root cause gap analysis:
/*
MATCH (s:Student {id: $studentId})
      -[:ANSWERED {passed: false}]->(q:Question)
      -[:ASSESSES]->(clo:CLO)
     <-[:MAPS_TO_CLO]-(symptomTopic:Topic)
MATCH path = (root:Topic)-[:PREREQUISITE_OF*1..5]->(symptomTopic)
WHERE NOT (s)-[:MASTERED]->(root)
RETURN symptomTopic.title AS symptom,
       root.title AS rootCause,
       root.gradeLevel AS rootGrade,
       LENGTH(path) AS depth
ORDER BY depth DESC
*/