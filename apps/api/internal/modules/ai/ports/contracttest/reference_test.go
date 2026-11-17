package contracttest_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports/contracttest"
)

// This reference double exists only in tests. It is not an AI implementation,
// fallback, durable store, or evidence of SQL transaction correctness.
type record struct {
	req       contract.TurnRequest
	admission contract.Admission
	result    contract.TurnResult
}
type reference struct {
	mu       sync.Mutex
	now      time.Time
	owners   map[contract.ID]contract.ID
	turns    map[[2]contract.ID]*record
	messages map[contract.ID][]contract.Message
	leases   map[contract.ID]ports.LeaseHandle
	failNext bool
}

var (
	errDenied                           = errors.New("resource unavailable")
	errConflict                         = errors.New("turn conflict")
	_           ports.ConversationStore = (*reference)(nil)
	_           ports.TurnFinalizer     = (*reference)(nil)
	_           ports.TurnSerialiser    = (*reference)(nil)
)

func fixture(t *testing.T) contracttest.Fixture {
	t.Helper()
	f := contracttest.Fixture{UserA: contract.NewID(), UserB: contract.NewID(), ConversationA: contract.NewID(), OtherConversationA: contract.NewID(), ConversationB: contract.NewID()}
	r := &reference{now: time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC), owners: map[contract.ID]contract.ID{f.ConversationA: f.UserA, f.OtherConversationA: f.UserA, f.ConversationB: f.UserB}, turns: map[[2]contract.ID]*record{}, messages: map[contract.ID][]contract.Message{}, leases: map[contract.ID]ports.LeaseHandle{}}
	f.Store, f.Finalizer, f.Serialiser = r, r, r
	f.AdvancePastLease = func() { r.mu.Lock(); defer r.mu.Unlock(); r.now = r.now.Add(2 * time.Minute) }
	f.FailNextCompletion = func() { r.mu.Lock(); defer r.mu.Unlock(); r.failNext = true }
	return f
}
func (r *reference) ValidateOwner(ctx context.Context, user, conv contract.ID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if !contract.ValidID(user) || r.owners[conv] != user {
		return errDenied
	}
	return nil
}
func (r *reference) AdmitUserMessage(ctx context.Context, req contract.TurnRequest) (contract.Admission, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return contract.Admission{}, err
	}
	if !contract.ValidID(req.UserID) || r.owners[req.ConversationID] != req.UserID {
		return contract.Admission{}, errDenied
	}
	key := [2]contract.ID{req.UserID, req.ClientTurnID}
	if old := r.turns[key]; old != nil {
		if old.req.Text != req.Text || old.req.ConversationID != req.ConversationID {
			return contract.Admission{}, errConflict
		}
		a := old.admission
		id := a.TurnID
		a.DuplicateOf = &id
		return a, nil
	}
	turn := contract.NewID()
	lease, err := r.acquire(req.UserID, turn)
	if err != nil {
		return contract.Admission{}, err
	}
	a := contract.Admission{UserID: req.UserID, ConversationID: req.ConversationID, TurnID: turn, UserMessageID: contract.NewID(), SessionID: contract.NewID(), LeaseToken: lease.Token}
	r.turns[key] = &record{req: req, admission: a, result: contract.TurnResult{TurnID: turn, UserMessageID: a.UserMessageID, Status: contract.TurnRunning}}
	r.messages[req.ConversationID] = append(r.messages[req.ConversationID], contract.Message{ID: a.UserMessageID, UserID: req.UserID, ConvID: req.ConversationID, TurnID: turn, Role: contract.RoleUser, Content: req.Text, CreatedAt: r.now})
	return a, nil
}
func (r *reference) Recent(ctx context.Context, user, conv contract.ID, limit int) ([]contract.Message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !contract.ValidID(user) || r.owners[conv] != user {
		return nil, errDenied
	}
	if limit < 1 {
		return nil, errConflict
	}
	items := r.messages[conv]
	if len(items) > limit {
		items = items[len(items)-limit:]
	}
	return append([]contract.Message{}, items...), nil
}
func (r *reference) GetTurn(ctx context.Context, user, client contract.ID) (contract.TurnResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return contract.TurnResult{}, err
	}
	rec := r.turns[[2]contract.ID{user, client}]
	if rec == nil {
		return contract.TurnResult{}, errDenied
	}
	out := rec.result
	if out.AssistantMessageID != nil {
		id := *out.AssistantMessageID
		out.AssistantMessageID = &id
	}
	return out, nil
}
func (r *reference) admitted(a contract.Admission) (*record, error) {
	lease, ok := r.leases[a.UserID]
	if !ok || lease.TurnID != a.TurnID || lease.Token != a.LeaseToken || !lease.ExpiresAt.After(r.now) {
		return nil, errConflict
	}
	for _, rec := range r.turns {
		if rec.admission.TurnID == a.TurnID && rec.req.UserID == a.UserID && rec.req.ConversationID == a.ConversationID && rec.admission.UserMessageID == a.UserMessageID {
			return rec, nil
		}
	}
	return nil, errDenied
}
func (r *reference) Complete(ctx context.Context, a contract.Admission, p contract.PreparedTurn) (contract.TurnResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return contract.TurnResult{}, err
	}
	rec, err := r.admitted(a)
	if err != nil {
		return contract.TurnResult{}, err
	}
	if p.TurnID != a.TurnID || p.UserMessageID != a.UserMessageID || p.State.UserID != a.UserID || p.State.TurnID != a.TurnID || p.Text == "" {
		return contract.TurnResult{}, errConflict
	}
	if r.failNext {
		r.failNext = false
		return contract.TurnResult{}, &contract.Error{Code: contract.ErrStateCommit}
	}
	id := contract.NewID()
	r.messages[a.ConversationID] = append(r.messages[a.ConversationID], contract.Message{ID: id, UserID: a.UserID, ConvID: a.ConversationID, TurnID: a.TurnID, Role: contract.RoleAssistant, Content: p.Text, CreatedAt: r.now})
	rec.result = contract.TurnResult{TurnID: a.TurnID, UserMessageID: a.UserMessageID, AssistantMessageID: &id, Text: p.Text, Status: contract.TurnCompleted, Route: p.Route, Degraded: p.Degraded}
	delete(r.leases, a.UserID)
	return rec.result, nil
}
func (r *reference) Fail(ctx context.Context, a contract.Admission, code contract.Code) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	rec, err := r.admitted(a)
	if err != nil {
		return err
	}
	if !code.Valid() {
		return errConflict
	}
	rec.result.Status = contract.TurnFailed
	delete(r.leases, a.UserID)
	return nil
}
func (r *reference) acquire(user, turn contract.ID) (ports.LeaseHandle, error) {
	if lease, ok := r.leases[user]; ok && lease.ExpiresAt.After(r.now) {
		return ports.LeaseHandle{}, errConflict
	}
	lease := ports.LeaseHandle{UserID: user, TurnID: turn, Token: contract.NewID(), ExpiresAt: r.now.Add(time.Minute)}
	r.leases[user] = lease
	return lease, nil
}
func (r *reference) Acquire(ctx context.Context, user, turn contract.ID) (ports.LeaseHandle, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return ports.LeaseHandle{}, err
	}
	return r.acquire(user, turn)
}
func (r *reference) Release(ctx context.Context, lease ports.LeaseHandle) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	current, ok := r.leases[lease.UserID]
	if !ok || current.Token != lease.Token || current.TurnID != lease.TurnID {
		return errConflict
	}
	delete(r.leases, lease.UserID)
	return nil
}

func TestReferenceContract(t *testing.T) {
	contracttest.CheckLifecycle(t, fixture)
	contracttest.CheckLeases(t, fixture)
}
