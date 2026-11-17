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
	FailureTransport FailureKind = "transport"
	FailureServer    FailureKind = "server"
	FailureRateLimit FailureKind = "rate_limit"
	FailurePermanent FailureKind = "permanent"
	FailureInvalid   FailureKind = "invalid_response"
)

// Failure is intentionally sanitized. Do not wrap a vendor error, body, URL,
// auth header or any prompt in it. RetryAfter applies only to rate limits.
type Failure struct {
	Kind       FailureKind
	RetryAfter time.Duration
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
