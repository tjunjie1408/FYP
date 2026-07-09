package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	mlv1alpha1 "github.com/tjunjie1408/FYP/operator/api/v1alpha1"
)

const (
	finalizerName = "mlplatform.fyp.io/finalizer"
	// queuedRequeueInterval bounds how long a Queued job waits to re-check
	// quota. Peer completions also trigger reconciles via the Workflow watch,
	// so this is a backstop, not the primary signal.
	queuedRequeueInterval = 15 * time.Second
)

// TrainingJobReconciler reconciles a TrainingJob object.
type TrainingJobReconciler struct {
	client.Client
	Recorder record.EventRecorder
	// TemplateName is the WorkflowTemplate referenced by submitted Workflows.
	TemplateName string
}

// +kubebuilder:rbac:groups=mlplatform.fyp.io,resources=trainingjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mlplatform.fyp.io,resources=trainingjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=mlplatform.fyp.io,resources=trainingjobs/finalizers,verbs=update
// +kubebuilder:rbac:groups=mlplatform.fyp.io,resources=gpuquotas,verbs=get;list;watch
// +kubebuilder:rbac:groups=mlplatform.fyp.io,resources=gpuquotas/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=argoproj.io,resources=workflows,verbs=get;list;watch;create;delete;deletecollection
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *TrainingJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	job := &mlv1alpha1.TrainingJob{}
	if err := r.Get(ctx, req.NamespacedName, job); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// --- deletion: clean up owned Workflows, then release the finalizer ---
	if !job.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(job, finalizerName) {
			if err := r.deleteOwnedWorkflows(ctx, job); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(job, finalizerName)
			if err := r.Update(ctx, job); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(job, finalizerName) {
		controllerutil.AddFinalizer(job, finalizerName)
		if err := r.Update(ctx, job); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil // update re-triggers reconcile
	}

	// --- terminal phases: only TTL cleanup remains ---
	if job.IsTerminal() {
		return r.reconcileTTL(ctx, job)
	}

	// --- quota gate for jobs that have not started their current attempt ---
	if job.Status.Phase == "" || job.Status.Phase == mlv1alpha1.PhasePending || job.Status.Phase == mlv1alpha1.PhaseQueued {
		admitted, res, err := r.quotaGate(ctx, job)
		if err != nil || !admitted {
			return res, err
		}
	}

	// --- ensure the Workflow for the current attempt exists ---
	attempt := job.Status.Attempts
	creating := false
	if job.Status.Phase != mlv1alpha1.PhaseRunning {
		attempt++ // a new attempt is being started
		creating = true
	}
	if attempt == 0 { // Running but attempts unset: corrupted status; restart
		attempt, creating = 1, true
	}
	wfName := WorkflowName(job, attempt)

	wf := &unstructured.Unstructured{}
	wf.SetGroupVersionKind(WorkflowGVK())
	err := r.Get(ctx, types.NamespacedName{Namespace: job.Namespace, Name: wfName}, wf)
	switch {
	case apierrors.IsNotFound(err):
		wf = BuildWorkflow(job, attempt, r.TemplateName)
		if err := controllerutil.SetControllerReference(job, wf, r.Scheme()); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, wf); err != nil {
			if apierrors.IsAlreadyExists(err) {
				return ctrl.Result{Requeue: true}, nil
			}
			return ctrl.Result{}, err
		}
		log.Info("workflow submitted", "workflow", wfName, "attempt", attempt)
		r.Recorder.Eventf(job, corev1.EventTypeNormal, "WorkflowSubmitted", "Submitted workflow %s (attempt %d)", wfName, attempt)

		job.Status.Phase = mlv1alpha1.PhaseRunning
		if creating {
			// Attempts is incremented in exactly one place: here, on creation.
			job.Status.Attempts = attempt
		}
		job.Status.WorkflowName = wfName
		if job.Status.StartTime == nil {
			now := metav1.Now()
			job.Status.StartTime = &now
		}
		r.setCondition(job, mlv1alpha1.ConditionWorkflowSubmitted, metav1.ConditionTrue,
			mlv1alpha1.ReasonWorkflowSubmitted, fmt.Sprintf("workflow %s created", wfName))
		if err := r.Status().Update(ctx, job); err != nil {
			return ctrl.Result{}, err
		}
		r.updateQuotaStatus(ctx, job.Namespace)
		return ctrl.Result{}, nil

	case err != nil:
		return ctrl.Result{}, err
	}

	// --- propagate workflow status ---
	phase, terminal := MapWorkflowPhase(WorkflowPhase(wf))
	if !terminal {
		// Includes internal step-retry churn: still Running, nothing consumed.
		if job.Status.Phase != phase || job.Status.WorkflowName != wfName {
			job.Status.Phase = phase
			job.Status.WorkflowName = wfName
			if err := r.Status().Update(ctx, job); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	now := metav1.Now()
	switch phase {
	case mlv1alpha1.PhaseSucceeded:
		job.Status.Phase = mlv1alpha1.PhaseSucceeded
		job.Status.CompletionTime = &now
		job.Status.ModelURI = ModelURI(job)
		r.setCondition(job, mlv1alpha1.ConditionComplete, metav1.ConditionTrue,
			mlv1alpha1.ReasonTrainingSucceeded, "training pipeline succeeded")
		r.Recorder.Eventf(job, corev1.EventTypeNormal, "TrainingSucceeded", "Model uploaded to %s", job.Status.ModelURI)

	case mlv1alpha1.PhaseFailed:
		if job.Status.Attempts <= job.Spec.Retry.BackoffLimit {
			// Retry: delete the failed Workflow; the next reconcile passes the
			// quota gate again and creates attempt N+1.
			if err := r.Delete(ctx, wf); err != nil && !apierrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
			job.Status.Phase = mlv1alpha1.PhasePending
			r.Recorder.Eventf(job, corev1.EventTypeWarning, "Retrying",
				"Workflow %s failed; retrying (%d/%d retries used)", wfName, job.Status.Attempts, job.Spec.Retry.BackoffLimit+1)
			if err := r.Status().Update(ctx, job); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
		job.Status.Phase = mlv1alpha1.PhaseFailed
		job.Status.CompletionTime = &now
		r.setCondition(job, mlv1alpha1.ConditionComplete, metav1.ConditionFalse,
			mlv1alpha1.ReasonBackoffLimitExceeded,
			fmt.Sprintf("workflow failed after %d attempt(s)", job.Status.Attempts))
		r.Recorder.Eventf(job, corev1.EventTypeWarning, "TrainingFailed", "Workflow %s failed; backoff limit exceeded", wfName)
	}

	if err := r.Status().Update(ctx, job); err != nil {
		return ctrl.Result{}, err
	}
	r.updateQuotaStatus(ctx, job.Namespace)
	// Re-enter promptly so the TTL branch schedules itself.
	return ctrl.Result{Requeue: true}, nil
}

// quotaGate admits the job against the namespace GPUQuota, or parks it in
// Queued. Returns admitted=false with a requeue when the job must wait.
func (r *TrainingJobReconciler) quotaGate(ctx context.Context, job *mlv1alpha1.TrainingJob) (bool, ctrl.Result, error) {
	quota := &mlv1alpha1.GPUQuota{}
	err := r.Get(ctx, types.NamespacedName{Namespace: job.Namespace, Name: mlv1alpha1.GPUQuotaName}, quota)
	if apierrors.IsNotFound(err) {
		return true, ctrl.Result{}, nil // no quota => unlimited
	}
	if err != nil {
		return false, ctrl.Result{}, err
	}

	peers := &mlv1alpha1.TrainingJobList{}
	if err := r.List(ctx, peers, client.InNamespace(job.Namespace)); err != nil {
		return false, ctrl.Result{}, err
	}

	used := UsedGPUs(peers.Items, job.Name)
	if used+job.Spec.GPUs > quota.Spec.HardGPUs {
		if job.Status.Phase != mlv1alpha1.PhaseQueued {
			r.Recorder.Eventf(job, corev1.EventTypeWarning, "QuotaExceeded",
				"Queued: %d GPU(s) requested, %d/%d in use", job.Spec.GPUs, used, quota.Spec.HardGPUs)
		}
		job.Status.Phase = mlv1alpha1.PhaseQueued
		r.setCondition(job, mlv1alpha1.ConditionQuotaGranted, metav1.ConditionFalse,
			mlv1alpha1.ReasonQuotaExceeded,
			fmt.Sprintf("%d/%d GPUs in use; need %d", used, quota.Spec.HardGPUs, job.Spec.GPUs))
		if err := r.Status().Update(ctx, job); err != nil {
			return false, ctrl.Result{}, err
		}
		r.updateQuotaStatus(ctx, job.Namespace)
		return false, ctrl.Result{RequeueAfter: queuedRequeueInterval}, nil
	}

	r.setCondition(job, mlv1alpha1.ConditionQuotaGranted, metav1.ConditionTrue,
		mlv1alpha1.ReasonQuotaAvailable,
		fmt.Sprintf("%d/%d GPUs in use", used, quota.Spec.HardGPUs))
	return true, ctrl.Result{}, nil
}

// reconcileTTL deletes the job once ttlSecondsAfterFinished elapses.
func (r *TrainingJobReconciler) reconcileTTL(ctx context.Context, job *mlv1alpha1.TrainingJob) (ctrl.Result, error) {
	if job.Spec.TTLSecondsAfterFinished == nil || job.Status.CompletionTime == nil {
		return ctrl.Result{}, nil
	}
	expiry := job.Status.CompletionTime.Add(time.Duration(*job.Spec.TTLSecondsAfterFinished) * time.Second)
	if remaining := time.Until(expiry); remaining > 0 {
		return ctrl.Result{RequeueAfter: remaining}, nil
	}
	logf.FromContext(ctx).Info("TTL expired, deleting TrainingJob")
	if err := r.Delete(ctx, job); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	return ctrl.Result{}, nil
}

func (r *TrainingJobReconciler) deleteOwnedWorkflows(ctx context.Context, job *mlv1alpha1.TrainingJob) error {
	wf := &unstructured.Unstructured{}
	wf.SetGroupVersionKind(WorkflowGVK())
	err := r.DeleteAllOf(ctx, wf,
		client.InNamespace(job.Namespace),
		client.MatchingLabels{LabelTrainingJob: job.Name})
	if err != nil && !apierrors.IsNotFound(err) && !meta.IsNoMatchError(err) {
		return err
	}
	return nil
}

// updateQuotaStatus refreshes GPUQuota.status for kubectl visibility.
// Best-effort: failures are logged, never block the main reconcile.
func (r *TrainingJobReconciler) updateQuotaStatus(ctx context.Context, namespace string) {
	log := logf.FromContext(ctx)
	quota := &mlv1alpha1.GPUQuota{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: mlv1alpha1.GPUQuotaName}, quota); err != nil {
		return
	}
	jobs := &mlv1alpha1.TrainingJobList{}
	if err := r.List(ctx, jobs, client.InNamespace(namespace)); err != nil {
		return
	}
	used := UsedGPUs(jobs.Items, "")
	queued := CountQueued(jobs.Items)
	if quota.Status.UsedGPUs == used && quota.Status.QueuedJobs == queued {
		return
	}
	quota.Status.UsedGPUs = used
	quota.Status.QueuedJobs = queued
	if err := r.Status().Update(ctx, quota); err != nil {
		log.V(1).Info("gpuquota status update skipped", "error", err)
	}
}

func (r *TrainingJobReconciler) setCondition(job *mlv1alpha1.TrainingJob, condType string, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&job.Status.Conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: job.Generation,
	})
}

// SetupWithManager wires the controller: reconciles TrainingJobs and watches
// owned Argo Workflows (unstructured) via owner references — no polling.
func (r *TrainingJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	wf := &unstructured.Unstructured{}
	wf.SetGroupVersionKind(WorkflowGVK())
	return ctrl.NewControllerManagedBy(mgr).
		For(&mlv1alpha1.TrainingJob{}).
		Owns(wf).
		Named("trainingjob").
		Complete(r)
}
