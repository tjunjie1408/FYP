# Light local stack for OPERATOR development — kind + Argo Workflows + MinIO
# + CRDs + training pipeline. ~2 GB; fits a 16 GB laptop easily.
# NOT the full demo (no Cilium/Knative/KServe/Prometheus).
$ErrorActionPreference = "Stop"
$cluster = "fyp-dev"
$argoVersion = "v3.7.14"

Write-Host ">> [1/5] kind cluster ($cluster, default CNI)"
$existing = kind get clusters 2>$null
if ($existing -notcontains $cluster) {
  kind create cluster --name $cluster --config deploy/kind/dev-slice-config.yaml
}
kubectl config use-context "kind-$cluster"

Write-Host ">> [2/5] namespaces"
foreach ($ns in @("argo", "minio", "training", "models")) {
  kubectl create namespace $ns --dry-run=client -o yaml | kubectl apply -f -
}

Write-Host ">> [3/5] Argo Workflows $argoVersion"
kubectl apply -n argo -f "https://github.com/argoproj/argo-workflows/releases/download/$argoVersion/install.yaml"

Write-Host ">> [4/5] MinIO + bucket"
kubectl apply -k gitops/components/minio

Write-Host ">> [5/5] operator CRDs + training pipeline"
kubectl apply -k operator/config/crd
kubectl apply -k gitops/apps/training-demo

Write-Host ""
Write-Host ">> dev-slice ready on context kind-$cluster."
Write-Host "   Next:  make -C operator run"
Write-Host "   Then:  kubectl apply -f operator/config/samples/mlplatform_v1alpha1_trainingjob.yaml"
Write-Host "   Watch: kubectl -n training get trainingjobs,workflows -w"
Write-Host "   Down:  kind delete cluster --name $cluster"
