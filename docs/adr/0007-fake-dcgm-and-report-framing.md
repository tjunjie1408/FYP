# ADR-0007: Simulated DCGM metrics, and honest cost-report framing

**Status:** Accepted

## Context
No local GPUs. We want credible GPU dashboards and a cost report, without
overclaiming. There is a real risk of **circular reasoning**: using data from
our own simulator to "prove" our own scheduler saves real money.

## Decision
Ship a fake DCGM exporter that emits real DCGM metric names with synthetic
values driven by actual pod lifecycle events. Frame the report as
**"Verification of platform scheduling capabilities based on deterministic
simulation metrics"** — not "we saved X% of real cost".

## Rationale
- The fake exporter emits `DCGM_FI_DEV_*` with the real label schema, so public
  DCGM Grafana dashboards work unmodified.
- The crucial distinction:
  - **Measured (real):** scheduling responses — quota queueing, GPU
    assignment/release events, replica transitions, cold-start latency. These
    are genuine control-plane behaviours regardless of whether the GPU is
    physical.
  - **Projected (assumption-based):** cost figures, computed from idle-GPU-hours
    × public T4 pricing. Disclosed up front in METHODOLOGY.md.
- The methodology is identical to one run against real DCGM data; only the
  metric source is synthetic. That framing survives viva scrutiny.

## Consequences
- The report headline is labeled a *projection*, with sensitivity analysis.
- Optional M9 stretch: rewrite the exporter in Rust for a footprint/latency
  comparison (a language-level highlight, not on the critical path).
