# Live Demo Script (target < 10 min)

## Pre-demo checklist (the night before)
- [ ] `scripts/up.sh` from a clean machine; all Argo CD apps Healthy.
- [ ] `hack/e2e-smoke.sh` passes end to end (this *is* the rehearsal).
- [ ] **Pre-generate `make report`** + screenshot all 4 Grafana dashboards +
      one `hubble observe --verdict DROPPED` capture. The live demo must never
      depend on the monitoring stack being up (see degradation ladder).
- [ ] WSL2 at 16 GB+; close other memory hogs.
- [ ] Two terminals: one for `kubectl get trainingjobs -w`, one for commands.

## The narrative (what to say)

**0:00 — The thesis.** "Models aren't the point; platform capability is. This
is a self-service ML platform: a researcher submits one YAML, the platform does
quota, scheduling, training, serving, scale-to-zero, and observability — no
proprietary cloud lock-in."

**0:30 — One-click origin.** Show `terraform/envs/local` + Argo CD UI.
"Terraform installs only Cilium and Argo CD; everything else is GitOps in sync
waves. This is the whole platform as code."

**1:30 — Submit work + quota queueing.** Run `scripts/demo.sh` (or by hand):
apply the GPUQuota (2 GPUs) and three 1-GPU TrainingJobs.
> Point at the `kubectl get trainingjobs -w` terminal: two go `Running`, the
> third sits **`Queued`** — the operator's quota gate. "This is admission-free
> queueing: no rejected jobs, capacity-aware scheduling. Watch it start the
> moment a peer finishes."

**3:00 — The operator did real Kubernetes work.** `kubectl describe trainingjob
mnist-a` — show phases, conditions (`QuotaGranted`, `WorkflowSubmitted`,
`Complete`), the owned Argo Workflow, events. "A Kubebuilder controller:
reconcile loop, finalizers, owner refs, status conditions — production patterns."

**4:30 — Artifact + serving.** Model lands in MinIO; apply the InferenceService.
`kubectl get inferenceservice` → Ready. Predict with the Host-header curl.
"KServe pulled the model from in-cluster S3 and is serving it."

**6:00 — Scale-to-zero (the cost story).** Wait idle; show predictor replicas
1 → 0. "No traffic, no GPU cost. The next request cold-starts." Fire one more
request, show the timed cold start — "that latency is the *price* of the
saving; the report quantifies the trade-off."

**7:30 — Observability + report.** Grafana GPU Fleet + Cost & Reliability
dashboards (live if RAM allows, else screenshots). "Idle GPU-hours, projected
cost reduction. The GPUs are simulated — I'm explicit about that — but the
scheduling behaviour being measured is real."

**9:00 — Security & wrap.** Show one `CiliumNetworkPolicy` + a Hubble drop:
"training pods can reach MinIO but not the internet; default-deny per
namespace." Close on the thesis.

## Demo-day degradation ladder

Core path that must survive intact: **TrainingJob → operator queueing/phases →
Argo Workflow → model in MinIO → KServe predict → scale-to-zero.** If RAM
threatens, shed in this order:

1. **Default posture:** all report artifacts and dashboard screenshots
   pre-generated days before — the live demo never needs the monitoring stack.
2. **Drop Grafana live** (~300 Mi): use the screenshots; keep Prometheus if it
   fits.
3. **Drop kube-prometheus-stack entirely** (~1.2 Gi, biggest single recovery):
   fake-dcgm still serves `/metrics` — `curl` it live to prove metrics exist;
   cost evidence is the pre-generated report. Zero impact on the core path.
4. **Time-share:** run in two acts — Act 1 train+queue with monitoring scaled
   to 0 (`kubectl -n monitoring scale deploy --all --replicas=0`); Act 2 scale
   it back up after training pods finish to show live dashboards.
5. **Last resort:** switch the demo venue to GKE (`terraform/envs/gke`) — it
   exists by design; ~2 hours of T4 spot cost for the whole session.

**Never degraded:** the operator, Argo Workflows, MinIO, KServe/Knative,
Cilium. They *are* the demo.

## If something breaks live
- Predict 404/timeout → Host header? Kourier NodePort mapped? (runbook: kind ingress)
- InferenceService stuck Init → storage-initializer / S3 secret (runbook).
- A job stuck Queued forever → check `kubectl get gpuquota default -n training`
  usage; a peer may not have completed.
- Talk through it — diagnosing live *is* platform-engineering competence.
