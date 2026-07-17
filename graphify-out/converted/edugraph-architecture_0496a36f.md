<!-- converted from edugraph-architecture.docx -->





EduGraph AI
Full System Architecture Document
National Educational Intelligence Platform — Ethiopia

Version 1.0  ·  June 2026  ·  AASTU Innovation Program
CONFIDENTIAL — For internal and government review only







# Table of Contents


## 1.1 Purpose
This document describes the complete system architecture of EduGraph AI — a national educational intelligence platform built for Ethiopia's K-12 public education system.
It covers every layer of the system: frontend client applications, API gateway and backend services, AI microservices, database architecture, offline sync infrastructure, infrastructure and orchestration, security controls, and operational monitoring. The document is intended for engineering teams, government technical reviewers, infrastructure partners, and security auditors.
## 1.2 Scope
EduGraph AI serves six stakeholder tiers simultaneously: Students (26M+), Teachers (500K+), School Administrators (50,000+ schools), Regional Education Bureaus, Ministry of Education, and Career Intelligence consumers. The system operates both online (cloud-hosted) and fully offline (via School Box hardware nodes) to serve rural schools with zero internet connectivity.
## 1.3 Core Design Principles
- Offline-first: every feature works without internet, sync happens opportunistically
- Security-by-default: row-level isolation, encrypted at rest and in transit, audit trails on all access
- Horizontal scalability: stateless services, Kubernetes autoscaling, no single point of failure
- Data sovereignty: all student PII stored in-country, East Africa region, compliant with Ethiopian data protection law
- Separation of concerns: AI inference isolated from transactional data, each service owns its domain
- Immutable infrastructure: Terraform-managed, GitOps-deployed, no manual server changes
## 1.4 Technology Stack Summary



## 2.1 Application Structure
EduGraph AI's frontend is a single React 18 + TypeScript application served as a progressive web app (PWA). It presents five distinct dashboard experiences depending on the authenticated user's role, all within a single codebase using role-based route guards.
## 2.2 Technology Stack
## 2.3 Role-Based Dashboard Routing
Each stakeholder role gets a dedicated route namespace, lazy-loaded on demand. Route guards check the JWT role claim before rendering.
/student/*         → Student learning dashboard
/teacher/*         → Assessment intelligence dashboard
/school-admin/*    → Quality and compliance monitor
/regional/*        → Regional performance analytics
/ministry/*        → National intelligence dashboard
/career/*          → Career intelligence portal
## 2.4 State Management Strategy
- Server state (API data): TanStack Query with stale-while-revalidate. Cache keys scoped by user ID and tenant.
- Form state: React Hook Form + Zod schemas. All forms validated client-side before submission.
- UI state (modals, sidebar, theme): Zustand atoms. Never mixed with server state.
- Offline state: IndexedDB via Dexie.js for locally cached student responses. Syncs via ElectricSQL on reconnect.
## 2.5 Offline PWA Capability
The PWA operates at three levels of offline support:
- Level 1 — Static shell: all UI assets cached by Workbox. App loads and navigates offline.
- Level 2 — Read cache: last-fetched data for the student's own profile, study plan, and gap analysis is persisted in IndexedDB. Dashboard renders stale data with a banner indicating offline mode.
- Level 3 — Write queue: student exam submissions and answers written to IndexedDB. On reconnect, TanStack Query's background mutation queue flushes these to the API in order.
## 2.6 Performance Targets


## 3.1 Gateway Role
Kong Gateway is the single ingress point for all client traffic. No service is reachable from the public internet except through Kong. It handles: SSL/TLS termination, JWT validation, rate limiting, request routing, CORS, IP allowlisting for Ministry routes, and audit logging of every request.
## 3.2 Kong Plugins Enabled
## 3.3 Routing Rules
/api/v1/student/*     → Go Core API  :8080/student
/api/v1/teacher/*     → Go Core API  :8080/teacher
/api/v1/admin/*       → Go Core API  :8080/admin
/api/v1/ministry/*    → Go Core API  :8080/ministry  [IP restricted]
/api/v1/ai/*          → Python AI Service :8000
/api/v1/sync/*        → ElectricSQL Sync Service :5133
/ws/*                 → WebSocket Go API (real-time dashboard updates)
## 3.4 TLS Configuration
- TLS 1.3 minimum. TLS 1.0 and 1.1 explicitly disabled.
- Certificates issued by Let's Encrypt (auto-renewed via cert-manager on Kubernetes).
- HSTS header: max-age=31536000; includeSubDomains; preload
- mTLS enforced between Kong and all upstream services (Kong acts as CA).


## 4.1 Why Go
Go is chosen for the core API over Node.js or Python for three reasons: goroutines handle 50,000+ concurrent WebSocket connections at under 2KB memory per connection (Node.js uses 8–10KB per connection); a single Go binary with no runtime dependency deploys trivially in a minimal container; and Go's type system catches the most common API bugs (null pointer, type mismatch) at compile time, not in production.
## 4.2 Service Structure
The API is structured as a monolith with internal domain packages — not microservices — because the data model is highly relational and separating it would create chatty inter-service calls with no benefit at current scale. It will be split if individual domains exceed 100 req/sec sustained.
cmd/api/main.go              ← entrypoint, dependency injection
internal/student/            ← student domain: profile, gaps, study plan
internal/teacher/            ← teacher domain: exams, assessments, class view
internal/admin/              ← school admin: compliance, quality scores
internal/ministry/           ← ministry: national aggregates, policy engine
internal/auth/               ← JWT issuance, refresh, RBAC middleware
internal/sync/               ← offline sync queue handler
internal/notification/       ← real-time push (WebSocket hub)
internal/storage/            ← S3 presigned URL generation
pkg/db/                      ← PostgreSQL pool (pgx v5)
pkg/graph/                   ← Neo4j driver wrapper
pkg/cache/                   ← Redis client wrapper
## 4.3 Key Libraries
## 4.4 Request Lifecycle
- Kong validates JWT, adds X-User-ID and X-User-Role headers, forwards to Go API.
- Gin middleware extracts user context, sets request-scoped logger with trace ID.
- RBAC middleware checks role claim against endpoint permission map.
- Row-Level Security: every DB query includes WHERE school_id = $userSchoolId or WHERE region_id = $userRegionId, enforced in pgx middleware, not in handler code.
- Handler calls service layer. Service orchestrates: PostgreSQL for transactional data, Neo4j for graph queries, Redis for caching.
- Response serialized to JSON. Zap logs request duration, status, user ID. Prometheus histogram updated.
## 4.5 WebSocket Real-Time Updates
The Ministry and Regional dashboards show live counters (active students, exam submissions in progress). This uses a WebSocket hub pattern:
- Client connects to /ws/{dashboardType}. Kong proxies to the Go API pod that owns the connection.
- When a student submits an exam, the handler publishes a Redis pub-sub event to channel dashboard:ministry and dashboard:region:{regionId}.
- Every Go API pod subscribes to Redis pub-sub. On receiving an event, the pod's hub broadcasts to all connected WebSocket clients interested in that channel.
- This scales horizontally: any pod can receive a submission, and the Redis fan-out ensures all pods' connected clients are updated.
## 4.6 Concurrency and Connection Pooling


## 5.1 Why a Separate AI Service
The AI service is isolated from the Go core API because: GPU inference requires different hardware (GPU nodes vs. CPU nodes); Python owns the ML ecosystem (LangChain, Hugging Face, sentence-transformers); LLM calls are inherently slow (2–30s) and should never block the core API event loop; and the AI service can be scaled, updated, or replaced independently without touching student data handling.
## 5.2 Technology Stack
## 5.3 AI Capabilities
### 5.3.1 Exam Assessment Verifier
Teacher uploads exam PDF/DOCX. The AI service: extracts questions using PyMuPDF; embeds each question using multilingual-e5-large; queries pgvector for nearest CLO by cosine similarity; returns coverage report showing which CLOs are covered, which are missing, and bloom's taxonomy distribution.
ℹ  Threshold: cosine similarity ≥ 0.82 is considered a CLO match. Below 0.65 flagged as unverifiable. Teacher reviews edge cases.
### 5.3.2 Learning Gap Engine
Receives student ID. Queries Neo4j for failed questions → CLOs → topics → prerequisite chains. LLM (Qwen-7B via Ollama) generates plain-language explanation of each gap in the student's language. Returns structured gap object with severity score, prerequisite depth, and recommended study order (topological sort from Neo4j GDS).
### 5.3.3 Study Plan Generator
Takes gap list + student's exam schedule + estimated hours per day. LLM assembles a day-by-day study schedule. Constraints: topics with earlier exam dates are prioritized; prerequisite topics always precede dependent ones; daily load capped at student's declared availability. Output is a JSON schedule rendered by the React frontend as a calendar view.
### 5.3.4 Career Recommendation Engine
Queries Neo4j: MATCH (career)-[:REQUIRES]->(topic) for all careers. Scores each career by mastered/required ratio. Returns top 5 careers with readiness score, gap topics, and a focused study plan for the student's top match.
### 5.3.5 Policy Intelligence (Ministry)
Aggregates Neo4j graph traversals across all schools and regions. LLM generates natural-language policy insights: 'Grade 11 Physics CLO coverage nationwide dropped 8% this semester, concentrated in SNNPR and Sidama regions.' Output feeds the Ministry intelligence dashboard and monthly automated reports.
## 5.4 Offline AI on School Box
When a school operates offline, the Go API on the School Box routes AI requests to the local Ollama instance instead of the cloud AI service. The School Box runs Qwen2.5-7B (quantized Q4_K_M, 4.5GB RAM) on a CPU or optional GPU node. Gap explanations, study plan generation, and CLO Q&A are all available offline. Embedding generation uses a locally cached multilingual-e5-large model.
⚠  School Box AI inference is 3–8× slower than GPU cloud inference. The frontend shows a 'Local AI — processing' indicator with longer expected wait times.
## 5.5 AI Service Endpoints
POST /ai/verify-exam            → CLO coverage report
POST /ai/gap-analysis           → Student gap chain with plain-language explanation
POST /ai/study-plan             → Personalized study schedule
POST /ai/career-match           → Career readiness scores + focused plan
POST /ai/policy-insight         → Ministry/Regional insight generation
GET  /ai/jobs/{jobId}           → Poll async job status (Celery task)
GET  /ai/health                 → GPU availability, model loaded status


## 6.1 PostgreSQL — Primary Transactional Database
PostgreSQL 16 is the system of record for all transactional data: student profiles, exam submissions, grade records, school enrollment, user authentication, and audit logs. It is the authoritative source for everything that requires ACID guarantees.
### 6.1.1 Multi-Tenant Isolation
All tables include a tenant_id column (UUID). Row-Level Security (RLS) is enabled on every student-scoped table. This means even a misconfigured API query physically cannot return data from another tenant — the isolation is enforced at the PostgreSQL engine level.
-- RLS policy example
CREATE POLICY school_isolation ON student_answers
USING (school_id = current_setting('app.current_school_id')::uuid);

-- Set in pgx middleware before every query
SET LOCAL app.current_school_id = '$schoolId';
### 6.1.2 Core Table Groups
### 6.1.3 pgvector Extension
The pgvector extension is installed for embedding-based CLO matching. CLO descriptions are pre-embedded using multilingual-e5-large and stored in the clo_embeddings table. Exam question embeddings are compared against these at verification time.
SELECT clo_id, 1 - (embedding <=> $questionEmbedding) AS similarity
FROM clo_embeddings
ORDER BY embedding <=> $questionEmbedding
LIMIT 5;
### 6.1.4 PostgreSQL Configuration
## 6.2 Neo4j — Curriculum Knowledge Graph
Neo4j 5.x Enterprise stores the curriculum intelligence graph: subjects, topics, CLOs, questions, careers, and all relationships between them. Its value over PostgreSQL is variable-depth traversal — finding prerequisite chains, CLO coverage gaps, and career readiness scores without complex recursive SQL.
### 6.2.1 Node Types and Properties
### 6.2.2 Relationship Types
### 6.2.3 Neo4j Configuration
## 6.3 Redis — Cache, Queue, and Pub-Sub
Redis cluster mode: 3 primary shards × 1 replica each (AWS ElastiCache for Redis 7, cluster mode enabled). af-south-1.
## 6.4 Object Storage — AWS S3


## 7.1 The Problem
Over 60% of Ethiopian public schools have no reliable internet connection. A purely online system excludes the schools that need EduGraph AI the most. The School Box solves this by making the entire platform — including AI inference — run on local hardware with zero internet dependency.
## 7.2 School Box Hardware Spec
## 7.3 School Box Software Stack
The School Box runs the same software stack as the cloud, containerized in Docker Compose:
docker-compose.yml (School Box)
edugraph-api        ← Go API binary (same image as cloud)
postgres            ← PostgreSQL 16 (local copy of school data only)
neo4j               ← Neo4j Community (subset of curriculum graph)
redis               ← Redis 7 (local cache and queue)
ollama              ← Local LLM inference
electric-sync       ← ElectricSQL sync agent
caddy               ← Local HTTPS reverse proxy (self-signed cert)
watchtower          ← Auto-pulls updated images when internet available
## 7.4 Data Scope on School Box
The School Box does not hold a full copy of the national database — only the data relevant to its school:
- Full curriculum graph: all subjects, topics, CLOs, questions for the school's grade levels.
- Student data: only students enrolled in that school. No cross-school data.
- AI models: the currently deployed LLM (Qwen2.5-7B Q4), embedding model, and CLO embeddings.
- School-level aggregates: quality scores, compliance reports, teacher assessments for this school only.
## 7.5 ElectricSQL Sync — How It Works
ElectricSQL is a Postgres sync engine built on CRDTs (Conflict-free Replicated Data Types). It provides bidirectional sync between the School Box PostgreSQL and the cloud PostgreSQL:
- The School Box operates fully offline, writing student answers, teacher exams, and school events to local PostgreSQL.
- When internet connectivity is detected (periodic ping to the cloud), the ElectricSQL sync agent initiates a sync session.
- ElectricSQL uses the Satellite protocol: each client maintains a logical clock (Lamport timestamp). On reconnect, the cloud and School Box exchange their operation logs since the last successful sync.
- CRDT merge rules handle conflicts: last-write-wins on scalar fields (grade score updated by teacher), set-union on append-only fields (student answers, audit log entries). Conflicts are logged to the sync_conflict table for human review.
- After sync, Neo4j graph data is updated via a Go sync worker that translates PostgreSQL change events (via logical replication slot) into Neo4j Cypher write operations.
## 7.6 Conflict Resolution Rules
## 7.7 School Box Update Mechanism
The Watchtower container runs on the School Box and polls Docker Hub (private registry) for updated images. When internet is available: new Go API image, updated Ollama model files, and Neo4j schema migrations are pulled and applied automatically during a low-usage window (02:00–04:00 local time). A canary rollout is used: the School Box stays on N-1 version until the cloud has verified the new version for 48 hours.


## 8.1 Kubernetes Cluster Design
The entire cloud system runs on Kubernetes 1.29, hosted on AWS EKS in the af-south-1 (Cape Town) region. An additional read-only DR cluster exists in eu-west-1 (Ireland) for disaster recovery.
### 8.1.1 Node Groups
### 8.1.2 Namespace Layout
edugraph-prod          ← all production services
edugraph-staging        ← staging environment (identical config, smaller instances)
edugraph-monitoring     ← Prometheus, Grafana, AlertManager
edugraph-security       ← Vault agent, OPA Gatekeeper
kong                    ← Kong Gateway and its DB
cert-manager            ← TLS certificate automation
### 8.1.3 Pod Autoscaling
Horizontal Pod Autoscaler (HPA) governs each service:
## 8.2 Terraform Infrastructure as Code
All cloud resources are defined in Terraform. No manual changes to any infrastructure resource are permitted. The Terraform state is stored in an S3 backend with DynamoDB locking.
infra/
modules/
eks/           ← EKS cluster, node groups, IRSA roles
rds/           ← PostgreSQL RDS, parameter groups, read replica
redis/         ← ElastiCache Redis cluster
s3/            ← All S3 buckets with policies and lifecycle rules
vpc/           ← VPC, subnets, NACLs, security groups
vault/         ← HashiCorp Vault on EKS
waf/           ← AWS WAF rules attached to Kong ALB
envs/
prod/          ← terraform.tfvars for production
staging/       ← terraform.tfvars for staging
## 8.3 CI/CD Pipeline
GitHub Actions drives all deployments. Direct pushes to main are blocked — all changes require a pull request with at least one review.
### 8.3.1 Pipeline Stages
## 8.4 Helm Chart Structure
helm/edugraph/
Chart.yaml
values.yaml           ← default values (overridden per env)
values.prod.yaml      ← production overrides
values.staging.yaml   ← staging overrides
templates/
api-deployment.yaml
ai-deployment.yaml
hpa.yaml
ingress.yaml        ← Kong IngressClass
networkpolicies.yaml
serviceaccounts.yaml
configmap.yaml


Security is not a feature — it is a constraint applied at every layer. For a system handling the academic records of 26 million students and national education policy data, any breach is a national crisis.
## 9.1 Authentication
### 9.1.1 JWT + OAuth 2.0 / OpenID Connect
All user sessions are managed via short-lived access tokens (JWT, RS256, 15-minute expiry) and long-lived refresh tokens (stored encrypted in Redis, 7-day TTL). The Go auth service issues and validates tokens. Kong validates the access token on every request.
### 9.1.2 Multi-Factor Authentication
MFA is mandatory for all roles above Student. TOTP (RFC 6238, 30-second window, 6-digit code) via any authenticator app. Backup codes: 8 single-use codes generated at MFA enrollment, stored as bcrypt hashes. SMS fallback available for Ministry users via AWS SNS (Amharic OTP message).
## 9.2 Authorization — Role-Based Access Control
### 9.2.1 Role Hierarchy
### 9.2.2 Row-Level Security Enforcement
RLS is the last line of defense. Even if a role check in Go middleware fails silently (bug or misconfiguration), PostgreSQL RLS prevents the data from being returned. RLS policies are set at the schema level and cannot be bypassed by application code.
-- Set before every query in pgx middleware
SET LOCAL app.current_school_id = $1;
SET LOCAL app.current_region_id = $2;
SET LOCAL app.current_role      = $3;

-- Example policy (students table)
CREATE POLICY rls_students ON students
FOR ALL USING (
school_id = current_setting('app.current_school_id')::uuid
OR current_setting('app.current_role') IN ('regional_bureau','ministry','super_admin')
);
### 9.2.3 Neo4j Authorization
Neo4j Enterprise's built-in RBAC is used. Each role maps to a Neo4j database-level role with fine-grained privilege grants. Ministry queries run against a read-only database replica with no access to individual student nodes — only aggregate projections.
## 9.3 Encryption
## 9.4 Secrets Management — HashiCorp Vault
No secret (database password, API key, private key) is stored in environment variables, Kubernetes Secrets, or source code. All secrets are stored in HashiCorp Vault and injected via the Vault Agent sidecar pattern:
- Each Kubernetes service account has a Vault policy granting read access to only its required secrets.
- Vault Agent runs as a sidecar, authenticates via Kubernetes Service Account token, and writes secrets to a tmpfs volume mounted in the application container.
- Database passwords are dynamic secrets: Vault generates a unique PostgreSQL role for each pod on startup, rotates it every hour. A compromised pod's credentials expire quickly.
- Vault audit log captures every secret access — stored in S3 with Object Lock (immutable).
## 9.5 Network Security
## 9.6 Input Validation and Injection Prevention
- All API inputs validated with go-playground/validator (Go) and Pydantic v2 (Python) before any database operation.
- PostgreSQL: all queries use pgx prepared statements with parameter binding. No string interpolation in SQL ever.
- Neo4j: all Cypher queries use parameterized queries via the Go driver. No string-built Cypher.
- File uploads: MIME type validated server-side (magic bytes), not by Content-Type header. Max file sizes enforced at Kong.
- Exam documents scanned with ClamAV before processing. Malware-infected uploads rejected and quarantined in S3.
## 9.7 Audit Logging
Every data access and mutation is logged. Audit logs are immutable (S3 Object Lock, Governance mode, 7-year retention) and cannot be deleted or modified by application code.
## 9.8 Vulnerability Management
- Trivy runs on every Docker image build. Critical CVEs block deployment. High CVEs require tracking issue within 24 hours.
- govulncheck runs daily on all Go modules. Dependabot PRs for all dependency updates with automated merge on patch-level changes.
- Penetration test: annual third-party pentest + quarterly automated DAST (OWASP ZAP) against staging environment.
- Bug bounty program planned pre-national-launch: coordinated disclosure policy published, scope covering all EduGraph AI-hosted services.


## 10.1 Expected Traffic Profile
## 10.2 Caching Strategy
### 10.2.1 Cache Layers
### 10.2.2 Cache Invalidation
Cache invalidation is event-driven via Redis pub-sub. When a teacher submits a graded exam: the Go API publishes an invalidation event to the student:{studentId}:gap channel. The Redis cache key for that student's gap analysis is deleted. The next request regenerates the gap from the database and re-caches. Ministry aggregate caches are invalidated on a TTL basis — no event-driven invalidation needed at that aggregation level.
## 10.3 Database Scaling
## 10.4 Failover and High Availability
## 10.5 Results Day Surge Handling
National exam results release is the single highest-traffic event. The system pre-provisions for this:
- 48 hours before results release: HPA minimum replicas raised to 12 for Go API, CDN cache pre-warmed with static assets.
- Results data pre-computed: individual student result objects generated as JSON and cached in Redis during off-peak hours the night before. On results day, API reads from Redis, not from PostgreSQL — zero DB load.
- Queue smoothing: if all 200,000 students hit the site simultaneously, a Redis-backed queue serves result requests at 10,000/second, returning a 'Your result is loading' response after 2 seconds rather than timing out.
- Graceful degradation: if Redis cache is exhausted, the API falls back to PostgreSQL read replica with a 5-second response time rather than returning errors.


## 11.1 Observability Stack
## 11.2 Service Level Objectives
## 11.3 Key Alerts


## 12.1 Data Sovereignty
All student PII and educational records are stored exclusively in the AWS af-south-1 (Cape Town) region. No student data is transferred to any region outside Africa without explicit Ministry of Education authorization. This satisfies Ethiopian data protection requirements and aligns with the African Union Convention on Cyber Security and Personal Data Protection.
## 12.2 PII Classification
## 12.3 Data Retention Policy
## 12.4 Right to Access and Erasure
The system supports data subject rights under Ethiopian law:
- Right to access: students and parents can request a full data export (JSON/PDF) of all records via a verified request to the school admin dashboard. Generated asynchronously and delivered via secure download link.
- Right to erasure: deletion of a student's PII is handled by a pseudo-anonymization function — name, national ID, and contact fields are replaced with a deterministic hash. Academic records are retained (required by law) but unlinked from personal identity. True erasure available only after retention period expires.
- Ministry and school admin erasure requests require dual approval (two authorized officers) and generate an immutable audit record even if the data is subsequently deleted.


## 13.1 Environment Promotion
All code flows through: feature branch → staging (automatic on merge to main) → production (manual gate, requires senior engineer approval). Database migrations run automatically via Flyway as part of the deployment. Migrations are always backward-compatible — no breaking schema changes that would require downtime.


## ADR-001: Go over Node.js for Core API
Decision: Use Go for the primary backend API. Considered: Node.js (Express/Fastify), Python (FastAPI), Java (Spring Boot).
Rationale: Go's goroutine model handles 50,000+ concurrent WebSocket connections at <2KB memory/connection — Node.js requires 8–10KB. Go compiles to a single binary with no runtime, reducing container size and startup time. Go's static typing catches bugs at compile time. The tradeoff is a smaller hiring pool than Node.js, mitigated by Go's simplicity.
## ADR-002: Neo4j alongside PostgreSQL
Decision: Use Neo4j for the curriculum knowledge graph; PostgreSQL for all transactional data. Considered: PostgreSQL-only with recursive CTEs, Apache AGE (graph extension for PG).
Rationale: Variable-depth prerequisite traversal (MATCH (topic)-[:PREREQUISITE_OF*1..5]->) is a graph-native operation. PostgreSQL recursive CTEs for this become fragile and slow at depth >3. Neo4j GDS library provides native topological sort for study plan ordering. The operational cost of maintaining two databases is offset by the query expressiveness gained. Apache AGE was considered but is less mature and would require PostgreSQL expertise in graph query optimization.
## ADR-003: ElectricSQL for offline sync over custom solution
Decision: Use ElectricSQL for offline-to-cloud sync. Considered: custom sync engine, Couchbase Lite, PowerSync.
Rationale: ElectricSQL is built on Postgres → Postgres replication using proven CRDT semantics. It handles the hardest problem (conflict resolution) with well-tested merge rules. A custom sync solution would take 6–12 months to reach production-grade reliability. PowerSync was the nearest alternative but requires a proprietary cloud service, creating a vendor lock dependency for a national system. ElectricSQL is open-source and self-hostable.
## ADR-004: Python AI service isolation
Decision: Run AI inference in a separate Python FastAPI service rather than embedding it in the Go API. Considered: Go LLM libraries (llm.go), embedding AI calls in Go via subprocess.
Rationale: GPU drivers, CUDA, Ollama, LangChain, and the Hugging Face ecosystem are all Python-first. Attempting to use these from Go via CGO or subprocess calls adds complexity without benefit. Isolation also means AI service failures do not crash the core API, and the AI service can be scaled on GPU nodes independently.
## ADR-005: Kong over AWS API Gateway
Decision: Use Kong Gateway (self-hosted on EKS) over AWS API Gateway. Considered: AWS API Gateway v2, Traefik, Nginx.
Rationale: AWS API Gateway has a 29MB request payload limit (blocks large exam file uploads), $3.50/million requests cost at scale (expensive at 26M students), and limited plugin flexibility. Kong self-hosted has no per-request cost, supports custom Lua plugins, and runs on EKS alongside other services. The operational overhead of managing Kong is offset by the cost savings and flexibility at national scale.



EduGraph AI — System Architecture Document v1.0  ·  June 2026  ·  CONFIDENTIAL
Built for AASTU Innovation Program — Addis Ababa Science and Technology University
| 01 — Document Overview
Purpose, scope, and key design principles |
| --- |
| Layer | Technology | Role |
| --- | --- | --- |
| Frontend | React 18 + TypeScript + Vite | Role-specific dashboards |
| API Gateway | Kong Gateway | Rate limiting, auth, routing |
| Core Backend | Go 1.22 + Gin | REST + WebSocket APIs |
| AI Service | Python 3.12 + FastAPI | LLM, Graph-RAG, gap engine |
| Primary DB | PostgreSQL 16 | Transactional student data |
| Graph DB | Neo4j 5.x Enterprise | Curriculum knowledge graph |
| Cache / Queue | Redis 7 + Bull MQ | Sessions, jobs, pub-sub |
| Object Storage | AWS S3 (af-south-1) | Documents, exam files |
| Offline AI | Ollama (Qwen / Llama) | School Box local inference |
| Sync Engine | ElectricSQL + CRDT | Offline-to-cloud sync |
| Orchestration | Kubernetes 1.29 | Container management |
| IaC | Terraform + Helm | Infrastructure as code |
| CI/CD | GitHub Actions | Automated deploy pipeline |
| Monitoring | Prometheus + Grafana | Observability stack |
| Secrets | HashiCorp Vault | Credentials management |
| 02 — Frontend Architecture
React client applications, state management, and offline capability |
| --- |
| Technology | Purpose / Rationale |
| --- | --- |
| React 18 + TypeScript | Component framework. TypeScript enforced — no `any` allowed in production code |
| Vite 5 | Build tool. Sub-second HMR in development, tree-shaken production bundles |
| TanStack Router v1 | Type-safe routing with built-in code splitting and route-level auth guards |
| TanStack Query v5 | Server state: caching, background refetch, stale-while-revalidate, optimistic updates |
| Zustand | Client state (UI-only state that doesn't belong on the server) |
| Recharts + D3 | Data visualization: line charts, heatmaps, treemaps for dashboards |
| AG Grid (Community) | High-performance data tables for Ministry and Regional dashboards (1M+ row virtual scroll) |
| Tailwind CSS v4 | Utility CSS. Design tokens enforced via a shared `edugraph.config.ts` |
| Radix UI | Accessible headless components (dialogs, dropdowns, tooltips) — WCAG 2.1 AA |
| Zod | Runtime schema validation on all API responses — prevents silent type drift |
| i18next | Internationalization: Amharic, English, Oromo (Afaan Oromoo) |
| Workbox (PWA) | Service worker for offline caching of static assets and critical API responses |
| Metric | Target |
| --- | --- |
| First Contentful Paint | < 1.2s (CDN-cached assets) |
| Largest Contentful Paint | < 2.5s (Core Web Vital) |
| Time to Interactive | < 3.5s on 3G connection |
| Bundle size (initial) | < 180KB gzipped (code split per route) |
| Ministry dashboard 1M rows | < 200ms virtual scroll render (AG Grid) |
| Offline startup | < 0.8s (service worker serves shell instantly) |
| 03 — API Gateway
Kong Gateway — traffic ingress, authentication, and rate limiting |
| --- |
| Plugin | Configuration |
| --- | --- |
| jwt | Validates RS256-signed JWTs. Public key fetched from Vault on startup. |
| rate-limiting-advanced | Per-user: 1000 req/min students, 500 req/min teachers, 200 req/min admins. Burst allowed 2×. |
| ip-restriction | Ministry dashboard (/ministry/*) restricted to MoE IP ranges + VPN CIDR. |
| cors | Allowlist: edugraph.edu.et and school-local-*.edugraph.internal |
| request-size-limiting | Max 50MB (exam file uploads). Larger goes directly to presigned S3 URL. |
| response-ratelimiting | Prevent scraping of bulk student data via Ministry export endpoints. |
| prometheus | Exposes /metrics for Kong itself — request counts, latency P50/P95/P99 per route. |
| file-log | Full request/response audit log to S3 (used for government audit trails). |
| openid-connect | SSO integration with MoE's existing Active Directory via OIDC federation. |
| 04 — Core Backend API
Go + Gin REST and WebSocket service |
| --- |
| Package | Purpose |
| --- | --- |
| gin-gonic/gin | HTTP router. Chosen for performance (radix tree routing) and middleware ecosystem. |
| jackc/pgx v5 | PostgreSQL driver. Native protocol, prepared statements, pgxpool for connection pooling. |
| neo4j/neo4j-go-driver v5 | Official Neo4j driver. Bolt protocol, connection pooling, retry on transient errors. |
| redis/rueidis | Redis client. Pipeline-aware, safer than go-redis for high-throughput batch ops. |
| golang-jwt/jwt v5 | RS256 JWT signing and validation. Keys loaded from Vault via envconsul sidecar. |
| go-playground/validator v10 | Request body validation. Struct tags like `validate:"required,email"`. |
| uber-go/zap | Structured JSON logging. Zero-allocation hot path. Outputs to stdout for k8s log collection. |
| prometheus/client_golang | Exposes /metrics. Custom histograms per endpoint, DB query latency, cache hit rate. |
| gorilla/websocket | WebSocket hub for real-time dashboard updates. One hub per k8s pod, scaled via Redis pub-sub. |
| Resource | Configuration |
| --- | --- |
| PostgreSQL pool size | max_conns: 80 per pod, min_conns: 10. pgxpool manages lifecycle. |
| Neo4j pool size | max_connection_pool_size: 50 per pod. Neo4j driver handles Bolt multiplexing. |
| Redis pool | rueidis manages a single multiplexed connection. No pool needed (pipelining). |
| Goroutine leak prevention | context.WithTimeout on every external call. errgroup for concurrent fan-out queries. |
| Circuit breaker | sony/gobreaker wraps Neo4j and AI service calls. Opens after 5 failures in 10s. |
| 05 — AI Microservice
Python FastAPI — LLM inference, knowledge graph RAG, gap engine |
| --- |
| Technology | Purpose |
| --- | --- |
| FastAPI | Async HTTP framework. Native async/await matches LLM streaming responses. |
| LangChain | Orchestration for RAG pipelines: document loading, chunking, retrieval, prompt assembly. |
| LlamaIndex | Secondary RAG framework used for curriculum graph Q&A over Neo4j-stored CLOs. |
| sentence-transformers | Local embedding generation. Model: intfloat/multilingual-e5-large (supports Amharic). |
| Ollama (local) | School Box inference. Serves Qwen2.5-7B, Llama3.1-8B, Gemma2-9B locally on GPU/CPU. |
| vLLM (cloud) | Cloud inference for high-throughput scenarios. Served on AWS g5.xlarge (NVIDIA A10G GPU). |
| pgvector | Vector similarity search inside PostgreSQL for embedding-based CLO matching. |
| Celery + Redis | Async task queue for long-running AI jobs (generating study plans, batch gap analysis). |
| Pydantic v2 | Request/response validation. Models shared with Go via JSON Schema generation. |
| 06 — Database Architecture
PostgreSQL, Neo4j, Redis, S3 — purpose and schema design |
| --- |
| Schema | Tables |
| --- | --- |
| identity.* | users, roles, sessions, oauth_tokens, mfa_challenges |
| curriculum.* | subjects, topics, clos, topic_prerequisites (mirror of Neo4j for RLS queries) |
| assessment.* | exams, questions, student_answers, exam_attempts |
| students.* | student_profiles, enrollment, digital_twin_snapshots, gap_records |
| schools.* | schools, school_quality_scores, compliance_reports |
| regions.* | regions, region_performance_aggregates |
| ministry.* | national_aggregates, policy_events, curriculum_change_log |
| sync.* | sync_operations, conflict_log, school_box_heartbeats |
| audit.* | access_log, data_change_log, ai_request_log |
| Parameter | Value |
| --- | --- |
| Version | PostgreSQL 16 on AWS RDS (af-south-1 / Cape Town region) |
| Instance | db.r7g.2xlarge (8 vCPU, 64GB RAM) — Multi-AZ with automatic failover |
| Storage | 2TB gp3 SSD, autoscale to 8TB. Encrypted with AWS KMS. |
| Connection pooling | PgBouncer in transaction mode, 200 max server connections per Go API pod set |
| Backups | Continuous WAL archiving to S3. Point-in-time recovery to any 5-minute window, 35 days retention. |
| Read replica | One read replica in same AZ for Ministry report queries (heavy aggregations isolated here). |
| Monitoring | pg_stat_statements enabled. Slow query log (>200ms) to CloudWatch. pgBadger weekly report. |
| Node Label | Key Properties |
| --- | --- |
| (:Subject) | code, name, gradeLevel, description, updatedAt |
| (:Topic) | id, name, subjectCode, estimatedHours, examWeight, bloomLevel |
| (:CLO) | code, description, bloomLevel, gradeLevel, mandatory, createdAt |
| (:Question) | id, text, difficultyLevel, bloomLevel, type, examYear |
| (:Exam) | id, subjectCode, gradeLevel, year, term, totalMarks |
| (:Student) | id, schoolId, gradeLevel — lightweight mirror for graph traversal |
| (:Career) | id, name, sector, requiredEducationLevel, description |
| (:School) | id, name, regionId, type — for school-scoped aggregations |
| Relationship | Direction and Properties |
| --- | --- |
| [:HAS_TOPIC] | Subject → Topic |
| [:MAPS_TO_CLO] | Topic → CLO |
| [:ASSESSES] | Question → CLO  {alignmentScore: float} |
| [:PART_OF] | Question → Exam |
| [:ATTEMPTED] | Student → Exam  {score, attemptDate} |
| [:ANSWERED] | Student → Question  {score, passed, timeSpent} |
| [:MASTERED] | Student → Topic  {masteredAt, confidence} |
| [:STRUGGLED_WITH] | Student → Topic  {severity: 0–1, attempts} |
| [:PREREQUISITE_OF] | Topic → Topic  {weight: float} |
| [:REQUIRES] | Career → Topic  {importance: high/medium/low} |
| [:ENROLLS] | School → Student |
| Parameter | Value |
| --- | --- |
| Deployment | Neo4j AuraDB Enterprise (AWS af-south-1) or self-hosted on r6g.2xlarge |
| Clustering | Causal cluster: 3 core members (one writer, two followers) + 2 read replicas |
| Memory | heap.initial_size=8g, heap.max_size=16g, pagecache.size=32g |
| Indexes | Composite index on (:Student {id, schoolId}), (:CLO {code, gradeLevel}), (:Topic {id, subjectCode}) |
| Backups | Neo4j Admin backup to S3, daily full + incremental. 30-day retention. |
| GDS Plugin | Graph Data Science plugin enabled for topological sort (study plan ordering) and community detection |
| Usage | Detail |
| --- | --- |
| Session store | JWT refresh tokens. TTL: 7 days. Key: session:{userId} |
| API response cache | Ministry aggregate queries cached 5 min. School quality scores cached 1 hour. |
| Rate limit counters | Kong rate-limiting plugin uses Redis. Shared across all Kong nodes. |
| Bull MQ job queue | AI service jobs (study plan generation, batch gap analysis). Persistent, with retry. |
| WebSocket pub-sub | Channels: dashboard:ministry, dashboard:region:{id}. Fan-out to all API pods. |
| Sync operation lock | Distributed lock during School Box sync to prevent write conflicts. TTL: 30s. |
| Bucket / Prefix | Purpose and Policy |
| --- | --- |
| exam-documents/ | Teacher-uploaded exam PDFs. Encrypted (SSE-KMS). Lifecycle: delete after 3 years. |
| student-submissions/ | Student answer sheets (scanned images). Encrypted. Lifecycle: archive to Glacier after 1 year. |
| school-reports/ | Generated quality reports (PDF). Public presigned URLs, 1-hour expiry. |
| sync-snapshots/ | School Box sync state snapshots. Used for disaster recovery of offline nodes. |
| audit-logs/ | Kong + API audit logs. Versioned, WORM-compliant (Object Lock), 7-year retention. |
| model-artifacts/ | Ollama model files for School Box updates. Served over HTTPS to School Boxes. |
| 07 — Offline Architecture (School Box)
Local node design, CRDT sync, and conflict resolution |
| --- |
| Component | Specification |
| --- | --- |
| CPU | AMD Ryzen 7 8-core (or equivalent) — adequate for 4-bit quantized LLM inference |
| RAM | 32GB DDR5 — 16GB reserved for OS/services, 16GB for Ollama model context |
| Storage | 2TB NVMe SSD — OS, Docker, models (5–15GB per model), student data |
| Network | Gigabit Ethernet for LAN serving school devices. Optional WiFi 6 AP. |
| Power | UPS with 4-hour battery backup — handles Ethiopia's frequent power interruptions |
| OS | Ubuntu Server 24.04 LTS minimal install |
| GPU (optional) | NVIDIA RTX 3060 12GB — reduces LLM inference latency from ~8s to ~1.5s |
| Operation | Resolution Rule |
| --- | --- |
| Student answer submitted | Append-only. No conflict possible — answers are immutable once submitted. |
| Teacher grades student answer | Last-write-wins with timestamps. Cloud wins if both modified after last sync. |
| School quality score recalculated | Recomputed from source data after sync — not merged. Source data wins. |
| CLO updated by Ministry | Ministry (cloud) always wins. School Box CLOs are read-only replicas. |
| Student profile updated | School wins for school-scoped fields (enrollment). Ministry wins for national fields. |
| Exam uploaded by teacher | School Box origin wins. Cloud receives the exam as-created. No conflict. |
| 08 — Infrastructure & Orchestration
Kubernetes, Terraform, CI/CD, and cloud configuration |
| --- |
| Node Group | Instance Type and Role |
| --- | --- |
| api-nodes | c7g.xlarge × 3–10 (auto-scale). Runs Go API pods. ARM64, spot instances for 60% cost saving. |
| ai-nodes | g5.xlarge × 1–4 (auto-scale on GPU utilization). Runs Python AI service with NVIDIA A10G GPU. |
| data-nodes | r7g.2xlarge × 2 (on-demand). Runs PgBouncer, Redis, monitoring stack. Memory-optimized. |
| system-nodes | t4g.medium × 2 (on-demand). Cluster-critical: cert-manager, Vault agent, Kong, Watchtower. |
| Service | HPA Configuration |
| --- | --- |
| Go Core API | min: 3, max: 20, scale on CPU >65% or WebSocket connections >800/pod |
| Python AI Service | min: 1, max: 8, scale on GPU utilization >70% or queue depth >50 jobs |
| Kong Gateway | min: 2, max: 6, scale on CPU >60% |
| ElectricSQL | min: 2, max: 4, scale on active sync connections >100 |
| Stage | Details |
| --- | --- |
| 1. Lint & format | golangci-lint (Go), ruff (Python), eslint (TypeScript). PR blocked on lint failure. |
| 2. Unit tests | go test ./... with -race flag. pytest. Vitest for React. Coverage gate: >80% on new code. |
| 3. Integration tests | Testcontainers spins up real PostgreSQL + Neo4j + Redis. Tests run against live DBs in Docker. |
| 4. Security scan | Trivy scans all Docker images for CVEs. govulncheck for Go. snyk for npm dependencies. |
| 5. Build & push | Docker images built with buildx (multi-arch: arm64 + amd64). Tagged with git SHA, pushed to ECR. |
| 6. Staging deploy | Helm upgrade --atomic --timeout 5m. Smoke tests run against staging. Auto-rollback on failure. |
| 7. Prod deploy | Manual approval required. Deploys with 25% canary first, 100% after 10-minute health check. |
| 8. Post-deploy | Datadog synthetic monitor confirms all critical user flows. Slack notification with diff summary. |
| 09 — Security Architecture
Authentication, authorization, encryption, and compliance controls |
| --- |
| Token Type | Detail |
| --- | --- |
| Access token | JWT RS256. Expiry: 15 minutes. Claims: sub, role, schoolId, regionId, iat, exp, jti. |
| Refresh token | UUID v4, stored as SHA-256 hash in Redis with user binding. Expiry: 7 days. Single-use (rotated on each refresh). |
| Ministry SSO | MoE Active Directory via OIDC federation. Kong openid-connect plugin handles the exchange. |
| School Box auth | Local JWT issued by School Box Go API. No cloud call required. Synced revocation list. |
| Token rotation | Silent refresh: frontend TanStack Query retries 401 once with refresh token before logout. |
| Role | Data Access Scope |
| --- | --- |
| student | Own profile, own gap analysis, own study plan, own career recommendations. |
| teacher | Own school's students, own exams, school-level class analytics. Read-only school quality data. |
| school_admin | All data for their school. Quality scores, compliance reports, teacher performance. |
| regional_bureau | All schools and students in their region. No access to other regions. |
| ministry | National aggregates only. Individual student data is anonymized at this level. |
| curriculum_officer | Write access to CLO and curriculum graph. Read-only on student data. |
| super_admin | Internal EduGraph staff only. Full access. All actions double-logged and require 2-person approval for destructive operations. |
| Layer | Mechanism |
| --- | --- |
| In transit (external) | TLS 1.3 minimum. HSTS. Certificate pinning on School Box sync connections. |
| In transit (internal) | mTLS between all Kubernetes services via Istio service mesh. All pod-to-pod traffic encrypted. |
| At rest (PostgreSQL) | AWS RDS encryption with dedicated KMS key (aws/rds). Backups encrypted with same key. |
| At rest (Neo4j) | Volume encryption with AWS EBS KMS. Neo4j native encryption enabled for on-disk store. |
| At rest (Redis) | AWS ElastiCache encryption at rest enabled. TLS in-transit encryption between clients and Redis. |
| At rest (S3) | SSE-KMS on all buckets with dedicated per-bucket KMS keys. S3 Block Public Access enforced. |
| Student PII | Sensitive fields (student name, parent contact) additionally encrypted at application layer with AES-256-GCM before storage. Key stored in Vault. |
| School Box storage | LUKS full-disk encryption on School Box NVMe. Unlock key stored in Vault, fetched on boot over VPN. |
| Control | Implementation |
| --- | --- |
| AWS VPC | All resources in private subnets. No public IPs except Kong ALB and School Box VPN endpoint. |
| Security Groups | Least-privilege: only required ports open. DB security groups accept only from API security group. |
| Network Policies | Kubernetes NetworkPolicy: each pod only communicates with explicitly allowed pods. Default deny. |
| AWS WAF | Attached to Kong ALB: SQL injection, XSS, rate-based rules, AWS Managed Rules (Core Rule Set + Known Bad Inputs). |
| DDoS protection | AWS Shield Standard (included). Shield Advanced for Ministry routes. |
| School Box VPN | WireGuard VPN between School Box and cloud for sync operations. Only sync traffic allowed over VPN. |
| Bastion access | No SSH directly to any node. AWS SSM Session Manager for emergency access. All sessions logged to CloudWatch. |
| Log Type | Contents |
| --- | --- |
| API request log | Kong logs: timestamp, user ID, role, endpoint, HTTP status, response time. PII fields redacted. |
| Data change log | PostgreSQL trigger on all student and grade tables: old values, new values, user ID, timestamp. |
| AI request log | Every LLM call logged: user ID, model used, prompt hash (not content), response time, tokens used. |
| Auth events | Login, logout, MFA success/failure, token refresh, permission denial — all logged. |
| Vault audit | Every secret access: which service, which secret path, timestamp. |
| Schema changes | Flyway migration history — every database schema change recorded with author and timestamp. |
| 10 — Traffic Handling & Scalability
Load profiles, autoscaling, caching strategy, and failover |
| --- |
| Scenario | Expected Load |
| --- | --- |
| Baseline (off-peak) | ~500 concurrent users — evening school staff, after-hours study |
| Peak (exam season) | ~50,000 concurrent users — nationwide exam period |
| Spike (results day) | ~200,000 concurrent users in first 30 minutes — students checking results |
| Ministry bulk export | 1–2 times/month: 1M+ row aggregate queries, 5–15 minutes each |
| AI job queue (daily) | ~100,000 gap analysis jobs / day during school year (batch overnight) |
| School Box sync | ~50,000 School Boxes syncing every 6 hours when online — staggered |
| Cache Layer | Strategy |
| --- | --- |
| CDN (CloudFront) | Static assets (JS, CSS, fonts). TTL: 1 year (content-hash filenames). Global edge caching. |
| API response cache (Redis) | Ministry aggregates: TTL 5 min. School quality scores: TTL 1 hour. Student gap list: TTL 10 min (invalidated on new exam submission). |
| Database query cache | pgBouncer query-level cache for read replicas (Ministry report queries). TTL: 2 minutes. |
| Neo4j result cache | Neo4j 5.x has built-in result caching for repeated read queries. Enabled for curriculum graph traversals. |
| TanStack Query (client) | Client-side stale-while-revalidate. Student dashboard: stale 5 min, refetch 10 min. Ministry: stale 1 min. |
| Layer | Scaling Approach |
| --- | --- |
| PostgreSQL write scale | Single primary sufficient to 500 writes/sec. Connection pooling via PgBouncer prevents connection exhaustion. Vertical scale (r7g.4xlarge) before horizontal sharding. |
| PostgreSQL read scale | One read replica for Ministry report queries. Horizontal read replicas added per 1000 concurrent read users. |
| Neo4j read scale | Two read replicas in the causal cluster absorb graph traversal load. Curriculum graph is read-mostly. |
| Redis scale | Cluster mode with 3 shards. Add shards on memory utilization >70%. |
| Exam season prep | Pre-scale: HPA min replicas raised 2 weeks before national exam. Read replica added. Redis shards pre-warmed. |
| Component | HA Mechanism |
| --- | --- |
| Go API | 3 replicas minimum. K8s pod disruption budget: max 1 unavailable. Anti-affinity across AZs. |
| PostgreSQL | Multi-AZ RDS. Automatic failover in <60s. Clients use pgBouncer which retries on primary failure. |
| Neo4j | Causal cluster: automatic leader election. Followers continue serving reads during leader re-election. |
| Redis | ElastiCache Multi-AZ with automatic failover. Read replicas promoted to primary in <30s. |
| Kong | 2 replicas. Session state in Redis (shared). Either pod handles any request. |
| School Box | Self-contained — continues operating with no cloud dependency indefinitely when offline. |
| 11 — Monitoring & Observability
Metrics, logging, tracing, alerting, and SLOs |
| --- |
| Tool | Role |
| --- | --- |
| Prometheus | Metrics collection. Scrapes all services every 15s. 30-day retention. |
| Grafana | Dashboards: API health, DB performance, AI queue depth, School Box sync status. |
| Loki | Log aggregation from all pods via Promtail. Structured JSON logs indexed by trace ID. |
| Jaeger | Distributed tracing. Every API request generates a trace ID propagated to all downstream calls. |
| AlertManager | Routes alerts: PagerDuty (P1/P2, on-call), Slack (P3/P4), email (weekly digests). |
| Uptime Robot | External synthetic monitoring. Checks 6 critical endpoints every 60s from 3 regions. |
| Service | SLO Target |
| --- | --- |
| API availability | 99.9% monthly (< 44 minutes downtime/month) |
| API P99 response time | < 400ms for all non-AI endpoints |
| AI job completion (study plan) | 95% of jobs complete in < 30 seconds |
| School Box sync lag | < 6 hours behind cloud data when connectivity available |
| Database failover RTO | < 60 seconds (PostgreSQL Multi-AZ automatic failover) |
| Database RPO | < 5 minutes (continuous WAL archiving) |
| Results day availability | 99.95% — dedicated SLO for national exam results release |
| Severity | Trigger Conditions |
| --- | --- |
| P1 — Page immediately | API availability < 99%, DB replication lag > 5min, active security incident, data exfiltration detected |
| P2 — Page within 15min | P99 latency > 1s sustained 5min, Redis memory > 90%, AI service GPU down, Kong rate limit spike |
| P3 — Slack notification | Failed CI/CD deploy, certificate expiry < 30 days, School Box sync failure > 24h, disk > 80% |
| P4 — Weekly digest | Dependency CVEs found, slow query log entries, cache miss rate increase, cost anomaly |
| 12 — Data Governance & Compliance
Data sovereignty, privacy, retention, and legal compliance |
| --- |
| Classification | Definition and Access |
| --- | --- |
| Level 1 — Highly sensitive | Student name, national ID, parent contact, home region, disability status. Encrypted at application layer. Access: teacher and above. |
| Level 2 — Sensitive | Exam scores, gap analysis, study performance. Encrypted at rest. Access: student (own), teacher, school admin. |
| Level 3 — Internal | Aggregated school/region performance. Not personally identifiable. Access: Ministry and above. |
| Level 4 — Public | National pass rate statistics, anonymous curriculum coverage data. Published in Ministry reports. |
| Data Type | Retention Period |
| --- | --- |
| Student academic records | 7 years after graduation or school exit (Ethiopian education records law) |
| Exam answer sheets | 3 years active, then Glacier archive for 4 more years |
| Audit and access logs | 7 years (S3 Object Lock — cannot be deleted) |
| AI request logs | 1 year (prompt hashes only — no student content) |
| School Box sync logs | 90 days |
| Session tokens (Redis) | 7 days max then auto-expire |
| Backup snapshots | PostgreSQL: 35 days PITR. Neo4j: 30 days. S3 versioned objects: 7 years. |
| 13 — Deployment Environments
Development, staging, production, and DR |
| --- |
| Environment | Purpose | Infrastructure |
| --- | --- | --- |
| Environment | Purpose | Infrastructure |
| local | Developer laptops. Docker Compose. No cloud resources. | Docker Compose + Testcontainers |
| staging | Pre-production. Real infrastructure, 20% of prod capacity. | EKS (2 nodes), RDS small, ElastiCache small |
| production | Live system. Full capacity. Multi-AZ. | EKS (10–20 nodes autoscale), RDS Multi-AZ, ElastiCache cluster |
| dr | Disaster recovery. Passive — restored from S3 backups. RTO: 1 hour. | EKS (eu-west-1), standby RDS read replica |
| school-box | Physical hardware at each school. Offline-capable. | Docker Compose on local Ubuntu Server |
| 14 — Key Architecture Decisions
Rationale behind major technology choices |
| --- |