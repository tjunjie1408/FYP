#!/usr/bin/env bash
# Bring up the local platform and wait for Argo CD to converge.
# Usage: scripts/up.sh [GITOPS_REPO_URL]
set -euo pipefail

REPO_URL="${1:-${GITOPS_REPO_URL:-https://github.com/tjunjie1408/FYP}}"
TF_DIR="terraform/envs/local"
TIMEOUT_SECS="${TIMEOUT_SECS:-1200}"

echo ">> terraform apply (kind + Cilium + Argo CD + root app)"
terraform -chdir="$TF_DIR" init -upgrade
terraform -chdir="$TF_DIR" apply -auto-approve -var "gitops_repo_url=$REPO_URL"

echo ">> waiting for all Argo CD Applications to be Healthy + Synced (timeout ${TIMEOUT_SECS}s)"
deadline=$(( $(date +%s) + TIMEOUT_SECS ))
while true; do
  # Count apps that are NOT (Healthy AND Synced).
  not_ready=$(kubectl -n argocd get applications -o json 2>/dev/null \
    | jq '[.items[] | select((.status.health.status != "Healthy") or (.status.sync.status != "Synced"))] | length' 2>/dev/null || echo "99")
  total=$(kubectl -n argocd get applications -o json 2>/dev/null | jq '.items | length' 2>/dev/null || echo "0")
  echo "   $(( total - not_ready ))/$total applications healthy+synced"
  if [[ "$not_ready" == "0" && "$total" != "0" ]]; then
    break
  fi
  if (( $(date +%s) > deadline )); then
    echo "!! timed out. Inspect: kubectl -n argocd get applications" >&2
    echo "   Troubleshooting: docs/runbook.md" >&2
    exit 1
  fi
  sleep 15
done

echo ""
echo ">> Platform is up."
PW=$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d)
echo "   Argo CD:  scripts/port-forward.sh argocd   -> https://localhost:8443  (admin / $PW)"
echo "   Grafana:  scripts/port-forward.sh grafana  -> http://localhost:3000   (admin / admin)"
echo "   Next:     scripts/demo.sh"
