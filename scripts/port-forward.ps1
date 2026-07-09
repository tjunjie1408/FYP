# Port-forward a platform UI. Usage: scripts\port-forward.ps1 <argocd|grafana|minio|argo|prom>
param([Parameter(Mandatory = $true)][string]$Target)
$ErrorActionPreference = "Stop"

switch ($Target) {
  "argocd"  { Write-Host "https://localhost:8443"; kubectl -n argocd port-forward svc/argocd-server 8443:443 }
  "grafana" { Write-Host "http://localhost:3000 (admin/admin)"; kubectl -n monitoring port-forward svc/kube-prometheus-stack-grafana 3000:80 }
  "minio"   { Write-Host "http://localhost:9001 (minioadmin/minioadmin)"; kubectl -n minio port-forward svc/minio 9001:9001 }
  "argo"    { Write-Host "https://localhost:2746"; kubectl -n argo port-forward svc/argo-workflows-server 2746:2746 }
  "prom"    { Write-Host "http://localhost:9090"; kubectl -n monitoring port-forward svc/kube-prometheus-stack-prometheus 9090:9090 }
  default   { Write-Error "usage: port-forward.ps1 <argocd|grafana|minio|argo|prom>" }
}
