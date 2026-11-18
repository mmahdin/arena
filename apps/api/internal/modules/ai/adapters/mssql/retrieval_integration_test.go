package mssql

import (
	"context"
	"crypto/rand"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/microsoft/go-mssqldb"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// Run ONLY against a disposable, fully migrated SQL Server 2022 database:
// THEIA_WP09_SQLSERVER_DSN='sqlserver://...' THEIA_WP09_SQLSERVER_ALLOW_FIXTURES=1
// go test ./internal/modules/ai/adapters/mssql -run TestWP09SQLServerIntegration
// Every fixture is written in one transaction and rolled back. No DELETE.
func TestWP09SQLServerIntegration(t *testing.T) {
	dsn := os.Getenv("THEIA_WP09_SQLSERVER_DSN")
	if dsn == "" || os.Getenv("THEIA_WP09_SQLSERVER_ALLOW_FIXTURES") != "1" {
		t.Skip("SQL Server fixture DSN/explicit opt-in unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	r, err := NewRetrievalRepositoryTx(tx)
	if err != nil {
		t.Fatal(err)
	}
	ownerA, ownerB := c.NewID(), c.NewID()
	convA, convB := c.NewID(), c.NewID()
	earlierA, triggerTurn, otherTurn := c.NewID(), c.NewID(), c.NewID()
	msgA, trigger, otherMsg := c.NewID(), c.NewID(), c.NewID()
	memoryA, memoryB, embeddingA, thread := c.NewID(), c.NewID(), c.NewID(), c.NewID()
	// Unique synthetic users and explicit memory-unpaused read state.
	for _, u := range []c.ID{ownerA, ownerB} {
		if _, err = tx.ExecContext(ctx, `INSERT INTO dbo.users(id,test_key,is_test) VALUES(@p1,@p2,1)`, u, "wp09-"+u.String()); err != nil {
			t.Fatal(err)
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO dbo.ai_user_state(user_id,tz,locale,memory_paused) VALUES(@p1,N'UTC',N'en',0)`, u); err != nil {
			t.Fatal(err)
		}
	}
	for _, v := range []struct{ conversation, owner c.ID }{{convA, ownerA}, {convB, ownerB}} {
		if _, err = tx.ExecContext(ctx, `INSERT INTO dbo.conversations(id,user_id) VALUES(@p1,@p2)`, v.conversation, v.owner); err != nil {
			t.Fatal(err)
		}
	}
	insertTurn := func(id, conv, owner, message c.ID) {
		t.Helper()
		h := make([]byte, 32)
		if _, e := rand.Read(h); e != nil {
			t.Fatal(e)
		}
		if _, e := tx.ExecContext(ctx, `INSERT INTO dbo.turns(id,conversation_id,user_id,input_hash,status,user_message_id,client_turn_id,lifecycle_version)
            VALUES(@p1,@p2,@p3,@p4,N'running',@p5,@p6,1)`, id, conv, owner, h, message, c.NewID()); e != nil {
			t.Fatal(e)
		}
	}
	insertTurn(earlierA, convA, ownerA, msgA)
	insertTurn(otherTurn, convB, ownerB, otherMsg)
	if _, err = tx.ExecContext(ctx, `INSERT INTO dbo.messages(id,conversation_id,turn_id,role,content)
        VALUES(@p1,@p2,@p3,N'user',N'My boss was supportive')`, msgA, convA, earlierA); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO dbo.messages(id,conversation_id,turn_id,role,content)
        VALUES(@p1,@p2,@p3,N'user',N'Other user secret')`, otherMsg, convB, otherTurn); err != nil {
		t.Fatal(err)
	}
	evidenceSource := `{"origin":"synthetic_wp09_test","model":"none"}`
	entities := `[{"role":"boss"}]`
	for _, v := range []struct {
		id, owner c.ID
		text      string
	}{{memoryA, ownerA, "my boss was supportive"}, {memoryB, ownerB, "other user's memory"}} {
		threads := "[]"
		if v.owner == ownerA {
			threads = `["` + thread.String() + `"]`
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO dbo.ai_memory_items(id,user_id,kind,content,epistemic,status,thread_ids_json,entities_json,source_json)
            VALUES(@p1,@p2,N'episode',@p3,N'reported',N'active',@p4,@p5,@p6)`, v.id, v.owner, v.text, threads, entities, evidenceSource); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO dbo.ai_memory_evidence(user_id,item_id,ref_kind,ref_id,role,evidence_at)
        VALUES(@p1,@p2,N'message',@p3,N'supports',SYSUTCDATETIME())`, ownerA, memoryA, msgA); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO dbo.ai_memory_evidence(user_id,item_id,ref_kind,ref_id,role,evidence_at)
        VALUES(@p1,@p2,N'message',@p3,N'supports',SYSUTCDATETIME())`, ownerB, memoryB, otherMsg); err != nil {
		t.Fatal(err)
	}
	vector, err := EncodeEmbedding("synthetic-wp09", []float32{3, 4})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO dbo.ai_embeddings(id,user_id,ref_kind,ref_id,codec_version,model_id,dim,vector_bytes)
        VALUES(@p1,@p2,N'memory',@p3,1,N'synthetic-wp09',2,@p4)`, embeddingA, ownerA, memoryA, vector); err != nil {
		t.Fatal(err)
	}
	insertTurn(triggerTurn, convA, ownerA, trigger)
	if _, err = tx.ExecContext(ctx, `INSERT INTO dbo.messages(id,conversation_id,turn_id,role,content)
        VALUES(@p1,@p2,@p3,N'user',N'What did I say before?')`, trigger, convA, triggerTurn); err != nil {
		t.Fatal(err)
	}
	state, err := r.UserState(ctx, ownerA)
	if err != nil || state.MemoryPaused || state.TZ != "UTC" {
		t.Fatalf("state %+v %v", state, err)
	}
	query := retrieval.MemoryQuery{UserID: ownerA, TriggerID: trigger, Model: "synthetic-wp09", Dimension: 2, Limit: 20}
	got, err := r.Memories(ctx, query)
	if err != nil || len(got) != 1 || got[0].ID != memoryA || len(got[0].Vector) != 2 || got[0].EvidenceN != 1 {
		t.Fatalf("owner-scoped memory/evidence/vector result: %+v %v", got, err)
	}
	query.ThreadOnly = true
	query.FocusThread = &thread
	got, err = r.Memories(ctx, query)
	if err != nil || len(got) != 1 {
		t.Fatalf("thread OPENJSON lookup: %+v %v", got, err)
	}
	query.ThreadOnly = false
	query.EntityRole = "mother"
	got, err = r.Memories(ctx, query)
	if err != nil || len(got) != 0 {
		t.Fatalf("incorrect entity match: %+v %v", got, err)
	}
	query.EntityRole = "boss"
	got, err = r.Memories(ctx, query)
	if err != nil || len(got) != 1 {
		t.Fatalf("entity OPENJSON lookup: %+v %v", got, err)
	}
	// Exact history respects the authenticated owner and message order.
	now := time.Now().UTC()
	msgs, err := r.Messages(ctx, retrieval.MessageQuery{UserID: ownerA, TriggerID: trigger, From: now.Add(-time.Hour), To: now.Add(time.Hour), Limit: 12})
	if err != nil || len(msgs) != 1 || msgs[0].ID != msgA {
		t.Fatalf("temporal ownership/order: %+v %v", msgs, err)
	}
	recent, err := r.RecentBefore(ctx, ownerA, trigger, 8)
	if err != nil || len(recent) != 1 || recent[0] != "My boss was supportive" {
		t.Fatalf("recent order: %+v %v", recent, err)
	}
	query.EntityRole = ""
	query.UserID = ownerB // cross-user trigger denied even if a memory exists
	got, err = r.Memories(ctx, query)
	if err != nil || len(got) != 0 {
		t.Fatalf("cross-user lookup leaked: %+v %v", got, err)
	}
	// Visibility metadata changes never delete/overwrite the stored content.
	if _, err = tx.ExecContext(ctx, `UPDATE dbo.ai_memory_items SET deleted_at=SYSUTCDATETIME(),deleted_by=@p1,deletion_reason=N'user_request' WHERE id=@p2 AND user_id=@p1`, ownerA, memoryA); err != nil {
		t.Fatal(err)
	}
	query.UserID = ownerA
	got, err = r.Memories(ctx, query)
	if err != nil || len(got) != 0 {
		t.Fatalf("soft-deleted memory visible: %+v %v", got, err)
	}
	var content string
	if err = tx.QueryRowContext(ctx, `SELECT content FROM dbo.ai_memory_items WHERE id=@p1 AND user_id=@p2`, memoryA, ownerA).Scan(&content); err != nil || content != "my boss was supportive" {
		t.Fatalf("soft delete destroyed source: %s %v", content, err)
	}
	// Rollback is the only cleanup; no physical delete operations are run.
}
