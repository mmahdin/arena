package integration

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	aidb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	rf "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/reflection"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
)

func TestInspectorDerivedJobsAndTransactionIsolationSQL(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f := aiSetup(t, db)
	r := wp12New(t, db)
	dir := filepath.Join(t.TempDir(), "private")
	ins, e := trace.NewInspector(trace.InspectorOptions{Directory: dir, Environment: "local", Address: "localhost:8080"})
	if e != nil {
		t.Fatal(e)
	}
	req := f.req()
	req.Locale = "en"
	req.TZ = "UTC"
	ctx, root := ins.StartTurn(context.Background(), req)
	a, e := f.store.AdmitUserMessage(ctx, req)
	if e != nil {
		t.Fatal(e)
	}
	p := prepared(a)
	p.Jobs = nil
	if _, e = f.store.Complete(ctx, a, p); e != nil {
		t.Fatal(e)
	}
	root.End(nil, nil)
	wp12Enable(t, r, f.a)
	// Real Memory Writer persistence generates a second-generation embedding.
	jctx, job := trace.StartJob(trace.WithInspector(context.Background(), ins), a.TurnID, c.NewID(), "memory", 1)
	meta := mem.Meta{UserID: f.a, TurnID: a.TurnID, OperationID: c.NewID(), Bundle: r.bundle.Reference()}
	report, e := r.service.Apply(mem.WithMeta(jctx, meta), f.a, []c.MemoryOpProposal{wp12Op(a.UserMessageID, "Synthetic derived memory")}, wp12Origin(r))
	if e != nil || len(report.Applied) != 1 {
		t.Fatal(report, e)
	}
	item := report.Applied[0]
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_jobs WHERE user_id=@p1 AND turn_id=@p2 AND kind=N'memory_embed' AND TRY_CONVERT(uniqueidentifier,JSON_VALUE(payload_json,'$.source_id'))=@p3`, f.a, a.TurnID, item); n != 1 {
		t.Fatal("memory descendant lost Turn", n)
	}
	// The explicit Reflection enqueue used to drop the Turn column.
	rr, e := aidb.NewReflectionRepository(r.repo, contextPolicy(t, internalBundle(t)))
	if e != nil {
		t.Fatal(e)
	}
	task := rf.Task{Schema: "reflection_job.v1", UserID: f.a, TurnID: a.TurnID, SessionID: a.SessionID, ThreadID: c.NewID(), Trigger: "session_closed", Bundle: r.bundle.Reference()}
	if e = rr.EnqueueReflection(jctx, task); e != nil {
		t.Fatal(e)
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_jobs WHERE turn_id=@p1 AND kind=N'reflection'`, a.TurnID); n != 1 {
		t.Fatal("Reflection origin missing", n)
	}
	job.End(nil, nil)
	// Rebuilding the exact memory revision/model reuses proven source-job origin.
	aiExec(t, db, `UPDATE dbo.ai_jobs SET status=N'completed' WHERE user_id=@p1 AND kind=N'memory_embed'`, f.a)
	if e = r.repo.BackfillEmbeddings(context.Background(), 128); e != nil {
		t.Fatal(e)
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_jobs WHERE user_id=@p1 AND kind=N'memory_embed' AND dedupe_key LIKE N'backfill:%' AND turn_id IS NULL`, f.a); n != 0 {
		t.Fatal("backfill lost provable origin", n)
	}
	// Reuse ONE physical connection after a committed and an aborted transaction.
	// Even same-user work must not inherit its previous transaction's Turn.
	conn, e := db.Conn(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Close()
	setOrigin := func(tx *sql.Tx, user, turn c.ID) {
		t.Helper()
		_, e := tx.Exec(`DECLARE @v nvarchar(4000)=(SELECT CURRENT_TRANSACTION_ID() transaction_id,CONVERT(varchar(36),@p1) user_id,CONVERT(varchar(36),@p2) turn_id FOR JSON PATH,WITHOUT_ARRAY_WRAPPER); EXEC sys.sp_set_session_context @key=N'theia_trace_origin',@value=@v`, user, turn)
		if e != nil {
			t.Fatal(e)
		}
	}
	for _, abort := range []bool{false, true} {
		tx, e := conn.BeginTx(context.Background(), nil)
		if e != nil {
			t.Fatal(e)
		}
		setOrigin(tx, f.a, a.TurnID)
		if _, e = tx.Exec(`UPDATE dbo.ai_sessions SET summary=N'Synthetic correlated session' WHERE id=@p1`, a.SessionID); e != nil {
			t.Fatal(e)
		}
		if abort {
			e = tx.Rollback()
		} else {
			e = tx.Commit()
		}
		if e != nil {
			t.Fatal(e)
		}
		tx, e = conn.BeginTx(context.Background(), nil)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = tx.Exec(`UPDATE dbo.ai_sessions SET summary=N'Synthetic independent session' WHERE id=@p1`, a.SessionID); e != nil {
			t.Fatal(e)
		}
		if e = tx.Commit(); e != nil {
			t.Fatal(e)
		}
		if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_jobs j JOIN dbo.ai_sessions s ON TRY_CONVERT(uniqueidentifier,JSON_VALUE(j.payload_json,'$.source_id'))=s.id AND JSON_VALUE(j.payload_json,'$.source_version')=CONVERT(varchar(16),CONVERT(binary(8),s.search_version),2) WHERE s.id=@p1 AND j.turn_id IS NULL`, a.SessionID); n != 1 {
			t.Fatal("pooled transaction leaked Turn", abort, n)
		}
	}
	tx, e := conn.BeginTx(context.Background(), nil)
	if e != nil {
		t.Fatal(e)
	}
	// A Turn belonging to another owner cannot be used even with the correct
	// transaction ID. The UDF checks BOTH context ownership and durable ownership.
	setOrigin(tx, f.b, a.TurnID)
	var wrong sql.NullString
	if e = tx.QueryRow(`SELECT CONVERT(varchar(36),dbo.ai_trace_origin_turn(@p1))`, f.b).Scan(&wrong); e != nil {
		t.Fatal(e)
	}
	if wrong.Valid {
		t.Fatal("wrong-owner Turn accepted")
	}
	_ = tx.Rollback()
	// The queued snapshot records work that has not yet executed.
	trace.RefreshJobs(trace.WithInspector(context.Background(), ins), db, a.TurnID)
	docs := inspectorDocuments(t, dir)
	if len(docs) != 1 || docs[0].Complete || docs[0].BackgroundStatus != "pending" {
		t.Fatal("derived queue split or hidden", docs)
	}
}
