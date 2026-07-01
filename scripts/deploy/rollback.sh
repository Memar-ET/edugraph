#!/usr/bin/env bash
set -euo pipefail

ENV=${1:-production}
NAMESPACE="edugraph-${ENV}"

echo "⚠️  Rolling back $ENV deployment..."
helm rollback edugraph 0 --namespace "$NAMESPACE" --wait
echo "✅ Rollback complete"
