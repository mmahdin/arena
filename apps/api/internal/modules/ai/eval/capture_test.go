package eval

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type stubGateway struct{ adm c.Admission }

func (s stubGateway) Admit(context.Context, c.TurnRequest) (c.Admission, error)     { return s.adm, nil }
func (s stubGateway) Release(context.Context, c.Admission, ports.TurnOutcome) error { return nil }

type stubBrain struct{}

func (stubBrain) WP10()                                     {}
func (stubBrain) Reference() c.BundleRef                    { return c.BundleRef{Version: "synthetic"} }
func (stubBrain) MaxHops() int                              { return 1 }
func (stubBrain) MaxRetrieved() int                         { return 5 }
func (stubBrain) RetrievalTimeout() time.Duration           { return time.Second }
func (stubBrain) BrainTimeout() time.Duration               { return time.Second }
func (stubBrain) Fallback(c.WorkingContext) c.BrainDecision { return c.BrainDecision{} }
func (stubBrain) Decide(context.Context, ports.BrainInput) (ports.BrainOutput, error) {
	return ports.BrainOutput{Decision: c.BrainDecision{Decision: c.Decision{Primary: c.ObjReflect}}, Raw: []byte("SECRET PRIVATE")}, nil
}

type stubGenerator struct{}

func (stubGenerator) WP11()                      {}
func (stubGenerator) Reference() c.BundleRef     { return c.BundleRef{Version: "synthetic"} }
func (stubGenerator) TurnTimeout() time.Duration { return time.Second }
func (stubGenerator) Generate(context.Context, ports.GenInput, ports.TokenSink) (ports.GenOutput, error) {
	return ports.GenOutput{Text: "SECRET DRAFT"}, nil
}

type stubValidator struct{}

func (stubValidator) StreamGuard(c.ResponsePlan, c.RiskLevel) ports.StreamGuard { return nil }
func (stubValidator) Finalize(context.Context, string, c.ResponsePlan, c.WorkingContext, c.RiskLevel, *c.CrisisPlan) (c.ValidationReport, error) {
	return c.ValidationReport{Passed: true, Action: c.VAAccept}, nil
}

type stubFinalizer struct{ fail bool }

func (s stubFinalizer) Complete(_ context.Context, a c.Admission, p c.PreparedTurn) (c.TurnResult, error) {
	if s.fail {
		return c.TurnResult{}, errors.New("synthetic error")
	}
	return c.TurnResult{TurnID: a.TurnID, Status: "completed"}, nil
}
func (s stubFinalizer) Fail(context.Context, c.Admission, c.Code) error { return nil }

type stubRetriever struct{ id c.ID }

func (s stubRetriever) Retrieve(context.Context, c.RetrievalRequest) (c.RetrievalResult, error) {
	return c.RetrievalResult{Items: []c.MemoryView{{ID: s.id, Status: c.MemActive}}}, nil
}
func TestCaptureContractsWithoutOrchestratorMutation(t *testing.T) {
	ctx := context.Background()
	user := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	turn := uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-000000000001")
	admission := c.Admission{UserID: user, TurnID: turn, LeaseToken: c.NewID()}
	capture := NewCapture()
	gw := CaptureGateway{Next: stubGateway{adm: admission}, Capture: capture}
	if _, err := gw.Admit(ctx, c.TurnRequest{UserID: user}); err != nil {
		t.Fatal(err)
	}
	wc := c.WorkingContext{Turn: c.TurnView{TurnID: turn}}
	b := CaptureBrain{Next: stubBrain{}, Capture: capture}
	if _, err := b.Decide(ctx, ports.BrainInput{WCtx: wc}); err != nil {
		t.Fatal(err)
	}
	g := CaptureGenerator{Next: stubGenerator{}, Capture: capture}
	if _, err := g.Generate(ctx, ports.GenInput{Turn: wc.Turn}, nil); err != nil {
		t.Fatal(err)
	}
	v := CaptureValidator{Next: stubValidator{}, Capture: capture}
	if _, err := v.Finalize(ctx, "SECRET DRAFT", c.ResponsePlan{}, wc, c.RiskLow, nil); err != nil {
		t.Fatal(err)
	}
	finalizer := CaptureFinalizer{Next: stubFinalizer{}, Capture: capture}
	patch := c.StateDelta{Session: c.SessionPatch{LastObjective: c.ObjReflect}}
	if _, err := finalizer.Complete(ctx, admission, c.PreparedTurn{Route: c.RouteNormal, State: patch}); err != nil {
		t.Fatal(err)
	}
	o, ok := capture.Snapshot(turn)
	if !ok || o.Brain == nil || o.Generator == nil || o.Validator == nil || o.State == nil || o.Route != "normal" {
		t.Fatal("capture missed actual outcomes")
	}
	report := Runner{Executor: ExecuteFunc(func(context.Context, *Environment, SequenceTurn) (Observation, error) { return o, nil })}.Run(ctx, []Scenario{golden(t, "anaphora")})
	bts, _ := EncodeReport(report)
	if strings.Contains(string(bts), "SECRET DRAFT") || strings.Contains(string(bts), "SECRET PRIVATE") {
		t.Fatal("capture leaked content in report")
	}
}
func TestCaptureRetrieverMustVerifyOwnership(t *testing.T) {
	ctx := context.Background()
	user := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	turn := uuid.MustParse("aaaaaaaa-aaaa-4aaa-8aaa-000000000001")
	item := uuid.MustParse("33333333-3333-4333-8333-333333333333")
	cap := NewCapture()
	cap.begin(user, turn)
	adapter := CaptureRetriever{Next: stubRetriever{id: item}, Capture: cap}
	if _, err := adapter.Retrieve(ctx, c.RetrievalRequest{UserID: user, Turn: c.TurnView{TurnID: turn}}); err != nil {
		t.Fatal(err)
	}
	out, _ := cap.Snapshot(turn)
	if len(out.Retrieval) != 1 || out.Retrieval[0].UserID != "" || validOwnerScope(out, user.String()) {
		t.Fatal("unverified owner accepted")
	}
	adapter.VerifyOwner = func(_ context.Context, actual c.ID, ids []c.ID) error {
		if actual != user || len(ids) != 1 || ids[0] != item {
			return ErrExecution
		}
		return nil
	}
	if _, err := adapter.Retrieve(ctx, c.RetrievalRequest{UserID: user, Turn: c.TurnView{TurnID: turn}}); err != nil {
		t.Fatal(err)
	}
	out, _ = cap.Snapshot(turn)
	if out.RetrievalStatusesObserved {
		t.Fatal("later owner check must not erase earlier unverified evidence")
	}
}
func TestCaptureFailedFinalizationDoesNotClaimSuccess(t *testing.T) {
	user := c.NewID()
	turn := c.NewID()
	cap := NewCapture()
	cap.begin(user, turn)
	w := CaptureFinalizer{Next: stubFinalizer{fail: true}, Capture: cap}
	if _, err := w.Complete(context.Background(), c.Admission{UserID: user, TurnID: turn}, c.PreparedTurn{Route: c.RouteNormal}); err == nil {
		t.Fatal("expected error")
	}
	got, _ := cap.Snapshot(turn)
	if got.State != nil || got.Route != "" {
		t.Fatal("failed transaction was treated as committed")
	}
}

func TestCaptureReplayAndQueuedAdmissionPreserveCommittedObservation(t *testing.T) {
	ctx := context.Background()
	user, turn := c.NewID(), c.NewID()
	capture := NewCapture()
	capture.begin(user, turn)
	capture.record(turn, func(o *Observation) { o.Route = string(c.RouteNormal) })
	for _, admission := range []c.Admission{
		{UserID: user, TurnID: turn}, // completed replay has no execution lease
		{UserID: user, TurnID: turn, LeaseToken: c.NewID(), Queued: true},
	} {
		gateway := CaptureGateway{Next: stubGateway{adm: admission}, Capture: capture}
		if _, err := gateway.Admit(ctx, c.TurnRequest{UserID: user}); err != nil {
			t.Fatal(err)
		}
		got, ok := capture.Snapshot(turn)
		if !ok || got.Route != string(c.RouteNormal) {
			t.Fatal("replay erased the actual committed observation")
		}
	}
}

type budgetGenerator struct{ stubGenerator }

func (budgetGenerator) MaxSyncCalls() int                { return 4 }
func (budgetGenerator) MaxRegenerations() int            { return 1 }
func (budgetGenerator) GenerationTimeout() time.Duration { return 2 * time.Second }
func (budgetGenerator) OutagePlan(context.Context, string) (c.CrisisPlan, error) {
	return c.CrisisPlan{TemplateFamily: "synthetic_outage"}, nil
}

func TestCapturePreservesGeneratorBudgetsAndFallback(t *testing.T) {
	wrapper := CaptureGenerator{Next: budgetGenerator{}}
	plan, err := wrapper.OutagePlan(context.Background(), "en")
	if err != nil || plan.TemplateFamily != "synthetic_outage" || wrapper.MaxSyncCalls() != 4 || wrapper.MaxRegenerations() != 1 || wrapper.GenerationTimeout() != 2*time.Second {
		t.Fatal("instrumentation changed the generation contract")
	}
	wrapper.Next = stubGenerator{}
	if _, err = wrapper.OutagePlan(context.Background(), "en"); !errors.Is(err, ErrExecution) {
		t.Fatal("missing capability did not fail closed")
	}
	if wrapper.MaxSyncCalls() != 0 || wrapper.MaxRegenerations() != 0 || wrapper.GenerationTimeout() != 0 {
		t.Fatal("instrumentation invented provider budgets")
	}
}
