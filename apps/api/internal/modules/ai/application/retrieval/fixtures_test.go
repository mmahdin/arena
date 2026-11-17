package retrieval

import (
	"context"
	"errors"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// ---- Knowledge Bundle fixture ----
//
// The YAML below is a test-only fixture with the same shape and limits as the
// synthetic WP-03 bundle. It is NOT clinical content and is never loaded from
// knowledge/ at test time, so the tests do not depend on bundle file layout.

const retrievalYAML = `schema: retrieval.v1
defaults:
  max_items: 5
  min_score: 0.32
  per_item_tokens: 60
  hard_cap_with_hop: 7
weights:
  similarity: 0.5
  recency: 0.2
  importance: 0.15
  thread_match: 0.15
recency:
  half_life_days: 21
anaphora:
  recent_window_days: 14
  emotion_boost: 0.15
  topic_boost: 0.1
temporal:
  max_messages: 12
  default_roles:
  - user
  week_starts_on: monday
  tz_source: user_state
hypotheses:
  include_statuses:
  - proposed
  - supported
  require_thread_match: true
novelty_filter:
  skip_if_in_recent_messages: 8
  similarity_gte: 0.92
timeouts:
  total_ms: 300
  per_strategy_ms: 200
`

const budgetsYAML = `schema: budgets.v1
turn:
  total_ms: 20000
  safety_ms: 400
  retrieval_ms: 300
  brain_ms: 2500
  gen_first_token_ms: 3000
  gen_total_ms: 12000
limits:
  max_sync_llm_calls: 3
  max_regenerations: 1
  max_brain_hops: 1
tokens:
  brain_in: 3500
  brain_out: 700
  gen_in: 4500
  gen_out: 600
  wctx_target: 1500
quotas:
  reflection_per_user_per_day: 4
  memory_writer_per_user_per_hour: 30
  user_msgs_per_minute: 12
`

type fakeBundle struct {
	files     map[string][]byte
	synthetic bool
}

func (f fakeBundle) Reference() c.BundleRef {
	return c.BundleRef{Version: "kb@test", PromptSHAs: map[string]string{}}
}

func (f fakeBundle) Artifact(_ context.Context, name string) ([]byte, error) {
	raw, ok := f.files[name]
	if !ok {
		return nil, errors.New("artifact not found")
	}
	return append([]byte(nil), raw...), nil
}

func (f fakeBundle) SyntheticOnly() bool { return f.synthetic }

type fakeRegistry struct {
	mu     sync.Mutex
	bundle ports.Bundle
	err    error
	pins   int
}

func (r *fakeRegistry) Active(context.Context, c.ID) (ports.Bundle, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pins++
	if r.err != nil {
		return nil, r.err
	}
	return r.bundle, nil
}

func testRegistry() *fakeRegistry {
	return &fakeRegistry{bundle: fakeBundle{files: map[string][]byte{
		"config/retrieval.yaml": []byte(retrievalYAML),
		"config/budgets.yaml":   []byte(budgetsYAML),
	}, synthetic: true}}
}

// ---- Store fixture ----

type fakeStore struct {
	mu sync.Mutex

	user     c.ID
	messages []retrievalMessage
	memory   []c.MemoryItem
	vectors  map[c.ID]retrievalVector
	threads  []c.ThreadRecord
	tz       string

	errWindow   error
	errThread   error
	errEntity   error
	errCand     error
	errEmbed    error
	errTZ       error
	calls       map[string]int
	lastWindow  MessageWindowQuery
	lastCand    CandidateQuery
	lastEntity  EntityQuery
}

type retrievalMessage struct {
	row retrievalMessageRow
}

type retrievalMessageRow = struct {
	ID             c.ID
	ConversationID c.ID
	TurnID         c.ID
	Role           c.Role
	Text           string
	CreatedAt      time.Time
	User           c.ID
}

type retrievalVector struct {
	vector []float32
	model  string
	codec  int
}

func newFakeStore(user c.ID) *fakeStore {
	return &fakeStore{user: user, vectors: map[c.ID]retrievalVector{}, calls: map[string]int{}, tz: "UTC"}
}

func (s *fakeStore) addMessage(user c.ID, role c.Role, text string, at time.Time) c.ID {
	id := c.NewID()
	s.messages = append(s.messages, retrievalMessage{row: retrievalMessageRow{
		ID: id, ConversationID: c.NewID(), TurnID: c.NewID(), Role: role, Text: text, CreatedAt: at, User: user,
	}})
	return id
}

func (s *fakeStore) MessagesInWindow(_ context.Context, q MessageWindowQuery) ([]MessageRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls["window"]++
	s.lastWindow = q
	if s.errWindow != nil {
		return nil, s.errWindow
	}
	var out []MessageRow
	for _, m := range s.messages {
		// Owner isolation is implemented here exactly as the SQL adapter does:
		// a record belonging to another user can never be returned.
		if m.row.User != q.UserID {
			continue
		}
		if m.row.CreatedAt.Before(q.From) || !m.row.CreatedAt.Before(q.To) {
			continue
		}
		roleOK := false
		for _, r := range q.Roles {
			if m.row.Role == r {
				roleOK = true
			}
		}
		if !roleOK {
			continue
		}
		out = append(out, MessageRow{ID: m.row.ID, ConversationID: m.row.ConversationID, TurnID: m.row.TurnID,
			Role: m.row.Role, Text: m.row.Text, CreatedAt: m.row.CreatedAt})
		if len(out) == q.Limit {
			break
		}
	}
	return out, nil
}

func (s *fakeStore) RecentThreadMessages(_ context.Context, userID, conversationID c.ID, limit int) ([]MessageRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []MessageRow
	for _, m := range s.messages {
		if m.row.User != userID || m.row.ConversationID != conversationID {
			continue
		}
		out = append(out, MessageRow{ID: m.row.ID, ConversationID: m.row.ConversationID, TurnID: m.row.TurnID,
			Role: m.row.Role, Text: m.row.Text, CreatedAt: m.row.CreatedAt})
	}
	if len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out, nil
}

func (s *fakeStore) MemoryByThread(_ context.Context, userID, threadID c.ID, kinds []c.MemoryKind, limit int) ([]c.MemoryItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls["by_thread"]++
	if s.errThread != nil {
		return nil, s.errThread
	}
	var out []c.MemoryItem
	for _, item := range s.memory {
		if item.UserID != userID {
			continue
		}
		linked := false
		for _, id := range item.ThreadIDs {
			if id == threadID {
				linked = true
			}
		}
		if !linked || !kindAllowed(item.Kind, kinds) {
			continue
		}
		out = append(out, item)
		if len(out) == limit {
			break
		}
	}
	return out, nil
}

func (s *fakeStore) MemoryByEntity(_ context.Context, q EntityQuery) ([]c.MemoryItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls["by_entity"]++
	s.lastEntity = q
	if s.errEntity != nil {
		return nil, s.errEntity
	}
	var out []c.MemoryItem
	for _, item := range s.memory {
		if item.UserID != q.UserID {
			continue
		}
		for _, ref := range item.Entities {
			if ref.Role != q.Role {
				continue
			}
			if q.Label != "" && (ref.Label == nil || *ref.Label != q.Label) {
				continue
			}
			out = append(out, item)
			break
		}
		if len(out) == q.Limit {
			break
		}
	}
	return out, nil
}

func (s *fakeStore) MemoryCandidates(_ context.Context, q CandidateQuery) ([]c.MemoryItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls["candidates"]++
	s.lastCand = q
	if s.errCand != nil {
		return nil, s.errCand
	}
	var out []c.MemoryItem
	for _, item := range s.memory {
		if item.UserID != q.UserID {
			continue // the SQL adapter filters owner first; so does this double
		}
		if !kindAllowed(item.Kind, q.Kinds) || !statusAllowed(item.Status, q.Statuses) {
			continue
		}
		if q.MinImport > 0 && item.Importance < q.MinImport {
			continue
		}
		if !q.Since.IsZero() && item.UpdatedAt.Before(q.Since) && item.CreatedAt.Before(q.Since) {
			continue
		}
		if len(q.Terms) > 0 {
			hit := false
			low := strings.ToLower(item.Content)
			for _, t := range q.Terms {
				if strings.Contains(low, strings.ToLower(t)) {
					hit = true
					break
				}
			}
			if !hit && len(q.Terms) > 0 {
				// The double keeps non-lexical recent/important items too, as
				// §7.3 allows the pool to be the union of recent + lexical.
				if item.Importance < 0.5 {
					continue
				}
			}
		}
		out = append(out, item)
		if len(out) == q.Limit {
			break
		}
	}
	return out, nil
}

func (s *fakeStore) MemoryByIDs(_ context.Context, userID c.ID, ids []c.ID) ([]c.MemoryItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	want := map[c.ID]bool{}
	for _, id := range ids {
		want[id] = true
	}
	var out []c.MemoryItem
	for _, item := range s.memory {
		if item.UserID == userID && want[item.ID] {
			out = append(out, item)
		}
	}
	return out, nil
}

func (s *fakeStore) Embeddings(_ context.Context, userID c.ID, refs []EmbeddingRef) ([]StoredEmbedding, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls["embeddings"]++
	if s.errEmbed != nil {
		return nil, s.errEmbed
	}
	var out []StoredEmbedding
	for _, ref := range refs {
		v, ok := s.vectors[ref.ID]
		if !ok {
			continue // a missing embedding is simply absent
		}
		out = append(out, StoredEmbedding{RefID: ref.ID, IsMessage: ref.IsMsg, ModelID: v.model,
			CodecVer: v.codec, Dimensions: len(v.vector), Vector: v.vector})
	}
	return out, nil
}

func (s *fakeStore) Thread(_ context.Context, userID, threadID c.ID) (c.ThreadRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls["thread"]++
	if s.errThread != nil {
		return c.ThreadRecord{}, s.errThread
	}
	for _, t := range s.threads {
		if t.ID == threadID && t.UserID == userID {
			return t, nil
		}
	}
	return c.ThreadRecord{}, ErrNotVisible
}

func (s *fakeStore) ActiveThreads(_ context.Context, userID c.ID, limit int) ([]c.ThreadRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []c.ThreadRecord
	for _, t := range s.threads {
		if t.UserID != userID || t.Phase == c.PhaseResolved || t.Phase == c.PhaseDormant {
			continue
		}
		out = append(out, t)
		if len(out) == limit {
			break
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Priority > out[j].Priority })
	return out, nil
}

func (s *fakeStore) UserTimeZone(_ context.Context, userID c.ID) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.errTZ != nil {
		return "", s.errTZ
	}
	if userID != s.user {
		return "", nil
	}
	return s.tz, nil
}

func kindAllowed(kind c.MemoryKind, kinds []c.MemoryKind) bool {
	if len(kinds) == 0 {
		return true
	}
	for _, k := range kinds {
		if k == kind {
			return true
		}
	}
	return false
}

func statusAllowed(status c.MemoryStatus, statuses []c.MemoryStatus) bool {
	if len(statuses) == 0 {
		return true
	}
	for _, s := range statuses {
		if s == status {
			return true
		}
	}
	return false
}

// fakeEmbedder is a deterministic provider double. It maps a tiny vocabulary to
// unit vectors so cosine similarity is exact and reproducible.
type fakeEmbedder struct {
	mu     sync.Mutex
	err    error
	calls  int
	model  string
	dims   int
	delay  time.Duration
}

func newEmbedder() *fakeEmbedder { return &fakeEmbedder{model: "test-embed", dims: 3} }

func (e *fakeEmbedder) Embed(ctx context.Context, text string) ([]float32, string, error) {
	e.mu.Lock()
	e.calls++
	err, model := e.err, e.model
	delay := e.delay
	e.mu.Unlock()
	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, "", ctx.Err()
		}
	}
	if err != nil {
		return nil, "", err
	}
	low := strings.ToLower(text)
	// Three orthogonal axes: "sister/family", "work/boss", "calm/sleep".
	var v [3]float64
	for _, word := range strings.FieldsFunc(low, func(r rune) bool { return r < 'a' || r > 'z' }) {
		switch word {
		case "sister", "family", "mother", "mom", "brother":
			v[0] += 1
		case "boss", "work", "deadline", "job", "manager":
			v[1] += 1
		case "calm", "sleep", "bed", "rest":
			v[2] += 1
		}
	}
	if v[0] == 0 && v[1] == 0 && v[2] == 0 {
		v[0] = 0.2
	}
	var norm float64
	for _, x := range v {
		norm += x * x
	}
	norm = math.Sqrt(norm)
	if norm == 0 {
		norm = 1
	}
	return []float32{float32(v[0] / norm), float32(v[1] / norm), float32(v[2] / norm)}, model, nil
}

// fixture bundles a service with its doubles.
type fixture struct {
	svc      *Service
	store    *fakeStore
	registry *fakeRegistry
	embedder *fakeEmbedder
	now      time.Time
	user     c.ID
}

func newFixture(t interface{ Helper() }, opts ...Option) *fixture {
	t.Helper()
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	user := c.NewID()
	store := newFakeStore(user)
	registry := testRegistry()
	embedder := newEmbedder()
	svc, err := New(store, registry, embedder, append([]Option{WithClock(func() time.Time { return now })}, opts...)...)
	if err != nil {
		panic(err)
	}
	return &fixture{svc: svc, store: store, registry: registry, embedder: embedder, now: now, user: user}
}

func (f *fixture) request(text string) c.RetrievalRequest {
	return c.RetrievalRequest{
		UserID: f.user,
		Turn: c.TurnView{
			TurnID: c.NewID(), MessageID: c.NewID(), Text: text, Lang: "en",
		},
		MaxItems: 5,
	}
}

func (f *fixture) memoryItem(content string, kind c.MemoryKind, status c.MemoryStatus, updated time.Time) c.MemoryItem {
	return c.MemoryItem{
		ID: c.NewID(), UserID: f.user, Kind: kind, Content: content, Lang: "en",
		Epistemic: c.EpReported, Status: status, Confidence: c.ConfLow,
		Sensitivity: c.SensNormal, Importance: 0.8,
		CreatedAt: updated, UpdatedAt: updated,
		Source: c.ProvenanceSrc{Origin: "memory_writer", Model: "test", PromptVersion: "v1"},
	}
}

func ptr[T any](v T) *T { return &v }

func idsOf(items []c.MemoryView) []c.ID {
	out := make([]c.ID, 0, len(items))
	for _, item := range items {
		out = append(out, item.ID)
	}
	return out
}

func containsID(items []c.MemoryView, id c.ID) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

// Compile-time assertions that the fixture conforms to the real contracts.
var (
	_ Store                 = (*fakeStore)(nil)
	_ ports.BundleRegistry  = (*fakeRegistry)(nil)
	_ ports.Bundle          = fakeBundle{}
	_ Embedder              = (*fakeEmbedder)(nil)
	_ bundle.Retrieval      = bundle.Retrieval{}
)
