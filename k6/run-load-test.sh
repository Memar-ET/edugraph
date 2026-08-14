#!/usr/bin/env bash
# Checklist 13.2: drives exam-submit-load-test.js through increasing
# concurrency stages against the REAL running api binary, reporting
# what the current single-instance system actually sustains -- not
# asserting a specific pass/fail target, since the PRD's own stated
# target (50,000 concurrent, see docs/architecture/performance.md) is
# for infrastructure (Kong/Kubernetes/horizontal scaling) this repo
# deliberately doesn't build. This finds the real ceiling instead.
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:18090}"
STAGES=(10 25 50 100 200)
USERS_FILE="/tmp/loadtest-users.json"
RESULTS_DIR="${RESULTS_DIR:-/tmp/loadtest-results}"

mkdir -p "$RESULTS_DIR"

total_users=0
for s in "${STAGES[@]}"; do total_users=$((total_users + s)); done
echo "== Seeding $total_users synthetic student accounts + 1 exam =="
(cd "$(dirname "$0")/../backend" && go run ./cmd/seed-load-test -count "$total_users" -out "$USERS_FILE")

offset=0
for vus in "${STAGES[@]}"; do
    echo
    echo "== Stage: $vus concurrent students =="
    k6 run \
        -e BASE_URL="$BASE_URL" \
        -e USERS_FILE="$USERS_FILE" \
        -e VUS="$vus" \
        -e OFFSET="$offset" \
        --summary-export "$RESULTS_DIR/stage-$vus.json" \
        "$(dirname "$0")/exam-submit-load-test.js" \
        || echo "  (k6 reported a non-zero exit for this stage -- see its output above; still recorded to $RESULTS_DIR/stage-$vus.json)"
    offset=$((offset + vus))
done

echo
echo "== Summary =="
for vus in "${STAGES[@]}"; do
    f="$RESULTS_DIR/stage-$vus.json"
    if [ -f "$f" ]; then
        python3 -c "
import json
d = json.load(open('$f'))
m = d['metrics']
def get(name, stat):
    return m.get(name, {}).get(stat)
dur = m.get('http_req_duration', {})
failed = m.get('http_req_failed', {})
print(f\"$vus VUs: p95={dur.get('p(95)', 0):.0f}ms  p99={dur.get('p(99)', 0):.0f}ms  avg={dur.get('avg', 0):.0f}ms  fail_rate={failed.get('rate', 0)*100:.1f}%\")
"
    fi
done
