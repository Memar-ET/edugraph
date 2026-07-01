# EduGraph AI

**National Educational Intelligence Platform — Ethiopia**

An offline-first AI platform that transforms how Ethiopia learns, teaches, and makes educational decisions — from the classroom to the Ministry.

## Repository Structure

```
edugraph/
├── backend/          Go 1.22 — Core API (REST + WebSocket)
├── ai-service/       Python 3.12 + FastAPI — AI inference service  
├── frontend/         React 18 + TypeScript + Vite — Web application
├── school-box/       Docker Compose stack for offline School Box hardware
├── infra/            Terraform (AWS) + Helm (Kubernetes)
├── docs/             Architecture docs, ADRs, API specs, runbooks
└── scripts/          Dev setup, DB management, deploy helpers
```

## Quick Start

### Prerequisites
- Docker + Docker Compose
- Go 1.22, Node.js 20, Python 3.12
- pnpm 9 (`npm install -g pnpm@9`)

### 1. Clone and configure
```bash
git clone https://github.com/edugraph-ai/edugraph.git
cd edugraph
cp .env.example .env          # Edit with your local values
cp ai-service/.env.example ai-service/.env
```

### 2. Start all services
```bash
make dev
# or: docker compose up --build
```

Services:
| Service | URL |
|---|---|
| Backend API | http://localhost:8080 |
| Frontend | http://localhost:5173 |
| AI Service | http://localhost:8000 |
| Neo4j Browser | http://localhost:7474 |
| API Docs | http://localhost:8000/docs |

### 3. Run migrations
```bash
make migrate
make migrate-neo4j
```

### 4. Seed development data
```bash
make seed
```

## Development Commands

```bash
make help          # All available commands
make test          # Run all tests
make lint          # Lint all code
make build         # Build all services
make scan          # Security scans
```

## Branch Strategy

```
main       — production. Direct pushes blocked. PR + 1 senior review required.
develop    — staging. Auto-deploys to staging on merge.
feature/*  — feature branches. PR into develop.
hotfix/*   — critical fixes. PR into main + develop simultaneously.
```

## Architecture

See [docs/architecture/system-overview.md](docs/architecture/system-overview.md)

## Database Design

See [docs/architecture/database-design.md](docs/architecture/database-design.md)

## GitHub Secrets Setup

See [.github/SECRETS_TEMPLATE.md](.github/SECRETS_TEMPLATE.md)

---

**AASTU Innovation Program · June 2026 · CONFIDENTIAL**
