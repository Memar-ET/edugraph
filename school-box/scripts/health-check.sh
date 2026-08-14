#!/usr/bin/env bash
# Checks every School Box service. Exit 0 if everything's healthy, exit 1
# if anything critical is down. Meant to be run interactively by an
# operator and by install.sh at the end of a fresh install -- add it to
# cron yourself (e.g. */15 * * * *) if you want unattended monitoring;
# this script only checks, it never restarts anything.
set -uo pipefail  # not -e: we want to run every check and report all failures, not stop at the first

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=_common.sh
source "$SCRIPT_DIR/_common.sh"
require_docker
require_env_file
load_env_vars

# Matches docker-compose.yml's sync-agent SYNC_INTERVAL_MINUTES=360 --
# not read from .env.school-box (it isn't set there), so this mirrors
# the compose file's own hardcoded default rather than pretending it's
# configurable here when it isn't.
SYNC_INTERVAL_MINUTES=360

# checklist 13.3: optional webhook alerting -- set SCHOOL_BOX_ALERT_WEBHOOK
# in .env.school-box to a Slack incoming-webhook URL (or any endpoint
# that accepts {"text": "..."} JSON) to get notified without watching
# cron output. Unset by default: this script works standalone (exit code
# + stdout) with no webhook configured, matching the rest of this
# project's "real infra, not aspirational" stance -- no webhook exists
# to send to until an operator actually sets one up.
send_alert() {
    local message="$1"
    if [ -n "${SCHOOL_BOX_ALERT_WEBHOOK:-}" ]; then
        curl -fsS --max-time 5 -X POST "$SCHOOL_BOX_ALERT_WEBHOOK" \
            -H "Content-Type: application/json" \
            -d "{\"text\": \"[School Box ${SCHOOL_BOX_ID:-unknown}] ${message}\"}" >/dev/null 2>&1 \
            || echo "  [WARN] alert webhook call failed -- see the [ALERT]/[FAIL] lines above for what it would have said"
    fi
}

FAILED=0

ok()   { echo "  [OK]   $1"; }
fail() { echo "  [FAIL] $1"; FAILED=1; }

echo "== Container status =="
# One line per service: NAME  STATUS. A service missing from this output
# entirely (never created) fails the same as one that's Exited.
SERVICES="api ai-service frontend postgres neo4j redis ollama sync-agent caddy watchtower"
STATUS_OUT="$(dc ps --format '{{.Service}} {{.State}}' 2>/dev/null || true)"
for svc in $SERVICES; do
    line="$(echo "$STATUS_OUT" | grep -E "^${svc} " || true)"
    if [ -z "$line" ]; then
        fail "$svc: not running"
    elif echo "$line" | grep -q "running"; then
        ok "$svc: running"
    else
        fail "$svc: $(echo "$line" | awk '{print $2}')"
    fi
done

echo "== Service-level health =="

# Go API
if curl -fsS --max-time 5 http://localhost:8080/health >/dev/null 2>&1; then
    ok "api: /health responding"
else
    fail "api: /health not responding on :8080"
fi

# ai-service (only route it exposes unconditionally, see main.py's "@app.get(\"/\")")
if curl -fsS --max-time 5 http://localhost:8000/ >/dev/null 2>&1; then
    ok "ai-service: / responding"
else
    fail "ai-service: / not responding on :8000"
fi

# Postgres
if dc exec -T postgres pg_isready -U edugraph >/dev/null 2>&1; then
    ok "postgres: accepting connections"
else
    fail "postgres: pg_isready failed"
fi

# Neo4j
if dc exec -T neo4j cypher-shell -u neo4j -p "${NEO4J_PASSWORD:-}" "RETURN 1" >/dev/null 2>&1; then
    ok "neo4j: accepting queries"
else
    fail "neo4j: cypher-shell probe failed"
fi

# Redis
if [ "$(dc exec -T redis redis-cli ping 2>/dev/null | tr -d '\r')" = "PONG" ]; then
    ok "redis: PONG"
else
    fail "redis: no PONG from redis-cli"
fi

# Ollama
if dc exec -T ollama ollama list >/dev/null 2>&1; then
    ok "ollama: responding"
else
    fail "ollama: 'ollama list' failed"
fi

# sync-agent -- the one service this script checks in real depth, since
# checklist 9.1/9.4's whole point is "did this box actually sync." See
# sync-agent/internal/health/health.go for what these two endpoints mean.
if curl -fsS --max-time 5 http://localhost:9090/health >/dev/null 2>&1; then
    ok "sync-agent: /health responding"
    SYNC_JSON="$(curl -fsS --max-time 5 http://localhost:9090/health/sync 2>/dev/null || echo '{}')"
    echo "  sync status: $SYNC_JSON"
    PENDING="$(echo "$SYNC_JSON" | grep -o '"pending_outbox_count":[0-9]*' | grep -o '[0-9]*$' || echo "")"
    SECS_SINCE="$(echo "$SYNC_JSON" | grep -o '"seconds_since_success":[0-9.]*' | grep -o '[0-9.]*$' || echo "")"
    if [ -n "$SECS_SINCE" ]; then
        # checklist 13.3: thresholds match the PRD's own stated numbers
        # (edugraph-architecture.docx's SLO/severity tables), not an
        # invented heuristic -- "School Box sync lag: < 6 hours behind
        # cloud data when connectivity available" (SLO) and "School Box
        # sync failure > 24h" (P3 -- Slack notification). A School Box
        # legitimately going offline for a while is expected behavior,
        # not a fault (see agent.go / README.md's "Offline behavior"
        # section) -- WARN at the SLO, ALERT (webhook) at the P3 trigger.
        SECS_INT="${SECS_SINCE%.*}"
        SLO_SECS=$((6 * 3600))
        P3_SECS=$((24 * 3600))
        if [ "${SECS_INT:-0}" -gt "$P3_SECS" ]; then
            echo "  [ALERT] last successful sync was ${SECS_INT}s ago (>24h -- PRD P3 trigger) -- check CLOUD_SYNC_ENDPOINT reachability"
            send_alert "School Box sync failure >24h (SCHOOL_BOX_ID=${SCHOOL_BOX_ID:-unset}): last successful sync ${SECS_INT}s ago"
        elif [ "${SECS_INT:-0}" -gt "$SLO_SECS" ]; then
            echo "  [WARN] last successful sync was ${SECS_INT}s ago (>6h -- past the PRD's SLO, not yet the 24h P3 trigger)"
        fi
    fi
    if [ -n "$PENDING" ] && [ "$PENDING" != "0" ]; then
        echo "  [INFO] $PENDING change(s) queued in sync.outbox, not yet pushed to cloud"
    fi
else
    fail "sync-agent: /health not responding on :9090"
fi

echo
if [ "$FAILED" -eq 0 ]; then
    echo "All checks passed."
else
    echo "One or more checks FAILED -- see [FAIL] lines above."
    send_alert "One or more service checks FAILED on this box -- run health-check.sh there for details."
fi
exit "$FAILED"
