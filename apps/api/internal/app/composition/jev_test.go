package composition

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/fake"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/llmhttp"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/knowledge"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

func TestJEVRuntimeUsesOnlyNativeFreeModel(t *testing.T) {
	cfg, _, e := readRuntime(runtimeExamplePath)
	if e != nil {
		t.Fatal(e)
	}
	p := cfg.Providers["OpenCode"]
	if p.KeyEnv != "OPENCODE_API_KEY" || p.BaseURL != "https://opencode.ai/inference/systemone/v1" {
		t.Fatal("incorrect native endpoint or key variable")
	}
	for _, mutate := range []func(*runtimeComponent){
		func(c *runtimeComponent) { c.Chain[0].Provider = "openai" },
		func(c *runtimeComponent) { c.Chain[0].Name = "gpt-6-luna" },
		func(c *runtimeComponent) { c.Chain[0].BillingSource = "paid" },
		func(c *runtimeComponent) { c.Chain[0].InputMicroUSDPerM = 1 },
		func(c *runtimeComponent) { c.Chain = append(c.Chain, c.Chain[0]) },
	} {
		trial, _, e := readRuntime(runtimeExamplePath)
		if e != nil {
			t.Fatal(e)
		}
		settings := trial.LLM.Components[ports.ComponentJEV]
		mutate(&settings)
		trial.LLM.Components[ports.ComponentJEV] = settings
		if _, e = trial.LLM.clientConfig(); e == nil {
			t.Fatal("invalid JEV configuration accepted")
		}
	}
}

// Explicit opt-in only: sends fabricated context to the real configured JEV API.
// The caller supplies OPENCODE_API_KEY through the environment, never a fixture.
func TestJEVSystemOneLive(t *testing.T) {
	if os.Getenv("THEIA_JEV_LIVE_TEST") != "true" {
		t.Skip("explicit synthetic live verification only")
	}
	cfg, _, e := readRuntime(runtimeExamplePath)
	if e != nil {
		t.Fatal(e)
	}
	settings := cfg.Providers["OpenCode"]
	provider, e := llmhttp.NewSystemOne(settings.BaseURL, os.Getenv(settings.KeyEnv), nil)
	if e != nil {
		t.Fatal(e)
	}
	config, e := cfg.LLM.clientConfig()
	if e != nil {
		t.Fatal(e)
	}
	client, e := llm.New(config, map[string]llm.Provider{"OpenCode": provider, "openai": fake.New()}, llm.Options{})
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pinned, _, e := bundle.Read(ctx, "../../../../../knowledge", bundle.Options{})
	if e != nil {
		t.Fatal(e)
	}
	selector, e := knowledge.New(ctx, configuredComponentClient{next: client, components: cfg.LLM.Components}, pinned, knowledge.Options{Timeout: time.Duration(cfg.LLM.Components[ports.ComponentJEV].TimeoutMS) * time.Millisecond})
	if e != nil {
		t.Fatal(e)
	}
	result, e := selector.Select(ctx, knowledge.Input{UserID: c.NewID(), Bundle: pinned.Reference(), Context: c.WorkingContext{Turn: c.TurnView{TurnID: c.NewID(), MessageID: c.NewID(), Text: "Synthetic fixture: I would like to plan my busy week."}}})
	if e != nil {
		t.Fatal(e)
	}
	if result.Degraded {
		t.Fatalf("native JEV unavailable: %s", result.Reason)
	}
	if result.Usage.Provider != "OpenCode" || result.Usage.Model != "jev-1.13-free" || result.Usage.Attempts != 1 || result.Usage.CostMicroUSD != 0 {
		t.Fatal("incorrect native usage metadata")
	}
	t.Logf("native JEV succeeded: %d resources; input=%d output=%d cost=0", len(result.Selected), result.Usage.InTokens, result.Usage.OutTokens)
}
