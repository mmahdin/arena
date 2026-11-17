// Package embedding adapts the WP-04 provider-neutral EmbeddingClient to the
// narrow retrieval.Embedder port. It adds no provider SDK, no second model client
// and no caching: WP-04 owns transport, timeouts, breakers, quota and usage
// metering. Retrieval must never construct its own HTTP client.
package embedding

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// ModelID is the storage identity recorded next to every vector. Vectors from a
// different model/dimension are never compared; see the SQL adapter's codec.
// The concrete model/dimensions remain the [OPEN-4] product decision.
type ModelID struct {
	Model      string
	Dimensions int
}

// Adapter implements retrieval.Embedder on top of ports.EmbeddingClient.
type Adapter struct {
	client     ports.EmbeddingClient
	tier       ports.ModelTier
	meta       contract.CallMeta
	model      ModelID
	timeout    time.Duration
	onUsage    func(contract.LLMUsage)
	dimensions int
}

// Options are explicit dependencies; there is no global registry.
type Options struct {
	Client     ports.EmbeddingClient
	Model      ModelID
	Timeout    time.Duration
	OnUsage    func(contract.LLMUsage) // optional: metering/trace hook
	Dimensions int                     // optional override, must match Model
}

var errConfig = errors.New("embedding adapter requires a client, model id and dimensions")

// New validates the configuration. A zero timeout falls back to the WP-04
// client default rather than an unbounded call.
func New(opts Options) (*Adapter, error) {
	if opts.Client == nil || opts.Model.Model == "" {
		return nil, errConfig
	}
	dims := opts.Model.Dimensions
	if opts.Dimensions > 0 {
		dims = opts.Dimensions
	}
	if dims < 1 || dims > 8192 {
		return nil, fmt.Errorf("embedding dimensions %d outside the stored codec range", dims)
	}
	if len(opts.Model.Model) > 120 {
		return nil, errors.New("embedding model id too long for storage")
	}
	return &Adapter{
		client: opts.Client, tier: ports.TierEmbedding, model: ModelID{Model: opts.Model.Model, Dimensions: dims},
		timeout: opts.Timeout, onUsage: opts.OnUsage, dimensions: dims,
	}, nil
}

// WithCallMeta attaches the turn/user identity required by the WP-04 quota and
// trace layer. Without a valid owner the adapter refuses to call the provider.
func (a *Adapter) WithCallMeta(meta contract.CallMeta) *Adapter {
	clone := *a
	clone.meta = meta
	return &clone
}

// Embed returns the vector and the storage model id. A provider error is
// returned unchanged so retrieval can degrade; the adapter never returns a
// fabricated or truncated vector.
func (a *Adapter) Embed(ctx context.Context, text string) ([]float32, string, error) {
	if text == "" {
		return nil, "", errors.New("empty embedding input")
	}
	resp, err := a.client.Embed(ctx, ports.EmbeddingRequest{
		Tier: a.tier, Input: text, Timeout: a.timeout, Meta: a.meta,
	})
	if err != nil {
		return nil, "", err
	}
	if len(resp.Vector) != a.dimensions {
		return nil, "", fmt.Errorf("embedding dimension %d does not match stored model %d", len(resp.Vector), a.dimensions)
	}
	if resp.Model != "" && resp.Model != a.model.Model {
		return nil, "", fmt.Errorf("embedding provider returned a different model")
	}
	if a.onUsage != nil {
		a.onUsage(resp.Usage)
	}
	return resp.Vector, a.model.Model, nil
}
