package integration

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	aiembed "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/embed"
	aimssql "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	retrieval "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	aicontract "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	domain "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/domain/retrieval"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/database"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/migrations"
)

// TestRetrievalSQLServer exercises the WP-09 retrieval read model against a real
// SQL Server 2022 instance. It follows the repository convention: opt-in via
// WELLBEING_INTEGRATION=true, APP_ENV=local, and a disposable database that is
// created and dropped per run. It never touches developer data.
func TestRetrievalSQLServer(t *testing.T) {
	if os.Getenv("WELLBEING_INTEGRATION") != "true" {
		t.Skip("SQL Server integration opt-in (set WELLBEING_INTEGRATION=true with local DATABASE_URL)")
	}
	parsed, err := url.Parse(os.Getenv("DATABASE_URL"))
	if err != nil || parsed.Scheme != "sqlserver" || os.Getenv("APP_ENV") != "local" {
		t.Fatal("integration requires APP_ENV=local and a sqlserver DATABASE_URL")
	}
	if host := strings.ToLower(parsed.Hostname()); host != "127.0.0.1" && host != "localhost" && host != "mssql" {
		t.Fatal("local-only SQL Server required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	name := "wellbeing_retrieval_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	q := parsed.Query()
	q.Set("database", name)
	parsed.RawQuery = q.Encode()
	if err := database.EnsureLocalDatabase(ctx, parsed.String()); err != nil {
		t.Fatal(err)
	}
	q.Set("database", "master")
	masterURL := *parsed
	masterURL.RawQuery = q.Encode()
	master, err := database.Connect(ctx, masterURL.String())
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer func() {
		if _, e := master.ExecContext(ctx, "ALTER DATABASE ["+name+"] SET SINGLE_USER WITH ROLLBACK IMMEDIATE; DROP DATABASE ["+name+"]"); e != nil {
			t.Errorf("drop isolated database: %v", e)
		}
	}()
	db, err := database.Connect(ctx, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrations.Apply(ctx, db, "../../db/migrations"); err != nil {
		t.Fatal(err)
	}
	// The runner is checksummed and idempotent: a second apply must be a no-op.
	if err := migrations.Apply(ctx, db, "../../db/migrations"); err != nil {
		t.Fatalf("repeat migration: %v", err)
	}

	alice, bob := uuid.New(), uuid.New()
	seedUsers(t, ctx, db, alice, bob)
	aliceConv := seedConversation(t, ctx, db, alice, "گفتگوی ساختگی برای بازیابی")
	bobConv := seedConversation(t, ctx, db, bob, "another synthetic conversation")

	aliceThread := uuid.New()
	execOrFail(t, ctx, db, `INSERT dbo.ai_threads(id,user_id,topic,phase,problem_tags_json)
        VALUES(CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),N'کار و مقایسه',N'exploring',N'["work"]')`,
		aliceThread.String(), alice.String())

	// --- memories for Alice -------------------------------------------------
	matchID := uuid.New()
	deletedID := uuid.New()
	supersededID := uuid.New()
	hypothesisID := uuid.New()
	contestedHypID := uuid.New()
	highlySensitiveID := uuid.New()
	preferenceID := uuid.New()
	entityID := uuid.New()
	noEvidenceID := uuid.New()
	clusterA, clusterB := uuid.New(), uuid.New()
	clusterID := uuid.New()

	insertMemory(t, ctx, db, memoryRow{ID: matchID, User: alice, Kind: "fact", Epistemic: "reported", Status: "active",
		Content: "manager criticized my work in front of the team", Importance: 0.9, Threads: []uuid.UUID{aliceThread},
		Entities: []string{"boss"}, Tags: []string{"work", "criticism"}})
	insertMemory(t, ctx, db, memoryRow{ID: deletedID, User: alice, Kind: "fact", Epistemic: "reported", Status: "active",
		Content: "manager criticized my work in front of the team", Importance: 0.9, Deleted: true})
	insertMemory(t, ctx, db, memoryRow{ID: supersededID, User: alice, Kind: "fact", Epistemic: "reported", Status: "superseded",
		Content: "I am a student", Importance: 0.5})
	insertMemory(t, ctx, db, memoryRow{ID: hypothesisID, User: alice, Kind: "hypothesis", Epistemic: "inferred", Status: "supported",
		Content: "may associate work criticism with self-worth", Importance: 0.8, Threads: []uuid.UUID{aliceThread}})
	insertMemory(t, ctx, db, memoryRow{ID: contestedHypID, User: alice, Kind: "hypothesis", Epistemic: "inferred", Status: "contested",
		Content: "may avoid conflict with authority", Importance: 0.9})
	insertMemory(t, ctx, db, memoryRow{ID: highlySensitiveID, User: alice, Kind: "event", Epistemic: "reported", Status: "active",
		Content: "highly sensitive synthetic disclosure", Importance: 1, Sensitivity: "highly_sensitive"})
	insertMemory(t, ctx, db, memoryRow{ID: preferenceID, User: alice, Kind: "preference", Epistemic: "reported", Status: "active",
		Content: "prefers short replies", Importance: 0.6})
	insertMemory(t, ctx, db, memoryRow{ID: entityID, User: alice, Kind: "event", Epistemic: "reported", Status: "active",
		Content: "my sister moved away", Importance: 0.4, Entities: []string{"sister"}})
	insertMemory(t, ctx, db, memoryRow{ID: noEvidenceID, User: alice, Kind: "fact", Epistemic: "reported", Status: "active",
		Content: "claim with no provenance", Importance: 1, NoEvidence: true})
	insertMemory(t, ctx, db, memoryRow{ID: clusterA, User: alice, Kind: "episode", Epistemic: "reported", Status: "active",
		Content: "argument with my manager about the deadline", Importance: 0.8, Cluster: &clusterID})
	insertMemory(t, ctx, db, memoryRow{ID: clusterB, User: alice, Kind: "episode", Epistemic: "reported", Status: "active",
		Content: "argument with my manager about the deadline (repeat)", Importance: 0.7, Cluster: &clusterID})

	// --- Bob's private memory that must never leak --------------------------
	bobMemoryID := uuid.New()
	insertMemory(t, ctx, db, memoryRow{ID: bobMemoryID, User: bob, Kind: "fact", Epistemic: "reported", Status: "active",
		Content: "BOB_PRIVATE manager criticized my work in front of the team", Importance: 1})

	// --- messages ----------------------------------------------------------
	target := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC) // "last Tuesday" for the fixed clock below
	inWindow := seedMessage(t, ctx, db, alice, aliceConv, aicontract.RoleUser, "my boss told me the review went badly", target)
	outsideWindow := seedMessage(t, ctx, db, alice, aliceConv, aicontract.RoleUser, "unrelated earlier text", target.AddDate(0, 0, -7))
	suppressed := seedMessage(t, ctx, db, alice, aliceConv, aicontract.RoleUser, "my boss suppressed message", target.Add(time.Hour))
	bobMessage := seedMessage(t, ctx, db, bob, bobConv, aicontract.RoleUser, "BOB_PRIVATE my boss", target)
	execOrFail(t, ctx, db, `INSERT dbo.ai_message_suppressions(message_id,user_id,reason)
        VALUES(CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),N'user_request')`, suppressed.String(), alice.String())

	// --- embeddings --------------------------------------------------------
	writeEmbedding(t, ctx, db, alice, aicontract.RefKindMemory, matchID, "deterministic-test-embed", []float32{1, 0, 0, 0})
	writeEmbedding(t, ctx, db, alice, aicontract.RefKindMemory, hypothesisID, "deterministic-test-embed", []float32{0, 1, 0, 0})
	// A vector from a different model must never be compared with the query one.
	writeEmbedding(t, ctx, db, alice, aicontract.RefKindMemory, entityID, "legacy-model", []float32{1, 0, 0, 0})
	// A corrupt row (wrong byte length for its declared dimension) must be skipped.
	if _, err := db.ExecContext(ctx, `INSERT dbo.ai_embeddings(id,user_id,ref_kind,ref_id,model_id,dim,vector_bytes)
        VALUES(NEWID(),CONVERT(uniqueidentifier,@p1),N'memory',CONVERT(uniqueidentifier,@p2),N'deterministic-test-embed',4,@p3)`,
		alice.String(), preferenceID.String(), []byte{1, 2}); err != nil {
		t.Fatal(err)
	}

	clock := fixedRetrievalClock{now: target.Add(7 * 24 * time.Hour)}
	repo := aimssql.New(db)

	t.Run("repository scopes to the owner", func(t *testing.T) {
		mine, err := repo.RecentMemory(ctx, alice, 100)
		if err != nil {
			t.Fatal(err)
		}
		for _, rec := range mine {
			if rec.UserID != alice {
				t.Fatalf("foreign row returned: %s", rec.ID)
			}
			if strings.Contains(rec.Content, "BOB_PRIVATE") {
				t.Fatal("cross-user content leaked")
			}
		}
		theirs, err := repo.RecentMemory(ctx, bob, 100)
		if err != nil {
			t.Fatal(err)
		}
		if len(theirs) != 1 || theirs[0].ID != bobMemoryID {
			t.Fatalf("bob must see only his own row, got %d rows", len(theirs))
		}
	})

	t.Run("soft-deleted rows are never read", func(t *testing.T) {
		rows, err := repo.RecentMemory(ctx, alice, 100)
		if err != nil {
			t.Fatal(err)
		}
		for _, rec := range rows {
			if rec.ID == deletedID || rec.DeletedAt != nil {
				t.Fatal("soft-deleted memory surfaced")
			}
		}
		// The row still exists: deletion is a flag, never a DELETE.
		var count int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.ai_memory_items
            WHERE id=CONVERT(uniqueidentifier,@p1) AND deleted_at IS NOT NULL`, deletedID.String()).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatal("soft delete must retain the row")
		}
	})

	t.Run("thread and entity lookups", func(t *testing.T) {
		items, err := repo.MemoryByThread(ctx, alice, aliceThread, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 2 {
			t.Fatalf("want 2 thread items, got %d", len(items))
		}
		for _, rec := range items {
			if rec.UserID != alice {
				t.Fatal("foreign thread item")
			}
		}
		// Bob may not read Alice's thread items even if he guesses the thread id.
		if foreign, err := repo.MemoryByThread(ctx, bob, aliceThread, 10); err != nil || len(foreign) != 0 {
			t.Fatalf("cross-user thread read leaked: %v %v", foreign, err)
		}
		sister, err := repo.MemoryByEntity(ctx, alice, retrieval.EntityFilter{Role: "sister"}, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(sister) != 1 || sister[0].ID != entityID {
			t.Fatalf("entity lookup failed: %+v", sister)
		}
		if foreign, err := repo.MemoryByEntity(ctx, bob, retrieval.EntityFilter{Role: "sister"}, 10); err != nil || len(foreign) != 0 {
			t.Fatalf("cross-user entity read leaked: %v %v", foreign, err)
		}
	})

	t.Run("message window is owner-scoped and excludes suppressions", func(t *testing.T) {
		msgs, err := repo.MessageWindow(ctx, alice, target.Add(-12*time.Hour), target.Add(12*time.Hour),
			[]aicontract.Role{aicontract.RoleUser}, nil, 12)
		if err != nil {
			t.Fatal(err)
		}
		var ids []uuid.UUID
		for _, m := range msgs {
			if m.UserID != alice {
				t.Fatal("foreign message returned")
			}
			ids = append(ids, m.ID)
		}
		if len(ids) != 1 || ids[0] != inWindow {
			t.Fatalf("window should return exactly the in-window, non-suppressed message: %v", ids)
		}
		// Bob's window never sees Alice's rows.
		bobMsgs, err := repo.MessageWindow(ctx, bob, target.Add(-12*time.Hour), target.Add(12*time.Hour),
			[]aicontract.Role{aicontract.RoleUser}, nil, 12)
		if err != nil {
			t.Fatal(err)
		}
		if len(bobMsgs) != 1 || bobMsgs[0].ID != bobMessage {
			t.Fatalf("bob window incorrect: %+v", bobMsgs)
		}
	})

	t.Run("embeddings decode and reject corrupt rows", func(t *testing.T) {
		got, err := repo.Embeddings(ctx, alice, []aicontract.EmbeddingKey{
			{Kind: aicontract.RefKindMemory, ID: matchID},
			{Kind: aicontract.RefKindMemory, ID: preferenceID},
			{Kind: aicontract.RefKindMemory, ID: hypothesisID},
		})
		if err != nil {
			t.Fatal(err)
		}
		if v, ok := got[aicontract.EmbeddingKey{Kind: aicontract.RefKindMemory, ID: matchID}]; !ok || v.Dim != 4 || v.Values[0] != 1 {
			t.Fatalf("stored vector not decoded: %+v", v)
		}
		if _, ok := got[aicontract.EmbeddingKey{Kind: aicontract.RefKindMemory, ID: preferenceID}]; ok {
			t.Fatal("a corrupt (length/dim mismatch) row must be skipped")
		}
		if foreign, err := repo.Embeddings(ctx, bob, []aicontract.EmbeddingKey{{Kind: aicontract.RefKindMemory, ID: matchID}}); err != nil || len(foreign) != 0 {
			t.Fatalf("cross-user embedding read leaked: %v %v", foreign, err)
		}
	})

	t.Run("service retrieval end to end", func(t *testing.T) {
		svc := newRetrievalService(t, repo, clock)
		res, err := svc.Retrieve(ctx, aicontract.RetrievalRequest{
			UserID: alice,
			Turn: aicontract.TurnView{
				TurnID: uuid.New(), MessageID: uuid.New(), ConversationID: aliceConv,
				Text: "my manager criticized my work again", Lang: "en",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if res.Meta.Degraded {
			t.Fatalf("retrieval degraded unexpectedly: %+v", res.Meta)
		}
		if len(res.Items) == 0 {
			t.Fatal("expected retrieved context")
		}
		if len(res.Items) > svc.Tunables().MaxItems {
			t.Fatalf("top-k exceeded: %d", len(res.Items))
		}
		seen := map[uuid.UUID]bool{}
		for _, item := range res.Items {
			seen[item.ID] = true
			if item.ID == deletedID {
				t.Fatal("soft-deleted item retrieved")
			}
			if item.ID == supersededID {
				t.Fatal("superseded item retrieved")
			}
			if item.ID == contestedHypID {
				t.Fatal("contested hypothesis retrieved")
			}
			if item.ID == highlySensitiveID {
				t.Fatal("highly_sensitive item retrieved by default")
			}
			if item.ID == noEvidenceID {
				t.Fatal("item without provenance retrieved")
			}
			if item.ID == bobMemoryID || strings.Contains(item.Text, "BOB_PRIVATE") {
				t.Fatal("cross-user leakage")
			}
		}
		if !seen[matchID] {
			t.Fatalf("the strongly matching item must be retrieved: %+v", res.Items)
		}
		if len(seen) != len(res.Items) {
			t.Fatal("duplicate items returned")
		}
	})

	t.Run("cluster duplicates collapse", func(t *testing.T) {
		svc := newRetrievalService(t, repo, clock)
		res, err := svc.Retrieve(ctx, aicontract.RetrievalRequest{
			UserID: alice,
			Turn: aicontract.TurnView{
				TurnID: uuid.New(), MessageID: uuid.New(), ConversationID: aliceConv,
				Text: "argument with my manager about the deadline",
			},
			MaxItems: 7,
		})
		if err != nil {
			t.Fatal(err)
		}
		both := 0
		for _, item := range res.Items {
			if item.ID == clusterA || item.ID == clusterB {
				both++
			}
		}
		if both > 1 {
			t.Fatal("cluster duplicates must collapse to one representative")
		}
	})

	t.Run("temporal exact recall", func(t *testing.T) {
		svc := newRetrievalService(t, repo, clock)
		res, err := svc.Retrieve(ctx, aicontract.RetrievalRequest{
			UserID: alice,
			Turn: aicontract.TurnView{
				TurnID: uuid.New(), MessageID: uuid.New(), ConversationID: aliceConv,
				Text: "what did I tell you last Tuesday about my boss", Lang: "en",
			},
			MaxItems: 7,
		})
		if err != nil {
			t.Fatal(err)
		}
		if res.Meta.ExactMatchFound == nil || !*res.Meta.ExactMatchFound {
			t.Fatalf("expected an exact match: %+v", res.Meta)
		}
		var found bool
		for _, item := range res.Items {
			if item.ID == inWindow {
				found = true
				if !item.Verbatim {
					t.Fatal("temporal hit must be verbatim")
				}
			}
			if item.ID == outsideWindow {
				t.Fatal("out-of-window message returned")
			}
			if item.ID == suppressed {
				t.Fatal("suppressed message returned")
			}
			if item.ID == bobMessage {
				t.Fatal("foreign message returned")
			}
		}
		if !found {
			t.Fatalf("in-window message missing: %+v", res.Items)
		}
	})

	t.Run("empty temporal window reports not found", func(t *testing.T) {
		svc := newRetrievalService(t, repo, clock)
		res, err := svc.Retrieve(ctx, aicontract.RetrievalRequest{
			UserID: alice,
			Turn: aicontract.TurnView{
				TurnID: uuid.New(), MessageID: uuid.New(), ConversationID: aliceConv,
				Text: "what did I tell you last Wednesday about my boss",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if res.Meta.ExactMatchFound == nil || *res.Meta.ExactMatchFound {
			t.Fatalf("a day with no messages must not claim a match: %+v", res.Meta)
		}
	})

	t.Run("preferences are the always-on core set", func(t *testing.T) {
		prefs, err := repo.Preferences(ctx, alice, 3)
		if err != nil {
			t.Fatal(err)
		}
		if len(prefs) != 1 || prefs[0].ID != preferenceID {
			t.Fatalf("preference core set incorrect: %+v", prefs)
		}
		if foreign, err := repo.Preferences(ctx, bob, 3); err != nil || len(foreign) != 0 {
			t.Fatalf("cross-user preference leak: %v %v", foreign, err)
		}
	})

	t.Run("recent messages feed the novelty window", func(t *testing.T) {
		ctx2 := context.Background()
		texts, err := repo.RecentMessages(ctx2, alice, aliceConv, 8)
		if err != nil {
			t.Fatal(err)
		}
		for _, text := range texts {
			if strings.Contains(text, "suppressed message") {
				t.Fatal("suppressed message appeared in the novelty window")
			}
		}
		if foreign, err := repo.RecentMessages(ctx2, bob, aliceConv, 8); err != nil || len(foreign) != 0 {
			t.Fatalf("cross-user conversation read leaked: %v %v", foreign, err)
		}
	})

	t.Run("nil or invalid identifiers are rejected", func(t *testing.T) {
		if _, err := repo.RecentMemory(ctx, uuid.Nil, 5); err == nil {
			t.Fatal("a nil user id must be rejected")
		}
		if _, err := repo.MemoryByThread(ctx, alice, uuid.Nil, 5); err == nil {
			t.Fatal("a nil thread id must be rejected")
		}
		if _, err := repo.MessageWindow(ctx, uuid.Nil, target, target.Add(time.Hour), nil, nil, 5); err == nil {
			t.Fatal("a nil user id must be rejected for message windows")
		}
		if msgs, err := repo.MessageWindow(ctx, alice, target, target, nil, nil, 5); err != nil || len(msgs) != 0 {
			t.Fatalf("an empty window returns nothing without error: %v %v", msgs, err)
		}
	})
}

// fixedRetrievalClock pins "now" so weekday parsing is deterministic.
type fixedRetrievalClock struct{ now time.Time }

func (c fixedRetrievalClock) Now() time.Time { return c.now }

func newRetrievalService(t *testing.T, repo *aimssql.Repository, clock fixedRetrievalClock) *retrieval.Service {
	t.Helper()
	tu := domain.Defaults()
	tu.MinScore = 0.05 // exercise ranking, not the production floor
	svc, err := retrieval.New(retrieval.Config{
		Store:    repo,
		Embedder: aiembed.NewDeterministic(4),
		Clock:    clock,
		Tunables: tu,
		// Reference moment is Tuesday 2026-11-10 09:00 UTC -> "last Tuesday"
		// must resolve to 2026-11-10 in the fixture timezone.
		Location:           time.UTC,
		WeekStartsOnMonday: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

type memoryRow struct {
	ID          uuid.UUID
	User        uuid.UUID
	Kind        string
	Epistemic   string
	Status      string
	Content     string
	Sensitivity string
	Importance  float32
	Threads     []uuid.UUID
	Entities    []string
	Tags        []string
	Cluster     *uuid.UUID
	Deleted     bool
	NoEvidence  bool
}

func insertMemory(t *testing.T, ctx context.Context, db *sql.DB, row memoryRow) {
	t.Helper()
	if row.Sensitivity == "" {
		row.Sensitivity = "normal"
	}
	threads := "[]"
	if len(row.Threads) > 0 {
		parts := make([]string, 0, len(row.Threads))
		for _, id := range row.Threads {
			parts = append(parts, `"`+id.String()+`"`)
		}
		threads = "[" + strings.Join(parts, ",") + "]"
	}
	entities := "[]"
	if len(row.Entities) > 0 {
		parts := make([]string, 0, len(row.Entities))
		for _, e := range row.Entities {
			parts = append(parts, `{"role":"`+e+`","label":""}`)
		}
		entities = "[" + strings.Join(parts, ",") + "]"
	}
	tags := "[]"
	if len(row.Tags) > 0 {
		parts := make([]string, 0, len(row.Tags))
		for _, tag := range row.Tags {
			parts = append(parts, `"`+tag+`"`)
		}
		tags = "[" + strings.Join(parts, ",") + "]"
	}
	cluster := sql.NullString{}
	if row.Cluster != nil {
		cluster = sql.NullString{String: row.Cluster.String(), Valid: true}
	}
	deletedAt := sql.NullTime{}
	if row.Deleted {
		deletedAt = sql.NullTime{Time: time.Now().UTC(), Valid: true}
	}
	execOrFail(t, ctx, db, `INSERT dbo.ai_memory_items(
        id,user_id,kind,content,epistemic,status,sensitivity,importance,
        thread_ids_json,entities_json,tags_json,cluster_id,deleted_at)
      VALUES(CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),@p3,@p4,@p5,@p6,@p7,@p8,
        @p9,@p10,@p11,
        CASE WHEN @p12 IS NULL THEN NULL ELSE CONVERT(uniqueidentifier,@p12) END,
        CASE WHEN @p13 IS NULL THEN NULL ELSE CONVERT(datetime2(3),@p13) END)`,
		row.ID.String(), row.User.String(), row.Kind, row.Content, row.Epistemic, row.Status, row.Sensitivity, row.Importance,
		threads, entities, tags, nullableString(cluster), nullableTime(deletedAt))
	if !row.NoEvidence {
		execOrFail(t, ctx, db, `INSERT dbo.ai_memory_evidence(item_id,ref_kind,ref_id,role,evidence_at)
            VALUES(CONVERT(uniqueidentifier,@p1),N'message',@p2,N'supports',SYSUTCDATETIME())`,
			row.ID.String(), uuid.New().String())
	}
}

func seedUsers(t *testing.T, ctx context.Context, db *sql.DB, ids ...uuid.UUID) {
	t.Helper()
	for _, id := range ids {
		execOrFail(t, ctx, db, `INSERT dbo.users(id,test_key,is_test) VALUES(CONVERT(uniqueidentifier,@p1),@p2,1)`,
			id.String(), "retrieval-"+id.String())
	}
}

func seedConversation(t *testing.T, ctx context.Context, db *sql.DB, user uuid.UUID, title string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	execOrFail(t, ctx, db, `INSERT dbo.conversations(id,user_id,title)
        VALUES(CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),@p3)`, id.String(), user.String(), title)
	return id
}

func seedMessage(t *testing.T, ctx context.Context, db *sql.DB, user, conversation uuid.UUID, role aicontract.Role, content string, at time.Time) uuid.UUID {
	t.Helper()
	turnID, msgID := uuid.New(), uuid.New()
	execOrFail(t, ctx, db, `INSERT dbo.turns(id,conversation_id,user_id,input_hash,status,completed_at)
        VALUES(CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),CONVERT(uniqueidentifier,@p3),@p4,N'completed',SYSUTCDATETIME())`,
		turnID.String(), conversation.String(), user.String(), make([]byte, 32))
	execOrFail(t, ctx, db, `INSERT dbo.messages(id,conversation_id,turn_id,role,content,created_at)
        VALUES(CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),CONVERT(uniqueidentifier,@p3),@p4,@p5,@p6)`,
		msgID.String(), conversation.String(), turnID.String(), string(role), content, at)
	return msgID
}

func writeEmbedding(t *testing.T, ctx context.Context, db *sql.DB, user uuid.UUID, kind string, ref uuid.UUID, model string, values []float32) {
	t.Helper()
	blob, err := aimssql.EncodeVector(values)
	if err != nil {
		t.Fatal(err)
	}
	execOrFail(t, ctx, db, `INSERT dbo.ai_embeddings(id,user_id,ref_kind,ref_id,model_id,dim,vector_bytes)
        VALUES(NEWID(),CONVERT(uniqueidentifier,@p1),@p2,CONVERT(uniqueidentifier,@p3),@p4,@p5,@p6)`,
		user.String(), kind, ref.String(), model, len(values), blob)
}

func execOrFail(t *testing.T, ctx context.Context, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(ctx, query, args...); err != nil {
		t.Fatalf("exec failed: %v", err)
	}
}

func nullableString(v sql.NullString) any {
	if !v.Valid {
		return nil
	}
	return v.String
}

func nullableTime(v sql.NullTime) any {
	if !v.Valid {
		return nil
	}
	return v.Time
}
