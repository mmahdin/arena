package llm

import (
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// EmbeddingProfile exposes only immutable execution identity, never credentials.
// The provider chain itself, its retries, ledger and breakers stay in Client.
func (c *Client) EmbeddingProfile(tier ports.ModelTier) (ports.EmbeddingProfile, error) {
	if tier != ports.ComponentEmbedding && tier != ports.TierEmbedding {
		return ports.EmbeddingProfile{}, ErrConfig
	}
	chain := c.config.Chains[tier]
	if len(chain) == 0 {
		return ports.EmbeddingProfile{}, ErrConfig
	}
	profile := ports.EmbeddingProfile{ExecutionTimeout: c.config.DefaultTimeout}
	for _, model := range chain {
		profile.Models = append(profile.Models, ports.EmbeddingModel{Provider: model.Provider, Model: model.Name, Dimensions: model.Dimensions})
	}
	return profile, nil
}
