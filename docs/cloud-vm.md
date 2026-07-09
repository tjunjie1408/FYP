# Running the full platform on a cloud VM

The full stack (Cilium + Knative + KServe + Prometheus + Argo + MinIO + training
pods) needs ~6–7 GB steady-state and bursts higher. On a 16 GB Windows laptop
that's OOM-risky for a graded live demo. Instead, run the **exact same local
kind setup** on a 32 GB cloud VM — "a bigger laptop." Same code path, no new
failure modes; ~$0.27/h, and a 2-hour session is covered by GCP's $300 free
credit.

> Why a VM running kind, not GKE? Identical to local (`terraform/envs/local`),
> so nothing new can break. Choose GKE (`terraform/envs/gke`) only if you
> specifically want a managed-Kubernetes story for your viva.

## Prerequisites (on your laptop, not the VM)

- A GCP project + billing enabled (free credit is fine).
- `gcloud` CLI, authenticated: `gcloud auth login && gcloud auth application-default login`
- `terraform` (only to provision the VM; the VM installs its own copy).

## 1. Provision the VM

```bash
cd terraform/envs/vm
terraform init
terraform apply -var project_id=YOUR_GCP_PROJECT
# (add -var use_spot=true for ~70% cheaper dev runs; keep false for the demo)
terraform output next_steps
```

The VM auto-installs Docker, kind, kubectl, helm, terraform, and Go via its
startup script (~3 min). Confirm it finished:

```bash
gcloud compute ssh fyp-platform --zone us-central1-a --command 'cat /var/log/fyp-bootstrap-done'
```

## 2. Bring up the platform (on the VM)

```bash
gcloud compute ssh fyp-platform --zone us-central1-a

# one-time: let your user run docker without sudo
sudo usermod -aG docker $(whoami) && newgrp docker

git clone https://github.com/tjunjie1408/FYP && cd FYP
make up GITOPS_REPO_URL=https://github.com/tjunjie1408/FYP
```

`make up` runs `terraform/envs/local` *inside the VM* — kind, Cilium, Argo CD,
then the GitOps waves. Argo CD syncs from your GitHub fork, so make sure it's
pushed and `gitops_repo_url` points at it.

## 3. Access the UIs from your laptop (SSH tunnel)

No inbound demo ports are opened (safer). Forward them over SSH instead — copy
the `ssh_with_tunnels` Terraform output, e.g.:

```bash
gcloud compute ssh fyp-platform --zone us-central1-a -- \
  -L 8443:localhost:8443 -L 3000:localhost:3000 \
  -L 9001:localhost:9001 -L 8080:localhost:8080
```

Then on your laptop browser:
- Argo CD  → https://localhost:8443
- Grafana  → http://localhost:3000  (admin/admin)
- MinIO    → http://localhost:9001  (minioadmin/minioadmin)
- Predict  → http://localhost:8080  (with `Host: mnist-demo.models.example.com`)

The Argo CD password and port-forwards work exactly as in the local runbook —
run `scripts/port-forward.sh <ui>` on the VM if a service isn't already on the
expected localhost port.

## 4. Run the demo / smoke test (on the VM)

```bash
hack/e2e-smoke.sh     # rehearsal — must pass before the demo
scripts/demo.sh       # the live demo driver
make report           # cost/reliability report
```

## 5. Cost control — STOP or DESTROY when idle

```bash
# Stop (keeps the disk; ~$0.01/h for storage, fast restart):
gcloud compute instances stop fyp-platform --zone us-central1-a
gcloud compute instances start fyp-platform --zone us-central1-a   # resume

# Or destroy everything (no further cost):
terraform -chdir=terraform/envs/vm destroy -var project_id=YOUR_GCP_PROJECT
```

A `stop` between work sessions is the cheapest habit; the kind cluster and your
clone survive a stop/start. A full reboot of the VM loses the kind cluster
(it's in Docker) — re-run `make up`, which is idempotent.

## Day-to-day dev stays local

You don't need this VM to develop the operator — that's a ~2 GB footprint your
laptop handles easily. Use `scripts/dev-slice.sh` locally and reserve the VM
for full-stack integration and the demo. See [local-dev.md](local-dev.md).
