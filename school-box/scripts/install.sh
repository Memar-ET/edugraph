#!/usr/bin/env bash
# First-time setup on fresh School Box hardware. Meant to be read, not
# just trusted -- see README.md's "Deployment" section for the expected
# invocation (cp the env example first, then this).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=_common.sh
source "$SCRIPT_DIR/_common.sh"

require_docker
require_env_file
load_env_vars

echo "== Pulling images (ECR_REGISTRY=${ECR_REGISTRY}, IMAGE_TAG=${IMAGE_TAG}) =="
# Best-effort: a box being provisioned with no internet yet must have
# had these images pre-loaded some other way (`docker save` on a
# connected machine -> copy the tarball over -> `docker load` here)
# before running this script -- see README.md's "Offline behavior"
# section. Don't abort the whole install just because the pull failed;
# `docker compose up -d` below will fail loudly and specifically if an
# image is actually missing, which is a clearer signal than this script
# guessing why the pull failed (no internet vs. bad registry vs. bad tag).
if ! dc pull; then
    echo "WARNING: image pull failed. If this box has no internet, pre-load images with 'docker load' first. Continuing -- 'docker compose up' below will fail clearly if an image is actually missing." >&2
fi

echo "== Starting services =="
dc up -d

echo "== Waiting for Postgres and Neo4j to report healthy =="
# Both have compose-level healthchecks (see docker-compose.yml); redis/
# ollama/api/ai-service don't, so this only waits on the two that do --
# health-check.sh below covers the rest once they've had time to start.
WAIT_SECS=120
ELAPSED=0
while true; do
    PG_STATUS="$(dc ps --format '{{.Service}} {{.Health}}' postgres 2>/dev/null | awk '{print $2}')"
    NEO_STATUS="$(dc ps --format '{{.Service}} {{.Health}}' neo4j 2>/dev/null | awk '{print $2}')"
    if [ "$PG_STATUS" = "healthy" ] && [ "$NEO_STATUS" = "healthy" ]; then
        echo "postgres and neo4j are healthy."
        break
    fi
    if [ "$ELAPSED" -ge "$WAIT_SECS" ]; then
        echo "ERROR: postgres/neo4j did not become healthy within ${WAIT_SECS}s (postgres=${PG_STATUS:-unknown}, neo4j=${NEO_STATUS:-unknown})." >&2
        echo "Check logs: docker compose --env-file '$ENV_FILE' -f '$COMPOSE_DIR/docker-compose.yml' logs postgres neo4j" >&2
        exit 1
    fi
    sleep 5
    ELAPSED=$((ELAPSED + 5))
done

echo "== Pulling the local LLM (${OLLAMA_MODEL:-qwen2.5:7b-instruct-q4_K_M}) =="
# Same offline caveat as the image pull above: this needs internet the
# first time, ever, on this box. If it fails here, AI features degrade
# to the cloud fallback (if LLM_PROVIDER/GEMINI_API_KEY allow it, which
# they deliberately don't for a School Box -- see .env.school-box.example)
# or return "unavailable" until this succeeds -- see
# docs/architecture/ai-models-local-vs-cloud.md.
if ! dc exec -T ollama ollama pull "${OLLAMA_MODEL:-qwen2.5:7b-instruct-q4_K_M}"; then
    echo "WARNING: failed to pull the Ollama model. AI features (tutor, gap analysis, study plans) won't work until this succeeds -- re-run 'docker compose exec ollama ollama pull ${OLLAMA_MODEL:-qwen2.5:7b-instruct-q4_K_M}' once this box has internet." >&2
fi

echo "== Running database migrations =="
# Flyway isn't part of this compose stack (it targets a Postgres this
# box owns directly, unlike the cloud side which runs it against
# Supabase -- see CLAUDE.md's Migrations note); run it once against this
# box's own Postgres the same way, from the repo's migrations directory.
if command -v docker >/dev/null 2>&1 && [ -d "$ROOT_DIR/backend/db/migrations" ]; then
    # Derived from the running postgres container rather than assumed
    # from the compose project name (basename of $COMPOSE_DIR) -- the
    # latter breaks silently if this checkout ever lives in a
    # differently-named directory or COMPOSE_PROJECT_NAME is overridden.
    PG_CONTAINER_ID="$(dc ps -q postgres)"
    NETWORK_NAME="$(docker inspect -f '{{range $k, $v := .NetworkSettings.Networks}}{{$k}}{{end}}' "$PG_CONTAINER_ID" 2>/dev/null)"
    docker run --rm --network "${NETWORK_NAME:?could not determine the postgres containers docker network}" \
        -e FLYWAY_URL="jdbc:postgresql://postgres:5432/edugraph" \
        -e FLYWAY_USER="edugraph" \
        -e FLYWAY_PASSWORD="${POSTGRES_PASSWORD}" \
        -v "$ROOT_DIR/backend/db/migrations:/flyway/sql" \
        flyway/flyway:10 -locations=filesystem:/flyway/sql migrate \
        || echo "WARNING: Flyway migration failed -- run it manually before trusting this box's schema (see CLAUDE.md's Migrations note for the exact command shape)." >&2
else
    echo "WARNING: backend/db/migrations not found next to this checkout -- skipping automatic migration. Run Flyway manually against this box's Postgres before using it." >&2
fi

echo "== First health check =="
"$SCRIPT_DIR/health-check.sh" || echo "NOTE: one or more services aren't healthy yet -- this can be normal right after first start (Ollama model pull, first Postgres/Neo4j warm-up). Re-run health-check.sh in a minute." >&2

echo
echo "Install complete. Frontend: http://localhost/ (via Caddy on :80). Re-run scripts/health-check.sh any time to check status."
