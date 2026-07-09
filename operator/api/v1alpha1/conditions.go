package v1alpha1

// Condition types recorded on TrainingJob.status.conditions.
const (
	// ConditionQuotaGranted is True once the namespace GPUQuota admits the job.
	ConditionQuotaGranted = "QuotaGranted"
	// ConditionWorkflowSubmitted is True once a Workflow for the current
	// attempt has been created.
	ConditionWorkflowSubmitted = "WorkflowSubmitted"
	// ConditionComplete is True on success, False with a reason on failure.
	ConditionComplete = "Complete"
)

// Condition reasons.
const (
	ReasonQuotaAvailable       = "QuotaAvailable"
	ReasonQuotaExceeded        = "QuotaExceeded"
	ReasonWorkflowSubmitted    = "WorkflowSubmitted"
	ReasonTrainingSucceeded    = "TrainingSucceeded"
	ReasonBackoffLimitExceeded = "BackoffLimitExceeded"
)
