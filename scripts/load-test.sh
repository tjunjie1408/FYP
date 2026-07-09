#!/usr/bin/env bash
# Bursty inference traffic to exercise KServe scale-up / scale-to-zero cycles
# for the cost report. Pure curl+sleep, no extra tooling.
# Usage: scripts/load-test.sh [BURSTS] [RPS] [BURST_SECS] [IDLE_SECS]
set -euo pipefail

BURSTS="${1:-5}"
RPS="${2:-5}"
BURST_SECS="${3:-60}"
IDLE_SECS="${4:-180}"
HOST="${HOST:-mnist-demo.models.example.com}"
URL="${URL:-http://localhost:8080/v1/models/mnist-demo:predict}"
PAYLOAD="${PAYLOAD:-serving/samples/sample-request.json}"

sleep_between=$(awk "BEGIN { print 1.0 / $RPS }")

for b in $(seq 1 "$BURSTS"); do
  echo ">> burst $b/$BURSTS: ${BURST_SECS}s at ~${RPS} rps"
  end=$(( $(date +%s) + BURST_SECS ))
  while (( $(date +%s) < end )); do
    curl -s -o /dev/null -w "%{http_code} %{time_total}s\n" \
      -H "Host: $HOST" -H "Content-Type: application/json" \
      --data-binary "@$PAYLOAD" "$URL" || true
    sleep "$sleep_between"
  done
  if (( b < BURSTS )); then
    echo ">> idle ${IDLE_SECS}s (predictor should scale to zero)"
    sleep "$IDLE_SECS"
  fi
done
echo ">> done. Generate the report: make report"
