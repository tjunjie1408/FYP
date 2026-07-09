# ADR-0003: Operator talks to Argo Workflows via unstructured objects

**Status:** Accepted

## Context
The operator must create Argo `Workflow` objects and read their `status.phase`.
Two options: import the Argo Workflows Go module for typed objects, or use
`unstructured.Unstructured`.

## Decision
Use `unstructured.Unstructured` with the `argoproj.io/v1alpha1 Workflow` GVK.
Do not import `github.com/argoproj/argo-workflows`.

## Rationale
- The Argo module pins its own `k8s.io/*` and controller-runtime versions and
  is a notorious source of dependency conflicts with current controller-runtime.
- The operator's needs are tiny: build a Workflow that is just
  `workflowTemplateRef` + parameters, and read one string (`status.phase`).
  `unstructured.NestedString` covers this trivially.
- `Owns()` works fine with unstructured objects, so owner-ref watches still
  re-trigger reconciles without polling.

## Consequences
- No compile-time typing of the Workflow spec — mitigated by the pure,
  unit-tested `workflow_builder.go` and envtest with a vendored minimal Argo
  CRD.
- The pipeline logic lives in a `WorkflowTemplate` (installed by GitOps), so the
  operator only emits a thin parameterized pointer.
