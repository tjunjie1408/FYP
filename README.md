# End-to-End AI Platform Template

A self-contained MLOps platform demonstrating **Platform Engineering** capabilities:
*"Models are not important, platform capabilities are key."*

| Capability | Implementation |
|---|---|
| Training-job control plane | Custom **Kubebuilder (Go) operator**: `TrainingJob` + `GPUQuota` CRDs — queueing, retry, TTL |
| Pipeline orchestration | **Argo Workflows** (train → evaluate → upload) |
| GitOps delivery | **Argo CD** app-of-apps with sync waves |
| Model serving | **KServe** (Knative mode) with **scale-to-zero** |
| Observability | **Prometheus + Grafana + (simulated) DCGM** GPU metrics |
| Network security | **Cilium** CNI with default-deny `CiliumNetworkPolicy` in workload namespaces |
| One-click bootstrap | **Terraform** → kind cluster + Cilium + Argo CD → everything else via GitOps |
| Cost / reliability evidence | Prometheus-derived report: projected GPU idle-cost reduction, cold-start trade-off |

## Architecture

```
terraform apply
   └─► kind cluster (CNI-less, kube-proxy-less)
         └─► Cilium (helm, via Terraform)
               └─► Argo CD (helm, via Terraform)
                     └─► root Application (app-of-apps)
                           wave -1  namespaces + AppProject
                           wave  0  cert-manager
                           wave  1  kube-prometheus-stack
                           wave  2  MinIO · Knative Serving + Kourier
                           wave  3  KServe · Argo Workflows
                           wave  4  fake-dcgm-exporter · CiliumNetworkPolicies
                           wave  5  ml-operator (TrainingJob / GPUQuota CRDs)
                           wave  6  demo apps (WorkflowTemplate, TrainingJob, InferenceService)
```

End-to-end data path:

```
kubectl apply TrainingJob
  → operator: quota gate (Queued if over GPUQuota) → submits Argo Workflow
  → Workflow: train.py (PyTorch MNIST) → evaluate.py → upload.py → MinIO s3://models/...
  → InferenceService (KServe) pulls model from MinIO → serves → scales to zero when idle
  → fake DCGM exporter tracks simulated GPU assignment → Grafana dashboards → cost report
```

## Where to run it — pick by your hardware

The full stack needs ~6–7 GB steady-state and bursts higher. Match the venue to
the task:

| Task | Setup | ~RAM | Doc |
|---|---|---|---|
| Operator logic (most dev) | `make -C operator test` (in-memory apiserver) | <0.5 GB | [local-dev.md](docs/local-dev.md) |
| Operator on a real cluster | `scripts/dev-slice.sh` (kind + Argo WF + MinIO) | ~2 GB | [local-dev.md](docs/local-dev.md) |
| **Full stack / demo** | **cloud VM running kind** (`make vm-up`) | ~6–7 GB | [cloud-vm.md](docs/cloud-vm.md) |
| Full stack on ≥16 GB laptop | `make up` + degradation ladder | ~6–7 GB | [runbook.md](docs/runbook.md) |
| Managed-Kubernetes story | `terraform/envs/gke` (T4 spot) | n/a | — |

**Recommended for a 16 GB laptop:** develop locally with the dev-slice, and run
the full demo on a 32 GB cloud VM (`make vm-up` — ~$0.27/h, free with GCP's $300
credit, *identical* code path to local so nothing new can break).

## Prerequisites

- For the full stack locally: **Docker Desktop / WSL2 backend, 16 GB minimum, 24 GB safe**
  (see [runbook.md](docs/runbook.md) for the `.wslconfig`). 12 GB OOMs.
- Tooling: `terraform` ≥ 1.7, `kubectl`, `helm`, `kind` v0.32+, Go 1.24+, Python 3.11+,
  `jq` (used by `scripts/up.sh` / `hack/e2e-smoke.sh`). The cloud VM installs all of this for you.
- This repo pushed to a Git remote Argo CD can reach (set `gitops_repo_url`).

## Quickstart (3 commands)

```bash
# 1. Bring up cluster + Cilium + Argo CD + root app (≈ 8–10 min to all-Healthy)
terraform -chdir=terraform/envs/local init
terraform -chdir=terraform/envs/local apply -auto-approve -var gitops_repo_url=https://github.com/tjunjie1408/FYP

# 2. Watch the platform converge
scripts/port-forward.sh argocd   # https://localhost:8443 (password printed by terraform output)

# 3. Run the demo: train → queue → serve → scale-to-zero
scripts/demo.sh
```

`make report` generates the Cost/Reliability report from recorded Prometheus data —
see `reports/cost-reliability/METHODOLOGY.md` for the (deliberately careful) framing:
scheduling behavior is **measured**, cost figures are **projections** under stated
assumptions, because GPU metrics are simulated.

## Repository layout

```
terraform/    one-click bootstrap (kind default, GKE optional)
gitops/       Argo CD app-of-apps: platform Applications, values, components, demo apps
operator/     Kubebuilder project: TrainingJob + GPUQuota CRDs and controller
training/     PyTorch MNIST trainer image (train / evaluate / upload)
serving/      KServe custom predictor runtime
pipelines/    Argo WorkflowTemplate (the training pipeline contract)
components/   fake DCGM exporter source
charts/       in-repo helm chart for the fake DCGM exporter
dashboards/   Grafana dashboard JSON (auto-loaded via sidecar)
reports/      cost/reliability report methodology + generator
scripts/      up / demo / port-forward / load-test (.sh + .ps1)
hack/         e2e smoke test, replica measurement
docs/         architecture, runbook, demo script, ADRs
```

## Documentation

- [docs/architecture.md](docs/architecture.md) — components, state machine, contracts
- [docs/runbook.md](docs/runbook.md) — troubleshooting (Cilium, Knative, kind on WSL2)
- [docs/demo-script.md](docs/demo-script.md) — minute-by-minute live demo + degradation ladder
- [docs/adr/](docs/adr/) — architecture decision records
