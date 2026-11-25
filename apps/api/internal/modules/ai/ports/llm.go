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

type ModelTier string // Execution component identity; never a specific provider/model

type StructuredRequest struct {
	// SystemOne is JEV's native decision input. Schema still validates the
	// downstream selection after the transport maps typed answers to IDs.
	SystemOne      *SystemOneRequest `json:",omitempty"`
	Tier           ModelTier
	System         []Block // each Block{Text string; Cache bool; Name string}
	Messages       []ChatMessage
	Schema         []byte // JSON Schema (post-injection)
	SchemaName     string // e.g. "brain.v1"
	MaxTokens      int
	MaxAttempts    int // 0 = client policy; 1 = single attempt, no schema repair/fallback (Router)
	Temperature    float32
	Timeout        time.Duration
	Meta           contract.CallMeta
	IdempotencyKey string
}

// SystemOneRequest deliberately exposes only the Noul questions used by JEV.
// A resource:<catalog ID> question evaluates that card; topic_continues evaluates
// continuity. Neither is a text-generation request or an instruction to act.
type SystemOneRequest struct {
	State     string                       `json:"state"`
	Questions map[string]SystemOneQuestion `json:"questions"`
}

type SystemOneQuestion struct {
	Type         string `json:"type"`
	Instructions string `json:"instructions"`
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
	Consumer string // Code-owned Inspector label; never part of vector identity.
	Tier     ModelTier
	Input    string
	Timeout  time.Duration
	Meta     contract.CallMeta
}

type EmbeddingResponse struct {
	Vector []float32
	Model  string
	Usage  contract.LLMUsage
}
