#!/usr/bin/env bash
# Poll the predictor's replica count to CSV for the cost report
# (the reliable floor when Prometheus metric names are uncertain).
# Usage: hack/measure-replicas.sh [DURATION_SECS] [OUT_CSV]
set -euo pipefail

DURATION="${1:-1200}"
OUT="${2:-/dev/stdout}"
NS_MODELS=models

echo "timestamp,replicas" > "$OUT"
end=$(( $(date +%s) + DURATION ))
while (( $(date +%s) < end )); do
  reps=$(kubectl -n "$NS_MODELS" get deploy -l serving.knative.dev/service=mnist-demo-predictor \
    -o jsonpath='{.items[*].status.replicas}' 2>/dev/null || echo "0")
  echo "$(date +%s),${reps:-0}" >> "$OUT"
  sleep 5
done
