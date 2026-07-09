package controller

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	mlv1alpha1 "github.com/tjunjie1408/FYP/operator/api/v1alpha1"
)

// Integration tests run only when envtest binaries are available
// (KUBEBUILDER_ASSETS set — `make test` arranges this via setup-envtest).
// Pure unit tests in this package always run.
var (
	testEnv     *envtest.Environment
	k8sClient   client.Client
	integration bool
	cancelMgr   context.CancelFunc
)

func TestMain(m *testing.M) {
	if os.Getenv("KUBEBUILDER_ASSETS") != "" {
		if err := startEnvtest(); err != nil {
			fmt.Fprintf(os.Stderr, "envtest startup failed: %v\n", err)
			os.Exit(1)
		}
		integration = true
	}
	code := m.Run()
	if integration {
		cancelMgr()
		_ = testEnv.Stop()
	}
	os.Exit(code)
}

func startEnvtest() error {
	logf.SetLogger(zap.New(zap.WriteTo(os.Stderr), zap.UseDevMode(true)))

	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "config", "crd", "bases"),
			filepath.Join("..", "..", "test", "crds"), // minimal Argo Workflow CRD
		},
		ErrorIfCRDPathMissing: true,
	}
	cfg, err := testEnv.Start()
	if err != nil {
		return err
	}

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(mlv1alpha1.AddToScheme(scheme))

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return err
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:  scheme,
		Metrics: metricsserver.Options{BindAddress: "0"},
	})
	if err != nil {
		return err
	}
	if err := (&TrainingJobReconciler{
		Client:       mgr.GetClient(),
		Recorder:     mgr.GetEventRecorderFor("ml-operator-test"),
		TemplateName: DefaultTemplateName,
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	var ctx context.Context
	ctx, cancelMgr = context.WithCancel(context.Background())
	go func() {
		if err := mgr.Start(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "manager exited: %v\n", err)
		}
	}()
	return nil
}

// --- shared helpers for integration tests ---

func requireIntegration(t *testing.T) {
	t.Helper()
	if !integration {
		t.Skip("KUBEBUILDER_ASSETS not set; skipping envtest integration test (run via `make test`)")
	}
}

func createNamespace(t *testing.T, name string) {
	t.Helper()
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	if err := k8sClient.Create(context.Background(), ns); err != nil {
		t.Fatalf("create namespace %s: %v", name, err)
	}
}

func newJob(name, ns string, gpus, backoffLimit int32) *mlv1alpha1.TrainingJob {
	return &mlv1alpha1.TrainingJob{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: mlv1alpha1.TrainingJobSpec{
			Model:   mlv1alpha1.ModelSpec{Name: "mnist-cnn"},
			Dataset: mlv1alpha1.DatasetSpec{Name: "mnist"},
			GPUs:    gpus,
			Retry:   mlv1alpha1.RetrySpec{BackoffLimit: backoffLimit},
			Output:  mlv1alpha1.OutputSpec{Bucket: "models"},
		},
	}
}

func getWorkflow(ns, name string) (*unstructured.Unstructured, error) {
	wf := &unstructured.Unstructured{}
	wf.SetGroupVersionKind(WorkflowGVK())
	err := k8sClient.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: name}, wf)
	return wf, err
}

// setWorkflowPhase hand-patches the Workflow the way the Argo controller
// would. The vendored test CRD has no status subresource (matching Argo's
// real CRD), so a plain Update writes status.
func setWorkflowPhase(t *testing.T, ns, name, phase string) {
	t.Helper()
	wf, err := getWorkflow(ns, name)
	if err != nil {
		t.Fatalf("get workflow %s: %v", name, err)
	}
	if err := unstructured.SetNestedField(wf.Object, phase, "status", "phase"); err != nil {
		t.Fatalf("set phase: %v", err)
	}
	if err := k8sClient.Update(context.Background(), wf); err != nil {
		t.Fatalf("update workflow %s: %v", name, err)
	}
}

func getJob(t *testing.T, ns, name string) *mlv1alpha1.TrainingJob {
	t.Helper()
	job := &mlv1alpha1.TrainingJob{}
	if err := k8sClient.Get(context.Background(), types.NamespacedName{Namespace: ns, Name: name}, job); err != nil {
		t.Fatalf("get trainingjob %s: %v", name, err)
	}
	return job
}
