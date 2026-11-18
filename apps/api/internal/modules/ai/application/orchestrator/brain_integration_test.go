package orchestrator_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	brainpkg "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/brain"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/gateway"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/orchestrator"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	outputvalidator "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/validator"
)

func (b *testIntegrationBundle) ArtifactNames() []string {
	v := []string{}
	for p := range b.files {
		v = append(v, p)
	}
	return v
}

type wp10Provider struct {
	calls        int
	requests     []ports.StructuredRequest
	makeDecision func(int, ports.StructuredRequest) c.BrainDecision
	invalid      bool
}

func (p *wp10Provider) Structured(_ context.Context, _ llm.Model, req ports.StructuredRequest, _ llm.Mode) (llm.StructuredResult, error) {
	p.calls++
	p.requests = append(p.requests, req)
	if p.invalid {
		return llm.StructuredResult{JSON: []byte(`{"untrusted":"bad"}`)}, nil
	}
	raw, e := json.Marshal(p.makeDecision(p.calls, req))
	return llm.StructuredResult{JSON: raw, Usage: c.LLMUsage{InTokens: 10, OutTokens: 5}}, e
}
func (*wp10Provider) Stream(context.Context, llm.Model, ports.StreamRequest) (ports.Stream, error) {
	return nil, errors.New("test Brain only")
}
func (*wp10Provider) Embed(context.Context, llm.Model, ports.EmbeddingRequest) (llm.EmbeddingResult, error) {
	return llm.EmbeddingResult{}, errors.New("test Brain only")
}
func wp10Decision(_ int, req ports.StructuredRequest) c.BrainDecision {
	return c.BrainDecision{SchemaVersion: c.BrainSchemaVersion, TurnAnalysis: c.TurnAnalysis{Intent: c.IntentDiscloseEvent, MessageType: c.MessageDisclosure, ThreadRelation: c.RelContinues, UserReadiness: c.ConfLow, Emotions: []c.EmotionRead{}, PastRefs: []c.PastRef{}, Ambiguities: []c.Ambiguity{}}, Understanding: c.UnderstandingUpdate{Slots: []c.SlotUpdate{}, Hypotheses: []c.HypothesisSignal{}, Unknowns: []string{}}, Decision: c.Decision{Primary: c.ObjExplore, Rationale: []string{"high_affect"}, Alternatives: []c.Objective{}, Confidence: c.ConfHigh}, Plan: c.ResponsePlan{Skills: c.SkillSelection{Primary: "reflect_emotion", Secondary: []string{}}, Question: c.QuestionPlan{Allowed: false, MaxCount: 0}, Tone: c.ToneWarmSteady, Length: c.LengthShort, PermittedContext: []c.ID{}, MustAvoid: []string{"advice"}}}
}

type wp10Retrieval struct {
	calls []c.RetrievalRequest
	extra c.MemoryView
	fail  bool
}

func (r *wp10Retrieval) Retrieve(_ context.Context, req c.RetrievalRequest) (c.RetrievalResult, error) {
	r.calls = append(r.calls, req)
	if req.Extra != nil {
		if r.fail {
			return c.RetrievalResult{}, errors.New("private provider/source detail")
		}
		return c.RetrievalResult{Items: []c.MemoryView{r.extra}, Meta: c.RetrievalMeta{Hops: 1}}, nil
	}
	return c.RetrievalResult{Meta: c.RetrievalMeta{Strategies: []string{"entity"}}}, nil
}

type wp10Gen struct {
	calls []ports.GenInput
	text  string
}

func (g *wp10Gen) Generate(_ context.Context, in ports.GenInput, _ ports.TokenSink) (ports.GenOutput, error) {
	g.calls = append(g.calls, in)
	return ports.GenOutput{Text: g.text}, nil
}

func wp10Runtime(t *testing.T, h *testIntegrationHarness, p *wp10Provider, r ports.Retriever, g ports.Generator, actualValidator bool, memoryDeps ...orchestrator.TurnMemory) *orchestrator.Service {
	t.Helper()
	raw, e := os.ReadFile("../brain/testdata/assets.json")
	if e != nil {
		t.Fatal(e)
	}
	var files map[string]string
	if json.Unmarshal(raw, &files) != nil {
		t.Fatal("bad synthetic fixtures")
	}
	for path, v := range files {
		h.registry.b.files[path] = []byte(v)
	}
	// Synthetic fixtures share the SAME pinned reference across Safety, Brain,
	// Assembler, State and Validator. Nothing in knowledge/ is changed.
	var v ports.Validator = h.validator
	if actualValidator {
		lex, e := os.ReadFile("../../validator/testdata/validator.en.yaml")
		if e != nil {
			t.Fatal(e)
		}
		h.registry.b.files["lexicons/validator.en.yaml"] = lex
		var threshold map[string]any
		_ = json.Unmarshal(h.registry.b.files["config/thresholds.yaml"], &threshold)
		threshold["validator"] = map[string]float64{"directive_advice": .8, "certainty_about_inference": .8, "dependency_encouraging": .8, "invalidating": .8}
		h.registry.b.files["config/thresholds.yaml"], _ = json.Marshal(threshold)
		v, e = outputvalidator.New(context.Background(), h.registry.b, outputvalidator.Options{AllowSyntheticForTests: true})
		if e != nil {
			t.Fatal(e)
		}
	}
	a, e := brainpkg.Load(context.Background(), h.registry.b, brainpkg.LoadOptions{AllowSyntheticForTests: true})
	if e != nil {
		t.Fatal(e)
	}
	model := llm.Model{Provider: "wp10", Name: "test", NativeJSONSchema: true}
	cfg := llm.Config{Chains: map[ports.ModelTier][]llm.Model{ports.TierSmall: {model}, ports.TierStrong: {model}, ports.TierClassifier: {model}, ports.TierEmbedding: {{Provider: "wp10", Name: "embed", Dimensions: 3}}}, DefaultTimeout: time.Second, BreakerWindow: time.Minute, BreakerCooldown: time.Minute, BreakerErrorRate: .5, BreakerMinSamples: 10}
	client, e := llm.New(cfg, map[string]llm.Provider{"wp10": p}, llm.Options{})
	if e != nil {
		t.Fatal(e)
	}
	b, e := brainpkg.New(client, a, h.r, brainpkg.Options{})
	if e != nil {
		t.Fatal(e)
	}
	gw, e := gateway.New(h.s, h.s)
	if e != nil {
		t.Fatal(e)
	}
	var memoryDependency orchestrator.TurnMemory
	if len(memoryDeps) > 0 {
		memoryDependency = memoryDeps[0]
	}
	o, e := orchestrator.NewIntegrated(orchestrator.Dependencies{Memory: memoryDependency, Gateway: gw, Store: h.s, Finalizer: testIntegrationFinalizer{h.s, h.r}, Recovery: h.s, Brain: b, Retriever: r, Generator: g, Validator: v, Tracer: trace.NewRecorder()}, orchestrator.Options{TotalTimeout: 5 * time.Second}, orchestrator.IntegratedOptions{Reader: h.r, Bundles: h.registry, Assets: h.assets, Classifier: h.classifier, AllowSyntheticForTests: true, Now: func() time.Time { return time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC) }})
	if e != nil {
		t.Fatal(e)
	}
	return o
}
func TestWP10BoundedRetrievalAndPermittedContext(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "degraded"}[fail], func(t *testing.T) {
			h := newTestIntegrationHarness(t)
			item := c.MemoryView{ID: c.NewID(), Kind: c.MemEpisode, Epistemic: c.EpReported, Status: c.MemActive, Text: "Synthetic previous event", Age: "2d", EvidenceN: 1, Sensitivity: c.SensNormal}
			retr := &wp10Retrieval{extra: item, fail: fail}
			p := &wp10Provider{}
			p.makeDecision = func(n int, req ports.StructuredRequest) c.BrainDecision {
				d := wp10Decision(n, req)
				d.Needs.Context = &c.NeedsContext{Kind: c.ContextEntity, Query: "boss", Why: c.ContextMissingBackground}
				if n == 2 {
					d.Plan.PermittedContext = []c.ID{item.ID}
				}
				return d
			}
			g := &wp10Gen{text: "Synthetic answer"}
			o := wp10Runtime(t, h, p, retr, g, true)
			req := h.request("ordinary synthetic event")
			result, e := o.ProcessTurn(context.Background(), req)
			if e != nil || result.Status != c.TurnCompleted {
				t.Fatal(result, e)
			}
			if p.calls != 2 || len(retr.calls) != 2 || retr.calls[1].MaxItems != 1 || retr.calls[1].UserID != h.user || retr.calls[1].Extra == nil || len(g.calls) != 1 {
				t.Fatal("unbounded/missing hop", p.calls, retr.calls, g.calls)
			}
			if !fail && (len(g.calls[0].Permitted) != 1 || g.calls[0].Permitted[0].ID != item.ID || g.calls[0].Permitted[0].Epistemic != c.EpReported) {
				t.Fatal("lost provenance", g.calls)
			}
			if fail && len(g.calls[0].Permitted) != 0 {
				t.Fatal("degraded memory leaked")
			}
			again, e := o.ProcessTurn(context.Background(), req)
			if e != nil || again.TurnID != result.TurnID || p.calls != 2 {
				t.Fatal("replay called Brain", e)
			}
		})
	}
}
func TestWP10SafetyBypassAndRaisesBeforeGeneration(t *testing.T) {
	for _, mode := range []string{"pre_gate", "brain_raise", "second_brain_raise"} {
		t.Run(mode, func(t *testing.T) {
			h := newTestIntegrationHarness(t)
			retr := &wp10Retrieval{}
			g := &wp10Gen{text: "Synthetic answer"}
			p := &wp10Provider{}
			p.makeDecision = func(n int, req ports.StructuredRequest) c.BrainDecision {
				d := wp10Decision(n, req)
				if mode == "second_brain_raise" && n == 1 {
					d.Needs.Context = &c.NeedsContext{Kind: c.ContextEntity, Query: "boss", Why: c.ContextMissingBackground}
				} else {
					d.RiskRead = &c.BrainRiskRead{Level: c.RiskHigh, Categories: []c.RiskCategory{c.RiskOther}, ReasonTag: "synthetic_raise"}
				}
				return d
			}
			o := wp10Runtime(t, h, p, retr, g, false)
			text := "ordinary synthetic event"
			if mode == "pre_gate" {
				text = "TEST_HIGH_SIGNAL"
			}
			result, e := o.ProcessTurn(context.Background(), h.request(text))
			if e != nil || result.Route != c.RouteCrisis || len(g.calls) != 1 || g.calls[0].Crisis == nil {
				t.Fatal(result, e, g.calls)
			}
			expected := 1
			if mode == "pre_gate" {
				expected = 0
			}
			if mode == "second_brain_raise" {
				expected = 2
			}
			if p.calls != expected {
				t.Fatal("safety routing Brain calls", p.calls)
			}
			if mode == "brain_raise" && len(retr.calls) != 1 {
				t.Fatal("retrieved after high raise")
			}
			if h.last(t).State.User.Risk == nil || h.last(t).State.User.Risk.To != c.RiskHigh {
				t.Fatal("raise not committed")
			}
		})
	}
}
func TestWP10ReportedEvidenceAndStateProposal(t *testing.T) {
	h := newTestIntegrationHarness(t)
	tid := c.NewID()
	h.r.threads[h.conv] = c.ThreadRecord{ID: tid, UserID: h.user, Phase: c.PhaseExploring, Version: 1, Slots: map[string]c.SlotView{}}
	p := &wp10Provider{makeDecision: func(n int, req ports.StructuredRequest) c.BrainDecision {
		d := wp10Decision(n, req)
		re := regexp.MustCompile(`<user_message id="([^"]+)"`)
		matches := re.FindStringSubmatch(req.Messages[0].Content)
		id := uuid.MustParse(matches[1])
		d.Understanding.ThreadID = &tid
		d.Understanding.Slots = []c.SlotUpdate{{Slot: "situation", Value: "synthetic event", Status: c.EpReported, Evidence: []c.ID{id}}}
		return d
	}}
	g := &wp10Gen{text: "Synthetic answer"}
	o := wp10Runtime(t, h, p, &wp10Retrieval{}, g, true)
	res, e := o.ProcessTurn(context.Background(), h.request("ordinary synthetic event"))
	if e != nil || res.Status != c.TurnCompleted {
		t.Fatal(res, e)
	}
	patch := h.last(t).State.Thread
	if patch == nil || len(patch.Slots) != 1 || patch.Slots[0].Status != c.EpReported || patch.Slots[0].Evidence[0] != res.UserMessageID {
		t.Fatal("state provenance lost", patch)
	}
}
func TestWP10InvalidBrainAndOutputValidatorFallback(t *testing.T) {
	h := newTestIntegrationHarness(t)
	p := &wp10Provider{invalid: true, makeDecision: wp10Decision}
	g := &wp10Gen{text: "You have depression."}
	o := wp10Runtime(t, h, p, &wp10Retrieval{}, g, true)
	res, e := o.ProcessTurn(context.Background(), h.request("ordinary synthetic event"))
	if e != nil || res.Text != "SYNTHETIC NORMAL FALLBACK" || !res.Degraded {
		t.Fatal(res, e)
	}
	if p.calls != 1 || len(g.calls) != 2 || g.calls[0].Objective != c.ObjReflect || !strings.Contains(*g.calls[1].RegenNote, "OV-") {
		t.Fatal("fallback/validation bypass", p.calls, g.calls)
	}
	if h.last(t).State.Thread != nil {
		t.Fatal("fallback applied unexecuted state plan")
	}
}

func TestWP10NFCAndValidatedQuestionState(t *testing.T) {
	h := newTestIntegrationHarness(t)
	p := &wp10Provider{makeDecision: func(n int, req ports.StructuredRequest) c.BrainDecision {
		d := wp10Decision(n, req)
		d.Plan.Question = c.QuestionPlan{Allowed: true, MaxCount: 1, Purpose: "invite_detail", Slot: "situation"}
		return d
	}}
	g := &wp10Gen{text: "What happened next?"}
	o := wp10Runtime(t, h, p, &wp10Retrieval{}, g, true)
	res, e := o.ProcessTurn(context.Background(), h.request("synthetic cafe\u0301 event"))
	if e != nil || res.Status != c.TurnCompleted {
		t.Fatal(res, e)
	}
	if len(p.requests) != 1 || !strings.Contains(p.requests[0].Messages[0].Content, "café") {
		t.Fatal("non-NFC provider input")
	}
	state := h.last(t).State.Session
	if state.PendingQuestion == nil || state.PendingQuestion.Slot != "situation" || state.PendingQuestion.Text != "What happened next?" || state.ConsecutiveQuestions != 1 {
		t.Fatal("validator question metadata lost", state)
	}
}
