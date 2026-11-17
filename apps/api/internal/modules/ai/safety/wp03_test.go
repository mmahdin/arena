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

// Runs with REAL Go dependencies in CI. Confirms the supplied WP-03 bundle
// remains synthetic/inert: it may be parsed, but never acts as a valid L0 and
// never supplies a crisis plan or reply. No WP-03 files are modified.
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
	sig, err := gate.Assess(context.Background(), ports.SafetyInput{Text: "synthetic only", Lang: "en"})
	if !errors.Is(err, ErrLocale) || sig.Level != contract.RiskElevated || !sig.Degraded {
		t.Fatalf("empty lexicon claimed safe: %+v %v", sig, err)
	}
	planner, _ := NewPlanner(a, nil)
	plan, err := planner.Plan(context.Background(), contract.SafetySignal{Level: contract.RiskHigh, Categories: []contract.RiskCategory{contract.RiskOther}}, "en", contract.RiskState{})
	if !errors.Is(err, ErrCrisis) || plan.TemplateText != "" || len(plan.Resources) != 0 {
		t.Fatalf("WP-03 synthetic resources accidentally served: %+v %v", plan, err)
	}
	if _, ok := a.FallbackFunc(contract.RouteNormal, contract.ObjReflect, "en"); ok {
		t.Fatal("synthetic fallback text was served")
	}
}
