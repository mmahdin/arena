package ports

import (
	"context"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"time"
)

type Client interface {
	Structured(ctx context.Context, req StructuredRequest) (StructuredResponse, error)
	Stream(ctx context.Context, req StreamRequest) (Stream, error)
}

type ModelTier string // "small" | "strong" | "classifier" | "embedding"

type StructuredRequest struct {
	Tier           ModelTier
	System         []Block // each Block{Text string; Cache bool; Name string}
	Messages       []ChatMessage
	Schema         []byte // JSON Schema (post-injection)
	SchemaName     string // e.g. "brain.v1"
	MaxTokens      int
	Temperature    float32
	Timeout        time.Duration
	Meta           contract.CallMeta
	IdempotencyKey string
}
type StructuredResponse struct {
	JSON     []byte
	Usage    contract.LLMUsage
	Model    string
	Attempts int
	Provider string
}

type StreamRequest struct {
	Tier                ModelTier
	System              []Block
	Messages            []ChatMessage
	MaxTokens           int
	Temperature         float32
	Stop                []string
	Timeout, FirstToken time.Duration
	Meta                contract.CallMeta
}
type Stream interface {
	Recv() (Chunk, error)
	Close() error
}

// EmbeddingClient is an additive companion port, rather than a change to the
// v1.2 Client interface (which deliberately exposes Structured and Stream only).
// WP-09 may consume it without casting embedding requests to text completions.
type EmbeddingClient interface {
	Embed(ctx context.Context, req EmbeddingRequest) (EmbeddingResponse, error)
}

type EmbeddingRequest struct {
	Tier    ModelTier
	Input   string
	Timeout time.Duration
	Meta    contract.CallMeta
}

type EmbeddingResponse struct {
	Vector []float32
	Model  string
	Usage  contract.LLMUsage
}
