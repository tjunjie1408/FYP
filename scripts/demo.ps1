# End-to-end demo: train -> queue -> serve -> scale-to-zero.
# Assumes the platform is up (scripts\up.ps1). Failures -> docs\runbook.md.
$ErrorActionPreference = "Stop"
$nsTrain = "training"; $nsModels = "models"
$predictHost = "mnist-demo.models.example.com"
$predictUrl = "http://localhost:8080/v1/models/mnist-demo:predict"

function Banner($m) { Write-Host ""; Write-Host ("=" * 60); Write-Host ">> $m"; Write-Host ("=" * 60) }

function Wait-Phase($name, $want, $timeout = 600) {
  $deadline = (Get-Date).AddSeconds($timeout)
  while ($true) {
    $p = kubectl -n $nsTrain get trainingjob $name -o jsonpath='{.status.phase}' 2>$null
    Write-Host "   $name: $p"
    if ($p -eq $want) { return }
    if ($p -eq "Failed" -and $want -ne "Failed") { Write-Error "$name Failed unexpectedly" }
    if ((Get-Date) -gt $deadline) { Write-Error "timeout waiting for $name=$want" }
    Start-Sleep -Seconds 5
  }
}

Banner "1/6  GPUQuota (hardGPUs=2) + three 1-GPU TrainingJobs -> watch queueing"
kubectl apply -f gitops/apps/training-demo/gpuquota.yaml
foreach ($n in @("mnist-a", "mnist-b", "mnist-c")) {
  $tj = @"
apiVersion: mlplatform.fyp.io/v1alpha1
kind: TrainingJob
metadata:
  name: $n
  namespace: $nsTrain
spec:
  model: { name: mnist-cnn }
  dataset: { name: mnist }
  hyperparameters: { epochs: "2", batchSize: "128", learningRate: "0.001" }
  gpus: 1
  retry: { backoffLimit: 2 }
  ttlSecondsAfterFinished: 3600
  output: { bucket: models }
"@
  $tj | kubectl apply -f -
}
Start-Sleep -Seconds 8
Write-Host "   With 2 GPUs and three 1-GPU jobs, exactly one should be Queued:"
kubectl -n $nsTrain get trainingjobs
kubectl -n $nsTrain get gpuquota default

Banner "2/6  Wait for the first job to finish training"
Wait-Phase "mnist-a" "Succeeded" 900

Banner "3/6  Model artifact in MinIO"
kubectl -n minio delete pod mc-stat --ignore-not-found 2>$null
kubectl -n minio run mc-stat --rm -i --restart=Never --image=quay.io/minio/mc:RELEASE.2025-04-16T18-13-26Z -- `
  sh -c 'mc alias set m http://minio.minio.svc.cluster.local:9000 minioadmin minioadmin >/dev/null && mc ls -r m/models/training/mnist-a/'

Banner "4/6  Deploy InferenceService (KServe, scale-to-zero)"
(Get-Content gitops/apps/inference-demo/inferenceservice.yaml) -replace "training/mnist-demo", "training/mnist-a" | kubectl apply -f -
Write-Host "   waiting for InferenceService Ready (storage-initializer pull can take a minute)..."
kubectl -n $nsModels wait --for=condition=Ready inferenceservice/mnist-demo --timeout=600s

Banner "5/6  Predict (traffic enters via Kourier on localhost:8080)"
curl.exe -s -H "Host: $predictHost" -H "Content-Type: application/json" `
  --data-binary "@serving/samples/sample-request.json" $predictUrl
Write-Host ""

Banner "6/6  Idle -> scale to zero, then cold start"
Write-Host "   waiting up to 3 min for predictor replicas -> 0 ..."
$deadline = (Get-Date).AddSeconds(180)
while ($true) {
  $reps = kubectl -n $nsModels get deploy -l serving.knative.dev/service=mnist-demo-predictor -o jsonpath='{.items[*].status.replicas}' 2>$null
  Write-Host "   replicas: $reps"
  if (-not $reps -or $reps -eq "0") { Write-Host "   >> scaled to zero (GPU idle cost avoided)"; break }
  if ((Get-Date) -gt $deadline) { Write-Host "   (did not reach zero in time; check autoscaler window)"; break }
  Start-Sleep -Seconds 10
}
Write-Host "   cold-start request (timed):"
curl.exe -s -o NUL -w "   HTTP %{http_code} in %{time_total}s`n" `
  -H "Host: $predictHost" -H "Content-Type: application/json" `
  --data-binary "@serving/samples/sample-request.json" $predictUrl

Banner "Demo complete. Grafana 'Cost & Reliability' dashboard shows the GPU-hours story."
