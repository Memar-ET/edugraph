# Graph Report - edugraph  (2026-07-18)

## Corpus Check
- 290 files · ~67,828 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 1603 nodes · 2567 edges · 322 communities (220 shown, 102 thin omitted)
- Extraction: 86% EXTRACTED · 14% INFERRED · 0% AMBIGUOUS · INFERRED: 367 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `a4efbe12`
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
- lifespan
- DB Design: Assessment Exam Attempts
- DB Design: Audit Access Log
- DB Design: Identity Backup Codes
- Frontend: Curriculum Review UI
- CI Job: Ci Ai Test
- StorageProvider Interface / Dual-Storage System
- dto.go
- Capability 1A — Curriculum Document Ingestion Pipeline
- curriculum handler.Upload — fixed wrong context key + double body-read bugs
- autoprefixer
- ag-grid-community
- DB Design: Bullmq Job Queues
- @playwright/test
- autoprefixer
- Architecture Doc: 1 Exam Assessment Verifier
- App Core Config
- DB Design: Design Career Node
- DB Design: Students Gap Records
- Frontend Package
- Architecture Doc: 1 Multi Tenant Isolation
- DB Migration V003: regions and schools
- DB Migration V006: assessments
- Lib Validations Curriculum
- Stores Auth Store
- Architecture Doc: Node Types And Properties
- Architecture Doc: 1 1 Node Groups
- Architecture Doc: 2 1 Role Hierarchy
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
- Architecture Doc: 2 1 Cache Layers
- Architecture Doc: 2 0 Openid Connect
- Frontend: NPM Dependencies
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
- Lib Query Config
- Lib Query Keys
- Architecture Doc: 3 1 Pipeline Stages
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
- Step2 Curriculum Parsing
- Service Requirements Asyncpg
- Service Requirements Neo4j
- Login Curriculum Upload
- Step4 Neo4j Finalization
- Docker Compose Grafana
- Docker Compose Jaeger
- Docker Compose Prometheus
- Impl Plan: Plan Phase 5
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

## God Nodes (most connected - your core abstractions)
1. `WriteError()` - 76 edges
2. `Table of Contents` - 67 edges
3. `Internal()` - 58 edges
4. `WriteJSON()` - 50 edges
5. `BadRequest()` - 41 edges
6. `NotFound()` - 24 edges
7. `Struct()` - 24 edges
8. `compilerOptions` - 20 edges
9. `Phase 1 — Curriculum Intelligence Foundation` - 17 edges
10. `PostgreSQL 16 (AWS RDS af-south-1)` - 15 edges

## Surprising Connections (you probably didn't know these)
- `Capability 1A — Curriculum Document Ingestion Pipeline` --semantically_similar_to--> `run_forever()`  [INFERRED] [semantically similar]
  graphify-out/converted/edugraph-impl-plan_89427a8d.md → ai-service/app/workers/curriculum_worker.py
- `Capability 1A — Curriculum Document Ingestion Pipeline` --semantically_similar_to--> `curriculum handler.Upload — fixed wrong context key + double body-read bugs`  [INFERRED] [semantically similar]
  graphify-out/converted/edugraph-impl-plan_89427a8d.md → backend/CHANGES.md
- `Capability 1B — Neo4j Curriculum Graph Construction` --semantically_similar_to--> `repository.syncCurriculumGraph — MERGEs Subject/Unit/Topic + HAS_UNIT/HAS_TOPIC into Neo4j`  [INFERRED] [semantically similar]
  graphify-out/converted/edugraph-impl-plan_89427a8d.md → backend/CHANGES_STEP4.md
- `POST /api/v1/curriculum/jobs/{id}/approve — promotes parsed tree into curriculum.subjects/units/topics/clos` --shares_data_with--> `process_job()`  [EXTRACTED]
  backend/CHANGES_STEP3.md → ai-service/app/services/curriculum_parser/service.py
- `GET /api/v1/curriculum/jobs/{id} — returns full parsedStructure tree` --shares_data_with--> `process_job()`  [EXTRACTED]
  backend/CHANGES_STEP3.md → ai-service/app/services/curriculum_parser/service.py

## Import Cycles
- None detected.

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

## Communities (322 total, 102 thin omitted)

### Community 0 - "Authresponse"
Cohesion: 0.06
Nodes (46): AuthResponse, Context, Repository, Service, hashToken(), New(), nilIfEmpty(), toUserResponse() (+38 more)

### Community 1 - "Architecture Doc: (root)"
Cohesion: 0.03
Nodes (59): 10.1 Expected Traffic Profile, 10.3 Database Scaling, 10.4 Failover and High Availability, 10.5 Results Day Surge Handling, 11.1 Observability Stack, 11.2 Service Level Objectives, 11.3 Key Alerts, 12.1 Data Sovereignty (+51 more)

### Community 2 - "Assessment Dto"
Cohesion: 0.10
Nodes (26): Time, Context, DriverWithContext, Pool, Repository, Time, New(), Context (+18 more)

### Community 3 - "Ai Careermatchrequest"
Cohesion: 0.07
Nodes (29): CareerMatchRequest, CareerMatchResult, Client, Time, Context, Repository, Service, New() (+21 more)

### Community 4 - "Frontend: TS Config"
Cohesion: 0.06
Nodes (35): compilerOptions, allowImportingTsExtensions, baseUrl, isolatedModules, jsx, lib, module, moduleDetection (+27 more)

### Community 5 - "Curriculum Dto"
Cohesion: 0.19
Nodes (14): ApproveResponse, Context, DriverWithContext, JobStatus, ParsedStructurePayload, Pool, Repository, UUID (+6 more)

### Community 6 - "Parser Docx Extractor"
Cohesion: 0.07
Nodes (57): _build_units_topics_clos(), _extract_legacy(), extract_structure(), _group_clos_into_topics(), _heading_level(), _iter_block_items(), DOCX counterpart to extractor.py.  Word documents almost always carry proper par, Dict-based equivalent of extractor.py's _group_clos_into_topics --     groups CL (+49 more)

### Community 7 - "Teacher Dto"
Cohesion: 0.13
Nodes (19): Time, Context, Pool, Repository, Row, Time, New(), scanTeacher() (+11 more)

### Community 8 - "Jobs Dto"
Cohesion: 0.15
Nodes (17): Time, Context, Pool, Repository, Row, Time, New(), scan() (+9 more)

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
Cohesion: 0.15
Nodes (16): close_pool(), fetch_file_bytes(), fetch_job(), get_pool(), mark_failed(), mark_parsing(), Pool, Postgres access layer for the curriculum-parsing worker.  Two tables matter here (+8 more)

### Community 16 - "Auth Repository"
Cohesion: 0.25
Nodes (10): Context, Pool, Repository, Row, Time, New(), scanUser(), CreateUserParams (+2 more)

### Community 17 - "Career Repository"
Cohesion: 0.24
Nodes (10): Context, DriverWithContext, Pool, Repository, Row, Time, New(), scanCareerPath() (+2 more)

### Community 18 - "Notification Handler"
Cohesion: 0.21
Nodes (10): Handler, Request, ResponseWriter, Service, New(), FromRequest(), Request, NewMeta() (+2 more)

### Community 19 - "Pkg Crypto Crypto"
Cohesion: 0.27
Nodes (9): Handler, Request, ResponseWriter, Service, New(), sniffCurriculumMime(), writeServiceError(), NotImplemented() (+1 more)

### Community 21 - "DB Design: Compat Migration Rules"
Cohesion: 0.17
Nodes (17): Backward-compatible migration rules, PostgreSQL-Neo4j eventual consistency model, (:Exam) Neo4j node label, Neo4j 5.x Enterprise (AuraDB af-south-1), Four-store polyglot persistence strategy, PostgreSQL 16 (AWS RDS af-south-1), (:Region) Neo4j node label, [:ATTEMPTED] relationship (+9 more)

### Community 22 - "Assessment Handler"
Cohesion: 0.38
Nodes (7): decode(), Handler, Request, ResponseWriter, Service, New(), WriteError()

### Community 23 - "Student Repository"
Cohesion: 0.27
Nodes (10): Context, DriverWithContext, Pool, Repository, Row, Time, New(), scanStudent() (+2 more)

### Community 24 - "School Repository"
Cohesion: 0.29
Nodes (9): Context, Pool, Repository, Row, Time, New(), scanSchool(), CreateSchoolParams (+1 more)

### Community 25 - "Sync Dto"
Cohesion: 0.21
Nodes (11): Time, Context, Repository, Service, Time, New(), ChangeItem, PulledChange (+3 more)

### Community 26 - "Compose Ai Service"
Cohesion: 0.17
Nodes (14): AI Service (Python/FastAPI), API (Go), API_PROXY_TARGET routing fix, Flyway (DB Migrations), Frontend (React/Vite), Neo4j Graph Database, Ollama (LLM), PostgreSQL + pgvector (+6 more)

### Community 27 - "Frontend: NPM Dependencies"
Cohesion: 0.13
Nodes (15): eslint, devDependencies, eslint, postcss, @testing-library/jest-dom, @typescript-eslint/eslint-plugin, @vitejs/plugin-react, vitest (+7 more)

### Community 28 - "Src Types Api"
Cohesion: 0.13
Nodes (14): ApproveRequest, ApproveResponse, AuthResponse, Envelope, JobStatus, JobStatusValue, LoginRequest, ParsedClo (+6 more)

### Community 29 - "Main Startup Event"
Cohesion: 0.26
Nodes (11): Time, UUID, ApproveRequest, ApproveResponse, JobStatus, ParsedCLO, ParsedStructurePayload, ParsedTopic (+3 more)

### Community 30 - "Auth Handler"
Cohesion: 0.40
Nodes (7): decode(), Handler, Request, ResponseWriter, Service, New(), BadRequest()

### Community 31 - "Region Repository"
Cohesion: 0.31
Nodes (8): Context, Pool, Repository, Row, Time, New(), scanRegion(), Region

### Community 32 - "Frontend: NPM Scripts"
Cohesion: 0.14
Nodes (14): scripts, build, coverage, dev, format, lint, lint:fix, playwright (+6 more)

### Community 33 - "DB Design Doc: 1 Database Architecture Overview"
Cohesion: 0.28
Nodes (7): DriverWithContext, main(), runMigrations(), splitStatements(), DriverWithContext, NewDriver(), Neo4jConfig

### Community 34 - "Region Handler"
Cohesion: 0.38
Nodes (6): decode(), Handler, Request, ResponseWriter, Service, New()

### Community 35 - "Pkg Config Config"
Cohesion: 0.33
Nodes (9): getenv(), getenvInt(), Duration, Load(), AWSConfig, Config, JWTConfig, PostgresConfig (+1 more)

### Community 36 - "Frontend: API Client"
Cohesion: 0.22
Nodes (8): apiClient, RetriableConfig, approveCurriculumJob(), getCurriculumJob(), login(), unwrap(), uploadCurriculum(), UploadCurriculumPayload

### Community 37 - "Impl Plan Doc: The Fundamental Dependency Chain"
Cohesion: 0.31
Nodes (6): Context, Pool, ReadCloser, Reader, NewPostgresStorage(), PostgresStorage

### Community 38 - "Main Neo4jdriver Wiring"
Cohesion: 0.18
Nodes (12): cmd/api/main.go — curriculumrepo.New(pgPool) to curriculumrepo.New(pgPool, neo4jDriver), Upsert-not-insert design so re-approving a job updates rather than duplicates rows, Migration V016 — unique constraints units(subject_code,number), topics(unit_id,sequence_order), internal/career/repository (existing Neo4j MERGE pattern, referenced), ApproveResponse DTO — gained graphSynced/graphSyncError fields, POST /api/v1/curriculum/jobs/{id}/approve — promotes parsed tree into curriculum.subjects/units/topics/clos, repository.ApproveAndPromote — Postgres promotion transaction, then Neo4j sync after commit, repository.syncCurriculumGraph — MERGEs Subject/Unit/Topic + HAS_UNIT/HAS_TOPIC into Neo4j (+4 more)

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
Cohesion: 0.17
Nodes (9): indexRoute, jobReviewRoute, loginRoute, Register, rootRoute, router, routeTree, @tanstack/react-router (+1 more)

### Community 44 - "Cmd Api Main"
Cohesion: 0.20
Nodes (11): ComparePassword(), Duration, HashPassword(), loadPrivateKey(), loadPublicKey(), NewJWTSigner(), Claims, JWTSigner (+3 more)

### Community 45 - "Career Handler"
Cohesion: 0.26
Nodes (10): decode(), Handler, Request, ResponseWriter, Service, New(), ResponseWriter, WriteJSON() (+2 more)

### Community 46 - "Jobs Handler"
Cohesion: 0.36
Nodes (7): decode(), Handler, Request, ResponseWriter, Service, New(), UserID()

### Community 47 - "Storage Dto"
Cohesion: 0.36
Nodes (7): Any(), Error(), Logger, Int(), New(), String(), Field

### Community 48 - "Impl Plan Doc: (root)"
Cohesion: 0.29
Nodes (5): main(), EnsureDevKeyPair(), InitTracer(), OTELConfig, TracerProvider

### Community 49 - "Values Ai Service"
Cohesion: 0.20
Nodes (11): AI Service (Helm default values), API Service (Helm default values), API Service (Production values override: 5 replicas), ai-service (School Box docker-compose), api service (School Box docker-compose), neo4j (neo4j:5.18-community, School Box), ollama (ollama/ollama:latest, School Box), postgres (pgvector/pgvector:pg16, School Box) (+3 more)

### Community 50 - "Ministry Dto"
Cohesion: 0.29
Nodes (6): Context, Repository, Service, New(), OverviewResponse, RegionStatsResponse

### Community 51 - "Storage Handler"
Cohesion: 0.30
Nodes (8): decode(), Handler, Request, ResponseWriter, Service, New(), As(), Struct()

### Community 52 - "DB Design: Regions Regional Aggregates"
Cohesion: 0.29
Nodes (10): regions.regional_aggregates table, regions.regions table, regions schema, schools.quality_scores table, schools schema, schools.schools table, sync schema, sync.school_box_heartbeats table (+2 more)

### Community 53 - "DB Design Doc: (root)"
Cohesion: 0.29
Nodes (7): Context, Repository, Service, New(), PresignDownloadRequest, PresignResponse, PresignUploadRequest

### Community 54 - "DB Design Doc: Node Labels And Properties"
Cohesion: 0.39
Nodes (5): Handler, Request, ResponseWriter, Service, New()

### Community 55 - "Frontend: NPM Dependencies"
Cohesion: 0.22
Nodes (9): ag-grid-community, dependencies, ag-grid-community, react, react-dom, tailwind-merge, react, react-dom (+1 more)

### Community 56 - "Ministry Repository"
Cohesion: 0.36
Nodes (6): Context, Pool, Repository, New(), Overview, RegionStats

### Community 58 - "DB Design: Design Clo Node"
Cohesion: 0.25
Nodes (9): (:CLO) Neo4j node label, [:ANSWERED] relationship, [:HAS_SUBTOPIC] relationship, [:MAPS_TO_CLO] relationship, [:MASTERED] relationship, [:PREREQUISITE_OF] relationship, [:STRUGGLED_WITH] relationship, (:Student) Neo4j node label (+1 more)

### Community 59 - "DB Design: Design Curriculum Clos"
Cohesion: 0.47
Nodes (9): curriculum.clos table, curriculum schema, curriculum.subjects table, curriculum.topic_clo_mappings table, curriculum.topic_prerequisites table, curriculum.topics table, curriculum.units table, curriculum.upload_jobs table (+1 more)

### Community 60 - "DB Design: Audit Logs Bucket"
Cohesion: 0.22
Nodes (9): edugraph-audit-logs S3 bucket, edugraph-curriculum-docs S3 bucket, edugraph-exam-files S3 bucket, edugraph-exports S3 bucket, edugraph-model-artifacts S3 bucket, edugraph-reports S3 bucket, AWS S3 (af-south-1), edugraph-student-submissions S3 bucket (+1 more)

### Community 62 - "redis.py"
Cohesion: 0.33
Nodes (6): brpop_job(), close_redis(), get_redis(), Redis access for the AI service.  The Go backend pushes a plain job id string on, Block for up to `timeout` seconds waiting for a job id on `queue`.     Returns N, Redis

### Community 63 - "Ministry Handler"
Cohesion: 0.39
Nodes (5): Handler, Request, ResponseWriter, Service, New()

### Community 64 - "Storage Repository"
Cohesion: 0.39
Nodes (5): Context, Duration, Repository, New(), PresignClient

### Community 65 - "Sync Handler"
Cohesion: 0.40
Nodes (3): main(), Pool, NewPool()

### Community 66 - "lifespan"
Cohesion: 0.18
Nodes (10): lifespan(), FastAPI startup event (launches curriculum_worker.run_forever as background task), Consumes curriculum-upload jobs off Redis and parses them.  The Go backend pushe, request_shutdown(), run_forever(), Decision to use plain Redis list consumer instead of Celery task (until Go side pushes Celery-formatted messages), celery==5.4.0 (async task queue dependency, currently unused by curriculum_worker), redis[hiredis]==5.0.7 (queue + BRPOP dependency) (+2 more)

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
Cohesion: 0.39
Nodes (4): IN_PROGRESS_STATUSES, JobReviewPage(), linesToList(), UploadPage()

### Community 71 - "CI Job: Ci Ai Test"
Cohesion: 0.25
Nodes (8): CI: AI Service — Tests, CI: Backend — Unit + Integration Tests, CI: Build Docker Images, CI: Frontend — Unit Tests, Deploy Production: Check Deployment Prerequisites, Deploy to Production (canary rollout), Deploy Staging: Check Deployment Prerequisites, Deploy to Staging

### Community 72 - "StorageProvider Interface / Dual-Storage System"
Cohesion: 0.52
Nodes (7): cmd/api/main.go — wires NewPostgresStorage into curriculumsvc.New, GET /api/v1/storage/files/{jobId} — dev-mode file proxy, StorageProvider interface (Upload/Download), PostgresStorage (StorageProvider impl over storage.local_files BYTEA), StorageProvider Interface / Dual-Storage System, Step 1: The Upload (Go API + Dual Storage), Step 3: The Human Review (Frontend + Go API)

### Community 73 - "dto.go"
Cohesion: 0.33
Nodes (6): Time, AuthResponse, LoginRequest, RefreshRequest, RegisterRequest, UserResponse

### Community 74 - "Capability 1A — Curriculum Document Ingestion Pipeline"
Cohesion: 0.57
Nodes (7): Capability 1A — Curriculum Document Ingestion Pipeline, Capability 1B — Neo4j Curriculum Graph Construction, Capability 1C — CLO Matching and Verification, Capability 1D — Cross-Grade Prerequisite Graph, Critical Path Items (Section 7.1), MOE National CLO Document (Ministry external input), Phase 1 — Integration Test Suite (QA Gate)

### Community 75 - "curriculum handler.Upload — fixed wrong context key + double body-read bugs"
Cohesion: 0.40
Nodes (5): Added curriculum_officer to user_role enum + registration validator, contextkeys.UserIDKey (typed context key for authenticated user id), curriculum handler.Upload — fixed wrong context key + double body-read bugs, middleware.UserID(ctx) helper, pkg/validator (request field validation)

### Community 76 - "autoprefixer"
Cohesion: 0.67
Nodes (3): getEnv(), main(), DemoUser

### Community 78 - "DB Design: Bullmq Job Queues"
Cohesion: 0.33
Nodes (7): Bull MQ job queues (bull:*), Redis API response cache keys (cache:*), Redis distributed locks (lock:*), Redis WebSocket pub-sub channels (ws:*), Redis rate-limit counters (ratelimit:*, Kong plugin), Redis session keys (session:{userId}), Redis 7 (ElastiCache af-south-1)

### Community 82 - "Architecture Doc: 1 Exam Assessment Verifier"
Cohesion: 0.33
Nodes (6): 5.3.1 Exam Assessment Verifier, 5.3.2 Learning Gap Engine, 5.3.3 Study Plan Generator, 5.3.4 Career Recommendation Engine, 5.3.5 Policy Intelligence (Ministry), 5.3 AI Capabilities

### Community 83 - "App Core Config"
Cohesion: 0.40
Nodes (3): asyncpg connection string built from the individual settings above., Settings, BaseSettings

### Community 85 - "DB Design: Design Career Node"
Cohesion: 0.50
Nodes (5): (:Career) Neo4j node label, careers.career_topic_requirements table, careers.careers table, careers schema, [:REQUIRES] relationship

### Community 86 - "DB Design: Students Gap Records"
Cohesion: 0.80
Nodes (5): students.gap_records table, students.mastery_records table, students schema, students.student_profiles table, students.study_plans table

### Community 87 - "Frontend Package"
Cohesion: 0.40
Nodes (4): name, private, type, version

### Community 88 - "Architecture Doc: 1 Multi Tenant Isolation"
Cohesion: 0.40
Nodes (5): 6.1.1 Multi-Tenant Isolation, 6.1.2 Core Table Groups, 6.1.3 pgvector Extension, 6.1.4 PostgreSQL Configuration, 6.1 PostgreSQL — Primary Transactional Database

### Community 94 - "DB Migration V003: regions and schools"
Cohesion: 0.67
Nodes (3): regions, schools, users

### Community 95 - "DB Migration V006: assessments"
Cohesion: 0.83
Nodes (3): assessment_questions, assessment_results, assessments

### Community 97 - "Lib Validations Curriculum"
Cohesion: 0.50
Nodes (3): ACCEPTED_TYPES, UploadCurriculumFormValues, uploadCurriculumSchema

### Community 99 - "Architecture Doc: Node Types And Properties"
Cohesion: 0.50
Nodes (4): 6.2.1 Node Types and Properties, 6.2.2 Relationship Types, 6.2.3 Neo4j Configuration, 6.2 Neo4j — Curriculum Knowledge Graph

### Community 100 - "Architecture Doc: 1 1 Node Groups"
Cohesion: 0.50
Nodes (4): 8.1.1 Node Groups, 8.1.2 Namespace Layout, 8.1.3 Pod Autoscaling, 8.1 Kubernetes Cluster Design

### Community 101 - "Architecture Doc: 2 1 Role Hierarchy"
Cohesion: 0.50
Nodes (4): 9.2.1 Role Hierarchy, 9.2.2 Row-Level Security Enforcement, 9.2.3 Neo4j Authorization, 9.2 Authorization — Role-Based Access Control

### Community 119 - "Architecture Doc: 2 1 Cache Layers"
Cohesion: 0.67
Nodes (3): 10.2.1 Cache Layers, 10.2.2 Cache Invalidation, 10.2 Caching Strategy

### Community 120 - "Architecture Doc: 2 0 Openid Connect"
Cohesion: 0.67
Nodes (3): 9.1.1 JWT + OAuth 2.0 / OpenID Connect, 9.1.2 Multi-Factor Authentication, 9.1 Authentication

### Community 147 - "Claude"
Cohesion: 0.25
Nodes (7): Architecture, Critical Decisions, Database Structure, Development Rules, graphify, Important Files, Key Components — The Curriculum Pipeline (Phase 1)

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
- **341 isolated node(s):** `users`, `students`, `teachers`, `notifications`, `sync_logs` (+336 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **102 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

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
- **Why does `Internal()` connect `Authresponse` to `Assessment Dto`, `Ai Careermatchrequest`, `Teacher Dto`, `Jobs Dto`, `Notification Dto`, `Ministry Dto`, `Pkg Crypto Crypto`, `DB Design Doc: (root)`, `Assessment Handler`, `Sync Dto`?**
  _High betweenness centrality (0.111) - this node is a cross-community bridge._
- **Why does `WriteError()` connect `Assessment Handler` to `Authresponse`, `Region Handler`, `School Handler`, `Student Handler`, `Teacher Handler`, `Api Handlers`, `Career Handler`, `Jobs Handler`, `Notification Handler`, `Pkg Crypto Crypto`, `Storage Handler`, `DB Design Doc: Node Labels And Properties`, `Auth Handler`, `Ministry Handler`?**
  _High betweenness centrality (0.072) - this node is a cross-community bridge._
- **Why does `newRouter()` connect `Api Handlers` to `Impl Plan Doc: (root)`, `Pkg Config Config`, `Career Handler`?**
  _High betweenness centrality (0.018) - this node is a cross-community bridge._