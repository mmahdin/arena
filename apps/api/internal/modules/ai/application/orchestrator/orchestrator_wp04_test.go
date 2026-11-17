package orchestrator_test

import (
	"context"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/fake"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/orchestrator"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// Unlike the scripted-client unit tests, this executes the actual WP-04
// provider-neutral Client. Only its provider and stage adapters are synthetic;
// it is not wired to an unreviewed live psychological-response endpoint.
func TestTwoStageWithRealWP04ClientAndFakeProvider(t *testing.T) {
	provider := fake.New()
	provider.QueueStructured("small", fake.StructuredReply{JSON: []byte(`{"objective":"reflect"}`), Usage: c.LLMUsage{InTokens: 5, OutTokens: 2, FinishReason: "stop"}})
	provider.QueueStream("strong", fake.StreamReply{FailAt: -1, Chunks: []ports.Chunk{
		{Text: "Synthetic "}, {Text: "answer"}, {Usage: &c.LLMUsage{InTokens: 8, OutTokens: 3}}, {FinishReason: "stop"},
	}})
	model := func(name string) llm.Model {
		return llm.Model{Provider: "fake", Name: name, NativeJSONSchema: true, InputMicroUSDPerM: 1_000_000, OutputMicroUSDPerM: 1_000_000}
	}
	cfg := llm.Config{
		Chains: map[ports.ModelTier][]llm.Model{
			ports.TierSmall: {model("small")}, ports.TierStrong: {model("strong")}, ports.TierClassifier: {model("classifier")},
			ports.TierEmbedding: {{Provider: "fake", Name: "embedding", Dimensions: 3}},
		}, DefaultTimeout: time.Second, RetryBase: 0, RetryMax: 0, BreakerWindow: time.Minute, BreakerCooldown: time.Minute, BreakerErrorRate: .5, BreakerMinSamples: 2,
	}
	cli, err := llm.New(cfg, map[string]llm.Provider{"fake": provider}, llm.Options{})
	if err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, orchestrator.Options{})
	h.b.client = cli
	h.g.client = cli
	r, err := h.o.ProcessTurn(context.Background(), h.request(h.conv))
	if err != nil || r.Status != c.TurnCompleted || r.Text != "Synthetic answer" {
		t.Fatalf("real WP04 client did not complete two stages: %+v %v", r, err)
	}
	history := provider.History()
	if len(history) != 2 || history[0].Operation != "structured" || history[1].Operation != "stream" || history[0].Model != "small" || history[1].Model != "strong" {
		t.Fatalf("wrong model call inventory: %+v", history)
	}
	snap, ok := h.tr.Snapshot(r.TurnID)
	if !ok || snap.Usage.InTokens != 13 || snap.Usage.OutTokens != 5 {
		t.Fatalf("usage not traced: %+v", snap)
	}
}
