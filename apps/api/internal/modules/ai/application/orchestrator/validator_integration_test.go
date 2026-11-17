package orchestrator_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/gateway"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/orchestrator"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	outputvalidator "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/validator"
)

type wp07Fixture struct{ files map[string][]byte }

func (b wp07Fixture) Artifact(ctx context.Context, name string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	raw, ok := b.files[name]
	if !ok {
		return nil, errors.New("not found")
	}
	return append([]byte(nil), raw...), nil
}
func (b wp07Fixture) Reference() c.BundleRef {
	return c.BundleRef{Version: "kb@0.0.1", PromptSHAs: map[string]string{}}
}
func (b wp07Fixture) SyntheticOnly() bool { return true }

func wp07Validator(t *testing.T) *outputvalidator.Service {
	t.Helper()
	lex, err := os.ReadFile(filepath.Join("..", "..", "validator", "testdata", "validator.en.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{
		"bundle.yaml":                     []byte("version: 0.0.1\nlocales: [en]\nsynthetic_only: true\n"),
		"lexicons/validator.en.yaml":      lex,
		"resources/crisis_resources.yaml": []byte("schema: crisis_resources.v1\nfallback_locale: en\nlocales:\n  en:\n    - {id: test_988, name: test_only, kind: hotline, contact: '988', verified_on: '2026-11-16'}\n"),
		"config/thresholds.yaml":          []byte("schema: thresholds.v1\nvalidator:\n  directive_advice: 0.8\n  certainty_about_inference: 0.8\n  dependency_encouraging: 0.8\n  invalidating: 0.8\n"),
	}
	v, err := outputvalidator.New(context.Background(), wp07Fixture{files}, outputvalidator.Options{AllowSyntheticForTests: true})
	if err != nil {
		t.Fatal(err)
	}
	return v
}

type wp07Assembler struct{}

func (wp07Assembler) Build(_ context.Context, req c.TurnRequest, _ c.SafetySignal, _ c.RetrievalResult) (c.WorkingContext, ports.StateSnapshot, error) {
	return c.WorkingContext{Turn: c.TurnView{Text: req.Text, Lang: req.Locale}, Prefs: c.PrefsView{Language: req.Locale}, Eligibility: c.Eligibility{QuestionAllowed: true, MemoryAvailable: true}, Risk: c.RiskView{Current: c.RiskLow}}, ports.StateSnapshot{}, nil
}

type wp07Generator struct {
	mu           sync.Mutex
	texts        []string
	calls        int
	regenNotes   []string
	sentinels    []string
	finishReason string
}

func (g *wp07Generator) Generate(ctx context.Context, in ports.GenInput, sink ports.TokenSink) (ports.GenOutput, error) {
	g.mu.Lock()
	i := g.calls
	g.calls++
	note := ""
	if in.RegenNote != nil {
		note = *in.RegenNote
	}
	g.regenNotes = append(g.regenNotes, note)
	if i >= len(g.texts) {
		i = len(g.texts) - 1
	}
	txt := g.texts[i]
	sentinel := ""
	if i < len(g.sentinels) {
		sentinel = g.sentinels[i]
	}
	g.mu.Unlock()
	if sentinel != "" {
		return ports.GenOutput{Sentinel: sentinel}, nil
	}
	for _, piece := range []string{txt[:len(txt)/2], txt[len(txt)/2:]} {
		if err := ctx.Err(); err != nil {
			return ports.GenOutput{}, err
		}
		if err := sink.Send(piece); err != nil {
			return ports.GenOutput{}, err
		}
	}
	return ports.GenOutput{Text: txt, Usage: c.LLMUsage{FinishReason: g.finishReason, OutTokens: 4}}, nil
}
func (g *wp07Generator) snapshot() (int, []string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.calls, append([]string(nil), g.regenNotes...)
}

type wp07Planner struct{ plan c.CrisisPlan }

func (p wp07Planner) Plan(context.Context, c.SafetySignal, string, c.RiskState) (c.CrisisPlan, error) {
	return p.plan, nil
}

func setupWP07(t *testing.T, texts []string, fallback string, risk c.RiskLevel, cp c.CrisisPlan) (*harness, *wp07Generator) {
	t.Helper()
	h := newHarness(t, orchestrator.Options{})
	g := &wp07Generator{texts: texts}
	gatewayService, err := gateway.New(h.s, h.s)
	if err != nil {
		t.Fatal(err)
	}
	if fallback == "" {
		fallback = "Synthetic fallback copy."
	}
	service, err := orchestrator.New(orchestrator.Dependencies{Gateway: gatewayService, Store: h.s, Finalizer: h.s, Recovery: h.s,
		Safety: safety{level: risk}, Assembler: wp07Assembler{}, Brain: h.b, Generator: g, Validator: wp07Validator(t), Tracer: h.tr, Crisis: wp07Planner{cp}},
		orchestrator.Options{TotalTimeout: 5 * time.Second, GeneratorTimeout: time.Second,
			Fallback: func(_ c.Route, _ c.Objective, _ string) (string, bool) { return fallback, true }})
	if err != nil {
		t.Fatal(err)
	}
	h.o = service
	return h, g
}

func TestWP07RegeneratesOnceAndPersistsOnlyApprovedText(t *testing.T) {
	h, g := setupWP07(t, []string{"You have depression.", "That sounds difficult, and I'm listening."}, "Synthetic fallback copy.", c.RiskLow, c.CrisisPlan{})
	req := h.request(h.conv)
	result, err := h.o.ProcessTurn(context.Background(), req)
	if err != nil || result.Status != c.TurnCompleted || result.Text != "That sounds difficult, and I'm listening." || !result.Degraded {
		t.Fatalf("regen outcome %+v %v", result, err)
	}
	calls, notes := g.snapshot()
	if calls != 2 || len(notes) != 2 || notes[0] != "" || !strings.Contains(notes[1], "OV-003") {
		t.Fatalf("regeneration contract violated %d %+v", calls, notes)
	}
	if n, completed, _ := h.s.snapshot(); n != 2 || completed != 1 {
		t.Fatalf("unsafe intermediate output was persisted: %d/%d", n, completed)
	}
	snap, ok := h.tr.Snapshot(result.TurnID)
	if !ok {
		t.Fatal("missing validator metadata")
	}
	seen := false
	for _, step := range snap.Steps {
		if step.RuleID == "OV-003" {
			seen = true
		}
	}
	if !seen {
		t.Fatal("rule code omitted from metadata trace")
	}
	replay, err := h.o.ProcessTurn(context.Background(), req)
	if err != nil || replay != result {
		t.Fatalf("replay reran validator: %+v %v", replay, err)
	}
	calls, _ = g.snapshot()
	if calls != 2 {
		t.Fatal("duplicate generator call")
	}
}
func TestWP07SecondViolationFallsBackAndChecksTemplate(t *testing.T) {
	h, g := setupWP07(t, []string{"You have depression.", "As your therapist, I know best."}, "Reviewed synthetic fallback.", c.RiskLow, c.CrisisPlan{})
	result, err := h.o.ProcessTurn(context.Background(), h.request(h.conv))
	if err != nil || result.Text != "Reviewed synthetic fallback." || !result.Degraded {
		t.Fatalf("fallback outcome %+v %v", result, err)
	}
	if calls, _ := g.snapshot(); calls != 2 {
		t.Fatalf("unexpected attempts %d", calls)
	}
	n, completed, _ := h.s.snapshot()
	if n != 2 || completed != 1 {
		t.Fatal("invalid intermediate draft was saved")
	}
}
func TestWP07BlockedFallbackLeavesDurableUserOnly(t *testing.T) {
	h, g := setupWP07(t, []string{"You have depression.", "As your therapist, I know best."}, "Take 20mg of that medication.", c.RiskLow, c.CrisisPlan{})
	result, err := h.o.ProcessTurn(context.Background(), h.request(h.conv))
	if err == nil || result.Status != c.TurnFailed || result.AssistantMessageID != nil {
		t.Fatalf("unsafe fallback persisted: %+v %v", result, err)
	}
	if calls, _ := g.snapshot(); calls != 2 {
		t.Fatalf("unexpected attempts %d", calls)
	}
	if n, completed, _ := h.s.snapshot(); n != 1 || completed != 0 {
		t.Fatalf("user-only failure not retained: %d/%d", n, completed)
	}
}
func TestWP07SentinelReroutesWithoutRegeneration(t *testing.T) {
	h, g := setupWP07(t, []string{"<<PLAN_UNSAFE>>", "Unreachable next generation."}, "Reviewed synthetic reflection.", c.RiskLow, c.CrisisPlan{})
	result, err := h.o.ProcessTurn(context.Background(), h.request(h.conv))
	if err != nil || result.Text != "Reviewed synthetic reflection." || !result.Degraded {
		t.Fatalf("sentinel reroute %+v %v", result, err)
	}
	if calls, _ := g.snapshot(); calls != 1 {
		t.Fatalf("unsafe plan was regenerated %d times", calls)
	}
}
func TestWP07HighRiskBuffersAndUsesApprovedCrisisTemplate(t *testing.T) {
	cp := c.CrisisPlan{TemplateFamily: "synthetic_case", Buffered: true, RequiredElements: []string{"acknowledge", "care", "ask_safety", "resources", "invite_continue"},
		Resources:    []c.CrisisResource{{ID: "test_988", Contact: "988", Locale: "en"}},
		TemplateText: "I hear you and care about your safety. Are you safe? Please call 988. I'm here if you want to continue."}
	h, g := setupWP07(t, []string{"I hear you.", "Unsafe generic script."}, "Non-crisis fallback not used.", c.RiskHigh, cp)
	result, err := h.o.ProcessTurn(context.Background(), h.request(h.conv))
	if err != nil || result.Status != c.TurnCompleted || result.Route != c.RouteCrisis || result.Text != cp.TemplateText || !result.Degraded || result.Safety == nil {
		t.Fatalf("crisis static fallback %+v %v", result, err)
	}
	if calls, _ := g.snapshot(); calls != 2 {
		t.Fatalf("crisis validator not retried: %d", calls)
	}
	if h.b.count() != 0 {
		t.Fatal("crisis bypassed Brain incorrectly")
	}
}

func TestWP07StructuredSentinelReroutes(t *testing.T) {
	h, g := setupWP07(t, []string{"unused"}, "Synthetic reviewed reflection.", c.RiskLow, c.CrisisPlan{})
	g.sentinels = []string{"PLAN_UNSAFE"}
	result, err := h.o.ProcessTurn(context.Background(), h.request(h.conv))
	if err != nil || result.Status != c.TurnCompleted || result.Text != "Synthetic reviewed reflection." || !result.Degraded {
		t.Fatalf("structured sentinel was not safely rerouted: %+v %v", result, err)
	}
	if n, _ := g.snapshot(); n != 1 {
		t.Fatalf("structured sentinel caused regeneration: %d", n)
	}
}
