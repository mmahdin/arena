package integration

import (
	"context"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/fake"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"testing"
	"time"
)

type syntheticBundle map[string][]byte

func (b syntheticBundle) Reference() c.BundleRef {
	return c.BundleRef{Version: "kb@0.9.0", PromptSHAs: map[string]string{"brain": "synthetic"}}
}
func (b syntheticBundle) SyntheticOnly() bool { return true }
func (b syntheticBundle) ArtifactNames() []string {
	a := []string{}
	for p := range b {
		a = append(a, p)
	}
	return a
}
func (b syntheticBundle) Artifact(_ context.Context, p string) ([]byte, error) {
	v, ok := b[p]
	if !ok {
		return nil, errors.New("missing synthetic asset")
	}
	return v, nil
}

type syntheticRegistry struct{ b syntheticBundle }

func (r syntheticRegistry) Active(context.Context, c.ID) (ports.Bundle, error) { return r.b, nil }

type syntheticUsePolicy struct{ allowed bool }

func (u syntheticUsePolicy) CanSearch(context.Context, c.ID, c.ID) (bool, error) {
	return u.allowed, nil
}
func (u syntheticUsePolicy) CanUse(context.Context, c.ID, retrieval.SourceKind, c.ID) (bool, error) {
	return u.allowed, nil
}

func backgroundClient(t *testing.T, provider *fake.Provider) ports.Client {
	t.Helper()
	model := func(name string) llm.Model {
		return llm.Model{Provider: "synthetic", Name: name, NativeJSONSchema: true, InputMicroUSDPerM: 100, OutputMicroUSDPerM: 100}
	}
	cfg := llm.Config{Chains: map[ports.ModelTier][]llm.Model{
		ports.ComponentBrain: {model("synthetic-memory-writer")}, ports.ComponentMemoryWriter: {model("synthetic-memory-writer")}, ports.ComponentGenerator: {model("synthetic-reflection")}, ports.ComponentReflection: {model("synthetic-reflection")}, ports.ComponentSafety: {model("synthetic-classifier")}, ports.ComponentValidator: {model("synthetic-classifier")},
		ports.TierEmbedding: {{Provider: "synthetic", Name: "synthetic-embedding", Dimensions: 3}},
	}, DefaultTimeout: 2 * time.Second, BreakerWindow: time.Minute, BreakerCooldown: time.Minute, BreakerErrorRate: .5, BreakerMinSamples: 2}
	client, err := llm.New(cfg, map[string]llm.Provider{"synthetic": provider}, llm.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return client
}
