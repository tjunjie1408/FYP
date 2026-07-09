# Local development (16 GB laptop friendly)

You do **not** need the full platform running to develop the centerpiece — the
operator. Two light loops cover almost all day-to-day work; reserve the heavy
full stack for the cloud VM (see [cloud-vm.md](cloud-vm.md)).

## Loop 1 — no cluster at all (fastest)

The reconcile logic is covered by unit + envtest tests that spin up an
in-memory API server (no Docker, no kind):

```bash
make -C operator test
```

This exercises submit→Running, Succeeded+modelURI, retry→attempt-2, backoff
exhausted→Failed, quota Queued→Running, TTL deletion, finalizer cleanup, and
the step-retry-churn isolation case. Use this for almost all operator changes.

## Loop 2 — dev-slice (~2 GB, real cluster)

When you want to watch a real `TrainingJob` create a real Argo Workflow that
actually runs `train.py`:

```bash
scripts/dev-slice.sh            # kind + Argo Workflows + MinIO + CRDs + pipeline
make images load                # build + load the trainer image (one-time-ish)
make -C operator run            # run the operator locally against the cluster
# in another shell:
kubectl apply -f operator/config/samples/mlplatform_v1alpha1_trainingjob.yaml
kubectl -n training get trainingjobs,workflows -w
```

What the dev-slice deliberately omits (to stay light): Cilium, Knative, KServe,
Prometheus/Grafana. It uses kind's default CNI, so there are no network
policies to debug. Footprint ≈ 2 GB — comfortable on 16 GB.

Tear down: `make dev-slice-down`.

## Footprint comparison

| Setup | Components | ~RAM | Where |
|---|---|---|---|
| `make -C operator test` | envtest (in-memory apiserver) | <0.5 GB | laptop |
| `scripts/dev-slice.sh` | kind + Argo WF + MinIO | ~2 GB | laptop |
| `make up` (full) | + Cilium, Knative, KServe, Prometheus | ~6–7 GB (bursts more) | **cloud VM** |

## What still needs the full stack (→ cloud VM)

- KServe serving + scale-to-zero (the cost-report centerpiece)
- Cilium network policies + Hubble drops
- Grafana dashboards / live Prometheus
- `hack/e2e-smoke.sh` and the full `scripts/demo.sh`

Develop and unit-test everything locally; do integration + the demo on the VM.
