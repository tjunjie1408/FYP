#!/usr/bin/env bash
# Port-forward a platform UI. Usage: scripts/port-forward.sh <argocd|grafana|minio|argo>
set -euo pipefail

case "${1:-}" in
  argocd)  echo "https://localhost:8443"; kubectl -n argocd port-forward svc/argocd-server 8443:443 ;;
  grafana) echo "http://localhost:3000 (admin/admin)"; kubectl -n monitoring port-forward svc/kube-prometheus-stack-grafana 3000:80 ;;
  minio)   echo "http://localhost:9001 (minioadmin/minioadmin)"; kubectl -n minio port-forward svc/minio 9001:9001 ;;
  argo)    echo "https://localhost:2746"; kubectl -n argo port-forward svc/argo-workflows-server 2746:2746 ;;
  prom)    echo "http://localhost:9090"; kubectl -n monitoring port-forward svc/kube-prometheus-stack-prometheus 9090:9090 ;;
  *) echo "usage: $0 <argocd|grafana|minio|argo|prom>" >&2; exit 1 ;;
esac
