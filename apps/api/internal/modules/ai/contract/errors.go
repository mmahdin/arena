package contract

type Code string

const (
	ErrQueued            Code = "queued"    // accepted, not processed yet
	ErrDuplicate         Code = "duplicate" // idempotent replay
	ErrBudget            Code = "budget_exceeded"
	ErrSafetyDegraded    Code = "safety_degraded"
	ErrRetrievalDegraded Code = "retrieval_degraded"
	ErrBrainInvalid      Code = "brain_invalid"
	ErrBrainTimeout      Code = "brain_timeout"
	ErrGenFailed         Code = "generation_failed"
	ErrValidation        Code = "validation_failed"
	ErrStateConflict     Code = "state_conflict"
	ErrStateCommit       Code = "state_commit_failed"
	ErrBundle            Code = "bundle_unavailable"
	ErrProvider          Code = "provider_unavailable"
	ErrInternal          Code = "internal"
	ErrRateLimited       Code = "rate_limited"
	ErrForbidden         Code = "forbidden"
)

type Error struct {
	Code      Code   `json:"code"`
	Step      string `json:"-"`
	Cause     error  `json:"-"`
	Retryable bool   `json:"retryable"`
	Degrades  bool   `json:"degrades"`
}

// PublicError is the only error shape that leaves the AI Core. It never contains internal detail.
type PublicError struct {
	Code      Code   `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}
