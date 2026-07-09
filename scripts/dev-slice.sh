#!/usr/bin/env bash
# Light local stack for OPERATOR development — kind + Argo Workflows + MinIO
# + the TrainingJob/GPUQuota CRDs + the training pipeline. ~2 GB; fits a 16 GB
# laptop easily. NOT the full demo (no Cilium/Knative/KServe/Prometheus).
#
# After this finishes:
#   make -C operator run         # run the operator locally against this cluster
#   make images load             # (optional) build+load the trainer image so
#                                #  submitted TrainingJobs actually run train.py
#   kubectl apply -f operator/config/samples/mlplatform_v1alpha1_trainingjob.yaml
set -euo pipefail

CLUSTER=fyp-dev
ARGO_VERSION=v3.7.14

echo ">> [1/5] kind cluster ($CLUSTER, default CNI)"
if ! kind get clusters | grep -qx "$CLUSTER"; then
  kind create cluster --name "$CLUSTER" --config deploy/kind/dev-slice-config.yaml
fi
kubectl config use-context "kind-$CLUSTER"

echo ">> [2/5] namespaces"
for ns in argo minio training models; do
  kubectl create namespace "$ns" --dry-run=client -o yaml | kubectl apply -f -
done

echo ">> [3/5] Argo Workflows $ARGO_VERSION"
kubectl apply -n argo -f "https://github.com/argoproj/argo-workflows/releases/download/${ARGO_VERSION}/install.yaml"

echo ">> [4/5] MinIO + bucket (reuses gitops manifests; bucket-init hook runs as a plain Job here)"
kubectl apply -k gitops/components/minio

echo ">> [5/5] operator CRDs + training pipeline (WorkflowTemplate, RBAC, GPUQuota)"
kubectl apply -k operator/config/crd
kubectl apply -k gitops/apps/training-demo

echo ""
echo ">> dev-slice ready on context kind-$CLUSTER."
echo "   Next:  make -C operator run"
echo "   Then:  kubectl apply -f operator/config/samples/mlplatform_v1alpha1_trainingjob.yaml"
echo "   Watch: kubectl -n training get trainingjobs,workflows -w"
echo "   Down:  make dev-slice-down   (kind delete cluster --name $CLUSTER)"
