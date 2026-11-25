package bundle_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/knowledge"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/protocol"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/reflection"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle/testfixture"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/eval"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/safety"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/validator"
)

// Runs against the real pinned bundle and module loaders, without live users,
// remote model calls, databases, approvals or release-signing overrides.
func TestPsychologyDraftLoadsInEveryRuntimeModule(t *testing.T) {
	ctx := context.Background()
	root := filepath.Join("..", "..", "..", "..", "..", "..", "knowledge")
	pinned, report, err := bundle.Read(ctx, root, bundle.Options{CoreVersion: "1.0.0"})
	if err != nil || !report.OK() {
		t.Fatalf("bundle must lint/load: %v, %+v", err, report.Errors)
	}
	if pinned.Reference().Version != "kb@0.1.0" || !pinned.SyntheticOnly() {
		t.Fatal("unreviewed version must remain synthetic-only")
	}
	names := pinned.ArtifactNames()
	count := func(prefix string) int {
		n := 0
		for _, name := range names {
			if strings.HasPrefix(name, prefix) {
				n++
			}
		}
		return n
	}
	if count("skills/") != 18 || count("protocols/") != 7 {
		t.Fatalf("unexpected catalog: %d skills, %d protocols", count("skills/"), count("protocols/"))
	}
	goldens, err := eval.Files(filepath.Join(root, "evals", "golden"))
	if err != nil || len(goldens) < 28 {
		t.Fatalf("expected baseline plus multi-turn product scenarios: %d %v", len(goldens), err)
	}
	for _, file := range goldens {
		if _, err := eval.LoadFile(file); err != nil {
			t.Fatalf("invalid evaluation candidate %s: %v", filepath.Base(file), err)
		}
	}
	withRuntime := testfixture.WithRuntimeSettings(t, pinned)
	if _, _, err := knowledge.Load(ctx, withRuntime); err != nil {
		t.Fatal(err)
	}
	fixedNow := func() time.Time { return time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC) }
	if _, err := protocol.Load(ctx, pinned, protocol.LoadOptions{AllowSyntheticForTests: true}); err != nil {
		t.Fatalf("protocol: %v", err)
	}
	if _, err := memory.Load(ctx, withRuntime, memory.LoadOptions{AllowSyntheticForTests: true}); err != nil {
		t.Fatalf("memory: %v", err)
	}
	if _, err := reflection.Load(ctx, withRuntime, reflection.LoadOptions{AllowSyntheticForTests: true}); err != nil {
		t.Fatalf("reflection: %v", err)
	}
	if _, err := validator.New(ctx, pinned, validator.Options{AllowSyntheticForTests: true}); err != nil {
		t.Fatalf("validator: %v", err)
	}
	assets, err := safety.Load(ctx, pinned, safety.LoadOptions{AllowSyntheticForTests: true})
	if err != nil {
		t.Fatalf("safety: %v", err)
	}
	planner, err := safety.NewPlanner(assets, fixedNow)
	if err != nil {
		t.Fatalf("crisis planner: %v", err)
	}
	plan, err := planner.Plan(ctx, contract.SafetySignal{Level: contract.RiskHigh, Categories: []contract.RiskCategory{contract.RiskSuicidalIdeation}}, "en", contract.RiskState{Level: contract.RiskHigh})
	if err != nil || !plan.Buffered || plan.TemplateFamily != "immediate_safety" || len(plan.Resources) == 0 {
		t.Fatalf("static crisis plan missing: %v %+v", err, plan)
	}
	text, ok := safety.StaticCrisisFallback(plan)
	if !ok || !strings.Contains(text, "https://findahelpline.com") {
		t.Fatal("static crisis fallback must use pinned directory")
	}
	if _, err := protocol.Load(ctx, pinned, protocol.LoadOptions{}); err == nil {
		t.Fatal("production default accepted unreviewed protocols")
	}
	if _, err := validator.New(ctx, pinned, validator.Options{}); err == nil {
		t.Fatal("production default accepted unreviewed validator")
	}
}
