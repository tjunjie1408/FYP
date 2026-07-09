"""Fake DCGM exporter: a simulated fleet of 4 Tesla T4 GPUs.

Watches pods cluster-wide labeled `gpu.fyp/simulated: "true"` and first-fit
assigns each Running pod to a free simulated GPU, releasing it when the pod
finishes. Emits real dcgm-exporter metric names and label schema so public
DCGM Grafana dashboards work unmodified, plus `fyp_gpu_assigned` (0/1 per
GPU) which powers the cost report's allocated-GPU-hours integral.

Honest-methodology note (reports/cost-reliability/METHODOLOGY.md): the metric
VALUES are synthetic; the assignment/release EVENTS mirror real pod lifecycle
and are what the platform's scheduling behavior is verified against.
"""

import math
import os
import random
import threading
import time

from kubernetes import client, config, watch
from prometheus_client import Gauge, start_http_server

GPU_COUNT = int(os.environ.get("GPU_COUNT", "4"))
HOSTNAME = os.environ.get("NODE_NAME", "fyp-worker")
LABEL_SELECTOR = "gpu.fyp/simulated=true"
MODEL_NAME = "Tesla T4"
FB_TOTAL_MB = 15360  # T4: 16 GB framebuffer
UPDATE_INTERVAL_S = 5

DCGM_LABELS = ["gpu", "UUID", "device", "modelName", "Hostname", "namespace", "pod", "container"]

GPU_UTIL = Gauge("DCGM_FI_DEV_GPU_UTIL", "GPU utilization (%)", DCGM_LABELS)
FB_USED = Gauge("DCGM_FI_DEV_FB_USED", "Framebuffer used (MiB)", DCGM_LABELS)
FB_FREE = Gauge("DCGM_FI_DEV_FB_FREE", "Framebuffer free (MiB)", DCGM_LABELS)
POWER = Gauge("DCGM_FI_DEV_POWER_USAGE", "Power draw (W)", DCGM_LABELS)
TEMP = Gauge("DCGM_FI_DEV_GPU_TEMP", "GPU temperature (C)", DCGM_LABELS)
SM_CLOCK = Gauge("DCGM_FI_DEV_SM_CLOCK", "SM clock (MHz)", DCGM_LABELS)
ASSIGNED = Gauge("fyp_gpu_assigned", "1 if the simulated GPU is assigned to a pod", ["gpu", "UUID"])

ALL_DCGM = [GPU_UTIL, FB_USED, FB_FREE, POWER, TEMP, SM_CLOCK]


class Fleet:
    """4 simulated GPUs with stable UUIDs; first-fit pod assignment."""

    def __init__(self):
        self.lock = threading.Lock()
        self.uuids = [f"GPU-0000feed-cafe-beef-0000-00000000000{i}" for i in range(GPU_COUNT)]
        # gpu index -> {"uid", "namespace", "pod", "container", "since"}
        self.assignments = {}
        self.pod_to_gpu = {}

    def assign(self, uid, namespace, pod, container):
        with self.lock:
            if uid in self.pod_to_gpu:
                return
            for i in range(GPU_COUNT):
                if i not in self.assignments:
                    self.assignments[i] = {
                        "uid": uid,
                        "namespace": namespace,
                        "pod": pod,
                        "container": container,
                        "since": time.time(),
                    }
                    self.pod_to_gpu[uid] = i
                    print(f"assign gpu={i} pod={namespace}/{pod}", flush=True)
                    return
            print(f"no free GPU for pod {namespace}/{pod} (all {GPU_COUNT} busy)", flush=True)

    def release(self, uid):
        with self.lock:
            i = self.pod_to_gpu.pop(uid, None)
            if i is not None:
                a = self.assignments.pop(i)
                print(f"release gpu={i} pod={a['namespace']}/{a['pod']}", flush=True)

    def snapshot(self):
        with self.lock:
            return dict(self.assignments)


def watch_pods(fleet: Fleet):
    """Relist+watch loop; resilient to API timeouts and restarts."""
    v1 = client.CoreV1Api()
    while True:
        try:
            pods = v1.list_pod_for_all_namespaces(label_selector=LABEL_SELECTOR)
            seen = set()
            for p in pods.items:
                uid = p.metadata.uid
                if p.status.phase == "Running" and p.metadata.deletion_timestamp is None:
                    seen.add(uid)
                    fleet.assign(uid, p.metadata.namespace, p.metadata.name,
                                 p.spec.containers[0].name)
            for uid in list(fleet.pod_to_gpu):
                if uid not in seen:
                    fleet.release(uid)

            w = watch.Watch()
            for event in w.stream(v1.list_pod_for_all_namespaces,
                                  label_selector=LABEL_SELECTOR,
                                  resource_version=pods.metadata.resource_version,
                                  timeout_seconds=300):
                p = event["object"]
                uid = p.metadata.uid
                phase = p.status.phase if p.status else None
                gone = event["type"] == "DELETED" or phase in ("Succeeded", "Failed")
                if gone:
                    fleet.release(uid)
                elif phase == "Running" and p.metadata.deletion_timestamp is None:
                    fleet.assign(uid, p.metadata.namespace, p.metadata.name,
                                 p.spec.containers[0].name)
        except Exception as e:  # noqa: BLE001 — keep the watcher alive
            print(f"watch error, relisting in 5s: {e}", flush=True)
            time.sleep(5)


def update_metrics(fleet: Fleet):
    """Re-emit every series each tick; clear() first so released pods'
    labeled series disappear instead of going stale."""
    t = time.time()
    assignments = fleet.snapshot()
    for g in ALL_DCGM:
        g.clear()
    ASSIGNED.clear()

    for i, uuid in enumerate(fleet.uuids):
        a = assignments.get(i)
        ASSIGNED.labels(gpu=str(i), UUID=uuid).set(1 if a else 0)
        labels = {
            "gpu": str(i),
            "UUID": uuid,
            "device": f"nvidia{i}",
            "modelName": MODEL_NAME,
            "Hostname": HOSTNAME,
            "namespace": a["namespace"] if a else "",
            "pod": a["pod"] if a else "",
            "container": a["container"] if a else "",
        }
        if a:
            # Busy: high util with a slow sine wobble + noise — the pattern
            # that makes dashboard screenshots look like real training.
            util = max(0.0, min(100.0, 88 + 8 * math.sin(t / 45) + random.gauss(0, 3)))
            ramp = min(1.0, (t - a["since"]) / 120)  # data loading -> training
            fb_used = 1000 + ramp * 12000
            power = random.uniform(65, 70)
            temp = 55 + 10 * ramp + random.uniform(0, 3)
            clock = 1590
        else:
            util = random.uniform(0, 2)
            fb_used = random.uniform(0, 10)
            power = random.uniform(9, 12)
            temp = random.uniform(30, 35)
            clock = 300
        GPU_UTIL.labels(**labels).set(round(util, 1))
        FB_USED.labels(**labels).set(round(fb_used))
        FB_FREE.labels(**labels).set(round(FB_TOTAL_MB - fb_used))
        POWER.labels(**labels).set(round(power, 1))
        TEMP.labels(**labels).set(round(temp))
        SM_CLOCK.labels(**labels).set(clock)


def main():
    try:
        config.load_incluster_config()
    except config.ConfigException:
        config.load_kube_config()  # local dev fallback

    fleet = Fleet()
    threading.Thread(target=watch_pods, args=(fleet,), daemon=True).start()

    start_http_server(9400)
    print(f"fake-dcgm-exporter serving :9400 with {GPU_COUNT}x {MODEL_NAME} on {HOSTNAME}", flush=True)
    while True:
        update_metrics(fleet)
        time.sleep(UPDATE_INTERVAL_S)


if __name__ == "__main__":
    main()
