package orchestrator_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/gateway"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/orchestrator"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	safetypkg "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/safety"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
)

// These fixtures are deliberately NON-CLINICAL and cannot be promoted to a
// released bundle. WP-08 JSON files are equivalent to the WP-03 YAML fixtures.
type testIntegrationBundle struct{ files map[string][]byte }

func (b *testIntegrationBundle) Reference() c.BundleRef {
	return c.BundleRef{Version: "kb@0.9.0", PromptSHAs: map[string]string{"prompts/safety_l1.system.tmpl": "synthetic-sha"}}
}
func (*testIntegrationBundle) SyntheticOnly() bool { return true }
func (b *testIntegrationBundle) Artifact(ctx context.Context, p string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	x, ok := b.files[p]
	if !ok {
		return nil, errors.New("missing synthetic fixture: " + p)
	}
	return append([]byte(nil), x...), nil
}

type testIntegrationRegistry struct{ b *testIntegrationBundle }

func (r testIntegrationRegistry) Active(_ context.Context, _ c.ID) (ports.Bundle, error) {
	return r.b, nil
}
func testIntegrationAssets(t *testing.T) (testIntegrationRegistry, *safetypkg.Assets) {
	t.Helper()
	f := map[string][]byte{}
	for _, p := range []string{"schemas/problem_understanding.yaml", "config/thresholds.yaml", "config/policy_gate.yaml", "config/budgets.yaml"} {
		v, e := os.ReadFile(filepath.Join("testdata", "wp03", p))
		if e != nil {
			t.Fatal(e)
		}
		f[p] = v
	}
	for p, v := range map[string]string{
		"bundle.yaml":                          `{"name":"synthetic","version":"0.9.0","locales":["en"],"synthetic_only":true}`,
		"policy/safety_policy.yaml":            `{"schema":"safety_policy.v1","categories":{"other":{"template_family":"synthetic_case","escalate_to":"high"}},"template_families":{"synthetic_case":{"required_elements":["acknowledge","care","ask_safety","resources","invite_continue"],"forbidden":["method_details","promises"]}},"risk_fsm":{"follow_up_hours":24,"deescalate":{"from_high_to_elevated":{"min_hours_since_signal":12,"require":["no_new_signals"]},"from_elevated_to_low":{"min_hours_since_signal":24,"require":["no_new_signals","user_statement_safe_or_topic_neutral_3_turns"]}}},"second_look":{"downgrade":{"from":"high","to":"elevated","when_context_flags_any":["quoted","fictional"],"never_when":["first_person_present_tense"]}}}`,
		"lexicons/safety_l0.en.yaml":           `{"schema":"lexicon.v1","version":"synthetic-0.9.0","locale":"en","rules":[{"id":"synthetic.high.001","category":"other","level":"high","match":{"type":"literal","pattern":"TEST_HIGH_SIGNAL"},"second_look":true},{"id":"synthetic.imminent.001","category":"other","level":"imminent","match":{"type":"regex","pattern":"TEST_IMMINENT_SIGNAL"},"second_look":false}]}`,
		"resources/crisis_resources.yaml":      `{"schema":"crisis_resources.v1","last_full_verification":"2026-11-15","stale_after_days":180,"fallback_locale":"en","locales":{"en":[{"id":"synthetic_generic_resource","name":"Synthetic Resource","kind":"chat","contact":"TEST-ONLY-CONTACT","verified_on":"2026-11-15"}]}}`,
		"templates/crisis_templates.yaml":      `{"schema":"crisis_templates.v1","templates":{"synthetic_case":{"text":"SYNTHETIC TEST CRISIS SCRIPT (not for people)"},"resources_general_notice":{"text":"SYNTHETIC GENERAL RESOURCES NOTICE"}}}`,
		"templates/fallback_by_objective.yaml": `{"schema":"fallback.v1","templates":{"reflect":{"id":"test_reflect","text":"SYNTHETIC NORMAL FALLBACK"}}}`,
		"prompts/safety_l1.system.tmpl":        "SYNTHETIC UNIT TEST CLASSIFIER BLOCK: non-clinical marker only.",
	} {
		f[p] = []byte(v)
	}
	r := testIntegrationRegistry{b: &testIntegrationBundle{files: f}}
	a, err := safetypkg.Load(context.Background(), r.b, safetypkg.LoadOptions{AllowSyntheticForTests: true})
	if err != nil {
		t.Fatalf("safety fixture: %v", err)
	}
	return r, a
}

type testIntegrationClassifier struct {
	mu      sync.Mutex
	level   c.RiskLevel
	failure error
	calls   int
	risks   []c.RiskLevel
}

func (a *testIntegrationClassifier) Classify(ctx context.Context, in ports.SafetyInput) (safetypkg.Assessment, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.calls++
	a.risks = append(a.risks, in.Risk.Level)
	if err := ctx.Err(); err != nil {
		return safetypkg.Assessment{}, err
	}
	if a.failure != nil {
		return safetypkg.Assessment{}, a.failure
	}
	return safetypkg.Assessment{Version: "test-l1", Output: c.SafetyClassifierOutput{Level: a.level, Confidence: .99}}, nil
}
func (a *testIntegrationClassifier) Count() int { a.mu.Lock(); defer a.mu.Unlock(); return a.calls }

// The read model tracks existing WP-02 durable fixtures. It does not expose
// prototype state. The test finalizer changes the projection only after
// the fenced, all-or-nothing Complete has committed successfully.
type testIntegrationReader struct {
	store          *store
	mu             sync.Mutex
	risk           map[c.ID]c.RiskState
	threads        map[c.ID]c.ThreadRecord
	protocols      map[c.ID]*c.ProtocolInstance
	modes          map[c.ID]c.SessionMode
	counts         map[c.ID]int
	projectionSkew bool // deliberately stale WP-08 assembler projection
}

func newTestIntegrationReader(s *store) *testIntegrationReader {
	return &testIntegrationReader{store: s, risk: map[c.ID]c.RiskState{}, threads: map[c.ID]c.ThreadRecord{}, protocols: map[c.ID]*c.ProtocolInstance{}, modes: map[c.ID]c.SessionMode{}, counts: map[c.ID]int{}}
}
func (r *testIntegrationReader) ForTurn(_ context.Context, u, id c.ID) (contextstate.Snapshot, error) {
	s := r.store
	s.mu.Lock()
	var a c.Admission
	ok := false
	for _, t := range s.order {
		if t.admit.UserID == u && t.admit.TurnID == id && s.visible(u, t.req.ConversationID) {
			a = t.admit
			ok = true
			break
		}
	}
	if !ok {
		s.mu.Unlock()
		return contextstate.Snapshot{}, notFound()
	}
	sv, uv := s.sessionVersion[a.ConversationID], s.userVersion[u]
	s.mu.Unlock()
	r.mu.Lock()
	defer r.mu.Unlock()
	risk, exists := r.risk[u]
	if !exists {
		risk = c.RiskState{Level: c.RiskNone}
	}
	mode := r.modes[a.ConversationID]
	if !mode.Valid() {
		mode = c.ModeExploring
	}
	snap := contextstate.Snapshot{Session: c.SessionRecord{ID: a.SessionID, UserID: u, ConversationID: a.ConversationID, Version: sv, Mode: mode, TurnCount: r.counts[a.ConversationID]}, User: c.UserState{UserID: u, Version: uv, Risk: risk, TZ: "UTC", Locale: "en", ComfortBudget: .8}, ThreadUserTurns: map[c.ID]int{}}
	if thread, found := r.threads[a.ConversationID]; found {
		tid := thread.ID
		snap.Threads = []c.ThreadRecord{thread}
		snap.Session.FocusThreadID = &tid
		snap.ThreadUserTurns[tid] = 3
	}
	if proto := r.protocols[a.ConversationID]; proto != nil {
		clone := *proto
		snap.Protocol = &clone
	}
	return snap, nil
}
func (r *testIntegrationReader) Read(ctx context.Context, u, conv c.ID) (contextstate.Snapshot, error) {
	s := r.store
	s.mu.Lock()
	var id c.ID
	for i := len(s.order) - 1; i >= 0; i-- {
		t := s.order[i]
		if t.req.UserID == u && t.req.ConversationID == conv && t.result.Status == c.TurnRunning && s.visible(u, conv) {
			id = t.admit.TurnID
			break
		}
	}
	s.mu.Unlock()
	if !c.ValidID(id) {
		return contextstate.Snapshot{}, notFound()
	}
	snapshot, err := r.ForTurn(ctx, u, id)
	if err != nil {
		return snapshot, err
	}
	r.mu.Lock()
	skew := r.projectionSkew
	r.mu.Unlock()
	if skew {
		snapshot.User.Version++
	}
	return snapshot, nil
}
func (r *testIntegrationReader) UserEvidence(_ context.Context, u c.ID, ids []c.ID) (map[c.ID]bool, error) {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	ret := map[c.ID]bool{}
	for _, id := range ids {
		for _, t := range r.store.order {
			if t.req.UserID == u && r.store.visible(u, t.req.ConversationID) {
				for _, m := range t.messages {
					if m.Role == c.RoleUser && m.ID == id {
						ret[id] = true
					}
				}
			}
		}
	}
	return ret, nil
}
func (*testIntegrationReader) ExistingHypotheses(_ context.Context, _ c.ID, _ []c.ID) (map[c.ID]bool, error) {
	return map[c.ID]bool{}, nil
}
func (*testIntegrationReader) MissingFinalizations(_ context.Context, _ c.ID) (int, error) {
	return 0, nil
}
func (r *testIntegrationReader) Completed(_ context.Context, u, id c.ID) (bool, error) {
	r.store.mu.Lock()
	defer r.store.mu.Unlock()
	for _, t := range r.store.order {
		if t.admit.UserID == u && t.admit.TurnID == id && r.store.visible(u, t.req.ConversationID) {
			return t.result.Status == c.TurnCompleted, nil
		}
	}
	return false, notFound()
}

type testIntegrationFinalizer struct {
	s *store
	r *testIntegrationReader
}

func (f testIntegrationFinalizer) Fail(ctx context.Context, a c.Admission, code c.Code) error {
	return f.s.Fail(ctx, a, code)
}
func (f testIntegrationFinalizer) Complete(ctx context.Context, a c.Admission, p c.PreparedTurn) (c.TurnResult, error) {
	res, err := f.s.Complete(ctx, a, p)
	if err != nil {
		return res, err
	}
	f.r.mu.Lock()
	defer f.r.mu.Unlock()
	if transition := p.State.User.Risk; transition != nil {
		risk := f.r.risk[a.UserID]
		risk.Level = transition.To
		risk.Since = transition.At
		risk.History = append(risk.History, *transition)
		for _, event := range p.State.Events {
			if event.Type == c.EvRiskLevelChanged {
				raw, _ := json.Marshal(event.Payload)
				var x struct {
					FollowUpUntil *time.Time `json:"follow_up_until"`
				}
				_ = json.Unmarshal(raw, &x)
				if x.FollowUpUntil != nil {
					risk.FollowUpUntil = x.FollowUpUntil
				}
			}
		}
		f.r.risk[a.UserID] = risk
	}
	if p.State.Session.Mode != nil {
		f.r.modes[a.ConversationID] = *p.State.Session.Mode
	}
	f.r.counts[a.ConversationID]++
	if patch := p.State.Thread; patch != nil {
		thread := f.r.threads[a.ConversationID]
		thread.Version++
		if patch.Phase != nil {
			thread.Phase = *patch.Phase
		}
		if thread.Slots == nil {
			thread.Slots = map[string]c.SlotView{}
		}
		for _, slot := range patch.Slots {
			thread.Slots[slot.Slot] = c.SlotView{Slot: slot.Slot, Value: slot.Value, Status: slot.Status, Evidence: slot.Evidence}
		}
		f.r.threads[a.ConversationID] = thread
	}
	return res, nil
}

// Preserve the existing WP-05 two-stage test adapter while adding only
// explicitly labelled, synthetic integration decisions.
type testIntegrationBrain struct {
	base   *brain
	enrich func(ports.BrainInput, *c.BrainDecision)
}

func (b *testIntegrationBrain) Decide(ctx context.Context, in ports.BrainInput) (ports.BrainOutput, error) {
	out, err := b.base.Decide(ctx, in)
	if err != nil {
		return out, err
	}
	if out.Decision.RiskRead != nil && out.Decision.RiskRead.Level.Rank() >= c.RiskHigh.Rank() {
		out.Decision.RiskRead.Categories = []c.RiskCategory{c.RiskOther} // explicit synthetic test classification
	}
	if b.enrich != nil {
		b.enrich(in, &out.Decision)
	}
	return out, nil
}

type testIntegrationHarness struct {
	s            *store
	r            *testIntegrationReader
	user, conv   c.ID
	classifier   *testIntegrationClassifier
	client       *scriptedClient
	brain        *brain
	brainAdapter *testIntegrationBrain
	generator    *generator
	validator    *validator
	assets       *safetypkg.Assets
	registry     testIntegrationRegistry
	service      *orchestrator.Service
}

func newTestIntegrationHarness(t *testing.T) *testIntegrationHarness {
	t.Helper()
	h := &testIntegrationHarness{s: newStore(), user: c.NewID(), conv: c.NewID(), classifier: &testIntegrationClassifier{level: c.RiskNone}, client: &scriptedClient{text: "Synthetic answer"}, validator: &validator{}}
	h.s.provision(h.user, h.conv)
	h.r = newTestIntegrationReader(h.s)
	h.brain = &brain{client: h.client, user: h.user}
	h.brainAdapter = &testIntegrationBrain{base: h.brain}
	h.generator = &generator{client: h.client, user: h.user}
	h.registry, h.assets = testIntegrationAssets(t)
	h.configure(t, true)
	return h
}
func (h *testIntegrationHarness) configure(t *testing.T, synthetic bool) {
	t.Helper()
	gw, e := gateway.New(h.s, h.s)
	if e != nil {
		t.Fatal(e)
	}
	h.service, e = orchestrator.NewIntegrated(orchestrator.Dependencies{Gateway: gw, Store: h.s, Finalizer: testIntegrationFinalizer{h.s, h.r}, Recovery: h.s, Brain: h.brainAdapter, Generator: h.generator, Validator: h.validator, Tracer: trace.NewRecorder()}, orchestrator.Options{TotalTimeout: 5 * time.Second}, orchestrator.IntegratedOptions{Reader: h.r, Bundles: h.registry, Assets: h.assets, Classifier: h.classifier, AllowSyntheticForTests: synthetic, Now: func() time.Time { return time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC) }})
	if e != nil {
		t.Fatal(e)
	}
}
func (h *testIntegrationHarness) request(body string) c.TurnRequest {
	return c.TurnRequest{UserID: h.user, ConversationID: h.conv, ClientTurnID: c.NewID(), Text: body, Locale: "en", TZ: "UTC", Channel: "app_chat", Origin: c.OriginUser}
}
func (h *testIntegrationHarness) last(t *testing.T) c.PreparedTurn {
	t.Helper()
	h.s.mu.Lock()
	defer h.s.mu.Unlock()
	if len(h.s.finalized) == 0 {
		t.Fatal("no finalized turn")
	}
	return h.s.finalized[len(h.s.finalized)-1]
}

func TestIntegratedNormalTurnAndReplay(t *testing.T) {
	h := newTestIntegrationHarness(t)
	req := h.request("ordinary synthetic content")
	res, err := h.service.ProcessTurn(context.Background(), req)
	if err != nil || res.Status != c.TurnCompleted || res.Route != c.RouteNormal || res.Text != "Synthetic answer" {
		t.Fatalf("normal turn %+v %v", res, err)
	}
	p := h.last(t)
	if p.Trace.BundleVersion != "kb@0.9.0" || p.State.Session.TurnCountInc != 1 || p.State.ExpectedVersions.Session != 1 || p.State.ExpectedVersions.User != 1 || p.State.User.Risk != nil {
		t.Fatalf("unexpected bundle/state %+v", p.State)
	}
	if h.brain.count() != 1 || h.generator.count() != 1 || h.classifier.Count() != 1 {
		t.Fatal("not a two-stage turn")
	}
	replay, err := h.service.ProcessTurn(context.Background(), req)
	if err != nil || replay != res || h.brain.count() != 1 || h.generator.count() != 1 || h.classifier.Count() != 1 {
		t.Fatalf("idempotency violated %+v %v", replay, err)
	}
	if n, completed, _ := h.s.snapshot(); n != 2 || completed != 1 {
		t.Fatal("messages duplicated", n, completed)
	}
}

func TestIntegratedHighRiskCrisisAndAtomicRisk(t *testing.T) {
	h := newTestIntegrationHarness(t)
	res, err := h.service.ProcessTurn(context.Background(), h.request("TEST_HIGH_SIGNAL"))
	if err != nil || res.Status != c.TurnCompleted || res.Route != c.RouteCrisis || res.Safety == nil || len(res.Safety.Resources) == 0 {
		t.Fatalf("high risk failure %+v %v", res, err)
	}
	if h.brain.count() != 0 {
		t.Fatal("crisis called normal Brain")
	}
	p := h.last(t)
	if p.State.User.Risk == nil || p.State.User.Risk.From != c.RiskNone || p.State.User.Risk.To != c.RiskHigh || len(p.State.Events) != 1 || p.State.Events[0].Type != c.EvRiskLevelChanged {
		t.Fatalf("risk/outbox missing %+v", p.State)
	}
	h.r.mu.Lock()
	risk := h.r.risk[h.user]
	h.r.mu.Unlock()
	if risk.Level != c.RiskHigh || risk.FollowUpUntil == nil || !risk.FollowUpUntil.After(risk.Since) {
		t.Fatalf("follow-up deadline absent %+v", risk)
	}
}

func TestIntegratedL1OutageCannotLowerL0(t *testing.T) {
	h := newTestIntegrationHarness(t)
	h.classifier.failure = errors.New("synthetic classifier outage")
	res, err := h.service.ProcessTurn(context.Background(), h.request("TEST_HIGH_SIGNAL"))
	if err != nil || res.Route != c.RouteCrisis || !res.Degraded {
		t.Fatalf("L0 downgraded on outage %+v %v", res, err)
	}
	if got := h.last(t).State.User.Risk; got == nil || got.To != c.RiskHigh {
		t.Fatal("high risk not finalized")
	}
}

func TestIntegratedBrainRaisesAndReroutes(t *testing.T) {
	h := newTestIntegrationHarness(t)
	h.brain.raise = c.RiskHigh
	res, err := h.service.ProcessTurn(context.Background(), h.request("ordinary content"))
	if err != nil || res.Route != c.RouteCrisis || res.Status != c.TurnCompleted || h.brain.count() != 1 {
		t.Fatalf("brain escalation not routed %+v %v", res, err)
	}
	if got := h.last(t).State.User.Risk; got == nil || got.To != c.RiskHigh {
		t.Fatal("raised risk not committed")
	}
}

func TestIntegratedCommitRollbackThenExplicitRecovery(t *testing.T) {
	h := newTestIntegrationHarness(t)
	h.s.mu.Lock()
	h.s.rejectCommit = true
	h.s.mu.Unlock()
	req := h.request("TEST_HIGH_SIGNAL")
	res, err := h.service.ProcessTurn(context.Background(), req)
	if err == nil || res.Status != c.TurnFailed || res.AssistantMessageID != nil {
		t.Fatalf("rollback leaked reply %+v %v", res, err)
	}
	n, done, rolled := h.s.snapshot()
	if n != 1 || done != 0 || rolled != 1 {
		t.Fatalf("rollback: n=%d complete=%d rollback=%d", n, done, rolled)
	}
	h.r.mu.Lock()
	risk := h.r.risk[h.user]
	h.r.mu.Unlock()
	if risk.Level == c.RiskHigh {
		t.Fatal("rollback persisted risk")
	}
	if err := h.service.RetryFailed(context.Background(), h.user, req.ClientTurnID); err != nil {
		t.Fatal(err)
	}
	res, err = h.service.RecoverNext(context.Background(), h.user)
	if err != nil || res.Status != c.TurnCompleted || res.Route != c.RouteCrisis {
		t.Fatalf("recovery %+v %v", res, err)
	}
	n, done, _ = h.s.snapshot()
	if n != 2 || done != 1 {
		t.Fatal("duplicate recovery", n, done)
	}
}

func TestIntegratedSyntheticProductionBundleIsDenied(t *testing.T) {
	h := newTestIntegrationHarness(t)
	h.configure(t, false)
	res, err := h.service.ProcessTurn(context.Background(), h.request("ordinary content"))
	if err == nil || res.Status != c.TurnFailed || h.classifier.Count() != 0 || h.brain.count() != 0 {
		t.Fatalf("synthetic clinical assets activated %+v %v", res, err)
	}
	if n, _, _ := h.s.snapshot(); n != 1 {
		t.Fatal("admitted message not retained", n)
	}
}

func TestIntegratedSoftHiddenConversationRetainsData(t *testing.T) {
	h := newTestIntegrationHarness(t)
	req := h.request("ordinary content")
	res, err := h.service.ProcessTurn(context.Background(), req)
	if err != nil || res.Status != c.TurnCompleted {
		t.Fatalf("setup %v", err)
	}
	h.s.softHideConversation(h.conv)
	if _, err = h.service.ProcessTurn(context.Background(), req); err == nil {
		t.Fatal("soft-deleted conversation visible")
	}
	if n, complete, _ := h.s.snapshot(); n != 2 || complete != 1 {
		t.Fatal("soft-delete removed data")
	}
}

func TestIntegratedThreadAndSessionDeltaValidation(t *testing.T) {
	h := newTestIntegrationHarness(t)
	focus := c.NewID()
	h.r.mu.Lock()
	h.r.threads[h.conv] = c.ThreadRecord{ID: focus, UserID: h.user, Phase: c.PhaseExploring, Version: 4, Slots: map[string]c.SlotView{}, Priority: .5}
	h.r.mu.Unlock()
	h.brainAdapter.enrich = func(input ports.BrainInput, d *c.BrainDecision) {
		d.Understanding.ThreadID = &focus
		mode := c.ModeProblem
		d.Understanding.SessionMode = &mode
		summary := "SYNTHETIC session summary"
		d.Understanding.SessionSummaryDelta = &summary
		d.Understanding.Slots = []c.SlotUpdate{{Slot: "situation", Value: "SYNTHETIC event", Status: c.EpInferred, Evidence: []c.ID{input.WCtx.Turn.MessageID}}}
	}
	req := h.request("ordinary synthetic content")
	result, err := h.service.ProcessTurn(context.Background(), req)
	if err != nil || result.Status != c.TurnCompleted || result.Route != c.RouteNormal {
		t.Fatalf("thread update failed %+v %v", result, err)
	}
	p := h.last(t)
	if p.State.ExpectedVersions.Thread != 4 || p.State.Thread == nil || p.State.Thread.ThreadID != focus || len(p.State.Thread.Slots) != 1 || p.State.Thread.Slots[0].Status != c.EpInferred || p.State.Session.Mode == nil || *p.State.Session.Mode != c.ModeProblem {
		t.Fatalf("WP-08 did not validate focus/session proposal %+v", p.State)
	}
	h.r.mu.Lock()
	thread := h.r.threads[h.conv]
	mode := h.r.modes[h.conv]
	h.r.mu.Unlock()
	if thread.Version != 5 || thread.Slots["situation"].Status != c.EpInferred || mode != c.ModeProblem {
		t.Fatal("state not finalized once with reply", thread, mode)
	}
	// Next admitted turn must read the newly committed versions and session mode.
	h.brainAdapter.enrich = nil
	next, err := h.service.ProcessTurn(context.Background(), h.request("more synthetic content"))
	if err != nil || next.Status != c.TurnCompleted || h.last(t).State.ExpectedVersions.Thread != 5 {
		t.Fatalf("new context lost committed state %+v %v", next, err)
	}
}

func TestIntegratedInvalidEvidenceIsRejectedAndDegraded(t *testing.T) {
	h := newTestIntegrationHarness(t)
	focus := c.NewID()
	h.r.mu.Lock()
	h.r.threads[h.conv] = c.ThreadRecord{ID: focus, UserID: h.user, Phase: c.PhaseExploring, Version: 3, Slots: map[string]c.SlotView{}}
	h.r.mu.Unlock()
	h.brainAdapter.enrich = func(_ ports.BrainInput, d *c.BrainDecision) {
		d.Understanding.ThreadID = &focus
		d.Understanding.Slots = []c.SlotUpdate{{Slot: "situation", Value: "unverified inference", Status: c.EpInferred, Evidence: []c.ID{c.NewID()}}}
	}
	result, err := h.service.ProcessTurn(context.Background(), h.request("ordinary synthetic content"))
	if err != nil || result.Status != c.TurnCompleted || !result.Degraded {
		t.Fatalf("invalid evidence not rejected %+v %v", result, err)
	}
	p := h.last(t)
	if p.State.Thread != nil {
		t.Fatalf("rejected evidence must not touch the thread %+v", p.State.Thread)
	}
}

func TestIntegratedPersistedHighRiskConstrainsLaterTurn(t *testing.T) {
	h := newTestIntegrationHarness(t)
	if _, err := h.service.ProcessTurn(context.Background(), h.request("TEST_HIGH_SIGNAL")); err != nil {
		t.Fatal(err)
	}
	result, err := h.service.ProcessTurn(context.Background(), h.request("ordinary synthetic content"))
	if err != nil || result.Route != c.RouteCrisis || h.brain.count() != 0 {
		t.Fatalf("prior high risk was forgotten %+v %v", result, err)
	}
	h.classifier.mu.Lock()
	defer h.classifier.mu.Unlock()
	if len(h.classifier.risks) != 2 || h.classifier.risks[1] != c.RiskHigh {
		t.Fatalf("persisted risk not passed into Safety pre-gate %v", h.classifier.risks)
	}
}

func TestIntegratedNormalAndCrisisFallbackStillValidated(t *testing.T) {
	for _, tc := range []struct {
		name, text, want string
		route            c.Route
	}{
		{"normal", "ordinary synthetic content", "SYNTHETIC NORMAL FALLBACK", c.RouteNormal},
		{"crisis", "TEST_HIGH_SIGNAL", "SYNTHETIC TEST CRISIS SCRIPT (not for people)", c.RouteCrisis},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestIntegrationHarness(t)
			h.client.failure = errors.New("synthetic generator failure")
			result, err := h.service.ProcessTurn(context.Background(), h.request(tc.text))
			if err != nil || result.Status != c.TurnCompleted || !result.Degraded || result.Text != tc.want || result.Route != tc.route {
				t.Fatalf("not safely fell back %+v %v", result, err)
			}
		})
	}
}

func TestIntegratedValidatorFailureRetainsUserOnly(t *testing.T) {
	h := newTestIntegrationHarness(t)
	h.validator.alwaysReject = true
	result, err := h.service.ProcessTurn(context.Background(), h.request("TEST_HIGH_SIGNAL"))
	if err == nil || result.Status != c.TurnFailed || result.AssistantMessageID != nil {
		t.Fatalf("invalid output committed %+v %v", result, err)
	}
	n, done, _ := h.s.snapshot()
	if n != 1 || done != 0 {
		t.Fatalf("partial unsafe state committed n=%d done=%d", n, done)
	}
}

func TestIntegratedStaleWorkingContextFailsBeforeGeneration(t *testing.T) {
	h := newTestIntegrationHarness(t)
	h.r.mu.Lock()
	h.r.projectionSkew = true
	h.r.mu.Unlock()
	result, err := h.service.ProcessTurn(context.Background(), h.request("ordinary synthetic content"))
	if err == nil || result.Status != c.TurnFailed || result.AssistantMessageID != nil {
		t.Fatalf("stale projection completed %+v %v", result, err)
	}
	if h.brain.count() != 0 || h.generator.count() != 0 {
		t.Fatal("inference proceeded after stale versions")
	}
	n, completed, _ := h.s.snapshot()
	if n != 1 || completed != 0 {
		t.Fatal("stale context partially finalized state")
	}
}

func TestIntegratedRouterPreservesBufferedSafetyConstraints(t *testing.T) {
	route := (orchestrator.Router{}).Select(ports.RouteInput{Safety: c.SafetySignal{Level: c.RiskHigh}, WCtx: c.WorkingContext{Risk: c.RiskView{Current: c.RiskLow}}})
	if route.Route != c.RouteCrisis || !route.ForceBufferedOutput || !route.RiskConstraints {
		t.Fatalf("high risk route not buffered %+v", route)
	}
	followup := (orchestrator.Router{}).Select(ports.RouteInput{Safety: c.SafetySignal{Level: c.RiskLow}, WCtx: c.WorkingContext{Risk: c.RiskView{Current: c.RiskElevated, FollowUpActive: true}}})
	if followup.Route != c.RouteNormal || !followup.ForceBufferedOutput || !followup.RiskConstraints {
		t.Fatalf("followup route not constrained %+v", followup)
	}
}
