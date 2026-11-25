package protocol

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/state"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

const syntheticProtocol = `schema: protocol.v1
id: test_exercise
version: 1.0.0
name: SYNTHETIC EXERCISE
family: planning
one_liner: SYNTHETIC TEST ONLY
est_minutes: 3
fit_tags: [synthetic_tag]
eligibility:
  require_phase: [working]
  require_slots_filled: [situation]
  require_risk_at_most: low
  min_user_readiness: medium
  forbid_flags: [synthetic_flag]
  decline_cooldown_days: 14
  min_days_between_runs: 2
  max_runs_per_thread: 3
steps:
  - id: first
    goal: SYNTHETIC GOAL
    instruction: SYNTHETIC FIRST INSTRUCTION
    expected_inputs: [situation]
    success_criteria: test_predicate
    on_success: last
    on_stuck: {adapt: simpler, after: 1, then: pause}
    on_distress: stop
    on_decline: stop
    adaptations:
      - {id: simpler, instruction: SYNTHETIC ADAPTED INSTRUCTION}
  - id: last
    goal: SYNTHETIC SECOND GOAL
    instruction: SYNTHETIC FINAL INSTRUCTION
    expected_inputs: []
    success_criteria: test_predicate
    on_success: complete
    on_stuck: {adapt: '', after: 2, then: pause}
    on_distress: stop
    on_decline: stop
offer:
  why_templates: [synthetic_why]
  must_say: [synthetic_optout]
stop_rules:
  - {when: risk_at_least_elevated, action: stop, note: too_distressed}
  - {when: stuck_count_gte_2, action: pause, follow_up_days: 2}
completion:
  record_outcome: true
  ask_rating: false
  follow_up_days: 0
  memory_write: {kind: event, tags: [protocol_outcome]}
safety_notes: SYNTHETIC, NOT CLINICALLY REVIEWED
`

type syntheticBundle struct {
	files     map[string][]byte
	synthetic bool
}

func (b syntheticBundle) Reference() c.BundleRef {
	return c.BundleRef{Version: "kb@1.0.0", PromptSHAs: map[string]string{}}
}
func (b syntheticBundle) Artifact(ctx context.Context, name string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	v, ok := b.files[name]
	if !ok {
		return nil, fmt.Errorf("absent %s", name)
	}
	return append([]byte(nil), v...), nil
}
func (b syntheticBundle) ArtifactNames() []string {
	out := []string{}
	for k := range b.files {
		out = append(out, k)
	}
	return out
}
func (b syntheticBundle) SyntheticOnly() bool { return b.synthetic }
func fakeBundle() syntheticBundle {
	return syntheticBundle{synthetic: true, files: map[string][]byte{
		"bundle.yaml":                   []byte("name: synthetic\nversion: 1.0.0\nsynthetic_only: true\n"),
		"protocols/test_exercise.yaml":  []byte(syntheticProtocol),
		"templates/protocol_offer.yaml": []byte("schema: protocol_offer.v1\nwhy_templates:\n  synthetic_why: {text: SYNTHETIC REASON}\nmust_say:\n  synthetic_optout: {text: SYNTHETIC OPT OUT}\n"),
	}}
}

type fakeReader struct {
	snapshot contextstate.Snapshot
	userTurn c.ID
	owned    bool
	err      error
}

func (f *fakeReader) Read(context.Context, c.ID, c.ID) (contextstate.Snapshot, error) {
	return f.snapshot, f.err
}
func (f *fakeReader) ForTurn(_ context.Context, u, turn c.ID) (contextstate.Snapshot, error) {
	if f.err != nil {
		return contextstate.Snapshot{}, f.err
	}
	if f.snapshot.User.UserID != u || turn != f.userTurn {
		return contextstate.Snapshot{}, ErrInvalid
	}
	return f.snapshot, nil
}
func (f *fakeReader) UserEvidence(_ context.Context, u c.ID, ids []c.ID) (map[c.ID]bool, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := map[c.ID]bool{}
	if u == f.snapshot.User.UserID && f.owned {
		for _, id := range ids {
			if c.ValidID(id) {
				out[id] = true
			}
		}
	}
	return out, nil
}
func (*fakeReader) ExistingHypotheses(context.Context, c.ID, []c.ID) (map[c.ID]bool, error) {
	return map[c.ID]bool{}, nil
}
func (*fakeReader) Completed(context.Context, c.ID, c.ID) (bool, error)     { return false, nil }
func (*fakeReader) MissingFinalizations(context.Context, c.ID) (int, error) { return 0, nil }

// Selected content paths are the server-owned result of JEV selection.
func selectedContentIDs(briefs []c.ProtocolBrief) []string {
	ids := make([]string, 0, len(briefs))
	for _, brief := range briefs {
		ids = append(ids, "protocols/"+brief.ProtocolID+".yaml")
	}
	return ids
}

type fixture struct {
	e   *Engine
	r   *fakeReader
	w   c.WorkingContext
	t   c.ThreadRecord
	u   c.UserState
	in  ResolveInput
	now time.Time
}

func setup(t *testing.T, custom ...syntheticBundle) fixture {
	t.Helper()
	ctx := context.Background()
	b := fakeBundle()
	if len(custom) > 0 {
		b = custom[0]
	}
	a, e := Load(ctx, b, LoadOptions{AllowSyntheticForTests: true})
	if e != nil {
		t.Fatal(e)
	}
	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	uid, tid, sid, turn, msg := c.NewID(), c.NewID(), c.NewID(), c.NewID(), c.NewID()
	thread := c.ThreadRecord{ID: tid, UserID: uid, Phase: c.PhaseWorking, ProblemTags: []string{"synthetic_tag"}, Slots: map[string]c.SlotView{"situation": {Slot: "situation", Status: c.EpReported, Value: "synthetic event", Evidence: []c.ID{msg}}}, Version: 1}
	view := &c.ThreadView{ID: tid, Phase: c.PhaseWorking, Slots: []c.SlotView{{Slot: "situation", Status: c.EpReported, Value: "synthetic event", Evidence: []c.ID{msg}}}}
	user := c.UserState{UserID: uid, Risk: c.RiskState{Level: c.RiskLow}, Prefs: c.PrefsView{Language: "en"}, Version: 1}
	wc := c.WorkingContext{Turn: c.TurnView{TurnID: turn, MessageID: msg, Text: "i want to try an exercise", Lang: "en"}, FocusThread: view, Session: c.SessionView{SessionID: sid}, Risk: c.RiskView{Current: c.RiskLow}, Prefs: user.Prefs, Eligibility: c.Eligibility{ProtocolsPossible: true}}
	r := &fakeReader{snapshot: contextstate.Snapshot{User: user, Session: c.SessionRecord{ID: sid, UserID: uid, Version: 1}, Threads: []c.ThreadRecord{thread}}, userTurn: turn, owned: true}
	engine, e := New(a, r, func() time.Time { return now })
	if e != nil {
		t.Fatal(e)
	}
	return fixture{e: engine, r: r, w: wc, t: thread, u: user, now: now, in: ResolveInput{UserID: uid, TurnID: turn, UserMessageID: msg, Bundle: a.Reference(), WCtx: wc, Risk: c.RiskLow}}
}
func (f *fixture) offered() c.ProtocolInstance {
	p := c.ProtocolInstance{ID: c.NewID(), UserID: f.u.UserID, ProtocolID: "test_exercise", ProtocolVersion: "1.0.0", Status: c.PIOffered, StepID: "first", Version: 1}
	f.r.snapshot.Protocol = &p
	f.in.Active = &p
	f.w.Protocol = &c.ProtocolView{InstanceID: p.ID, ProtocolID: p.ProtocolID, Version: p.ProtocolVersion, Status: p.Status, StepID: p.StepID}
	f.in.WCtx = f.w
	return p
}
func (f *fixture) active() c.ProtocolInstance {
	p := f.offered()
	p.Status = c.PIActive
	p.ConsentAt = &f.now
	p.ConsentMessageID = &f.in.UserMessageID
	f.r.snapshot.Protocol = &p
	f.in.Active = &p
	f.w.Protocol.Status = c.PIActive
	f.in.WCtx = f.w
	return p
}
func directive(action c.ProtocolAction, outcome c.StepOutcome, inputs map[string]string) *c.ProtocolDirective {
	d := &c.ProtocolDirective{Action: action, InputsCaptured: inputs}
	if outcome != "" {
		d.Outcome = &outcome
	}
	return d
}
func (f *fixture) resolve(d *c.ProtocolDirective) (c.ProtocolStepSpec, c.ProtocolPatch, error) {
	f.in.Plan = c.ResponsePlan{Protocol: d}
	return f.e.ResolveTurn(context.Background(), f.in)
}
func requireStatus(t *testing.T, p c.ProtocolPatch, want c.ProtocolStatus) {
	t.Helper()
	if p.NewStatus == nil || *p.NewStatus != want {
		t.Fatalf("status=%v want=%s", p.NewStatus, want)
	}
}

func TestAssetApprovalFailClosed(t *testing.T) {
	b := fakeBundle()
	if _, e := Load(context.Background(), b, LoadOptions{}); !errors.Is(e, ErrUnreviewed) {
		t.Fatalf("unreviewed accepted: %v", e)
	}
	b.synthetic = false
	if _, e := Load(context.Background(), b, LoadOptions{AllowSyntheticForTests: true}); !errors.Is(e, ErrUnreviewed) {
		t.Fatalf("forged production content accepted: %v", e)
	}
	b = fakeBundle()
	b.files["protocols/test_exercise.yaml"] = []byte(strings.ReplaceAll(syntheticProtocol, "on_success: last", "on_success: missing"))
	if _, e := Load(context.Background(), b, LoadOptions{AllowSyntheticForTests: true}); !errors.Is(e, ErrInvalid) {
		t.Fatalf("invalid edge accepted: %v", e)
	}
}
func TestShortlistEligibilityAndRanking(t *testing.T) {
	f := setup(t)
	req := ShortlistContext{WCtx: f.w, Thread: &f.t, User: f.u, Readiness: ReadinessHigh, FlagsVerified: true, Now: f.now}
	candidates, e := f.e.ShortlistWithContext(context.Background(), req)
	if e != nil || len(candidates) != 1 || candidates[0].ProtocolID != "test_exercise" {
		t.Fatalf("eligible=%v err=%v", candidates, e)
	}
	for _, scenario := range []struct {
		name   string
		mutate func(*ShortlistContext)
	}{
		{"contraindication", func(v *ShortlistContext) { v.Flags = []string{"synthetic_flag"} }},
		{"risk", func(v *ShortlistContext) { v.WCtx.Risk.Current = c.RiskElevated }},
		{"follow_up", func(v *ShortlistContext) { v.WCtx.Risk.FollowUpActive = true }},
		{"phase", func(v *ShortlistContext) {
			v.Thread = &c.ThreadRecord{ID: f.t.ID, UserID: f.t.UserID, Phase: c.PhaseExploring, ProblemTags: f.t.ProblemTags}
		}},
		{"missing_slot", func(v *ShortlistContext) { v.WCtx.FocusThread.Slots = nil }},
		{"declined", func(v *ShortlistContext) {
			v.User.ProtocolHistory = []c.ProtocolHistoryBrief{{ProtocolID: "test_exercise", Outcome: "declined", At: f.now.Add(-24 * time.Hour)}}
		}},
		{"last_run", func(v *ShortlistContext) {
			v.User.ProtocolHistory = []c.ProtocolHistoryBrief{{ProtocolID: "test_exercise", Outcome: "completed", At: f.now.Add(-24 * time.Hour)}}
		}},
		{"max_runs", func(v *ShortlistContext) {
			v.User.ProtocolHistory = []c.ProtocolHistoryBrief{{ProtocolID: "test_exercise", Outcome: "completed", At: f.now.Add(-10 * 24 * time.Hour)}, {ProtocolID: "test_exercise", Outcome: "completed", At: f.now.Add(-8 * 24 * time.Hour)}, {ProtocolID: "test_exercise", Outcome: "completed", At: f.now.Add(-6 * 24 * time.Hour)}}
		}},
		{"avoid", func(v *ShortlistContext) { v.WCtx.Prefs.Avoid = []string{"test_exercise"} }},
		{"wrong_owner", func(v *ShortlistContext) {
			v.Thread = &c.ThreadRecord{ID: f.t.ID, UserID: c.NewID(), Phase: c.PhaseWorking}
		}},
		{"non_english", func(v *ShortlistContext) { v.WCtx.Prefs.Language = "fa" }},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			v := req
			v.WCtx = f.w
			v.User = f.u
			scenario.mutate(&v)
			got, err := f.e.ShortlistWithContext(context.Background(), v)
			if err != nil || len(got) != 0 {
				t.Fatalf("unsafe shortlist=%v error=%v", got, err)
			}
		})
	}
	fresh := setup(t)
	if found, _ := fresh.e.Shortlist(context.Background(), fresh.w, &fresh.t, fresh.u); len(found) != 1 {
		t.Fatal("missing readiness hid eligible knowledge")
	}
}
func TestOfferThenExplicitConsent(t *testing.T) {
	f := setup(t)
	candidates, _ := f.e.ShortlistWithContext(context.Background(), ShortlistContext{WCtx: f.w, Thread: &f.t, User: f.u, Readiness: ReadinessHigh, FlagsVerified: true, Now: f.now})
	// This protocol has a contraindication requiring a verified flag source.
	// A UI/DB must supply a verified inventory before a real offer.
	f.in.SelectedContentIDs = selectedContentIDs(candidates)
	f.in.Objective = c.ObjOfferIntervention
	id := "test_exercise"
	_, _, err := f.resolve(&c.ProtocolDirective{Action: c.PAOffer, ProtocolID: &id})
	if err != nil {
		t.Fatal("optional offer required an absence inventory", err)
	}
	b := fakeBundle()
	b.files["protocols/test_exercise.yaml"] = []byte(strings.ReplaceAll(syntheticProtocol, "forbid_flags: [synthetic_flag]", "forbid_flags: []"))
	a := setup(t, b)
	list, e := a.e.ShortlistWithContext(context.Background(), ShortlistContext{WCtx: a.w, Thread: &a.t, User: a.u, Readiness: ReadinessHigh, Now: a.now})
	if e != nil || len(list) != 1 {
		t.Fatalf("reviewed-graph offer eligibility: %v %v", list, e)
	}
	a.in.SelectedContentIDs = selectedContentIDs(list)
	a.in.Objective = c.ObjOfferIntervention
	spec, patch, e := a.resolve(&c.ProtocolDirective{Action: c.PAOffer, ProtocolID: &id})
	if e != nil || patch.Create == nil || patch.Create.ProtocolID != id || spec.Offer == nil || len(spec.Offer.Why) == 0 || !spec.Offer.OptOutLine || patch.NewStep == nil || *patch.NewStep != "first" {
		t.Fatalf("offer spec=%+v patch=%+v err=%v", spec, patch, e)
	}
	// Test the consent lifecycle using a persisted offered instance. No clinical
	// instructions or synthetic offer are delivered in this test.
	p := f.offered()
	d := directive(c.PAAdvance, c.OutSuccess, nil)
	if _, _, e := f.resolve(d); !errors.Is(e, ErrConsent) {
		t.Fatalf("started without consent: %v", e)
	}
	f.in.UserAffirmed = true
	f.in.WCtx.Turn.Text = "yes"
	f.r.owned = false
	if _, _, e := f.resolve(d); !errors.Is(e, ErrConsent) {
		t.Fatalf("accepted unowned consent: %v", e)
	}
	f.r.owned = true
	spec, patch, e = f.resolve(d)
	if e != nil || patch.ConsentAt == nil || patch.ConsentMessageID == nil || *patch.ConsentMessageID != f.in.UserMessageID || spec.StepID != "first" || spec.InstanceID != p.ID {
		t.Fatalf("consent: spec=%+v patch=%+v err=%v", spec, patch, e)
	}
	requireStatus(t, patch, c.PIActive)
	f.in.Risk = c.RiskElevated
	_, patch, e = f.resolve(d)
	if e != nil {
		t.Fatal(e)
	}
	requireStatus(t, patch, c.PIStopped)
}
func TestTransitionsInterruptionsRecovery(t *testing.T) {
	f := setup(t)
	p := f.active()
	_ = p
	spec, patch, e := f.resolve(directive(c.PAAdvance, c.OutSuccess, nil))
	if e != nil || spec.StepID != "last" || patch.NewStep == nil {
		t.Fatalf("missing input: %+v %+v %v", spec, patch, e)
	}
	spec, patch, e = f.resolve(directive(c.PAAdapt, c.OutStuck, nil))
	if e != nil || spec.Instruction != "SYNTHETIC ADAPTED INSTRUCTION" || patch.StuckInc != 1 {
		t.Fatalf("adapt: %+v %+v %v", spec, patch, e)
	}
	f.in.Active.StuckCount = 1
	f.r.snapshot.Protocol.StuckCount = 1
	_, patch, e = f.resolve(directive(c.PARepeat, c.OutStuck, nil))
	if e != nil {
		t.Fatal(e)
	}
	if patch.NewStatus != nil {
		t.Fatal("difficulty forced protocol exit", patch)
	}
	f.in.Active.StuckCount = 0
	f.r.snapshot.Protocol.StuckCount = 0
	f.in.WCtx.Turn.Text = "synthetic input"
	spec, patch, e = f.resolve(directive(c.PAAdvance, c.OutSuccess, map[string]string{"situation": "synthetic input"}))
	if e != nil || spec.StepID != "last" || patch.NewStep == nil || *patch.NewStep != "last" {
		t.Fatalf("advance: %+v %+v %v", spec, patch, e)
	}
	f.in.Active.StepID = "last"
	f.r.snapshot.Protocol.StepID = "last"
	_, patch, e = f.resolve(directive(c.PAComplete, c.OutSuccess, nil))
	if e != nil {
		t.Fatal(e)
	}
	requireStatus(t, patch, c.PICompleted)
	_, patch, e = f.resolve(directive(c.PAStop, c.OutDecline, nil))
	if e != nil {
		t.Fatal(e)
	}
	requireStatus(t, patch, c.PIStopped)
	f.in.Risk = c.RiskElevated
	_, patch, e = f.resolve(directive(c.PARepeat, c.OutPartial, nil))
	if e != nil {
		t.Fatal(e)
	}
	requireStatus(t, patch, c.PIStopped)
	f.in.Risk = c.RiskLow
	f.in.Active.Status = c.PIPaused
	f.r.snapshot.Protocol.Status = c.PIPaused
	_, _, e = f.resolve(directive(c.PAAdvance, c.OutSuccess, nil))
	if !errors.Is(e, ErrConsent) {
		t.Fatalf("auto-resumed: %v", e)
	}
	f.in.UserAffirmed = true
	f.in.WCtx.Turn.Text = "yes"
	_, patch, e = f.resolve(directive(c.PAAdvance, c.OutSuccess, nil))
	if e != nil {
		t.Fatal(e)
	}
	requireStatus(t, patch, c.PIActive)
	f.in.UserAffirmed = false
	_, _, e = f.resolve(directive(c.PAAdvance, c.OutSuccess, nil))
	if !errors.Is(e, ErrConsent) {
		t.Fatalf("advanced paused instance: %v", e)
	}
}
func TestInvalidActionsAndOwnerIsolation(t *testing.T) {
	f := setup(t)
	f.active()
	for _, d := range []*c.ProtocolDirective{
		directive(c.PAComplete, c.OutSuccess, nil),
		directive(c.PAAdvance, c.OutSuccess, map[string]string{"unauthorized": "x"}),
		directive(c.PAAdvance, c.OutSuccess, map[string]string{"situation": strings.Repeat("word ", 31)}),
		directive(c.PAAdapt, c.OutSuccess, nil),
		directive(c.PAStart, c.OutSuccess, nil),
	} {
		_, _, e := f.resolve(d)
		if !errors.Is(e, ErrInvalid) {
			t.Fatalf("unexpected valid action %+v: %v", d, e)
		}
	}
	stale := *f.in.Active
	f.in.Active = &stale
	f.in.Active.Version++
	if _, _, e := f.resolve(directive(c.PAStop, c.OutDecline, nil)); !errors.Is(e, ErrInvalid) {
		t.Fatalf("accepted stale instance: %v", e)
	}
	f.in.Active.Version--
	f.in.UserID = c.NewID()
	if _, _, e := f.resolve(directive(c.PAStop, c.OutDecline, nil)); !errors.Is(e, ErrInvalid) {
		t.Fatalf("cross-user access: %v", e)
	}
	f.in.UserID = f.u.UserID
	f.in.Bundle.Version = "kb@2.0.0"
	if _, _, e := f.resolve(directive(c.PAStop, c.OutDecline, nil)); !errors.Is(e, ErrInvalid) {
		t.Fatalf("wrong pinned version: %v", e)
	}
}

// WP-13 -> WP-08 integration uses the same owner-scoped reader, exact pinned
// protocol graph and the production StateCommitter's sanitization logic.
type protocolRegistry struct{ source syntheticBundle }

func (r protocolRegistry) Active(ctx context.Context, _ c.ID) (ports.Bundle, error) {
	return r.source, ctx.Err()
}
func stateBundle() syntheticBundle {
	b := fakeBundle()
	b.files["protocols/test_exercise.yaml"] = []byte(strings.ReplaceAll(syntheticProtocol, "forbid_flags: [synthetic_flag]", "forbid_flags: []"))
	for _, item := range []struct{ path, schema string }{
		{"schemas/problem_understanding.yaml", "problem.v1"},
		{"config/thresholds.yaml", "thresholds.v1"},
		{"config/policy_gate.yaml", "policy_gate.v1"},
		{"config/budgets.yaml", "budgets.v1"},
	} {
		b.files[item.path] = []byte("schema: " + item.schema + "\n")
	}
	return b
}
func TestStateCommitterValidatesOfferAndConsent(t *testing.T) {
	f := setup(t, stateBundle())
	svc, err := state.New(f.r, protocolRegistry{source: stateBundle()})
	if err != nil {
		t.Fatal(err)
	}
	offers, err := f.e.ShortlistWithContext(context.Background(), ShortlistContext{WCtx: f.w, Thread: &f.t, User: f.u, Readiness: ReadinessHigh, Now: f.now})
	if err != nil || len(offers) != 1 {
		t.Fatalf("shortlist=%v error=%v", offers, err)
	}
	f.in.SelectedContentIDs, f.in.Objective = selectedContentIDs(offers), c.ObjOfferIntervention
	pid := "test_exercise"
	_, offer, err := f.resolve(&c.ProtocolDirective{Action: c.PAOffer, ProtocolID: &pid})
	if err != nil {
		t.Fatal(err)
	}
	offerDelta := c.StateDelta{UserID: f.u.UserID, TurnID: f.in.TurnID, ExpectedVersions: c.Versions{Session: 1, User: 1}, Session: c.SessionPatch{LastObjective: c.ObjOfferIntervention, TurnCountInc: 1}, Protocol: &offer}
	clean, rejected, err := svc.ValidateDelta(context.Background(), offerDelta)
	if err != nil || len(rejected) != 0 || clean.Protocol == nil || clean.Protocol.Create == nil {
		t.Fatalf("WP-08 offer: %+v %+v %v", clean.Protocol, rejected, err)
	}
	_ = f.offered() // emulate the first turn's durable finalization
	f.in.UserAffirmed = true
	f.in.WCtx.Turn.Text = "yes"
	_, start, err := f.resolve(directive(c.PAAdvance, c.OutSuccess, nil))
	if err != nil {
		t.Fatal(err)
	}
	startDelta := c.StateDelta{UserID: f.u.UserID, TurnID: f.in.TurnID, ExpectedVersions: c.Versions{Session: 1, User: 1, Protocol: 1}, Session: c.SessionPatch{LastObjective: c.ObjContinueIntervention, TurnCountInc: 1}, Protocol: &start}
	clean, rejected, err = svc.ValidateDelta(context.Background(), startDelta)
	if err != nil || len(rejected) != 0 || clean.Protocol == nil || clean.Protocol.NewStatus == nil || *clean.Protocol.NewStatus != c.PIActive || clean.Protocol.ConsentMessageID == nil {
		t.Fatalf("WP-08 consent: %+v %+v %v", clean.Protocol, rejected, err)
	}
	// A forged user-message role must be rejected even with an affirmative flag.
	f.r.owned = false
	clean, rejected, err = svc.ValidateDelta(context.Background(), startDelta)
	if err != nil || clean.Protocol == nil || clean.Protocol.NewStatus != nil {
		t.Fatalf("WP-08 accepted forged consent: %+v %+v %v", clean.Protocol, rejected, err)
	}
	found := false
	for _, r := range rejected {
		if r.RuleCode == "ST-020" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing consent rejection: %+v", rejected)
	}
	f.r.owned = true
	startDelta.ExpectedVersions.Protocol = 2
	if _, _, err = svc.ValidateDelta(context.Background(), startDelta); err == nil {
		t.Fatal("accepted stale protocol version")
	}
}

func TestFollowUpsRequireUserOptIn(t *testing.T) {
	b := fakeBundle()
	b.files["protocols/test_exercise.yaml"] = []byte(strings.ReplaceAll(syntheticProtocol, "follow_up_days: 0", "follow_up_days: 3"))
	f := setup(t, b)
	f.active()
	f.in.Active.StepID = "last"
	f.r.snapshot.Protocol.StepID = "last"
	_, patch, err := f.resolve(directive(c.PAComplete, c.OutSuccess, nil))
	if err != nil || patch.FollowUpAt != nil {
		t.Fatalf("unconsented follow-up: %v %+v", err, patch)
	}
	f.r.snapshot.User.ProactiveOptIn = true
	_, patch, err = f.resolve(directive(c.PAComplete, c.OutSuccess, nil))
	if err != nil || patch.FollowUpAt == nil || !patch.FollowUpAt.Equal(f.now.AddDate(0, 0, 3)) {
		t.Fatalf("opt-in follow-up due timestamp: %v %+v", err, patch)
	}
	f.in.Active.StepID = "first"
	f.r.snapshot.Protocol.StepID = "first"
	f.in.Risk = c.RiskElevated
	_, patch, err = f.resolve(directive(c.PARepeat, c.OutDistress, nil))
	if err != nil {
		t.Fatal(err)
	}
	requireStatus(t, patch, c.PIStopped)
}

func TestPinnedAssetsAndRequiredTextCannotBeMutated(t *testing.T) {
	f := setup(t)
	p, ok := f.e.assets.Protocol("test_exercise")
	if !ok {
		t.Fatal("fixture")
	}
	p.Steps[0].Instruction = "unapproved mutation"
	p.Offer.MustSay[0] = "invented"
	fresh, _ := f.e.assets.Protocol("test_exercise")
	if fresh.Steps[0].Instruction == p.Steps[0].Instruction || fresh.Offer.MustSay[0] == "invented" {
		t.Fatal("mutable release snapshot")
	}
	ref := f.e.Reference()
	ref.PromptSHAs["bad"] = "mutated"
	if _, ok := f.e.Reference().PromptSHAs["bad"]; ok {
		t.Fatal("mutable bundle reference")
	}
	text, e := f.e.RequiredText(c.ProtocolStepSpec{ProtocolID: "test_exercise", MustSay: []string{"synthetic_optout"}})
	if e != nil || text["synthetic_optout"] != "SYNTHETIC OPT OUT" {
		t.Fatal("approved required text", e)
	}
	if _, e = f.e.RequiredText(c.ProtocolStepSpec{ProtocolID: "test_exercise", MustSay: []string{"invented"}}); e == nil {
		t.Fatal("invented requirement")
	}
}
