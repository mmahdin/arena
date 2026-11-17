package integration

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// These tests exercise the WP-09 SQL Server read adapter against a disposable
// SQL Server 2022 database created by aiDatabase (opt-in via
// WELLBEING_INTEGRATION=true and a local DATABASE_URL). They assert owner
// isolation, soft-delete visibility, bounded reads and missing-embedding
// behaviour. They are skipped without the opt-in flag, so they never assert a
// result that was not executed.

type retrievalFixture struct {
	db      *sql.DB
	store   *mssql.RetrievalStore
	userA   c.ID
	userB   c.ID
	turnA   c.ID
	turnB   c.ID
	convA   c.ID
	convB   c.ID
}

func newRetrievalFixture(t *testing.T, db *sql.DB) retrievalFixture {
	t.Helper()
	f := retrievalFixture{db: db, userA: c.NewID(), userB: c.NewID(), convA: c.NewID(), convB: c.NewID(),
		turnA: c.NewID(), turnB: c.NewID()}
	for _, pair := range []struct {
		id   c.ID
		key  string
	}{{f.userA, "ret-a-" + f.userA.String()}, {f.userB, "ret-b-" + f.userB.String()}} {
		aiExec(t, db, `INSERT dbo.users(id,test_key,is_test) VALUES(@p1,@p2,1)`, pair.id, pair.key)
	}
	for _, pair := range []struct {
		conv c.ID
		user c.ID
	}{{f.convA, f.userA}, {f.convB, f.userB}} {
		aiExec(t, db, `INSERT dbo.conversations(id,user_id) VALUES(@p1,@p2)`, pair.conv, pair.user)
	}
	// Fenced WP-02 lifecycle turns in a completed state, so messages are
	// eligible for read.
	aiExec(t, db, `INSERT dbo.turns(id,conversation_id,user_id,input_hash,status,lifecycle_version,user_message_id,assistant_message_id,completed_at)
		VALUES(@p1,@p2,@p3,0x00,N'completed',1,@p4,@p5,SYSUTCDATETIME())`, f.turnA, f.convA, f.userA, c.NewID(), c.NewID())
	aiExec(t, db, `INSERT dbo.turns(id,conversation_id,user_id,input_hash,status,lifecycle_version,user_message_id,assistant_message_id,completed_at)
		VALUES(@p1,@p2,@p3,0x00,N'completed',1,@p4,@p5,SYSUTCDATETIME())`, f.turnB, f.convB, f.userB, c.NewID(), c.NewID())
	store, err := mssql.NewRetrievalStore(db)
	if err != nil {
		t.Fatalf("retrieval store: %v", err)
	}
	f.store = store
	return f
}

func (f retrievalFixture) addMessage(t *testing.T, turn c.ID, conv c.ID, role c.Role, text string, at time.Time) c.ID {
	t.Helper()
	id := c.NewID()
	aiExec(t, f.db, `INSERT dbo.messages(id,conversation_id,turn_id,role,content,created_at,lang) VALUES(@p1,@p2,@p3,@p4,@p5,@p6,N'en')`,
		id, conv, turn, string(role), text, at.UTC())
	return id
}

func (f retrievalFixture) addMemory(t *testing.T, user c.ID, kind c.MemoryKind, status c.MemoryStatus, text string, importance float64, opts ...string) c.ID {
	t.Helper()
	id := c.NewID()
	threads := "[]"
	entities := "[]"
	tags := "[]"
	if len(opts) > 0 {
		threads = opts[0]
	}
	if len(opts) > 1 {
		entities = opts[1]
	}
	if len(opts) > 2 {
		tags = opts[2]
	}
	epistemic := "reported"
	if kind == c.MemHypothesis {
		epistemic = "inferred"
	}
	aiExec(t, f.db, `INSERT dbo.ai_memory_items(id,user_id,kind,content,lang,epistemic,status,confidence,thread_ids_json,entities_json,tags_json,sensitivity,importance,source_json)
		VALUES(@p1,@p2,@p3,@p4,N'en',@p5,@p6,N'low',@p7,@p8,@p9,N'normal',@p10,N'{"prompt_version":"v1","model":"test","origin":"memory_writer"}')`,
		id, user, string(kind), text, epistemic, string(status), threads, entities, tags, importance)
	return id
}

func TestRetrievalSQLServerTemporalWindow(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f := newRetrievalFixture(t, db)
	base := time.Date(2026, 3, 3, 9, 0, 0, 0, time.UTC)
	inWindow := f.addMessage(t, f.turnA, f.convA, c.RoleUser, "I told my boss about the deadline", base.Add(time.Hour))
	alsoInWindow := f.addMessage(t, f.turnA, f.convA, c.RoleUser, "my boss pushed back", base.Add(2*time.Hour))
	before := f.addMessage(t, f.turnA, f.convA, c.RoleUser, "an earlier message", base.Add(-48*time.Hour))
	assistant := f.addMessage(t, f.turnA, f.convA, c.RoleAssistant, "assistant reply in window", base.Add(3*time.Hour))
	foreign := f.addMessage(t, f.turnB, f.convB, c.RoleUser, "another user's message in window", base.Add(time.Hour))

	rows, err := f.store.MessagesInWindow(context.Background(), retrieval.MessageWindowQuery{
		UserID: f.userA, From: base, To: base.Add(24 * time.Hour), Roles: []c.Role{c.RoleUser}, Limit: 10,
	})
	if err != nil {
		t.Fatalf("window read: %v", err)
	}
	seen := map[c.ID]bool{}
	for _, row := range rows {
		seen[row.ID] = true
	}
	if !seen[inWindow] || !seen[alsoInWindow] {
		t.Fatalf("in-window messages missing: %+v", rows)
	}
	if seen[before] {
		t.Fatal("message before the window was returned")
	}
	if seen[assistant] {
		t.Fatal("role filter was not applied")
	}
	if seen[foreign] {
		t.Fatal("cross-owner message leaked through the owner join")
	}
	// Ordering is chronological so the Brain can quote in sequence.
	if len(rows) >= 2 && rows[0].CreatedAt.After(rows[1].CreatedAt) {
		t.Fatal("messages not returned in chronological order")
	}
}

func TestRetrievalSQLServerSoftDeleteVisibility(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f := newRetrievalFixture(t, db)
	base := time.Date(2026, 3, 3, 9, 0, 0, 0, time.UTC)
	visible := f.addMessage(t, f.turnA, f.convA, c.RoleUser, "visible message", base.Add(time.Hour))
	hidden := f.addMessage(t, f.turnA, f.convA, c.RoleUser, "soft-deleted message", base.Add(2*time.Hour))
	aiExec(t, db, `UPDATE dbo.messages SET deleted_at=SYSUTCDATETIME(),deleted_by=@p2,deletion_reason=N'user_request' WHERE id=@p1`, hidden, f.userA)
	// A soft-deleted parent conversation hides its messages too.
	hiddenConv := c.NewID()
	hiddenTurn := c.NewID()
	aiExec(t, db, `INSERT dbo.conversations(id,user_id) VALUES(@p1,@p2)`, hiddenConv, f.userA)
	aiExec(t, db, `INSERT dbo.turns(id,conversation_id,user_id,input_hash,status,lifecycle_version,user_message_id,assistant_message_id,completed_at)
		VALUES(@p1,@p2,@p3,0x00,N'completed',1,@p4,@p5,SYSUTCDATETIME())`, hiddenTurn, hiddenConv, f.userA, c.NewID(), c.NewID())
	underDeleted := f.addMessage(t, hiddenTurn, hiddenConv, c.RoleUser, "message under a deleted conversation", base.Add(3*time.Hour))
	aiExec(t, db, `UPDATE dbo.conversations SET deleted_at=SYSUTCDATETIME(),deleted_by=@p2,deletion_reason=N'user_request' WHERE id=@p1`, hiddenConv, f.userA)

	rows, err := f.store.MessagesInWindow(context.Background(), retrieval.MessageWindowQuery{
		UserID: f.userA, From: base, To: base.Add(24 * time.Hour), Roles: []c.Role{c.RoleUser}, Limit: 10,
	})
	if err != nil {
		t.Fatalf("window read: %v", err)
	}
	seen := map[c.ID]bool{}
	for _, row := range rows {
		seen[row.ID] = true
	}
	if !seen[visible] {
		t.Fatal("visible message missing")
	}
	if seen[hidden] || seen[underDeleted] {
		t.Fatal("soft-deleted record was returned as retrieval context")
	}
	// Soft deletion must preserve the rows: retrieval hides, it never erases.
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.messages WHERE id IN (@p1,@p2)`, hidden, underDeleted); n != 2 {
		t.Fatalf("soft deletion erased rows: %d", n)
	}
}

func TestRetrievalSQLServerMemoryStrategies(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f := newRetrievalFixture(t, db)
	threadID := c.NewID()
	aiExec(t, db, `INSERT dbo.ai_threads(id,user_id,topic,phase,priority) VALUES(@p1,@p2,N'boss and deadlines',N'working',0.9)`, threadID, f.userA)

	linked := f.addMemory(t, f.userA, c.MemEpisode, c.MemActive, "felt anxious before my boss meeting", 0.8,
		`["`+threadID.String()+`"]`)
	sister := f.addMemory(t, f.userA, c.MemFact, c.MemActive, "my sister Lena lives in Tehran", 0.7,
		`[]`, `[{"role":"sister","label":"Lena"}]`)
	bossEntity := f.addMemory(t, f.userA, c.MemFact, c.MemActive, "my boss is called Reza", 0.6,
		`[]`, `[{"role":"boss"}]`)
	proposed := f.addMemory(t, f.userA, c.MemHypothesis, c.MemProposed, "self-worth may depend on approval", 0.6,
		`["`+threadID.String()+`"]`)
	unlinkedHypothesis := f.addMemory(t, f.userA, c.MemHypothesis, c.MemProposed, "an unrelated hypothesis", 0.9)
	rejected := f.addMemory(t, f.userA, c.MemEpisode, c.MemRejected, "rejected content about my boss", 1.0,
		`["`+threadID.String()+`"]`)
	foreignThread := f.addMemory(t, f.userB, c.MemEpisode, c.MemActive, "foreign user's thread memory", 1.0,
		`["`+threadID.String()+`"]`)
	foreignEntity := f.addMemory(t, f.userB, c.MemFact, c.MemActive, "foreign user's sister", 1.0,
		`[]`, `[{"role":"sister","label":"Lena"}]`)

	byThread, err := f.store.MemoryByThread(context.Background(), f.userA, threadID, nil, 20)
	if err != nil {
		t.Fatalf("by thread: %v", err)
	}
	threadIDs := itemIDs(byThread)
	if !threadIDs[linked] || !threadIDs[proposed] {
		t.Fatalf("thread-linked memory missing: %v", threadIDs)
	}
	if threadIDs[rejected] || threadIDs[foreignThread] {
		t.Fatal("ineligible or foreign memory returned by the thread read")
	}

	byEntity, err := f.store.MemoryByEntity(context.Background(), retrieval.EntityQuery{
		UserID: f.userA, Role: "sister", Label: "Lena", Limit: 10,
	})
	if err != nil {
		t.Fatalf("by entity: %v", err)
	}
	entityIDs := itemIDs(byEntity)
	if !entityIDs[sister] {
		t.Fatalf("entity match missing: %v", entityIDs)
	}
	if entityIDs[foreignEntity] || entityIDs[bossEntity] {
		t.Fatal("entity read returned a foreign or non-matching row")
	}

	byRole, err := f.store.MemoryByEntity(context.Background(), retrieval.EntityQuery{
		UserID: f.userA, Role: "boss", Limit: 10,
	})
	if err != nil {
		t.Fatalf("by entity role: %v", err)
	}
	if !itemIDs(byRole)[bossEntity] || itemIDs(byRole)[unlinkedHypothesis] {
		t.Fatal("role-only entity read produced wrong rows")
	}

	cands, err := f.store.MemoryCandidates(context.Background(), retrieval.CandidateQuery{
		UserID: f.userA, Terms: []string{"anxious"}, Limit: 20,
	})
	if err != nil {
		t.Fatalf("candidates: %v", err)
	}
	candIDs := itemIDs(cands)
	if !candIDs[linked] {
		t.Fatalf("lexical candidate missing: %v", candIDs)
	}
	if candIDs[foreignThread] || candIDs[foreignEntity] || candIDs[rejected] {
		t.Fatal("candidate pool breached owner or eligibility filters")
	}

	byIDs, err := f.store.MemoryByIDs(context.Background(), f.userA, []c.ID{linked, foreignThread})
	if err != nil {
		t.Fatalf("by ids: %v", err)
	}
	ids := itemIDs(byIDs)
	if !ids[linked] || ids[foreignThread] {
		t.Fatal("id read did not enforce ownership")
	}
}

func TestRetrievalSQLServerThreadsAndZones(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f := newRetrievalFixture(t, db)
	active := c.NewID()
	resolved := c.NewID()
	foreign := c.NewID()
	aiExec(t, db, `INSERT dbo.ai_threads(id,user_id,topic,phase,priority) VALUES(@p1,@p2,N'active thread',N'working',0.5)`, active, f.userA)
	aiExec(t, db, `INSERT dbo.ai_threads(id,user_id,topic,phase,priority) VALUES(@p1,@p2,N'resolved thread',N'resolved',0.9)`, resolved, f.userA)
	aiExec(t, db, `INSERT dbo.ai_threads(id,user_id,topic,phase,priority) VALUES(@p1,@p2,N'foreign thread',N'working',1.0)`, foreign, f.userB)

	threads, err := f.store.ActiveThreads(context.Background(), f.userA, 5)
	if err != nil {
		t.Fatalf("active threads: %v", err)
	}
	if len(threads) != 1 || threads[0].ID != active || threads[0].UserID != f.userA {
		t.Fatalf("active-thread read returned %+v", threads)
	}
	if _, err = f.store.Thread(context.Background(), f.userA, active); err != nil {
		t.Fatalf("own thread: %v", err)
	}
	if _, err = f.store.Thread(context.Background(), f.userA, foreign); !errors.Is(err, retrieval.ErrNotVisible) {
		t.Fatalf("foreign thread must be reported as not visible, got %v", err)
	}

	// tz_source=user_state: the persisted zone is read from ai_user_state.
	aiExec(t, db, `INSERT dbo.ai_user_state(user_id,tz) VALUES(@p1,N'Asia/Tehran')`, f.userA)
	tz, err := f.store.UserTimeZone(context.Background(), f.userA)
	if err != nil || tz != "Asia/Tehran" {
		t.Fatalf("user time zone: %q %v", tz, err)
	}
	// A user with no state row defaults to UTC instead of an error.
	tz, err = f.store.UserTimeZone(context.Background(), f.userB)
	if err != nil || tz != "UTC" {
		t.Fatalf("default time zone: %q %v", tz, err)
	}
}

func TestRetrievalSQLServerEmbeddingsAndMissingVectors(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f := newRetrievalFixture(t, db)
	withVector := f.addMemory(t, f.userA, c.MemEpisode, c.MemActive, "my boss moved the deadline", 0.8)
	withoutVector := f.addMemory(t, f.userA, c.MemEpisode, c.MemActive, "my boss moved the deadline before", 0.7)
	foreign := f.addMemory(t, f.userB, c.MemEpisode, c.MemActive, "foreign memory", 0.9)
	raw, err := mssql.EncodeEmbedding("test-embed", []float32{0, 3, 4})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	aiExec(t, db, `INSERT dbo.ai_embeddings(id,user_id,ref_kind,ref_id,model_id,dim,vector_bytes)
		VALUES(@p1,@p2,N'memory',@p3,N'test-embed',3,@p4)`, c.NewID(), f.userA, withVector, raw)
	aiExec(t, db, `INSERT dbo.ai_embeddings(id,user_id,ref_kind,ref_id,model_id,dim,vector_bytes)
		VALUES(@p1,@p2,N'memory',@p3,N'test-embed',3,@p4)`, c.NewID(), f.userB, foreign, raw)

	vectors, err := f.store.Embeddings(context.Background(), f.userA, []retrieval.EmbeddingRef{
		{ID: withVector}, {ID: withoutVector}, {ID: foreign},
	})
	if err != nil {
		t.Fatalf("embedding read: %v", err)
	}
	if len(vectors) != 1 {
		t.Fatalf("expected exactly the owner's stored vector, got %d", len(vectors))
	}
	if vectors[0].RefID != withVector || vectors[0].ModelID != "test-embed" || vectors[0].Dimensions != 3 {
		t.Fatalf("unexpected vector metadata: %+v", vectors[0])
	}
	// The decoded vector is unit-normalized by the storage codec.
	if got := vectors[0].Vector[1]; got < 0.59 || got > 0.61 {
		t.Fatalf("vector not decoded: %v", vectors[0].Vector)
	}
	// An empty ref set is not an error and issues no query.
	if out, err := f.store.Embeddings(context.Background(), f.userA, nil); err != nil || len(out) != 0 {
		t.Fatalf("empty ref set: %v %v", out, err)
	}
}

// TestRetrievalSQLServerEndToEnd runs the real Service against the real adapter
// to prove the ranking pipeline works with stored data, not just fakes.
func TestRetrievalSQLServerEndToEnd(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f := newRetrievalFixture(t, db)
	base := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	threadID := c.NewID()
	aiExec(t, db, `INSERT dbo.ai_threads(id,user_id,topic,phase,priority) VALUES(@p1,@p2,N'work stress',N'working',0.8)`, threadID, f.userA)
	linked := f.addMemory(t, f.userA, c.MemEpisode, c.MemActive, "my boss moved the deadline again", 0.9,
		`["`+threadID.String()+`"]`)
	foreign := f.addMemory(t, f.userB, c.MemEpisode, c.MemActive, "my boss moved the deadline again", 1.0,
		`["`+threadID.String()+`"]`)
	raw, _ := mssql.EncodeEmbedding("test-embed", []float32{0, 1, 0})
	aiExec(t, db, `INSERT dbo.ai_embeddings(id,user_id,ref_kind,ref_id,model_id,dim,vector_bytes)
		VALUES(@p1,@p2,N'memory',@p3,N'test-embed',3,@p4)`, c.NewID(), f.userA, linked, raw)

	reg := &stubRegistry{files: map[string][]byte{
		"config/retrieval.yaml": []byte(retrievalYAMLFixture),
		"config/budgets.yaml":   []byte(budgetsYAMLFixture),
	}}
	svc, err := retrieval.New(f.store, reg, stubEmbedder{})
	if err != nil {
		t.Fatalf("service: %v", err)
	}
	result, err := svc.Retrieve(context.Background(), c.RetrievalRequest{
		UserID: f.userA, FocusThread: &threadID, MaxItems: 5,
		Turn: c.TurnView{TurnID: c.NewID(), MessageID: c.NewID(), Text: "my boss moved the deadline", Lang: "en"},
	})
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if result.Meta.Degraded {
		t.Fatalf("retrieval degraded unexpectedly: %+v", result.Meta)
	}
	found := false
	for _, item := range result.Items {
		if item.ID == linked {
			found = true
		}
		if item.ID == foreign {
			t.Fatal("cross-user memory returned by the end-to-end pipeline")
		}
	}
	if !found {
		t.Fatalf("owner memory missing: %+v", result.Items)
	}
	if len(result.Items) > 5 {
		t.Fatalf("top-K violated: %d", len(result.Items))
	}
	_ = base
}

func itemIDs(items []c.MemoryItem) map[c.ID]bool {
	out := make(map[c.ID]bool, len(items))
	for _, item := range items {
		out[item.ID] = true
	}
	return out
}

// ---- test-only bundle and embedding doubles ----

type stubRegistry struct{ files map[string][]byte }

func (r *stubRegistry) Active(context.Context, c.ID) (ports.Bundle, error) {
	return stubBundle{files: r.files}, nil
}

type stubBundle struct{ files map[string][]byte }

func (b stubBundle) Reference() c.BundleRef { return c.BundleRef{Version: "kb@it", PromptSHAs: map[string]string{}} }
func (b stubBundle) Artifact(_ context.Context, name string) ([]byte, error) {
	raw, ok := b.files[name]
	if !ok {
		return nil, errors.New("artifact not found")
	}
	return raw, nil
}

type stubEmbedder struct{}

func (stubEmbedder) Embed(_ context.Context, text string) ([]float32, string, error) {
	low := strings.ToLower(text)
	switch {
	case strings.Contains(low, "boss") || strings.Contains(low, "work"):
		return []float32{0, 1, 0}, "test-embed", nil
	case strings.Contains(low, "calm") || strings.Contains(low, "sleep"):
		return []float32{0, 0, 1}, "test-embed", nil
	default:
		return []float32{1, 0, 0}, "test-embed", nil
	}
}

const retrievalYAMLFixture = `schema: retrieval.v1
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

const budgetsYAMLFixture = `schema: budgets.v1
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
