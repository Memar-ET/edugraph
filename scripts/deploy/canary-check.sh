#!/usr/bin/env bash
set -euo pipefail

ENV=${1:-staging}
BASE_URL="https://api-${ENV}.edugraph.edu.et"

echo "🔍 Checking canary health for $ENV..."

# Health check
STATUS=$(curl -sf -o /dev/null -w "%{http_code}" "$BASE_URL/health")
[ "$STATUS" = "200" ] || { echo "Health check failed: $STATUS"; exit 1; }

# Check error rate (Prometheus query)
ERROR_RATE=$(curl -sf "http://prometheus:9090/api/v1/query?query=rate(http_requests_total{status=~'5..'}[5m])" | jq '.data.result[0].value[1]' -r)
THRESHOLD="0.01"
awk "BEGIN{exit ($ERROR_RATE > $THRESHOLD)}" || { echo "Error rate $ERROR_RATE above threshold $THRESHOLD"; exit 1; }

echo "✅ Canary healthy. Error rate: $ERROR_RATE"
