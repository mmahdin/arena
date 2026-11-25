package orchestrator_test

import (
	"context"
	"errors"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"strings"
	"sync"
	"time"
)

type turn struct {
	req      c.TurnRequest
	admit    c.Admission
	result   c.TurnResult
	messages []c.Message
	fenced   bool
	version  int64
}
type store struct {
	mu             sync.Mutex
	users          map[c.ID]bool
	convs          map[c.ID]c.ID
	deleted        map[c.ID]bool
	turns          map[c.ID]map[c.ID]*turn
	order          []*turn
	active         map[c.ID]*turn
	sessionVersion map[c.ID]int64
	userVersion    map[c.ID]int64
	rejectCommit   bool
	failRenew      bool
	completed      int
	rollbackCount  int
	finalized      []c.PreparedTurn
}

func newStore() *store {
	return &store{users: map[c.ID]bool{}, convs: map[c.ID]c.ID{}, deleted: map[c.ID]bool{}, turns: map[c.ID]map[c.ID]*turn{}, active: map[c.ID]*turn{}, sessionVersion: map[c.ID]int64{}, userVersion: map[c.ID]int64{}}
}
func (s *store) provision(user, conv c.ID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[user] = true
	s.convs[conv] = user
	s.sessionVersion[conv] = 1
	s.userVersion[user] = 1
}
func notFound() error          { return cError(c.ErrForbidden) }
func cError(code c.Code) error { return &c.Error{Code: code} }
func (s *store) visible(user, conv c.ID) bool {
	return s.users[user] && !s.deleted[conv] && s.convs[conv] == user
}
func (s *store) ValidateOwner(_ context.Context, user, conv c.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.visible(user, conv) {
		return notFound()
	}
	return nil
}
func (s *store) byClient(user, client c.ID) *turn {
	if s.turns[user] == nil {
		return nil
	}
	return s.turns[user][client]
}
func (s *store) AdmitUserMessage(_ context.Context, r c.TurnRequest) (c.Admission, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.visible(r.UserID, r.ConversationID) {
		return c.Admission{}, notFound()
	}
	t := s.byClient(r.UserID, r.ClientTurnID)
	duplicate := t != nil
	if t == nil {
		a := c.Admission{UserID: r.UserID, ConversationID: r.ConversationID, TurnID: c.NewID(), UserMessageID: c.NewID(), SessionID: c.NewID()}
		t = &turn{req: r, admit: a, result: c.TurnResult{TurnID: a.TurnID, UserMessageID: a.UserMessageID, Status: c.TurnQueued}}
		t.messages = append(t.messages, c.Message{ID: a.UserMessageID, UserID: r.UserID, ConvID: r.ConversationID, TurnID: a.TurnID, Role: c.RoleUser, Content: r.Text})
		if s.turns[r.UserID] == nil {
			s.turns[r.UserID] = map[c.ID]*turn{}
		}
		s.turns[r.UserID][r.ClientTurnID] = t
		s.order = append(s.order, t)
	} else {
		if t.req.ConversationID != r.ConversationID || t.req.Text != r.Text {
			return c.Admission{}, cError(c.ErrDuplicate)
		}
	}
	a := t.admit
	if t.result.Status == c.TurnCompleted || t.result.Status == c.TurnFailed || t.result.Status == c.TurnCancelled {
		a.DuplicateOf = &a.TurnID
		a.LeaseToken = c.ID{}
		return a, nil
	}
	freshAcquisition := false
	if s.active[r.UserID] == nil {
		var earliest *turn
		for _, candidate := range s.order {
			if candidate.req.UserID == r.UserID && (candidate.result.Status == c.TurnQueued || candidate.result.Status == c.TurnRunning) {
				earliest = candidate
				break
			}
		}
		if earliest == t {
			freshAcquisition = true
			t.fenced = true
			s.active[r.UserID] = t
			t.admit.LeaseToken = c.NewID()
			t.admit.Versions = c.Versions{Session: s.sessionVersion[r.ConversationID], User: s.userVersion[r.UserID]}
			t.result.Status = c.TurnRunning
		}
	}
	a = t.admit
	if duplicate {
		a.DuplicateOf = &a.TurnID
	}
	// A replay of a live turn NEVER inherits its worker's fencing token.
	if t != s.active[r.UserID] || (duplicate && t.fenced && s.active[r.UserID] == t && t.admit.LeaseToken == a.LeaseToken && !freshAcquisition) {
		a.Queued = true
		a.LeaseToken = c.ID{}
	}
	return a, nil
}
func (s *store) Recent(_ context.Context, u, conv c.ID, limit int) ([]c.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.visible(u, conv) {
		return nil, notFound()
	}
	var list []c.Message
	for _, t := range s.order {
		if t.req.UserID == u && t.req.ConversationID == conv {
			list = append(list, t.messages...)
		}
	}
	if len(list) > limit {
		list = list[len(list)-limit:]
	}
	return append([]c.Message{}, list...), nil
}
func (s *store) GetTurn(_ context.Context, user, client c.ID) (c.TurnResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := s.byClient(user, client)
	if t == nil || !s.visible(user, t.req.ConversationID) {
		return c.TurnResult{}, notFound()
	}
	return t.result, nil
}
func (s *store) check(a c.Admission) (*turn, error) {
	for _, t := range s.order {
		if t.admit.TurnID == a.TurnID && s.visible(a.UserID, a.ConversationID) && t.admit.UserID == a.UserID && t.admit.ConversationID == a.ConversationID && t.admit.UserMessageID == a.UserMessageID && t.admit.LeaseToken == a.LeaseToken && t.fenced && s.active[a.UserID] == t && t.result.Status == c.TurnRunning {
			return t, nil
		}
	}
	return nil, cError(c.ErrStateConflict)
}
func (s *store) Complete(_ context.Context, a c.Admission, p c.PreparedTurn) (c.TurnResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, err := s.check(a)
	if err != nil {
		return c.TurnResult{}, err
	}
	if s.rejectCommit {
		s.rejectCommit = false
		s.rollbackCount++
		return c.TurnResult{}, errors.New("simulated SQL transaction rollback")
	}
	if p.State.ExpectedVersions.User != s.userVersion[a.UserID] || p.State.ExpectedVersions.Session != s.sessionVersion[a.ConversationID] {
		return c.TurnResult{}, cError(c.ErrStateConflict)
	}
	if p.State.UserID != a.UserID || p.State.TurnID != a.TurnID || p.UserMessageID != a.UserMessageID || p.TurnID != a.TurnID || !p.Route.Valid() || !p.Objective.Valid() || strings.TrimSpace(p.Text) == "" {
		return c.TurnResult{}, cError(c.ErrValidation)
	}
	mid := c.NewID()
	t.messages = append(t.messages, c.Message{ID: mid, UserID: a.UserID, ConvID: a.ConversationID, TurnID: a.TurnID, Role: c.RoleAssistant, Content: p.Text})
	t.result = c.TurnResult{TurnID: a.TurnID, UserMessageID: a.UserMessageID, AssistantMessageID: &mid, Text: p.Text, Status: c.TurnCompleted, Route: p.Route, Degraded: p.Degraded, Safety: p.Safety}
	s.sessionVersion[a.ConversationID]++
	s.userVersion[a.UserID]++
	s.finalized = append(s.finalized, p)
	s.completed++
	t.fenced = false
	delete(s.active, a.UserID)
	return t.result, nil
}
func (s *store) Fail(_ context.Context, a c.Admission, code c.Code) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, err := s.check(a)
	if err != nil {
		return err
	}
	if !code.Valid() {
		return cError(c.ErrValidation)
	}
	t.result.Status = c.TurnFailed
	t.fenced = false
	delete(s.active, a.UserID)
	return nil
}
func (s *store) Cancel(_ context.Context, a c.Admission) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, err := s.check(a)
	if err != nil {
		return err
	}
	t.result.Status = c.TurnCancelled
	t.fenced = false
	delete(s.active, a.UserID)
	return nil
}
func (s *store) Renew(_ context.Context, h ports.LeaseHandle) (ports.LeaseHandle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := s.active[h.UserID]
	if s.failRenew || t == nil || !t.fenced || t.admit.TurnID != h.TurnID || t.admit.LeaseToken != h.Token {
		return h, cError(c.ErrStateConflict)
	}
	h.ExpiresAt = time.Now().Add(time.Minute)
	return h, nil
}
func (s *store) Retry(_ context.Context, user, client c.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := s.byClient(user, client)
	if t == nil || !s.visible(user, t.req.ConversationID) {
		return notFound()
	}
	if t.result.Status != c.TurnFailed && t.result.Status != c.TurnCancelled {
		return cError(c.ErrStateConflict)
	}
	t.result.Status = c.TurnQueued
	t.admit.LeaseToken = c.ID{}
	return nil
}
func (s *store) CancelQueued(_ context.Context, user, client c.ID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := s.byClient(user, client)
	if t == nil || t.result.Status != c.TurnQueued {
		return cError(c.ErrStateConflict)
	}
	t.result.Status = c.TurnCancelled
	return nil
}
func (s *store) Next(_ context.Context, user c.ID) (c.TurnRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.order {
		if t.req.UserID == user && s.visible(user, t.req.ConversationID) && (t.result.Status == c.TurnQueued || t.result.Status == c.TurnRunning) {
			return t.req, nil
		}
	}
	return c.TurnRequest{}, notFound()
}
func (s *store) expire(user c.ID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t := s.active[user]; t != nil {
		t.fenced = false
	}
	delete(s.active, user)
}
func (s *store) softHideConversation(conv c.ID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleted[conv] = true
}
func (s *store) snapshot() (int, int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, t := range s.order {
		n += len(t.messages)
	}
	return n, s.completed, s.rollbackCount
}
func (s *store) status(user, client c.ID) (c.TurnStatus, c.ID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := s.byClient(user, client)
	return t.result.Status, t.admit.UserMessageID
}
