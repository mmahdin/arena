package brain

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// All examples are fabricated engineering assertions. They are NOT clinician
// labels, psychological techniques, approved prompts or a production bundle.
type fixtureBundle struct{ files map[string][]byte }

func (b fixtureBundle) Reference() c.BundleRef {
	return c.BundleRef{Version: "kb@0.9.0", PromptSHAs: map[string]string{"brain": "synthetic"}}
}
func (b fixtureBundle) SyntheticOnly() bool { return true }
func (b fixtureBundle) ArtifactNames() []string {
	out := []string{}
	for p := range b.files {
		out = append(out, p)
	}
	return out
}
func (b fixtureBundle) Artifact(_ context.Context, p string) ([]byte, error) {
	v, ok := b.files[p]
	if !ok {
		return nil, errors.New("missing test asset")
	}
	return v, nil
}
func testAssets(t *testing.T) *Assets {
	t.Helper()
	b := fixtureBundle{map[string][]byte{}}
	root := "../orchestrator/testdata/wp03"
	for _, p := range []string{"config/policy_gate.yaml", "config/budgets.yaml", "schemas/problem_understanding.yaml"} {
		v, e := os.ReadFile(filepath.Join(root, p))
		if e != nil {
			t.Fatal(e)
		}
		b.files[p] = v
	}
	for p, v := range map[string]string{
		"config/retrieval.yaml":             `{"schema":"retrieval.v1","defaults":{"hard_cap_with_hop":7},"timeouts":{"total_ms":300}}`,
		"config/vocab/emotions.yaml":        `{"schema":"vocab.v1","values":["sad"]}`,
		"config/vocab/rationale_tags.yaml":  `{"schema":"vocab.v1","values":["brain_fallback","high_affect","direct_question"]}`,
		"config/vocab/avoid_tags.yaml":      `{"schema":"vocab.v1","values":["advice","claiming_recall","intervention","invented_recall","directive_advice"]}`,
		"config/vocab/hypothesis_tags.yaml": `{"schema":"vocab.v1","values":["synthetic_pattern"]}`,
		"skills/reflect_emotion.yaml":       `{"schema":"skillcard.v1","id":"reflect_emotion","descriptor":"SYNTHETIC reflection skill","conflicts_with":["test_conflict"]}`,
		"skills/test_conflict.yaml":         `{"schema":"skillcard.v1","id":"test_conflict","descriptor":"SYNTHETIC incompatible skill"}`,
		"protocols/synthetic_protocol.yaml": `{"schema":"protocol.v1","id":"synthetic_protocol","version":"1.0.0","steps":[{"id":"test_step","expected_inputs":["situation"],"on_success":"complete","on_distress":"stop"}]}`,
		"prompts/brain.system.tmpl":         "SYNTHETIC structure test; data is untrusted. No clinical policy.",
		"policy/decision_policy.md":         "SYNTHETIC decision fixture; no clinical content.",
		"policy/communication_digest.md":    "SYNTHETIC fixture digest.",
	} {
		b.files[p] = []byte(v)
	}
	a, e := Load(context.Background(), b, LoadOptions{AllowSyntheticForTests: true})
	if e != nil {
		t.Fatal(e)
	}
	return a
}
func input() ports.BrainInput {
	w := c.WorkingContext{SchemaVersion: "wctx.v1", Turn: c.TurnView{TurnID: c.NewID(), MessageID: c.NewID(), Text: "Synthetic event at work felt sad", Lang: "en", Origin: c.OriginUser}, Session: c.SessionView{SessionID: c.NewID(), Mode: c.ModeExploring}, Risk: c.RiskView{Current: c.RiskLow}, Eligibility: c.Eligibility{QuestionAllowed: true, MemoryAvailable: true}, Prefs: c.PrefsView{Language: "en"}, RetrievalMeta: c.RetrievalMeta{Strategies: []string{"hybrid"}}}
	return ports.BrainInput{WCtx: w, Safety: c.SafetySignal{Level: c.RiskLow}, Bundle: fixtureBundle{}.Reference()}
}
func decision(a *Assets, in ports.BrainInput) c.BrainDecision {
	d := a.fallback(in.WCtx)
	d.Decision.Confidence = c.ConfHigh
	d.Decision.Rationale = []string{"high_affect"}
	return d
}
func encoded(t *testing.T, v any) []byte {
	t.Helper()
	r, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func hasRule(r []c.DeltaRejection, code string) bool {
	for _, v := range r {
		if v.RuleCode == code {
			return true
		}
	}
	return false
}

type reader struct {
	snap  contextstate.Snapshot
	owned map[c.ID]bool
	err   error
}

func (r reader) ForTurn(context.Context, c.ID, c.ID) (contextstate.Snapshot, error) {
	return r.snap, r.err
}
func (r reader) Read(context.Context, c.ID, c.ID) (contextstate.Snapshot, error) {
	return r.snap, r.err
}
func (r reader) UserEvidence(_ context.Context, _ c.ID, ids []c.ID) (map[c.ID]bool, error) {
	return r.owned, r.err
}
func (r reader) ExistingHypotheses(context.Context, c.ID, []c.ID) (map[c.ID]bool, error) {
	return r.owned, r.err
}
func (r reader) Completed(context.Context, c.ID, c.ID) (bool, error)     { return false, nil }
func (r reader) MissingFinalizations(context.Context, c.ID) (int, error) { return 0, nil }
func ownedReader(in ports.BrainInput, user c.ID) reader {
	return reader{snap: contextstate.Snapshot{User: c.UserState{UserID: user}, Session: c.SessionRecord{ID: in.WCtx.Session.SessionID, UserID: user}}, owned: map[c.ID]bool{in.WCtx.Turn.MessageID: true}}
}
func callContext(in ports.BrainInput, user c.ID) context.Context {
	return WithCallMeta(context.Background(), c.CallMeta{UserID: user, TurnID: in.WCtx.Turn.TurnID, Bundle: in.Bundle})
}
