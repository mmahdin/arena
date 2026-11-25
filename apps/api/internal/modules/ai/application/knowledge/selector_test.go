package knowledge

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/llmhttp"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type scripted struct {
	calls int
	raw   string
	err   error
	last  ports.StructuredRequest
}

func (f *scripted) Structured(ctx context.Context, r ports.StructuredRequest) (ports.StructuredResponse, error) {
	f.calls++
	f.last = r
	return ports.StructuredResponse{JSON: []byte(f.raw)}, f.err
}
func (*scripted) Stream(context.Context, ports.StreamRequest) (ports.Stream, error) {
	panic("JEV cannot generate a reply")
}
func setup(t *testing.T) (*Selector, *scripted, Input) {
	t.Helper()
	b, report, e := bundle.Read(context.Background(), "../../../../../../../knowledge", bundle.Options{})
	if e != nil {
		t.Fatal(e, report.Errors)
	}
	f := &scripted{raw: `{"schema_version":"jev_selection.v1","selected_resource_ids":["skill:reflect_emotion"],"topic_continues":true}`}
	s, e := New(context.Background(), f, b, Options{})
	if e != nil {
		t.Fatal(e)
	}
	return s, f, Input{UserID: c.NewID(), Bundle: b.Reference(), Context: c.WorkingContext{Turn: c.TurnView{TurnID: c.NewID(), MessageID: c.NewID(), Text: "Synthetic relationship concern"}}}
}
func TestJEVSeesCardsAndLoadsOnlySelectedContent(t *testing.T) {
	s, f, in := setup(t)
	out, e := s.Select(context.Background(), in)
	if e != nil || out.Degraded || len(out.Resources) != 1 || out.Resources[0].ContentID != "skills/reflect_emotion.yaml" || f.calls != 1 || f.last.Tier != ports.ComponentJEV || f.last.MaxAttempts != 1 {
		t.Fatal(out, e)
	}
	if len(s.Catalog()) != 28 {
		t.Fatal("pre-existing psychological resources lost", len(s.Catalog()))
	}
	for _, bad := range []string{"card_text", "on_success", "supportive-v", "safety_policy"} {
		if strings.Contains(f.last.SystemOne.State, bad) {
			t.Fatal("full knowledge or policies disclosed to JEV", bad)
		}
	}
}
func TestInvalidJEVOutputsDegradeWithoutAdditionalCalls(t *testing.T) {
	for _, raw := range []string{`{`, `{"schema_version":"jev_selection.v1","selected_resource_ids":["unknown"],"topic_continues":true}`, `{"schema_version":"jev_selection.v1","selected_resource_ids":[],"topic_continues":true,"state":"write"}`, `{"schema_version":"jev_selection.v1","selected_resource_ids":[],"topic_continues":true,"topic_continues":false}`} {
		s, f, in := setup(t)
		f.raw = raw
		in.Context.FutureHints = []c.FutureHint{{ResourceIDs: []string{"skill:reflect_emotion"}, RemainingTurns: 1, ExpiresAt: time.Now().Add(time.Hour)}}
		out, e := s.Select(context.Background(), in)
		if e != nil || !out.Degraded || len(out.Resources) != 1 || f.calls != 1 {
			t.Fatal(out, e)
		}
	}
}

type deniedVerifier struct{}

func (deniedVerifier) VerifyContext(context.Context, c.ID, c.ID, c.WorkingContext) error {
	return errors.New("source denied")
}
func TestJEVNeverReceivesUnverifiedPrivateSources(t *testing.T) {
	s, f, in := setup(t)
	s.opt.Verifier = deniedVerifier{}
	if _, e := s.Select(context.Background(), in); e == nil || f.calls != 0 {
		t.Fatal("denied context sent to JEV")
	}
}
func TestCarryOverExpiresAndTopicScopeDoesNotBecomeAuthority(t *testing.T) {
	now := time.Now()
	thread, other := c.NewID(), c.NewID()
	hints := []c.FutureHint{{Topic: "last useful turn", RemainingTurns: 1, ExpiresAt: now.Add(time.Hour), ThreadID: &thread}, {Topic: "expired", RemainingTurns: 3, ExpiresAt: now.Add(-time.Second)}, {Topic: "old thread", RemainingTurns: 3, ExpiresAt: now.Add(time.Hour), ThreadID: &other}, {Topic: "used up", RemainingTurns: 0, ExpiresAt: now.Add(time.Hour)}}
	got := Decay(hints, &thread, now)
	if len(got) != 1 || got[0].RemainingTurns != 0 || len(Decay(got, &thread, now)) != 0 {
		t.Fatal(got)
	}
}

func TestJEVNativeQuestionsAreBoundedAndCatalogScoped(t *testing.T) {
	s, f, in := setup(t)
	if _, e := s.Select(context.Background(), in); e != nil {
		t.Fatal(e)
	}
	req := f.last
	if req.SystemOne == nil || len(req.SystemOne.Questions) != len(s.Catalog())+1 || len(req.System) != 0 || len(req.Messages) != 0 {
		t.Fatal("JEV still uses text generation")
	}
	if !strings.Contains(req.SystemOne.State, in.Context.Turn.Text) {
		t.Fatal("native state lost working context")
	}
	var shared struct {
		Guidance string           `json:"selection_guidance"`
		Context  c.WorkingContext `json:"working_context"`
	}
	if json.Unmarshal([]byte(req.SystemOne.State), &shared) != nil || shared.Guidance != s.system || len(shared.Context.FutureHints) != 0 {
		t.Fatal("shared selection guidance or carry-over separation lost")
	}
	for _, card := range s.Catalog() {
		q, ok := req.SystemOne.Questions["resource:"+card.ID]
		raw, _ := json.Marshal(card)
		if !ok || q.Type != "noul" || !strings.Contains(q.Instructions, card.ID) || !strings.Contains(req.SystemOne.State, string(raw)) {
			t.Fatal("card decision missing", card.ID)
		}
		if strings.Contains(q.Instructions, s.system) || strings.Contains(q.Instructions, string(raw)) {
			t.Fatal("question repeats shared guidance or card", card.ID)
		}
		for _, bad := range []string{"Return only jev_selection", "on_success", "safety_policy"} {
			if strings.Contains(q.Instructions, bad) {
				t.Fatal("incorrect native instructions", bad)
			}
		}
	}
	s, f, in = setup(t)
	s.opt.MaxInputTokens = 1000
	out, e := s.Select(context.Background(), in)
	if e != nil || !out.Degraded || out.Reason != "jev_input_budget" || f.calls != 0 {
		t.Fatal("native questions escaped input budget", out, e)
	}
}

func TestNativeJEVPreservesSelectionAndOutageFallback(t *testing.T) {
	for _, scenario := range []string{"selected", "none", "invalid", "outage"} {
		t.Run(scenario, func(t *testing.T) {
			s, _, in := setup(t)
			in.Context.FutureHints = []c.FutureHint{{ResourceIDs: []string{"skill:reflect_emotion"}, RemainingTurns: 1, ExpiresAt: time.Now().Add(time.Hour)}}
			var calls atomic.Int32
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Path != "/systemone" {
					t.Error("incorrect native endpoint")
				}
				if scenario == "outage" {
					w.WriteHeader(503)
					return
				}
				var req ports.SystemOneRequest
				if json.NewDecoder(r.Body).Decode(&req) != nil || len(req.Questions) != len(s.cards)+1 {
					t.Error("native questions lost catalog")
				}
				answers := map[string]any{}
				for key := range req.Questions {
					probability := 0.1
					if scenario != "none" && (key == "resource:skill:reflect_emotion" || key == "topic_continues") {
						probability = .95
					}
					answers[key] = map[string]any{"type": "noul", "noul": probability}
				}
				if scenario == "invalid" {
					delete(answers, "topic_continues")
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"model": "jev-1.13", "answers": answers, "usage": map[string]int{"input_tokens": 300, "output_tokens": 29}})
			}))
			defer server.Close()
			provider, e := llmhttp.NewSystemOne(server.URL, "synthetic-only-key", server.Client())
			if e != nil {
				t.Fatal(e)
			}
			cfg := llm.Config{Chains: map[ports.ModelTier][]llm.Model{}, DefaultTimeout: time.Second, MaxRetriesPerModel: 1, BreakerWindow: time.Minute, BreakerCooldown: time.Second, BreakerErrorRate: .5, BreakerMinSamples: 2}
			for _, tier := range []ports.ModelTier{ports.ComponentJEV, ports.ComponentUnified, ports.ComponentMemoryWriter, ports.ComponentReflection, ports.ComponentEmbedding} {
				model := llm.Model{Provider: "other", Name: "unused"}
				if tier == ports.ComponentJEV {
					model = llm.Model{Provider: "OpenCode", Name: "jev-1.13-free", BillingSource: "free"}
				}
				if tier == ports.ComponentEmbedding {
					model.Dimensions = 3
				}
				cfg.Chains[tier] = []llm.Model{model}
			}
			client, e := llm.New(cfg, map[string]llm.Provider{"OpenCode": provider, "other": provider}, llm.Options{})
			if e != nil {
				t.Fatal(e)
			}
			s.client = client
			result, e := s.Select(context.Background(), in)
			if e != nil || calls.Load() != 1 {
				t.Fatal("native JEV must make exactly one call", e, calls.Load())
			}
			if result.Degraded != (scenario == "invalid" || scenario == "outage") {
				t.Fatal(result)
			}
			if scenario == "none" {
				if len(result.Resources) != 0 || result.TopicContinues {
					t.Fatal(result)
				}
			} else {
				if len(result.Resources) != 1 || result.Resources[0].ContentID != "skills/reflect_emotion.yaml" {
					t.Fatal("downstream resource contract changed", result)
				}
			}
			if scenario != "outage" && (result.Usage.InTokens != 300 || result.Usage.Attempts != 1 || result.Usage.CostMicroUSD != 0) {
				t.Fatal("native usage lost", result.Usage)
			}
		})
	}
}
