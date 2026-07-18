# Graph Report - edugraph  (2026-07-18)

## Corpus Check
- 359 files · ~95,022 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 2071 nodes · 3250 edges · 367 communities (263 shown, 104 thin omitted)
- Extraction: 84% EXTRACTED · 16% INFERRED · 0% AMBIGUOUS · INFERRED: 506 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `30a15ab5`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- Authresponse
- Architecture Doc: (root)
- Assessment Dto
- Ai Careermatchrequest
- Frontend: TS Config
- Curriculum Dto
- Parser Docx Extractor
- Teacher Dto
- Jobs Dto
- Impl Plan: Plan Capability 2a
- Notification Dto
- Api Handlers
- DB Migration V011: updated curriculum
- Frontend: UI Kit Components
- Frontend: TS Config
- App Db Postgres
- Auth Repository
- Career Repository
- Notification Handler
- Pkg Crypto Crypto
- DB Design: Compat Migration Rules
- Assessment Handler
- Student Repository
- School Repository
- Sync Dto
- Compose Ai Service
- Frontend: NPM Dependencies
- Src Types Api
- Main Startup Event
- Auth Handler
- Region Repository
- Frontend: NPM Scripts
- DB Design Doc: 1 Database Architecture Overview
- Region Handler
- Pkg Config Config
- Frontend: API Client
- Impl Plan Doc: The Fundamental Dependency Chain
- Main Neo4jdriver Wiring
- School Handler
- Student Handler
- Sync Repository
- Teacher Handler
- Frontend: App Router
- Cmd Api Main
- Career Handler
- Jobs Handler
- Storage Dto
- Impl Plan Doc: (root)
- Values Ai Service
- Ministry Dto
- Storage Handler
- DB Design: Regions Regional Aggregates
- DB Design Doc: (root)
- DB Design Doc: Node Labels And Properties
- Frontend: NPM Dependencies
- Ministry Repository
- DB Design: Design Clo Node
- DB Design: Design Curriculum Clos
- DB Design: Audit Logs Bucket
- Changelog: Backend Changes Step3
- redis.py
- Ministry Handler
- Storage Repository
- Sync Handler
- Service
- DB Design: Assessment Exam Attempts
- DB Design: Audit Access Log
- DB Design: Identity Backup Codes
- Frontend: Curriculum Review UI
- CI Job: Ci Ai Test
- StorageProvider Interface / Dual-Storage System
- dto.go
- Service
- curriculum handler.Upload — fixed wrong context key + double body-read bugs
- autoprefixer
- ag-grid-community
- DB Design: Bullmq Job Queues
- @playwright/test
- App Core Config
- DB Design: Design Career Node
- DB Design: Students Gap Records
- Frontend Package
- Repository
- .SubmitExam
- .UploadExam
- Context
- Internal
- DB Migration V003: regions and schools
- DB Migration V006: assessments
- Job
- Lib Validations Curriculum
- Stores Auth Store
- logger.go
- answer_key.py
- Run
- dto.go
- ag-grid-react
- dto.go
- exam_answer_key.go
- UploadAnswerKeyResponse
- DB Migration V002: users and auth
- DB Migration V004: students and teachers
- DB Migration V005: curriculum
- DB Migration V007: career
- DB Migration V016: curriculum unique constraints
- Components Layout Appheader
- Features Auth Index
- Lib Validations Auth
- Vite Env D
- Frontend: TS Config
- axios
- eslint-plugin-react-refresh
- .UploadAnswerKey
- clo_matcher_llm.py
- ag-grid-community
- eslint-plugin-react-hooks
- postgres_gap.py
- Internal
- 5.3 AI Capabilities
- fetch_clos_for_subject
- persist_analysis
- 6.1 PostgreSQL — Primary Transactional Database
- save_exam_questions
- 6.2 Neo4j — Curriculum Knowledge Graph
- 8.1 Kubernetes Cluster Design
- 9.2 Authorization — Role-Based Access Control
- 10.2 Caching Strategy
- DB Migration V008: notifications
- DB Migration V009: sync logs
- DB Migration V010: jobs
- DB Migration V012: add parsed structure to upload jobs
- DB Migration V013: create local storage
- CI Job: Golangci Lint Config
- Flyway Versions Fix
- Pkg Contextkeys Contextkeys
- Frontend: NPM Dependencies
- Claude
- Frontend: NPM Dependencies
- Frontend: NPM Dependencies
- Frontend: NPM Dependencies
- Frontend: NPM Dependencies
- 9.1 Authentication
- ag-grid-react
- Frontend: NPM Dependencies
- Frontend: NPM Dependencies
- Frontend: NPM Dependencies
- Frontend: NPM Dependencies
- Frontend: NPM Dependencies
- Frontend: NPM Dependencies
- Frontend: NPM Dependencies
- Frontend: NPM Dependencies
- Frontend: NPM Dependencies
- Frontend: NPM Dependencies
- Frontend: NPM Dependencies
- Frontend: NPM Dependencies
- Frontend: NPM Dependencies
- Frontend: NPM Dependencies
- Frontend: NPM Dependencies
- Frontend: NPM Dependencies
- Frontend: NPM Dependencies
- Frontend: NPM Dependencies
- Frontend: NPM Dependencies
- autoprefixer
- Frontend: NPM Dependencies
- Frontend: NPM Dependencies
- Frontend: NPM Dependencies
- Frontend: NPM Dependencies
- Frontend: NPM Dependencies
- Frontend: NPM Dependencies
- Frontend: NPM Dependencies
- Frontend: NPM Dependencies
- Frontend: NPM Dependencies
- Frontend: NPM Dependencies
- Lib Query Config
- Lib Query Keys
- 8.3 CI/CD Pipeline
- Values Autoscaling Api
- Edugraph Values Canary
- Shell Script: Scripts Destroy
- Shell Script: Rotate Secrets
- Shell Script: Scripts Setup
- Shell Script: Scripts Backup
- Shell Script: Health Check
- Shell Script: Scripts Install
- Shell Script: Scripts Reset
- Shell Script: Scripts Update
- Shell Script: Db Backup
- Shell Script: Db Restore
- Shell Script: Run Migrations
- Shell Script: Canary Check
- Shell Script: Deploy Rollback
- Shell Script: Reset Db
- Shell Script: Run All
- Shell Script: Dev Data
- Shell Script: Dev Setup
- V1 Routes Tutor
- Services Tutor Service
- Step2 Curriculum Parsing
- Service Requirements Asyncpg
- Service Requirements Neo4j
- Login Curriculum Upload
- Step4 Neo4j Finalization
- Docker Compose Grafana
- Docker Compose Jaeger
- Docker Compose Prometheus
- Impl Plan: Plan Phase 5
- Lib Validations Exam
- Ci Ai Lint
- Ci Frontend Lint
- Validate Pg Migrations
- Security Scan Govulncheck
- Scan Secret Scan
- Security Scan Trivy
- Values Autoscaling Ai
- Values Frontend Service
- Edugraph Values Ingress
- Edugraph Ai Edugraph
- Docker Compose Caddy
- .AskTutor
- ag-grid-community
- tutor.go
- @testing-library/jest-dom

## God Nodes (most connected - your core abstractions)
1. `WriteError()` - 84 edges
2. `Internal()` - 77 edges
3. `Table of Contents` - 67 edges
4. `WriteJSON()` - 60 edges
5. `BadRequest()` - 58 edges
6. `get_pool()` - 34 edges
7. `NotFound()` - 30 edges
8. `Struct()` - 27 edges
9. `compilerOptions` - 20 edges
10. `UserID()` - 18 edges

## Surprising Connections (you probably didn't know these)
- `Capability 1A — Curriculum Document Ingestion Pipeline` --semantically_similar_to--> `run_forever()`  [INFERRED] [semantically similar]
  graphify-out/converted/edugraph-impl-plan_89427a8d.md → ai-service/app/workers/curriculum_worker.py
- `GET /api/v1/curriculum/jobs/{id} — returns full parsedStructure tree` --shares_data_with--> `process_job()`  [EXTRACTED]
  backend/CHANGES_STEP3.md → ai-service/app/services/curriculum_parser/service.py
- `Capability 1A — Curriculum Document Ingestion Pipeline` --semantically_similar_to--> `curriculum handler.Upload — fixed wrong context key + double body-read bugs`  [INFERRED] [semantically similar]
  graphify-out/converted/edugraph-impl-plan_89427a8d.md → backend/CHANGES.md
- `Capability 1B — Neo4j Curriculum Graph Construction` --semantically_similar_to--> `repository.syncCurriculumGraph — MERGEs Subject/Unit/Topic + HAS_UNIT/HAS_TOPIC into Neo4j`  [INFERRED] [semantically similar]
  graphify-out/converted/edugraph-impl-plan_89427a8d.md → backend/CHANGES_STEP4.md
- `Canary Deployment (disabled by default, weight 0)` --semantically_similar_to--> `watchtower (containrrr/watchtower auto-updater, School Box)`  [INFERRED] [semantically similar]
  infra/helm/edugraph/values.yaml → school-box/compose/docker-compose.yml

## Import Cycles
- 1-file cycle: `ai-service/app/db/neo4j.py -> ai-service/app/db/neo4j.py`

## Hyperedges (group relationships)
- **CI/CD Deployment Pipeline (build → staging → production, gated by security scans)** — github_workflows_ci_build_images, github_workflows_deploy_staging_deploy_staging, github_workflows_deploy_production_deploy_production, github_workflows_security_scan_trivy [INFERRED 0.85]
- **Curriculum Upload-to-Knowledge-Graph Pipeline (upload → parse → review/approve → Neo4j sync)** — backend_internal_curriculum_handler_handler_upload, ai_service_app_workers_curriculum_worker_run_forever, ai_service_app_services_curriculum_parser_service_process_job, backend_internal_curriculum_handler_approve_job, backend_internal_curriculum_repository_repository_synccurriculumgraph [INFERRED 0.85]
- **Neo4j MERGE upsert pattern shared across domain repositories** — backend_internal_curriculum_repository_repository_synccurriculumgraph, backend_internal_student_repository, backend_internal_career_repository [INFERRED 0.85]
- **EduGraph Helm Chart Microservices (api, ai, frontend)** — infra_helm_edugraph_values_api_service, infra_helm_edugraph_values_ai_service, infra_helm_edugraph_values_frontend_service [EXTRACTED 1.00]
- **School Box Offline Deployment Stack** — school_box_compose_docker_compose_api, school_box_compose_docker_compose_ai_service, school_box_compose_docker_compose_postgres, school_box_compose_docker_compose_neo4j, school_box_compose_docker_compose_redis, school_box_compose_docker_compose_ollama, school_box_compose_docker_compose_sync_agent, school_box_compose_docker_compose_caddy, school_box_compose_docker_compose_watchtower [EXTRACTED 1.00]
- **School Box API Persistence & Cache Layer (postgres, neo4j, redis)** — school_box_compose_docker_compose_api, school_box_compose_docker_compose_postgres, school_box_compose_docker_compose_neo4j, school_box_compose_docker_compose_redis [INFERRED 0.85]
- **Backend services (api, ai-service) share a common data layer of Postgres, Neo4j, and Redis** — docker_compose_api, docker_compose_ai_service, docker_compose_postgres, docker_compose_neo4j, docker_compose_redis [EXTRACTED 1.00]
- **Three-tier request routing: frontend proxies to Go api, which in turn calls the Python ai-service** — docker_compose_frontend, docker_compose_api, docker_compose_ai_service [EXTRACTED 1.00]
- **Four-Store Polyglot Persistence Architecture** — edugraph_db_design_postgresql_store, edugraph_db_design_neo4j_store, edugraph_db_design_redis_store, edugraph_db_design_s3_store [EXTRACTED 1.00]
- **CLO-to-Topic/Question Semantic Matching Pipeline** — edugraph_db_design_curriculum_topics, edugraph_db_design_curriculum_clos, edugraph_db_design_embeddings_clo_embeddings, edugraph_db_design_embeddings_question_embeddings, edugraph_db_design_assessment_questions [INFERRED 0.85]
- **Gap Analysis Root-Cause Traversal Pattern** — edugraph_db_design_student_node, edugraph_db_design_topic_node, edugraph_db_design_rel_struggled_with, edugraph_db_design_rel_prerequisite_of, edugraph_db_design_rel_mastered [EXTRACTED 1.00]
- **Phase 1 Capabilities Form the Curriculum Intelligence Foundation** — edugraph_impl_plan_capability_1a, edugraph_impl_plan_capability_1b, edugraph_impl_plan_capability_1c, edugraph_impl_plan_capability_1d [INFERRED 0.90]
- **Phase 3 Capabilities Form the Student Intelligence Layer** — edugraph_impl_plan_capability_3a, edugraph_impl_plan_capability_3b, edugraph_impl_plan_capability_3c, edugraph_impl_plan_capability_3d [INFERRED 0.85]
- **Cross-Cutting Engineering Practices Spanning All Phases** — edugraph_impl_plan_cross_cutting_data_migration, edugraph_impl_plan_cross_cutting_multi_tenancy, edugraph_impl_plan_cross_cutting_api_versioning, edugraph_impl_plan_cross_cutting_testing_strategy, edugraph_impl_plan_cross_cutting_documentation [INFERRED 0.80]
- **4-Step Dual-Storage Curriculum Ingestion Pipeline** — untitled_document_step_1_upload, untitled_document_step_2_brain_work, untitled_document_step_3_human_review, untitled_document_step_4_finalization [EXTRACTED 1.00]

## Communities (367 total, 104 thin omitted)

### Community 0 - "Authresponse"
Cohesion: 0.13
Nodes (19): Time, Context, Pool, Repository, Row, Time, New(), scanSchool() (+11 more)

### Community 1 - "Architecture Doc: (root)"
Cohesion: 0.10
Nodes (20): Time, UUID, ValidationReport, Context, ExamStatus, Repository, UUID, Context (+12 more)

### Community 2 - "Assessment Dto"
Cohesion: 0.70
Nodes (4): DriverWithContext, Pool, Repository, New()

### Community 3 - "Ai Careermatchrequest"
Cohesion: 0.07
Nodes (34): CareerMatchRequest, CareerMatchResult, Client, TutorAskRequest, TutorAskResponse, Repository, Service, New() (+26 more)

### Community 4 - "Frontend: TS Config"
Cohesion: 0.06
Nodes (35): compilerOptions, allowImportingTsExtensions, baseUrl, isolatedModules, jsx, lib, module, moduleDetection (+27 more)

### Community 5 - "Curriculum Dto"
Cohesion: 0.10
Nodes (26): RawMessage, Time, UUID, ApproveResponse, Context, DriverWithContext, JobStatus, ParsedStructurePayload (+18 more)

### Community 6 - "Parser Docx Extractor"
Cohesion: 0.09
Nodes (50): _build_units_topics_clos(), _extract_legacy(), extract_structure(), _group_clos_into_topics(), _heading_level(), _iter_block_items(), DOCX counterpart to extractor.py.  Word documents almost always carry proper par, Dict-based equivalent of extractor.py's _group_clos_into_topics --     groups CL (+42 more)

### Community 7 - "Teacher Dto"
Cohesion: 0.03
Nodes (59): 10.1 Expected Traffic Profile, 10.3 Database Scaling, 10.4 Failover and High Availability, 10.5 Results Day Surge Handling, 11.1 Observability Stack, 11.2 Service Level Objectives, 11.3 Key Alerts, 12.1 Data Sovereignty (+51 more)

### Community 8 - "Jobs Dto"
Cohesion: 0.17
Nodes (9): ExamReviewPage(), IN_PROGRESS_STATUSES, pct(), ValidationReportView(), ExamUploadPage(), GradeExamPage(), StudentExamListPage(), MCQ_OPTIONS (+1 more)

### Community 9 - "Impl Plan: Plan Capability 2a"
Cohesion: 0.13
Nodes (25): Capability 2A — Exam Upload and Parsing, Capability 2B — Exam Validation Report, Capability 2C — Student Answer Ingestion, Capability 3A — Gap Analysis Engine, Capability 3B — Study Plan Generator, Capability 3C — AI Tutor / Academic Assistant, Capability 3D — Career Recommendation Engine, Capability 4A — Class-Wide Gap Heatmap (+17 more)

### Community 10 - "Notification Dto"
Cohesion: 0.14
Nodes (16): Time, Context, Pool, Repository, Row, Time, New(), scan() (+8 more)

### Community 11 - "Api Handlers"
Cohesion: 0.23
Nodes (15): handlers, Handler, Logger, newRouter(), Authenticate(), CORS(), Context, Handler (+7 more)

### Community 12 - "DB Migration V011: updated curriculum"
Cohesion: 0.24
Nodes (19): assessment.exam_attempts, assessment.exams, assessment.questions, assessment.student_answers, careers.career_matches, careers.career_topic_requirements, careers.careers, curriculum.clos (+11 more)

### Community 13 - "Frontend: UI Kit Components"
Cohesion: 0.16
Nodes (13): Banner(), BannerProps, toneStyles, Button, ButtonProps, buttonVariants, Card(), CardContent() (+5 more)

### Community 14 - "Frontend: TS Config"
Cohesion: 0.11
Nodes (17): compilerOptions, allowImportingTsExtensions, esModuleInterop, isolatedModules, lib, module, moduleDetection, moduleResolution (+9 more)

### Community 15 - "App Db Postgres"
Cohesion: 0.17
Nodes (15): fetch_file_bytes(), fetch_job(), get_pool(), mark_failed(), mark_parsing(), Pool, Record, Postgres access layer for the curriculum-parsing worker.  Two tables matter here (+7 more)

### Community 16 - "Auth Repository"
Cohesion: 0.25
Nodes (10): AuthResponse, Context, Repository, Service, hashToken(), New(), nilIfEmpty(), toUserResponse() (+2 more)

### Community 17 - "Career Repository"
Cohesion: 0.24
Nodes (10): Context, DriverWithContext, Pool, Repository, Row, Time, New(), scanCareerPath() (+2 more)

### Community 18 - "Notification Handler"
Cohesion: 0.24
Nodes (8): Handler, Request, ResponseWriter, Service, New(), ResponseWriter, WriteJSONMeta(), statusRecorder

### Community 19 - "Pkg Crypto Crypto"
Cohesion: 0.24
Nodes (9): Context, Repository, UUID, gradeMCQOrPend(), gradeTeacherEntry(), QuestionOption, GradedAnswer, QuestionForGrading (+1 more)

### Community 21 - "DB Design: Compat Migration Rules"
Cohesion: 0.17
Nodes (17): Backward-compatible migration rules, PostgreSQL-Neo4j eventual consistency model, (:Exam) Neo4j node label, Neo4j 5.x Enterprise (AuraDB af-south-1), Four-store polyglot persistence strategy, PostgreSQL 16 (AWS RDS af-south-1), (:Region) Neo4j node label, [:ATTEMPTED] relationship (+9 more)

### Community 22 - "Assessment Handler"
Cohesion: 0.83
Nodes (3): Handler, Service, New()

### Community 23 - "Student Repository"
Cohesion: 0.27
Nodes (10): Context, DriverWithContext, Pool, Repository, Row, Time, New(), scanStudent() (+2 more)

### Community 24 - "School Repository"
Cohesion: 0.13
Nodes (14): BloomBalanceReport, CLOCoverageReport, DifficultyDistributionReport, PrerequisiteWarningEntry, Time, TopicCoverageEntry, UUID, BloomBalanceReport (+6 more)

### Community 25 - "Sync Dto"
Cohesion: 0.21
Nodes (11): Time, Context, Repository, Service, Time, New(), ChangeItem, PulledChange (+3 more)

### Community 26 - "Compose Ai Service"
Cohesion: 0.17
Nodes (14): AI Service (Python/FastAPI), API (Go), API_PROXY_TARGET routing fix, Flyway (DB Migrations), Frontend (React/Vite), Neo4j Graph Database, Ollama (LLM), PostgreSQL + pgvector (+6 more)

### Community 27 - "Frontend: NPM Dependencies"
Cohesion: 0.13
Nodes (15): autoprefixer, eslint, devDependencies, autoprefixer, eslint, postcss, @typescript-eslint/eslint-plugin, @vitejs/plugin-react (+7 more)

### Community 28 - "Src Types Api"
Cohesion: 0.05
Nodes (36): AnswerInput, ApproveRequest, ApproveResponse, AuthResponse, BloomBalanceReport, BulkGradeRequest, BulkGradeResponse, CLOCoverageReport (+28 more)

### Community 29 - "Main Startup Event"
Cohesion: 0.22
Nodes (8): extract_questions(), DOCX counterpart to extractor.py (Capability 2A).  Exam DOCX files won't reliabl, _classify_question_type(), extract_questions(), _parse_lines(), Core "brain work" of exam PDF parsing (Capability 2A).  Unlike curriculum's PDF, Entry point: parses the PDF and returns a flat list of question dicts     ready, _section_type_hint()

### Community 30 - "Auth Handler"
Cohesion: 0.28
Nodes (9): decode(), Handler, Request, ResponseWriter, Service, New(), Request, ResponseWriter (+1 more)

### Community 31 - "Region Repository"
Cohesion: 0.31
Nodes (8): Context, Pool, Repository, Row, Time, New(), scanRegion(), Region

### Community 32 - "Frontend: NPM Scripts"
Cohesion: 0.14
Nodes (14): scripts, build, coverage, dev, format, lint, lint:fix, playwright (+6 more)

### Community 33 - "DB Design Doc: 1 Database Architecture Overview"
Cohesion: 0.32
Nodes (6): DriverWithContext, main(), runMigrations(), splitStatements(), DriverWithContext, NewDriver()

### Community 34 - "Region Handler"
Cohesion: 0.38
Nodes (6): decode(), Handler, Request, ResponseWriter, Service, New()

### Community 35 - "Pkg Config Config"
Cohesion: 0.28
Nodes (11): getenv(), getenvInt(), Duration, Load(), AWSConfig, Config, JWTConfig, Neo4jConfig (+3 more)

### Community 36 - "Frontend: API Client"
Cohesion: 0.14
Nodes (19): apiClient, RetriableConfig, approveCurriculumJob(), bulkGradeExam(), getCurriculumJob(), getExam(), listExamQuestions(), listQuestionsForGrading() (+11 more)

### Community 37 - "Impl Plan Doc: The Fundamental Dependency Chain"
Cohesion: 0.31
Nodes (6): Context, Pool, ReadCloser, Reader, NewPostgresStorage(), PostgresStorage

### Community 38 - "Main Neo4jdriver Wiring"
Cohesion: 0.06
Nodes (45): FastAPI startup event (launches curriculum_worker.run_forever as background task), PDF extractor (Strategy A: TOC / Strategy B: font heuristics), process_job(), Orchestrates one curriculum-parsing job end to end:    1. Fetch the job row (sub, Consumes curriculum-upload jobs off Redis and parses them.  The Go backend pushe, run_forever(), Two-strategy heading extraction design (TOC-first, font-size/bold fallback), Decision to use plain Redis list consumer instead of Celery task (until Go side pushes Celery-formatted messages) (+37 more)

### Community 39 - "School Handler"
Cohesion: 0.39
Nodes (6): decode(), Handler, Request, ResponseWriter, Service, New()

### Community 40 - "Student Handler"
Cohesion: 0.39
Nodes (6): decode(), Handler, Request, ResponseWriter, Service, New()

### Community 41 - "Sync Repository"
Cohesion: 0.36
Nodes (8): Context, Pool, Repository, Time, New(), scanLog(), CreateLogParams, SyncLog

### Community 42 - "Teacher Handler"
Cohesion: 0.39
Nodes (6): decode(), Handler, Request, ResponseWriter, Service, New()

### Community 43 - "Frontend: App Router"
Cohesion: 0.11
Nodes (14): examReviewRoute, examUploadRoute, gradeExamRoute, indexRoute, jobReviewRoute, loginRoute, Register, rootRoute (+6 more)

### Community 44 - "Cmd Api Main"
Cohesion: 0.16
Nodes (18): Handler, Request, ResponseWriter, Handler, Request, ResponseWriter, decode(), Request (+10 more)

### Community 45 - "Career Handler"
Cohesion: 0.38
Nodes (6): decode(), Handler, Request, ResponseWriter, Service, New()

### Community 46 - "Jobs Handler"
Cohesion: 0.22
Nodes (11): decode(), Handler, Request, ResponseWriter, Service, New(), FromRequest(), Request (+3 more)

### Community 47 - "Storage Dto"
Cohesion: 0.11
Nodes (26): Context, Repository, UUID, computeBloomBalance(), computeCLOCoverage(), computeDifficultyDistribution(), computeTopicCoverage(), describeScope() (+18 more)

### Community 48 - "Impl Plan Doc: (root)"
Cohesion: 0.18
Nodes (7): main(), main(), EnsureDevKeyPair(), Pool, NewPool(), InitTracer(), TracerProvider

### Community 49 - "Values Ai Service"
Cohesion: 0.20
Nodes (11): AI Service (Helm default values), API Service (Helm default values), API Service (Production values override: 5 replicas), ai-service (School Box docker-compose), api service (School Box docker-compose), neo4j (neo4j:5.18-community, School Box), ollama (ollama/ollama:latest, School Box), postgres (pgvector/pgvector:pg16, School Box) (+3 more)

### Community 50 - "Ministry Dto"
Cohesion: 0.29
Nodes (6): Context, Repository, Service, New(), OverviewResponse, RegionStatsResponse

### Community 51 - "Storage Handler"
Cohesion: 0.35
Nodes (7): decode(), Handler, Request, ResponseWriter, Service, New(), Struct()

### Community 52 - "DB Design: Regions Regional Aggregates"
Cohesion: 0.29
Nodes (10): regions.regional_aggregates table, regions.regions table, regions schema, schools.quality_scores table, schools schema, schools.schools table, sync schema, sync.school_box_heartbeats table (+2 more)

### Community 53 - "DB Design Doc: (root)"
Cohesion: 0.29
Nodes (7): Context, Repository, Service, New(), PresignDownloadRequest, PresignResponse, PresignUploadRequest

### Community 54 - "DB Design Doc: Node Labels And Properties"
Cohesion: 0.83
Nodes (3): Handler, Service, New()

### Community 55 - "Frontend: NPM Dependencies"
Cohesion: 0.22
Nodes (9): ag-grid-react, dependencies, ag-grid-react, react, react-dom, tailwind-merge, react, react-dom (+1 more)

### Community 56 - "Ministry Repository"
Cohesion: 0.36
Nodes (6): Context, Pool, Repository, New(), Overview, RegionStats

### Community 58 - "DB Design: Design Clo Node"
Cohesion: 0.25
Nodes (10): derive_exam_scope(), derive_unit_numbers(), extract_docx_metadata_table(), extract_pdf_metadata_table(), parse_metadata_table(), Metadata-table extraction for the exam format that has a Subject/Grade Level/Exa, Normalizes the raw {subject, gradeLevel, examType, totalMarks} table     dict in, Scans the first few pages for a table with Subject/Grade Level/Exam     Type/Tot (+2 more)

### Community 59 - "DB Design: Design Curriculum Clos"
Cohesion: 0.26
Nodes (14): (:CLO) Neo4j node label, curriculum.clos table, curriculum schema, curriculum.subjects table, curriculum.topic_clo_mappings table, curriculum.topic_prerequisites table, curriculum.topics table, curriculum.units table (+6 more)

### Community 60 - "DB Design: Audit Logs Bucket"
Cohesion: 0.22
Nodes (9): edugraph-audit-logs S3 bucket, edugraph-curriculum-docs S3 bucket, edugraph-exam-files S3 bucket, edugraph-exports S3 bucket, edugraph-model-artifacts S3 bucket, edugraph-reports S3 bucket, AWS S3 (af-south-1), edugraph-student-submissions S3 bucket (+1 more)

### Community 62 - "redis.py"
Cohesion: 0.13
Nodes (19): Time, Context, Pool, Repository, Row, Time, New(), scanTeacher() (+11 more)

### Community 63 - "Ministry Handler"
Cohesion: 0.24
Nodes (11): Handler, Request, ResponseWriter, Service, New(), writeServiceError(), Handler, Request (+3 more)

### Community 64 - "Storage Repository"
Cohesion: 0.39
Nodes (5): Context, Duration, Repository, New(), PresignClient

### Community 65 - "Sync Handler"
Cohesion: 0.05
Nodes (34): brpop_job(), close_redis(), get_redis(), Redis access for the AI service.  The Go backend pushes a plain job id string on, Block for up to `timeout` seconds waiting for a job id on `queue`.     Returns N, process_answer_key_job(), Processes a separately-uploaded answer-key document (Capability 2C).  Real "Extr, process_exam_job() (+26 more)

### Community 66 - "Service"
Cohesion: 0.23
Nodes (10): Time, Context, Repository, Service, New(), toResponse(), CreateStudentRequest, StudentResponse (+2 more)

### Community 67 - "DB Design: Assessment Exam Attempts"
Cohesion: 0.46
Nodes (8): assessment.exam_attempts table, assessment.exams table, assessment.questions table, assessment schema, assessment.student_answers table, (:Question) Neo4j node label, [:ASSESSES] relationship, [:PART_OF] relationship

### Community 68 - "DB Design: Audit Access Log"
Cohesion: 0.29
Nodes (8): audit.access_log table, audit.ai_request_log table, audit.data_change_log table, audit schema, embeddings.clo_embeddings table, embeddings.question_embeddings table, embeddings schema (pgvector), Row-Level Security as last line of defence

### Community 69 - "DB Design: Identity Backup Codes"
Cohesion: 0.36
Nodes (8): identity.backup_codes table, identity.mfa_challenges table, identity schema, identity.sessions table, identity.users table, ministry.curriculum_change_log table, ministry.national_aggregates table, ministry schema

### Community 70 - "Frontend: Curriculum Review UI"
Cohesion: 0.36
Nodes (4): IN_PROGRESS_STATUSES, JobReviewPage(), linesToList(), UploadPage()

### Community 71 - "CI Job: Ci Ai Test"
Cohesion: 0.25
Nodes (8): CI: AI Service — Tests, CI: Backend — Unit + Integration Tests, CI: Build Docker Images, CI: Frontend — Unit Tests, Deploy Production: Check Deployment Prerequisites, Deploy to Production (canary rollout), Deploy Staging: Check Deployment Prerequisites, Deploy to Staging

### Community 72 - "StorageProvider Interface / Dual-Storage System"
Cohesion: 0.47
Nodes (5): match_clo(), _normalize(), Keyword-overlap CLO matcher (Capability 2A).  Used when a document has no explic, clos: list of (code, description_en). Returns (best_code, score) or     (None, N, _tokenize()

### Community 73 - "dto.go"
Cohesion: 0.39
Nodes (5): Handler, Request, ResponseWriter, Service, New()

### Community 74 - "Service"
Cohesion: 0.26
Nodes (9): Time, Context, Repository, Service, New(), toResponse(), CreateRegionRequest, RegionResponse (+1 more)

### Community 75 - "curriculum handler.Upload — fixed wrong context key + double body-read bugs"
Cohesion: 0.11
Nodes (19): apply_answer_key(), _clo_align_method(), fetch_exam_school(), fetch_valid_clo_codes(), mark_exam_failed(), mark_exam_parsing(), match_subject_code(), Postgres access layer for the exam-parsing worker (Capability 2A).  Mirrors app/ (+11 more)

### Community 76 - "autoprefixer"
Cohesion: 0.67
Nodes (3): getEnv(), main(), DemoUser

### Community 77 - "ag-grid-community"
Cohesion: 0.15
Nodes (17): Time, Context, Pool, Repository, Row, Time, New(), scan() (+9 more)

### Community 78 - "DB Design: Bullmq Job Queues"
Cohesion: 0.33
Nodes (7): Bull MQ job queues (bull:*), Redis API response cache keys (cache:*), Redis distributed locks (lock:*), Redis WebSocket pub-sub channels (ws:*), Redis rate-limit counters (ratelimit:*, Kong plugin), Redis session keys (session:{userId}), Redis 7 (ElastiCache af-south-1)

### Community 83 - "App Core Config"
Cohesion: 0.40
Nodes (3): asyncpg connection string built from the individual settings above., Settings, BaseSettings

### Community 85 - "DB Design: Design Career Node"
Cohesion: 0.50
Nodes (5): (:Career) Neo4j node label, careers.career_topic_requirements table, careers.careers table, careers schema, [:REQUIRES] relationship

### Community 86 - "DB Design: Students Gap Records"
Cohesion: 0.36
Nodes (9): [:ANSWERED] relationship, [:MASTERED] relationship, [:STRUGGLED_WITH] relationship, (:Student) Neo4j node label, students.gap_records table, students.mastery_records table, students schema, students.student_profiles table (+1 more)

### Community 87 - "Frontend Package"
Cohesion: 0.40
Nodes (4): name, private, type, version

### Community 88 - "Repository"
Cohesion: 0.19
Nodes (11): Context, Repository, Tx, UUID, Context, Service, UUID, AddPrerequisiteRequest (+3 more)

### Community 90 - ".SubmitExam"
Cohesion: 0.39
Nodes (7): UUID, AnswerInput, BulkGradeRequest, BulkGradeResponse, GradeEntry, SubmitExamRequest, SubmitExamResponse

### Community 91 - ".UploadExam"
Cohesion: 0.20
Nodes (8): Handler, Request, ResponseWriter, Handler, Request, ResponseWriter, SniffPDFOrDOCX(), File

### Community 92 - "Context"
Cohesion: 0.33
Nodes (5): Context, Repository, UUID, UnsyncedAnswer, UnsyncedAttempt

### Community 93 - "Internal"
Cohesion: 0.30
Nodes (8): Context, Service, UUID, BulkGradeRequest, BulkGradeResponse, GradingQuestion, SubmitExamRequest, SubmitExamResponse

### Community 94 - "DB Migration V003: regions and schools"
Cohesion: 0.67
Nodes (3): regions, schools, users

### Community 95 - "DB Migration V006: assessments"
Cohesion: 0.83
Nodes (3): assessment_questions, assessment_results, assessments

### Community 96 - "Job"
Cohesion: 0.25
Nodes (10): Context, Pool, Repository, Row, Time, New(), scanUser(), CreateUserParams (+2 more)

### Community 97 - "Lib Validations Curriculum"
Cohesion: 0.50
Nodes (3): ACCEPTED_TYPES, UploadCurriculumFormValues, uploadCurriculumSchema

### Community 98 - "Stores Auth Store"
Cohesion: 0.43
Nodes (6): AuthState, canAccessCurriculumReview(), canAccessStudentDashboard(), canAccessTeacherDashboard(), landingPathFor(), useAuthStore

### Community 99 - "logger.go"
Cohesion: 0.36
Nodes (7): Any(), Error(), Logger, Int(), New(), String(), Field

### Community 100 - "answer_key.py"
Cohesion: 0.43
Nodes (6): extract_docx_answer_key(), extract_pdf_answer_key(), _find_column(), Answer Key table extraction (extends 2A, feeds 2C's MCQ auto-grading).  Teacher-, {sequenceNumber: correctOptionLetter}. Scans every table on every     page and m, _rows_to_answer_key()

### Community 101 - "Run"
Cohesion: 0.48
Nodes (6): Context, Duration, Logger, Repository, Run(), syncOnce()

### Community 102 - "dto.go"
Cohesion: 0.32
Nodes (6): Context, Repository, UUID, GradingQuestion, QuestionOption, StudentQuestion

### Community 103 - "ag-grid-react"
Cohesion: 0.42
Nodes (7): Conflict(), Forbidden(), New(), NotFound(), NotImplemented(), Wrap(), AppError

### Community 104 - "dto.go"
Cohesion: 0.28
Nodes (10): RawMessage, Time, UUID, Context, Repository, UUID, ExamInsight, ExamInsightListEntry (+2 more)

### Community 122 - ".UploadAnswerKey"
Cohesion: 0.33
Nodes (5): Context, Reader, Service, UUID, UploadAnswerKeyResponse

### Community 123 - "clo_matcher_llm.py"
Cohesion: 0.50
Nodes (3): match_clo_llm(), Gemini-backed CLO matcher (Capability 2A) -- an optional upgrade over clo_matche, clos: list of (code, description_en). Returns (best_code, confidence)     or (No

### Community 124 - "ag-grid-community"
Cohesion: 0.20
Nodes (11): ComparePassword(), Duration, HashPassword(), loadPrivateKey(), loadPublicKey(), NewJWTSigner(), Claims, JWTSigner (+3 more)

### Community 126 - "postgres_gap.py"
Cohesion: 0.13
Nodes (18): compute_subject_aggregate(), fetch_attempt(), fetch_missed_answers(), fetch_prerequisite_chain_pg(), fetch_topic_mastery(), fetch_topics(), persist_analysis(), Record (+10 more)

### Community 127 - "Internal"
Cohesion: 0.57
Nodes (4): Context, Service, UUID, Internal()

### Community 128 - "5.3 AI Capabilities"
Cohesion: 0.33
Nodes (6): 5.3.1 Exam Assessment Verifier, 5.3.2 Learning Gap Engine, 5.3.3 Study Plan Generator, 5.3.4 Career Recommendation Engine, 5.3.5 Policy Intelligence (Ministry), 5.3 AI Capabilities

### Community 129 - "fetch_clos_for_subject"
Cohesion: 0.40
Nodes (5): fetch_clos_for_subject(), fetch_exam(), Record, Fetch the assessment.exams row a queued exam id points to., CLOs (code + description_en) for the keyword-based matcher in     clo_matcher.py

### Community 130 - "persist_analysis"
Cohesion: 0.16
Nodes (12): RawMessage, Time, UUID, Context, Repository, UUID, Context, Service (+4 more)

### Community 131 - "6.1 PostgreSQL — Primary Transactional Database"
Cohesion: 0.40
Nodes (5): 6.1.1 Multi-Tenant Isolation, 6.1.2 Core Table Groups, 6.1.3 pgvector Extension, 6.1.4 PostgreSQL Configuration, 6.1 PostgreSQL — Primary Transactional Database

### Community 132 - "save_exam_questions"
Cohesion: 0.18
Nodes (11): fetch_gaps_for_plan(), fetch_prereq_edges_pg(), fetch_student_school(), fetch_topic_meta(), insert_study_plan(), Record, Postgres access layer for the study-plan generator (Capability 3B).  Reads the r, Deactivates the student's previous active plan for the same target     (IS NOT D (+3 more)

### Community 133 - "6.2 Neo4j — Curriculum Knowledge Graph"
Cohesion: 0.50
Nodes (4): 6.2.1 Node Types and Properties, 6.2.2 Relationship Types, 6.2.3 Neo4j Configuration, 6.2 Neo4j — Curriculum Knowledge Graph

### Community 134 - "8.1 Kubernetes Cluster Design"
Cohesion: 0.50
Nodes (4): 8.1.1 Node Groups, 8.1.2 Namespace Layout, 8.1.3 Pod Autoscaling, 8.1 Kubernetes Cluster Design

### Community 136 - "9.2 Authorization — Role-Based Access Control"
Cohesion: 0.50
Nodes (4): 9.2.1 Role Hierarchy, 9.2.2 Row-Level Security Enforcement, 9.2.3 Neo4j Authorization, 9.2 Authorization — Role-Based Access Control

### Community 137 - "10.2 Caching Strategy"
Cohesion: 0.67
Nodes (3): 10.2.1 Cache Layers, 10.2.2 Cache Invalidation, 10.2 Caching Strategy

### Community 147 - "Claude"
Cohesion: 0.25
Nodes (7): Architecture, Critical Decisions, Database Structure, Development Rules, graphify, Important Files, Key Components — The Curriculum Pipeline (Phase 1)

### Community 152 - "9.1 Authentication"
Cohesion: 0.67
Nodes (3): 9.1.1 JWT + OAuth 2.0 / OpenID Connect, 9.1.2 Multi-Factor Authentication, 9.1 Authentication

### Community 153 - "ag-grid-react"
Cohesion: 0.32
Nodes (7): fetch_candidate_topics(), fetch_gap_context(), fetch_student(), Record, Postgres access layer for the AI tutor (Capability 3C).  Retrieval side of the G, Topics the question could be about. Grade-scoped to the student's     grade AND, The student's unresolved gaps touching the matched topics -- as     symptom OR r

### Community 173 - "autoprefixer"
Cohesion: 0.33
Nodes (6): Time, AuthResponse, LoginRequest, RefreshRequest, RegisterRequest, UserResponse

### Community 216 - "V1 Routes Tutor"
Cohesion: 0.10
Nodes (22): ask_tutor(), AskRequest, POST /api/v1/tutor/ask (Capability 3C) -- called by the Go API's tutor proxy (in, close_neo4j(), fetch_prerequisite_chain(), fetch_prerequisite_edges_among(), get_driver(), Neo4j access for the AI service (Capability 3A: root-cause traversal).  The Go s (+14 more)

### Community 240 - "Services Tutor Service"
Cohesion: 0.60
Nodes (3): Handler, Request, ResponseWriter

### Community 290 - "Lib Validations Exam"
Cohesion: 0.33
Nodes (5): ACCEPTED_TYPES, UploadAnswerKeyFormValues, uploadAnswerKeySchema, UploadExamFormValues, uploadExamSchema

### Community 355 - ".AskTutor"
Cohesion: 0.40
Nodes (4): Context, Service, UUID, TutorAskRequest

## Ambiguous Edges - Review These
- `AI Service (Python/FastAPI)` → `Ollama (LLM)`  [AMBIGUOUS]
  docker-compose.yml · relation: semantically_similar_to
- `curriculum.clos table` → `ministry.curriculum_change_log table`  [AMBIGUOUS]
  graphify-out/converted/edugraph-db-design_6564a0e9.md · relation: conceptually_related_to
- `audit.ai_request_log table` → `embeddings.clo_embeddings table`  [AMBIGUOUS]
  graphify-out/converted/edugraph-db-design_6564a0e9.md · relation: conceptually_related_to
- `Step 4: The Finalization (Go API → PostgreSQL → Neo4j)` → `StorageProvider Interface / Dual-Storage System`  [AMBIGUOUS]
  graphify-out/converted/Untitled document_d184c67f.md · relation: not_applicable_to

## Knowledge Gaps
- **391 isolated node(s):** `users`, `students`, `teachers`, `notifications`, `sync_logs` (+386 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **104 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **What is the exact relationship between `AI Service (Python/FastAPI)` and `Ollama (LLM)`?**
  _Edge tagged AMBIGUOUS (relation: semantically_similar_to) - confidence is low._
- **What is the exact relationship between `curriculum.clos table` and `ministry.curriculum_change_log table`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **What is the exact relationship between `audit.ai_request_log table` and `embeddings.clo_embeddings table`?**
  _Edge tagged AMBIGUOUS (relation: conceptually_related_to) - confidence is low._
- **What is the exact relationship between `Step 4: The Finalization (Go API → PostgreSQL → Neo4j)` and `StorageProvider Interface / Dual-Storage System`?**
  _Edge tagged AMBIGUOUS (relation: not_applicable_to) - confidence is low._
- **Why does `Internal()` connect `Internal` to `Authresponse`, `Architecture Doc: (root)`, `persist_analysis`, `Ai Careermatchrequest`, `Notification Dto`, `Auth Repository`, `Sync Dto`, `Cmd Api Main`, `Storage Dto`, `Ministry Dto`, `DB Design Doc: (root)`, `redis.py`, `Ministry Handler`, `Service`, `Service`, `ag-grid-community`, `Repository`, `.UploadExam`, `Internal`, `.AskTutor`, `ag-grid-react`, `.UploadAnswerKey`?**
  _High betweenness centrality (0.130) - this node is a cross-community bridge._
- **Why does `WriteError()` connect `Ministry Handler` to `Region Handler`, `School Handler`, `Student Handler`, `dto.go`, `Teacher Handler`, `Api Handlers`, `Cmd Api Main`, `Career Handler`, `Jobs Handler`, `Services Tutor Service`, `Notification Handler`, `Storage Handler`, `.UploadExam`, `Auth Handler`, `Internal`?**
  _High betweenness centrality (0.049) - this node is a cross-community bridge._
- **Why does `newRouter()` connect `Api Handlers` to `Impl Plan Doc: (root)`, `Pkg Config Config`, `Auth Handler`?**
  _High betweenness centrality (0.023) - this node is a cross-community bridge._