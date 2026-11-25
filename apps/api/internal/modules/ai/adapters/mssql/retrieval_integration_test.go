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
	r, err := NewRetrievalRepositoryTx(tx, sqlFixtureUse{}, sqlFixtureCipher{})
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
	if _, err = tx.ExecContext(ctx, `INSERT INTO dbo.ai_embeddings(id,user_id,ref_kind,ref_id,codec_version,model_id,dim,vector_bytes,source_version)
        SELECT(@p1,@p2,N'memory',@p3,1,N'synthetic-wp09',2,@p4,(SELECT CONVERT(binary(8),search_version) FROM dbo.ai_memory_items WHERE id=@p3)`, embeddingA, ownerA, memoryA, vector); err != nil {
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
	query := retrieval.Query{UserID: ownerA, TriggerID: trigger, Model: "synthetic-wp09", Dimension: 2}
	scan := func(q retrieval.Query) []retrieval.Candidate {
		t.Helper()
		var out []retrieval.Candidate
		if e := r.Scan(ctx, q, func(v retrieval.Candidate) error { out = append(out, v); return nil }); e != nil {
			t.Fatal(e)
		}
		return out
	}
	got := scan(query)
	if len(got) != 2 {
		t.Fatalf("expected memory and raw history: %+v", got)
	}
	found := false
	for _, item := range got {
		if item.ID == memoryA {
			found = len(item.Vector) == 2 && item.EvidenceN == 1
		}
	}
	if !found {
		t.Fatal("memory embedding/provenance absent", got)
	}
	query.UserID = ownerB
	if got = scan(query); len(got) != 0 {
		t.Fatal("cross-user trigger leaked", got)
	}
	query.UserID = ownerA
	// Visibility metadata changes never delete/overwrite the stored content.
	if _, err = tx.ExecContext(ctx, `UPDATE dbo.ai_memory_items SET deleted_at=SYSUTCDATETIME(),deleted_by=@p1,deletion_reason=N'user_request' WHERE id=@p2 AND user_id=@p1`, ownerA, memoryA); err != nil {
		t.Fatal(err)
	}
	query.UserID = ownerA
	got = scan(query)
	for _, item := range got {
		if item.ID == memoryA {
			t.Fatal("hidden memory visible")
		}
	}
	var content string
	if err = tx.QueryRowContext(ctx, `SELECT content FROM dbo.ai_memory_items WHERE id=@p1 AND user_id=@p2`, memoryA, ownerA).Scan(&content); err != nil || content != "my boss was supportive" {
		t.Fatalf("soft delete destroyed source: %s %v", content, err)
	}
	// Rollback is the only cleanup; no physical delete operations are run.
}

type sqlFixtureUse struct{}

func (sqlFixtureUse) CanSearch(context.Context, c.ID, c.ID) (bool, error) { return true, nil }
func (sqlFixtureUse) CanUse(context.Context, c.ID, retrieval.SourceKind, c.ID) (bool, error) {
	return true, nil
}

type sqlFixtureCipher struct{}

func (sqlFixtureCipher) Seal(_ context.Context, _ c.ID, _ string, b []byte) ([]byte, error) {
	return b, nil
}
func (sqlFixtureCipher) Open(_ context.Context, _ c.ID, _ string, b []byte) ([]byte, error) {
	return b, nil
}
