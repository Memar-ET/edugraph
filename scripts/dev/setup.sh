#!/usr/bin/env bash
set -euo pipefail

echo "🚀 EduGraph AI — Dev Setup"

# Check prerequisites
command -v docker  >/dev/null || { echo "Docker required"; exit 1; }
command -v go      >/dev/null || { echo "Go 1.22+ required"; exit 1; }
command -v node    >/dev/null || { echo "Node 20+ required"; exit 1; }
command -v python3 >/dev/null || { echo "Python 3.12+ required"; exit 1; }
command -v pnpm    >/dev/null || npm install -g pnpm@9

# Backend tools
go install github.com/air-verse/air@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.57.0

# Frontend deps
cd frontend && pnpm install && cd ..

# Python deps
cd ai-service && pip install -r requirements-dev.txt && cd ..

# Copy env files
[ -f .env ] || cp .env.example .env
[ -f ai-service/.env ] || cp ai-service/.env.example ai-service/.env

echo "✅ Setup complete. Run: make dev"
