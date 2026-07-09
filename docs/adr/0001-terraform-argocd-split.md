# ADR-0001: Terraform installs only Cilium + Argo CD; everything else via GitOps

**Status:** Accepted

## Context
We need a one-click bootstrap. The two candidate boundaries: (a) Terraform
installs everything via `helm_release`, or (b) Terraform installs the minimum
and a GitOps tool installs the rest.

## Decision
Terraform installs **exactly two things** — Cilium and Argo CD — plus the
single root Application. Argo CD installs everything else (app-of-apps, sync
waves).

## Rationale
- With `disable_default_cni`, nodes are NotReady until a CNI exists; no pod —
  including Argo CD — can schedule. Cilium therefore *cannot* be GitOps-managed;
  it is a hard prerequisite. Argo CD is the same chicken-and-egg.
- Big charts (kube-prometheus-stack) in Terraform state are painful: apply
  timeouts, diff noise, CRD-upgrade breakage. Keeping them in Argo CD keeps
  Terraform state tiny.
- Argo CD gives self-heal, drift correction, sync-wave ordering, and a UI that
  doubles as the demo.

## Consequences
- `terraform destroy` won't gracefully uninstall Argo CD-managed apps — fine on
  kind (destroy deletes the whole container).
- The Git repo URL is a Terraform variable threaded into the root Application.
- Scaling path (mention in viva): ApplicationSet generators instead of explicit
  app-of-apps.
