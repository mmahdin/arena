package safety

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// The v0.1.0 synthetic bundle contains development-only L0 rules and
// template resources. It must remain unavailable through production loading,
// but local engineering can exercise those pinned assets explicitly.
func TestWP03RealSyntheticBundleFailsClosed(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "..", "..", "knowledge")
	// Offline verification may use an equivalent YAML-to-JSON converted copy.
	// Normal CI always reads the actual supplied WP-03 knowledge files.
	if alternate := os.Getenv("WP06_TEST_KNOWLEDGE_ROOT"); alternate != "" {
		root = alternate
	}
	b, report, err := bundle.Read(context.Background(), root, bundle.Options{CoreVersion: "1.0.0"})
	if err != nil || !report.OK() {
		t.Fatalf("WP-03 fixture no longer valid: %v %+v", err, report.Errors)
	}
	if _, err := Load(context.Background(), b, LoadOptions{}); !errors.Is(err, ErrAssets) {
		t.Fatalf("synthetic bundle allowed for real users: %v", err)
	}
	a, err := Load(context.Background(), b, LoadOptions{AllowSyntheticForTests: true})
	if err != nil {
		t.Fatalf("synthetic engineering load: %v", err)
	}
	gate, _ := NewGate(a, lowClassifier(), nil)
	sig, err := gate.Assess(context.Background(), ports.SafetyInput{Text: "I want to die", Lang: "en"})
	if err != nil || sig.Level != contract.RiskHigh || sig.Degraded || len(sig.Triggers) == 0 {
		t.Fatalf("v0.1.0 L0 did not preserve a high-risk signal: %+v %v", sig, err)
	}
	unavailable, err := gate.Assess(context.Background(), ports.SafetyInput{Text: "synthetic only", Lang: "de"})
	if !errors.Is(err, ErrLocale) || unavailable.Level != contract.RiskElevated || !unavailable.Degraded {
		t.Fatalf("unknown locale did not fail closed: %+v %v", unavailable, err)
	}
	planner, err := NewPlanner(a, nil)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := planner.Plan(context.Background(), contract.SafetySignal{Level: contract.RiskHigh, Categories: []contract.RiskCategory{contract.RiskOther}}, "en", contract.RiskState{})
	if err != nil || plan.TemplateText == "" || len(plan.Resources) == 0 {
		t.Fatalf("v0.1.0 pinned crisis plan not loaded for local tests: %+v %v", plan, err)
	}
	if text, ok := a.FallbackFunc(contract.RouteNormal, contract.ObjReflect, "en"); !ok || text == "" {
		t.Fatal("v0.1.0 pinned fallback not available to local tests")
	}
}

// A corrupted/empty L0 lexicon still fails closed, even with an opt-in
// synthetic development bundle. Never infer a negative safety assessment from
// an empty rule list.
type emptyLexiconBundle struct{ ports.Bundle }

func (b emptyLexiconBundle) SyntheticOnly() bool { return true }
func (b emptyLexiconBundle) Artifact(ctx context.Context, name string) ([]byte, error) {
	if name == "lexicons/safety_l0.en.yaml" {
		return []byte("schema: lexicon.v1\nversion: 0.1.0\nlocale: en\nrules: []\n"), nil
	}
	return b.Bundle.Artifact(ctx, name)
}

func TestLocalDraftEmptyLexiconFailsClosed(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "..", "..", "knowledge")
	b, report, err := bundle.Read(context.Background(), root, bundle.Options{CoreVersion: "1.0.0"})
	if err != nil || !report.OK() {
		t.Fatalf("invalid fixture: %v %+v", err, report.Errors)
	}
	assets, err := Load(context.Background(), emptyLexiconBundle{Bundle: b}, LoadOptions{AllowSyntheticForTests: true})
	if err != nil {
		t.Fatal(err)
	}
	gate, err := NewGate(assets, lowClassifier(), nil)
	if err != nil {
		t.Fatal(err)
	}
	sig, err := gate.Assess(context.Background(), ports.SafetyInput{Text: "any input", Lang: "en"})
	if !errors.Is(err, ErrLocale) || sig.Level != contract.RiskElevated || !sig.Degraded {
		t.Fatalf("empty L0 lexicon accepted: %+v %v", sig, err)
	}
}
