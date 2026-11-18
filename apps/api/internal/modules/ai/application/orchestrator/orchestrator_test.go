package orchestrator_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/gateway"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/orchestrator"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
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

// The WP-04 Client port is exercised in two distinct calls, using a purely
// synthetic schema and scripted output, NOT production clinical prompts.
type scriptedClient struct {
	mu      sync.Mutex
	calls   []string
	text    string
	failure error
}

func (l *scriptedClient) Structured(ctx context.Context, req ports.StructuredRequest) (ports.StructuredResponse, error) {
	l.mu.Lock()
	l.calls = append(l.calls, "structured:"+string(req.Tier))
	l.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return ports.StructuredResponse{}, err
	}
	return ports.StructuredResponse{JSON: []byte(`{"objective":"reflect"}`), Usage: c.LLMUsage{InTokens: 2, OutTokens: 1}}, nil
}
func (l *scriptedClient) Stream(ctx context.Context, req ports.StreamRequest) (ports.Stream, error) {
	l.mu.Lock()
	l.calls = append(l.calls, "stream:"+string(req.Tier))
	l.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if l.failure != nil {
		return nil, l.failure
	}
	return &scriptedStream{ctx: ctx, text: l.text}, nil
}
func (l *scriptedClient) history() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string{}, l.calls...)
}

type scriptedStream struct {
	ctx  context.Context
	text string
	n    int
}

func (s *scriptedStream) Recv() (ports.Chunk, error) {
	if err := s.ctx.Err(); err != nil {
		return ports.Chunk{}, err
	}
	s.n++
	if s.n == 1 {
		return ports.Chunk{Text: s.text}, nil
	}
	if s.n == 2 {
		return ports.Chunk{FinishReason: "stop", Usage: &c.LLMUsage{InTokens: 3, OutTokens: 4}}, nil
	}
	return ports.Chunk{}, io.EOF
}
func (s *scriptedStream) Close() error { return nil }

type brain struct {
	client  ports.Client
	user    c.ID
	fail    error
	enter   chan struct{}
	proceed chan struct{}
	calls   int
	mu      sync.Mutex
	raise   c.RiskLevel
}

func (b *brain) Decide(ctx context.Context, in ports.BrainInput) (ports.BrainOutput, error) {
	b.mu.Lock()
	b.calls++
	b.mu.Unlock()
	if b.enter != nil {
		select {
		case b.enter <- struct{}{}:
		default:
		}
	}
	if b.proceed != nil {
		select {
		case <-b.proceed:
		case <-ctx.Done():
			return ports.BrainOutput{}, ctx.Err()
		}
	}
	if b.fail != nil {
		return ports.BrainOutput{}, b.fail
	}
	raw, err := b.client.Structured(ctx, ports.StructuredRequest{Tier: ports.TierSmall, SchemaName: "wp05.synthetic", Schema: []byte(`{"type":"object","properties":{"objective":{"const":"reflect"}},"required":["objective"],"additionalProperties":false}`), MaxTokens: 16, Meta: c.CallMeta{UserID: b.user, TurnID: in.WCtx.Turn.TurnID}})
	if err != nil {
		return ports.BrainOutput{}, err
	}
	var choice struct {
		Objective c.Objective `json:"objective"`
	}
	if err = json.Unmarshal(raw.JSON, &choice); err != nil {
		return ports.BrainOutput{}, err
	}
	dec := c.BrainDecision{SchemaVersion: "brain.v1", Decision: c.Decision{Primary: choice.Objective}}
	if b.raise != "" {
		dec.RiskRead = &c.BrainRiskRead{Level: b.raise}
	}
	return ports.BrainOutput{Decision: dec, Usage: raw.Usage}, nil
}
func (b *brain) count() int { b.mu.Lock(); defer b.mu.Unlock(); return b.calls }

type generator struct {
	client ports.Client
	user   c.ID
	calls  int
	mu     sync.Mutex
}

func (g *generator) Generate(ctx context.Context, in ports.GenInput, sink ports.TokenSink) (ports.GenOutput, error) {
	g.mu.Lock()
	g.calls++
	g.mu.Unlock()
	stream, err := g.client.Stream(ctx, ports.StreamRequest{Tier: ports.TierStrong, MaxTokens: 100, Meta: c.CallMeta{UserID: g.user, TurnID: in.Turn.TurnID}})
	if err != nil {
		return ports.GenOutput{}, err
	}
	defer stream.Close()
	var out strings.Builder
	var usage c.LLMUsage
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return ports.GenOutput{}, err
		}
		if chunk.Usage != nil {
			usage = *chunk.Usage
		}
		if chunk.Text != "" {
			if err = sink.Send(chunk.Text); err != nil {
				return ports.GenOutput{}, err
			}
			out.WriteString(chunk.Text)
		}
	}
	return ports.GenOutput{Text: out.String(), Usage: usage}, nil
}
func (g *generator) count() int { g.mu.Lock(); defer g.mu.Unlock(); return g.calls }

type safety struct {
	level c.RiskLevel
	err   error
}

func (s safety) Assess(_ context.Context, _ ports.SafetyInput) (c.SafetySignal, error) {
	return c.SafetySignal{Level: s.level}, s.err
}

type assembler struct{}

func (assembler) Build(_ context.Context, _ c.TurnRequest, _ c.SafetySignal, _ c.RetrievalResult) (c.WorkingContext, ports.StateSnapshot, error) {
	return c.WorkingContext{Risk: c.RiskView{Current: c.RiskLow}}, ports.StateSnapshot{}, nil
}

type validator struct {
	mu           sync.Mutex
	rejectFirst  bool
	alwaysReject bool
}

func (*validator) StreamGuard(_ c.ResponsePlan, _ c.RiskLevel) ports.StreamGuard { return nil }
func (v *validator) Finalize(_ context.Context, _ string, _ c.ResponsePlan, _ c.WorkingContext, _ c.RiskLevel, _ *c.CrisisPlan) (c.ValidationReport, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.alwaysReject || v.rejectFirst {
		v.rejectFirst = false
		return c.ValidationReport{Action: c.VAFallback}, nil
	}
	return c.ValidationReport{Passed: true, Action: c.VAAccept}, nil
}

type crisisPlanner struct{}

func (crisisPlanner) Plan(_ context.Context, _ c.SafetySignal, _ string, _ c.RiskState) (c.CrisisPlan, error) {
	return c.CrisisPlan{Buffered: true, TemplateText: "Synthetic reviewed emergency placeholder"}, nil
}

type harness struct {
	s                 *store
	user, conv, other c.ID
	g                 *generator
	b                 *brain
	client            *scriptedClient
	tr                *trace.Recorder
	validator         *validator
	o                 *orchestrator.Service
}

func newHarness(t *testing.T, opts orchestrator.Options) *harness {
	t.Helper()
	h := &harness{s: newStore(), user: c.NewID(), conv: c.NewID(), other: c.NewID(), client: &scriptedClient{text: "Synthetic answer"}, tr: trace.NewRecorder(), validator: &validator{}}
	h.s.provision(h.user, h.conv)
	h.s.provision(h.user, h.other)
	h.b = &brain{client: h.client, user: h.user}
	h.g = &generator{client: h.client, user: h.user}
	gw, err := gateway.New(h.s, h.s)
	if err != nil {
		t.Fatal(err)
	}
	if opts.TotalTimeout == 0 {
		opts.TotalTimeout = 5 * time.Second
	}
	if opts.Fallback == nil {
		opts.Fallback = func(_ c.Route, _ c.Objective, _ string) (string, bool) { return "Synthetic reviewed fallback", true }
	}
	h.o, err = orchestrator.New(orchestrator.Dependencies{Gateway: gw, Store: h.s, Finalizer: h.s, Recovery: h.s, Safety: safety{level: c.RiskLow}, Assembler: assembler{}, Brain: h.b, Generator: h.g, Validator: h.validator, Tracer: h.tr, Crisis: crisisPlanner{}}, opts)
	if err != nil {
		t.Fatal(err)
	}
	return h
}
func (h *harness) request(conv c.ID) c.TurnRequest {
	return c.TurnRequest{UserID: h.user, ConversationID: conv, ClientTurnID: c.NewID(), Text: "synthetic content", Locale: "en", TZ: "UTC", Channel: "app_chat", Origin: c.OriginUser}
}
func (h *harness) mustNew(t *testing.T, s safety, opts orchestrator.Options) {
	t.Helper()
	gw, _ := gateway.New(h.s, h.s)
	o, err := orchestrator.New(orchestrator.Dependencies{Gateway: gw, Store: h.s, Finalizer: h.s, Recovery: h.s, Safety: s, Assembler: assembler{}, Brain: h.b, Generator: h.g, Validator: h.validator, Tracer: h.tr, Crisis: crisisPlanner{}}, opts)
	if err != nil {
		t.Fatal(err)
	}
	h.o = o
}

func TestEndToEndTwoStageAndIdempotentReplay(t *testing.T) {
	h := newHarness(t, orchestrator.Options{})
	req := h.request(h.conv)
	result, err := h.o.ProcessTurn(context.Background(), req)
	if err != nil || result.Status != c.TurnCompleted || result.AssistantMessageID == nil || result.Text != "Synthetic answer" {
		t.Fatalf("first turn %+v %v", result, err)
	}
	n, completed, _ := h.s.snapshot()
	if n != 2 || completed != 1 {
		t.Fatal("not exactly two durable messages", n, completed)
	}
	if got := h.client.history(); len(got) != 2 || got[0] != "structured:small" || got[1] != "stream:strong" {
		t.Fatalf("not WP-04 two-stage call sequence: %v", got)
	}
	replay, err := h.o.ProcessTurn(context.Background(), req)
	if err != nil || replay != result || len(h.client.history()) != 2 {
		t.Fatalf("replay changed result or reinferred: %+v %v", replay, err)
	}
	snap, ok := h.tr.Snapshot(result.TurnID)
	if !ok || !snap.Finished || snap.Route != c.RouteNormal || snap.Usage.InTokens != 5 || snap.Usage.OutTokens != 5 {
		t.Fatalf("trace metadata missing: %+v", snap)
	}
	blob, _ := json.Marshal(snap)
	if strings.Contains(string(blob), req.Text) || strings.Contains(string(blob), result.Text) {
		t.Fatal("trace exposed raw content")
	}
	if len(h.s.finalized) != 1 || h.s.finalized[0].State.Session.TurnCountInc != 1 || h.s.finalized[0].Trace.TurnID != result.TurnID {
		t.Fatal("state/trace not prepared atomically")
	}
}

func TestPerUserCrossConversationQueueAndRecovery(t *testing.T) {
	h := newHarness(t, orchestrator.Options{})
	h.b.enter = make(chan struct{}, 1)
	h.b.proceed = make(chan struct{})
	one := h.request(h.conv)
	two := h.request(h.other)
	done := make(chan error, 1)
	go func() { _, err := h.o.ProcessTurn(context.Background(), one); done <- err }()
	select {
	case <-h.b.enter:
	case <-time.After(time.Second):
		t.Fatal("first turn did not reach brain")
	}
	result, err := h.o.ProcessTurn(context.Background(), two)
	if err != nil || result.Status != c.TurnQueued || h.b.count() != 1 {
		t.Fatalf("second conversation not queued: %+v %v", result, err)
	}
	n, _, _ := h.s.snapshot()
	if n != 2 {
		t.Fatal("queued raw input was not persisted")
	}
	close(h.b.proceed)
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	result, err = h.o.RecoverNext(context.Background(), h.user)
	if err != nil || result.Status != c.TurnCompleted || result.UserMessageID == (c.ID{}) {
		t.Fatalf("queued turn did not resume: %+v %v", result, err)
	}
	n, completed, _ := h.s.snapshot()
	if n != 4 || completed != 2 {
		t.Fatal("queued recovery duplicate or lost messages", n, completed)
	}
}

func TestFailedUserOnlyReplayAndExplicitRetry(t *testing.T) {
	h := newHarness(t, orchestrator.Options{Fallback: func(_ c.Route, _ c.Objective, _ string) (string, bool) { return "", false }})
	h.client.failure = errors.New("synthetic generation failure")
	req := h.request(h.conv)
	r, err := h.o.ProcessTurn(context.Background(), req)
	if err == nil || r.Status != c.TurnFailed || r.AssistantMessageID != nil {
		t.Fatalf("unsafe assistant persisted: %+v %v", r, err)
	}
	n, _, _ := h.s.snapshot()
	if n != 1 {
		t.Fatal("failed turn lost or duplicated user message")
	}
	callsBeforeReplay := len(h.client.history())
	replay, replayErr := h.o.ProcessTurn(context.Background(), req)
	if replayErr != nil || replay.Status != c.TurnFailed || len(h.client.history()) != callsBeforeReplay {
		t.Fatal("terminal failure replay invoked model")
	}
	h.client.failure = nil
	retryErr := h.o.RetryFailed(context.Background(), req.UserID, req.ClientTurnID)
	if retryErr != nil {
		t.Fatal(retryErr)
	}
	res, retryErr := h.o.RecoverNext(context.Background(), req.UserID)
	if retryErr != nil || res.Status != c.TurnCompleted || res.UserMessageID != r.UserMessageID {
		t.Fatalf("authorized retry lost durable admission: %+v %v", res, retryErr)
	}
}

func TestFinalTransactionRollbackAndAuthorizedRetry(t *testing.T) {
	h := newHarness(t, orchestrator.Options{})
	h.s.rejectCommit = true
	req := h.request(h.conv)
	r, err := h.o.ProcessTurn(context.Background(), req)
	if err == nil || r.Status != c.TurnFailed {
		t.Fatalf("late transaction failed to abort: %+v %v", r, err)
	}
	n, completed, rollbacks := h.s.snapshot()
	if n != 1 || completed != 0 || rollbacks != 1 {
		t.Fatal("partial committed assistant/state on rollback")
	}
	r, err = h.o.ProcessTurn(context.Background(), req)
	if err != nil || r.Status != c.TurnFailed {
		t.Fatal("terminal replay silently retried")
	}
	if err = h.o.RetryFailed(context.Background(), req.UserID, req.ClientTurnID); err != nil {
		t.Fatal(err)
	}
	r, err = h.o.RecoverNext(context.Background(), req.UserID)
	if err != nil || r.Status != c.TurnCompleted {
		t.Fatalf("explicit retry failed: %+v %v", r, err)
	}
	n, completed, _ = h.s.snapshot()
	if n != 2 || completed != 1 || r.UserMessageID == (c.ID{}) {
		t.Fatal("retry inserted duplicate input or state", n, completed)
	}
}
func TestCancellationMarksDurableCancelledWithoutReply(t *testing.T) {
	h := newHarness(t, orchestrator.Options{})
	h.b.enter = make(chan struct{}, 1)
	h.b.proceed = make(chan struct{})
	req := h.request(h.conv)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan c.TurnResult, 1)
	go func() { r, _ := h.o.ProcessTurn(ctx, req); done <- r }()
	select {
	case <-h.b.enter:
	case <-time.After(time.Second):
		t.Fatal("not admitted")
	}
	cancel()
	r := <-done
	if r.Status != c.TurnCancelled {
		t.Fatalf("canceled turn not marked cancelled: %+v", r)
	}
	n, _, _ := h.s.snapshot()
	if n != 1 {
		t.Fatal("cancel persisted unsafe assistant")
	}
}
func TestLeaseFencePreventsStaleCompletion(t *testing.T) {
	h := newHarness(t, orchestrator.Options{})
	h.b.enter = make(chan struct{}, 1)
	h.b.proceed = make(chan struct{})
	req := h.request(h.conv)
	done := make(chan error, 1)
	go func() { _, err := h.o.ProcessTurn(context.Background(), req); done <- err }()
	select {
	case <-h.b.enter:
	case <-time.After(time.Second):
		t.Fatal("turn not processing")
	}
	h.s.expire(h.user)
	close(h.b.proceed)
	if err := <-done; err == nil {
		t.Fatal("expired lease permitted finalization")
	}
	n, completed, _ := h.s.snapshot()
	if n != 1 || completed != 0 {
		t.Fatal("stale completion wrote assistant")
	}
	res, err := h.o.RecoverNext(context.Background(), h.user)
	if err != nil || res.Status != c.TurnCompleted {
		t.Fatalf("recover after expired lease: %+v %v", res, err)
	}
}
func TestValidatorFallbackMustAlsoPass(t *testing.T) {
	h := newHarness(t, orchestrator.Options{})
	h.validator.rejectFirst = true
	req := h.request(h.conv)
	r, err := h.o.ProcessTurn(context.Background(), req)
	if err != nil || r.Status != c.TurnCompleted || r.Text != "Synthetic reviewed fallback" || !r.Degraded {
		t.Fatalf("fallback not validated: %+v %v", r, err)
	}
	h2 := newHarness(t, orchestrator.Options{})
	h2.validator.alwaysReject = true
	req = h2.request(h2.conv)
	r, err = h2.o.ProcessTurn(context.Background(), req)
	if err == nil || r.Status != c.TurnFailed || r.AssistantMessageID != nil {
		t.Fatalf("blocked fallback became assistant: %+v %v", r, err)
	}
}
func TestHighRiskSkipsBrainAndRequiresPlanner(t *testing.T) {
	h := newHarness(t, orchestrator.Options{})
	h.mustNew(t, safety{level: c.RiskHigh}, orchestrator.Options{})
	req := h.request(h.conv)
	r, err := h.o.ProcessTurn(context.Background(), req)
	if err != nil || r.Status != c.TurnCompleted || r.Route != c.RouteCrisis || r.Safety == nil || h.b.count() != 0 {
		t.Fatalf("crisis routing incorrect: %+v %v", r, err)
	}
}
func TestDeletedConversationNeverCompletedOrRecovered(t *testing.T) {
	h := newHarness(t, orchestrator.Options{})
	h.b.enter = make(chan struct{}, 1)
	h.b.proceed = make(chan struct{})
	req := h.request(h.conv)
	done := make(chan error, 1)
	go func() { _, err := h.o.ProcessTurn(context.Background(), req); done <- err }()
	select {
	case <-h.b.enter:
	case <-time.After(time.Second):
		t.Fatal("not processing")
	}
	h.s.softHideConversation(h.conv)
	close(h.b.proceed)
	if err := <-done; err == nil {
		t.Fatal("soft-deleted conversation completed")
	}
	n, completed, _ := h.s.snapshot()
	if n != 1 || completed != 0 {
		t.Fatal("soft deletion destroyed record or wrote assistant")
	}
	if _, err := h.o.RecoverNext(context.Background(), h.user); err == nil {
		t.Fatal("deleted conversation entered recovery")
	}
}

func TestSafetyFailureEscalatesAndRecordsDegradation(t *testing.T) {
	h := newHarness(t, orchestrator.Options{})
	h.mustNew(t, safety{level: c.RiskLow, err: errors.New("synthetic safety failure")}, orchestrator.Options{})
	r, err := h.o.ProcessTurn(context.Background(), h.request(h.conv))
	if err != nil || r.Status != c.TurnCompleted || r.Route != c.RouteNormal || !r.Degraded {
		t.Fatalf("safety failure must not become low-risk FAST: %+v %v", r, err)
	}
	p := h.s.finalized[0]
	if p.Trace.TurnID != r.TurnID || p.Route != c.RouteNormal {
		t.Fatalf("durable trace mismatch: %+v", p.Trace)
	}
	snap, ok := h.tr.Snapshot(r.TurnID)
	if !ok {
		t.Fatal("missing metadata trace")
	}
	var sawRoute, sawSafety bool
	for _, step := range snap.Steps {
		if step.Name == "route" && step.RuleID == "RT-099" {
			sawRoute = true
		}
		if step.Name == "safety" && step.Degraded {
			sawSafety = true
		}
	}
	if !sawRoute || !sawSafety {
		t.Fatalf("deterministic rule/degraded safety absent from trace: %+v", snap.Steps)
	}
}

func TestLeaseRenewalConflictCancelsInferenceAndPreservesInput(t *testing.T) {
	h := newHarness(t, orchestrator.Options{RenewInterval: time.Millisecond})
	h.b.enter = make(chan struct{}, 1)
	h.b.proceed = make(chan struct{})
	h.s.failRenew = true
	req := h.request(h.conv)
	result, err := h.o.ProcessTurn(context.Background(), req)
	if err == nil || result.Status != c.TurnFailed || result.AssistantMessageID != nil {
		t.Fatalf("renewal loss published output: %+v %v", result, err)
	}
	messages, _, _ := h.s.snapshot()
	if messages != 1 {
		t.Fatalf("renewal loss removed durable input or persisted a draft: %d", messages)
	}
}
