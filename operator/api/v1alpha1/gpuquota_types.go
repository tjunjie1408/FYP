package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GPUQuotaSpec defines the desired state of a GPUQuota.
type GPUQuotaSpec struct {
	// HardGPUs is the maximum number of (simulated) GPUs that TrainingJobs in
	// this namespace may hold concurrently.
	// +kubebuilder:validation:Minimum=0
	HardGPUs int32 `json:"hardGPUs"`
}

// GPUQuotaStatus is written by the TrainingJob reconciler for kubectl visibility.
type GPUQuotaStatus struct {
	// +optional
	UsedGPUs int32 `json:"usedGPUs,omitempty"`
	// +optional
	QueuedJobs int32 `json:"queuedJobs,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:path=gpuquotas
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Hard",type=integer,JSONPath=`.spec.hardGPUs`
// +kubebuilder:printcolumn:name="Used",type=integer,JSONPath=`.status.usedGPUs`
// +kubebuilder:printcolumn:name="Queued",type=integer,JSONPath=`.status.queuedJobs`

// GPUQuota caps concurrent GPU usage by TrainingJobs in a namespace.
// Singleton by convention: the reconciler reads the GPUQuota named "default"
// in the TrainingJob's namespace; if absent, usage is unlimited.
type GPUQuota struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GPUQuotaSpec   `json:"spec,omitempty"`
	Status GPUQuotaStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GPUQuotaList contains a list of GPUQuota.
type GPUQuotaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GPUQuota `json:"items"`
}

// GPUQuotaName is the conventional singleton name read by the reconciler.
const GPUQuotaName = "default"

func init() {
	SchemeBuilder.Register(&GPUQuota{}, &GPUQuotaList{})
}
