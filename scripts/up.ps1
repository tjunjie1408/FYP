# Bring up the local platform and wait for Argo CD to converge.
# Usage: scripts\up.ps1 [-RepoUrl https://github.com/you/FYP]
param(
  [string]$RepoUrl = $(if ($env:GITOPS_REPO_URL) { $env:GITOPS_REPO_URL } else { "https://github.com/tjunjie1408/FYP" }),
  [int]$TimeoutSecs = 1200
)
$ErrorActionPreference = "Stop"
$tfDir = "terraform/envs/local"

Write-Host ">> terraform apply (kind + Cilium + Argo CD + root app)"
terraform -chdir="$tfDir" init -upgrade
if ($?) { terraform -chdir="$tfDir" apply -auto-approve -var "gitops_repo_url=$RepoUrl" }

Write-Host ">> waiting for all Argo CD Applications Healthy + Synced (timeout ${TimeoutSecs}s)"
$deadline = (Get-Date).AddSeconds($TimeoutSecs)
while ($true) {
  $json = kubectl -n argocd get applications -o json 2>$null | ConvertFrom-Json
  $total = 0; $ready = 0
  if ($json -and $json.items) {
    $total = $json.items.Count
    foreach ($a in $json.items) {
      if ($a.status.health.status -eq "Healthy" -and $a.status.sync.status -eq "Synced") { $ready++ }
    }
  }
  Write-Host "   $ready/$total applications healthy+synced"
  if ($total -gt 0 -and $ready -eq $total) { break }
  if ((Get-Date) -gt $deadline) {
    Write-Error "Timed out. Inspect: kubectl -n argocd get applications  (see docs/runbook.md)"
  }
  Start-Sleep -Seconds 15
}

Write-Host ""
Write-Host ">> Platform is up."
$pwB64 = kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}'
$pw = [Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($pwB64))
Write-Host "   Argo CD:  scripts\port-forward.ps1 argocd   -> https://localhost:8443  (admin / $pw)"
Write-Host "   Grafana:  scripts\port-forward.ps1 grafana  -> http://localhost:3000   (admin / admin)"
Write-Host "   Next:     scripts\demo.ps1"
