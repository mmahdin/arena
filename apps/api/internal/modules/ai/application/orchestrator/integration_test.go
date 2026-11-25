package orchestrator_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"sync"
	"testing"
	"time"
)

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

type testIntegrationReader struct {
	carry              map[c.ID][]c.FutureHint
	store              *store
	mu                 sync.Mutex
	risk               map[c.ID]c.RiskState
	threads            map[c.ID]c.ThreadRecord
	protocols          map[c.ID]*c.ProtocolInstance
	modes              map[c.ID]c.SessionMode
	counts             map[c.ID]int
	languagePreference string // synthetic saved-preference projection
	projectionSkew     bool   // deliberately stale WP-08 assembler projection
}

func newTestIntegrationReader(s *store) *testIntegrationReader {
	return &testIntegrationReader{carry: map[c.ID][]c.FutureHint{}, store: s, risk: map[c.ID]c.RiskState{}, threads: map[c.ID]c.ThreadRecord{}, protocols: map[c.ID]*c.ProtocolInstance{}, modes: map[c.ID]c.SessionMode{}, counts: map[c.ID]int{}}
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
	snap := contextstate.Snapshot{FutureHints: r.carry[a.ConversationID], Session: c.SessionRecord{ID: a.SessionID, UserID: u, ConversationID: a.ConversationID, Version: sv, Mode: mode, TurnCount: r.counts[a.ConversationID]}, User: c.UserState{UserID: u, Version: uv, Risk: risk, TZ: "UTC", Locale: "en", Prefs: c.PrefsView{Language: r.languagePreference}, ComfortBudget: .8}, ThreadUserTurns: map[c.ID]int{}}
	if thread, found := r.threads[a.ConversationID]; found {
		tid := thread.ID
		snap.Threads = []c.ThreadRecord{thread}
		snap.Session.FocusThreadID = &tid
		snap.ThreadUserTurns[tid] = 3
	}
	if proto := r.protocols[a.ConversationID]; proto != nil && (proto.Status == c.PIActive || proto.Status == c.PIOffered || proto.Status == c.PIPaused) {
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
	if patch := p.State.Protocol; patch != nil {
		current := f.r.protocols[a.ConversationID]
		if patch.Create != nil {
			current = &c.ProtocolInstance{ID: patch.InstanceID, UserID: a.UserID, ThreadID: patch.Create.ThreadID, ProtocolID: patch.Create.ProtocolID, ProtocolVersion: patch.Create.Version, Status: c.PIOffered, Inputs: map[string]string{}}
		}
		if current != nil {
			next := *current
			next.Version++
			if patch.NewStatus != nil {
				next.Status = *patch.NewStatus
			}
			if patch.NewStep != nil {
				next.StepID = *patch.NewStep
			}
			if patch.ConsentAt != nil {
				next.ConsentAt = patch.ConsentAt
				next.ConsentMessageID = patch.ConsentMessageID
			}
			next.Attempts += patch.AttemptsInc
			next.StuckCount += patch.StuckInc
			if patch.StuckReset {
				next.StuckCount = 0
			}
			next.Inputs = map[string]string{}
			for k, v := range current.Inputs {
				next.Inputs[k] = v
			}
			for k, v := range patch.Inputs {
				next.Inputs[k] = v
			}
			f.r.protocols[a.ConversationID] = &next
		}
	}
	if p.Decisions != nil {
		f.r.carry[a.ConversationID] = p.Decisions.Carry
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

type testIntegrationHarness struct {
	s          *store
	r          *testIntegrationReader
	user, conv c.ID
	registry   testIntegrationRegistry
}

func newTestIntegrationHarness(t *testing.T) *testIntegrationHarness {
	t.Helper()
	h := &testIntegrationHarness{s: newStore(), user: c.NewID(), conv: c.NewID(), registry: testIntegrationRegistry{&testIntegrationBundle{files: map[string][]byte{}}}}
	h.s.provision(h.user, h.conv)
	h.r = newTestIntegrationReader(h.s)
	return h
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

func (b *testIntegrationBundle) ArtifactNames() []string {
	out := []string{}
	for name := range b.files {
		out = append(out, name)
	}
	return out
}
