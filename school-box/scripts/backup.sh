#!/usr/bin/env bash
# Backs up Postgres + Neo4j data. Run manually, or add to cron (e.g.
# `0 3 * * * /path/to/backup.sh` for a nightly 3 AM backup).
#
# What this does NOT cover: the sync-agent already gets most
# student-generated data (exam submissions, gap records, etc.) to the
# cloud via sync.outbox -- see README.md's "Backs up" line -- but
# curriculum content approved locally and anything still sitting unsynced
# in the outbox at backup time exists ONLY on this box until it syncs.
# This backup is this box's actual safety net for that, not a
# nice-to-have.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=_common.sh
source "$SCRIPT_DIR/_common.sh"
require_docker
require_env_file
load_env_vars

BACKUP_ROOT="${SCHOOL_BOX_BACKUP_DIR:-$ROOT_DIR/backups}"
TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
BACKUP_DIR="$BACKUP_ROOT/$TIMESTAMP"
mkdir -p "$BACKUP_DIR"
echo "== Backing up to $BACKUP_DIR =="

echo "-- Postgres (pg_dump, custom format) --"
# -Fc (custom format) rather than plain SQL: smaller, and restorable with
# pg_restore --clean without hand-editing, including just one schema if
# ever needed.
dc exec -T postgres pg_dump -U edugraph -Fc edugraph > "$BACKUP_DIR/postgres.dump"

echo "-- Neo4j (offline volume snapshot) --"
# Community edition has no online-backup tool the way Enterprise does
# (neo4j-admin's online backup is an Enterprise feature) -- the reliable
# option here is a brief stop, tar the data volume, restart. This is a
# real few-seconds-to-tens-of-seconds outage for the graph, which is why
# this is a manual/cron script rather than something triggered
# automatically on every write.
#
# The actual volume name is read off the container's own mount (docker
# inspect works on a stopped container too) rather than assumed as
# "<project>_neo4j_data" -- that prefix depends on the compose project
# name, which install.sh's network-name lookup already found reason not
# to hardcode.
NEO4J_CONTAINER_ID="$(dc ps -aq neo4j)"
NEO4J_VOLUME_NAME="$(docker inspect -f '{{range .Mounts}}{{if eq .Destination "/data"}}{{.Name}}{{end}}{{end}}' "$NEO4J_CONTAINER_ID" 2>/dev/null)"
dc stop neo4j
docker run --rm \
    -v "${NEO4J_VOLUME_NAME:?could not determine the neo4j data volume name}:/data:ro" \
    -v "$BACKUP_DIR:/backup" \
    alpine:3 tar czf /backup/neo4j_data.tar.gz -C /data . \
    || { echo "ERROR: neo4j volume snapshot failed -- restarting neo4j before exiting." >&2; dc start neo4j; exit 1; }
dc start neo4j

echo "-- Waiting for Neo4j to report healthy again --"
ELAPSED=0
while [ "$(dc ps --format '{{.Service}} {{.Health}}' neo4j 2>/dev/null | awk '{print $2}')" != "healthy" ]; do
    if [ "$ELAPSED" -ge 60 ]; then
        echo "WARNING: neo4j did not report healthy within 60s of restart -- check it manually." >&2
        break
    fi
    sleep 5
    ELAPSED=$((ELAPSED + 5))
done

# Record what this backup actually captures, since "postgres.dump +
# neo4j_data.tar.gz" isn't self-explanatory to whoever finds this
# directory later.
cat > "$BACKUP_DIR/MANIFEST.txt" <<EOF
School Box backup
Timestamp (UTC): $TIMESTAMP
SCHOOL_BOX_ID: ${SCHOOL_BOX_ID:-unset}
Contents:
  postgres.dump    -- pg_dump -Fc of the 'edugraph' database (all schemas: curriculum, assessment, students, embeddings, careers, schools, app_storage, sync.outbox, public)
  neo4j_data.tar.gz -- full Neo4j data directory (offline snapshot, neo4j briefly stopped to take it)
Restore: see backup.sh's header comment or README.md.
EOF

echo "== Backup complete: $BACKUP_DIR =="

# Retention: keep the most recent N backups, delete older ones. A School
# Box has finite local storage (see README.md's hardware baseline) and
# nobody's pruning this by hand.
KEEP="${SCHOOL_BOX_BACKUP_KEEP:-7}"
BACKUP_COUNT="$(find "$BACKUP_ROOT" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ')"
if [ "$BACKUP_COUNT" -gt "$KEEP" ]; then
    # `head -n -$KEEP` (all-but-last-N) is GNU-only and silently
    # misbehaves on BSD/macOS head -- compute the count to remove
    # explicitly instead so this works the same on the Linux hardware
    # this actually deploys to and on a Mac someone tests it from.
    TO_REMOVE=$((BACKUP_COUNT - KEEP))
    echo "-- Pruning old backups (keeping most recent $KEEP of $BACKUP_COUNT) --"
    find "$BACKUP_ROOT" -mindepth 1 -maxdepth 1 -type d | sort | head -n "$TO_REMOVE" | while read -r old; do
        echo "  removing $old"
        rm -rf "$old"
    done
fi
