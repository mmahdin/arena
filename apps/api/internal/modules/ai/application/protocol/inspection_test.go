package protocol

import (
	"context"
	"encoding/json"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"os"
	"path/filepath"
	"testing"
)

func TestInspectorProtocolEligibilityAndActiveTransition(t *testing.T) {
	f := setup(t)
	dir := filepath.Join(t.TempDir(), "private")
	ins, err := trace.NewInspector(trace.InspectorOptions{Directory: dir, Environment: "local", Address: "localhost:8080"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, root := ins.StartTurn(context.Background(), c.TurnRequest{Text: f.w.Turn.Text})
	if _, err = f.e.ShortlistWithContext(ctx, ShortlistContext{WCtx: f.w, Thread: &f.t, User: f.u, Readiness: ReadinessHigh, FlagsVerified: true, Now: f.now}); err != nil {
		t.Fatal(err)
	}
	f.active()
	f.in.Plan = c.ResponsePlan{Protocol: directive(c.PAAdapt, c.OutStuck, nil)}
	spec, patch, err := f.e.ResolveTurn(ctx, f.in)
	if err != nil {
		t.Fatal(err)
	}
	root.End(nil, nil)
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	raw, _ := os.ReadFile(files[0])
	if err = trace.ValidateJSON(raw); err != nil {
		t.Fatal(err)
	}
	var d trace.Document
	_ = json.Unmarshal(raw, &d)
	found := false
	for _, n := range d.Nodes {
		if n.Operation == "application/protocol/engine.ResolveTurn" {
			var actual struct {
				Spec  c.ProtocolStepSpec `json:"stepResult"`
				Patch c.ProtocolPatch    `json:"patchResult"`
			}
			_ = json.Unmarshal(n.Output, &actual)
			if actual.Spec.Instruction == spec.Instruction && actual.Patch.StuckInc == patch.StuckInc {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("actual protocol transition missing")
	}
}
