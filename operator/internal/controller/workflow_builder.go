package controller

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	mlv1alpha1 "github.com/tjunjie1408/FYP/operator/api/v1alpha1"
)

const (
	// LabelTrainingJob marks Workflows owned by a TrainingJob; used for
	// finalizer cleanup via DeleteAllOf.
	LabelTrainingJob = "mlplatform.fyp.io/trainingjob"
	// LabelManagedBy identifies operator-created objects.
	LabelManagedBy = "app.kubernetes.io/managed-by"
	managerName    = "ml-operator"

	// DefaultTemplateName is the WorkflowTemplate the operator references.
	// It must exist in the TrainingJob's namespace (installed by GitOps).
	DefaultTemplateName = "training-pipeline"
)

// Hyperparameter defaults; keys are the TrainingJob spec.hyperparameters keys,
// mapped to the WorkflowTemplate's parameter names.
var hyperparameterParams = []struct {
	specKey      string
	paramName    string
	defaultValue string
}{
	{"epochs", "epochs", "2"},
	{"batchSize", "batch-size", "128"},
	{"learningRate", "lr", "0.001"},
}

// WorkflowGVK returns the Argo Workflow GroupVersionKind. The operator talks
// to Argo via unstructured objects only — importing the Argo Go module pins
// conflicting k8s.io/* versions (see ADR-0003).
func WorkflowGVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{Group: "argoproj.io", Version: "v1alpha1", Kind: "Workflow"}
}

// WorkflowListGVK returns the GVK for listing Workflows.
func WorkflowListGVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{Group: "argoproj.io", Version: "v1alpha1", Kind: "WorkflowList"}
}

// WorkflowName returns the deterministic per-attempt Workflow name.
func WorkflowName(job *mlv1alpha1.TrainingJob, attempt int32) string {
	return fmt.Sprintf("%s-attempt-%d", job.Name, attempt)
}

// S3Path returns the bucket-prefixed object prefix the pipeline uploads to,
// e.g. "models/training/mnist-demo". Contract shared with upload.py and the
// InferenceService STORAGE_URI.
func S3Path(job *mlv1alpha1.TrainingJob) string {
	return fmt.Sprintf("%s/%s/%s", job.Spec.Output.Bucket, job.Namespace, job.Name)
}

// ModelURI returns the s3:// URI recorded on success.
func ModelURI(job *mlv1alpha1.TrainingJob) string {
	return fmt.Sprintf("s3://%s/", S3Path(job))
}

// BuildWorkflow renders the Argo Workflow for one attempt. Pure function —
// unit-tested without a cluster. The pipeline logic itself lives in the
// referenced WorkflowTemplate; this object is only a parameterized pointer.
func BuildWorkflow(job *mlv1alpha1.TrainingJob, attempt int32, templateName string) *unstructured.Unstructured {
	if templateName == "" {
		templateName = DefaultTemplateName
	}

	params := []interface{}{
		map[string]interface{}{"name": "job-name", "value": job.Name},
		map[string]interface{}{"name": "model", "value": job.Spec.Model.Name},
		map[string]interface{}{"name": "dataset", "value": job.Spec.Dataset.Name},
	}
	for _, hp := range hyperparameterParams {
		v := hp.defaultValue
		if s, ok := job.Spec.Hyperparameters[hp.specKey]; ok && s != "" {
			v = s
		}
		params = append(params, map[string]interface{}{"name": hp.paramName, "value": v})
	}
	params = append(params, map[string]interface{}{"name": "s3-path", "value": S3Path(job)})

	wf := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "argoproj.io/v1alpha1",
			"kind":       "Workflow",
			"metadata": map[string]interface{}{
				"name":      WorkflowName(job, attempt),
				"namespace": job.Namespace,
				"labels": map[string]interface{}{
					LabelTrainingJob: job.Name,
					LabelManagedBy:   managerName,
				},
			},
			"spec": map[string]interface{}{
				"workflowTemplateRef": map[string]interface{}{
					"name": templateName,
				},
				"arguments": map[string]interface{}{
					"parameters": params,
				},
			},
		},
	}
	return wf
}
