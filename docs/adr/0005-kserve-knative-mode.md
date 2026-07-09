# ADR-0005: KServe in Knative (serverless) mode + Kourier, custom predictor

**Status:** Accepted

## Context
KServe runs in either Knative (serverless, scale-to-zero) or Standard
(RawDeployment + HPA) mode, with Istio or Kourier as the gateway, and serves
models via packaged runtimes or custom predictors.

## Decision
Knative mode + Kourier + a ~50-line custom Python predictor (`kserve` SDK).

## Rationale
- **Scale-to-zero is the cost-report centerpiece** — native and first-class in
  Knative. Standard mode + KEDA's HTTP add-on (for scale-from-zero) is still
  beta-quality and would be the flakiest part of the demo.
- **Kourier over Istio:** far smaller footprint (RAM matters on a laptop);
  matches KServe's own quickstart.
- **Custom predictor over TorchServe runtime:** TorchServe needs `.mar`
  archiving + `config.properties` layout — a classic demo-killer. A custom
  predictor loads `model.pt` directly from `/mnt/models` and keeps the image
  small (kserve + torch-cpu).

## Consequences
- Cold-start latency exists (the price of scale-to-zero) — explicitly measured
  and discussed in the report.
- The predictor's model class must stay in sync with `training/src/train.py`
  (noted in both files).
- Fallback if Knative×Cilium proves irrecoverable on kind: Standard mode + HPA
  (loses scale-to-zero — last resort only).
