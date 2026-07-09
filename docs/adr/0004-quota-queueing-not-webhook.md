# ADR-0004: GPU quota via reconciler queueing, not an admission webhook

**Status:** Accepted

## Context
We want per-namespace GPU limits. Options: a validating admission webhook that
*rejects* over-quota TrainingJobs, or a reconciler that *queues* them.

## Decision
Enforce quota in the reconciler. Over-quota jobs are admitted but parked in
`phase: Queued` until capacity frees. No webhooks anywhere in the operator;
field validation uses OpenAPI + CEL markers instead.

## Rationale
- Queueing is the better demo: "submit three jobs, watch the third wait, then
  start automatically when a peer finishes" — capacity-aware scheduling, not a
  rejection.
- No admission-time races: usage is recomputed from a cached List each reconcile
  (sum of `spec.gpus` over Running jobs), so there are no persisted counters to
  corrupt.
- Skipping webhooks keeps `make run` (operator as a local process against the
  cluster) viable — the fastest dev loop — with zero TLS/cert wiring.

## Consequences
- A job over quota is not rejected at submit time; users see `Queued` instead of
  an error. Documented as intended behaviour.
- Validation (enum model names, immutability) is done with
  `+kubebuilder:validation:XValidation` CEL rules and OpenAPI markers.
- Queued jobs re-check on a 15s requeue backstop (plus event-driven reconciles
  when peers complete) — invisible latency in a demo.
