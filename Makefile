.PHONY: help dev test lint build migrate clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-22s\033[0m %s\n", $$1, $$2}'

# ── Dev ──────────────────────────────────────────────────────
dev: ## Start all services locally (Docker Compose)
	docker compose up --build

dev-backend: ## Start backend only with hot reload
	cd backend && air

dev-frontend: ## Start frontend dev server
	cd frontend && pnpm dev

dev-ai: ## Start AI service
	cd ai-service && uvicorn app.main:app --reload --port 8000

# ── Test ─────────────────────────────────────────────────────
test: test-backend test-frontend test-ai ## Run all tests

test-backend: ## Run Go tests
	cd backend && go test -race -cover ./...

test-frontend: ## Run frontend tests
	cd frontend && pnpm test

test-ai: ## Run Python tests
	cd ai-service && pytest

test-e2e: ## Run Playwright E2E tests
	cd frontend && pnpm playwright test

test-integration: ## Run integration tests (requires running DBs)
	cd backend && go test -race -tags=integration ./test/integration/...

# ── Lint ─────────────────────────────────────────────────────
lint: lint-backend lint-frontend lint-ai ## Lint all code

lint-backend: ## Lint Go
	cd backend && golangci-lint run

lint-frontend: ## Lint TypeScript
	cd frontend && pnpm lint

lint-ai: ## Lint Python
	cd ai-service && ruff check . && mypy app/

# ── Build ────────────────────────────────────────────────────
build: build-backend build-frontend build-ai ## Build all services

build-backend:
	cd backend && CGO_ENABLED=0 go build -o bin/api ./cmd/api

build-frontend:
	cd frontend && pnpm build

build-ai:
	cd ai-service && pip install -r requirements.txt

# ── Database ─────────────────────────────────────────────────
migrate: ## Run PostgreSQL migrations
	cd backend && flyway migrate

migrate-status: ## Show migration status
	cd backend && flyway info

migrate-neo4j: ## Run Neo4j migrations
	cd backend && go run ./cmd/migrate neo4j

seed: ## Seed development data
	bash scripts/dev/seed-dev-data.sh

# ── Infra ────────────────────────────────────────────────────
infra-plan-staging: ## Terraform plan for staging
	cd infra/terraform/envs/staging && terraform plan

infra-apply-staging: ## Terraform apply for staging
	cd infra/terraform/envs/staging && terraform apply

# ── Security ─────────────────────────────────────────────────
scan: ## Run security scans (Trivy + govulncheck + snyk)
	trivy fs .
	cd backend && govulncheck ./...
	cd frontend && pnpm audit

# ── Clean ────────────────────────────────────────────────────
clean: ## Remove build artifacts
	rm -rf backend/bin frontend/dist ai-service/__pycache__
	docker compose down --volumes
