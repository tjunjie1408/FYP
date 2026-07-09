#!/usr/bin/env bash
# End-to-end demo: train -> queue -> serve -> scale-to-zero.
# Assumes the platform is up (scripts/up.sh). On any failure see docs/runbook.md.
set -euo pipefail

NS_TRAIN=training
NS_MODELS=models
PREDICT_HOST="mnist-demo.models.example.com"
PREDICT_URL="http://localhost:8080/v1/models/mnist-demo:predict"

banner() { echo ""; echo "============================================================"; echo ">> $1"; echo "============================================================"; }

wait_phase() { # <name> <phase> <timeout>
  local name="$1" want="$2" timeout="${3:-600}" deadline
  deadline=$(( $(date +%s) + timeout ))
  while true; do
    local p
    p=$(kubectl -n "$NS_TRAIN" get trainingjob "$name" -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
    echo "   $name: ${p:-<none>}"
    [[ "$p" == "$want" ]] && return 0
    [[ "$p" == "Failed" && "$want" != "Failed" ]] && { echo "!! $name Failed unexpectedly" >&2; return 1; }
    (( $(date +%s) > deadline )) && { echo "!! timeout waiting for $name=$want" >&2; return 1; }
    sleep 5
  done
}

banner "1/6  GPUQuota (hardGPUs=2) + three 1-GPU TrainingJobs -> watch queueing"
kubectl apply -f gitops/apps/training-demo/gpuquota.yaml
for n in mnist-a mnist-b mnist-c; do
  kubectl apply -f - <<EOF
apiVersion: mlplatform.fyp.io/v1alpha1
kind: TrainingJob
metadata:
  name: $n
  namespace: $NS_TRAIN
spec:
  model: { name: mnist-cnn }
  dataset: { name: mnist }
  hyperparameters: { epochs: "2", batchSize: "128", learningRate: "0.001" }
  gpus: 1
  retry: { backoffLimit: 2 }
  ttlSecondsAfterFinished: 3600
  output: { bucket: models }
EOF
done
sleep 8
echo "   With 2 GPUs and three 1-GPU jobs, exactly one should be Queued:"
kubectl -n "$NS_TRAIN" get trainingjobs
kubectl -n "$NS_TRAIN" get gpuquota default

banner "2/6  Wait for the first job to finish training"
wait_phase mnist-a Succeeded 900

banner "3/6  Model artifact in MinIO"
kubectl -n minio delete pod mc-stat --ignore-not-found >/dev/null 2>&1 || true
kubectl -n minio run mc-stat --rm -i --restart=Never --image=quay.io/minio/mc:RELEASE.2025-04-16T18-13-26Z -- \
  sh -c 'mc alias set m http://minio.minio.svc.cluster.local:9000 minioadmin minioadmin >/dev/null && mc ls -r m/models/training/mnist-a/' \
  || echo "   (mc check is best-effort)"

banner "4/6  Deploy InferenceService (KServe, scale-to-zero) for mnist-a"
sed 's#training/mnist-demo#training/mnist-a#' \
  gitops/apps/inference-demo/inferenceservice.yaml | kubectl apply -f -
echo "   waiting for InferenceService Ready (storage-initializer pull can take a minute)..."
kubectl -n "$NS_MODELS" wait --for=condition=Ready inferenceservice/mnist-demo --timeout=600s

banner "5/6  Predict (traffic enters via Kourier on localhost:8080)"
# kind's extraPortMappings expose Kourier's NodePort on host 8080. If you're
# on a cluster without the mapping, run in another shell first:
#   kubectl -n kourier-system port-forward svc/kourier 8080:80
curl -s -H "Host: $PREDICT_HOST" -H "Content-Type: application/json" \
  --data-binary @serving/samples/sample-request.json "$PREDICT_URL" \
  && echo "" || echo "   (if this failed, see docs/runbook.md Knative/Kourier section)"

banner "6/6  Idle -> scale to zero, then cold start"
echo "   waiting up to 3 min for predictor replicas -> 0 ..."
deadline=$(( $(date +%s) + 180 ))
while true; do
  reps=$(kubectl -n "$NS_MODELS" get deploy -l serving.knative.dev/service=mnist-demo-predictor -o jsonpath='{.items[*].status.replicas}' 2>/dev/null || echo "")
  echo "   replicas: ${reps:-0}"
  [[ -z "$reps" || "$reps" == "0" ]] && { echo "   >> scaled to zero (GPU idle cost avoided)"; break; }
  (( $(date +%s) > deadline )) && { echo "   (did not reach zero in time; check autoscaler window)"; break; }
  sleep 10
done
echo "   cold-start request (timed):"
curl -s -o /dev/null -w "   HTTP %{http_code} in %{time_total}s\n" \
  -H "Host: $PREDICT_HOST" -H "Content-Type: application/json" \
  --data-binary @serving/samples/sample-request.json "$PREDICT_URL" || true

banner "Demo complete. Grafana 'Cost & Reliability' dashboard shows the GPU-hours story."
