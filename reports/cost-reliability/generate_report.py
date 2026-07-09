"""Generate the Cost & Reliability report from Prometheus.

Queries query_range over a window, computes GPU-hours / idle-hours / projected
cost, and writes output/report.md + PNG charts. See METHODOLOGY.md for the
measured-vs-projected framing — cost numbers are labeled PROJECTION.

Usage:
  python generate_report.py --prometheus-url http://localhost:9090 --window 2h
"""

import argparse
import os
import time

import requests

try:
    import matplotlib

    matplotlib.use("Agg")
    import matplotlib.pyplot as plt

    HAVE_MPL = True
except ImportError:  # report still works without charts
    HAVE_MPL = False

T4_HOURLY_USD = 0.35


def parse_window(w: str) -> int:
    units = {"s": 1, "m": 60, "h": 3600, "d": 86400}
    return int(w[:-1]) * units[w[-1]]


def query_range(base, expr, start, end, step):
    try:
        r = requests.get(
            f"{base}/api/v1/query_range",
            params={"query": expr, "start": start, "end": end, "step": step},
            timeout=30,
        )
        r.raise_for_status()
        data = r.json()["data"]["result"]
        if not data:
            print(f"   (empty result) {expr}")
        return data
    except Exception as e:  # noqa: BLE001
        print(f"   (query failed) {expr}: {e}")
        return []


def series_to_xy(result):
    """First series -> (timestamps, values)."""
    if not result:
        return [], []
    pairs = result[0]["values"]
    xs = [float(p[0]) for p in pairs]
    ys = [float(p[1]) for p in pairs]
    return xs, ys


def integral_gpu_hours(result, step_secs):
    """Sum across all series of (value * step) / 3600 — a Riemann GPU-hours sum."""
    total = 0.0
    for s in result:
        for _, v in s["values"]:
            total += float(v) * step_secs / 3600.0
    return total


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--prometheus-url", default="http://localhost:9090")
    p.add_argument("--window", default="2h")
    p.add_argument("--step", default="60s")
    p.add_argument("--output-dir", default=os.path.join(os.path.dirname(__file__), "output"))
    args = p.parse_args()

    os.makedirs(args.output_dir, exist_ok=True)
    end = int(time.time())
    start = end - parse_window(args.window)
    step_secs = parse_window(args.step)
    base = args.prometheus_url.rstrip("/")

    print(f">> querying {base} over {args.window} (step {args.step})")

    assigned = query_range(base, "sum(fyp_gpu_assigned)", start, end, args.step)
    util = query_range(base, "avg(DCGM_FI_DEV_GPU_UTIL)", start, end, args.step)
    idle_count = query_range(
        base, "count(fyp_gpu_assigned) - sum(fyp_gpu_assigned)", start, end, args.step
    )
    replicas = query_range(
        base,
        'sum(kube_deployment_status_replicas{namespace="models"})',
        start,
        end,
        args.step,
    )
    used_hours_series = query_range(
        base, "sum(DCGM_FI_DEV_GPU_UTIL / 100)", start, end, args.step
    )

    allocated_gph = integral_gpu_hours(assigned, step_secs)
    used_gph = integral_gpu_hours(used_hours_series, step_secs)
    idle_gph = max(0.0, allocated_gph - used_gph)
    projected_idle_cost = idle_gph * T4_HOURLY_USD

    # minutes at zero replicas
    zero_minutes = 0
    _, rep_ys = series_to_xy(replicas)
    for v in rep_ys:
        if v == 0:
            zero_minutes += step_secs / 60.0

    # charts
    charts = []
    if HAVE_MPL:
        for name, result, ylabel in [
            ("utilization", util, "avg GPU util %"),
            ("allocated", assigned, "GPUs allocated"),
            ("replicas", replicas, "inference replicas"),
        ]:
            xs, ys = series_to_xy(result)
            if not xs:
                continue
            t0 = xs[0]
            plt.figure(figsize=(8, 3))
            plt.plot([(x - t0) / 60 for x in xs], ys)
            plt.xlabel("minutes")
            plt.ylabel(ylabel)
            plt.title(name)
            plt.tight_layout()
            path = os.path.join(args.output_dir, f"{name}.png")
            plt.savefig(path, dpi=100)
            plt.close()
            charts.append(os.path.basename(path))
            print(f"   wrote {path}")

    report = f"""# Cost & Reliability Report

> Window: last {args.window} (step {args.step}). See METHODOLOGY.md for the
> measured-vs-projected distinction. **Cost figures below are PROJECTIONS**
> under stated assumptions (T4 @ ${T4_HOURLY_USD}/GPU-hour); GPUs are simulated.

## Measured scheduling behaviour (real)

| Metric | Value |
|---|---|
| Allocated GPU-hours | {allocated_gph:.2f} |
| Used GPU-hours (utilization-weighted) | {used_gph:.2f} |
| Idle GPU-hours (allocated − used) | {idle_gph:.2f} |
| Inference minutes at zero replicas | {zero_minutes:.0f} |

## Projected cost (assumption-based)

| Metric | Value |
|---|---|
| Projected idle cost over window | ${projected_idle_cost:.2f} |
| Projected idle cost / hour | ${(projected_idle_cost / max(parse_window(args.window) / 3600, 1e-9)):.3f} |

To produce the headline reduction %, run this over a Scenario-A window and a
Scenario-B window and compute `1 − idle_B / idle_A`. See METHODOLOGY.md.

## Charts

{os.linesep.join(f"![{c}]({c})" for c in charts) if charts else "_(no charts — matplotlib unavailable or empty queries)_"}

---
_Generated by reports/cost-reliability/generate_report.py_
"""
    out = os.path.join(args.output_dir, "report.md")
    with open(out, "w") as f:
        f.write(report)
    print(f">> wrote {out}")


if __name__ == "__main__":
    main()
