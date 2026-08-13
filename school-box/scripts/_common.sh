# Shared by every script in this directory -- source, don't execute.
# `SCRIPT_DIR`/`COMPOSE_DIR`/`ROOT_DIR`/`ENV_FILE` are resolved the same
# way everywhere so a script run from any cwd (cron, an operator's shell,
# another script) behaves identically.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_DIR="$SCRIPT_DIR/../compose"
ROOT_DIR="$SCRIPT_DIR/../.."
ENV_FILE="$ROOT_DIR/.env.school-box"

# docker compose needs --env-file pointed here explicitly: ENV_FILE lives
# at the repo root (school-box/../.env.school-box), not next to
# docker-compose.yml, so compose's own cwd-relative .env auto-discovery
# would never find it. See docker-compose.yml's api/sync-agent
# `env_file:` entries, which resolve relative to COMPOSE_DIR the same way.
dc() {
    docker compose --env-file "$ENV_FILE" -f "$COMPOSE_DIR/docker-compose.yml" "$@"
}

require_docker() {
    if ! command -v docker >/dev/null 2>&1; then
        echo "ERROR: docker is not installed or not on PATH." >&2
        exit 1
    fi
    if ! docker compose version >/dev/null 2>&1; then
        echo "ERROR: the 'docker compose' plugin is not available (docker compose version failed)." >&2
        exit 1
    fi
}

# Exports every var in .env.school-box into this shell process (not just
# into the containers docker compose starts) -- health-check.sh and
# backup.sh both need to read e.g. NEO4J_PASSWORD/POSTGRES_PASSWORD
# themselves (to probe the databases directly), not just pass them
# through to a container's environment the way `dc` alone would.
load_env_vars() {
    set -a
    # shellcheck disable=SC1090
    source "$ENV_FILE"
    set +a
}

require_env_file() {
    if [ ! -f "$ENV_FILE" ]; then
        echo "ERROR: $ENV_FILE not found." >&2
        echo "Run: cp \"$ROOT_DIR/.env.school-box.example\" \"$ENV_FILE\" then edit it." >&2
        exit 1
    fi
    # Catch the common "copied the example, forgot to edit it" mistake
    # before it produces a confusing failure three steps later (a
    # container crash-looping on a bad password, or two boxes silently
    # sharing SCHOOL_BOX_ID=changeme-generate-a-real-uuid).
    if grep -qE '^(SCHOOL_BOX_ID|SCHOOL_ID|POSTGRES_PASSWORD|NEO4J_PASSWORD|ECR_REGISTRY)=changeme' "$ENV_FILE"; then
        echo "ERROR: $ENV_FILE still has 'changeme' placeholder values." >&2
        echo "Edit SCHOOL_BOX_ID, SCHOOL_ID, POSTGRES_PASSWORD, NEO4J_PASSWORD, and ECR_REGISTRY before continuing." >&2
        exit 1
    fi
}
