#!/usr/bin/env bash
# Destructive factory-reset: stops every service and deletes ALL local
# data volumes (Postgres, Neo4j, Redis, Ollama models, Caddy). Anything
# not yet synced to the cloud (sync.outbox rows, curriculum approved
# locally but not yet pushed) is gone permanently. Does NOT delete
# .env.school-box -- this box's identity/config survives a reset.
#
# Requires typing this box's own SCHOOL_BOX_ID to confirm (not just
# "yes") -- a destructive command that only needs a single keystroke to
# confirm is too easy to run on the wrong box by muscle memory.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=_common.sh
source "$SCRIPT_DIR/_common.sh"
require_docker
require_env_file
load_env_vars

echo "!! THIS WILL PERMANENTLY DELETE ALL LOCAL DATA ON THIS SCHOOL BOX !!"
echo "SCHOOL_BOX_ID: ${SCHOOL_BOX_ID:-unset}"
echo "SCHOOL_ID:     ${SCHOOL_ID:-unset}"
echo
echo "This deletes: Postgres data, Neo4j data, Redis data, downloaded Ollama models, Caddy state."
echo "Anything not yet synced to the cloud is lost. Consider running backup.sh first (Ctrl+C now to do that)."
echo
read -r -p "Type this box's SCHOOL_BOX_ID exactly to confirm: " CONFIRM
if [ "$CONFIRM" != "${SCHOOL_BOX_ID:-}" ]; then
    echo "Confirmation did not match SCHOOL_BOX_ID. Aborting -- nothing was deleted." >&2
    exit 1
fi

echo "== Stopping services and removing volumes =="
dc down -v

echo "== Reset complete =="
echo "All containers and data volumes removed. .env.school-box was left in place."
echo "Run ./install.sh to bring this box back up from a clean state."
