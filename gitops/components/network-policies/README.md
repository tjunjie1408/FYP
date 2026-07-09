# Cilium network policies

**Posture: default-deny only in workload namespaces (`training`, `models`,
`minio`) — never in platform namespaces** (knative-serving, kserve,
cert-manager, monitoring, argocd, argo, mlops-system).

Why the exemption:
- Admission **webhooks** (apiserver → cert-manager/KServe/Knative pods) are the
  #1 thing default-deny breaks. On kind the apiserver is a host-network pod,
  so its traffic appears as the `host`/`kube-apiserver` *entity*, not a pod
  selector — easy to miss.
- On **scale-from-zero**, ALL traffic to a KServe predictor arrives via the
  Knative **activator** (knative-serving ns), not from the original client.
  Policies that only allow the caller namespace silently break cold starts.
- queue-proxy (in the predictor pod) pushes metrics to the Knative
  **autoscaler** — that egress must stay open from `models`.

Debugging dropped flows:

    kubectl -n kube-system exec ds/cilium -- hubble observe --verdict DROPPED -f

Policies sync at wave 4 (after the platform is healthy) so a policy mistake
never masks an install problem. Cilium evaluates deny before allow — stick to
allow-lists + default-deny, never explicit deny rules.
