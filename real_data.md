You are working on the EduGraph AI platform.

Your task is to create a COMPLETE, realistic, internally consistent synthetic education environment inside the existing EduGraph system and then use that environment to test the entire platform end-to-end.

This is NOT merely a database seeding task.

This is a:

REAL-WORLD EDUCATION SIMULATION
+
DATA GENERATION
+
SYSTEM INTEGRATION
+
MODEL VALIDATION
+
EXAM PLATFORM TEST
+
AI TUTOR TEST
+
STUDY PLAN TEST
+
ANALYTICS TEST
+
ROLE-BASED FRONTEND VALIDATION

project.

The final result should allow me to log into EduGraph as:

- Ministry administrator
- School administrator
- Teacher
- Student

and see a realistic education environment with meaningful data at every level.

The synthetic environment must behave as though it were a real academic term.

============================================================
0. CRITICAL CONSTRAINT
============================================================

The ONLY curriculum currently available in the system is:

BIOLOGY

Grades:

7
8
9
10
11
12

Do NOT create curriculum content for:

- Mathematics
- Physics
- Chemistry
- Geography
- English
- History
- other subjects

unless the existing system requires a generic placeholder field.

All actual academic data must be based on the uploaded Biology curriculum for Grades 7–12.

IMPORTANT:

First inspect the curriculum currently uploaded into EduGraph/Supabase.

Do not invent CLOs, units, topics or concepts if they already exist in the curriculum database.

Use the actual:

- curriculum
- grade
- units
- topics
- CLOs
- concepts
- prerequisite relationships
- learning objectives

that exist in the system.

If something is missing from the uploaded curriculum, clearly identify it rather than silently inventing it.

============================================================
1. FIRST — AUDIT THE EXISTING SYSTEM
============================================================

Before generating any data, inspect the entire existing system.

Inspect:

FRONTEND
- ministry dashboard
- school dashboard
- teacher dashboard
- student dashboard
- curriculum pages
- exam pages
- student exam-taking UI
- analysis dashboards
- gap analysis
- AI tutor
- study plan generator
- recommendations
- reports
- user management

BACKEND
- authentication
- authorization
- users
- schools
- teachers
- students
- classrooms
- enrollments
- curriculum
- exams
- questions
- attempts
- grading
- learning events
- EG-GCKT
- BKT
- DINA
- IRT
- GCSF
- gap analysis
- AI tutor
- study plan generation
- recommendations
- analytics

DATABASE
- Supabase/PostgreSQL
- migrations
- tables
- foreign keys
- RLS
- indexes
- enums
- triggers
- functions
- seed scripts

WORKERS
- Redis
- AI workers
- model workers
- analysis workers

Identify what already exists.

DO NOT duplicate existing entities.

DO NOT create parallel implementations.

Extend the existing architecture.

============================================================
2. OBJECTIVE
============================================================

Create a realistic national education simulation containing:

MINISTRY
   ↓
SCHOOLS
   ↓
TEACHERS
   ↓
CLASSROOMS
   ↓
STUDENTS
   ↓
BIOLOGY CURRICULUM
   ↓
EXAMS
   ↓
STUDENT ATTEMPTS
   ↓
RESULTS
   ↓
LEARNING EVENTS
   ↓
EG-GCKT
   ↓
GAP ANALYSIS
   ↓
AI TUTOR
   ↓
STUDY PLAN
   ↓
RECOMMENDATIONS
   ↓
DASHBOARDS
   ↓
MINISTRY/SCHOOL/TEACHER/STUDENT ANALYTICS

Everything must be connected.

============================================================
3. SCALE OF INITIAL DATASET
============================================================

Create AT LEAST:

3 schools

5 teachers per school

15 teachers total

At least 3 classrooms per school where appropriate.

At least 20 students per classroom.

Target:

3 schools
× 3 classrooms
× 20 students

= AT LEAST 180 students.

You may create more if useful.

Use Grades 7–12.

Each school should contain a realistic mixture of grades.

For example:

School A:
Grade 7
Grade 8
Grade 9

School B:
Grade 9
Grade 10
Grade 11

School C:
Grade 10
Grade 11
Grade 12

You may improve this structure if the existing system has better classroom/course modeling.

The important requirement is that ALL grades 7–12 are represented.

============================================================
4. CREATE A REALISTIC EDUCATION ORGANIZATION
============================================================

Create:

MINISTRY

Example:

Ethiopian Ministry of Education
Synthetic Demo Environment

Do NOT use real people's personal data.

Everything must be clearly synthetic/demo data.

Create ministry administrators.

Example roles:

MINISTRY_ADMIN
MINISTRY_ANALYST
CURRICULUM_OFFICER

If the existing RBAC uses different names, use the existing roles.

============================================================
5. CREATE 3 REALISTIC SCHOOLS
============================================================

Create three distinct fictional schools.

Do not use real school identities.

Example:

School 001:
Addis Horizon Secondary School

School 002:
Blue Nile Science Academy

School 003:
Unity Heights Preparatory School

You may create better names.

Each school must have:

- school_id
- school_code
- school_name
- region
- city
- address
- academic year
- principal/admin
- teachers
- classrooms
- students
- Biology classes

Use synthetic Ethiopian-style locations if useful, but clearly label the dataset as synthetic.

============================================================
6. SCHOOL IDENTIFIERS
============================================================

Implement realistic institutional identifiers.

Examples:

School:

SCH-001
SCH-002
SCH-003

Teacher:

TCH-S01-001

Student:

STU-S01-G07-001

Class:

CLS-S01-G07-A

Exam:

EXM-S01-G07-BIO-001

Attempt:

ATT-STU-S01-G07-001-EXM001

Use the existing ID architecture where possible.

Do not replace UUID primary keys if the system uses UUIDs.

Instead add human-readable identifiers such as:

student_code
teacher_code
school_code
class_code
exam_code

============================================================
7. STUDENT IDENTITIES
============================================================

Create realistic synthetic student profiles.

Every student must have:

- student_id
- student_code
- first_name
- last_name
- gender where supported
- date of birth/age where supported
- grade
- classroom
- school
- enrollment
- academic year
- status
- email
- password/test credential
- admission/enrollment date where supported

DO NOT use real people's information.

Generate synthetic identities.

Use realistic names appropriate to Ethiopia, but explicitly mark the dataset as synthetic.

============================================================
8. TEST LOGIN CREDENTIALS
============================================================

Create login credentials for:

- ministry users
- school administrators
- teachers
- students

At minimum, every teacher and student must have a test account.

IMPORTANT:

These are DEMO accounts only.

Do not use real passwords.

Use a deterministic test password scheme or generated credentials.

For example:

Demo password:

EduGraphDemo!2026

OR generate unique passwords.

If the system requires password hashes, insert proper hashes.

DO NOT store plaintext passwords in production tables unless the existing authentication system explicitly requires it.

Create a separate generated credential report for testing.

The report should contain:

Role
Name
Institution
Username/email
Password
ID

============================================================
9. TEACHERS
============================================================

Create at least 5 Biology teachers per school.

Each teacher should have:

- teacher_id
- teacher_code
- name
- email
- password
- school
- subject
- grade assignments
- classroom assignments
- employment/status
- role

Give teachers different teaching profiles.

Examples:

Teacher A:
experienced, high-performing classes

Teacher B:
mixed-performing classes

Teacher C:
new teacher

Teacher D:
strong at Grade 11/12

Teacher E:
strong at Grade 7/8

Do not make all teachers identical.

============================================================
10. CLASSROOMS
============================================================

Create realistic classrooms.

Example:

Grade 7A
Grade 7B
Grade 8A
Grade 9A
Grade 10A
Grade 11A
Grade 12A

At least 20 students per classroom.

Each student belongs to exactly the appropriate classroom according to the existing schema.

Create teacher/class assignments.

Where supported:

one primary Biology teacher per class.

============================================================
11. STUDENT PERFORMANCE PROFILES
============================================================

THIS IS ONE OF THE MOST IMPORTANT PARTS.

Do NOT randomly assign scores.

Create intentional learner profiles.

Create different types of students.

At minimum:

PROFILE 1 — High Performer

Strong across almost all Biology concepts.

Scores:

Test: 88%
Midterm: 91%
Final: 94%

Expected:

High mastery
High confidence
Few gaps
Positive trend
Minimal remediation

------------------------------------------------------------

PROFILE 2 — Consistent Struggling Student

Low performance throughout.

Test: 48%
Midterm: 45%
Final: 49%

Expected:

Persistent gaps
Low mastery
Prerequisite issues
Study plan required
AI tutor intervention

------------------------------------------------------------

PROFILE 3 — Improving Student

Test: 45%
Midterm: 63%
Final: 79%

Expected:

Strong positive learning trend.

Gap analysis should recognize improvement.

Study plan should reduce/remediate remaining gaps.

------------------------------------------------------------

PROFILE 4 — Declining Student

Test: 82%
Midterm: 67%
Final: 49%

Expected:

Negative trend.

System should identify emerging gaps.

------------------------------------------------------------

PROFILE 5 — High Score but Conceptual Gap

Overall:

88%

But one critical prerequisite concept:

35%

This is VERY IMPORTANT.

The system should not say:

"Student is excellent because score is 88%."

It should identify the specific conceptual weakness.

------------------------------------------------------------

PROFILE 6 — Low Score but Strong Prerequisites

Overall:

58%

But prerequisite concepts:

85%

Expected:

Possible difficulty with advanced/current concepts rather than foundational knowledge.

------------------------------------------------------------

PROFILE 7 — Inconsistent Student

Test:
72%

Midterm:
48%

Final:
81%

Expected:

Mixed evidence.

System should maintain uncertainty instead of making an overly confident diagnosis.

------------------------------------------------------------

PROFILE 8 — Memorization Pattern

Student performs well on direct recall questions but poorly on application questions.

Expected:

Conceptual/application weakness.

------------------------------------------------------------

PROFILE 9 — Prerequisite Gap

Student performs poorly on Concept B.

Historical evidence shows Concept A, which is a prerequisite, was also weak.

Expected:

Gap analysis should trace:

Concept B
↓
Prerequisite A
↓
Evidence

------------------------------------------------------------

PROFILE 10 — Recovery

Student initially struggles badly.

Then improves after intervention.

Test:
41%

Midterm:
58%

Final:
76%

Expected:

System should recognize recovery.

Study plan should adapt.

------------------------------------------------------------

PROFILE 11 — Random/Uncertain Evidence

Scores:

63%
57%
65%

Question-level performance inconsistent.

Expected:

Low/moderate confidence rather than false certainty.

------------------------------------------------------------

PROFILE 12 — Excellent but Specific Misconception

Overall high performance.

Repeatedly chooses the same incorrect distractor on questions associated with a particular concept.

Expected:

Potential misconception signal.

The AI tutor should address the misconception.

============================================================
12. MAP PERFORMANCE TO ACTUAL CURRICULUM
============================================================

DO NOT simply create:

"Biology weak = 40%"

Instead use the actual Biology curriculum.

For each exam:

Question
↓
CLO
↓
Topic
↓
Unit
↓
Prerequisite/concept
↓
Student evidence

Create intentional patterns.

For example:

Student performs poorly on several questions mapped to:

Concept X

and Concept X depends on:

Concept Y

Student previously performed poorly on Concept Y.

This should create a meaningful root-cause scenario.

============================================================
13. EXAMS
============================================================

Create exams for each classroom.

At minimum create:

TEST 1
MIDTERM
FINAL

for each Biology class.

Therefore:

Class
→ Test
→ Midterm
→ Final

Each exam must contain MCQs only.

Use realistic exam sizes.

Example:

Test:
20 questions

Midterm:
30 questions

Final:
40 questions

Adjust based on the existing assessment system.

Every question must have:

- question text
- 4 options
- exactly one correct answer
- marks
- curriculum mapping
- CLO mapping where supported
- topic
- difficulty metadata where supported

============================================================
14. EXAM DIFFICULTY
============================================================

Do not make every question equally easy.

Example:

20-question test:

Easy:
8

Medium:
8

Hard:
4

Midterm:

Easy:
10
Medium:
12
Hard:
8

Final:

Easy:
12
Medium:
18
Hard:
10

Use the actual curriculum concepts.

============================================================
15. QUESTION DESIGN
============================================================

Generate realistic Biology MCQs based on the uploaded curriculum.

Questions should include:

- direct knowledge
- conceptual understanding
- application
- interpretation
- scenario-based questions

Avoid:

- ambiguous wording
- multiple correct answers
- impossible distractors
- obviously incorrect distractors

Every question must be validated.

============================================================
16. STUDENT ANSWERS
============================================================

Do NOT only store final scores.

Generate actual answer patterns.

For each student:

Exam
↓
Question 1
↓
selected answer

Question 2
↓
selected answer

etc.

The final score must be derived from those answers.

DO NOT manually insert:

score = 72

without generating the underlying answers.

The platform must calculate:

correct
incorrect
unanswered
score
percentage

from actual attempt data.

============================================================
17. QUESTION-LEVEL PERFORMANCE PATTERNS
============================================================

Create meaningful answer patterns.

Example:

Student consistently gets:

Concept A questions correct.

Concept B questions wrong.

Concept C mixed.

This should produce the expected learner state.

For misconception scenarios:

Student repeatedly chooses the SAME incorrect distractor.

Example:

Correct:
C

Student repeatedly chooses:
B

Across multiple related questions.

This should create evidence for possible misconception.

============================================================
18. DIFFERENT EXAM BEHAVIOR
============================================================

Simulate realistic exam behavior.

Some students:

- finish early
- finish near deadline
- leave questions unanswered
- change answers
- revisit questions
- temporarily disconnect
- resume
- submit normally
- auto-submit after expiration

Do not make every student behave identically.

============================================================
19. EXAM PLATFORM TESTING
============================================================

Use the generated exams to test:

- login
- eligibility
- exam visibility
- start exam
- timer
- question navigation
- answer selection
- autosave
- refresh
- reconnect
- resume
- submit
- auto-submit
- grading
- result generation
- access control
- exam revision
- question randomization
- option randomization
- audit logs

Actually execute these workflows.

Do not merely insert data into the database.

============================================================
20. EG-GCKT TESTING
============================================================

For every student:

Run the actual EG-GCKT pipeline.

Verify:

Learning Event
↓
BKT
↓
DINA
↓
IRT
↓
GCSF
↓
Student Skill State

where supported by the existing implementation.

Do not fake the model output.

The model must process the generated evidence.

============================================================
21. STUDENT ISOLATION TEST
============================================================

This is mandatory.

Create:

Student A

and:

Student B

with different answer patterns.

Verify:

Student A's evidence

NEVER modifies:

Student B's state.

Run automated assertions.

Do this across:

- learning events
- BKT
- DINA
- IRT
- GCSF
- mastery
- gap analysis
- recommendations
- study plans

============================================================
22. GAP ANALYSIS TESTING
============================================================

For every student profile, run gap analysis.

Verify that:

High performer
→ few/no major gaps

Struggling student
→ multiple gaps

Improving student
→ declining gap severity

Declining student
→ emerging gaps

Prerequisite-gap student
→ root prerequisite identified where evidence supports it

High-score conceptual-gap student
→ specific weakness detected

Uncertain student
→ uncertainty retained

Do not force a diagnosis where evidence is insufficient.

============================================================
23. AI TUTOR TESTING
============================================================

For each major student profile, invoke the AI tutor using the actual student state.

Test questions such as:

"Why am I struggling with this topic?"

"What should I study next?"

"Explain my weakest Biology concept."

"Why did I get these questions wrong?"

"What should I review before learning this topic?"

"Give me practice for my weakest area."

The AI tutor should use the student's actual state.

It must NOT give the same generic response to every student.

Verify personalization.

============================================================
24. STUDY PLAN GENERATOR
============================================================

Generate study plans from the actual learner state.

For each student:

Student state
↓
Weak concepts
↓
Prerequisites
↓
Available curriculum content
↓
Study plan

Study plan should include where supported:

- priority
- concept/topic
- prerequisite
- recommended activity
- estimated effort
- sequence
- target
- reason

Different students must receive different study plans.

Example:

Student A:

Focus:
Advanced topic

Student B:

Focus:
Prerequisite concept

Student C:

Focus:
Misconception correction

============================================================
25. RECOMMENDATION ENGINE
============================================================

Test recommendations.

Examples:

"Review Unit 3 before continuing."

"Practice concept X."

"Work on prerequisite Y first."

"Take another assessment."

"Ask teacher for help."

Recommendations must be based on actual student evidence.

============================================================
26. STUDENT DASHBOARD VALIDATION
============================================================

After seeding and processing:

Log into every major student profile.

Verify:

- profile
- classroom
- exams
- exam history
- scores
- topic performance
- CLO performance
- mastery
- gaps
- trends
- AI tutor
- study plan
- recommendations

The dashboard should differ meaningfully between student profiles.

============================================================
27. TEACHER DASHBOARD VALIDATION
============================================================

Log in as every teacher.

Teacher should see only their permitted school/classes/students.

Verify:

- classes
- students
- exams
- submissions
- scores
- performance distribution
- topic performance
- CLO performance
- struggling students
- improving students
- declining students
- learning gaps
- study plan status
- intervention needs

Teacher A must NOT see unauthorized School B students.

============================================================
28. SCHOOL ADMIN DASHBOARD
============================================================

Log in as each school administrator.

Verify:

- school overview
- number of students
- teachers
- classrooms
- Biology performance
- grade performance
- exam participation
- exam performance
- learning gaps
- teacher/class comparisons
- trends

School A must not see School B's private student data.

============================================================
29. MINISTRY DASHBOARD
============================================================

Log in as ministry administrator/analyst.

Verify aggregate views across all three schools.

Example:

Ministry
↓
Schools
↓
Grades
↓
Classes
↓
Students

Show:

- total students
- schools
- teachers
- Biology participation
- average scores
- grade-level performance
- topic-level gaps
- CLO performance
- improvement trends
- schools needing attention

Do not expose unnecessary personally identifiable student information at ministry aggregate level.

============================================================
30. CROSS-SCHOOL SCENARIOS
============================================================

Create intentionally different school characteristics.

SCHOOL A:

Generally high-performing.

SCHOOL B:

Mixed performance.

SCHOOL C:

Lower performance but strong improvement.

This allows ministry dashboards to demonstrate meaningful differences.

============================================================
31. CLASSROOM DIFFERENCES
============================================================

Do not make all classes identical.

Example:

Grade 9A:
high performance

Grade 9B:
mixed performance

Grade 10A:
strong knowledge but weak application

Grade 11A:
strong performance

Grade 12A:
exam anxiety / inconsistent performance simulation

etc.

Use realistic distributions.

============================================================
32. TEACHER DIFFERENCES
============================================================

Teachers should have different classes and outcomes.

Do NOT artificially make teacher performance causal.

Do not conclude:

"Teacher X is bad"

from synthetic data.

Instead show descriptive patterns:

Class average
Topic gaps
Participation
Improvement

Avoid unsupported causal claims.

============================================================
33. DATA CONSISTENCY
============================================================

All generated relationships must be valid.

Examples:

Student
→ belongs to one school

Student
→ belongs to appropriate classroom

Teacher
→ belongs to school

Teacher
→ teaches class

Class
→ belongs to school

Exam
→ belongs to class/course

Exam
→ uses curriculum version

Question
→ belongs to exam revision

Attempt
→ belongs to student

Attempt
→ belongs to exam revision

Answer
→ belongs to attempt/question

Learning event
→ belongs to student/attempt/question

Model state
→ belongs to student/concept

============================================================
34. SUPABASE
============================================================

The final dataset must actually exist in Supabase.

Use the existing database architecture.

Do not create a separate SQLite database.

Do not create fake JSON-only state.

Seed the actual Supabase database used by the application.

Respect:

- foreign keys
- RLS
- migrations
- triggers
- existing auth architecture
- existing schemas

Use transactions where appropriate.

============================================================
35. SEEDING STRATEGY
============================================================

Create a reproducible seed system.

It must be possible to:

RESET DEMO ENVIRONMENT

and:

SEED DEMO ENVIRONMENT

with deterministic results.

Use a fixed seed.

Example:

EDUGRAPH_DEMO_SEED=20260818

Running the seed twice should NOT create duplicates.

Make the seed idempotent.

============================================================
36. DATA GENERATION MANIFEST
============================================================

Create a manifest describing every generated entity.

For example:

schools.json
teachers.json
students.json
classrooms.json
exams.json
questions.json
attempts.json
learning_events.json

or equivalent according to the existing architecture.

The manifest should allow debugging.

============================================================
37. CREDENTIAL REPORT
============================================================

Create a safe demo credential report containing:

ROLE
NAME
SCHOOL
CLASS
USER EMAIL
PASSWORD
USER ID
STUDENT/TEACHER CODE

Create separate accounts for:

Ministry
School Admin
Teachers
Students

Do not expose production credentials.

============================================================
38. DEMO SCENARIO REPORT
============================================================

Create a comprehensive DOCX report describing the entire synthetic environment.

The report should include:

1. Executive overview
2. Ministry structure
3. Schools
4. Teachers
5. Classrooms
6. Students
7. Student profiles
8. Biology curriculum usage
9. Exams
10. Question strategy
11. Student performance patterns
12. Expected EG-GCKT behavior
13. Expected gap analysis
14. Expected AI tutor behavior
15. Expected study plans
16. Expected dashboards
17. Test scenarios
18. Security scenarios
19. Failure scenarios
20. Cross-role scenarios
21. Login credentials
22. Data validation checklist

============================================================
39. EXPECTED RESULT MATRIX
============================================================

Create a test matrix.

Example:

Student Profile
Expected Score Trend
Expected Mastery
Expected Gaps
Expected Root Cause
Expected AI Tutor Behavior
Expected Study Plan
Expected Dashboard Signal

For every major student profile.

============================================================
40. MODEL VALIDATION
============================================================

Do not merely check:

"model returned a result."

Check:

Is the result logically consistent with the evidence?

Examples:

Student consistently correct:
mastery should generally increase.

Student consistently incorrect:
mastery should generally decrease/remain low.

Student improves:
trend should reflect improvement.

Student has insufficient evidence:
confidence should remain appropriately low/unknown.

Do not assert exact numerical mastery unless the existing model specification guarantees an exact value.

Validate qualitative behavior and statistical outputs appropriately.

============================================================
41. NO FAKE SUCCESS
============================================================

This is critical.

Do not report:

"Everything works"

because the database contains data.

Actually execute:

- APIs
- workers
- model pipeline
- frontend workflows
- dashboards

where possible.

If something fails:

FIX IT.

If something cannot be fixed:

document it clearly.

============================================================
42. FULL SYSTEM TEST MATRIX
============================================================

Test:

AUTHENTICATION

[ ] Ministry login
[ ] School admin login
[ ] Teacher login
[ ] Student login

AUTHORIZATION

[ ] Ministry access
[ ] School isolation
[ ] Teacher isolation
[ ] Student isolation

CURRICULUM

[ ] Grade 7 Biology
[ ] Grade 8 Biology
[ ] Grade 9 Biology
[ ] Grade 10 Biology
[ ] Grade 11 Biology
[ ] Grade 12 Biology

ASSESSMENT

[ ] Test
[ ] Midterm
[ ] Final

EXAM PLATFORM

[ ] Start
[ ] Timer
[ ] Navigation
[ ] Autosave
[ ] Refresh
[ ] Reconnect
[ ] Resume
[ ] Submit
[ ] Auto-submit
[ ] Grading

MODELS

[ ] Learning events
[ ] BKT
[ ] DINA
[ ] IRT
[ ] GCSF
[ ] Student state
[ ] Snapshot
[ ] Evidence
[ ] Gap analysis

AI

[ ] AI tutor
[ ] Study plan
[ ] Recommendations

ANALYTICS

[ ] Student
[ ] Teacher
[ ] School
[ ] Ministry

SECURITY

[ ] IDOR
[ ] RLS
[ ] Role isolation
[ ] Exam answer protection
[ ] Attempt isolation
[ ] Student state isolation

============================================================
43. IMPORTANT: TEST REAL DIFFERENCES BETWEEN STUDENTS
============================================================

This dataset is specifically intended to make it possible to visually inspect the platform.

Therefore:

Do NOT allow all dashboards to look identical.

Student A should have:

green/high mastery

Student B:

multiple gaps

Student C:

improvement

Student D:

decline

Student E:

specific prerequisite problem

Student F:

high score but conceptual weakness

etc.

When I switch student accounts, I should immediately see meaningful differences.

============================================================
44. DO NOT ALTER THE CURRICULUM WITHOUT REASON
============================================================

The uploaded Biology curriculum is the source of truth.

Use its actual structure.

If a generated scenario needs a concept that does not exist:

DO NOT invent a new curriculum node simply to make the test work.

Instead choose another existing curriculum concept.

If the curriculum mapping is incomplete:

report the missing mapping.

============================================================
45. TEST THE COMPLETE LEARNING LOOP
============================================================

At least one scenario must explicitly demonstrate:

Curriculum

→ exam question

→ student answer

→ incorrect answer

→ concept evidence

→ prerequisite evidence

→ EG-GCKT state

→ gap

→ AI tutor explanation

→ study plan

→ later exam

→ improvement

→ updated EG-GCKT state

This should demonstrate that EduGraph is actually a LEARNING SYSTEM rather than a grade database.

============================================================
46. INTERVENTION SCENARIO
============================================================

Create students where the system identifies a gap.

Then simulate:

Teacher reviews gap.

Student receives study plan.

Student completes recommended practice.

Student takes later assessment.

Performance improves.

Verify that:

before intervention:

gap exists

after intervention:

evidence improves

model state updates

recommendation changes

dashboard reflects improvement

============================================================
47. FAILURE SCENARIOS
============================================================

Also test negative cases.

Examples:

Student does not take exam.

Student submits incomplete exam.

Student loses network.

Student's exam expires.

Student attempts unauthorized exam.

Teacher attempts to access another school.

Teacher attempts to modify published exam.

Model worker fails.

AI tutor unavailable.

Study-plan worker fails.

Database temporarily unavailable.

The system should fail gracefully.

============================================================
48. FINAL REPORT
============================================================

After implementation create:

A. DATA GENERATION REPORT

B. CREDENTIAL REPORT

C. SYSTEM TEST REPORT

D. MODEL VALIDATION REPORT

E. EXAM PLATFORM TEST REPORT

F. AI TUTOR TEST REPORT

G. STUDY PLAN TEST REPORT

H. SECURITY TEST REPORT

I. FRONTEND ROLE VALIDATION REPORT

J. KNOWN ISSUES REPORT

============================================================
49. FINAL DELIVERABLES
============================================================

At the end there must be:

1. Seed scripts.
2. Reset scripts.
3. Synthetic data manifest.
4. Supabase populated with the complete environment.
5. Demo user credentials.
6. Comprehensive DOCX scenario document.
7. Automated tests.
8. E2E tests.
9. Model validation results.
10. Exam-platform validation results.
11. AI tutor validation results.
12. Study-plan validation results.
13. Dashboard validation results.
14. Security validation results.
15. Final production-readiness findings.

============================================================
50. FINAL SUCCESS CRITERIA
============================================================

I should be able to do this manually after you finish:

LOGIN AS MINISTRY

and see:

3 schools
15+ teachers
180+ students
multiple grades
Biology performance
cross-school analytics

LOGIN AS SCHOOL ADMIN

and see:

teachers
classrooms
students
Biology exams
school performance

LOGIN AS TEACHER

and see:

my classes
my students
their exams
scores
topic/CLO performance
learning gaps
student trends
intervention needs

LOGIN AS STUDENT

and see:

my profile
my Biology classes
my exams
my results
my mastery
my gaps
my AI tutor
my study plan
my recommendations

TAKE AN EXAM

and verify:

timer
questions
answer selection
autosave
submission
grading
results

THEN:

wait for EG-GCKT processing

and verify:

learning state
gap analysis
AI tutor
study plan

THEN TAKE ANOTHER EXAM

and verify:

the student's learner state changes according to the new evidence.

MOST IMPORTANTLY:

The system must demonstrate that two students who receive the same exam but answer differently develop DIFFERENT learner states.

That is the core proof that EduGraph is functioning as an individualized learning intelligence system.

============================================================
FINAL INSTRUCTION TO THE AI AGENT
============================================================

Take your time.

This is a LARGE implementation and validation task.

Do not rush to generate 180 students and declare success.

First understand the architecture.

Then design the synthetic education scenario.

Then generate the data.

Then insert it into Supabase.

Then execute the real platform workflows.

Then test the models.

Then test the AI tutor.

Then test study plans.

Then test dashboards.

Then test security/isolation.

Then fix problems.

Then produce the documentation.

If the existing architecture has a better way of implementing any part of this specification, use the better architecture while preserving the intent.

Do not create unnecessary parallel systems.

Do not bypass existing business logic by directly inserting derived results that the application is supposed to calculate.

Wherever possible:

SEED RAW INPUTS
→ RUN REAL APPLICATION LOGIC
→ GENERATE DERIVED RESULTS

rather than:

SEED RAW INPUTS + FAKE DERIVED RESULTS.

The purpose of this exercise is to prove that the real EduGraph system works.

Not to make the database LOOK like it works.