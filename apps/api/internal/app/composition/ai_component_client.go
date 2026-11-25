package composition

import (
	"context"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// configuredComponentClient is the single operational request-policy boundary.
// Component code owns prompts/schemas; the runtime owns model, deadline,
// generation parameters and fallback chains. This adapter is immutable.
type configuredComponentClient struct {
	next interface {
		ports.Client
		ports.EmbeddingClient
	}
	components map[ports.ModelTier]runtimeComponent
}

func (c configuredComponentClient) Structured(ctx context.Context, req ports.StructuredRequest) (ports.StructuredResponse, error) {
	settings, ok := c.components[req.Tier]
	if !ok || req.Tier == ports.ComponentEmbedding {
		return ports.StructuredResponse{}, ErrRuntime
	}
	req.Timeout = time.Duration(settings.TimeoutMS) * time.Millisecond
	req.MaxTokens = settings.MaxTokens
	if budget, ok := contract.BudgetFromContext(ctx); ok && budget.MaxTokensOut > 0 {
		req.MaxTokens = min(req.MaxTokens, budget.MaxTokensOut)
	}
	req.Temperature = settings.Temperature
	return c.next.Structured(ctx, req)
}
func (c configuredComponentClient) Stream(ctx context.Context, req ports.StreamRequest) (ports.Stream, error) {
	settings, ok := c.components[req.Tier]
	if !ok || req.Tier != ports.ComponentGenerator {
		return nil, ErrRuntime
	}
	req.Timeout = time.Duration(settings.TimeoutMS) * time.Millisecond
	req.MaxTokens = settings.MaxTokens
	if budget, ok := contract.BudgetFromContext(ctx); ok && budget.MaxTokensOut > 0 {
		req.MaxTokens = min(req.MaxTokens, budget.MaxTokensOut)
	}
	req.Temperature = settings.Temperature
	return c.next.Stream(ctx, req)
}
func (c configuredComponentClient) Embed(ctx context.Context, req ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
	settings, ok := c.components[req.Tier]
	if !ok || req.Tier != ports.ComponentEmbedding {
		return ports.EmbeddingResponse{}, ErrRuntime
	}
	req.Timeout = time.Duration(settings.TimeoutMS) * time.Millisecond
	return c.next.Embed(ctx, req)
}

// Resolve through the same runtime policy that governs the actual provider call.
func (c configuredComponentClient) EmbeddingProfile(tier ports.ModelTier) (ports.EmbeddingProfile, error) {
	settings, ok := c.components[tier]
	if !ok || tier != ports.ComponentEmbedding || len(settings.Chain) == 0 {
		return ports.EmbeddingProfile{}, ErrRuntime
	}
	profile := ports.EmbeddingProfile{ExecutionTimeout: time.Duration(settings.TimeoutMS) * time.Millisecond}
	for _, model := range settings.Chain {
		profile.Models = append(profile.Models, ports.EmbeddingModel{Provider: model.Provider, Model: model.Name, Dimensions: model.Dimensions})
	}
	return profile, nil
}
