<!-- converted from edugraph-db-design.docx -->






EduGraph AI
Full Database Design
PostgreSQL · Neo4j · Redis · S3 — Complete Schema Reference

Version 1.0  ·  June 2026  ·  EduGraph AI Engineering








# Table of Contents

# 1. Database Architecture Overview
EduGraph AI uses four data stores, each chosen for what it does best. No single database does everything.
The system separates concerns across four layers: PostgreSQL owns all transactional, ACID-compliant data; Neo4j owns the curriculum knowledge graph and all relationship-traversal intelligence; Redis owns ephemeral state (sessions, caches, queues); and S3 owns binary objects (documents, uploads, exports). Data that naturally lives as a graph (topics, CLOs, prerequisites, student mastery) is in Neo4j. Data that requires joins, foreign keys, and transactions (grades, enrollments, users) is in PostgreSQL.


ℹ  All student PII is stored in the af-south-1 (Cape Town) region. No student record is replicated outside Africa without explicit Ministry authorisation.


PostgreSQL is the system of record for all transactional data. It is organised into 14 schemas by domain. Row-Level Security (RLS) is enabled on every table that contains student or school data. All timestamps are stored as TIMESTAMPTZ (UTC). All primary keys are UUID v4 unless otherwise noted.


## 2.1 Schema: identity
### identity.users
The central user record. Every person who accesses the system — student, teacher, school admin, regional officer, ministry staff — has one row here.


### identity.sessions

### identity.mfa_challenges

### identity.backup_codes

## 2.2 Schema: curriculum
Stores the structured output of book parsing. This is the PostgreSQL mirror of the Neo4j curriculum graph — used for RLS-controlled reads and as the source for graph writes. The Neo4j graph is derived from these tables, not the other way around.

### curriculum.upload_jobs
Tracks every curriculum book upload from submission through parsing to Neo4j write.

### curriculum.subjects

### curriculum.units

### curriculum.topics
Central to the entire system. Every other table references topics directly or indirectly.


### curriculum.topic_prerequisites
PostgreSQL mirror of the Neo4j PREREQUISITE_OF relationships. Used for RLS queries and sync validation. The Neo4j graph is the authoritative traversal store; this table is the relational mirror.

### curriculum.clos
Curriculum Learning Outcomes from the Ministry of Education. This is the authoritative source — uploaded from the official MOE CLO document.

### curriculum.topic_clo_mappings
Each topic linked to its CLO(s). The match method and score are stored for auditability.

ℹ  UNIQUE constraint on (topic_id, clo_code) — a topic-CLO pair is mapped exactly once. A topic may have multiple CLO mappings, and a CLO may map to multiple topics.

## 2.3 Schema: assessment
Exams, questions, student submissions, and grades. The transactional heart of the student-facing system.

### assessment.exams

### assessment.questions


### assessment.exam_attempts

### assessment.student_answers
One row per question per exam attempt. The most heavily written table in the system — expected 500M+ rows at national scale.

ℹ  ⚠  Table PARTITIONED BY LIST (school_id) at national scale. Partition key is school_id to keep all answers for a school co-located. At 26M students × ~50 answers/exam × 4 exams/year = ~5.2B rows/year.

## 2.4 Schema: students

### students.student_profiles

### students.gap_records
Persisted output of the gap analysis engine. One row per topic per student, updated after each exam attempt.

### students.mastery_records
Positive mastery — topics a student has demonstrably mastered. Set when a student passes questions for all CLOs of a topic across ≥2 exams.

### students.study_plans

## 2.5 Schema: schools

### schools.schools

### schools.quality_scores
Nightly-computed composite quality score per school, per subject, per grade.

## 2.6 Schema: regions

### regions.regions

### regions.regional_aggregates
Pre-computed regional performance summaries. Updated nightly by a Celery batch job.

## 2.7 Schema: ministry

### ministry.national_aggregates
Ministry-level data is always anonymised. This table is only accessible by the ministry role and contains no individual student identifiers.

### ministry.curriculum_change_log
Every change to CLOs, topic structures, or prerequisites is logged here for policy audit.

## 2.8 Schema: careers

### careers.careers

### careers.career_topic_requirements

## 2.9 Schema: sync

### sync.school_box_heartbeats

### sync.sync_operations

### sync.sync_conflicts

## 2.10 Schema: audit
Immutable audit trail. Rows are INSERT-only — no UPDATE or DELETE permitted. Table is append-only enforced by GRANT (no UPDATE/DELETE granted to application roles).

### audit.access_log

### audit.data_change_log
Trigger-maintained. Every INSERT/UPDATE/DELETE on student_answers, gap_records, mastery_records, and exam_attempts creates a row here.

### audit.ai_request_log

## 2.11 Schema: embeddings (pgvector)
Stores vector embeddings for semantic similarity search. Used for CLO-to-question matching and topic-to-CLO alignment. The pgvector extension must be installed: CREATE EXTENSION vector;

### embeddings.clo_embeddings

### embeddings.question_embeddings

ℹ  HNSW index (Hierarchical Navigable Small World) is preferred over IVFFlat for production — it maintains accuracy during online inserts without requiring periodic index rebuilds.


Neo4j stores the curriculum intelligence graph. Every node was first written to PostgreSQL — Neo4j is populated by the sync worker that consumes PostgreSQL logical replication events. The graph is the traversal engine; PostgreSQL is the system of record.

## 3.1 Node Labels and Properties

### (:Subject)

### (:Unit)

### (:Topic) — most important node in the graph

### (:CLO)

### (:Question)

### (:Exam)

### (:Student) — lightweight mirror for traversal only

### (:School)

### (:Career)

## 3.2 Relationship Types — Complete Reference


## 3.3 Neo4j Indexes

## 3.4 Key Traversal Queries — Annotated Reference

### Gap analysis — root cause chain
MATCH (s:Student {id: $studentId})
-[:ANSWERED {passed: false}]->(q:Question)
-[:ASSESSES]->(clo:CLO)
<-[:MAPS_TO_CLO]-(symptomTopic:Topic)

// Walk prerequisites upstream (max 5 hops)
MATCH path = (root:Topic)
-[:PREREQUISITE_OF*1..5]->(symptomTopic)
WHERE NOT (s)-[:MASTERED]->(root)

RETURN symptomTopic.title  AS symptom,
symptomTopic.gradeLevel AS symptomGrade,
root.title          AS rootCause,
root.gradeLevel     AS rootGrade,
LENGTH(path)        AS depth
ORDER BY depth DESC

### Exam validation — missing mandatory CLOs
MATCH (sub:Subject {id: $subjectCode, gradeLevel: $grade})
-[:HAS_UNIT]->()-[:HAS_TOPIC]->(t:Topic)
-[:MAPS_TO_CLO]->(clo:CLO {mandatory: true})
WHERE NOT (clo)<-[:ASSESSES]-(:Question)-[:PART_OF]->(:Exam {id: $examId})
RETURN clo.code AS missingCLO, clo.description AS description

### Career readiness score
MATCH (career:Career)-[:REQUIRES]->(required:Topic)
OPTIONAL MATCH (s:Student {id: $studentId})-[:MASTERED]->(required)
WITH career, COUNT(required) AS total, COUNT(s) AS mastered
RETURN career.name,
ROUND(mastered * 100.0 / total) AS readinessPct
ORDER BY readinessPct DESC LIMIT 5


Redis is used for five distinct purposes: session caching, API response caching, Bull MQ job queues, WebSocket pub-sub channels, and rate-limit counters. Each purpose uses a distinct key prefix and data structure.

## 4.1 Key Naming Convention
All keys follow the pattern: {prefix}:{scope}:{identifier}  —  prefixed by purpose, scoped by tenant where applicable.

## 4.2 Session Keys

## 4.3 API Response Cache Keys

## 4.4 Bull MQ Job Queues
Bull MQ uses Redis sorted sets and lists to manage job state. Key patterns are managed by Bull MQ internally — do not write to these directly.

## 4.5 WebSocket Pub-Sub Channels

## 4.6 Rate Limit Counters (Kong plugin)

## 4.7 Distributed Locks

## 4.8 Redis Configuration


S3 stores all binary objects. Access to S3 is always via presigned URLs generated by the Go API — the frontend never holds permanent AWS credentials. All buckets are in af-south-1. Block Public Access is enabled on all buckets.

## 5.1 Bucket Inventory

## 5.2 Object Key Conventions

## 5.3 Presigned URL Policies

# 6. Cross-Database Synchronisation Design
PostgreSQL is the system of record. Neo4j is populated from PostgreSQL via logical replication. The sync worker ensures consistency without tight coupling.

## 6.1 PostgreSQL → Neo4j Sync Worker
A Go worker process subscribes to a PostgreSQL logical replication slot (using pglogrepl) and processes row-level changes. For each relevant INSERT or UPDATE, it writes the corresponding Neo4j Cypher operation.


## 6.2 Sync Guarantees
- At-least-once delivery: the replication slot preserves events even if the sync worker restarts. Events are consumed idempotently — MERGE in Neo4j makes all writes safe to re-apply.
- neo4j_written flag: after a successful Neo4j write, the sync worker updates the neo4j_written column in PostgreSQL to true. This allows auditing and recovery querying.
- Failure handling: if Neo4j write fails, the event is pushed to a Bull MQ retry queue with exponential backoff (1s, 4s, 16s, 64s). After 5 failures, the event goes to a dead-letter queue and alerts are fired.
- School Box sync: the same sync worker pattern runs on School Boxes, but syncs from local PostgreSQL to cloud PostgreSQL via ElectricSQL, then the cloud sync worker propagates to Neo4j.

## 6.3 Data Consistency Model

# 7. Row-Level Security Policy Reference
RLS is the last line of defence. Even a misconfigured application query cannot return cross-tenant data because PostgreSQL enforces these policies at the engine level.

The Go API middleware sets three session-local variables before every database operation. RLS policies read these variables:
SET LOCAL app.current_user_id   = '{userId}';
SET LOCAL app.current_school_id = '{schoolId}';
SET LOCAL app.current_region_id = '{regionId}';
SET LOCAL app.current_role      = '{role}';


# 8. Migration and Versioning Strategy
All schema changes are managed by Flyway (PostgreSQL) and a custom Cypher migration runner (Neo4j). No schema change may be applied manually to any environment.

## 8.1 Flyway Migration Naming
db/migrations/
V001__create_identity_schema.sql
V002__create_curriculum_schema.sql
V003__create_assessment_schema.sql
V004__create_students_schema.sql
V005__create_schools_schema.sql
V006__create_regions_schema.sql
V007__create_ministry_schema.sql
V008__create_careers_schema.sql
V009__create_sync_schema.sql
V010__create_audit_schema.sql
V011__create_jobs_schema.sql
V012__create_embeddings_schema.sql
V013__create_config_schema.sql
V014__enable_rls_all_student_tables.sql
V015__create_all_indexes.sql
V016__partition_student_answers.sql

## 8.2 Backward Compatibility Rules
- Never DROP a column — rename with alias pattern (add new column, copy data, deprecate old).
- Never rename a table — add a VIEW with the old name.
- New NOT NULL columns must have a DEFAULT value (otherwise existing rows break).
- Foreign key additions must be NOT VALID first, then validated in a separate migration during low-traffic window.
- All Neo4j schema migrations must be additive (MERGE not REPLACE). New properties use OPTIONAL MATCH to maintain backward compatibility.



EduGraph AI — Full Database Design v1.0  ·  June 2026
PostgreSQL · Neo4j · Redis · S3  ·  EduGraph AI Engineering
| Database | Role and Contents |
| --- | --- |
| PostgreSQL 16 (AWS RDS af-south-1) | Users, auth, students, schools, exams, grades, audit. Primary ACID store. 14 schemas, 60+ tables. |
| Neo4j 5.x Enterprise (AuraDB af-south-1) | Curriculum knowledge graph: subjects, topics, CLOs, questions, prerequisites, careers. All traversal intelligence. |
| Redis 7 (ElastiCache af-south-1) | Sessions, API caches, job queues (Bull MQ), WebSocket pub-sub, rate-limit counters. |
| AWS S3 (af-south-1) | Curriculum book PDFs, exam documents, student submission scans, generated reports, model artifacts, audit logs. |
| 2. PostgreSQL Database Design
Complete schema definitions — all tables, columns, indexes, constraints |
| --- |
| Schema | Tables and Purpose |
| --- | --- |
| identity | Users, roles, sessions, MFA, OAuth tokens, password reset |
| curriculum | Subjects, units, topics, CLOs, upload jobs, parse results |
| assessment | Exams, questions, student answers, exam attempts, grades |
| students | Student profiles, enrollment, digital twin snapshots, gap records |
| schools | Schools, quality scores, compliance reports |
| regions | Regions, regional aggregates |
| ministry | National aggregates, policy events, curriculum change log |
| careers | Career definitions, career-topic requirements |
| sync | Sync operations, School Box heartbeats, conflict log |
| notifications | Notification events, delivery log |
| audit | Access log, data change log, AI request log |
| jobs | Background job registry, job results |
| embeddings | CLO embeddings, question embeddings (pgvector) |
| config | System configuration, feature flags |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK, DEFAULT gen_random_uuid() | Unique user identifier |
| email | TEXT | UNIQUE, NOT NULL | Login email address |
| phone | TEXT | UNIQUE, NULLABLE | Mobile number for SMS MFA |
| password_hash | TEXT | NOT NULL | bcrypt hash, cost 12 minimum |
| role | TEXT | NOT NULL, CHECK(role IN (...)) | student | teacher | school_admin | regional_bureau | curriculum_officer | ministry | super_admin |
| school_id | UUID | FK → schools.schools.id, NULLABLE | Set for student/teacher/school_admin roles |
| region_id | UUID | FK → regions.regions.id, NULLABLE | Set for regional_bureau role |
| full_name | TEXT | NOT NULL | Display name (encrypted at application layer) |
| preferred_lang | TEXT | DEFAULT 'am' | am | en | om (Amharic / English / Oromo) |
| is_active | BOOLEAN | DEFAULT true | Soft disable without deletion |
| mfa_enabled | BOOLEAN | DEFAULT false | TOTP MFA enabled flag |
| mfa_secret | TEXT | NULLABLE, ENCRYPTED | TOTP secret, AES-256-GCM encrypted |
| last_login_at | TIMESTAMPTZ | NULLABLE | Last successful authentication |
| created_at | TIMESTAMPTZ | DEFAULT now() | Record creation time |
| updated_at | TIMESTAMPTZ | DEFAULT now() | Last update (trigger-maintained) |
| Index Name | Columns | Type | Purpose |
| --- | --- | --- | --- |
| idx_users_email | email | UNIQUE B-TREE | Login lookup |
| idx_users_school_id | school_id | B-TREE | All users in a school |
| idx_users_region_id | region_id | B-TREE | All users in a region |
| idx_users_role | role | B-TREE | Role-filtered queries |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK | Session identifier |
| user_id | UUID | FK → identity.users.id | Owning user |
| refresh_token | TEXT | UNIQUE, NOT NULL | SHA-256 hash of the actual refresh token |
| device_info | JSONB | NULLABLE | Browser/device metadata for session list UI |
| ip_address | INET | NULLABLE | Client IP at session creation |
| expires_at | TIMESTAMPTZ | NOT NULL | 7-day TTL from last refresh |
| revoked | BOOLEAN | DEFAULT false | Explicit logout or admin revocation |
| created_at | TIMESTAMPTZ | DEFAULT now() | Session start time |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK | Challenge ID |
| user_id | UUID | FK → identity.users | User attempting MFA |
| challenge | TEXT | NOT NULL | Hashed OTP value |
| method | TEXT | NOT NULL | totp | sms | backup_code |
| attempts | INT | DEFAULT 0 | Failed attempt counter (max 5) |
| expires_at | TIMESTAMPTZ | NOT NULL | 5-minute TTL |
| used_at | TIMESTAMPTZ | NULLABLE | Set when successfully verified |
| created_at | TIMESTAMPTZ | DEFAULT now() |  |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK |  |
| user_id | UUID | FK → identity.users |  |
| code_hash | TEXT | NOT NULL | bcrypt hash of the 8-char code |
| used_at | TIMESTAMPTZ | NULLABLE | Single-use — null until consumed |
| created_at | TIMESTAMPTZ | DEFAULT now() |  |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK | Job identifier |
| uploaded_by | UUID | FK → identity.users.id | Curriculum officer who uploaded |
| subject_code | TEXT | NOT NULL | e.g. PHY, MATH, BIO |
| grade_level | INT | NOT NULL, CHECK(1..12) | Grade 1–12 |
| academic_year | TEXT | NOT NULL | e.g. 2025-2026 |
| file_s3_key | TEXT | NOT NULL | S3 object key for the uploaded PDF/DOCX |
| file_name | TEXT | NOT NULL | Original filename for display |
| file_size_bytes | BIGINT | NOT NULL |  |
| status | TEXT | NOT NULL | pending | parsing | parsed | review | approved | rejected | failed |
| parse_error | TEXT | NULLABLE | Error message if status = failed |
| approved_by | UUID | FK → identity.users, NULLABLE | Curriculum officer who approved parsed structure |
| approved_at | TIMESTAMPTZ | NULLABLE |  |
| neo4j_written | BOOLEAN | DEFAULT false | True after successful Neo4j write |
| created_at | TIMESTAMPTZ | DEFAULT now() |  |
| updated_at | TIMESTAMPTZ | DEFAULT now() |  |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| code | TEXT | PK | e.g. PHY, MATH, BIO, CHEM, ENG |
| name_en | TEXT | NOT NULL | English name: Physics |
| name_am | TEXT | NULLABLE | Amharic name |
| grade_level | INT | NOT NULL |  |
| academic_year | TEXT | NOT NULL |  |
| moe_code | TEXT | NULLABLE | Ministry of Education official code |
| is_mandatory | BOOLEAN | DEFAULT true | Mandatory vs elective |
| upload_job_id | UUID | FK → curriculum.upload_jobs | Source upload |
| created_at | TIMESTAMPTZ | DEFAULT now() |  |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK |  |
| subject_code | TEXT | FK → curriculum.subjects.code |  |
| grade_level | INT | NOT NULL |  |
| number | INT | NOT NULL | Unit 1, 2, 3... |
| title_en | TEXT | NOT NULL |  |
| title_am | TEXT | NULLABLE |  |
| neo4j_node_id | TEXT | NULLABLE, UNIQUE | Neo4j internal node ID for sync validation |
| created_at | TIMESTAMPTZ | DEFAULT now() |  |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK |  |
| unit_id | UUID | FK → curriculum.units.id |  |
| subject_code | TEXT | FK → curriculum.subjects.code | Denormalised for query performance |
| grade_level | INT | NOT NULL |  |
| sequence_order | INT | NOT NULL | Position within unit |
| title_en | TEXT | NOT NULL |  |
| title_am | TEXT | NULLABLE |  |
| description | TEXT | NULLABLE | Summary paragraph from the book |
| estimated_hours | NUMERIC | DEFAULT 2.0 | Estimated study hours for this topic |
| exam_weight | NUMERIC | DEFAULT 1.0, CHECK(0..5) | Relative importance in national exam (MOE-defined) |
| bloom_level | TEXT | NULLABLE | remember | understand | apply | analyse | evaluate | create |
| key_concepts | TEXT[] | DEFAULT '{}' | Array of key vocabulary terms extracted from book |
| neo4j_node_id | TEXT | NULLABLE, UNIQUE | Neo4j node ID for sync checks |
| created_at | TIMESTAMPTZ | DEFAULT now() |  |
| updated_at | TIMESTAMPTZ | DEFAULT now() |  |
| Index Name | Columns | Type | Purpose |
| --- | --- | --- | --- |
| idx_topics_subject_grade | subject_code, grade_level | B-TREE | All topics for a subject+grade |
| idx_topics_unit_id | unit_id | B-TREE | All topics in a unit |
| idx_topics_bloom | bloom_level | B-TREE | Filter by Bloom level |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK |  |
| topic_id | UUID | FK → curriculum.topics.id | The downstream topic (requires the prerequisite) |
| prerequisite_id | UUID | FK → curriculum.topics.id | The upstream topic that must be mastered first |
| weight | NUMERIC | DEFAULT 1.0, CHECK(0..1) | Importance of prerequisite relationship |
| is_cross_grade | BOOLEAN | DEFAULT false | True when prerequisite is from a different grade level |
| infer_method | TEXT | NOT NULL | explicit | ai_inferred | manual | moe_document |
| reviewed_by | UUID | FK → identity.users, NULLABLE | Curriculum officer who confirmed |
| confirmed_at | TIMESTAMPTZ | NULLABLE |  |
| created_at | TIMESTAMPTZ | DEFAULT now() |  |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| code | TEXT | PK | e.g. CLO-PHY10-3.2 |
| subject_code | TEXT | FK → curriculum.subjects.code |  |
| grade_level | INT | NOT NULL |  |
| description_en | TEXT | NOT NULL | Full CLO text in English |
| description_am | TEXT | NULLABLE | Amharic translation |
| bloom_level | TEXT | NULLABLE | Bloom's taxonomy level |
| is_mandatory | BOOLEAN | DEFAULT true | Mandatory CLOs must appear in every exam |
| moe_version | TEXT | NOT NULL | MOE document version this came from |
| created_at | TIMESTAMPTZ | DEFAULT now() |  |
| updated_at | TIMESTAMPTZ | DEFAULT now() |  |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK |  |
| topic_id | UUID | FK → curriculum.topics.id |  |
| clo_code | TEXT | FK → curriculum.clos.code |  |
| alignment_score | NUMERIC | CHECK(0..1) | Cosine similarity from embedding match |
| match_method | TEXT | NOT NULL | embedding_auto | human_confirmed | ai_draft | manual |
| reviewed_by | UUID | FK → identity.users, NULLABLE |  |
| confirmed_at | TIMESTAMPTZ | NULLABLE |  |
| created_at | TIMESTAMPTZ | DEFAULT now() |  |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK |  |
| created_by | UUID | FK → identity.users.id | Teacher who created the exam |
| school_id | UUID | FK → schools.schools.id | School this exam belongs to (RLS anchor) |
| subject_code | TEXT | FK → curriculum.subjects.code |  |
| grade_level | INT | NOT NULL |  |
| academic_year | TEXT | NOT NULL |  |
| term | INT | CHECK(1..4) | Term 1-4 |
| title | TEXT | NOT NULL | Exam display name |
| total_marks | INT | NOT NULL |  |
| duration_mins | INT | NULLABLE | Allowed time in minutes |
| due_date | DATE | NULLABLE |  |
| status | TEXT | NOT NULL | draft | validation_pending | published | closed |
| validation_report_s3 | TEXT | NULLABLE | S3 key of generated validation PDF report |
| file_s3_key | TEXT | NULLABLE | Original uploaded exam file |
| neo4j_node_id | TEXT | NULLABLE, UNIQUE |  |
| created_at | TIMESTAMPTZ | DEFAULT now() |  |
| updated_at | TIMESTAMPTZ | DEFAULT now() |  |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK |  |
| exam_id | UUID | FK → assessment.exams.id |  |
| school_id | UUID | FK → schools.schools.id | RLS anchor — denormalised from exam |
| sequence_number | INT | NOT NULL | Question 1, 2, 3... |
| question_text | TEXT | NOT NULL | Full question text |
| question_type | TEXT | NOT NULL | mcq | short_answer | long_answer | essay | calculation |
| marks | INT | NOT NULL | Marks allocated to this question |
| difficulty_level | TEXT | NULLABLE | easy | medium | hard |
| topic_id | UUID | FK → curriculum.topics.id, NULLABLE | Curriculum topic this question tests |
| clo_code | TEXT | FK → curriculum.clos.code, NULLABLE | CLO this question assesses (from validation) |
| clo_align_score | NUMERIC | NULLABLE | Cosine similarity from AI validation |
| clo_align_method | TEXT | NULLABLE | embedding_auto | teacher_confirmed |
| answer_key | JSONB | NULLABLE | For MCQ: {correct_option: 'B'}. For others: null (manual grading) |
| neo4j_node_id | TEXT | NULLABLE, UNIQUE |  |
| created_at | TIMESTAMPTZ | DEFAULT now() |  |
| Index Name | Columns | Type | Purpose |
| --- | --- | --- | --- |
| idx_questions_exam_id | exam_id | B-TREE | All questions in an exam |
| idx_questions_topic_id | topic_id | B-TREE | All questions testing a topic |
| idx_questions_clo_code | clo_code | B-TREE | All questions for a CLO — used in validation |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK |  |
| student_id | UUID | FK → students.student_profiles.id |  |
| exam_id | UUID | FK → assessment.exams.id |  |
| school_id | UUID | FK → schools.schools.id | RLS anchor |
| attempt_number | INT | DEFAULT 1 | For re-takes |
| started_at | TIMESTAMPTZ | NULLABLE |  |
| submitted_at | TIMESTAMPTZ | NULLABLE | Null until submitted |
| total_score | NUMERIC | NULLABLE | Marks earned / total marks |
| percentage | NUMERIC | NULLABLE | 0.0–100.0 |
| passed | BOOLEAN | NULLABLE | Based on school's pass threshold |
| graded_at | TIMESTAMPTZ | NULLABLE | When marking was completed |
| graded_by | UUID | FK → identity.users, NULLABLE | Teacher who graded (null for auto-graded MCQ) |
| is_offline | BOOLEAN | DEFAULT false | True if submitted via School Box while offline |
| synced_at | TIMESTAMPTZ | NULLABLE | When offline submission synced to cloud |
| neo4j_written | BOOLEAN | DEFAULT false |  |
| created_at | TIMESTAMPTZ | DEFAULT now() |  |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK |  |
| attempt_id | UUID | FK → assessment.exam_attempts.id |  |
| student_id | UUID | FK → students.student_profiles.id | Denormalised for RLS + performance |
| question_id | UUID | FK → assessment.questions.id |  |
| school_id | UUID | FK → schools.schools.id | RLS anchor |
| answer_text | TEXT | NULLABLE | Raw answer text or MCQ option selected |
| answer_data | JSONB | NULLABLE | Structured answer (MCQ: {selected: 'B'}, upload: {s3_key}) |
| marks_awarded | NUMERIC | NULLABLE | Marks given (null until graded) |
| marks_possible | INT | NOT NULL | Max marks for this question |
| passed | BOOLEAN | NULLABLE | Derived after grading: marks_awarded >= passing_marks |
| time_spent_secs | INT | NULLABLE | Seconds spent on this question |
| grading_notes | TEXT | NULLABLE | Teacher comments on this answer |
| neo4j_written | BOOLEAN | DEFAULT false | Sync flag for Neo4j ANSWERED edge |
| created_at | TIMESTAMPTZ | DEFAULT now() |  |
| updated_at | TIMESTAMPTZ | DEFAULT now() |  |
| Index Name | Columns | Type | Purpose |
| --- | --- | --- | --- |
| idx_answers_attempt_id | attempt_id | B-TREE | All answers for an attempt |
| idx_answers_student_id | student_id | B-TREE | All answers by a student |
| idx_answers_question_id | question_id | B-TREE | All answers to a question (for class analytics) |
| idx_answers_school_id_passed | school_id, passed | B-TREE | School-scoped pass/fail filter (RLS + gap analysis) |
| idx_answers_neo4j | neo4j_written | PARTIAL B-TREE
(WHERE NOT neo4j_written) | Sync worker query — only unsynced rows |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK, FK → identity.users.id | Same UUID as identity.users — 1:1 |
| school_id | UUID | FK → schools.schools.id, NOT NULL | RLS anchor |
| student_number | TEXT | UNIQUE, NOT NULL | School-assigned student ID |
| national_id | TEXT | NULLABLE, ENCRYPTED | National ID number, AES-256-GCM encrypted |
| grade_level | INT | NOT NULL | Current grade (1–12) |
| section | TEXT | NULLABLE | Class section: A, B, C |
| date_of_birth | DATE | NULLABLE, ENCRYPTED | DOB, encrypted at application layer |
| gender | TEXT | NULLABLE | Stored as coded value, not displayed |
| disability_flags | TEXT[] | DEFAULT '{}' | Accessibility flags for UI/exam adjustments |
| parent_contact | JSONB | NULLABLE, ENCRYPTED | Guardian phone/email, fully encrypted |
| career_interests | TEXT[] | DEFAULT '{}' | Career IDs the student has expressed interest in |
| daily_study_hrs | NUMERIC | DEFAULT 2.0 | Used by study plan generator |
| enrolled_at | DATE | NOT NULL | Date enrolled in current school |
| graduated_at | DATE | NULLABLE | Set on graduation |
| created_at | TIMESTAMPTZ | DEFAULT now() |  |
| updated_at | TIMESTAMPTZ | DEFAULT now() |  |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK |  |
| student_id | UUID | FK → students.student_profiles.id |  |
| school_id | UUID | FK → schools.schools.id | RLS anchor |
| topic_id | UUID | FK → curriculum.topics.id | The topic with the gap |
| clo_code | TEXT | FK → curriculum.clos.code | The CLO that was failed |
| severity_score | NUMERIC | NOT NULL, CHECK(0..1) | 0=minor gap, 1=critical gap |
| is_root_cause | BOOLEAN | DEFAULT false | True if this is the deepest unmastered prerequisite |
| prerequisite_depth | INT | DEFAULT 0 | How many prerequisite hops from the symptom topic |
| detected_in_exam | UUID | FK → assessment.exams.id | Exam where this gap was first detected |
| detected_at | TIMESTAMPTZ | NOT NULL |  |
| resolved_at | TIMESTAMPTZ | NULLABLE | Set when student passes this topic in a subsequent exam |
| neo4j_written | BOOLEAN | DEFAULT false | Sync flag for STRUGGLED_WITH edge |
| created_at | TIMESTAMPTZ | DEFAULT now() |  |
| updated_at | TIMESTAMPTZ | DEFAULT now() |  |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK |  |
| student_id | UUID | FK → students.student_profiles.id |  |
| school_id | UUID | FK → schools.schools.id | RLS anchor |
| topic_id | UUID | FK → curriculum.topics.id |  |
| confidence | NUMERIC | CHECK(0..1) | Score consistency across multiple exams |
| mastered_at | TIMESTAMPTZ | NOT NULL | When mastery threshold was first met |
| neo4j_written | BOOLEAN | DEFAULT false | Sync flag for MASTERED edge |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK |  |
| student_id | UUID | FK → students.student_profiles.id |  |
| school_id | UUID | FK → schools.schools.id | RLS anchor |
| target_exam_id | UUID | FK → assessment.exams.id, NULLABLE | The exam this plan is preparing for |
| target_career_id | UUID | FK → careers.careers.id, NULLABLE | Set for career-focused plans |
| plan_data | JSONB | NOT NULL | Full plan: [{date, topics: [{id,title,grade,hours,why}]}] |
| total_days | INT | NOT NULL |  |
| total_hours | NUMERIC | NOT NULL |  |
| language | TEXT | DEFAULT 'am' | Language of AI-generated explanations in the plan |
| generated_at | TIMESTAMPTZ | NOT NULL |  |
| expires_at | TIMESTAMPTZ | NULLABLE | Plans for a specific exam expire after the exam |
| is_active | BOOLEAN | DEFAULT true | Only one active plan per student at a time |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK |  |
| region_id | UUID | FK → regions.regions.id |  |
| name | TEXT | NOT NULL |  |
| moe_school_code | TEXT | UNIQUE, NOT NULL | Ministry of Education official school code |
| school_type | TEXT | NOT NULL | public | private | community |
| connectivity | TEXT | DEFAULT 'none' | none | intermittent | reliable — affects sync strategy |
| has_school_box | BOOLEAN | DEFAULT false | Whether a School Box device is deployed |
| school_box_id | TEXT | NULLABLE, UNIQUE | School Box hardware identifier |
| location_lat | NUMERIC | NULLABLE | GPS coordinates for regional mapping |
| location_lng | NUMERIC | NULLABLE |  |
| woreda | TEXT | NULLABLE | Administrative sub-district (Ethiopian admin unit) |
| zone | TEXT | NULLABLE | Zone within region |
| created_at | TIMESTAMPTZ | DEFAULT now() |  |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK |  |
| school_id | UUID | FK → schools.schools.id |  |
| subject_code | TEXT | FK → curriculum.subjects.code |  |
| grade_level | INT | NOT NULL |  |
| academic_year | TEXT | NOT NULL |  |
| term | INT | NULLABLE | Term-level score or null for annual |
| overall_score | NUMERIC | CHECK(0..100) | Weighted composite 0–100 |
| clo_coverage_pct | NUMERIC | NOT NULL | % of mandatory CLOs tested in exams this period |
| student_mastery_pct | NUMERIC | NOT NULL | % of students with gap severity < 0.3 |
| exam_quality_avg | NUMERIC | NOT NULL | Average discrimination index across exams |
| curriculum_compliance | NUMERIC | NOT NULL | % of mandatory topics assessed |
| student_count | INT | NOT NULL | Students in scope for this calculation |
| computed_at | TIMESTAMPTZ | NOT NULL |  |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK |  |
| name | TEXT | NOT NULL | e.g. Addis Ababa, Oromia, Amhara |
| code | TEXT | UNIQUE, NOT NULL | Standardised region code |
| created_at | TIMESTAMPTZ | DEFAULT now() |  |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK |  |
| region_id | UUID | FK → regions.regions.id |  |
| subject_code | TEXT | NOT NULL |  |
| grade_level | INT | NOT NULL |  |
| academic_year | TEXT | NOT NULL |  |
| avg_quality_score | NUMERIC | NULLABLE | Average school quality score in this region |
| top_gap_topics | JSONB | NULLABLE | [{topicId, topicTitle, studentCount, avgSeverity}] — top 20 |
| root_cause_topics | JSONB | NULLABLE | [{topicId, topicTitle, rootCauseCount}] — topics that are root causes |
| school_count | INT | NOT NULL | Schools included in this aggregate |
| student_count | INT | NOT NULL | Students included |
| computed_at | TIMESTAMPTZ | NOT NULL |  |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK |  |
| subject_code | TEXT | NOT NULL |  |
| grade_level | INT | NOT NULL |  |
| academic_year | TEXT | NOT NULL |  |
| total_students | INT | NOT NULL |  |
| total_schools | INT | NOT NULL |  |
| pass_rate_pct | NUMERIC | NULLABLE | National exam pass rate |
| avg_clo_coverage | NUMERIC | NULLABLE | National average CLO coverage in exams |
| critical_gap_topics | JSONB | NULLABLE | Top root-cause topics nationally |
| policy_flags | JSONB | NULLABLE | System-generated policy warnings |
| computed_at | TIMESTAMPTZ | NOT NULL |  |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK |  |
| changed_by | UUID | FK → identity.users.id | Ministry/curriculum officer |
| change_type | TEXT | NOT NULL | clo_added | clo_updated | topic_added | prerequisite_changed |
| entity_id | TEXT | NOT NULL | ID of the changed entity |
| old_value | JSONB | NULLABLE | Previous state |
| new_value | JSONB | NOT NULL | New state |
| rationale | TEXT | NULLABLE | Officer's stated reason for change |
| effective_from | DATE | NOT NULL | When the change takes effect in the system |
| created_at | TIMESTAMPTZ | DEFAULT now() |  |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK |  |
| name_en | TEXT | NOT NULL | e.g. Electrical Engineer |
| name_am | TEXT | NULLABLE |  |
| sector | TEXT | NOT NULL | Engineering | Health | Education | Agriculture | etc. |
| min_edu_level | TEXT | NOT NULL | Grade 10 cert | Grade 12 cert | Diploma | Degree |
| description | TEXT | NULLABLE |  |
| moe_classification | TEXT | NULLABLE | MOE career pathway code |
| created_at | TIMESTAMPTZ | DEFAULT now() |  |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK |  |
| career_id | UUID | FK → careers.careers.id |  |
| topic_id | UUID | FK → curriculum.topics.id | Required topic |
| importance | TEXT | CHECK IN ('critical','important','helpful') | Weight in readiness scoring |
| neo4j_written | BOOLEAN | DEFAULT false | Sync flag for [:REQUIRES] edge |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK |  |
| school_id | UUID | FK → schools.schools.id |  |
| box_id | TEXT | NOT NULL | Hardware identifier |
| app_version | TEXT | NOT NULL | Currently running Docker image tag |
| db_lsn | TEXT | NULLABLE | Last PostgreSQL LSN synced from cloud |
| last_seen | TIMESTAMPTZ | NOT NULL | Last successful ping |
| is_online | BOOLEAN | DEFAULT false | True if last heartbeat < 5 min ago |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK |  |
| school_id | UUID | FK → schools.schools.id |  |
| operation_type | TEXT | NOT NULL | student_answers | exam_submission | curriculum_pull | model_update |
| status | TEXT | NOT NULL | pending | in_progress | completed | failed |
| record_count | INT | DEFAULT 0 | Records synced in this operation |
| conflict_count | INT | DEFAULT 0 | Conflicts detected |
| started_at | TIMESTAMPTZ | NULLABLE |  |
| completed_at | TIMESTAMPTZ | NULLABLE |  |
| error_message | TEXT | NULLABLE |  |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK |  |
| school_id | UUID | FK → schools.schools.id |  |
| operation_id | UUID | FK → sync.sync_operations.id |  |
| table_name | TEXT | NOT NULL | Which table had the conflict |
| record_id | UUID | NOT NULL | Conflicting record ID |
| local_value | JSONB | NOT NULL | School Box version |
| cloud_value | JSONB | NOT NULL | Cloud version |
| resolved_value | JSONB | NULLABLE | Final resolved value |
| resolution_rule | TEXT | NULLABLE | last_write_wins | teacher_override | append_only |
| resolved_at | TIMESTAMPTZ | NULLABLE |  |
| created_at | TIMESTAMPTZ | DEFAULT now() |  |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK, DEFAULT gen_random_uuid() |  |
| user_id | UUID | NULLABLE | Null for unauthenticated requests |
| role | TEXT | NULLABLE | Role at time of request |
| school_id | UUID | NULLABLE | School context |
| endpoint | TEXT | NOT NULL | API path (PII fields redacted) |
| http_method | TEXT | NOT NULL | GET | POST | PUT | DELETE |
| http_status | INT | NOT NULL |  |
| response_ms | INT | NULLABLE | Response time in milliseconds |
| ip_address | INET | NULLABLE | Client IP (from Kong) |
| trace_id | TEXT | NULLABLE | Distributed trace ID from Jaeger |
| created_at | TIMESTAMPTZ | DEFAULT now() |  |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK |  |
| table_name | TEXT | NOT NULL | Schema-qualified table name |
| record_id | UUID | NOT NULL | ID of the changed record |
| operation | TEXT | NOT NULL | INSERT | UPDATE | DELETE |
| old_data | JSONB | NULLABLE | Previous row values (null for INSERT) |
| new_data | JSONB | NULLABLE | New row values (null for DELETE) |
| changed_by | UUID | NULLABLE | User ID from SET LOCAL app.current_user_id |
| created_at | TIMESTAMPTZ | NOT NULL |  |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK |  |
| user_id | UUID | NULLABLE |  |
| feature | TEXT | NOT NULL | exam_validation | gap_analysis | study_plan | career_match | tutor | policy_insight |
| model_used | TEXT | NOT NULL | qwen2.5-7b | llama3.1-8b | multilingual-e5-large |
| prompt_hash | TEXT | NOT NULL | SHA-256 of prompt — content never stored |
| tokens_in | INT | NULLABLE |  |
| tokens_out | INT | NULLABLE |  |
| latency_ms | INT | NULLABLE |  |
| is_offline | BOOLEAN | DEFAULT false | School Box local inference vs cloud |
| created_at | TIMESTAMPTZ | NOT NULL |  |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK |  |
| clo_code | TEXT | UNIQUE, FK → curriculum.clos |  |
| embedding | vector(768) | NOT NULL | multilingual-e5-large embedding of CLO description |
| model_ver | TEXT | NOT NULL | Model version used — re-embed when model changes |
| created_at | TIMESTAMPTZ | DEFAULT now() |  |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | UUID | PK |  |
| question_id | UUID | UNIQUE, FK → assessment.questions |  |
| embedding | vector(768) | NOT NULL | Embedding of question_text |
| model_ver | TEXT | NOT NULL |  |
| created_at | TIMESTAMPTZ | DEFAULT now() |  |
| Index Name | Columns | Type | Purpose |
| --- | --- | --- | --- |
| idx_clo_emb_hnsw | embedding | HNSW (vector_cosine_ops) | Approximate nearest neighbour for CLO matching |
| idx_q_emb_hnsw | embedding | HNSW (vector_cosine_ops) | Approximate nearest neighbour for question matching |
| 3. Neo4j Graph Database Design
Node labels · Relationship types · Properties · Indexes · Example traversals |
| --- |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | String | UNIQUE, NOT NULL | Same as curriculum.subjects.code |
| name | String | NOT NULL | English subject name |
| gradeLevel | Integer | NOT NULL | 1–12 |
| academicYear | String | NOT NULL |  |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | String | UNIQUE, NOT NULL | curriculum.units.id |
| number | Integer | NOT NULL | Unit number within subject |
| title | String | NOT NULL |  |
| subjectCode | String | NOT NULL | Back-reference for query filters |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | String | UNIQUE, NOT NULL | curriculum.topics.id |
| title | String | NOT NULL |  |
| gradeLevel | Integer | NOT NULL | Critical for cross-grade traversal filtering |
| subjectCode | String | NOT NULL |  |
| estimatedHours | Float | NOT NULL | Study time estimate — used by study plan generator |
| examWeight | Float | DEFAULT 1.0 | Relative exam importance |
| bloomLevel | String | NULLABLE |  |
| keyConcepts | String[] | DEFAULT [] | For AI tutor context injection |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| code | String | UNIQUE, NOT NULL | e.g. CLO-PHY10-3.2 |
| description | String | NOT NULL | Full CLO text |
| bloomLevel | String | NULLABLE |  |
| gradeLevel | Integer | NOT NULL |  |
| mandatory | Boolean | DEFAULT true | Used in exam validation query |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | String | UNIQUE, NOT NULL | assessment.questions.id |
| difficultyLevel | String | NULLABLE | easy | medium | hard |
| bloomLevel | String | NULLABLE |  |
| marks | Integer | NOT NULL |  |
| type | String | NOT NULL | mcq | short_answer | etc. |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | String | UNIQUE, NOT NULL | assessment.exams.id |
| subjectCode | String | NOT NULL |  |
| gradeLevel | Integer | NOT NULL |  |
| academicYear | String | NOT NULL |  |
| term | Integer | NULLABLE |  |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | String | UNIQUE, NOT NULL | students.student_profiles.id |
| schoolId | String | NOT NULL | For school-scoped aggregation queries |
| gradeLevel | Integer | NOT NULL | Current grade |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | String | UNIQUE, NOT NULL | schools.schools.id |
| regionId | String | NOT NULL | For regional aggregation queries |
| name | String | NOT NULL |  |
| Column | Type | Constraints | Description |
| --- | --- | --- | --- |
| id | String | UNIQUE, NOT NULL | careers.careers.id |
| name | String | NOT NULL |  |
| sector | String | NOT NULL |  |
| Relationship | Properties |
| --- | --- |
| (Subject)-[:HAS_UNIT]->(Unit) | No properties |
| (Unit)-[:HAS_TOPIC]->(Topic) | No properties |
| (Topic)-[:HAS_SUBTOPIC]->(Topic) | No properties — subtopics are also Topic nodes |
| (Topic)-[:MAPS_TO_CLO]->(CLO) | alignmentScore: Float, matchMethod: String, confirmedAt: DateTime |
| (Topic)-[:PREREQUISITE_OF]->(Topic) | weight: Float (0–1), isCrossGrade: Boolean, inferMethod: String |
| (Question)-[:ASSESSES]->(CLO) | alignmentScore: Float, alignMethod: String |
| (Question)-[:PART_OF]->(Exam) | No properties |
| (Exam)-[:FOR_SUBJECT]->(Subject) | No properties |
| (Student)-[:ATTENDS]->(School) | enrolledAt: Date |
| (School)-[:IN_REGION]->(Region) | No properties |
| (Student)-[:ATTEMPTED]->(Exam) | totalScore: Float, percentage: Float, attemptDate: DateTime |
| (Student)-[:ANSWERED]->(Question) | marksAwarded: Float, marksPossible: Int, passed: Boolean, timeSpentSecs: Int |
| (Student)-[:MASTERED]->(Topic) | confidence: Float (0–1), masteredAt: DateTime |
| (Student)-[:STRUGGLED_WITH]->(Topic) | severity: Float (0–1), isRootCause: Boolean, prereqDepth: Int, detectedAt: DateTime |
| (Career)-[:REQUIRES]->(Topic) | importance: String (critical | important | helpful) |
| Index Name | Columns | Type | Purpose |
| --- | --- | --- | --- |
| idx_topic_id | Topic(id) | UNIQUE | Primary lookup — used in every traversal |
| idx_topic_grade_sub | Topic(gradeLevel, subjectCode) | COMPOSITE | Filter topics by grade+subject before traversal |
| idx_clo_code | CLO(code) | UNIQUE | CLO lookups in exam validation |
| idx_student_id | Student(id) | UNIQUE | Student-specific gap/mastery queries |
| idx_student_school | Student(schoolId) | B-TREE | All students in a school |
| idx_question_id | Question(id) | UNIQUE | Question lookups |
| idx_exam_id | Exam(id) | UNIQUE | Exam lookups |
| idx_school_region | School(regionId) | B-TREE | All schools in a region |
| idx_topic_prereq_grade | Topic(gradeLevel) on PREREQUISITE_OF | REL | Optimise cross-grade prerequisite traversal |
| 4. Redis Key Design
Key namespaces · TTLs · Data structures · Eviction policies |
| --- |
| Key Pattern | Structure and Purpose |
| --- | --- |
| session:{userId} | Hash — {refreshTokenHash, deviceInfo, expiresAt}. TTL: 604800s (7 days). One key per active session per user. |
| session:revoked:{jti} | String — '1'. TTL: 900s (15 min = access token lifetime). Used to invalidate access tokens before expiry. |
| mfa:challenge:{userId} | String — challenge hash. TTL: 300s (5 min). Deleted on first use. |
| Key Pattern | Structure, TTL, and Invalidation Trigger |
| --- | --- |
| cache:student:gaps:{studentId} | JSON string. TTL: 600s. Invalidated on new exam_attempt synced to Neo4j. |
| cache:student:plan:{studentId} | JSON string. TTL: 1800s. Invalidated when gap_records updated. |
| cache:school:quality:{schoolId}:{year} | JSON string. TTL: 3600s (1hr). Updated by nightly Celery job. |
| cache:regional:{regionId}:{year} | JSON string. TTL: 600s. Updated by nightly Celery job. |
| cache:ministry:national:{year} | JSON string. TTL: 300s. Updated by nightly Celery job. |
| cache:exam:validation:{examId} | JSON string. TTL: 3600s. Invalidated if exam is edited. |
| cache:curriculum:{subjectCode}:{grade} | JSON string. TTL: 86400s (24hr). Only changes when book re-uploaded. |
| cache:career:readiness:{studentId} | JSON string. TTL: 3600s. Invalidated when mastery_records updated. |
| Queue | Purpose and Concurrency |
| --- | --- |
| bull:ai-jobs:{jobId} | AI inference jobs: gap_analysis, study_plan, exam_validation, career_match. Concurrency: 4. |
| bull:sync-jobs:{jobId} | Neo4j sync jobs from PostgreSQL replication events. Concurrency: 8. |
| bull:report-jobs:{jobId} | Nightly quality score / aggregate recalculation. Concurrency: 2. |
| bull:embed-jobs:{jobId} | Embedding generation for new CLOs and questions. Concurrency: 4. |
| bull:export-jobs:{jobId} | Ministry bulk export PDF/CSV generation. Concurrency: 1 (CPU-heavy). |
| Channel | Trigger / Purpose |
| --- | --- |
| ws:dashboard:ministry | Ministry live counters — published on any national stat change |
| ws:dashboard:region:{regionId} | Regional live feed — published on school quality score update |
| ws:dashboard:school:{schoolId} | School admin live feed — exam submissions, student progress |
| ws:exam:progress:{examId} | Teacher live view of students submitting — count and % done |
| ws:sync:status:{schoolId} | School Box sync progress notifications |
| Key Pattern | Structure and Limits |
| --- | --- |
| ratelimit:{userId}:minute | Counter. TTL: 60s. Values: student=1000, teacher=500, admin=200. |
| ratelimit:{userId}:hour | Counter. TTL: 3600s. Hard ceiling for burst detection. |
| ratelimit:ip:{ipAddr}:minute | IP-level rate limit for unauthenticated requests (login, health check). |
| Lock Key | TTL and Purpose |
| --- | --- |
| lock:sync:{schoolId} | String '1'. TTL: 30s. Prevents concurrent sync operations for the same school. |
| lock:plan:generate:{studentId} | String '1'. TTL: 60s. Prevents duplicate concurrent study plan generation. |
| lock:gap:analyse:{studentId} | String '1'. TTL: 30s. Prevents duplicate concurrent gap analysis. |
| lock:clo:match:{subjectCode} | String '1'. TTL: 300s. Prevents concurrent CLO matching runs for same subject. |
| Config | Value and Rationale |
| --- | --- |
| maxmemory-policy | allkeys-lru — evict least recently used keys when memory is full. API caches are expendable; sessions are protected by TTL (they expire before eviction). |
| Cluster mode | 3 primary shards × 1 replica. Total ~24GB usable memory. Keys distributed by hash slot. |
| Persistence | AOF disabled (cache data is recreatable). RDB snapshot every 300s to S3 — used for Celery queue recovery only. |
| TLS | In-transit encryption to all clients. ElastiCache in-transit encryption enabled. |
| 5. S3 Object Storage Design
Bucket structure · Naming conventions · Lifecycle policies · Access controls |
| --- |
| Bucket Name | Purpose and Lifecycle Policy |
| --- | --- |
| edugraph-curriculum-docs | Uploaded curriculum book PDFs and DOCX files. Versioned. KMS-encrypted. Lifecycle: delete after 5 years. |
| edugraph-exam-files | Teacher-uploaded exam PDFs. Versioned. KMS-encrypted. Lifecycle: archive to Glacier after 2 years. |
| edugraph-student-submissions | Student answer scan images. KMS-encrypted. Lifecycle: Glacier after 1 year, delete after 7 years. |
| edugraph-reports | Generated quality reports, study plan PDFs, validation reports. Lifecycle: delete after 3 years. |
| edugraph-model-artifacts | Ollama model files for School Box deployment. Versioned. Public-read via CloudFront (models are not PII). |
| edugraph-audit-logs | Kong audit logs, application audit exports. Object Lock (Governance mode). 7-year retention. WORM. |
| edugraph-sync-snapshots | School Box sync state snapshots for disaster recovery. Versioned. Delete after 90 days. |
| edugraph-exports | Ministry bulk data exports (anonymised). Presigned URL, 1-hour expiry. Auto-delete after 24 hours. |
| Prefix | Key Template |
| --- | --- |
| curriculum-docs/ | {academicYear}/{gradeLevel}/{subjectCode}/{uploadJobId}/{originalFilename} |
| exam-files/ | {schoolId}/{academicYear}/{examId}/{filename} |
| student-submissions/ | {schoolId}/{studentId}/{examAttemptId}/{questionId}/{filename} |
| reports/ | {type}/{schoolId|regionId}/{year}/{reportId}.pdf |
| model-artifacts/ | {modelName}/{version}/{filename} |
| audit-logs/ | {year}/{month}/{day}/{serviceType}-{hour}.gz |
| exports/ | {requestUserId}/{exportId}.{format} |
| Operation | Expiry, Size Limit, Access Control |
| --- | --- |
| Curriculum book upload (POST) | 300s (5 min). Max 200MB. Only curriculum_officer role can obtain. |
| Exam file upload (POST) | 300s. Max 50MB. Only teacher role, school-scoped. |
| Student submission upload (POST) | 120s. Max 20MB per answer. Student role, own student_id only. |
| Report download (GET) | 3600s (1hr). Role-restricted: school_admin+. |
| Ministry export download (GET) | 3600s. ministry role only. IP-restricted to MoE IP ranges. |
| Model artifact download (GET) | No expiry (public-read via CloudFront). Not PII. |
| PostgreSQL Event | Neo4j Operation |
| --- | --- |
| curriculum.topics INSERT | MERGE (:Topic {id}) with all properties |
| curriculum.topic_prerequisites INSERT | MERGE (:Topic)-[:PREREQUISITE_OF]->(:Topic) |
| curriculum.topic_clo_mappings INSERT | MERGE (:Topic)-[:MAPS_TO_CLO]->(:CLO) |
| assessment.questions INSERT | MERGE (:Question), MERGE -[:PART_OF]->(:Exam), MERGE -[:ASSESSES]->(:CLO) |
| assessment.student_answers INSERT/UPDATE (passed=true) | MERGE (:Student)-[:ANSWERED]->(:Question) |
| students.gap_records INSERT | MERGE (:Student)-[:STRUGGLED_WITH]->(:Topic) |
| students.mastery_records INSERT | MERGE (:Student)-[:MASTERED]->(:Topic) |
| careers.career_topic_requirements INSERT | MERGE (:Career)-[:REQUIRES]->(:Topic) |
| Layer | Consistency Model and Behaviour |
| --- | --- |
| PostgreSQL ↔ Neo4j | Eventual consistency. Neo4j lags by 0–30 seconds in normal operation. Gap analysis and study plan generation always check the lag counter — if Neo4j is >2 minutes behind, the API returns a 503 with a Retry-After header rather than stale results. |
| Cloud ↔ School Box | Eventual consistency via ElectricSQL. School Box may lag by hours or days if offline. All student answers are ACID-committed locally first, then synced. |
| Redis ↔ PostgreSQL | Redis is a cache, not a source of truth. All Redis reads use stale-while-revalidate. Cache misses always fall through to PostgreSQL. |
| Table | RLS Policy (simplified) |
| --- | --- |
| assessment.student_answers | school_id = current_setting('app.current_school_id')::uuid OR current_role IN ('ministry','super_admin') |
| assessment.exam_attempts | school_id = current_setting('app.current_school_id')::uuid OR current_role IN ('regional_bureau','ministry','super_admin') |
| assessment.exams | school_id = current_setting('app.current_school_id')::uuid OR current_role IN ('regional_bureau','ministry','super_admin') |
| students.student_profiles | school_id = current_setting('app.current_school_id')::uuid OR current_role IN ('regional_bureau','ministry','super_admin') |
| students.gap_records | school_id = current_setting('app.current_school_id')::uuid |
| students.mastery_records | school_id = current_setting('app.current_school_id')::uuid |
| schools.quality_scores | school_id = current_setting('app.current_school_id')::uuid OR region check for regional_bureau |
| regions.regional_aggregates | region_id = current_setting('app.current_region_id')::uuid OR role = 'ministry' OR role = 'super_admin' |
| ministry.national_aggregates | role IN ('ministry','super_admin') — no school/region filter applies |
| audit.access_log | No RLS — super_admin only via GRANT. No application role has INSERT/UPDATE/DELETE. |