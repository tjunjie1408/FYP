# gke-cluster (optional)

Cloud fallback/primary-demo target when the laptop can't host the full stack
(see the demo-day degradation ladder in `docs/demo-script.md`).

- 1 × e2-standard-4 system node (~$0.13/h)
- 0..1 × n1-standard-4 + Tesla T4 **spot** GPU node (~$0.11–0.16/h, scales from zero)
- A full demo session is ~2 hours of spot cost.

Not applied by CI or `make up`; use `terraform -chdir=terraform/envs/gke apply`.
