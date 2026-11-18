package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	aidb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/eval"
)

type testTraceInspectionPolicy bool

func (p testTraceInspectionPolicy) AllowTraceInspection(context.Context, c.ID, c.ID) (bool, error) {
	return bool(p), nil
}

// This test is opt-in to the existing isolated local SQL Server fixture; it
// never decrypts or exports synthetic message content.
func TestHardeningSQLRuntimeTraceInspectionIsolationAndSoftDelete(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f := aiSetup(t, db)
	turn := wp12Complete(t, f)

	if _, e := aidb.NewRuntimeTraceInspector(db, nil); !errors.Is(e, eval.ErrReplay) {
		t.Fatal("missing access policy allowed")
	}
	denied, e := aidb.NewRuntimeTraceInspector(db, testTraceInspectionPolicy(false))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = denied.Inspect(context.Background(), f.a, turn.TurnID); !errors.Is(e, eval.ErrReplay) {
		t.Fatal("denied policy ignored")
	}
	reader, e := aidb.NewRuntimeTraceInspector(db, testTraceInspectionPolicy(true))
	if e != nil {
		t.Fatal(e)
	}
	coverage, e := reader.Inspect(context.Background(), f.a, turn.TurnID)
	if e != nil || coverage.Replayable || coverage.HasEncryptedPayload || !coverage.HasBundleVersion || coverage.HasModelPromptVersions {
		t.Fatalf("runtime summary falsely replayable: %+v %v", coverage, e)
	}
	if _, e = reader.Inspect(context.Background(), f.b, turn.TurnID); !errors.Is(e, eval.ErrReplay) {
		t.Fatal("cross-owner trace inspection allowed")
	}
	if _, e = reader.Inspect(context.Background(), f.a, c.NewID()); !errors.Is(e, eval.ErrReplay) {
		t.Fatal("absent trace reported as present")
	}
	aiExec(t, db, `UPDATE dbo.conversations SET deleted_at=SYSUTCDATETIME(),deleted_by=@p2,deletion_reason=N'user_request' WHERE id=@p1 AND user_id=@p2`, f.ca, f.a)
	if _, e = reader.Inspect(context.Background(), f.a, turn.TurnID); !errors.Is(e, eval.ErrReplay) {
		t.Fatal("hidden conversation trace exposed")
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_turn_traces WHERE user_id=@p1 AND turn_id=@p2`, f.a, turn.TurnID); n != 1 {
		t.Fatal("soft delete physically removed trace")
	}
}

// Pulse is a real internal Scheduler cycle against SQL Server. It writes a
// durable outbox event but does not start an AI turn or send to any user.
func TestHardeningSQLSchedulerPulseIdempotentInternalDispatch(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f, r, task, _ := wp14Seed(t, db)
	aiExec(t, db, `UPDATE dbo.ai_user_state SET proactive_opt_in=1,quiet_hours_json=NULL,version=version+1 WHERE user_id=@p1`, f.a)
	id := c.NewID()
	op := c.FollowUpOp{Op: "schedule", FollowUpID: id, DueAt: time.Now().UTC().Add(time.Hour), Reason: "check_in", Ref: &task.ThreadID}
	if err := r.timers.Apply(r.ctx(f.a, c.ID{}), []c.FollowUpOp{op}); err != nil {
		t.Fatal(err)
	}
	aiExec(t, db, `UPDATE dbo.ai_follow_ups SET due_at=DATEADD(minute,-1,SYSUTCDATETIME()) WHERE user_id=@p1 AND id=@p2`, f.a, id)
	count, err := r.timers.Pulse(context.Background())
	if err != nil || count != 1 {
		t.Fatalf("internal event not dispatched: %d %v", count, err)
	}
	count, err = r.timers.Pulse(context.Background())
	if err != nil || count != 0 {
		t.Fatalf("duplicate event dispatched: %d %v", count, err)
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_outbox WHERE user_id=@p1 AND type=N'follow_up_due' AND JSON_VALUE(payload_json,'$.schema')=N'wakeup.v1'`, f.a); n != 1 {
		t.Fatalf("outbox idempotency failed: %d", n)
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.turns WHERE user_id=@p1 AND lifecycle_version=1`, f.a); n != 1 {
		t.Fatal("internal timer forged an additional user-authored turn")
	}
	if h := r.timers.Health(); h.Cycles != 2 || h.EventsDispatched != 1 || h.Failures != 0 {
		t.Fatalf("wrong safe scheduler health counters: %+v", h)
	}
}

func TestHardeningSQLSchedulerExpiredReferenceCannotWake(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f, r, task, _ := wp14Seed(t, db)
	aiExec(t, db, `UPDATE dbo.ai_user_state SET proactive_opt_in=1,quiet_hours_json=NULL,version=version+1 WHERE user_id=@p1`, f.a)
	id := c.NewID()
	at := time.Now().UTC().Add(time.Hour)
	op := c.FollowUpOp{Op: "schedule", FollowUpID: id, DueAt: at, Reason: "goal_review", Ref: &task.ThreadID}
	if err := r.timers.Apply(r.ctx(f.a, c.ID{}), []c.FollowUpOp{op}); err != nil {
		t.Fatal(err)
	}
	aiExec(t, db, `UPDATE dbo.ai_threads SET phase=N'resolved' WHERE id=@p1 AND user_id=@p2`, task.ThreadID, f.a)
	got, err := r.timers.Due(context.Background(), at.Add(time.Second), 10)
	if err != nil || len(got) != 0 {
		t.Fatal("invalid followup reference emitted event", err)
	}
	var status, reason string
	if err := db.QueryRow(`SELECT status,skip_reason FROM dbo.ai_follow_ups WHERE id=@p1 AND user_id=@p2`, id, f.a).Scan(&status, &reason); err != nil || status != "skipped" || reason != "reference_ineligible" {
		t.Fatalf("invalid reference not durably skipped: %s %s %v", status, reason, err)
	}
	if n := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_outbox WHERE user_id=@p1 AND type=N'follow_up_due'`, f.a); n != 0 {
		t.Fatal("invalid reference produced an internal wakeup")
	}
}
