package llm

import (
	"context"
	"errors"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// Mode tells a provider to use native JSON Schema support, or JSON-object mode
// for compatible providers lacking that feature. No provider SDK leaks into ports.
type Mode string

const (
	ModeNativeSchema Mode = "native_schema"
	ModeJSONObject   Mode = "json_object"
	ModeSystemOne    Mode = "systemone"
)

type StructuredResult struct {
	JSON  []byte
	Usage contract.LLMUsage
}

type EmbeddingResult struct {
	Vector []float32
	Usage  contract.LLMUsage
}

// Provider is the only model-specific boundary. Implementations own HTTP/SDK
// details but never logging, persistence, prompt selection or user state.
type Provider interface {
	Structured(context.Context, Model, ports.StructuredRequest, Mode) (StructuredResult, error)
	Stream(context.Context, Model, ports.StreamRequest) (ports.Stream, error)
	Embed(context.Context, Model, ports.EmbeddingRequest) (EmbeddingResult, error)
}

type FailureKind string

const (
	FailureUnknown   FailureKind = "unclassified_error"
	FailureTimeout   FailureKind = "request_timeout"
	FailureCanceled  FailureKind = "request_canceled"
	FailureTransport FailureKind = "transport"
	FailureServer    FailureKind = "server"
	FailureRateLimit FailureKind = "rate_limit"
	FailurePermanent FailureKind = "permanent"
	FailureAuth      FailureKind = "authentication"
	FailureInvalid   FailureKind = "invalid_response"
)

// Failure exposes structural metadata only. Cause may retain transport errors
// in memory for errors.Is/As; never serialize it or a vendor payload. Ordinary
// logging uses observability.ErrorFields. RetryAfter applies only to rate limits.
type Failure struct {
	Reason       string
	ProviderCode string
	Timeout      string
	Cause        error
	Kind         FailureKind
	RetryAfter   time.Duration
	HTTPStatus   int    // safe numeric code; never vendor payloads
	Origin       string // internally captured code location only
}

func (e *Failure) DiagnosticReason() string {
	if e == nil {
		return "unknown"
	}
	if e.Reason != "" {
		return e.Reason
	}
	return string(e.Kind)
}
func (e *Failure) DiagnosticRetryable() bool { return e.Retryable() }
func (e *Failure) DiagnosticOrigin() string {
	if e == nil {
		return ""
	}
	return e.Origin
}
func (e *Failure) DiagnosticStatus() int {
	if e == nil {
		return 0
	}
	return e.HTTPStatus
}
func (e *Failure) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}
func (e *Failure) DiagnosticProviderCode() string {
	if e == nil {
		return ""
	}
	return e.ProviderCode
}
func (e *Failure) DiagnosticTimeout() string {
	if e == nil {
		return ""
	}
	return e.Timeout
}
func (e *Failure) Error() string { return "llm provider request failed" }
func (e *Failure) Retryable() bool {
	return e != nil && (e.Kind == FailureTransport || e.Kind == FailureServer || e.Kind == FailureRateLimit)
}

var (
	ErrUnavailable = errors.New("llm provider unavailable")
	ErrInvalid     = errors.New("llm structured response invalid")
	ErrUsage       = errors.New("llm usage invalid")
	ErrQuota       = errors.New("llm daily quota exceeded")
	ErrCancelled   = errors.New("llm call cancelled")
)

// A UsageLedger must charge every provider response/stream usage event, even
// when a later repair, fallback, cancellation or validation fails. Implementers
// must not store prompts, answers or private user text in usage records.
type UsageLedger interface {
	Check(context.Context, contract.ID, DailyQuota) error
	Record(context.Context, contract.ID, contract.LLMUsage) error
}

type Event struct {
	Tier                              ports.ModelTier
	Provider, Model                   string
	Attempt                           int
	Elapsed                           time.Duration
	Failure                           FailureKind
	FirstToken                        bool
	Charged                           bool
	InTokens, OutTokens, CachedTokens int
	CostMicroUSD                      int64
	Fallback                          bool
}

// Metrics has metadata only; it must not receive messages, model payloads,
// schema bodies or user IDs. The trace writer is the only content-audit owner.
type Metrics interface{ Observe(Event) }
