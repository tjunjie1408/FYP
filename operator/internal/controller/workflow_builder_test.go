package controller

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	mlv1alpha1 "github.com/tjunjie1408/FYP/operator/api/v1alpha1"
)

func sampleJob() *mlv1alpha1.TrainingJob {
	return &mlv1alpha1.TrainingJob{
		ObjectMeta: metav1.ObjectMeta{Name: "mnist-demo", Namespace: "training"},
		Spec: mlv1alpha1.TrainingJobSpec{
			Model:   mlv1alpha1.ModelSpec{Name: "mnist-cnn"},
			Dataset: mlv1alpha1.DatasetSpec{Name: "mnist"},
			Hyperparameters: map[string]string{
				"epochs":    "3",
				"batchSize": "64",
			},
			GPUs:   1,
			Output: mlv1alpha1.OutputSpec{Bucket: "models"},
		},
	}
}

func paramMap(t *testing.T, wf *unstructured.Unstructured) map[string]string {
	t.Helper()
	params, found, err := unstructured.NestedSlice(wf.Object, "spec", "arguments", "parameters")
	if err != nil || !found {
		t.Fatalf("parameters not found: %v", err)
	}
	out := map[string]string{}
	for _, p := range params {
		m := p.(map[string]interface{})
		out[m["name"].(string)] = m["value"].(string)
	}
	return out
}

func TestBuildWorkflowParameters(t *testing.T) {
	wf := BuildWorkflow(sampleJob(), 1, "training-pipeline")

	if got := wf.GetName(); got != "mnist-demo-attempt-1" {
		t.Errorf("name = %q, want mnist-demo-attempt-1", got)
	}
	if got := wf.GetNamespace(); got != "training" {
		t.Errorf("namespace = %q, want training", got)
	}
	if got := wf.GetLabels()[LabelTrainingJob]; got != "mnist-demo" {
		t.Errorf("trainingjob label = %q, want mnist-demo", got)
	}

	ref, _, _ := unstructured.NestedString(wf.Object, "spec", "workflowTemplateRef", "name")
	if ref != "training-pipeline" {
		t.Errorf("workflowTemplateRef = %q, want training-pipeline", ref)
	}

	params := paramMap(t, wf)
	want := map[string]string{
		"job-name":   "mnist-demo",
		"model":      "mnist-cnn",
		"dataset":    "mnist",
		"epochs":     "3",     // from hyperparameters
		"batch-size": "64",    // from hyperparameters (batchSize -> batch-size)
		"lr":         "0.001", // default: learningRate not set
		"s3-path":    "models/training/mnist-demo",
	}
	for k, v := range want {
		if params[k] != v {
			t.Errorf("param %s = %q, want %q", k, params[k], v)
		}
	}
	if len(params) != len(want) {
		t.Errorf("got %d params, want %d: %v", len(params), len(want), params)
	}
}

func TestBuildWorkflowDefaultsAndAttempts(t *testing.T) {
	job := sampleJob()
	job.Spec.Hyperparameters = nil
	wf := BuildWorkflow(job, 2, "")

	if got := wf.GetName(); got != "mnist-demo-attempt-2" {
		t.Errorf("name = %q, want mnist-demo-attempt-2", got)
	}
	ref, _, _ := unstructured.NestedString(wf.Object, "spec", "workflowTemplateRef", "name")
	if ref != DefaultTemplateName {
		t.Errorf("templateRef = %q, want %q", ref, DefaultTemplateName)
	}
	params := paramMap(t, wf)
	if params["epochs"] != "2" || params["batch-size"] != "128" || params["lr"] != "0.001" {
		t.Errorf("defaults not applied: %v", params)
	}
}

func TestModelURI(t *testing.T) {
	if got := ModelURI(sampleJob()); got != "s3://models/training/mnist-demo/" {
		t.Errorf("ModelURI = %q", got)
	}
}
