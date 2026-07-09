package controller

import (
	mlv1alpha1 "github.com/tjunjie1408/FYP/operator/api/v1alpha1"
)

// UsedGPUs sums spec.gpus over Running TrainingJobs, excluding the named job.
// Usage is computed fresh from a cached List on every reconcile — there are
// no persisted counters to corrupt.
func UsedGPUs(jobs []mlv1alpha1.TrainingJob, excludeName string) int32 {
	var used int32
	for i := range jobs {
		j := &jobs[i]
		if j.Name == excludeName {
			continue
		}
		if j.Status.Phase == mlv1alpha1.PhaseRunning {
			used += j.Spec.GPUs
		}
	}
	return used
}

// CountQueued counts TrainingJobs currently in phase Queued.
func CountQueued(jobs []mlv1alpha1.TrainingJob) int32 {
	var n int32
	for i := range jobs {
		if jobs[i].Status.Phase == mlv1alpha1.PhaseQueued {
			n++
		}
	}
	return n
}
