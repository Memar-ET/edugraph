#!/usr/bin/env bash
# Manual pull-and-restart. Watchtower already does this automatically
# every night at 2 AM (see docker-compose.yml's WATCHTOWER_SCHEDULE) --
# this is for an operator who wants to update on demand, or a box where
# Watchtower itself has been disabled.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=_common.sh
source "$SCRIPT_DIR/_common.sh"
require_docker
require_env_file
load_env_vars

echo "== Backing up before update =="
# An update that goes wrong (bad image, migration mismatch) is exactly
# when you want a fresh backup to already exist -- see backup.sh's own
# header for what it does and doesn't cover.
"$SCRIPT_DIR/backup.sh"

echo "== Pulling latest images (ECR_REGISTRY=${ECR_REGISTRY}, IMAGE_TAG=${IMAGE_TAG}) =="
dc pull

echo "== Recreating containers with new images =="
dc up -d

echo "== Post-update health check =="
sleep 5
"$SCRIPT_DIR/health-check.sh" || echo "NOTE: one or more services aren't healthy yet -- containers may still be warming up. Re-run health-check.sh in a minute; if it's still failing, check 'docker compose logs <service>'." >&2

echo
echo "Update complete."
