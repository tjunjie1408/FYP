#!/usr/bin/env bash
# Full end-to-end smoke test — the demo rehearsal. Run before every demo.
# Assumes the platform is up. Exits non-zero on any failure.
set -euo pipefail

NS_TRAIN=training
NS_MODELS=models
fail() { echo "!! SMOKE FAIL: $1" >&2; exit 1; }
ok()   { echo ">> OK: $1"; }

echo ">> [1] Argo CD applications healthy"
not_ready=$(kubectl -n argocd get applications -o json | jq '[.items[] | select((.status.health.status != "Healthy") or (.status.sync.status != "Synced"))] | length')
[[ "$not_ready" == "0" ]] || fail "$not_ready applications not healthy/synced"
ok "all applications healthy+synced"

echo ">> [2] quota + two 1-GPU jobs, expect one Queued"
kubectl apply -f gitops/apps/training-demo/gpuquota.yaml
for n in smoke-1 smoke-2 smoke-3; do
  kubectl apply -f - <<EOF
apiVersion: mlplatform.fyp.io/v1alpha1
kind: TrainingJob
metadata: { name: $n, namespace: $NS_TRAIN }
spec:
  model: { name: mnist-cnn }
  dataset: { name: mnist }
  hyperparameters: { epochs: "1" }
  gpus: 1
  retry: { backoffLimit: 2 }
  ttlSecondsAfterFinished: 600
  output: { bucket: models }
EOF
done
sleep 10
queued=$(kubectl -n "$NS_TRAIN" get trainingjobs -o json | jq '[.items[] | select(.status.phase == "Queued")] | length')
[[ "$queued" -ge 1 ]] || fail "expected >=1 Queued job, got $queued"
ok "$queued job(s) Queued (quota enforced)"

echo ">> [3] wait for smoke-1 Succeeded (<=15m)"
deadline=$(( $(date +%s) + 900 ))
while true; do
  p=$(kubectl -n "$NS_TRAIN" get trainingjob smoke-1 -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
  [[ "$p" == "Succeeded" ]] && break
  [[ "$p" == "Failed" ]] && fail "smoke-1 Failed"
  (( $(date +%s) > deadline )) && fail "smoke-1 did not Succeed in time (phase=$p)"
  sleep 10
done
ok "smoke-1 Succeeded"

echo ">> [4] model artifact present in MinIO"
kubectl -n minio run mc-stat-$$ --rm -i --restart=Never --image=quay.io/minio/mc:RELEASE.2025-04-16T18-13-26Z -- \
  sh -c 'mc alias set m http://minio.minio.svc.cluster.local:9000 minioadmin minioadmin >/dev/null && mc stat m/models/training/smoke-1/model.pt' \
  || fail "model.pt not found in MinIO"
ok "model.pt in MinIO"

echo ">> [5] InferenceService Ready + predict returns 200"
sed 's#training/mnist-demo#training/smoke-1#' gitops/apps/inference-demo/inferenceservice.yaml | kubectl apply -f -
kubectl -n "$NS_MODELS" wait --for=condition=Ready inferenceservice/mnist-demo --timeout=600s || fail "InferenceService not Ready"
code=$(curl -s -o /dev/null -w "%{http_code}" -H "Host: mnist-demo.models.example.com" \
  -H "Content-Type: application/json" --data-binary @serving/samples/sample-request.json \
  http://localhost:8080/v1/models/mnist-demo:predict || echo "000")
[[ "$code" == "200" ]] || fail "predict returned HTTP $code"
ok "predict returned 200"

echo ">> [6] scale-to-zero within 3 min idle"
deadline=$(( $(date +%s) + 180 ))
while true; do
  reps=$(kubectl -n "$NS_MODELS" get deploy -l serving.knative.dev/service=mnist-demo-predictor -o jsonpath='{.items[*].status.replicas}' 2>/dev/null || echo "0")
  [[ -z "$reps" || "$reps" == "0" ]] && break
  (( $(date +%s) > deadline )) && fail "predictor did not scale to zero (replicas=$reps)"
  sleep 10
done
ok "scaled to zero"

echo ""
echo ">> SMOKE PASS — platform is demo-ready."
