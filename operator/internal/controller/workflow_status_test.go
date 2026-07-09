package controller

import (
	"testing"

	mlv1alpha1 "github.com/tjunjie1408/FYP/operator/api/v1alpha1"
)

func TestMapWorkflowPhase(t *testing.T) {
	cases := []struct {
		wfPhase  string
		want     mlv1alpha1.TrainingJobPhase
		terminal bool
	}{
		{"Succeeded", mlv1alpha1.PhaseSucceeded, true},
		{"Failed", mlv1alpha1.PhaseFailed, true},
		{"Error", mlv1alpha1.PhaseFailed, true},
		// Retry-layer isolation: every non-terminal Argo phase — including
		// Running while internal steps retry — maps to Running.
		{"Running", mlv1alpha1.PhaseRunning, false},
		{"Pending", mlv1alpha1.PhaseRunning, false},
		{"", mlv1alpha1.PhaseRunning, false},
	}
	for _, c := range cases {
		got, terminal := MapWorkflowPhase(c.wfPhase)
		if got != c.want || terminal != c.terminal {
			t.Errorf("MapWorkflowPhase(%q) = (%v, %v), want (%v, %v)",
				c.wfPhase, got, terminal, c.want, c.terminal)
		}
	}
}
