package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TrainingJobPhase describes the lifecycle phase of a TrainingJob.
// +kubebuilder:validation:Enum=Pending;Queued;Running;Succeeded;Failed
type TrainingJobPhase string

const (
	PhasePending   TrainingJobPhase = "Pending"
	PhaseQueued    TrainingJobPhase = "Queued"
	PhaseRunning   TrainingJobPhase = "Running"
	PhaseSucceeded TrainingJobPhase = "Succeeded"
	PhaseFailed    TrainingJobPhase = "Failed"
)

// ModelSpec selects which model architecture the trainer builds.
type ModelSpec struct {
	// +kubebuilder:validation:Enum=mnist-cnn;cifar10-cnn
	Name string `json:"name"`
}

// DatasetSpec selects the dataset the trainer downloads.
type DatasetSpec struct {
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// RetrySpec controls operator-level re-submission of failed workflows.
// This is independent of Argo's per-step retryStrategy: the operator reacts
// only to terminal Workflow phases and never consumes backoffLimit on
// internal step retries.
type RetrySpec struct {
	// BackoffLimit is the number of workflow re-submissions after the first
	// attempt fails. Total attempts = backoffLimit + 1.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=0
	// +optional
	BackoffLimit int32 `json:"backoffLimit,omitempty"`
}

// OutputSpec describes where the trained model artifact is stored.
type OutputSpec struct {
	// Bucket is the S3 (MinIO) bucket. The object key is conventional:
	// <namespace>/<name>/model.pt
	// +kubebuilder:validation:MinLength=1
	Bucket string `json:"bucket"`
}

// TrainingJobSpec defines the desired state of a TrainingJob.
type TrainingJobSpec struct {
	Model   ModelSpec   `json:"model"`
	Dataset DatasetSpec `json:"dataset"`

	// Hyperparameters are passed through as workflow parameters.
	// Recognized keys: epochs, batchSize, learningRate.
	// +optional
	Hyperparameters map[string]string `json:"hyperparameters,omitempty"`

	// GPUs requested; counted against the namespace GPUQuota (simulated GPUs).
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=1
	// +optional
	GPUs int32 `json:"gpus,omitempty"`

	// +optional
	Retry RetrySpec `json:"retry,omitempty"`

	// TTLSecondsAfterFinished: the operator deletes this TrainingJob (and its
	// owned Workflows) this many seconds after it reaches a terminal phase.
	// +kubebuilder:validation:Minimum=0
	// +optional
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"`

	Output OutputSpec `json:"output"`
}

// TrainingJobStatus defines the observed state of a TrainingJob.
type TrainingJobStatus struct {
	// +optional
	Phase TrainingJobPhase `json:"phase,omitempty"`

	// Attempts counts Workflows created for this job. It is incremented in
	// exactly one place: when the operator creates a new Workflow object.
	// +optional
	Attempts int32 `json:"attempts,omitempty"`

	// WorkflowName of the current attempt.
	// +optional
	WorkflowName string `json:"workflowName,omitempty"`

	// ModelURI is set on success, e.g. s3://models/training/mnist-demo/
	// +optional
	ModelURI string `json:"modelURI,omitempty"`

	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="GPUs",type=integer,JSONPath=`.spec.gpus`
// +kubebuilder:printcolumn:name="Attempts",type=integer,JSONPath=`.status.attempts`
// +kubebuilder:printcolumn:name="Model",type=string,JSONPath=`.status.modelURI`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// TrainingJob is the Schema for the trainingjobs API.
type TrainingJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is immutable after creation (validated by CEL instead of a webhook).
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="spec is immutable"
	Spec   TrainingJobSpec   `json:"spec,omitempty"`
	Status TrainingJobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TrainingJobList contains a list of TrainingJob.
type TrainingJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TrainingJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TrainingJob{}, &TrainingJobList{})
}

// IsTerminal reports whether the job reached a terminal phase.
func (t *TrainingJob) IsTerminal() bool {
	return t.Status.Phase == PhaseSucceeded || t.Status.Phase == PhaseFailed
}
