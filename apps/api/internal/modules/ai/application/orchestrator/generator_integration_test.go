package orchestrator_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/fake"
	brainpkg "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/brain"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/gateway"
	genpkg "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/generator"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/orchestrator"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	safetypkg "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/safety"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	outputvalidator "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/validator"
)

type wp11Provider struct {
	wp10Provider
	script    *fake.Provider
	requests  []ports.StreamRequest
	retriever ports.Retriever
}

func (p *wp11Provider) Stream(ctx context.Context, m llm.Model, in ports.StreamRequest) (ports.Stream, error) {
	p.requests = append(p.requests, in)
	return p.script.Stream(ctx, m, in)
}
func wp11Reply(text string) fake.StreamReply {
	return fake.StreamReply{FailAt: -1, Chunks: []ports.Chunk{{Text: text}, {FinishReason: "stop", Usage: &c.LLMUsage{InTokens: 30, OutTokens: 8}}}}
}

type wp11Verify func(context.Context, c.ID, ports.GenInput) error

func (f wp11Verify) VerifyGeneration(ctx context.Context, u c.ID, in ports.GenInput) error {
	return f(ctx, u, in)
}
func wp11Runtime(t *testing.T, h *testIntegrationHarness, p *wp11Provider, verifier genpkg.SourceVerifier, modify func(*testIntegrationBundle), classifiers ...outputvalidator.Classifier) *orchestrator.Service {
	t.Helper()
	raw, e := os.ReadFile("../generator/testdata/assets.json")
	if e != nil {
		t.Fatal(e)
	}
	var files map[string]string
	if json.Unmarshal(raw, &files) != nil {
		t.Fatal("fixture JSON")
	}
	for k, v := range files {
		h.registry.b.files[k] = []byte(v)
	}
	lex, e := os.ReadFile("../../validator/testdata/validator.en.yaml")
	if e != nil {
		t.Fatal(e)
	}
	h.registry.b.files["lexicons/validator.en.yaml"] = lex
	if modify != nil {
		modify(h.registry.b)
	}
	now := func() time.Time { return time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC) }
	h.assets, e = safetypkg.Load(context.Background(), h.registry.b, safetypkg.LoadOptions{AllowSyntheticForTests: true})
	if e != nil {
		t.Fatal(e)
	}
	validatorOptions := outputvalidator.Options{AllowSyntheticForTests: true}
	if len(classifiers) > 0 {
		validatorOptions.Classifier = classifiers[0]
	}
	v, e := outputvalidator.New(context.Background(), h.registry.b, validatorOptions)
	if e != nil {
		t.Fatal(e)
	}
	ba, e := brainpkg.Load(context.Background(), h.registry.b, brainpkg.LoadOptions{AllowSyntheticForTests: true})
	if e != nil {
		t.Fatal(e)
	}
	ga, e := genpkg.Load(context.Background(), h.registry.b, genpkg.LoadOptions{AllowSyntheticForTests: true, Now: now})
	if e != nil {
		t.Fatal(e)
	}
	m := llm.Model{Provider: "wp11", Name: "primary", NativeJSONSchema: true}
	secondary := llm.Model{Provider: "wp11", Name: "secondary"}
	cfg := llm.Config{Chains: map[ports.ModelTier][]llm.Model{ports.TierSmall: {m}, ports.TierStrong: {m, secondary}, ports.TierClassifier: {m}, ports.TierEmbedding: {{Provider: "wp11", Name: "embed", Dimensions: 3}}}, DefaultTimeout: time.Second, BreakerWindow: time.Minute, BreakerCooldown: time.Minute, BreakerErrorRate: .5, BreakerMinSamples: 10}
	client, e := llm.New(cfg, map[string]llm.Provider{"wp11": p}, llm.Options{})
	if e != nil {
		t.Fatal(e)
	}
	b, e := brainpkg.New(client, ba, h.r, brainpkg.Options{})
	if e != nil {
		t.Fatal(e)
	}
	g, e := genpkg.New(client, ga, genpkg.Options{Verifier: verifier})
	if e != nil {
		t.Fatal(e)
	}
	gw, e := gateway.New(h.s, h.s)
	if e != nil {
		t.Fatal(e)
	}
	if p.retriever == nil {
		p.retriever = &wp10Retrieval{}
	}
	o, e := orchestrator.NewIntegrated(orchestrator.Dependencies{Gateway: gw, Store: h.s, Finalizer: testIntegrationFinalizer{h.s, h.r}, Recovery: h.s, Brain: b, Retriever: p.retriever, Generator: g, Validator: v, Tracer: trace.NewRecorder()}, orchestrator.Options{TotalTimeout: 5 * time.Second, RouterConfig: ports.RouterConfig{TrivialPatterns: map[string][]string{"en": {"hello"}}}}, orchestrator.IntegratedOptions{Reader: h.r, Bundles: h.registry, Assets: h.assets, Classifier: h.classifier, AllowSyntheticForTests: true, Now: now})
	if e != nil {
		t.Fatal(e)
	}
	return o
}

func TestWP11ValidatedDeliveryRegenerationAndFallback(t *testing.T) {
	for _, mode := range []string{"success", "stream_guard_regen", "posthoc_regen", "second_block_template", "partial_stream_retry", "sentinel", "provider_outage", "crisis_static", "fast"} {
		t.Run(mode, func(t *testing.T) {
			h := newTestIntegrationHarness(t)
			p := &wp11Provider{wp10Provider: wp10Provider{makeDecision: wp10Decision}, script: fake.New()}
			want := "Synthetic accepted answer."
			body := "ordinary synthetic event"
			calls := 1
			degraded := false
			switch mode {
			case "success":
				p.script.QueueStream("primary", wp11Reply(want))
			case "stream_guard_regen":
				p.script.QueueStream("primary", wp11Reply("You have depression."), wp11Reply(want))
				calls = 2
				degraded = true
			case "posthoc_regen":
				p.script.QueueStream("primary", wp11Reply("You should act now."), wp11Reply(want))
				calls = 2
				degraded = true
			case "second_block_template":
				p.script.QueueStream("primary", wp11Reply("You have depression."), wp11Reply("You have depression."))
				want = "SYNTHETIC NORMAL FALLBACK"
				calls = 2
				degraded = true
			case "partial_stream_retry":
				p.script.QueueStream("primary", fake.StreamReply{FailAt: 1, Chunks: []ports.Chunk{{Text: "PRIVATE abandoned draft"}}, Err: &llm.Failure{Kind: llm.FailureTransport}}, wp11Reply(want))
				calls = 2
				degraded = true
			case "sentinel":
				p.script.QueueStream("primary", wp11Reply("<<PLAN_UNSAFE>>"))
				want = "SYNTHETIC NORMAL FALLBACK"
				degraded = true
			case "provider_outage":
				want = "SYNTHETIC NORMAL FALLBACK"
				calls = 2
				degraded = true
			case "crisis_static":
				p.script.QueueStream("primary", wp11Reply("Synthetic invalid crisis output."), wp11Reply("Synthetic invalid crisis output."))
				body = "TEST_HIGH_SIGNAL"
				want = "SYNTHETIC TEST CRISIS SCRIPT (not for people). https://resources.invalid/wp11"
				calls = 2
				degraded = true
			case "fast":
				p.script.QueueStream("primary", wp11Reply(want))
				body = "hello"
			}
			o := wp11Runtime(t, h, p, nil, nil)
			req := h.request(body)
			res, e := o.ProcessTurn(context.Background(), req)
			if e != nil || res.Status != c.TurnCompleted || res.Text != want || res.Degraded != degraded {
				t.Fatal("incorrect final delivery", res, e)
			}
			if len(p.requests) != calls {
				t.Fatal("unbounded retries", len(p.requests), calls)
			}
			if mode == "fast" || mode == "crisis_static" {
				if p.calls != 0 {
					t.Fatal("bypass called Brain")
				}
			} else if p.calls != 1 {
				t.Fatal("unexpected Brain calls", p.calls)
			}
			if mode == "stream_guard_regen" || mode == "posthoc_regen" {
				if !strings.Contains(p.requests[1].Messages[0].Content, "<regen_note>OV-") || strings.Contains(p.requests[1].Messages[0].Content, "You have depression") {
					t.Fatal("regen note included draft or lost codes")
				}
			}
			if mode == "sentinel" && h.last(t).Objective != c.ObjReflect {
				t.Fatal("unsafe plan not rerouted")
			}
			if strings.Contains(res.Text, "PRIVATE") || strings.Contains(res.Text, "<<PLAN_UNSAFE>>") {
				t.Fatal("discarded draft reached output")
			}
			n, done, _ := h.s.snapshot()
			if n != 2 || done != 1 {
				t.Fatal("unvalidated draft persisted", n, done)
			}
			again, e := o.ProcessTurn(context.Background(), req)
			if e != nil || again != res || len(p.requests) != calls {
				t.Fatal("replay regenerated", again, e)
			}
		})
	}
}

func TestWP11AuthorizationAndValidationFailClosed(t *testing.T) {
	for _, mode := range []string{"foreign_source", "fallback_invalid", "commit_failure", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			h := newTestIntegrationHarness(t)
			p := &wp11Provider{wp10Provider: wp10Provider{makeDecision: wp10Decision}, script: fake.New()}
			p.script.QueueStream("primary", wp11Reply("Synthetic accepted answer."))
			var verify genpkg.SourceVerifier
			var modify func(*testIntegrationBundle)
			switch mode {
			case "foreign_source":
				verify = wp11Verify(func(_ context.Context, u c.ID, in ports.GenInput) error {
					if u != h.user || !c.ValidID(in.Turn.TurnID) {
						t.Fatal("missing identity")
					}
					return errors.New("PRIVATE foreign source")
				})
			case "fallback_invalid":
				p.script = fake.New()
				modify = func(b *testIntegrationBundle) {
					b.files["templates/fallback_by_objective.yaml"] = []byte(`{"schema":"fallback.v1","templates":{"reflect":{"id":"test","text":"You have depression."}}}`)
				}
			case "commit_failure":
				h.s.rejectCommit = true
			}
			o := wp11Runtime(t, h, p, verify, modify)

			ctx := context.Background()
			if mode == "cancelled" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			res, e := o.ProcessTurn(ctx, h.request("ordinary synthetic event"))
			if e == nil || res.AssistantMessageID != nil || res.Text != "" {
				t.Fatal("failed turn exposed output", res, e)
			}
			if mode == "foreign_source" && len(p.requests) != 0 {
				t.Fatal("foreign source reached provider")
			}
			if mode != "cancelled" {
				n, done, _ := h.s.snapshot()
				if n != 1 || done != 0 {
					t.Fatal("failure did not retain user-only turn", n, done)
				}
			}
		})
	}
}

func TestWP11GenerationStageDeadlineAndNoRegenerationBudget(t *testing.T) {
	for _, mode := range []string{"deadline", "zero_regens"} {
		t.Run(mode, func(t *testing.T) {
			h := newTestIntegrationHarness(t)
			p := &wp11Provider{wp10Provider: wp10Provider{makeDecision: wp10Decision}, script: fake.New()}
			reply := wp11Reply("You have depression.")
			if mode == "deadline" {
				reply = wp11Reply("Synthetic delayed reply.")
				reply.Delay = 100 * time.Millisecond
			}
			p.script.QueueStream("primary", reply, wp11Reply("Synthetic should not run."))
			o := wp11Runtime(t, h, p, nil, func(b *testIntegrationBundle) {
				var cfg map[string]any
				_ = json.Unmarshal(b.files["config/budgets.yaml"], &cfg)
				if mode == "deadline" {
					cfg["turn"].(map[string]any)["gen_total_ms"] = 20
					cfg["turn"].(map[string]any)["gen_first_token_ms"] = 20
				} else {
					cfg["limits"].(map[string]any)["max_regenerations"] = 0
				}
				b.files["config/budgets.yaml"], _ = json.Marshal(cfg)
			})
			res, e := o.ProcessTurn(context.Background(), h.request("ordinary synthetic event"))
			if e != nil || res.Text != "SYNTHETIC NORMAL FALLBACK" || !res.Degraded {
				t.Fatal(res, e)
			}
			if len(p.requests) > 2 || mode == "zero_regens" && len(p.requests) != 1 {
				t.Fatal("generation exceeded configured stage/retry bounds", len(p.requests))
			}
		})
	}
}

func TestWP11ElevatedAndFollowUpProviderOutage(t *testing.T) {
	for _, mode := range []string{"elevated", "follow_up", "missing_approved_family"} {
		t.Run(mode, func(t *testing.T) {
			h := newTestIntegrationHarness(t)
			if mode == "follow_up" {
				until := time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC)
				h.r.risk[h.user] = c.RiskState{Level: c.RiskElevated, FollowUpUntil: &until}
			} else {
				h.classifier.level = c.RiskElevated
			}
			p := &wp11Provider{wp10Provider: wp10Provider{makeDecision: wp10Decision}, script: fake.New()}
			o := wp11Runtime(t, h, p, nil, func(b *testIntegrationBundle) {
				if mode != "missing_approved_family" {
					return
				}
				var cfg map[string]any
				_ = json.Unmarshal(b.files["policy/safety_policy.yaml"], &cfg)
				categories := cfg["categories"].(map[string]any)
				categories["self_harm"] = categories["other"]
				delete(categories, "other")
				b.files["policy/safety_policy.yaml"], _ = json.Marshal(cfg)
			})
			res, e := o.ProcessTurn(context.Background(), h.request("ordinary synthetic event"))
			if mode == "missing_approved_family" {
				if e == nil || res.AssistantMessageID != nil {
					t.Fatal("invented crisis family used", res, e)
				}
				return
			}
			if e != nil || res.Route != c.RouteCrisis || res.Safety == nil || len(res.Safety.Resources) == 0 || res.Text != "SYNTHETIC TEST CRISIS SCRIPT (not for people). https://resources.invalid/wp11" {
				t.Fatal("FB-006 lost curated crisis fallback", res, e)
			}
			if len(p.requests) != 2 {
				t.Fatal("outage exceeded turn-wide attempt budget", len(p.requests))
			}
			if h.last(t).State.User.Risk != nil && h.last(t).State.User.Risk.To.Rank() > c.RiskElevated.Rank() {
				t.Fatal("template selection fabricated risk escalation")
			}
		})
	}
}

func TestWP11TurnCallBudgetIncludesBrainAndRegeneration(t *testing.T) {
	h := newTestIntegrationHarness(t)
	p := &wp11Provider{wp10Provider: wp10Provider{makeDecision: wp10Decision}, script: fake.New()}
	p.script.QueueStream("primary", wp11Reply("You have depression."), wp11Reply("Synthetic must not be attempted."))
	o := wp11Runtime(t, h, p, nil, nil)
	ctx := c.WithBudget(context.Background(), c.Budget{MaxLLMCalls: 2})
	res, e := o.ProcessTurn(ctx, h.request("ordinary synthetic event"))
	if e != nil || res.Text != "SYNTHETIC NORMAL FALLBACK" || p.calls != 1 || len(p.requests) != 1 {
		t.Fatal("regeneration exceeded shared call allowance", res, e, p.calls, len(p.requests))
	}
}

type wp11ValidatorOutage struct{}

func (wp11ValidatorOutage) Scores(context.Context, string, []c.MsgView, string) (map[string]float32, error) {
	return nil, errors.New("PRIVATE classifier outage")
}

func TestWP11ValidatorOutageAlsoBlocksFallback(t *testing.T) {
	h := newTestIntegrationHarness(t)
	p := &wp11Provider{wp10Provider: wp10Provider{makeDecision: wp10Decision}, script: fake.New()}
	p.script.QueueStream("primary", wp11Reply("Synthetic accepted candidate."))
	o := wp11Runtime(t, h, p, nil, func(b *testIntegrationBundle) {
		var cfg map[string]any
		_ = json.Unmarshal(b.files["config/thresholds.yaml"], &cfg)
		cfg["validator"] = map[string]float64{"directive_advice": .8, "certainty_about_inference": .8, "dependency_encouraging": .8, "invalidating": .8}
		b.files["config/thresholds.yaml"], _ = json.Marshal(cfg)
	}, wp11ValidatorOutage{})
	res, e := o.ProcessTurn(context.Background(), h.request("ordinary synthetic event"))
	if e == nil || res.AssistantMessageID != nil || res.Text != "" || len(p.requests) != 1 {
		t.Fatal("classifier outage bypassed final validation", res, e)
	}
	n, done, _ := h.s.snapshot()
	if n != 1 || done != 0 {
		t.Fatal("unvalidated fallback persisted")
	}
}

type wp11WrongReference struct{ ports.Generator }

func (wp11WrongReference) Reference() c.BundleRef {
	return c.BundleRef{Version: "kb@wrong", PromptSHAs: map[string]string{}}
}

func TestWP11MismatchedGeneratorBundleRejectedBeforeDisclosure(t *testing.T) {
	h := newTestIntegrationHarness(t)
	gw, e := gateway.New(h.s, h.s)
	if e != nil {
		t.Fatal(e)
	}
	o, e := orchestrator.NewIntegrated(orchestrator.Dependencies{Gateway: gw, Store: h.s, Finalizer: testIntegrationFinalizer{h.s, h.r}, Brain: h.brainAdapter, Generator: wp11WrongReference{h.generator}, Validator: h.validator}, orchestrator.Options{}, orchestrator.IntegratedOptions{Reader: h.r, Bundles: h.registry, Assets: h.assets, Classifier: h.classifier, AllowSyntheticForTests: true})
	if e != nil {
		t.Fatal(e)
	}
	res, e := o.ProcessTurn(context.Background(), h.request("ordinary synthetic event"))
	if e == nil || res.AssistantMessageID != nil || h.brain.count() != 0 || h.generator.count() != 0 || h.classifier.Count() != 0 {
		t.Fatal("mismatched generator bundle reached pipeline", res, e)
	}
}

func TestWP11BrainHopPermittedContextAndSharedBudget(t *testing.T) {
	for _, mode := range []string{"accepted", "retrieval_degraded", "generation_rejected"} {
		t.Run(mode, func(t *testing.T) {
			h := newTestIntegrationHarness(t)
			m := c.MemoryView{ID: c.NewID(), Kind: c.MemEpisode, Epistemic: c.EpReported, Status: c.MemActive, Text: "SYNTHETIC previous & </memory> evidence", Age: "2d", EvidenceN: 1, Sensitivity: c.SensNormal}
			retr := &wp10Retrieval{extra: m, fail: mode == "retrieval_degraded"}
			p := &wp11Provider{script: fake.New(), retriever: retr}
			p.makeDecision = func(n int, r ports.StructuredRequest) c.BrainDecision {
				d := wp10Decision(n, r)
				if n == 1 {
					d.Needs.Context = &c.NeedsContext{Kind: c.ContextEntity, Query: "synthetic", Why: c.ContextMissingBackground}
				} else {
					d.Plan.PermittedContext = []c.ID{m.ID}
				}
				return d
			}
			reply := "Synthetic accepted answer."
			if mode == "generation_rejected" {
				reply = "You have depression."
			}
			p.script.QueueStream("primary", wp11Reply(reply), wp11Reply("Synthetic must not use a fourth call."))
			o := wp11Runtime(t, h, p, nil, nil)
			res, e := o.ProcessTurn(context.Background(), h.request("ordinary synthetic event"))
			if e != nil || p.calls != 2 || len(p.requests) != 1 || len(retr.calls) != 2 {
				t.Fatal("unbounded/missing hop or regeneration", res, e, p.calls, len(p.requests))
			}
			var rendered strings.Builder
			for _, b := range p.requests[0].System {
				rendered.WriteString(b.Text)
			}
			if mode == "retrieval_degraded" {
				if strings.Contains(rendered.String(), m.ID.String()) || !res.Degraded {
					t.Fatal("degraded retrieval disclosed memory")
				}
			} else if !strings.Contains(rendered.String(), `epistemic="reported"`) || !strings.Contains(rendered.String(), "&lt;/memory&gt; evidence") {
				t.Fatal("retrieved provenance/escaping lost")
			}
			if mode == "generation_rejected" && res.Text != "SYNTHETIC NORMAL FALLBACK" {
				t.Fatal("fourth reasoning/generation call bypassed budget", res)
			}
		})
	}
}

func TestWP11ExpiredTurnBudgetNeverDeliversFallback(t *testing.T) {
	h := newTestIntegrationHarness(t)
	p := &wp11Provider{wp10Provider: wp10Provider{makeDecision: wp10Decision}, script: fake.New()}
	o := wp11Runtime(t, h, p, nil, nil)
	ctx := c.WithBudget(context.Background(), c.Budget{TotalDeadline: time.Now().Add(-time.Second)})
	res, e := o.ProcessTurn(ctx, h.request("ordinary synthetic event"))
	if e == nil || res.AssistantMessageID != nil || p.calls != 0 || len(p.requests) != 0 {
		t.Fatal("expired hard deadline permitted inference/output", res, e)
	}
}
