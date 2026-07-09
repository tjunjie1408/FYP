# Runbook / Troubleshooting

## Resource pressure (the #1 risk)

The full stack steady-states at ~5.5–7 GiB but **bursts** (training pods +
Prometheus scrape + image pulls). Plus Windows host overhead → **12 GB is
OOMKiller territory**. Give WSL2 **16 GB minimum, 24 GB safe**:

`%UserProfile%\.wslconfig`:
```ini
[wsl2]
memory=16GB        # 24GB if you have it
processors=6
swap=8GB
```
Then `wsl --shutdown` and restart Docker Desktop.

**OOM symptoms:** pods `OOMKilled` / `CrashLoopBackOff`, Argo CD apps flapping
Healthy↔Degraded, kubelet evictions (`kubectl get events -A | grep Evicted`).
**If it can't fit:** promote the GKE module (`terraform/envs/gke`) to the demo
target — decide by end of M1, not on demo day. See the degradation ladder in
[demo-script.md](demo-script.md).

## Cilium on kind

```bash
kubectl -n kube-system exec ds/cilium -- cilium status
kubectl -n kube-system exec ds/cilium -- cilium-dbg status --verbose
# Find dropped flows when a NetworkPolicy is too strict:
kubectl -n kube-system exec ds/cilium -- hubble observe --verdict DROPPED -f
```
Nodes stuck `NotReady` right after `terraform apply` is **expected** until the
Cilium helm release finishes — that's why Cilium is a Terraform step, not a
GitOps one.

To debug a policy without enforcing it, set Cilium to audit mode:
`--set policyAuditMode=true` (helm upgrade), inspect Hubble, then re-enforce.

## Knative / KServe

### storage-initializer hangs (pod stuck in Init)
The classic failure. Almost always the S3 secret annotations:
- `serving.kserve.io/s3-endpoint` must have **no scheme** (`minio.minio.svc.cluster.local:9000`, not `http://...`).
- `serving.kserve.io/s3-usehttps: "0"` for plain-HTTP MinIO.
- Secret must be attached to the predictor's ServiceAccount (`kserve-s3-sa`).

Smoke-test the credentials independently before blaming KServe:
```bash
kubectl -n models run s3test --rm -i --restart=Never --image=quay.io/minio/mc:RELEASE.2025-04-16T18-13-26Z -- \
  sh -c 'mc alias set m http://minio.minio.svc.cluster.local:9000 minioadmin minioadmin && mc ls m/models'
```

### scale-from-zero never serves (cold start hangs)
On scale-from-zero, traffic routes through the Knative **activator** in
`knative-serving`, not from the original caller. A NetworkPolicy on `models`
that only allows the caller namespace silently breaks this. The
`models-allow-knative-dataplane` policy allows the activator in — don't remove
it.

### kind ingress / Host header
There is no LoadBalancer on kind. Kourier is a NodePort (31080/31443) mapped to
host 8080/8443 by `extra_port_mappings`. Always send the Host header:
```bash
curl -H "Host: mnist-demo.models.example.com" \
  http://localhost:8080/v1/models/mnist-demo:predict -d @serving/samples/sample-request.json
```

### Knative metric names (Grafana panels empty)
Knative ≥1.19 moved to OpenTelemetry naming; `activator_request_count` etc. may
differ from older blog posts. Verify against the deployed version; the
kube-state-metrics replica panels (`kube_deployment_status_replicas`) are the
reliable fallback and carry the cost story regardless.

## M5 bisection protocol (Knative × Cilium × kind)

When serving / scale-from-zero misbehaves, **prove the serving path on a
Cilium-free kind cluster first**, then on the real cluster. Any failure then
bisects cleanly to Cilium/eBPF vs KServe/Knative config:

```bash
kind create cluster --name kserve-check          # default kindnet, no Cilium
# follow KServe quickstart (cert-manager → knative → kourier → kserve),
# deploy one InferenceService from MinIO, confirm predict + scale-to-zero,
# THEN delete and validate on the Cilium cluster.
kind delete cluster --name kserve-check
```

## Argo CD CRD races
Symptom: an Application Degraded with "no matches for kind ... ServiceMonitor".
Cause: a CR synced before its CRD. Mitigations already in place: sync waves +
`ServerSideApply=true` + app retry (limit 10). Just wait for the retry, or
`argocd app sync <name>`.

## Windows specifics
- `.gitattributes` forces LF on `*.sh` — never commit CRLF shell scripts (they
  fail inside containers with `\r: command not found`).
- `make` lives in WSL2 / Git Bash; the `*.ps1` mirrors cover the demo from a
  native PowerShell terminal.
- `setup-envtest` trips the UAC "this looks like an installer" heuristic on
  Windows — install it and **rename the exe** (see `operator/Makefile`).

## MinIO image pin
Community MinIO went source-only / stopped publishing free Docker images in
late 2025. We pin `RELEASE.2025-04-22T22-12-26Z` (last full console image).
If it ever vanishes, SeaweedFS or Garage are S3-compatible drop-ins (ADR-0006).
