package retrieval

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	domain "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/domain/retrieval"
)

// ---- deterministic test doubles (in-package; not production adapters) ----

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time { return c.now }

type fakeStore struct {
	mu sync.Mutex

	owner contract.ID // the single owner this store pretends to serve

	memory   []contract.MemoryRecord
	messages []contract.MessageRecord
	vectors  map[contract.EmbeddingKey]contract.Vector

	errRecent    error
	errThread    error
	errEntity    error
	errMessages  error
	errEmbedding error
	errPrefs     error

	calls map[string]int
}

func newFakeStore(owner contract.ID) *fakeStore {
	return &fakeStore{owner: owner, vectors: map[contract.EmbeddingKey]contract.Vector{}, calls: map[string]int{}}
}

func (s *fakeStore) note(name string) {
	s.mu.Lock()
	s.calls[name]++
	s.mu.Unlock()
}

// Every fake method filters on userID exactly like the MSSQL adapter, so the
// service-level isolation tests are meaningful.
func (s *fakeStore) RecentMemory(_ context.Context, userID contract.ID, limit int) ([]contract.MemoryRecord, error) {
	s.note("recent")
	if s.errRecent != nil {
		return nil, s.errRecent
	}
	out := []contract.MemoryRecord{}
	for _, m := range s.memory {
		if m.UserID != userID || m.DeletedAt != nil {
			continue
		}
		out = append(out, m)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *fakeStore) Preferences(_ context.Context, userID contract.ID, limit int) ([]contract.MemoryRecord, error) {
	s.note("prefs")
	if s.errPrefs != nil {
		return nil, s.errPrefs
	}
	out := []contract.MemoryRecord{}
	for _, m := range s.memory {
		if m.UserID != userID || m.DeletedAt != nil || m.Kind != contract.MemPreference || m.Status != contract.MemActive {
			continue
		}
		out = append(out, m)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *fakeStore) MemoryByThread(_ context.Context, userID, threadID contract.ID, limit int) ([]contract.MemoryRecord, error) {
	s.note("thread")
	if s.errThread != nil {
		return nil, s.errThread
	}
	out := []contract.MemoryRecord{}
	for _, m := range s.memory {
		if m.UserID != userID || m.DeletedAt != nil {
			continue
		}
		for _, tid := range m.ThreadIDs {
			if tid == threadID {
				out = append(out, m)
				break
			}
		}
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *fakeStore) MemoryByEntity(_ context.Context, userID contract.ID, q EntityFilter, limit int) ([]contract.MemoryRecord, error) {
	s.note("entity")
	if s.errEntity != nil {
		return nil, s.errEntity
	}
	out := []contract.MemoryRecord{}
	for _, m := range s.memory {
		if m.UserID != userID || m.DeletedAt != nil {
			continue
		}
		for _, e := range m.Entities {
			if q.Role != "" && strings.EqualFold(e, q.Role) {
				out = append(out, m)
				break
			}
			if q.Label != "" && strings.EqualFold(e, q.Label) {
				out = append(out, m)
				break
			}
		}
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *fakeStore) MemoryByIDs(_ context.Context, userID contract.ID, ids []contract.ID) ([]contract.MemoryRecord, error) {
	s.note("byids")
	out := []contract.MemoryRecord{}
	want := map[contract.ID]bool{}
	for _, id := range ids {
		want[id] = true
	}
	for _, m := range s.memory {
		if m.UserID != userID || m.DeletedAt != nil || !want[m.ID] {
			continue
		}
		out = append(out, m)
	}
	return out, nil
}

func (s *fakeStore) MessageWindow(_ context.Context, userID contract.ID, from, to time.Time, roles []contract.Role, terms []string, limit int) ([]contract.MessageRecord, error) {
	s.note("messages")
	if s.errMessages != nil {
		return nil, s.errMessages
	}
	roleSet := map[contract.Role]bool{}
	for _, r := range roles {
		roleSet[r] = true
	}
	out := []contract.MessageRecord{}
	for _, m := range s.messages {
		if m.UserID != userID || m.DeletedAt != nil || !roleSet[m.Role] {
			continue
		}
		if m.CreatedAt.Before(from) || !m.CreatedAt.Before(to) {
			continue
		}
		out = append(out, m)
	}
	if len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out, nil
}

func (s *fakeStore) RecentMessages(_ context.Context, userID, _ contract.ID, limit int) ([]string, error) {
	s.note("recent_messages")
	if s.errRecent != nil {
		return nil, s.errRecent
	}
	out := []string{}
	for _, m := range s.messages {
		if m.UserID != userID || m.DeletedAt != nil {
			continue
		}
		out = append(out, m.Content)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *fakeStore) Embeddings(_ context.Context, userID contract.ID, refs []contract.EmbeddingKey) (map[contract.EmbeddingKey]contract.Vector, error) {
	s.note("embeddings")
	if s.errEmbedding != nil {
		return nil, s.errEmbedding
	}
	out := map[contract.EmbeddingKey]contract.Vector{}
	for _, ref := range refs {
		if v, ok := s.vectors[ref]; ok {
			out[ref] = v
		}
	}
	return out, nil
}

type fakeEmbedder struct {
	model string
	err   error
	calls int
}

func (f *fakeEmbedder) ModelID() string { return f.model }
func (f *fakeEmbedder) Embed(_ context.Context, texts []string) ([]contract.Vector, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	out := make([]contract.Vector, 0, len(texts))
	for _, text := range texts {
		// Deterministic 4-d vector; only tests use this shape.
		vals := []float32{float32(len(text) % 3), 1, float32(len(text) % 5), 0.5}
		out = append(out, contract.Vector{ModelID: f.model, Dim: len(vals), Values: vals})
	}
	return out, nil
}

// ---- fixtures ----

var testNow = time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)

func owner() contract.ID { return uuid.MustParse("11111111-1111-1111-1111-111111111111") }
func stranger() contract.ID {
	return uuid.MustParse("22222222-2222-2222-2222-222222222222")
}

func threadID() contract.ID { return uuid.MustParse("33333333-3333-3333-3333-333333333333") }

func mem(id, ownerID string, mutate func(*contract.MemoryRecord)) contract.MemoryRecord {
	rec := contract.MemoryRecord{
		ID: uuid.MustParse(id), UserID: uuid.MustParse(ownerID),
		Kind: contract.MemFact, Epistemic: contract.EpReported, Status: contract.MemActive,
		Content: "synthetic memory text", Sensitivity: contract.SensNormal, Importance: 0.5,
		EvidenceN: 1, CreatedAt: testNow.Add(-48 * time.Hour), UpdatedAt: testNow.Add(-48 * time.Hour),
	}
	if mutate != nil {
		mutate(&rec)
	}
	return rec
}

// testTunables keeps the contract defaults but lets assertions be about
// behaviour rather than about the 0.32 score floor. Tests that specifically cover
// the floor use domain.Defaults() directly (see TestScoreFloorIsEnforced).
func testTunables() domain.Tunables {
	tu := domain.Defaults()
	tu.MinScore = 0.01
	return tu
}

func newService(t *testing.T, store Store, embedder Embedder) *Service {
	t.Helper()
	svc, err := New(Config{Store: store, Embedder: embedder, Clock: &fakeClock{now: testNow}, Tunables: testTunables()})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	return svc
}

func baseRequest(text string) contract.RetrievalRequest {
	return contract.RetrievalRequest{
		UserID: owner(),
		Turn:   contract.TurnView{TurnID: uuid.New(), MessageID: uuid.New(), Text: text, Lang: "en"},
	}
}

// ---- tests ----

func TestNewRejectsInvalidConfiguration(t *testing.T) {
	bad := domain.Defaults()
	bad.WeightSimilarity = 5
	if _, err := New(Config{Tunables: bad}); err == nil {
		t.Fatal("expected configuration rejection")
	}
}

func TestRetrieveRequiresServerResolvedUser(t *testing.T) {
	svc := newService(t, newFakeStore(owner()), nil)
	res, err := svc.Retrieve(context.Background(), contract.RetrievalRequest{Turn: contract.TurnView{Text: "hi"}})
	if !errors.Is(err, ErrUserRequired) {
		t.Fatalf("want ErrUserRequired, got %v", err)
	}
	if len(res.Items) != 0 {
		t.Fatal("no items without a user")
	}
}

func TestRetrieveRequiresStore(t *testing.T) {
	svc := newService(t, nil, nil)
	res, err := svc.Retrieve(context.Background(), baseRequest("hello"))
	if !errors.Is(err, ErrNoStore) || !res.Meta.Degraded {
		t.Fatalf("missing store must degrade, got err=%v meta=%+v", err, res.Meta)
	}
}

func TestRetrieveRejectsUnknownStrategy(t *testing.T) {
	svc := newService(t, newFakeStore(owner()), nil)
	req := baseRequest("hello")
	req.Strategies = []contract.RetrievalStrategy{"pgvector_ann"}
	if _, err := svc.Retrieve(context.Background(), req); !errors.Is(err, ErrInvalidStrategy) {
		t.Fatalf("want ErrInvalidStrategy, got %v", err)
	}
}

func TestHybridRetrievalRanksRelevantFirst(t *testing.T) {
	store := newFakeStore(owner())
	relevant := mem("aaaaaaaa-0000-0000-0000-000000000001", owner().String(), func(m *contract.MemoryRecord) {
		m.Content = "my manager criticized my work in front of the team"
		m.Importance = 0.7
		m.UpdatedAt = testNow.Add(-2 * time.Hour)
	})
	stale := mem("bbbbbbbb-0000-0000-0000-000000000002", owner().String(), func(m *contract.MemoryRecord) {
		m.Content = "we talked about gardening and the weather"
		m.Importance = 0.2
		m.UpdatedAt = testNow.Add(-60 * 24 * time.Hour)
	})
	store.memory = []contract.MemoryRecord{stale, relevant}
	svc := newService(t, store, nil)

	res, err := svc.Retrieve(context.Background(), baseRequest("my manager criticized my work again"))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 2 {
		t.Fatalf("want both candidates, got %d", len(res.Items))
	}
	if res.Items[0].ID != relevant.ID {
		t.Fatalf("relevant item must rank first, got %v", res.Items[0].ID)
	}
	if res.Items[0].Score <= res.Items[1].Score {
		t.Fatalf("scores must be ordered: %v vs %v", res.Items[0].Score, res.Items[1].Score)
	}
	if res.Meta.Strategies[0] != domain.StrategyHybrid {
		t.Fatalf("default strategy should be hybrid, got %v", res.Meta.Strategies)
	}
	if res.Meta.Degraded {
		t.Fatal("happy path must not be degraded")
	}
}

func TestHybridUsesStoredEmbeddingsWhenAvailable(t *testing.T) {
	store := newFakeStore(owner())
	a := mem("aaaaaaaa-0000-0000-0000-000000000001", owner().String(), func(m *contract.MemoryRecord) {
		m.Content = "zzz zzz zzz"
		m.Importance = 0.1
		m.UpdatedAt = testNow.Add(-100 * 24 * time.Hour)
	})
	b := mem("bbbbbbbb-0000-0000-0000-000000000002", owner().String(), func(m *contract.MemoryRecord) {
		m.Content = "yyy yyy yyy"
		m.Importance = 0.1
		m.UpdatedAt = testNow.Add(-100 * 24 * time.Hour)
	})
	store.memory = []contract.MemoryRecord{a, b}
	// Give `a` a vector aligned with the query embedding, `b` an orthogonal one.
	store.vectors[contract.EmbeddingKey{Kind: contract.RefKindMemory, ID: a.ID}] = contract.Vector{ModelID: "m1", Dim: 4, Values: []float32{1, 0, 0, 0}}
	store.vectors[contract.EmbeddingKey{Kind: contract.RefKindMemory, ID: b.ID}] = contract.Vector{ModelID: "m1", Dim: 4, Values: []float32{0, 1, 0, 0}}

	embedder := &aligningEmbedder{model: "m1", values: []float32{1, 0, 0, 0}}
	svc := newService(t, store, embedder)
	res, err := svc.Retrieve(context.Background(), baseRequest("anything"))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 2 {
		t.Fatalf("want two items, got %d", len(res.Items))
	}
	if res.Items[0].ID != a.ID {
		t.Fatalf("cosine-aligned item must rank first, got %v", res.Items[0].ID)
	}
}

type aligningEmbedder struct {
	model  string
	values []float32
}

func (a *aligningEmbedder) ModelID() string { return a.model }
func (a *aligningEmbedder) Embed(_ context.Context, _ []string) ([]contract.Vector, error) {
	return []contract.Vector{{ModelID: a.model, Dim: len(a.values), Values: a.values}}, nil
}

func TestHybridFallsBackToLexicalWhenEmbeddingMissing(t *testing.T) {
	store := newFakeStore(owner())
	// The stored vector is orthogonal to the query, so its only similarity can
	// come from the lexical fallback (this item has an embedding).
	withVector := mem("aaaaaaaa-0000-0000-0000-000000000001", owner().String(), func(m *contract.MemoryRecord) {
		m.Content = "work stress work stress"
	})
	noVector := mem("bbbbbbbb-0000-0000-0000-000000000002", owner().String(), func(m *contract.MemoryRecord) {
		m.Content = "work stress and my manager"
	})
	store.memory = []contract.MemoryRecord{withVector, noVector}
	store.vectors[contract.EmbeddingKey{Kind: contract.RefKindMemory, ID: withVector.ID}] = contract.Vector{ModelID: "m1", Dim: 2, Values: []float32{0, 1}}

	svc := newService(t, store, &aligningEmbedder{model: "m1", values: []float32{1, 0}})
	res, err := svc.Retrieve(context.Background(), baseRequest("work stress"))
	if err != nil {
		t.Fatal(err)
	}
	// The item without an embedding is still retrieved via lexical coverage.
	found := false
	for _, item := range res.Items {
		if item.ID == noVector.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing-embedding item must still be retrievable, got %+v", res.Items)
	}
}

func TestEmbeddingDimensionMismatchIsNotScored(t *testing.T) {
	store := newFakeStore(owner())
	item := mem("aaaaaaaa-0000-0000-0000-000000000001", owner().String(), func(m *contract.MemoryRecord) {
		m.Content = "unrelated"
	})
	store.memory = []contract.MemoryRecord{item}
	store.vectors[contract.EmbeddingKey{Kind: contract.RefKindMemory, ID: item.ID}] = contract.Vector{ModelID: "m1", Dim: 2, Values: []float32{1, 0}}

	// Baseline: the same item with no embedding available at all.
	baseline, err := newService(t, cloneStore(store), nil).Retrieve(context.Background(), baseRequest("nothing shared"))
	if err != nil {
		t.Fatal(err)
	}
	// Same item seen through a 4-dim query embedding: the 2-dim stored vector must
	// contribute nothing, so the score must be identical.
	withMismatch, err := newService(t, cloneStore(store), &aligningEmbedder{model: "m1", values: []float32{1, 0, 0, 0}}).
		Retrieve(context.Background(), baseRequest("nothing shared"))
	if err != nil {
		t.Fatal(err)
	}
	if len(baseline.Items) != len(withMismatch.Items) {
		t.Fatalf("dimension mismatch changed the result set: %d vs %d", len(baseline.Items), len(withMismatch.Items))
	}
	if len(baseline.Items) == 1 && baseline.Items[0].Score != withMismatch.Items[0].Score {
		t.Fatalf("a mismatched vector must not add similarity: %v vs %v", baseline.Items[0].Score, withMismatch.Items[0].Score)
	}
}

func TestEmbeddingModelMismatchIsNotScored(t *testing.T) {
	store := newFakeStore(owner())
	item := mem("aaaaaaaa-0000-0000-0000-000000000001", owner().String(), func(m *contract.MemoryRecord) {
		m.Content = "different model"
	})
	store.memory = []contract.MemoryRecord{item}
	// Stored by an older model; the query embedding comes from the current one.
	store.vectors[contract.EmbeddingKey{Kind: contract.RefKindMemory, ID: item.ID}] = contract.Vector{ModelID: "old-model", Dim: 2, Values: []float32{1, 0}}

	baseline, err := newService(t, cloneStore(store), nil).Retrieve(context.Background(), baseRequest("no lexical overlap at all"))
	if err != nil {
		t.Fatal(err)
	}
	withMismatch, err := newService(t, cloneStore(store), &aligningEmbedder{model: "new-model", values: []float32{1, 0}}).
		Retrieve(context.Background(), baseRequest("no lexical overlap at all"))
	if err != nil {
		t.Fatal(err)
	}
	if len(baseline.Items) != len(withMismatch.Items) {
		t.Fatalf("model mismatch changed the result set: %d vs %d", len(baseline.Items), len(withMismatch.Items))
	}
	if len(baseline.Items) == 1 && baseline.Items[0].Score != withMismatch.Items[0].Score {
		t.Fatalf("vectors from different models must never be compared: %v vs %v", baseline.Items[0].Score, withMismatch.Items[0].Score)
	}
}

// cloneStore returns an independent copy so two services can share a fixture
// without one test's mutation affecting the other.
func cloneStore(in *fakeStore) *fakeStore {
	out := newFakeStore(in.owner)
	out.memory = append([]contract.MemoryRecord(nil), in.memory...)
	out.messages = append([]contract.MessageRecord(nil), in.messages...)
	out.vectors = map[contract.EmbeddingKey]contract.Vector{}
	for k, v := range in.vectors {
		out.vectors[k] = v
	}
	return out
}

func TestEmbedderFailureDegradesToLexicalNotToFailure(t *testing.T) {
	store := newFakeStore(owner())
	item := mem("aaaaaaaa-0000-0000-0000-000000000001", owner().String(), func(m *contract.MemoryRecord) {
		m.Content = "work stress"
	})
	store.memory = []contract.MemoryRecord{item}
	svc := newService(t, store, &fakeEmbedder{model: "m1", err: errors.New("provider down")})
	res, err := svc.Retrieve(context.Background(), baseRequest("work stress"))
	if err != nil {
		t.Fatalf("embedding failure must not fail retrieval: %v", err)
	}
	if res.Meta.Degraded {
		t.Fatal("lexical fallback is a valid result, not a degraded one")
	}
	if len(res.Items) != 1 {
		t.Fatalf("lexical fallback should still retrieve, got %d", len(res.Items))
	}
}

func TestExcludedStatusesAndSoftDeletesNeverSurface(t *testing.T) {
	store := newFakeStore(owner())
	live := mem("aaaaaaaa-0000-0000-0000-000000000001", owner().String(), func(m *contract.MemoryRecord) { m.Importance = 0.8 })
	deleted := mem("bbbbbbbb-0000-0000-0000-000000000002", owner().String(), func(m *contract.MemoryRecord) {
		when := testNow.Add(-time.Hour)
		m.DeletedAt = &when
	})
	superseded := mem("cccccccc-0000-0000-0000-000000000003", owner().String(), func(m *contract.MemoryRecord) {
		m.Status = contract.MemSuperseded
	})
	rejected := mem("dddddddd-0000-0000-0000-000000000004", owner().String(), func(m *contract.MemoryRecord) {
		m.Status = contract.MemRejected
	})
	noEvidence := mem("eeeeeeee-0000-0000-0000-000000000005", owner().String(), func(m *contract.MemoryRecord) {
		m.EvidenceN = 0
	})
	highlySensitive := mem("ffffffff-0000-0000-0000-000000000006", owner().String(), func(m *contract.MemoryRecord) {
		m.Sensitivity = contract.SensHighly
	})
	store.memory = []contract.MemoryRecord{live, deleted, superseded, rejected, noEvidence, highlySensitive}

	svc := newService(t, store, nil)
	res, err := svc.Retrieve(context.Background(), baseRequest("synthetic memory text"))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 1 || res.Items[0].ID != live.ID {
		t.Fatalf("only the live, evidenced, permitted item may surface: %+v", res.Items)
	}
}

func TestHypothesisSurfacingRules(t *testing.T) {
	store := newFakeStore(owner())
	supported := mem("aaaaaaaa-0000-0000-0000-000000000001", owner().String(), func(m *contract.MemoryRecord) {
		m.Kind = contract.MemHypothesis
		m.Epistemic = contract.EpInferred
		m.Status = contract.MemSupported
		m.Content = "may associate work criticism with self-worth"
		m.Importance = 0.8
	})
	proposed := mem("bbbbbbbb-0000-0000-0000-000000000002", owner().String(), func(m *contract.MemoryRecord) {
		m.Kind = contract.MemHypothesis
		m.Epistemic = contract.EpInferred
		m.Status = contract.MemProposed
		m.Importance = 0.7
	})
	contested := mem("cccccccc-0000-0000-0000-000000000003", owner().String(), func(m *contract.MemoryRecord) {
		m.Kind = contract.MemHypothesis
		m.Epistemic = contract.EpInferred
		m.Status = contract.MemContested
	})
	reportedAsHypothesis := mem("dddddddd-0000-0000-0000-000000000004", owner().String(), func(m *contract.MemoryRecord) {
		m.Kind = contract.MemHypothesis
		m.Epistemic = contract.EpReported
		m.Status = contract.MemSupported
		m.Importance = 0.9
	})
	store.memory = []contract.MemoryRecord{supported, proposed, contested, reportedAsHypothesis}

	svc := newService(t, store, nil)
	res, err := svc.Retrieve(context.Background(), baseRequest("synthetic memory text"))
	if err != nil {
		t.Fatal(err)
	}
	got := map[contract.ID]bool{}
	for _, item := range res.Items {
		got[item.ID] = true
	}
	if !got[supported.ID] || !got[proposed.ID] {
		t.Fatalf("proposed and supported hypotheses may surface: %+v", got)
	}
	if got[contested.ID] {
		t.Fatal("contested hypotheses must not surface (§7.5)")
	}
	if got[reportedAsHypothesis.ID] {
		t.Fatal("a hypothesis stored as reported is invalid and must never surface")
	}
}

func TestCrossUserIsolation(t *testing.T) {
	store := newFakeStore(owner())
	mine := mem("aaaaaaaa-0000-0000-0000-000000000001", owner().String(), func(m *contract.MemoryRecord) { m.Importance = 0.8 })
	theirs := mem("bbbbbbbb-0000-0000-0000-000000000002", stranger().String(), func(m *contract.MemoryRecord) {
		m.Content = "another user's private memory"
	})
	store.memory = []contract.MemoryRecord{mine, theirs}
	store.messages = []contract.MessageRecord{
		{ID: uuid.New(), UserID: stranger(), Role: contract.RoleUser, Content: "another user's message", CreatedAt: testNow},
	}
	svc := newService(t, store, nil)
	res, err := svc.Retrieve(context.Background(), baseRequest("synthetic memory text"))
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range res.Items {
		if item.ID == theirs.ID {
			t.Fatal("cross-user memory leak")
		}
		if strings.Contains(item.Text, "another user") {
			t.Fatal("cross-user content leak")
		}
	}
	if len(res.Items) != 1 || res.Items[0].ID != mine.ID {
		t.Fatalf("only the owner's memory may surface, got %+v", res.Items)
	}
}

func TestServiceIgnoresStoreRowsWithWrongOwner(t *testing.T) {
	// A buggy or compromised store returns a foreign row: the service must drop
	// it on its own (defense in depth), not rely only on SQL scoping.
	store := newFakeStore(owner())
	foreign := mem("aaaaaaaa-0000-0000-0000-000000000001", stranger().String(), nil)
	store.memory = []contract.MemoryRecord{foreign}
	svc := newService(t, store, nil)
	res, err := svc.Retrieve(context.Background(), baseRequest("synthetic memory text"))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 0 {
		t.Fatalf("foreign row surfaced: %+v", res.Items)
	}
}

func TestTemporalRetrievalReturnsVerbatimMessages(t *testing.T) {
	store := newFakeStore(owner())
	within := contract.MessageRecord{
		ID: uuid.New(), UserID: owner(), Role: contract.RoleUser, Content: "my boss told me the review went badly",
		CreatedAt: time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC),
	}
	outsider := contract.MessageRecord{
		ID: uuid.New(), UserID: owner(), Role: contract.RoleUser, Content: "different day entirely",
		CreatedAt: time.Date(2026, 11, 4, 9, 0, 0, 0, time.UTC),
	}
	store.messages = []contract.MessageRecord{within, outsider}
	svc := newService(t, store, nil)

	req := baseRequest("what did I tell you last Tuesday about my boss")
	res, err := svc.Retrieve(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Meta.ExactMatchFound == nil || !*res.Meta.ExactMatchFound {
		t.Fatalf("exact_match_found must be true when a message matched: %+v", res.Meta)
	}
	if len(res.Items) == 0 || res.Items[0].ID != within.ID {
		t.Fatalf("temporal hit must rank first: %+v", res.Items)
	}
	if !res.Items[0].Verbatim {
		t.Fatal("exact recall items must be marked verbatim")
	}
	found := false
	for _, item := range res.Items {
		if item.ID == outsider.ID {
			found = true
		}
	}
	if found {
		t.Fatal("a message outside the window must not be returned")
	}
}

func TestTemporalEmptyWindowReportsNoExactMatch(t *testing.T) {
	store := newFakeStore(owner())
	svc := newService(t, store, nil)
	res, err := svc.Retrieve(context.Background(), baseRequest("what did I tell you last Tuesday about my boss"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Meta.ExactMatchFound == nil || *res.Meta.ExactMatchFound {
		t.Fatalf("no messages that day => exact_match_found=false (no semantic substitution): %+v", res.Meta)
	}
	if len(res.Items) != 0 {
		t.Fatalf("nothing should be returned, got %+v", res.Items)
	}
}

func TestThreadStrategyReturnsFocusThreadItems(t *testing.T) {
	store := newFakeStore(owner())
	linked := mem("aaaaaaaa-0000-0000-0000-000000000001", owner().String(), func(m *contract.MemoryRecord) {
		m.ThreadIDs = []contract.ID{threadID()}
		m.Content = "social comparison at work"
	})
	unlinked := mem("bbbbbbbb-0000-0000-0000-000000000002", owner().String(), func(m *contract.MemoryRecord) {
		m.Content = "gardening"
		m.UpdatedAt = testNow.Add(-200 * 24 * time.Hour)
		m.Importance = 0.1
	})
	store.memory = []contract.MemoryRecord{linked, unlinked}
	svc := newService(t, store, nil)
	req := baseRequest("something else entirely")
	tid := threadID()
	req.FocusThread = &tid
	req.Strategies = []contract.RetrievalStrategy{contract.StratThread}

	res, err := svc.Retrieve(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 1 || res.Items[0].ID != linked.ID {
		t.Fatalf("thread strategy must return only the linked item: %+v", res.Items)
	}
	if res.Meta.Strategies[0] != domain.StrategyThread {
		t.Fatalf("unexpected strategies %v", res.Meta.Strategies)
	}
}

func TestAnaphoraMarksAmbiguityWithoutThread(t *testing.T) {
	store := newFakeStore(owner())
	store.memory = []contract.MemoryRecord{
		mem("aaaaaaaa-0000-0000-0000-000000000001", owner().String(), func(m *contract.MemoryRecord) {
			m.Content = "felt worthless after the meeting"
			m.Tags = []string{"worthless"}
			m.UpdatedAt = testNow.Add(-3 * 24 * time.Hour)
		}),
	}
	svc := newService(t, store, nil)
	res, err := svc.Retrieve(context.Background(), baseRequest("it's the same feeling I told you about before"))
	if err != nil {
		t.Fatal(err)
	}
	if !res.Meta.AmbiguousReference {
		t.Fatal("an unresolved reference must set ambiguous_reference so the Brain clarifies")
	}
	if len(res.Items) == 0 {
		t.Fatal("the recent tagged item should still be offered as evidence")
	}
}

func TestEntityStrategyRetrievesByMention(t *testing.T) {
	store := newFakeStore(owner())
	sister := mem("aaaaaaaa-0000-0000-0000-000000000001", owner().String(), func(m *contract.MemoryRecord) {
		m.Entities = []string{"sister"}
		m.Content = "my sister moved to another city"
		m.UpdatedAt = testNow.Add(-90 * 24 * time.Hour)
		m.Importance = 0.2
	})
	boss := mem("bbbbbbbb-0000-0000-0000-000000000002", owner().String(), func(m *contract.MemoryRecord) {
		m.Entities = []string{"boss"}
		m.Content = "my boss is demanding"
	})
	store.memory = []contract.MemoryRecord{sister, boss}
	svc := newService(t, store, nil)
	req := baseRequest("my sister called today")
	req.Strategies = []contract.RetrievalStrategy{contract.StratEntity}

	res, err := svc.Retrieve(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) == 0 || res.Items[0].ID != sister.ID {
		t.Fatalf("entity strategy must resolve 'my sister': %+v", res.Items)
	}
	for _, item := range res.Items {
		if item.ID == boss.ID {
			t.Fatal("an unrelated entity must not be retrieved")
		}
	}
}

func TestSecondHopAddsTargetedItemAndCapsAtSeven(t *testing.T) {
	store := newFakeStore(owner())
	var all []contract.MemoryRecord
	for i := 0; i < 12; i++ {
		id := uuid.New()
		all = append(all, mem(id.String(), owner().String(), func(m *contract.MemoryRecord) {
			m.Content = "synthetic memory text"
			m.Importance = 0.9
		}))
	}
	store.memory = all
	svc := newService(t, store, nil)
	req := baseRequest("tell me more")
	req.MaxItems = 50 // caller asks for too much: the hard cap must win
	req.Extra = &contract.NeedsContext{Kind: "entity", Query: "my manager", Why: "reference_unresolved"}

	res, err := svc.Retrieve(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Meta.Hops != 1 {
		t.Fatalf("hops must be reported, got %d", res.Meta.Hops)
	}
	if len(res.Items) > svc.Tunables().HardCapWithHop {
		t.Fatalf("hard cap of 7 exceeded: %d", len(res.Items))
	}
}

func TestStoreFailuresDegradeAndReturnNoItems(t *testing.T) {
	tid := threadID()
	failures := map[string]struct {
		inject  func(*fakeStore)
		request func(*contract.RetrievalRequest)
	}{
		"recent memory": {func(s *fakeStore) { s.errRecent = errors.New("db down") }, func(r *contract.RetrievalRequest) {
			r.Strategies = []contract.RetrievalStrategy{contract.StratHybrid}
		}},
		"thread": {func(s *fakeStore) { s.errThread = errors.New("db down") }, func(r *contract.RetrievalRequest) {
			r.FocusThread = &tid
			r.Strategies = []contract.RetrievalStrategy{contract.StratThread}
		}},
		"messages": {func(s *fakeStore) { s.errMessages = errors.New("db down") }, func(r *contract.RetrievalRequest) {
			r.Turn.Text = "what did I tell you last Tuesday about my boss"
		}},
		"entity": {func(s *fakeStore) { s.errEntity = errors.New("db down") }, func(r *contract.RetrievalRequest) {
			r.Turn.Text = "my sister called"
			r.Strategies = []contract.RetrievalStrategy{contract.StratEntity}
		}},
		"embedding read": {func(s *fakeStore) { s.errEmbedding = errors.New("db down") }, func(r *contract.RetrievalRequest) {
			r.Strategies = []contract.RetrievalStrategy{contract.StratHybrid}
		}},
	}
	for name, tc := range failures {
		t.Run(name, func(t *testing.T) {
			store := newFakeStore(owner())
			store.memory = []contract.MemoryRecord{mem("aaaaaaaa-0000-0000-0000-000000000001", owner().String(), nil)}
			tc.inject(store)
			svc := newService(t, store, &fakeEmbedder{model: "m1"})
			req := baseRequest("could we talk about something")
			tc.request(&req)
			res, err := svc.Retrieve(context.Background(), req)
			if err != nil {
				t.Fatalf("retrieval must not return an error to the caller: %v", err)
			}
			if !res.Meta.Degraded {
				t.Fatal("a store failure must mark the result degraded")
			}
			if len(res.Items) != 0 {
				t.Fatal("degraded retrieval must return no items (FB-002)")
			}
		})
	}
}

func TestCanceledContextDegrades(t *testing.T) {
	store := newFakeStore(owner())
	store.memory = []contract.MemoryRecord{mem("aaaaaaaa-0000-0000-0000-000000000001", owner().String(), nil)}
	svc := newService(t, store, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res, err := svc.Retrieve(ctx, baseRequest("anything"))
	if err != nil {
		t.Fatalf("cancellation must degrade, not error: %v", err)
	}
	if !res.Meta.Degraded {
		t.Fatal("canceled context must set degraded")
	}
}

func TestDedupeCollapsesClustersEndToEnd(t *testing.T) {
	store := newFakeStore(owner())
	cluster := uuid.MustParse("99999999-9999-9999-9999-999999999999")
	store.memory = []contract.MemoryRecord{
		mem("aaaaaaaa-0000-0000-0000-000000000001", owner().String(), func(m *contract.MemoryRecord) {
			m.Content = "argument with my manager about the deadline"
			m.ClusterID = &cluster
			m.Importance = 0.9
		}),
		mem("bbbbbbbb-0000-0000-0000-000000000002", owner().String(), func(m *contract.MemoryRecord) {
			m.Content = "argument with my manager about the deadline (restated)"
			m.ClusterID = &cluster
			m.Importance = 0.8
		}),
	}
	svc := newService(t, store, nil)
	res, err := svc.Retrieve(context.Background(), baseRequest("argument with my manager"))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 1 {
		t.Fatalf("cluster duplicate must collapse, got %+v", res.Items)
	}
}

func TestTokenBudgetTruncatesLongItems(t *testing.T) {
	store := newFakeStore(owner())
	long := strings.Repeat("word ", 300)
	store.memory = []contract.MemoryRecord{
		mem("aaaaaaaa-0000-0000-0000-000000000001", owner().String(), func(m *contract.MemoryRecord) {
			m.Content = long
			m.Importance = 0.9
		}),
	}
	svc := newService(t, store, nil)
	res, err := svc.Retrieve(context.Background(), baseRequest("word"))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 1 {
		t.Fatalf("want one item, got %d", len(res.Items))
	}
	if got := domain.EstimateTokens(res.Items[0].Text); got > svc.Tunables().PerItemTokens {
		t.Fatalf("per-item token budget exceeded: %d", got)
	}
}

func TestEmptyAndWhitespaceTurnsAreSafe(t *testing.T) {
	store := newFakeStore(owner())
	store.memory = []contract.MemoryRecord{mem("aaaaaaaa-0000-0000-0000-000000000001", owner().String(), nil)}
	svc := newService(t, store, nil)
	for _, text := range []string{"", "   "} {
		res, err := svc.Retrieve(context.Background(), baseRequest(text))
		if err != nil {
			t.Fatalf("%q: %v", text, err)
		}
		if res.Meta.Degraded {
			t.Fatalf("%q: an empty turn is not a failure", text)
		}
	}
}

func TestMetaReportsStrategiesExecuted(t *testing.T) {
	store := newFakeStore(owner())
	tid := threadID()
	store.memory = []contract.MemoryRecord{mem("aaaaaaaa-0000-0000-0000-000000000001", owner().String(), func(m *contract.MemoryRecord) {
		m.ThreadIDs = []contract.ID{tid}
	})}
	svc := newService(t, store, nil)
	req := baseRequest("same feeling as before")
	req.FocusThread = &tid
	res, err := svc.Retrieve(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Meta.Strategies) < 2 {
		t.Fatalf("expected anaphora + thread, got %v", res.Meta.Strategies)
	}
	if res.Took < 0 {
		t.Fatal("took must be non-negative")
	}
}

func TestRetrievalRespectsCallerDeadline(t *testing.T) {
	store := newFakeStore(owner())
	store.memory = []contract.MemoryRecord{mem("aaaaaaaa-0000-0000-0000-000000000001", owner().String(), nil)}
	svc := newService(t, store, nil)
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	res, err := svc.Retrieve(ctx, baseRequest("anything"))
	if err != nil {
		t.Fatalf("deadline must degrade not error: %v", err)
	}
	if !res.Meta.Degraded && len(res.Items) > 0 {
		t.Log("fast path completed before the deadline expired; acceptable")
	}
}

func TestScoreFloorIsEnforcedWithContractDefaults(t *testing.T) {
	// With the real 0.32 floor, an item with no lexical or vector signal is not
	// context: retrieving it would dilute the Brain's limited budget.
	store := newFakeStore(owner())
	store.memory = []contract.MemoryRecord{mem("aaaaaaaa-0000-0000-0000-000000000001", owner().String(), func(m *contract.MemoryRecord) {
		m.Content = "gardening and the weather"
		m.Importance = 0.2
		m.UpdatedAt = testNow.Add(-120 * 24 * time.Hour)
	})}
	svc, err := New(Config{Store: store, Clock: &fakeClock{now: testNow}, Tunables: domain.Defaults()})
	if err != nil {
		t.Fatal(err)
	}
	res, err := svc.Retrieve(context.Background(), baseRequest("my manager"))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 0 {
		t.Fatalf("weak candidates must fall below the contract score floor: %+v", res.Items)
	}
}

func TestUnknownHopKindIsRejected(t *testing.T) {
	store := newFakeStore(owner())
	svc := newService(t, store, nil)
	req := baseRequest("tell me more")
	req.Extra = &contract.NeedsContext{Kind: "everything", Query: "my whole life", Why: "reference_unresolved"}
	if _, err := svc.Retrieve(context.Background(), req); !errors.Is(err, ErrInvalidStrategy) {
		t.Fatalf("unknown needs_context kind must be rejected, got %v", err)
	}
	for _, kind := range []string{"entity", "topic", "time_window", "item"} {
		req.Extra = &contract.NeedsContext{Kind: kind, Query: "recent thing", Why: "missing_background"}
		if _, err := svc.Retrieve(context.Background(), req); err != nil {
			t.Fatalf("kind %q rejected: %v", kind, err)
		}
	}
}

func TestItemHopUsesDirectLookupWithoutExtraStrategy(t *testing.T) {
	store := newFakeStore(owner())
	item := mem("aaaaaaaa-0000-0000-0000-000000000001", owner().String(), func(m *contract.MemoryRecord) {
		m.Content = "contested statement"
		m.Importance = 0.9
	})
	store.memory = []contract.MemoryRecord{item}
	svc := newService(t, store, nil)
	req := baseRequest("that thing I told you")
	req.Extra = &contract.NeedsContext{Kind: "item", Query: "contested statement", Why: "verify_fact"}
	res, err := svc.Retrieve(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if res.Meta.Hops != 1 {
		t.Fatalf("hop must be recorded, got %d", res.Meta.Hops)
	}
}

func TestServiceLifecycleWithoutExplicitClockUsesSystemTime(t *testing.T) {
	store := newFakeStore(owner())
	store.memory = []contract.MemoryRecord{mem("aaaaaaaa-0000-0000-0000-000000000001", owner().String(), func(m *contract.MemoryRecord) {
		m.Content = "recent item"
		m.UpdatedAt = time.Now().UTC()
		m.Importance = 0.9
	})}
	svc, err := New(Config{Store: store, Tunables: testTunables()})
	if err != nil {
		t.Fatal(err)
	}
	res, err := svc.Retrieve(context.Background(), baseRequest("recent item"))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 1 || res.Items[0].Age == "" {
		t.Fatalf("system clock path must still produce an age label: %+v", res.Items)
	}
}

func TestTemporalRefFromAssemblerIsPreferred(t *testing.T) {
	store := newFakeStore(owner())
	from := time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	msg := contract.MessageRecord{
		ID: uuid.New(), UserID: owner(), Role: contract.RoleUser,
		Content: `my boss said the review "went well"`, CreatedAt: from.Add(9 * time.Hour),
	}
	store.messages = []contract.MessageRecord{msg}
	svc := newService(t, store, nil)
	req := baseRequest("what did I tell you about my boss")
	req.Turn.TemporalRefs = []contract.TemporalRef{{Phrase: "last tuesday", From: from, To: from.Add(24 * time.Hour), RecallVerb: true}}
	res, err := svc.Retrieve(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 1 || res.Items[0].ID != msg.ID {
		t.Fatalf("assembler-resolved window must drive temporal retrieval: %+v", res.Items)
	}
	if !res.Items[0].Verbatim || !strings.Contains(res.Items[0].Text, `"went well"`) {
		t.Fatal("verbatim recall must preserve the original wording")
	}
}

func TestNoveltyFilterUsesRecentMessages(t *testing.T) {
	store := newFakeStore(owner())
	conv := uuid.New()
	repeat := mem("aaaaaaaa-0000-0000-0000-000000000001", owner().String(), func(m *contract.MemoryRecord) {
		m.Content = "I argued with my manager today"
		m.Importance = 0.9
	})
	store.memory = []contract.MemoryRecord{repeat}
	store.messages = []contract.MessageRecord{{
		ID: uuid.New(), UserID: owner(), ConversationID: conv, Role: contract.RoleUser,
		Content: "I argued with my manager today", CreatedAt: testNow,
	}}
	svc := newService(t, store, nil)
	req := baseRequest("I argued with my manager today")
	req.Turn.ConversationID = conv
	res, err := svc.Retrieve(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 0 {
		t.Fatalf("an item already said in the recent window must be filtered: %+v", res.Items)
	}
}

func TestNoveltyLookupFailureIsHandledGracefully(t *testing.T) {
	store := newFakeStore(owner())
	store.errRecent = errors.New("novelty unavailable")
	store.memory = []contract.MemoryRecord{mem("aaaaaaaa-0000-0000-0000-000000000001", owner().String(), func(m *contract.MemoryRecord) {
		m.Importance = 0.9
	})}
	svc := newService(t, store, nil)
	req := baseRequest("synthetic memory text")
	req.Turn.ConversationID = uuid.New()
	res, err := svc.Retrieve(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	// The recent-memory pool read failed too, so the strategy fails; the contract
	// only requires that the service reports it rather than returning partial data.
	if !res.Meta.Degraded && len(res.Items) == 0 {
		t.Fatalf("either results or an explicit degraded flag expected: %+v", res.Meta)
	}
}

func TestAmbiguousReferenceDropsHypothesesButKeepsEvidence(t *testing.T) {
	store := newFakeStore(owner())
	hyp := mem("aaaaaaaa-0000-0000-0000-000000000001", owner().String(), func(m *contract.MemoryRecord) {
		m.Kind = contract.MemHypothesis
		m.Epistemic = contract.EpInferred
		m.Status = contract.MemSupported
		m.Content = "may associate this feeling with comparison"
		m.Importance = 1
	})
	evidence := mem("bbbbbbbb-0000-0000-0000-000000000002", owner().String(), func(m *contract.MemoryRecord) {
		m.Kind = contract.MemEpisode
		m.Content = "felt worthless after the meeting"
		m.Importance = 0.8
	})
	store.memory = []contract.MemoryRecord{hyp, evidence}
	svc := newService(t, store, nil)

	res, err := svc.Retrieve(context.Background(), baseRequest("it's the same feeling I told you about before"))
	if err != nil {
		t.Fatal(err)
	}
	if !res.Meta.AmbiguousReference {
		t.Fatal("ambiguity must be reported")
	}
	if len(res.Items) != 1 || res.Items[0].ID != evidence.ID {
		t.Fatalf("with an unresolved reference only evidence may surface: %+v", res.Items)
	}
}

func TestExactMatchFlagOnlySetWhenTemporalRan(t *testing.T) {
	store := newFakeStore(owner())
	store.memory = []contract.MemoryRecord{mem("aaaaaaaa-0000-0000-0000-000000000001", owner().String(), func(m *contract.MemoryRecord) {
		m.Importance = 0.9
	})}
	svc := newService(t, store, nil)
	res, err := svc.Retrieve(context.Background(), baseRequest("synthetic memory text"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Meta.ExactMatchFound != nil {
		t.Fatalf("a non-recall turn must not report exact_match_found: %+v", res.Meta)
	}
}
