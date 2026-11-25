package unified

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle/testfixture"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"strings"
	"testing"
)

type client struct {
	calls int
	fn    func(ports.StructuredRequest) ([]byte, error)
	last  ports.StructuredRequest
}

func (f *client) Structured(_ context.Context, r ports.StructuredRequest) (ports.StructuredResponse, error) {
	f.calls++
	f.last = r
	raw, e := f.fn(r)
	return ports.StructuredResponse{JSON: raw, Usage: c.LLMUsage{Attempts: 1}}, e
}
func (*client) Stream(context.Context, ports.StreamRequest) (ports.Stream, error) {
	panic("unified must never stream a second model")
}
func sample() c.UnifiedTurn {
	return c.UnifiedTurn{SchemaVersion: c.UnifiedSchemaVersion, Reply: "That fictional meeting sounded tiring.", Conversation: c.ConversationIntent{Intent: c.IntentDiscloseEvent, Objective: c.ObjReflect}, MemorySignals: []string{}, UsedContext: []c.ID{}}
}
func setup(t *testing.T) (*Service, *client, ports.UnifiedInput) {
	t.Helper()
	b, report, e := bundle.Read(context.Background(), "../../../../../../../knowledge", bundle.Options{})
	if e != nil {
		t.Fatal(e, report.Errors)
	}
	overlay := testfixture.WithRuntimeSettings(t, b)
	f := &client{}
	f.fn = func(ports.StructuredRequest) ([]byte, error) { return json.Marshal(sample()) }
	s, e := New(context.Background(), f, overlay, nil, Options{AllowSyntheticForTests: true, MaxInputTokens: 18000})
	if e != nil {
		t.Fatal(e)
	}
	w := c.WorkingContext{SchemaVersion: "wctx.v2", Turn: c.TurnView{TurnID: c.NewID(), MessageID: c.NewID(), Text: "A fictional meeting was tiring.", Lang: "en"}, Session: c.SessionView{SessionID: c.NewID()}, Prefs: c.PrefsView{Language: "en"}, Risk: c.RiskView{Current: c.RiskNone}}
	return s, f, ports.UnifiedInput{UserID: c.NewID(), Context: w, Bundle: b.Reference()}
}
func TestSingleStructuredCallUsesCompleteContextAndSavedLanguage(t *testing.T) {
	for _, lang := range []string{"fa", "en"} {
		t.Run(lang, func(t *testing.T) {
			s, f, in := setup(t)
			in.Context.Prefs.Language = lang
			in.Context.Turn.Text = "می‌خواهم درباره همان جلسهٔ ساختگی حرف بزنم"
			id := c.NewID()
			in.Context.Persistent.Items = []c.MemoryView{{ID: id, Text: "The user prefers concise replies", Kind: c.MemPreference, Epistemic: c.EpReported, Status: c.MemActive}}
			f.fn = func(r ports.StructuredRequest) ([]byte, error) {
				var got c.WorkingContext
				if json.Unmarshal([]byte(r.Messages[0].Content), &got) != nil || got.Turn.Text != in.Context.Turn.Text || got.Prefs.Language != lang || len(got.Persistent.Items) != 1 {
					t.Fatal("complete context or language lost")
				}
				u := sample()
				if lang == "fa" {
					u.Reply = "آن جلسهٔ ساختگی خسته‌کننده به نظر می‌رسد."
				}
				u.UsedContext = []c.ID{id}
				return json.Marshal(u)
			}
			out, e := s.Infer(context.Background(), in)
			if e != nil || out.Turn.Reply == "" || f.calls != 1 || f.last.Tier != ports.ComponentUnified || f.last.MaxAttempts != 1 {
				t.Fatal(out, e, f.calls)
			}
			t.Logf("synthetic %s rendered input estimate=%d", lang, promptSize(f.last))
		})
	}
}
func promptSize(r ports.StructuredRequest) int {
	n := 0
	for _, b := range r.System {
		n += (len(b.Text) + 3) / 4
	}
	for _, m := range r.Messages {
		n += (len(m.Content) + 3) / 4
	}
	return n
}
func TestInvalidOutputAndProviderFailureNeverRetry(t *testing.T) {
	for _, mode := range []string{"malformed", "unknown", "duplicate", "missing_reply", "bad_reference", "provider", "wrong_version"} {
		t.Run(mode, func(t *testing.T) {
			s, f, in := setup(t)
			f.fn = func(ports.StructuredRequest) ([]byte, error) {
				u := sample()
				switch mode {
				case "malformed":
					return []byte(`{"reply":`), nil
				case "unknown":
					raw, _ := json.Marshal(u)
					return []byte(strings.Replace(string(raw), `"reply":`, `"secret":true,"reply":`, 1)), nil
				case "duplicate":
					raw, _ := json.Marshal(u)
					return []byte(strings.Replace(string(raw), `"reply":`, `"reply":"duplicate","reply":`, 1)), nil
				case "missing_reply":
					u.Reply = ""
				case "bad_reference":
					u.UsedContext = []c.ID{c.NewID()}
				case "provider":
					return nil, errors.New("synthetic outage")
				case "wrong_version":
					u.SchemaVersion = "unified_turn.v0"
				}
				return json.Marshal(u)
			}
			if _, e := s.Infer(context.Background(), in); e == nil || f.calls != 1 {
				t.Fatal(mode, e, f.calls)
			}
		})
	}
}
func TestProposalEvidenceAndThreadReferences(t *testing.T) {
	s, f, in := setup(t)
	f.fn = func(ports.StructuredRequest) ([]byte, error) {
		u := sample()
		topic := "fictional meeting"
		u.Thread = &c.ThreadIntent{Relation: c.RelNew, Topic: &topic, Slots: []c.SlotUpdate{{Slot: "situation", Value: "fictional meeting", Status: c.EpReported, Evidence: []c.ID{in.Context.Turn.MessageID}}}, Hypotheses: []c.HypothesisSignal{}}
		return json.Marshal(u)
	}
	if _, e := s.Infer(context.Background(), in); e != nil {
		t.Fatal(e)
	}
	f.fn = func(ports.StructuredRequest) ([]byte, error) {
		u := sample()
		target := c.NewID()
		u.Thread = &c.ThreadIntent{Relation: c.RelReturns, Target: &target, Slots: []c.SlotUpdate{}, Hypotheses: []c.HypothesisSignal{}}
		return json.Marshal(u)
	}
	out, e := s.Infer(context.Background(), in)
	if e != nil || ValidateReferences(out.Turn, in.Context) == nil {
		t.Fatal("independent invalid thread was authorized", e)
	}
}
