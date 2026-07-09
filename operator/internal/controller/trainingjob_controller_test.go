package controller

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	mlv1alpha1 "github.com/tjunjie1408/FYP/operator/api/v1alpha1"
)

const (
	timeout  = 10 * time.Second
	interval = 250 * time.Millisecond
	// quotaTimeout covers the 15s Queued requeue backstop.
	quotaTimeout = 40 * time.Second
)

func TestLifecycleSucceeds(t *testing.T) {
	requireIntegration(t)
	g := NewWithT(t)
	ns := "t-success"
	createNamespace(t, ns)
	ctx := context.Background()

	job := newJob("mnist", ns, 1, 0)
	job.Spec.Hyperparameters = map[string]string{"epochs": "5"}
	g.Expect(k8sClient.Create(ctx, job)).To(Succeed())

	// Workflow for attempt 1 appears, owned and parameterized.
	g.Eventually(func() error {
		_, err := getWorkflow(ns, "mnist-attempt-1")
		return err
	}, timeout, interval).Should(Succeed())

	wf, _ := getWorkflow(ns, "mnist-attempt-1")
	g.Expect(wf.GetOwnerReferences()).To(HaveLen(1))
	g.Expect(wf.GetOwnerReferences()[0].Kind).To(Equal("TrainingJob"))
	g.Expect(*wf.GetOwnerReferences()[0].Controller).To(BeTrue())

	g.Eventually(func() mlv1alpha1.TrainingJobPhase {
		return getJob(t, ns, "mnist").Status.Phase
	}, timeout, interval).Should(Equal(mlv1alpha1.PhaseRunning))

	setWorkflowPhase(t, ns, "mnist-attempt-1", "Succeeded")

	g.Eventually(func() mlv1alpha1.TrainingJobPhase {
		return getJob(t, ns, "mnist").Status.Phase
	}, timeout, interval).Should(Equal(mlv1alpha1.PhaseSucceeded))

	final := getJob(t, ns, "mnist")
	g.Expect(final.Status.ModelURI).To(Equal("s3://models/t-success/mnist/"))
	g.Expect(final.Status.Attempts).To(Equal(int32(1)))
	g.Expect(final.Status.CompletionTime).NotTo(BeNil())
	cond := meta.FindStatusCondition(final.Status.Conditions, mlv1alpha1.ConditionComplete)
	g.Expect(cond).NotTo(BeNil())
	g.Expect(cond.Status).To(Equal(metav1.ConditionTrue))
}

func TestRetryThenBackoffExceeded(t *testing.T) {
	requireIntegration(t)
	g := NewWithT(t)
	ns := "t-retry"
	createNamespace(t, ns)
	ctx := context.Background()

	g.Expect(k8sClient.Create(ctx, newJob("flaky", ns, 1, 1))).To(Succeed())

	g.Eventually(func() error {
		_, err := getWorkflow(ns, "flaky-attempt-1")
		return err
	}, timeout, interval).Should(Succeed())

	setWorkflowPhase(t, ns, "flaky-attempt-1", "Failed")

	// Operator deletes attempt 1 and creates attempt 2; attempts == 2.
	g.Eventually(func() error {
		_, err := getWorkflow(ns, "flaky-attempt-2")
		return err
	}, timeout, interval).Should(Succeed())
	g.Eventually(func() int32 {
		return getJob(t, ns, "flaky").Status.Attempts
	}, timeout, interval).Should(Equal(int32(2)))

	setWorkflowPhase(t, ns, "flaky-attempt-2", "Failed")

	g.Eventually(func() mlv1alpha1.TrainingJobPhase {
		return getJob(t, ns, "flaky").Status.Phase
	}, timeout, interval).Should(Equal(mlv1alpha1.PhaseFailed))

	final := getJob(t, ns, "flaky")
	g.Expect(final.Status.Attempts).To(Equal(int32(2))) // 1 + backoffLimit
	cond := meta.FindStatusCondition(final.Status.Conditions, mlv1alpha1.ConditionComplete)
	g.Expect(cond).NotTo(BeNil())
	g.Expect(cond.Status).To(Equal(metav1.ConditionFalse))
	g.Expect(cond.Reason).To(Equal(mlv1alpha1.ReasonBackoffLimitExceeded))
}

func TestQuotaQueueing(t *testing.T) {
	requireIntegration(t)
	g := NewWithT(t)
	ns := "t-quota"
	createNamespace(t, ns)
	ctx := context.Background()

	quota := &mlv1alpha1.GPUQuota{
		ObjectMeta: metav1.ObjectMeta{Name: mlv1alpha1.GPUQuotaName, Namespace: ns},
		Spec:       mlv1alpha1.GPUQuotaSpec{HardGPUs: 1},
	}
	g.Expect(k8sClient.Create(ctx, quota)).To(Succeed())

	g.Expect(k8sClient.Create(ctx, newJob("first", ns, 1, 0))).To(Succeed())
	g.Eventually(func() mlv1alpha1.TrainingJobPhase {
		return getJob(t, ns, "first").Status.Phase
	}, timeout, interval).Should(Equal(mlv1alpha1.PhaseRunning))

	// Second job must queue: 1/1 GPUs in use.
	g.Expect(k8sClient.Create(ctx, newJob("second", ns, 1, 0))).To(Succeed())
	g.Eventually(func() mlv1alpha1.TrainingJobPhase {
		return getJob(t, ns, "second").Status.Phase
	}, timeout, interval).Should(Equal(mlv1alpha1.PhaseQueued))

	// Quota status reflects usage.
	g.Eventually(func() int32 {
		q := &mlv1alpha1.GPUQuota{}
		_ = k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: mlv1alpha1.GPUQuotaName}, q)
		return q.Status.QueuedJobs
	}, timeout, interval).Should(Equal(int32(1)))

	// First completes -> second starts (within the 15s requeue backstop).
	setWorkflowPhase(t, ns, "first-attempt-1", "Succeeded")
	g.Eventually(func() mlv1alpha1.TrainingJobPhase {
		return getJob(t, ns, "second").Status.Phase
	}, quotaTimeout, interval).Should(Equal(mlv1alpha1.PhaseRunning))
}

func TestStepRetryChurnDoesNotConsumeAttempts(t *testing.T) {
	requireIntegration(t)
	g := NewWithT(t)
	ns := "t-churn"
	createNamespace(t, ns)
	ctx := context.Background()

	g.Expect(k8sClient.Create(ctx, newJob("churn", ns, 1, 2))).To(Succeed())
	g.Eventually(func() error {
		_, err := getWorkflow(ns, "churn-attempt-1")
		return err
	}, timeout, interval).Should(Succeed())

	// Simulate Argo's internal step retries: the Workflow stays Running while
	// its status churns. The operator must not consume backoffLimit.
	for i := 0; i < 3; i++ {
		setWorkflowPhase(t, ns, "churn-attempt-1", "Running")
		time.Sleep(300 * time.Millisecond)
	}
	g.Consistently(func() int32 {
		return getJob(t, ns, "churn").Status.Attempts
	}, 2*time.Second, interval).Should(Equal(int32(1)))

	// Only the terminal phase moves the state machine.
	setWorkflowPhase(t, ns, "churn-attempt-1", "Succeeded")
	g.Eventually(func() mlv1alpha1.TrainingJobPhase {
		return getJob(t, ns, "churn").Status.Phase
	}, timeout, interval).Should(Equal(mlv1alpha1.PhaseSucceeded))
	g.Expect(getJob(t, ns, "churn").Status.Attempts).To(Equal(int32(1)))
}

func TestTTLDeletesJobAndFinalizerCleansWorkflows(t *testing.T) {
	requireIntegration(t)
	g := NewWithT(t)
	ns := "t-ttl"
	createNamespace(t, ns)
	ctx := context.Background()

	job := newJob("ephemeral", ns, 1, 0)
	ttl := int32(1)
	job.Spec.TTLSecondsAfterFinished = &ttl
	g.Expect(k8sClient.Create(ctx, job)).To(Succeed())

	g.Eventually(func() error {
		_, err := getWorkflow(ns, "ephemeral-attempt-1")
		return err
	}, timeout, interval).Should(Succeed())
	setWorkflowPhase(t, ns, "ephemeral-attempt-1", "Succeeded")

	// TTL elapses -> CR deleted; finalizer explicitly deletes the Workflow
	// (envtest has no garbage collector, so this proves operator cleanup).
	g.Eventually(func() bool {
		j := &mlv1alpha1.TrainingJob{}
		err := k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: "ephemeral"}, j)
		return apierrors.IsNotFound(err)
	}, timeout, interval).Should(BeTrue())
	g.Eventually(func() bool {
		_, err := getWorkflow(ns, "ephemeral-attempt-1")
		return apierrors.IsNotFound(err)
	}, timeout, interval).Should(BeTrue())
}
