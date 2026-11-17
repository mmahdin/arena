package integration

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	saveddb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/bookmark/adapters/mssql"
	convdb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/conversation/adapters/mssql"
	lettersdb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/letters/adapters/mssql"
	letters "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/letters/application"
	diarydb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/reflection/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/migrations"
)

// Snapshot actual stored columns, not a checksum: deletion must preserve the
// complete payloads, references, consent flags and analyses byte for byte.
func storedJSON(t *testing.T, db *sql.DB, query string, args ...any) string {
	t.Helper()
	var value string
	if err := db.QueryRowContext(context.Background(), query+" FOR JSON PATH, INCLUDE_NULL_VALUES", args...).Scan(&value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestSoftDeleteSQLServer(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	ctx := context.Background()
	f := aiSetup(t, db)
	r := f.req()
	a := admit(t, f, r)
	p := prepared(a)
	p.Jobs = []c.AsyncJobProposal{{Kind: "synthetic", DedupeKey: "preserve", Payload: []byte(`{"fixture":true}`)}}
	if _, err := f.store.Complete(ctx, a, p); err != nil {
		t.Fatal(err)
	}
	thread, item, formulation := c.NewID(), c.NewID(), c.NewID()
	aiExec(t, db, `INSERT dbo.ai_threads(id,user_id,topic) VALUES(@p1,@p2,N'synthetic topic');
 INSERT dbo.ai_formulations(id,user_id,thread_id,version,framework,fields_json,paragraph) VALUES(@p3,@p2,@p1,1,N'synthetic',N'{}',N'synthetic analysis');
 INSERT dbo.ai_memory_items(id,user_id,kind,content,epistemic,source_json) VALUES(@p4,@p2,N'fact',N'synthetic memory',N'reported',N'{}');
 INSERT dbo.ai_memory_evidence(user_id,item_id,ref_kind,ref_id,role,evidence_at) VALUES(@p2,@p4,N'message',@p5,N'supports',SYSUTCDATETIME());
 INSERT dbo.ai_embeddings(id,user_id,ref_kind,ref_id,model_id,dim,vector_bytes) VALUES(NEWID(),@p2,N'memory',@p4,N'synthetic',1,0x0000803F)`, thread, f.a, formulation, item, a.UserMessageID)
	aiExec(t, db, `INSERT dbo.ai_protocol_instances(id,user_id,thread_id,protocol_id,protocol_version,status) VALUES(NEWID(),@p1,@p2,N'synthetic',N'1',N'offered');
 INSERT dbo.ai_open_loops(id,user_id,kind,text,ref_id) VALUES(NEWID(),@p1,N'confirm_fact',N'Synthetic question',@p3);
 INSERT dbo.ai_follow_ups(id,user_id,reason,due_at,ref_id,status,skip_reason) VALUES(NEWID(),@p1,N'synthetic',SYSUTCDATETIME(),@p2,N'skipped',N'not_opted_in');
 INSERT dbo.ai_memory_tombstones(id,user_id,reason) VALUES(NEWID(),@p1,N'explicit_prior_control');
 INSERT dbo.ai_hypothesis_cooldowns(user_id,tag,thread_id,until_at,rejected_item_id) VALUES(@p1,N'synthetic',@p2,DATEADD(day,1,SYSUTCDATETIME()),@p3)`, f.a, thread, item)
	bookmark := saveddb.New(db)
	if err := bookmark.Save(ctx, f.a, a.UserMessageID); err != nil {
		t.Fatal(err)
	}
	letterRepo := lettersdb.New(db)
	letter, err := letterRepo.Save(ctx, f.a, f.ca, letters.Draft{Title: "Synthetic letter", Paragraphs: []string{"Synthetic analysis", "Synthetic second paragraph"}})
	if err != nil {
		t.Fatal(err)
	}
	tables := []string{"turns", "saved_messages", "personalized_letters", "ai_sessions", "ai_user_state", "ai_threads", "ai_formulations", "ai_memory_items", "ai_memory_evidence", "ai_embeddings", "ai_turn_context", "ai_turn_events", "ai_jobs", "ai_outbox", "ai_turn_traces", "ai_protocol_instances", "ai_open_loops", "ai_follow_ups", "ai_memory_tombstones", "ai_hypothesis_cooldowns"}
	before := map[string]string{}
	for _, table := range tables {
		before[table] = storedJSON(t, db, "SELECT * FROM dbo."+table+" WHERE user_id=@p1", f.a)
	}
	raw := storedJSON(t, db, `SELECT * FROM dbo.messages WHERE conversation_id=@p1 ORDER BY message_order`, f.ca)
	repo := convdb.New(db)
	if err := repo.Delete(ctx, f.b, f.ca); !errors.Is(err, httpx.ErrNotFound) {
		t.Fatal("wrong-owner deletion", err)
	}
	if err := repo.Delete(ctx, f.a, f.ca); err != nil {
		t.Fatal(err)
	}
	metadata := storedJSON(t, db, `SELECT deleted_at,deleted_by,deletion_reason FROM dbo.conversations WHERE id=@p1`, f.ca)
	if err := repo.Delete(ctx, f.a, f.ca); err != nil {
		t.Fatal(err)
	}
	if metadata != storedJSON(t, db, `SELECT deleted_at,deleted_by,deletion_reason FROM dbo.conversations WHERE id=@p1`, f.ca) {
		t.Fatal("repeat delete changed metadata")
	}
	for _, table := range tables {
		if got := storedJSON(t, db, "SELECT * FROM dbo."+table+" WHERE user_id=@p1", f.a); got != before[table] {
			t.Fatal("retained rows changed", table)
		}
	}
	if raw != storedJSON(t, db, `SELECT * FROM dbo.messages WHERE conversation_id=@p1 ORDER BY message_order`, f.ca) {
		t.Fatal("original messages changed")
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.conversations WHERE id=@p1 AND deleted_at IS NOT NULL AND deleted_by=@p2 AND deletion_reason=N'user_request'`, f.ca, f.a) != 1 {
		t.Fatal("missing deletion metadata")
	}
	list, err := repo.List(ctx, f.a)
	if err != nil || len(list) != 1 || list[0].ID != f.ca2 {
		t.Fatal("hidden conversation leaked", err)
	}
	if _, err := repo.Messages(ctx, f.a, f.ca, 20); !errors.Is(err, httpx.ErrNotFound) {
		t.Fatal("hidden history leaked", err)
	}
	if _, err := repo.Rename(ctx, f.a, f.ca, "changed"); !errors.Is(err, httpx.ErrNotFound) {
		t.Fatal("hidden conversation renamed", err)
	}
	if _, err := f.store.Recent(ctx, f.a, f.ca, 20); !errors.Is(err, httpx.ErrNotFound) {
		t.Fatal("hidden recent history leaked", err)
	}
	if _, err := f.store.GetTurn(ctx, f.a, r.ClientTurnID); !errors.Is(err, httpx.ErrNotFound) {
		t.Fatal("hidden result replayed", err)
	}
	if _, err := f.store.AdmitUserMessage(ctx, r); !errors.Is(err, httpx.ErrNotFound) {
		t.Fatal("hidden admission accepted", err)
	}
	if _, err := f.store.Complete(ctx, a, p); !errors.Is(err, httpx.ErrNotFound) {
		t.Fatal("hidden final result replayed", err)
	}
	if _, err := repo.ClaimTurn(ctx, f.a, f.ca, c.NewID(), "synthetic"); !errors.Is(err, httpx.ErrNotFound) {
		t.Fatal("prototype admitted hidden conversation", err)
	}
	bookmarks, err := bookmark.List(ctx, f.a)
	if err != nil || len(bookmarks) != 0 {
		t.Fatal("hidden message bookmarked", err)
	}
	// A derived letter has its own visibility; hiding a source does not delete it.
	visible, err := letterRepo.List(ctx, f.a)
	if err != nil || len(visible) != 1 || visible[0].ID != letter.ID {
		t.Fatal("derived letter was automatically hidden", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE dbo.conversations SET deleted_at=SYSUTCDATETIME(),deleted_by=@p2 WHERE id=@p1`, f.ca2, f.a); err == nil {
		t.Fatal("incomplete deletion metadata accepted")
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM sys.foreign_keys WHERE delete_referential_action<>0`) != 0 {
		t.Fatal("delete cascade remains")
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM sys.procedures WHERE name=N'ai_purge_user_derivatives'`) != 0 {
		t.Fatal("purge procedure remains")
	}
	for _, query := range []string{`DELETE dbo.messages WHERE conversation_id=@p1`, `DELETE dbo.conversations WHERE id=@p1`, `DELETE dbo.ai_turn_traces WHERE turn_id=@p2`, `DELETE dbo.ai_jobs WHERE turn_id=@p2`, `DELETE dbo.ai_memory_items WHERE id=@p3`, `DELETE dbo.ai_formulations WHERE id=@p4`} {
		if _, err := db.ExecContext(ctx, query, f.ca, a.TurnID, item, formulation); err == nil {
			t.Fatal("physical deletion succeeded")
		}
	}
	for _, table := range tables {
		if got := storedJSON(t, db, "SELECT * FROM dbo."+table+" WHERE user_id=@p1", f.a); got != before[table] {
			t.Fatal("physical delete attempt changed data", table)
		}
	}
}

func TestSoftDeleteRecreationSQLServer(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	ctx := context.Background()
	f := aiSetup(t, db)
	a := admit(t, f, f.req())
	if _, err := f.store.Complete(ctx, a, prepared(a)); err != nil {
		t.Fatal(err)
	}
	diary := diarydb.New(db)
	saved := saveddb.New(db)
	letterRepo := lettersdb.New(db)
	if _, err := diary.Save(ctx, f.a, "2026-11-16", "Synthetic original diary"); err != nil {
		t.Fatal(err)
	}
	if err := saved.Save(ctx, f.a, a.UserMessageID); err != nil {
		t.Fatal(err)
	}
	original, err := letterRepo.Save(ctx, f.a, f.ca, letters.Draft{Title: "Original", Paragraphs: []string{"Synthetic original", "Synthetic second paragraph"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := letterRepo.Delete(ctx, f.b, original.ID); !errors.Is(err, httpx.ErrNotFound) {
		t.Fatal("foreign letter deletion", err)
	}
	for i := 0; i < 2; i++ {
		if err := diary.Delete(ctx, f.a, "2026-11-16"); err != nil {
			t.Fatal(err)
		}
		if err := saved.Remove(ctx, f.a, a.UserMessageID); err != nil {
			t.Fatal(err)
		}
		if err := letterRepo.Delete(ctx, f.a, original.ID); err != nil {
			t.Fatal(err)
		}
	}
	if rows, err := letterRepo.List(ctx, f.a); err != nil || len(rows) != 0 {
		t.Fatal("deleted letter listed", err)
	}
	if _, err := letterRepo.BySource(ctx, f.a, f.ca); !errors.Is(err, httpx.ErrNotFound) {
		t.Fatal("deleted letter replayed", err)
	}
	if rows, err := diary.List(ctx, f.a); err != nil || len(rows) != 0 {
		t.Fatal("deleted diary listed", err)
	}
	if rows, err := saved.List(ctx, f.a); err != nil || len(rows) != 0 {
		t.Fatal("deleted bookmark listed", err)
	}
	for _, table := range []string{"daily_reflections", "saved_messages", "personalized_letters"} {
		if aiCount(t, db, "SELECT COUNT(*) FROM dbo."+table+" WHERE user_id=@p1 AND deleted_by=@p1 AND deleted_at IS NOT NULL AND deletion_reason=N'user_request'", f.a) != 1 {
			t.Fatal("missing preserved record", table)
		}
	}
	if _, err := diary.Save(ctx, f.a, "2026-11-16", "Synthetic replacement diary"); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := saved.Save(ctx, f.a, a.UserMessageID); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	replacement, err := letterRepo.Save(ctx, f.a, f.ca, letters.Draft{Title: "Replacement", Paragraphs: []string{"Synthetic replacement", "Synthetic second paragraph"}})
	if err != nil || replacement.ID == original.ID {
		t.Fatal("letter replacement", err)
	}
	for _, table := range []string{"daily_reflections", "saved_messages", "personalized_letters"} {
		if aiCount(t, db, "SELECT COUNT(*) FROM dbo."+table+" WHERE user_id=@p1", f.a) != 2 {
			t.Fatal("recreation overwrote archived record", table)
		}
		if aiCount(t, db, "SELECT COUNT(*) FROM dbo."+table+" WHERE user_id=@p1 AND deleted_at IS NULL", f.a) != 1 {
			t.Fatal("active uniqueness", table)
		}
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.daily_reflections WHERE user_id=@p1 AND deleted_at IS NOT NULL AND body=N'Synthetic original diary'`, f.a) != 1 {
		t.Fatal("original diary text lost")
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.personalized_letters WHERE id=@p1 AND deleted_at IS NOT NULL AND title=N'Original'`, original.ID) != 1 {
		t.Fatal("original letter lost")
	}
}

func TestSoftDeleteLifecycleSQLServer(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	ctx := context.Background()
	f := aiSetup(t, db)
	repo := convdb.New(db)
	req := f.req()
	running := admit(t, f, req)
	queuedReq := f.req()
	queued := admit(t, f, queuedReq)
	otherReq := f.req()
	otherReq.ConversationID = f.ca2
	other := admit(t, f, otherReq)
	if !queued.Queued || !other.Queued {
		t.Fatal("expected queued requests")
	}
	// Inject a failure after the visibility update; its transaction must also
	// preserve the live lease. No update trigger remains after this test step.
	aiExec(t, db, `CREATE TRIGGER dbo.synthetic_delete_failure ON dbo.conversations AFTER UPDATE AS BEGIN IF EXISTS(SELECT 1 FROM inserted WHERE deleted_at IS NOT NULL) THROW 51021,'synthetic failure',1; END`)
	if err := repo.Delete(ctx, f.a, f.ca); err == nil {
		t.Fatal("expected transaction failure")
	}
	aiExec(t, db, `DROP TRIGGER dbo.synthetic_delete_failure`)
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.conversations WHERE id=@p1 AND deleted_at IS NULL`, f.ca) != 1 {
		t.Fatal("failed delete hid conversation")
	}
	if _, err := f.store.Renew(ctx, ports.LeaseHandle{UserID: f.a, TurnID: running.TurnID, Token: running.LeaseToken}); err != nil {
		t.Fatal("failed delete expired lease", err)
	}
	if err := repo.Delete(ctx, f.a, f.ca); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.Complete(ctx, running, prepared(running)); err == nil {
		t.Fatal("stale worker completed hidden turn")
	}
	if _, err := f.store.Renew(ctx, ports.LeaseHandle{UserID: f.a, TurnID: running.TurnID, Token: running.LeaseToken}); err == nil {
		t.Fatal("hidden turn renewed")
	}
	if err := f.store.Retry(ctx, f.a, req.ClientTurnID); err == nil {
		t.Fatal("hidden turn retried")
	}
	if _, err := f.store.Acquire(ctx, f.a, queued.TurnID); err == nil {
		t.Fatal("hidden queued turn acquired")
	}
	next, err := f.store.Next(ctx, f.a)
	if err != nil || next.ClientTurnID != otherReq.ClientTurnID {
		t.Fatal("hidden queue blocks other conversation", err)
	}
	claimed := admit(t, f, next)
	if claimed.Queued {
		t.Fatal("hidden conversation holds user lease")
	}
	if _, err := f.store.Complete(ctx, claimed, prepared(claimed)); err != nil {
		t.Fatal(err)
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.messages WHERE conversation_id=@p1 AND role=N'user'`, f.ca) != 2 {
		t.Fatal("lost pending user messages")
	}
	// Prototype error recovery also retains its turn; completion after deletion fails.
	legacy, err := repo.Create(ctx, f.a)
	if err != nil {
		t.Fatal(err)
	}
	id := c.NewID()
	claim, err := repo.ClaimTurn(ctx, f.a, legacy.ID, id, "Synthetic legacy")
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.ReleaseTurn(ctx, f.a, legacy.ID, id, claim.LeaseID); err != nil {
		t.Fatal(err)
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.turns WHERE id=@p1`, id) != 1 {
		t.Fatal("release discarded legacy turn")
	}
	claim, err = repo.ClaimTurn(ctx, f.a, legacy.ID, id, "Synthetic legacy")
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(ctx, f.a, legacy.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CompleteTurn(ctx, f.a, legacy.ID, id, claim.LeaseID, "Synthetic legacy", "Synthetic reply", "test"); !errors.Is(err, httpx.ErrNotFound) {
		t.Fatal("legacy worker published after deletion", err)
	}
}

func TestSoftDeleteUpgradeSQLServer(t *testing.T) {
	db := aiDatabase(t)
	ctx := context.Background()
	previous := t.TempDir()
	files, err := os.ReadDir("../../db/migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if file.Name() >= "0020" || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join("../../db/migrations", file.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(previous, file.Name()), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := migrations.Apply(ctx, db, previous); err != nil {
		t.Fatal(err)
	}
	f := aiSetup(t, db)
	// Use 0019 SQL directly: current adapters require 0020's metadata.
	aiExec(t, db, `INSERT dbo.daily_reflections(user_id,entry_date,body) VALUES(@p1,'2026-11-16',N'Synthetic original');
 INSERT dbo.personalized_letters(id,user_id,source_conversation_id,title,paragraphs_json) VALUES(NEWID(),@p1,@p2,N'Synthetic letter',N'["Synthetic text"]')`, f.a, f.ca)
	before := storedJSON(t, db, `SELECT user_id,entry_date,body,created_at,updated_at FROM dbo.daily_reflections WHERE user_id=@p1`, f.a)
	aiMigrate(t, db)
	aiMigrate(t, db)
	if before != storedJSON(t, db, `SELECT user_id,entry_date,body,created_at,updated_at FROM dbo.daily_reflections WHERE user_id=@p1`, f.a) {
		t.Fatal("upgrade changed diary content")
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.daily_reflections WHERE user_id=@p1 AND id IS NOT NULL AND deleted_at IS NULL`, f.a) != 1 {
		t.Fatal("upgrade backfill")
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM sys.foreign_keys WHERE is_disabled=1 OR is_not_trusted=1 OR delete_referential_action<>0`) != 0 {
		t.Fatal("unvalidated or cascading foreign key")
	}
	// This catches dynamically named constraints, and the historical trigger/proc.
	if aiCount(t, db, `SELECT COUNT(*) FROM sys.tables t WHERE schema_id=SCHEMA_ID(N'dbo') AND name<>N'schema_migrations' AND NOT EXISTS(SELECT 1 FROM sys.triggers tr WHERE tr.parent_id=t.object_id AND tr.name=t.name+N'_preserve_delete_guard')`) != 0 {
		t.Fatal("unguarded data table")
	}
	for _, query := range []string{`DELETE dbo.daily_reflections WHERE user_id=@p1`, `DELETE dbo.personalized_letters WHERE user_id=@p1`, `DELETE dbo.users WHERE id=@p1`} {
		if _, err := db.ExecContext(ctx, query, f.a); err == nil {
			t.Fatal("physical deletion permitted")
		}
	}
	// Ensure the immutable migration ledger still includes all previous files.
	migrationCount := 0
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".sql") {
			migrationCount++
		}
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.schema_migrations WHERE name LIKE N'00%'`) != migrationCount {
		t.Fatal("migration ledger changed")
	}
}

// Both possible lock orderings must yield a preserved completed turn or a
// preserved user-only turn; no worker may write after deletion returns.
func TestSoftDeleteConcurrentFinalizationSQLServer(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	ctx := context.Background()
	for i := 0; i < 8; i++ {
		f := aiSetup(t, db)
		a := admit(t, f, f.req())
		start := make(chan struct{})
		completed := make(chan error, 1)
		deleted := make(chan error, 1)
		go func() { <-start; _, err := f.store.Complete(ctx, a, prepared(a)); completed <- err }()
		go func() { <-start; deleted <- convdb.New(db).Delete(ctx, f.a, f.ca) }()
		close(start)
		completionErr := <-completed
		if err := <-deleted; err != nil {
			t.Fatal("concurrent soft deletion failed", err)
		}
		expected := 1
		if completionErr == nil {
			expected = 2
		} else if !errors.Is(completionErr, httpx.ErrNotFound) {
			t.Fatal("unexpected finalization failure", completionErr)
		}
		if aiCount(t, db, `SELECT COUNT(*) FROM dbo.messages WHERE conversation_id=@p1`, f.ca) != expected {
			t.Fatal("lost raw messages")
		}
		before := storedJSON(t, db, `SELECT * FROM dbo.messages WHERE conversation_id=@p1 ORDER BY message_order`, f.ca)
		if _, err := f.store.Complete(ctx, a, prepared(a)); !errors.Is(err, httpx.ErrNotFound) {
			t.Fatal("late publication after soft deletion", err)
		}
		if before != storedJSON(t, db, `SELECT * FROM dbo.messages WHERE conversation_id=@p1 ORDER BY message_order`, f.ca) {
			t.Fatal("late write changed hidden history")
		}
	}
}
