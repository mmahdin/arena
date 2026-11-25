package integration

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	aidb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/migrations"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"
)

func TestUnifiedDecisionUpgradeAtomicReplayAndOwnerIsolationSQL(t *testing.T) {
	db := aiDatabase(t)
	ctx := context.Background()
	old := t.TempDir()
	files, e := filepath.Glob("../../db/migrations/*.sql")
	if e != nil {
		t.Fatal(e)
	}
	for _, path := range files {
		if filepath.Base(path) >= "0041" {
			continue
		}
		raw, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(old, filepath.Base(path)), raw, 0600); e != nil {
			t.Fatal(e)
		}
	}
	if e = migrations.Apply(ctx, db, old); e != nil {
		t.Fatal(e)
	}
	f := aiSetup(t, db)
	prior := admit(t, f, f.req())
	if _, e = f.store.Complete(ctx, prior, prepared(prior)); e != nil {
		t.Fatal(e)
	}
	aiMigrate(t, db)
	aiMigrate(t, db)
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.messages WHERE turn_id=@p1`, prior.TurnID) != 2 {
		t.Fatal("upgrade lost prior history")
	}
	a := admit(t, f, f.req())
	p := prepared(a)
	p.Decisions = &c.TurnDecisions{Carry: []c.FutureHint{{Topic: "synthetic continuation", ResourceIDs: []string{"skill:reflect_emotion"}, RemainingTurns: 2, ExpiresAt: time.Now().Add(time.Hour)}}, SelectedKnowledge: []string{"skill:reflect_emotion"}, Accepted: []string{"summary"}, Rejected: []c.DeltaRejection{{Path: "background_requests.1", RuleCode: "BG-001", Detail: "synthetic rejection"}}}
	f.state.fail = true
	if _, e = f.store.Complete(ctx, a, p); e == nil {
		t.Fatal("injected late rollback succeeded")
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_turn_decisions WHERE turn_id=@p1`, a.TurnID) != 0 || aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_jobs WHERE turn_id=@p1`, a.TurnID) != 0 {
		t.Fatal("decisions/jobs escaped rollback")
	}
	result, e := f.store.Complete(ctx, a, p)
	if e != nil {
		t.Fatal(e)
	}
	replayed, e := f.store.Complete(ctx, a, p)
	if e != nil || replayed.AssistantMessageID == nil || *replayed.AssistantMessageID != *result.AssistantMessageID {
		t.Fatal("decision replay duplicated finalization", e)
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_turn_decisions WHERE turn_id=@p1`, a.TurnID) != 1 {
		t.Fatal("decision receipt is not idempotent")
	}
	r, _ := aidb.NewContextReader(db)
	owned, e := r.Read(ctx, f.a, f.ca)
	if e != nil || len(owned.FutureHints) != 1 {
		t.Fatal("carry-over was not restored", e)
	}
	if _, e = r.Read(ctx, f.b, f.ca); e == nil {
		t.Fatal("foreign owner read carry-over")
	}
	if _, e = db.Exec(`UPDATE dbo.ai_turn_decisions SET carry_json=N'[]' WHERE turn_id=@p1`, a.TurnID); e == nil {
		t.Fatal("immutable decisions were overwritten")
	}
	if _, e = db.Exec(`DELETE FROM dbo.ai_turn_decisions WHERE turn_id=@p1`, a.TurnID); e == nil {
		t.Fatal("retained decisions were deleted")
	}
	other := admit(t, f, f.req())
	if _, e = db.Exec(`INSERT dbo.ai_turn_decisions(turn_id,user_id,carry_json,resolution_json) VALUES(@p1,@p2,N'[]',N'{}')`, other.TurnID, f.b); e == nil {
		t.Fatal("cross-owner decision inserted")
	}
	if e = f.store.Cancel(ctx, other); e != nil {
		t.Fatal(e)
	}
}

func TestUnifiedBackgroundResearchCannotCommitAfterSourceRevisionSQL(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f := aiSetup(t, db)
	a := admit(t, f, f.req())
	if _, e := f.store.Complete(context.Background(), a, prepared(a)); e != nil {
		t.Fatal(e)
	}
	runtime := wp12New(t, db)
	ctx := context.Background()
	aiExec(t, db, `UPDATE dbo.ai_user_state SET locale=N'en',consent_json=N'{"synthetic_memory":true}' WHERE user_id=@p1`, f.a)
	var version string
	if e := db.QueryRow(`SELECT CONVERT(varchar(16),CONVERT(binary(8),search_version),2) FROM dbo.messages WHERE id=@p1`, a.UserMessageID).Scan(&version); e != nil {
		t.Fatal(e)
	}
	m := mem.Meta{UserID: f.a, TurnID: a.TurnID, OperationID: c.NewID(), Bundle: runtime.bundle.Reference(), ResearchVersions: []c.MemoryView{{ID: a.UserMessageID, Source: "message", SourceVersion: version}}}
	aiExec(t, db, `UPDATE dbo.messages SET content=content WHERE id=@p1`, a.UserMessageID)
	snapshot, e := runtime.repo.Snapshot(ctx, f.a, m, nil)
	if e != nil {
		t.Fatal("source-fence snapshot", e)
	}
	e = runtime.repo.Save(ctx, m, snapshot, mem.Plan{}, "source-fence")
	if !errors.Is(e, mem.ErrConflict) {
		t.Fatal("stale research was not fenced", e, observability.ErrorFields(e), "version", version, "denied", errors.Is(e, mem.ErrDenied))
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_batches WHERE user_id=@p1 AND operation_id=@p2`, f.a, m.OperationID) != 0 {
		t.Fatal("stale research changed durable memory")
	}
}
