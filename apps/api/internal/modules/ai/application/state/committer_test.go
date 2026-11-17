package state

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type fakeReader struct {
	snapshot   contextstate.Snapshot
	evidence   map[c.ID]bool
	hypotheses map[c.ID]bool
	missing    int
	completed  bool
}

func (r *fakeReader) Read(_ context.Context, _, _ c.ID) (contextstate.Snapshot, error) {
	return r.snapshot, nil
}
func (r *fakeReader) ForTurn(_ context.Context, _, _ c.ID) (contextstate.Snapshot, error) {
	return r.snapshot, nil
}
func (r *fakeReader) UserEvidence(_ context.Context, _ c.ID, ids []c.ID) (map[c.ID]bool, error) {
	out := map[c.ID]bool{}
	for _, id := range ids {
		out[id] = r.evidence[id]
	}
	return out, nil
}
func (r *fakeReader) MissingFinalizations(_ context.Context, _ c.ID) (int, error) {
	return r.missing, nil
}
func (r *fakeReader) ExistingHypotheses(_ context.Context, _ c.ID, ids []c.ID) (map[c.ID]bool, error) {
	out := map[c.ID]bool{}
	for _, id := range ids {
		out[id] = r.hypotheses[id]
	}
	return out, nil
}
func (r *fakeReader) Completed(_ context.Context, _, _ c.ID) (bool, error) { return r.completed, nil }

type fixtureBundle struct{ root string }

func (b fixtureBundle) Reference() c.BundleRef { return c.BundleRef{Version: "kb@0.1.0"} }
func (b fixtureBundle) Artifact(_ context.Context, path string) ([]byte, error) {
	return os.ReadFile(filepath.Join(b.root, path))
}
func (b fixtureBundle) SyntheticOnly() bool { return true }

type fakeRegistry struct{ root string }

func (b fakeRegistry) Active(_ context.Context, _ c.ID) (ports.Bundle, error) {
	return fixtureBundle{root: b.root}, nil
}
func fixtureRoot(t *testing.T) string {
	t.Helper()
	dir, _ := os.Getwd()
	for i := 0; i < 12; i++ {
		p := filepath.Join(dir, "knowledge", "bundle.yaml")
		if _, e := os.Stat(p); e == nil {
			return filepath.Dir(p)
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("missing WP03 fixture")
	return ""
}
func testState(t *testing.T) (*Service, *fakeReader, c.StateDelta, time.Time) {
	t.Helper()
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	user, session, conv, turn, focus := c.NewID(), c.NewID(), c.NewID(), c.NewID(), c.NewID()
	reader := &fakeReader{
		snapshot: contextstate.Snapshot{
			Session:         c.SessionRecord{ID: session, UserID: user, ConversationID: conv, Version: 7, Mode: c.ModeExploring, FocusThreadID: &focus},
			User:            c.UserState{UserID: user, Version: 9, TZ: "UTC", Locale: "en", Risk: c.RiskState{Level: c.RiskNone}, ComfortBudget: .5},
			Threads:         []c.ThreadRecord{{ID: focus, UserID: user, Phase: c.PhaseExploring, Slots: map[string]c.SlotView{}, Version: 4, Priority: .5}},
			ThreadUserTurns: map[c.ID]int{focus: 3},
		},
		evidence: map[c.ID]bool{}, hypotheses: map[c.ID]bool{},
	}
	committer, err := New(reader, fakeRegistry{root: fixtureRoot(t)})
	if err != nil {
		t.Fatal(err)
	}
	committer.WithClock(func() time.Time { return now })
	d := c.StateDelta{UserID: user, TurnID: turn, ExpectedVersions: c.Versions{Session: 7, User: 9}, Session: c.SessionPatch{LastObjective: c.ObjReflect, TurnCountInc: 1}}
	return committer, reader, d, now
}
func hasRule(r []c.DeltaRejection, code string) bool {
	for _, v := range r {
		if v.RuleCode == code {
			return true
		}
	}
	return false
}
func TestMinimalWP05DeltaAndIdempotentCopy(t *testing.T) {
	s, _, d, _ := testState(t)
	out, rejected, err := s.ValidateDelta(context.Background(), d)
	if err != nil || len(rejected) != 0 || out.Session.TurnCountInc != 1 || out.ExpectedVersions != d.ExpectedVersions {
		t.Fatalf("simple delta: %+v %+v %v", out, rejected, err)
	}
	if d.Session.LastObjective != c.ObjReflect {
		t.Fatal("input was mutated")
	}
}
func TestVersionsConflictFailsClosed(t *testing.T) {
	s, _, d, _ := testState(t)
	d.ExpectedVersions.User = 8
	_, rej, err := s.ValidateDelta(context.Background(), d)
	var ce *c.Error
	if !errors.As(err, &ce) || ce.Code != c.ErrStateConflict || !hasRule(rej, "ST-001") {
		t.Fatalf("stale state accepted: %v %v", err, rej)
	}
}
func TestThreadPhaseReadinessAndEvidence(t *testing.T) {
	s, reader, d, _ := testState(t)
	focus := reader.snapshot.Threads[0].ID
	evidence := c.NewID()
	reader.evidence[evidence] = true
	phase := c.PhaseUnderstanding
	d.Thread = &c.ThreadPatch{ThreadID: focus, Phase: &phase, Slots: []c.SlotUpdate{
		{Slot: "situation", Value: "synthetic", Status: c.EpReported, Evidence: []c.ID{evidence}},
		{Slot: "emotions", Value: "synthetic", Status: c.EpInferred, Evidence: []c.ID{evidence}},
		{Slot: "goal", Value: "synthetic", Status: c.EpInferred, Evidence: []c.ID{evidence}},
	}}
	d.ExpectedVersions.Thread = 4
	out, rej, err := s.ValidateDelta(context.Background(), d)
	if err != nil || out.Thread.Phase == nil || *out.Thread.Phase != phase || len(out.Thread.Slots) != 3 {
		t.Fatalf("valid readiness rejected: %+v %v %v", out.Thread, rej, err)
	}
	reader.snapshot.ThreadUserTurns[focus] = 2
	out, rej, err = s.ValidateDelta(context.Background(), d)
	if err != nil || out.Thread.Phase != nil || !hasRule(rej, "ST-003") {
		t.Fatalf("premature readiness allowed: %+v %v %v", out.Thread, rej, err)
	}
	reader.snapshot.ThreadUserTurns[focus] = 3
	reader.evidence[evidence] = false
	out, rej, err = s.ValidateDelta(context.Background(), d)
	if err != nil || len(out.Thread.Slots) != 0 || out.Thread.Phase != nil || !hasRule(rej, "ST-004") {
		t.Fatalf("unowned evidence accepted: %+v %+v %v", out.Thread, rej, err)
	}
	reader.evidence[evidence] = true
	invalid := c.PhaseResolved
	d.Thread.Phase = &invalid
	out, rej, err = s.ValidateDelta(context.Background(), d)
	if err != nil || out.Thread.Phase != nil || !hasRule(rej, "ST-002") {
		t.Fatalf("invalid transition allowed: %+v %+v %v", out.Thread, rej, err)
	}
}
func TestUnknownSlotAndHypothesisReferencesRejected(t *testing.T) {
	s, r, d, _ := testState(t)
	evidence := c.NewID()
	r.evidence[evidence] = true
	d.ExpectedVersions.Thread = 4
	d.Thread = &c.ThreadPatch{ThreadID: r.snapshot.Threads[0].ID, Slots: []c.SlotUpdate{{Slot: "not_in_schema", Value: "bad", Status: c.EpReported, Evidence: []c.ID{evidence}}}, HypothesisSignals: []c.HypothesisSignal{{Op: c.HypSupport, ID: &evidence, Evidence: []c.ID{evidence}}}}
	out, rejected, err := s.ValidateDelta(context.Background(), d)
	if err != nil || len(out.Thread.Slots) != 0 || len(out.Thread.HypothesisSignals) != 0 || !hasRule(rejected, "BR-007") || !hasRule(rejected, "ST-007") {
		t.Fatalf("unknown state accepted: %+v %+v %v", out.Thread, rejected, err)
	}
}
func TestRiskLoweringRejectedAndElevationPausesActiveProtocol(t *testing.T) {
	s, r, d, now := testState(t)
	r.snapshot.User.Risk.Level = c.RiskHigh
	d.User.Risk = &c.RiskTransition{At: now, From: c.RiskHigh, To: c.RiskLow, TurnID: d.TurnID}
	out, rej, err := s.ValidateDelta(context.Background(), d)
	if err != nil || out.User.Risk != nil || !hasRule(rej, "ST-010") {
		t.Fatalf("risk lowering accepted: %+v %v %v", out.User, rej, err)
	}
	r.snapshot.User.Risk.Level = c.RiskNone
	pi := c.ProtocolInstance{ID: c.NewID(), UserID: d.UserID, ProtocolID: "placeholder_protocol", ProtocolVersion: "0.1.0", Status: c.PIActive, Version: 3}
	r.snapshot.Protocol = &pi
	d.User.Risk = &c.RiskTransition{At: now, From: c.RiskNone, To: c.RiskHigh, TurnID: d.TurnID}
	out, rej, err = s.ValidateDelta(context.Background(), d)
	if err != nil || out.User.Risk == nil || out.User.Risk.To != c.RiskHigh || out.Protocol == nil || out.Protocol.NewStatus == nil || *out.Protocol.NewStatus != c.PIPaused || out.ExpectedVersions.Protocol != 3 || !hasRule(rej, "ST-012") {
		t.Fatalf("elevation not applied: %+v %+v %v", out, rej, err)
	}
}
func TestProtocolConsentAndGraph(t *testing.T) {
	s, r, d, _ := testState(t)
	pi := c.ProtocolInstance{ID: c.NewID(), UserID: d.UserID, ProtocolID: "placeholder_protocol", ProtocolVersion: "0.1.0", Status: c.PIOffered, Version: 3, StepID: "placeholder_step"}
	r.snapshot.Protocol = &pi
	active := c.PIActive
	badStep := "not_reachable"
	d.ExpectedVersions.Protocol = 3
	d.Protocol = &c.ProtocolPatch{InstanceID: pi.ID, NewStatus: &active, NewStep: &badStep}
	out, rej, err := s.ValidateDelta(context.Background(), d)
	if err != nil || out.Protocol == nil || out.Protocol.NewStatus != nil || out.Protocol.NewStep != nil || !hasRule(rej, "ST-020") || !hasRule(rej, "ST-021") {
		t.Fatalf("invalid protocol transition: %+v %+v %v", out.Protocol, rej, err)
	}
	consent := c.NewID()
	r.evidence[consent] = true
	at := time.Now().UTC()
	d.Protocol.ConsentAt = &at
	d.Protocol.ConsentMessageID = &consent
	d.Protocol.NewStep = nil
	out, _, err = s.ValidateDelta(context.Background(), d)
	if err != nil || out.Protocol.NewStatus == nil || *out.Protocol.NewStatus != c.PIActive {
		t.Fatalf("verified consent rejected: %+v %v", out.Protocol, err)
	}
}
func TestActiveThreadLimitAndFocusOwnership(t *testing.T) {
	s, r, d, now := testState(t)
	for i := 0; i < 2; i++ {
		r.snapshot.Threads = append(r.snapshot.Threads, c.ThreadRecord{ID: c.NewID(), UserID: d.UserID, Phase: c.PhaseWorking, Priority: float32(i+1) * .3, LastActiveAt: now, Version: 2})
	}
	r.snapshot.Threads[0].Priority = .1
	newThread := c.NewID()
	notOwned := c.NewID()
	d.Thread = &c.ThreadPatch{ThreadID: newThread, Create: &c.ThreadInit{Topic: "Synthetic"}}
	d.Session.FocusThread = &notOwned
	out, rej, err := s.ValidateDelta(context.Background(), d)
	if err != nil || out.Thread == nil || out.Session.FocusThread != nil || !hasRule(rej, "ST-006") || !hasRule(rej, "ST-005") || len(out.ThreadOps) != 1 || out.ThreadOps[0].ThreadID != r.snapshot.Threads[1].ID || out.ThreadOps[0].Op != "pause" {
		t.Fatalf("active limit or focus invalid: %+v %+v %v", out, rej, err)
	}
}
func TestOpenLoopDedupeAndFollowupPolicy(t *testing.T) {
	s, r, d, now := testState(t)
	id, ref := c.NewID(), c.NewID()
	r.snapshot.OpenLoops = []c.OpenLoop{{ID: id, Kind: "confirm_fact", Ref: &ref, Text: "old"}}
	d.OpenLoops = []c.OpenLoopOp{{Op: "add", Loop: c.OpenLoop{ID: c.NewID(), Kind: "confirm_fact", Ref: &ref, Text: "new"}}, {Op: "add", Loop: c.OpenLoop{ID: c.NewID(), Kind: "confirm_fact", Text: "okay"}}}
	follow := c.FollowUpOp{Op: "schedule", FollowUpID: c.NewID(), DueAt: now.Add(26 * time.Hour), Reason: "check_in"}
	d.FollowUps = []c.FollowUpOp{follow}
	out, rej, err := s.ValidateDelta(context.Background(), d)
	if err != nil || len(out.OpenLoops) != 1 || len(out.FollowUps) != 0 || !hasRule(rej, "ST-040") || !hasRule(rej, "ST-050") {
		t.Fatalf("loop/opt-in restrictions: %+v %+v %v", out, rej, err)
	}
	r.snapshot.User.ProactiveOptIn = true
	r.snapshot.User.QuietHours = &c.QuietHours{Start: "00:00", End: "00:00"}
	out, rej, err = s.ValidateDelta(context.Background(), d)
	if err != nil || len(out.FollowUps) != 0 || !hasRule(rej, "ST-050") {
		t.Fatalf("quiet-hours restriction: %+v %+v %v", out.FollowUps, rej, err)
	}
	r.snapshot.User.QuietHours = nil
	r.snapshot.FollowUps = []c.FollowUp{{ID: c.NewID(), UserID: d.UserID, DueAt: follow.DueAt.Add(time.Hour), Status: "scheduled"}}
	out, rej, err = s.ValidateDelta(context.Background(), d)
	if err != nil || len(out.FollowUps) != 0 || !hasRule(rej, "ST-050") {
		t.Fatalf("24h restriction: %+v %+v %v", out.FollowUps, rej, err)
	}
	r.snapshot.FollowUps = nil
	out, _, err = s.ValidateDelta(context.Background(), d)
	if err != nil || len(out.FollowUps) != 1 {
		t.Fatalf("opted-in due time rejected: %+v %v", out.FollowUps, err)
	}
}

type fakeFinalizer struct {
	complete int
	failed   int
	received c.PreparedTurn
}

func (f *fakeFinalizer) Complete(_ context.Context, a c.Admission, p c.PreparedTurn) (c.TurnResult, error) {
	f.complete++
	f.received = p
	return c.TurnResult{TurnID: a.TurnID, UserMessageID: a.UserMessageID, Status: c.TurnCompleted}, nil
}
func (f *fakeFinalizer) Fail(_ context.Context, _ c.Admission, _ c.Code) error {
	f.failed++
	return nil
}
func TestWrappedFinalizerValidatesBeforeAtomicWP02Port(t *testing.T) {
	s, r, d, _ := testState(t)
	f := &fakeFinalizer{}
	wrapper, err := NewValidatedFinalizer(f, s)
	if err != nil {
		t.Fatal(err)
	}
	a := c.Admission{TurnID: d.TurnID, UserID: d.UserID, UserMessageID: c.NewID(), LeaseToken: c.NewID()}
	prepared := c.PreparedTurn{TurnID: d.TurnID, UserMessageID: a.UserMessageID, Text: "Synthetic answer?", State: d}
	result, err := wrapper.Complete(context.Background(), a, prepared)
	if err != nil || result.Status != c.TurnCompleted || f.complete != 1 || f.received.State.Session.ConsecutiveQuestions != 1 || f.received.State.Session.PendingQuestion == nil {
		t.Fatalf("finalizer integration: %+v %+v %v", result, f.received.State, err)
	}
	// Original draft is not mutated.
	if prepared.State.Session.PendingQuestion != nil {
		t.Fatal("input state mutated")
	}
	r.snapshot.Session.ConsecutiveQuestions = 2
	if _, err = wrapper.Complete(context.Background(), a, prepared); err == nil || f.complete != 1 {
		t.Fatalf("question limit bypassed, complete=%d", f.complete)
	}
	r.completed = true
	_, err = wrapper.Complete(context.Background(), a, prepared)
	if err != nil || f.complete != 2 || f.received.State.Session.PendingQuestion != nil {
		t.Fatalf("replay not delegated: %+v %v", f.received.State, err)
	}
	r.completed = false
	d.ExpectedVersions.User = 0
	prepared.State = d
	if _, err = wrapper.Complete(context.Background(), a, prepared); err == nil || f.complete != 2 {
		t.Fatal("version conflict reached finalizer")
	}
}
func TestReconcileDoesNotInventMissingHistory(t *testing.T) {
	s, r, d, _ := testState(t)
	report, err := s.Reconcile(context.Background(), d.UserID)
	if err != nil || report.Repaired {
		t.Fatalf("no-op reconcile: %+v %v", report, err)
	}
	r.missing = 2
	report, err = s.Reconcile(context.Background(), d.UserID)
	if err == nil || report.Repaired || len(report.Rejections) != 1 {
		t.Fatalf("corruption was silently repaired: %+v %v", report, err)
	}
}
func TestCancelledValidation(t *testing.T) {
	s, _, d, _ := testState(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := s.ValidateDelta(ctx, d); !errors.Is(err, context.Canceled) {
		t.Fatalf("ignored context cancellation: %v", err)
	}
}

func TestPreservesRelatedThreadsOnSlotOnlyPatch(t *testing.T) {
	s, r, d, _ := testState(t)
	target := r.snapshot.Threads[0].ID
	related := c.NewID()
	r.snapshot.Threads = append(r.snapshot.Threads, c.ThreadRecord{ID: related, UserID: d.UserID, Phase: c.PhasePaused, Version: 2})
	r.snapshot.Threads[0].RelatedThreadIDs = []c.ID{related}
	d.ExpectedVersions.Thread = 4
	d.Thread = &c.ThreadPatch{ThreadID: target, Touch: true}
	out, rej, err := s.ValidateDelta(context.Background(), d)
	if err != nil || len(rej) != 0 || out.Thread == nil || len(out.Thread.RelatedThreadIDs) != 1 || out.Thread.RelatedThreadIDs[0] != related {
		t.Fatalf("slot-only patch lost stored relation: %+v %+v %v", out.Thread, rej, err)
	}
}
func TestExistingPauseSatisfiesActiveThreadCap(t *testing.T) {
	s, r, d, now := testState(t)
	second := c.NewID()
	third := c.NewID()
	r.snapshot.Threads = append(r.snapshot.Threads,
		c.ThreadRecord{ID: second, UserID: d.UserID, Phase: c.PhaseWorking, Priority: .4, LastActiveAt: now, Version: 2},
		c.ThreadRecord{ID: third, UserID: d.UserID, Phase: c.PhaseWorking, Priority: .6, LastActiveAt: now, Version: 2})
	d.Thread = &c.ThreadPatch{ThreadID: c.NewID(), Create: &c.ThreadInit{Topic: "Synthetic"}}
	d.ThreadOps = []c.ThreadOp{{ThreadID: second, Op: "pause", Reason: "test"}}
	out, rej, err := s.ValidateDelta(context.Background(), d)
	if err != nil || len(out.ThreadOps) != 1 || hasRule(rej, "ST-005") {
		t.Fatalf("double pause despite existing capacity: %+v %+v %v", out.ThreadOps, rej, err)
	}
}
func TestFocusCannotBeOrphanedByThreadPatch(t *testing.T) {
	s, r, d, _ := testState(t)
	focus := r.snapshot.Threads[0].ID
	r.snapshot.Threads[0].Phase = c.PhaseMaintaining
	resolved := c.PhaseResolved
	d.ExpectedVersions.Thread = 4
	d.Thread = &c.ThreadPatch{ThreadID: focus, Phase: &resolved}
	out, rej, err := s.ValidateDelta(context.Background(), d)
	if err != nil || out.Thread.Phase != nil || !hasRule(rej, "ST-006") {
		t.Fatalf("allowed orphaned session focus: %+v %+v %v", out.Thread, rej, err)
	}
	next := c.NewID()
	r.snapshot.Threads = append(r.snapshot.Threads, c.ThreadRecord{ID: next, UserID: d.UserID, Phase: c.PhaseExploring, Version: 2})
	d.Session.FocusThread = &next
	out, _, err = s.ValidateDelta(context.Background(), d)
	if err != nil || out.Thread.Phase == nil || *out.Thread.Phase != c.PhaseResolved {
		t.Fatalf("valid refocus rejected: %+v %v", out.Thread, err)
	}
}
func TestRejectedNewThreadDoesNotBecomeFocus(t *testing.T) {
	s, _, d, _ := testState(t)
	tid := c.NewID()
	d.Session.FocusThread = &tid
	d.Thread = &c.ThreadPatch{ThreadID: tid, Create: &c.ThreadInit{Topic: ""}}
	out, rej, err := s.ValidateDelta(context.Background(), d)
	if err != nil || out.Session.FocusThread != nil || out.Thread != nil || !hasRule(rej, "ST-006") {
		t.Fatalf("stale focus allowed: %+v %+v %v", out, rej, err)
	}
}

func TestHypothesisRequiresOwnedActiveMemoryAndConsent(t *testing.T) {
	s, r, d, _ := testState(t)
	hypothesis := c.NewID()
	msg := c.NewID()
	r.snapshot.Threads[0].HypothesisIDs = []c.ID{hypothesis}
	r.hypotheses[hypothesis] = true
	r.evidence[msg] = true
	d.ExpectedVersions.Thread = 4
	d.Thread = &c.ThreadPatch{ThreadID: r.snapshot.Threads[0].ID, HypothesisSignals: []c.HypothesisSignal{{Op: c.HypSupport, ID: &hypothesis, Evidence: []c.ID{msg}}}}
	out, rej, err := s.ValidateDelta(context.Background(), d)
	if err != nil || len(rej) != 0 || len(out.Thread.HypothesisSignals) != 1 {
		t.Fatalf("verified hypothesis denied: %+v %+v %v", out.Thread, rej, err)
	}
	r.snapshot.User.MemoryPaused = true
	out, rej, err = s.ValidateDelta(context.Background(), d)
	if err != nil || len(out.Thread.HypothesisSignals) != 0 || !hasRule(rej, "ST-007") {
		t.Fatalf("memory pause ignored: %+v %+v %v", out.Thread, rej, err)
	}
	r.snapshot.User.MemoryPaused = false
	r.hypotheses[hypothesis] = false
	out, rej, err = s.ValidateDelta(context.Background(), d)
	if err != nil || len(out.Thread.HypothesisSignals) != 0 || !hasRule(rej, "ST-007") {
		t.Fatalf("soft-deleted/foreign hypothesis accepted: %+v %+v %v", out.Thread, rej, err)
	}
}
func TestProtocolTerminalStatusesCannotBeReactivated(t *testing.T) {
	s, r, d, _ := testState(t)
	pi := c.ProtocolInstance{ID: c.NewID(), UserID: d.UserID, ProtocolID: "placeholder_protocol", ProtocolVersion: "0.1.0", Status: c.PICompleted, Version: 3}
	r.snapshot.Protocol = &pi
	d.ExpectedVersions.Protocol = 3
	active := c.PIActive
	d.Protocol = &c.ProtocolPatch{InstanceID: pi.ID, NewStatus: &active}
	out, rej, err := s.ValidateDelta(context.Background(), d)
	if err != nil || out.Protocol.NewStatus != nil || !hasRule(rej, "ST-021") {
		t.Fatalf("terminal protocol resurrected: %+v %+v %v", out.Protocol, rej, err)
	}
}
