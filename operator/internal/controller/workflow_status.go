package controller

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	mlv1alpha1 "github.com/tjunjie1408/FYP/operator/api/v1alpha1"
)

// MapWorkflowPhase maps an Argo Workflow status.phase to a TrainingJob phase.
//
// Retry-layer isolation rule: the operator reacts ONLY to terminal workflow
// phases (Succeeded, Failed, Error). Everything else — including "Running"
// while an internal step is failing and being retried by Argo's own
// retryStrategy — maps to Running and consumes nothing. The operator's
// backoffLimit is spent only when a whole Workflow terminates Failed/Error.
func MapWorkflowPhase(wfPhase string) (phase mlv1alpha1.TrainingJobPhase, terminal bool) {
	switch wfPhase {
	case "Succeeded":
		return mlv1alpha1.PhaseSucceeded, true
	case "Failed", "Error":
		return mlv1alpha1.PhaseFailed, true
	default: // "", Pending, Running, ...
		return mlv1alpha1.PhaseRunning, false
	}
}

// WorkflowPhase extracts status.phase from an unstructured Workflow.
func WorkflowPhase(wf *unstructured.Unstructured) string {
	phase, _, _ := unstructured.NestedString(wf.Object, "status", "phase")
	return phase
}
