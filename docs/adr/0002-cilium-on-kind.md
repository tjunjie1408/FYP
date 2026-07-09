# ADR-0002: Cilium as CNI on kind, with platform-namespace policy exemption

**Status:** Accepted

## Context
The brief mandates Cilium network policies. Cilium on kind interacts with
Knative/KServe's dataplane (activator hop, webhooks) in ways that are easy to
break, and kind has no LoadBalancer.

## Decision
Use Cilium with `disable_default_cni` + `kubeProxyReplacement`. Apply
**default-deny only in workload namespaces** (`training`, `models`, `minio`);
never in platform namespaces. Sync policies at wave 4 (after the platform is
healthy).

## Rationale
- Admission webhooks (apiserver → cert-manager/KServe/Knative) break under
  default-deny; on kind the apiserver appears as the `host`/`kube-apiserver`
  entity, not a pod selector — easy to miss.
- Scale-from-zero routes through the Knative activator; policies must allow the
  `knative-serving` dataplane into `models`.
- Wave-4 ordering means a policy mistake never masks an install failure.

## Consequences
- Platform namespaces are not micro-segmented (acceptable for an FYP; noted as
  future work).
- Debugging relies on Hubble (`--verdict DROPPED`) — itself a demo artifact.
- Fallback if Cilium blocks serving irrecoverably: validate on a Cilium-free
  kind cluster first (M5 bisection protocol) to localize the fault.
