<!-- converted from edugraph-impl-plan.docx -->






EduGraph AI
Backend Implementation Plan
Phase-Based Roadmap — Curriculum Intelligence to National Analytics

Version 1.0  ·  June 2026  ·  EduGraph AI Engineering







# Table of Contents

# 1. Implementation Philosophy
Every downstream feature — exam validation, gap analysis, study plans, the AI tutor — depends on the Curriculum Intelligence Graph. Build the foundation before building the house.
The most important architectural insight in EduGraph AI is that the system is not a set of independent features — it is a dependency graph of data. A student's gap analysis cannot work unless topics have been extracted from the curriculum. A study plan cannot be ordered unless prerequisite relationships exist between topics. Exam validation cannot check CLO coverage unless each question has been matched to a CLO. The AI assistant cannot explain a concept unless the concept exists as a node in the knowledge graph.
This means the implementation order is not determined by what stakeholders want first — it is determined by what data must exist before any downstream operation is possible.

## 1.1 The Fundamental Dependency Chain
The chain below shows what must exist before each subsequent step can function. This is the primary driver of phase ordering throughout this document.


## 1.2 Why Cross-Grade Relationships Are Non-Negotiable from Phase 1
You specifically asked about this — and it is the insight that separates a basic EdTech tool from a genuine intelligence system. Consider this scenario:
- A Grade 10 student fails 'Newton's Second Law' on their exam.
- Naively, the gap analysis says: 'Study Newton's Second Law.'
- But Newton's Second Law requires understanding 'Net Force', which requires 'Vector Addition', which was a Grade 9 topic.
- If the student never mastered Vector Addition in Grade 9, telling them to re-study Newton's Second Law will not help.
- The real intervention is: 'Go back and master Grade 9 Vector Addition first.'
This is only possible if the prerequisite graph spans grades. The curriculum graph must know that Grade 10 Topic X has Grade 9 Topic Y as a prerequisite. This cross-grade linking must be built in Phase 1 — not added later — because every phase after it depends on correct traversal depth.

## 1.3 Interdependency Summary Map
The table below shows which Phase 1 outputs each subsequent phase consumes. A blank cell means no dependency; a dot means 'required before this phase can function'.



ℹ  Duration: 8–10 weeks  ·  Team: 2 backend engineers + 1 AI/NLP engineer + 1 curriculum domain expert

Phase 1 is not 'the first phase' — it is the only phase that must be completely correct before any other work begins. Every feature in every other phase traverses the graph built here.
The entire value of EduGraph AI rests on a correctly structured curriculum knowledge graph. If a topic is missing, a CLO is mismatched, or a prerequisite link is wrong, every downstream system — gap analysis, study plans, exam validation, the AI tutor — will produce wrong results. Phase 1 must be treated as foundational infrastructure, not as a feature.

## 1.1 Capability 1A — Curriculum Document Ingestion Pipeline
When a curriculum officer or teacher uploads a Grade 10 Physics book (PDF or DOCX), the system must break it down into its structural components: units, chapters, topics, and subtopics. This is not trivial — Ethiopian curriculum books follow a consistent format, but the parser must handle variability.
### What the parser must extract
- Unit number and title (e.g., Unit 3: Dynamics)
- Chapter number and title within each unit
- Topic titles and their position within the chapter hierarchy
- Subtopics (if explicitly listed in the table of contents or section headers)
- Learning outcome statements (if the book includes them — not all do)
- Key vocabulary terms per topic (used for embedding and CLO matching)
- Example problems and their topic tags (used for question difficulty calibration)
### Technical implementation
- Primary tool: PyMuPDF (fitz) for PDF structure extraction. Apache POI via Python bridge for DOCX.
- Heading hierarchy detection: parse font size + bold flags to infer Unit/Chapter/Topic levels.
- Table of contents extraction: if a TOC exists, parse it first as the authoritative structure — do not rely solely on body text parsing.
- Fallback: if structural detection fails, send the full text to the AI service (Claude/Qwen) with a structured extraction prompt that returns a JSON tree.
- Human review queue: all parsed structures go into a review queue before being written to Neo4j. A curriculum officer must confirm the hierarchy before it becomes authoritative.
### API endpoints (Go Core API)
POST /api/v1/curriculum/upload
Body: multipart/form-data {file, gradeLevel, subject, academicYear}
→ Queues parsing job, returns jobId

GET  /api/v1/curriculum/jobs/{jobId}
→ Returns parse status, preview of extracted structure

POST /api/v1/curriculum/jobs/{jobId}/approve
→ Curriculum officer confirms structure, triggers Neo4j write

GET  /api/v1/curriculum/subjects/{subjectCode}/structure
→ Returns full unit/chapter/topic tree for a subject+grade
### PostgreSQL schema (curriculum.*)
curriculum.upload_jobs    (id, subject_code, grade_level, status, file_s3_key, created_at)
curriculum.parse_results  (job_id, raw_structure_json, reviewed_by, approved_at)
curriculum.subjects       (code, name, grade_level, academic_year)
curriculum.units          (id, subject_code, number, title)
curriculum.topics         (id, unit_id, title, sequence_order, estimated_hours)

ℹ  → DEPENDS ON: Nothing. This is the root of the entire system.
ℹ  ✓ DELIVERABLE: Parsed curriculum structure written to PostgreSQL and Neo4j. Human-reviewed and approved.

## 1.2 Capability 1B — Neo4j Curriculum Graph Construction
Once the curriculum structure is approved, it is written to Neo4j as a traversable graph. This is not just storage — the graph relationships are the intelligence. The topology of the graph determines what gap analysis finds, what study plans recommend, and what the AI tutor knows about concept dependencies.
### Node types created in this phase
### Relationships created in this phase
### Cypher — writing a topic node
MERGE (t:Topic {id: $topicId})
ON CREATE SET
t.title          = $title,
t.gradeLevel     = $gradeLevel,
t.subjectCode    = $subjectCode,
t.estimatedHours = $estimatedHours,
t.createdAt      = datetime()
WITH t
MATCH (u:Unit {id: $unitId})
MERGE (u)-[:HAS_TOPIC]->(t)
ℹ  → DEPENDS ON: Capability 1A — approved parsed curriculum structure must exist.
ℹ  ✓ DELIVERABLE: All subjects, units, topics, subtopics written as Neo4j nodes with structural relationships.

## 1.3 Capability 1C — CLO Matching and Verification
This is the most technically sensitive step in Phase 1. Every topic must be linked to its Curriculum Learning Outcome(s) from the official Ministry of Education CLO document. There are three sources of CLOs: the MOE national CLO document (authoritative), the curriculum book's own stated learning objectives (secondary), and AI-inferred CLOs for topics with no explicit match (tertiary, human-reviewed).
### CLO ingestion pipeline
- Ministry uploads the national CLO Excel/CSV. Each row: subjectCode, gradeLevel, cloCode, description, bloomLevel, mandatory.
- MERGE operation writes CLO nodes — idempotent, safe to re-upload when MOE updates the document.
- Each CLO description is embedded using multilingual-e5-large (supports Amharic + English).
- Each topic title + key concepts are also embedded.
- pgvector cosine similarity search: for each topic, find top-5 CLO candidates by embedding similarity.
- Matches above 0.85 similarity are auto-accepted.
- Matches between 0.65–0.85 go to a human review queue for curriculum officer confirmation.
- Topics with no CLO match above 0.65 are flagged: AI generates a draft CLO for human review.
### API endpoints
POST /api/v1/curriculum/clos/upload
→ Ingest MOE CLO document, embed all CLOs, store in PostgreSQL + pgvector

POST /api/v1/curriculum/clos/match-topics
→ Trigger embedding-based matching for all topics in a subject
→ Returns match report: auto-accepted, needs-review, unmatched

GET  /api/v1/curriculum/clos/review-queue
→ Curriculum officer reviews pending matches

POST /api/v1/curriculum/clos/review/{matchId}/approve
POST /api/v1/curriculum/clos/review/{matchId}/reject
→ Human decision, triggers Neo4j MERGE of (Topic)-[:MAPS_TO_CLO]->(CLO)
### The Neo4j relationship written
MATCH (t:Topic {id: $topicId}), (c:CLO {code: $cloCode})
MERGE (t)-[:MAPS_TO_CLO {
alignmentScore: $score,
matchMethod:    $method,   // 'embedding' | 'manual' | 'ai_draft'
reviewedBy:     $userId,
confirmedAt:    datetime()
}]->(c)
ℹ  → DEPENDS ON: 1A (topics exist in Neo4j) + CLO document from Ministry.
ℹ  ✓ DELIVERABLE: All topics linked to CLOs. No topic is matchless. Human review complete.

## 1.4 Capability 1D — Cross-Grade Prerequisite Graph
This is the capability that transforms EduGraph from a course management system into an intelligence system. The prerequisite graph must span grades — a Grade 10 topic can (and usually does) depend on Grade 9 topics.
### Three sources of prerequisite relationships
- Curriculum book cross-references: Ethiopian textbooks often contain 'Recall from Grade 9' sections. The parser extracts these as explicit prerequisite signals.
- MOE prerequisite document: the Ministry may provide an explicit topic dependency map. If available, this is the authoritative source.
- AI-inferred prerequisites: for topics without explicit cross-references, the AI service (using the topic title, CLO description, and key concepts) infers likely prerequisites from the same and adjacent grade levels. All AI-inferred relationships require curriculum officer approval.
### The critical relationship
// A Grade 10 topic that requires a Grade 9 topic
MATCH (g10:Topic {id: $topicId, gradeLevel: 10})
MATCH (g9:Topic  {id: $prereqId, gradeLevel: 9})
MERGE (g9)-[:PREREQUISITE_OF {
weight:      $importanceScore,  // 0.0–1.0
inferredBy:  $method,           // 'explicit' | 'ai_inferred' | 'manual'
gradeSpan:   true               // cross-grade flag for traversal filtering
}]->(g10)
### Prerequisite quality validation
- Cycle detection: the prerequisite graph must be a DAG (directed acyclic graph). Neo4j GDS shortestPath detects any cycles and rejects them.
- Depth limit: prerequisite chains are limited to 5 grade levels back during ingestion (prevents absurd chains).
- Orphan check: every topic must have at least one incoming prerequisite (except Grade 1 topics) OR must be a root concept. Orphan topics are flagged for curriculum review.
ℹ  → DEPENDS ON: 1B (topic nodes in Neo4j) + curriculum books for at least 2 consecutive grades.
ℹ  ✓ DELIVERABLE: Full cross-grade prerequisite graph. Validated as DAG. AI-inferred links reviewed.

## 1.5 Phase 1 — Integration Test Suite
Before Phase 2 begins, the following integration tests must all pass. These are not optional:
- Upload Grade 9 Physics book → 100% of topics appear as nodes in Neo4j.
- Upload Grade 10 Physics book → all Grade 10 topics exist AND at least 40% have Grade 9 prerequisites.
- Upload MOE CLO document → all CLOs ingested, all Grade 10 topics have ≥1 CLO match.
- CLO cosine similarity test: a known question text returns its correct CLO as the top match.
- Cross-grade prerequisite traversal: MATCH path = (t:Topic {gradeLevel:9})-[:PREREQUISITE_OF*]->(t2:Topic {gradeLevel:11}) returns results.
- Cycle detection: artificially introduce a cycle, confirm ingest pipeline rejects it.
- Human review queue: confirm no auto-accepted CLO match exists with similarity below 0.85.
ℹ  Phase 1 is done only when all 7 tests pass. No exceptions.


ℹ  Duration: 6–8 weeks  ·  Team: 2 backend engineers + 1 AI/NLP engineer  ·  Starts: after Phase 1 integration tests pass

Phase 2 makes the curriculum graph useful to teachers. An exam is only as good as its alignment to the curriculum. Phase 2 validates every question before students ever see it.
## 2.1 Capability 2A — Exam Upload and Parsing
Teachers upload exam files (PDF or DOCX). The system parses individual questions, recognizes their subject and grade level, and prepares them for CLO alignment.
### Question extraction
- PyMuPDF + regex patterns extract numbered questions from standard Ethiopian exam formats.
- Each question stored as a Question node in PostgreSQL with: text, questionType (MCQ/short/essay), difficultyLevel, marks, subjectCode, gradeLevel.
- Question text embedded using multilingual-e5-large for CLO similarity matching.
### API endpoints
POST /api/v1/exams/upload
Body: {file, subjectCode, gradeLevel, term, totalMarks, dueDate}
→ Parse job queued, returns examId

GET  /api/v1/exams/{examId}/questions
→ Returns extracted questions with auto-detected CLO matches

POST /api/v1/exams/{examId}/publish
→ Locks exam, makes available to students, writes to Neo4j
### Neo4j writes on exam creation
MERGE (exam:Exam {id: $examId})
SET exam.subjectCode = $subjectCode, exam.gradeLevel = $gradeLevel

MATCH (q:Question {id: $questionId})
MERGE (q)-[:PART_OF]->(exam)

// CLO alignment (from similarity match)
MATCH (clo:CLO {code: $cloCode})
MERGE (q)-[:ASSESSES {alignmentScore: $score}]->(clo)
ℹ  → DEPENDS ON: Phase 1 complete — CLO nodes and topic nodes must exist for alignment to work.

## 2.2 Capability 2B — Exam Validation Report
This is the flagship feature of Phase 2. When a teacher uploads an exam, the system immediately validates it against the curriculum and returns a detailed report before any student sees the exam. This prevents the systemic problem of teachers writing exams that do not cover the required CLOs.
### What the validation report shows
- CLO coverage: which of the mandatory CLOs for this subject/grade are tested, which are missing.
- Bloom's taxonomy distribution: is the exam appropriately balanced (recall vs analysis vs synthesis)?
- Difficulty distribution: proportion of easy/medium/hard questions vs the recommended distribution.
- Topic coverage: which topics from the curriculum are represented, which are absent.
- Cross-grade prerequisite warning: 'Question 7 tests Electromagnetic Induction — this assumes Grade 9 Magnetic Flux is mastered. Verify students have covered this prerequisite.'
### The core validation Cypher query
// Find CLOs required by this subject/grade that the exam does NOT cover
MATCH (sub:Subject {code: $subjectCode, gradeLevel: $gradeLevel})
-[:HAS_UNIT]->()-[:HAS_TOPIC]->(t:Topic)
-[:MAPS_TO_CLO]->(clo:CLO {mandatory: true})
WHERE NOT (clo)<-[:ASSESSES]-(:Question)-[:PART_OF]->(:Exam {id: $examId})
RETURN clo.code AS missingCLO, clo.description AS description
ORDER BY clo.code
ℹ  → DEPENDS ON: 2A (exam + questions in Neo4j with CLO alignment) + Phase 1 complete.
ℹ  ✓ DELIVERABLE: Exam validation report API. Teacher can iterate on exam before publishing.

## 2.3 Capability 2C — Student Answer Ingestion
When a student submits an exam, each answer is recorded with the question, the marks awarded, and whether the student passed that question. This is the data that feeds all of Phase 3.
### Data model
// PostgreSQL — assessment.student_answers
student_id, question_id, exam_id, raw_answer_text,
marks_awarded, marks_possible, passed (bool),
time_spent_seconds, submitted_at

// Neo4j — written asynchronously after PostgreSQL commit
(s:Student)-[:ANSWERED {
marks:    $marksAwarded,
passed:   $passed,
timeSec:  $timeSpent
}]->(q:Question)

(s:Student)-[:ATTEMPTED {
totalScore:  $score,
attemptDate: datetime()
}]->(exam:Exam)
### Sync strategy
- Answers written to PostgreSQL first (ACID, transactional).
- PostgreSQL logical replication slot captures the insert event.
- Go sync worker consumes the replication event and writes to Neo4j asynchronously.
- If Neo4j write fails, it is retried via Bull MQ job queue. PostgreSQL is the source of truth.
ℹ  → DEPENDS ON: 2A (exams + questions exist) + Phase 1 (topic nodes exist for future traversal).
ℹ  ✓ DELIVERABLE: Student answers stored. Neo4j [:ANSWERED] edges created. Phase 3 can now read these.

## 2.4 Phase 2 — Summary of Neo4j State After Completion
After Phase 2, the graph has grown from curriculum structure to a full assessment graph:


ℹ  Duration: 8–10 weeks  ·  Team: 2 backend engineers + 2 AI engineers  ·  Starts: after Phase 2 integration tests pass

Phase 3 is where the investment in Phase 1 pays off. Without the cross-grade prerequisite graph, gap analysis returns symptom topics. With it, gap analysis returns root causes.
## 3.1 Capability 3A — Gap Analysis Engine
Gap analysis is the core intelligence of the student-facing system. It must answer two questions: (1) which topics did this student fail, and (2) of those, which failures are symptoms of deeper, root-cause gaps in earlier grades?
### The two-pass algorithm
- Pass 1 — Surface gaps: collect all questions the student answered with passed=false. Traverse [:ASSESSES] to CLOs, traverse [:MAPS_TO_CLO] backwards to topics. This gives the symptom-level gap list.
- Pass 2 — Root cause traversal: for each symptom topic, traverse [:PREREQUISITE_OF*1..5] upstream. Find the deepest ancestor topic that the student has NOT mastered. That is the root cause gap.
### Cypher — root cause gap traversal
// Get topics the student failed
MATCH (s:Student {id: $studentId})
-[:ANSWERED {passed: false}]->(q:Question)
-[:ASSESSES]->(clo:CLO)
<-[:MAPS_TO_CLO]-(failedTopic:Topic)

// For each failed topic, walk prerequisites upstream
MATCH path = (root:Topic)-[:PREREQUISITE_OF*1..5]->(failedTopic)
WHERE NOT (s)-[:MASTERED]->(root)
AND NOT (s)-[:ANSWERED {passed: true}]->
(:Question)-[:ASSESSES]->(:CLO)<-[:MAPS_TO_CLO]-(root)

RETURN failedTopic.title    AS symptomTopic,
root.title           AS rootCause,
root.gradeLevel      AS rootGrade,
LENGTH(path)         AS prerequisiteDepth,
clo.description      AS whatWasTested
ORDER BY prerequisiteDepth DESC
### Gap severity scoring
- Severity = (marks lost / total marks for that CLO) × (prerequisite depth / 5) × (mandatory CLO weight).
- Severity stored as a property on the [:STRUGGLED_WITH] edge created after analysis.
- Topics with severity > 0.7 are flagged as critical — these appear first in the study plan.
### Gap result storage
// Written to PostgreSQL for persistence
students.gap_records (
student_id, topic_id, clo_code,
severity_score, is_root_cause, prerequisite_depth,
detected_at, exam_id)

// Written to Neo4j for future traversal
MERGE (s)-[:STRUGGLED_WITH {
severity:         $score,
isRootCause:      $isRoot,
prereqDepth:      $depth,
detectedAt:       datetime()
}]->(topic)
ℹ  → DEPENDS ON: Phase 1 (CLO graph + cross-grade prerequisites) + Phase 2 (student answers).
ℹ  ✓ DELIVERABLE: Gap analysis API. Every student has a gap list with root causes identified.

## 3.2 Capability 3B — Study Plan Generator
A study plan without the prerequisite graph is a list of topics. A study plan with the prerequisite graph is an ordered learning path that ensures no topic is studied before its foundations are in place.
### Algorithm
- Input: student's gap list from 3A (root cause gaps ranked by severity).
- Step 1 — Collect all topics to study: the failed topics + all their unmastered prerequisites.
- Step 2 — Topological sort using Neo4j GDS: gds.dag.topologicalSort on the prerequisite subgraph for this student's gap set. Prerequisites always come before the topics they enable.
- Step 3 — Prioritization: within the same prerequisite tier, topics are sorted by (upcoming exam date × CLO mandatory flag × severity score).
- Step 4 — Time estimation: each topic has an estimatedHours property. The plan is assembled day-by-day respecting the student's declared daily availability.
- Step 5 — LLM enrichment: the AI service (Qwen-7B) generates a plain-language description for each day's study goal. 'Today: master Grade 9 Vector Addition — this unlocks Newton's Second Law which appears on your exam in 12 days.'
### API endpoint
POST /api/v1/students/{studentId}/study-plan
Body: {
examDate:          '2026-07-15',
dailyHoursAvail:   2.5,
languagePref:      'am',  // Amharic
includePrereqs:    true   // include cross-grade prerequisite topics
}

Response: {
totalDays:   14,
days: [
{ date: '2026-07-01', topics: [{id, title, grade, hours, why}] },
...
],
rootCauseWarnings: ['Grade 9 Vector Addition critical — study first']
}
ℹ  → DEPENDS ON: 3A (gap analysis complete) + Phase 1 (prerequisite graph + estimatedHours on topics).
ℹ  ✓ DELIVERABLE: Study plan API. Plans are topologically correct — prerequisites always precede dependents.

## 3.3 Capability 3C — AI Tutor / Academic Assistant
The AI tutor uses the curriculum graph as its knowledge base. It can explain any topic in the student's curriculum, answer questions about specific CLOs, and generate practice problems aligned to the student's current gaps.
### How the knowledge graph feeds the tutor
- When a student asks 'Explain Newton's Second Law', the AI service does NOT rely purely on the LLM's training data.
- It first queries Neo4j: MATCH (t:Topic {title: 'Newton's Second Law'})-[:HAS_CONCEPT]->(c:KeyConcept) to retrieve the exact concepts, definitions, and CLO description from the curriculum graph.
- It also queries: MATCH (prereq:Topic)-[:PREREQUISITE_OF]->(t) to know what the student should already understand.
- This context is injected into the LLM prompt: 'You are an Ethiopian Grade 10 Physics tutor. The following concepts are from the official curriculum: [context]. The student's known prerequisite gaps are: [gap list]. Explain Newton's Second Law in Amharic at their level.'
- The response is curriculum-grounded, not generic — it uses the exact CLO language the exam will test.
### Practice problem generation
POST /ai/tutor/practice-problem
Body: {
topicId:     'TOPIC-PHY10-FORCE-02',
cloCode:     'CLO-PHY10-3.2',
difficulty:  'medium',
language:    'am'
}

→ LLM generates a problem aligned to the exact CLO
→ Answer + explanation also generated
→ Problem stored for teacher review before serving to students
ℹ  → DEPENDS ON: Phase 1 (topic concepts + CLO descriptions in Neo4j) + 3A (gap list for context).
ℹ  ✓ DELIVERABLE: AI tutor API. All responses are curriculum-grounded via graph context injection.

## 3.4 Capability 3D — Career Recommendation Engine
Career nodes are linked to required Topic nodes via [:REQUIRES] relationships. The engine scores each career by the student's current mastery level and recommends the most achievable careers while showing what topics the student needs to study to unlock others.
### Career graph setup (prerequisite for this feature)
- Career nodes created manually by curriculum officers (one-time setup, not per upload).
- Career-to-topic requirements defined: 'Electrical Engineering requires Grade 10 Electromagnetic Induction, Grade 11 Circuit Analysis...'
- This is a seed dataset — Ethiopian university admission requirements are a known document.
### Readiness scoring query
MATCH (career:Career)-[:REQUIRES]->(required:Topic)
OPTIONAL MATCH (s:Student {id: $studentId})
-[:MASTERED]->(required)
WITH career,
COUNT(required)   AS totalRequired,
COUNT(s)          AS alreadyMastered
RETURN career.name,
ROUND(alreadyMastered * 100.0 / totalRequired) AS readinessPct
ORDER BY readinessPct DESC LIMIT 5
ℹ  → DEPENDS ON: Phase 1 (topic graph) + 3A (mastered/struggled edges) + career seed data.
ℹ  ✓ DELIVERABLE: Career recommendation API with readiness scores and gap-to-career focused study plans.


ℹ  Duration: 6–8 weeks  ·  Team: 2 backend engineers  ·  Starts: after Phase 3 core (3A + 3B) is stable in production

Phase 4 aggregates student-level intelligence upward to the class and school level. Teachers and administrators can now see patterns that are invisible when looking at individual students.
## 4.1 Capability 4A — Class-Wide Gap Heatmap
The teacher dashboard shows which topics the entire class is struggling with — not just one student. This allows the teacher to identify systematic curriculum delivery problems (not just individual student weaknesses) and prioritize reteaching.
### Heatmap query
// Topics most students in this class are failing
MATCH (school:School {id: $schoolId})
-[:ENROLLS]->(s:Student)
-[:STRUGGLED_WITH]->(t:Topic)
WHERE t.subjectCode = $subjectCode
AND t.gradeLevel  = $gradeLevel
RETURN t.title            AS topic,
COUNT(DISTINCT s)  AS studentsStruggling,
AVG(r.severity)    AS avgSeverity
ORDER BY studentsStruggling DESC
LIMIT 15
### Cross-grade prerequisite alert
If more than 40% of students are struggling with the same Grade 10 topic, the system automatically traverses the prerequisite graph and checks whether the root cause is a Grade 9 concept that was inadequately covered. This alert is surfaced to the teacher: 'Warning: 68% of your class is struggling with Electromagnetic Induction. Root cause analysis suggests Grade 9 Magnetic Flux was not adequately mastered by this cohort.'
ℹ  → DEPENDS ON: Phase 3A (gap records with STRUGGLED_WITH edges) + Phase 1 (prerequisite graph).
ℹ  ✓ DELIVERABLE: Class gap heatmap API. Cross-grade root cause alerts for teachers.

## 4.2 Capability 4B — Exam Quality Scoring
After an exam is taken, the system retroactively scores its quality: did questions successfully discriminate between students who mastered the CLO and those who didn't? Were some questions too easy or too hard for their stated difficulty level?
ℹ  → DEPENDS ON: Phase 2C (student answers) + Phase 2B (CLO alignment) + Phase 1 (CLO-topic graph).
ℹ  ✓ DELIVERABLE: Exam quality report API. Teachers receive actionable feedback after each exam.

## 4.3 Capability 4C — School Quality Scoring
The school admin dashboard displays a quality score per subject per grade — a composite metric derived from: CLO coverage in exams, student gap severity distribution, teacher exam quality scores, and curriculum compliance (are all mandatory topics being assessed?).
### Quality score formula
school_quality_score = (
0.30 × clo_coverage_pct          // % of mandatory CLOs tested
+ 0.25 × student_mastery_pct      // % of students with severity < 0.3
+ 0.25 × exam_quality_avg         // avg discrimination index across exams
+ 0.20 × curriculum_compliance    // % of mandatory topics assessed this term
)
Quality scores are recalculated nightly via a Celery batch job and cached in Redis for dashboard performance. The calculation reads from PostgreSQL (student_answers, exam data) and Neo4j (CLO coverage traversal).
ℹ  → DEPENDS ON: 4A + 4B + all Phase 2 and Phase 3 data populated.
ℹ  ✓ DELIVERABLE: School quality score API. Nightly recalculation job. School admin dashboard feeds.


ℹ  Duration: 8–10 weeks  ·  Team: 2 backend engineers + 1 infrastructure engineer  ·  Starts: after Phase 4 is stable

Phase 5 elevates the system from a school tool to a national intelligence platform. It requires every previous phase to be stable, because it aggregates their outputs across all 50,000+ schools.
## 5.1 Capability 5A — Regional Analytics
Regional Education Bureaus need cross-school comparisons within their region: which schools have the highest CLO coverage? Which subjects have the most systemic student gaps? Which curriculum topics are most commonly the root cause of failures across the region?
### Regional gap root cause ranking
// Most common root-cause topics across all schools in a region
MATCH (region:Region {id: $regionId})
<-[:IN_REGION]-(school:School)
-[:ENROLLS]->(s:Student)
-[r:STRUGGLED_WITH {isRootCause: true}]->(t:Topic)
RETURN t.title           AS rootCauseTopic,
t.gradeLevel      AS grade,
COUNT(DISTINCT s) AS affectedStudents,
AVG(r.severity)   AS avgSeverity
ORDER BY affectedStudents DESC
LIMIT 20
This query reveals systemic curriculum delivery failures at the regional level — for example, if Grade 9 Vector Addition is a root cause gap for 70% of Grade 10 Physics failures across an entire region, that is a teacher training problem, not a student problem.
ℹ  → DEPENDS ON: Phase 4 (school quality scores + gap records at scale).
ℹ  ✓ DELIVERABLE: Regional analytics API. Cross-school gap heatmap. Root cause distribution by region.

## 5.2 Capability 5B — Ministry National Intelligence Dashboard
The Ministry dashboard shows national-level curriculum intelligence: CLO coverage trends over time, national exam pass rate forecasting, curriculum gap patterns by region, and policy intervention recommendations.
### Key national metrics
### Data sovereignty at this layer
Ministry queries aggregate over student data but return only group-level statistics — individual student records are never returned at the Ministry level. This is enforced by a PostgreSQL view that is the only data source for Ministry queries: student PII fields are replaced with anonymous cohort identifiers before aggregation.
ℹ  → DEPENDS ON: All previous phases. Specifically requires Phase 4 quality scores and Phase 3 gap records at national scale.
ℹ  ✓ DELIVERABLE: Ministry intelligence API. Anonymized aggregate views. Policy recommendation engine.

## 5.3 Capability 5C — School Box Offline Sync (Production Hardening)
Offline sync was architected in the system design and prototyped in earlier phases. Phase 5 is where it is hardened to production quality with the full dataset complexity.
### Phase 5 sync requirements
- School Box must sync curriculum graph updates when connectivity is available (MOE updates CLOs, new topics added).
- Student answer queue: answers accumulated offline during a connectivity outage must be flushed to cloud in correct chronological order, with conflict detection.
- Gap analysis re-run: after sync, the cloud re-runs gap analysis for all students who submitted answers offline, using the full cloud prerequisite graph (which may be more complete than the School Box's local copy).
- Model updates: when a new Ollama model version is deployed, School Boxes receive the update during their next sync window.
### CRDT conflict resolution for exam submissions
// Conflict case: teacher grades a question offline,
// and the system also auto-graded it from the MCQ key.
// Resolution: teacher grade always wins (explicit human override).
IF teacher_grade.timestamp > auto_grade.timestamp:
marks_awarded = teacher_grade.marks
ELSE:
marks_awarded = auto_grade.marks
// Flag for teacher review if delta > 20% of marks
ℹ  → DEPENDS ON: All phases operational. Full student dataset available for sync validation.
ℹ  ✓ DELIVERABLE: Production-grade sync. Conflict resolution tested with 50,000 School Box simulation.

## 5.4 Capability 5D — Performance at National Scale
With all 26M+ students, 500K+ teachers, and 50,000+ schools in the system, certain queries that ran in milliseconds on a small dataset will be slow. Phase 5 addresses this systematically.
### Performance work items
- Neo4j query profiling: PROFILE all graph queries on production-scale data. Add composite indexes where traversal time exceeds 100ms.
- PostgreSQL partitioning: partition student_answers by grade_level and academic_year. Partition audit.access_log by month.
- Ministry aggregate materialization: pre-compute and store national aggregate metrics nightly. Ministry dashboard reads from materialized tables, not live queries.
- Redis query result caching: school quality scores (TTL 1hr), ministry aggregates (TTL 5min), regional heatmaps (TTL 10min).
- Read replica routing: TanStack Query on the frontend sends Ministry bulk exports to the PostgreSQL read replica automatically via Kong routing rules.

# 6. Cross-Cutting Concerns
These are not phases — they run continuously throughout all phases. Skipping them to 'ship faster' creates debt that is much more expensive to fix after users are in the system.
## 6.1 Data Migration and Versioning
Every change to the PostgreSQL schema is managed by Flyway. Every change to the Neo4j schema (new node labels, new relationship types, new property indexes) is managed by a custom Neo4j migration runner. No schema change is deployed without a migration script.
- Flyway migrations live in db/migrations/V{version}__{description}.sql
- All migrations are backward-compatible: never drop a column, never rename without an alias.
- Neo4j schema migrations in neo4j/migrations/{version}_{description}.cypher
- Rollback procedures documented for every migration that cannot be automatically reversed.
## 6.2 Multi-Tenancy Enforcement
Multi-tenancy is enforced at four layers simultaneously — any single layer failing must not expose cross-tenant data:
- Layer 1: Kong JWT validation — user's schoolId and regionId extracted from token claims.
- Layer 2: Go middleware — request context always carries schoolId and regionId.
- Layer 3: PostgreSQL RLS — every student-scoped table has a policy enforcing school_id match.
- Layer 4: Neo4j RBAC — Ministry users query anonymized aggregate projections, not individual student nodes.
## 6.3 API Versioning
All APIs are versioned from day one (/api/v1/...). When a breaking change is needed, /api/v2/ is introduced and /api/v1/ is deprecated with a 90-day sunset. School Boxes (which may not update immediately) must continue working during the deprecation window.
## 6.4 Testing Strategy by Phase

## 6.5 Documentation Requirements
- Every API endpoint documented in OpenAPI 3.1 spec (generated from Go code annotations via swaggo).
- Every Cypher query annotated with its performance profile and index requirements.
- ADR (Architecture Decision Record) written for every non-obvious technical decision.
- Runbook for every Celery background job: what it does, when it runs, how to manually trigger, how to check if it failed.

# 7. Master Timeline and Milestone Summary


ℹ  ⚠  The timeline above assumes Phase 1 does not run long. It commonly does — curriculum book parsing and CLO matching require significant iteration with curriculum domain experts. Build in a 2-week buffer after Phase 1 before committing Phase 2 start dates.

## 7.1 Critical Path Items
These items, if delayed, delay every phase that comes after them. They should be started immediately and assigned to the most experienced engineer:
- Curriculum document parser (1A) — all other work is blocked on this.
- Neo4j schema and CLO matching pipeline (1B + 1C) — the graph must be correct before Phase 2.
- Cross-grade prerequisite ingestion and validation (1D) — cannot be added as an afterthought.
- PostgreSQL to Neo4j sync worker (used first in 2C) — needed every phase after Phase 2.
- Embedding model deployment for CLO matching — multilingual-e5-large must be running locally before Phase 1 can finish.
## 7.2 Dependencies on Non-Engineering Teams



EduGraph AI — Backend Implementation Plan v1.0  ·  June 2026
AASTU Innovation Program  ·  For engineering team and technical reviewers
| Step | What depends on it being done first |
| --- | --- |
| Book/Curriculum Upload | The starting point. Every other piece of data originates here. |
| Document Parsing & Structure Extraction | Units, chapters, and topics extracted from the uploaded book. Nothing downstream works without this. |
| Topic Node Creation in Neo4j | Topics become graph nodes. CLO matching, prerequisite linking, and gap analysis all traverse these nodes. |
| CLO Matching per Topic | Each topic linked to its Curriculum Learning Outcome(s). Required for exam validation and gap analysis. |
| Cross-Grade Prerequisite Linking | Grade 9 → Grade 10 → Grade 11 prerequisite chains. Required for root-cause gap analysis and study plans. |
| Exam Validation | Can now check: does each question assess a real CLO for this subject/grade? Requires CLOs + topic graph. |
| Student Answer Ingestion | Student answers recorded against exam questions. Requires exams to exist. |
| Gap Analysis Engine | Which topics did the student fail? Which CLOs? Requires student answers + CLO-to-topic mapping + prerequisites. |
| Study Plan Generator | Ordered study sequence. Requires gap analysis output + prerequisite graph for topological sort. |
| AI Tutor / Assistant | Explains topics, answers questions. Requires topic graph + CLO descriptions + gap context. |
| Teacher Dashboard | Shows class-wide gap heatmap. Requires gap analysis results for all students in a class. |
| School / Regional Analytics | Aggregates over many students and schools. Requires gap analysis at scale. |
| Ministry Intelligence | National curriculum coverage, exam quality. Requires everything above. |
| Capability | Phase 3 uses it | Phase 4 uses it | Phase 5 uses it |
| --- | --- | --- | --- |
| Book Parsing (P1) | Required | Required | Required |
| Topic Graph (P1) | Required | Required | Required |
| CLO Matching (P1) | Required | Required | Required |
| Cross-Grade Links (P1) | Required | Required | Required |
| Exam Ingestion (P2) | — | Required | Required |
| Student Answers (P2) | — | Required | Required |
| Gap Analysis (P3) | — | Required | Required |
| Study Plans (P3) | — | Required | Required |
| AI Tutor (P3) | — | Enriches | Required |
| School Aggregates (P4) | — | — | Required |
| Ministry Analytics (P5) | — | — | Required |
| Phase 1 — Curriculum Intelligence Foundation
Book parsing · Topic graph · CLO matching · Cross-grade prerequisites |
| --- |
| Node Label | Properties |
| --- | --- |
| (:Subject) | Code, name, gradeLevel, academicYear |
| (:Unit) | id, number, title, subjectCode |
| (:Topic) | id, title, unitId, gradeLevel, estimatedHours, examWeight, bloomLevel |
| (:Subtopic) | id, title, topicId, description |
| (:CLO) | code, description, bloomLevel, mandatory, gradeLevel — created in 1C |
| (:KeyConcept) | id, term, definition, topicId — used by AI tutor |
| Relationship | Semantics |
| --- | --- |
| (Subject)-[:HAS_UNIT]->(Unit) | One subject has many units |
| (Unit)-[:HAS_TOPIC]->(Topic) | One unit has many topics |
| (Topic)-[:HAS_SUBTOPIC]->(Subtopic) | One topic has many subtopics |
| (Topic)-[:HAS_CONCEPT]->(KeyConcept) | Topics reference key concepts |
| (Topic)-[:PART_OF_GRADE {grade}]->(Subject) | Allows cross-grade topic queries |
| Phase 2 — Exam & Assessment Infrastructure
Exam creation · Teacher upload · Question-CLO alignment · Student answer ingestion |
| --- |
| Graph Element | Status |
| --- | --- |
| (:Exam) nodes | All published exams, linked to subject/grade nodes |
| (:Question) nodes | All exam questions with [:ASSESSES] edges to CLOs |
| [:PART_OF] edges | Questions linked to their parent exam |
| [:ATTEMPTED] edges | Students linked to exams they took (with score) |
| [:ANSWERED] edges | Students linked to individual questions (with pass/fail) |
| Student nodes | (:Student) nodes linked to their school (:School) nodes |
| Phase 3 — Student Intelligence Layer
Gap analysis · Study plan · AI tutor · Career recommendation |
| --- |
| Phase 4 — Teacher & School Intelligence
Teacher dashboard · Class analytics · School quality scoring · Compliance monitoring |
| --- |
| Metric | Description |
| --- | --- |
| CLO discrimination index | For each CLO: do students who mastered the topic pass the question? If not, the question may be poorly written. |
| Difficulty calibration score | Was the stated difficulty (easy/medium/hard) accurate? Recalibrates based on actual student performance. |
| Time anomaly detection | Questions with extremely short time_spent may indicate answer copying or guessing. |
| Missing CLO coverage post-exam | CLOs that zero students answered correctly — these concepts need reteaching regardless of the exam score. |
| Phase 5 — Regional & Ministry Intelligence
Regional analytics · National policy engine · Offline sync · Performance at scale |
| --- |
| Metric | Description |
| --- | --- |
| National CLO coverage | % of mandatory CLOs being tested in exams nationwide, by subject and grade |
| Root cause concentration | Top 20 topics nationally that are root causes of Grade 12 failure paths — these are curriculum design problems |
| Teacher exam quality trend | Average exam quality score by region — identifies teacher training needs |
| Prerequisite gap propagation | Which Grade 9 gaps are causing Grade 10+ failures at national scale |
| Curriculum change impact | When MOE updates a CLO, what percentage of existing exams are now misaligned |
| Test Type | Starts in Phase | Implementation |
| --- | --- | --- |
| Unit tests | All phases | Go: testify. Python: pytest. React: Vitest. Coverage gate: >80%. |
| Integration tests | All phases | Testcontainers: real PostgreSQL + Neo4j + Redis in Docker. Run on every PR. |
| Graph traversal tests | Phase 1 onwards | Seed test data. Assert specific Cypher query results. Critical for cross-grade prerequisites. |
| End-to-end tests | Phase 2 onwards | Playwright: full user journey from exam upload to study plan. Run nightly. |
| Load tests | Phase 4+ | k6: simulate 50,000 concurrent students submitting exams. Assert P99 < 400ms. |
| Chaos tests | Phase 5 | Chaos Monkey in staging: kill pods, inject DB latency, simulate School Box disconnection. |
| Phase | Timeline | Duration | Key Deliverables |
| --- | --- | --- | --- |
| Phase 1 | Weeks 1–10 | 8–10 weeks | Book ingestion · Topic graph · CLO matching · Cross-grade prerequisites |
| Phase 1 QA Gate | Week 10 | 1 week | All 7 integration tests pass. No exceptions. |
| Phase 2 | Weeks 11–18 | 6–8 weeks | Exam upload · Validation report · Student answer ingestion |
| Phase 3 | Weeks 19–28 | 8–10 weeks | Gap analysis · Study plans · AI tutor · Career recommendation |
| Phase 4 | Weeks 29–36 | 6–8 weeks | Teacher dashboard · Class analytics · School quality scoring |
| Phase 5 | Weeks 37–46 | 8–10 weeks | Regional/Ministry analytics · Sync hardening · National scale perf |
| Total | ~46 weeks | ~11 months | Full backend from curriculum ingestion to national intelligence |
| Team | What Engineering Needs From Them and When |
| --- | --- |
| Ministry of Education | CLO document (needed in Phase 1). Prerequisite dependency map (needed in Phase 1). MOE grade level book PDFs for all subjects. |
| Curriculum Domain Expert | Must be available throughout Phase 1 to review parsed structures, CLO matches, and AI-inferred prerequisites. This is a full-time review role during Phase 1. |
| School Admin (AASTU pilot) | Must upload first real curriculum book for Phase 1 test. Must provide exam for Phase 2 test. |
| Legal/Compliance | Data classification sign-off before Phase 3 goes live (student PII). Ethiopian data protection compliance review. |