# Cost & Reliability — Methodology

## Title

**Verification of Platform Scheduling Capabilities Based on Deterministic
Simulation Metrics.**

Deliberately *not* "this platform saved X% of GPU cost." GPUs here are
simulated; using our own simulator's numbers to prove our own scheduler saves
real money would be circular. Instead we separate what is measured from what is
projected.

## Measured vs projected

| Quantity | Status | Source |
|---|---|---|
| GPU assignment / release events | **Measured (real)** | Actual pod lifecycle (fake DCGM watches real pods) |
| Quota queue wait time (Queued → Running) | **Measured (real)** | Operator status transitions |
| Predictor replica transitions (1↔0) | **Measured (real)** | kube-state-metrics |
| Cold-start latency (p95) | **Measured (real)** | Knative / request timing |
| Container restarts, recovery time | **Measured (real)** | kube-state-metrics |
| GPU utilization values | Simulated (deterministic) | Fake DCGM exporter |
| GPU-hours, idle-hours | Derived from simulated util + real assignment | PromQL integrals |
| **$ cost / cost reduction %** | **Projected (assumption-based)** | idle-hours × public T4 price |

The key honest sentence: *the metric **source** is synthetic, but the
**scheduling responses** being verified are real; the same methodology applied
to real DCGM data yields the cost result.*

## Pricing assumptions

- NVIDIA T4: **$0.35 / GPU-hour** on-demand; **$0.11–0.16 / GPU-hour** spot
  (GCP `us-central1`, cite live pricing page in the final report).
- Fleet: 4 simulated T4s.

## Scenarios

- **Scenario A — baseline (anti-pattern):** inference as an always-on
  Deployment holding 1 GPU 24/7; training jobs launched ad hoc with no
  queueing (GPUs held idle between jobs).
- **Scenario B — optimized:** KServe scale-to-zero (GPU held only while
  serving + cold-start window) + operator quota/queueing (GPUs released at job
  completion; over-quota jobs wait instead of over-provisioning).

Both scenarios use the same PromQL.

## PromQL inventory

```promql
# allocated GPU-hours (assignment integral) over the window
sum(sum_over_time(fyp_gpu_assigned[$WINDOW:1m])) / 60

# used GPU-hours (utilization-weighted)
sum(sum_over_time((DCGM_FI_DEV_GPU_UTIL / 100)[$WINDOW:1m])) / 60

# idle GPU-hours (util < 5%)
sum(sum_over_time((DCGM_FI_DEV_GPU_UTIL < bool 5)[$WINDOW:1m])) / 60

# minutes at zero inference replicas
sum_over_time((sum(kube_deployment_status_replicas{namespace="models"}) == bool 0)[$WINDOW:1m])

# projected hourly idle cost (USD)
(count(fyp_gpu_assigned) - sum(fyp_gpu_assigned)) * 0.35
```

`idle = allocated − used`. Headline: **projected idle-cost reduction
% = 1 − idle_B / idle_A** (expect ~60–85% on inference via scale-to-zero,
~30–50% on training via queueing — let the measured data speak).

## Reliability half (fully real)

- Container restart counts across platform namespaces.
- Time-to-recover after a chaos poke (`kubectl delete pod` the predictor /
  operator).
- Cold-start p95 as the **cost of** scale-to-zero — the trade-off that makes
  the analysis honest.
- Argo CD sync/health history.

## Sensitivity analysis

Vary: on-demand vs spot pricing; traffic profile (bursty vs steady); idle
window length (Knative `window`). Report the cost-reduction range, not a single
number.

## Limitations

- GPUs are simulated; absolute cost figures are projections, not measurements.
- Single-node kind; no multi-node bin-packing effects.
- Demo windows are short and extrapolated to monthly figures — stated explicitly
  wherever extrapolation is used.

## Mechanics

`generate_report.py` queries the Prometheus HTTP API (`query_range`) over a
recorded window and emits `output/report.md` + matplotlib PNGs. Run via
`make report`. **Generate before demo day** — the live demo never depends on
the monitoring stack being up (see `docs/demo-script.md`).
