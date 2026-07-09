package controller

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mlv1alpha1 "github.com/tjunjie1408/FYP/operator/api/v1alpha1"
)

func job(name string, gpus int32, phase mlv1alpha1.TrainingJobPhase) mlv1alpha1.TrainingJob {
	return mlv1alpha1.TrainingJob{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "training"},
		Spec:       mlv1alpha1.TrainingJobSpec{GPUs: gpus},
		Status:     mlv1alpha1.TrainingJobStatus{Phase: phase},
	}
}

func TestUsedGPUs(t *testing.T) {
	jobs := []mlv1alpha1.TrainingJob{
		job("a", 2, mlv1alpha1.PhaseRunning),
		job("b", 1, mlv1alpha1.PhaseRunning),
		job("c", 4, mlv1alpha1.PhaseQueued),    // queued does not count
		job("d", 8, mlv1alpha1.PhaseSucceeded), // terminal does not count
		job("e", 1, mlv1alpha1.PhaseFailed),
	}
	if got := UsedGPUs(jobs, ""); got != 3 {
		t.Errorf("UsedGPUs = %d, want 3", got)
	}
	// excluding a Running job removes its usage
	if got := UsedGPUs(jobs, "a"); got != 1 {
		t.Errorf("UsedGPUs excluding a = %d, want 1", got)
	}
}

func TestCountQueued(t *testing.T) {
	jobs := []mlv1alpha1.TrainingJob{
		job("a", 1, mlv1alpha1.PhaseQueued),
		job("b", 1, mlv1alpha1.PhaseRunning),
		job("c", 1, mlv1alpha1.PhaseQueued),
	}
	if got := CountQueued(jobs); got != 2 {
		t.Errorf("CountQueued = %d, want 2", got)
	}
}
