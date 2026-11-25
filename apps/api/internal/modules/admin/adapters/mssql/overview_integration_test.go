package mssql

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/microsoft/go-mssqldb"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
)

// Run ONLY on an explicitly acknowledged, disposable SQL Server database with
// migrations through 0028 applied. Never use a shared Stage or Production DB.
func TestBO05OverviewDisposableSQL(t *testing.T) {
	dsn := os.Getenv("THEIA_DISPOSABLE_OVERVIEW_SQL_DSN")
	if dsn == "" || os.Getenv("THEIA_DISPOSABLE_OVERVIEW_SQL_ACK") != "YES_DISPOSABLE_ONLY" {
		t.Skip("requires disposable SQL Server, explicit acknowledgement, migrations through 0028")
	}
	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	at := time.Now().UTC().Truncate(time.Millisecond)
	from := at.Add(-2 * time.Minute)
	recorded := at.Add(-time.Minute)
	userID, conversationID := uuid.New(), uuid.New()
	completedID, failedID := uuid.New(), uuid.New()
	attemptID, unfinishedID := uuid.New(), uuid.New()
	stageID, workerID, jobID := uuid.New(), uuid.New(), uuid.New()
	// Cleanup is a courtesy on disposable DBs, not a substitute for using
	// one. Keep the test opt-in even though every inserted ID is random.
	defer func() {
		ctx := context.Background()
		db.ExecContext(ctx, `DELETE FROM dbo.ai_worker_heartbeats WHERE instance_id=CONVERT(uniqueidentifier,@p1)`, workerID.String())
		db.ExecContext(ctx, `DELETE FROM dbo.ai_stage_events WHERE id=CONVERT(uniqueidentifier,@p1)`, stageID.String())
		db.ExecContext(ctx, `DELETE FROM dbo.ai_llm_attempts WHERE id IN (CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2))`, attemptID.String(), unfinishedID.String())
		db.ExecContext(ctx, `DELETE FROM dbo.ai_jobs WHERE id=CONVERT(uniqueidentifier,@p1)`, jobID.String())
		db.ExecContext(ctx, `DELETE FROM dbo.turns WHERE id IN (CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2))`, completedID.String(), failedID.String())
		db.ExecContext(ctx, `DELETE FROM dbo.conversations WHERE id=CONVERT(uniqueidentifier,@p1)`, conversationID.String())
		db.ExecContext(ctx, `DELETE FROM dbo.users WHERE id=CONVERT(uniqueidentifier,@p1)`, userID.String())
	}()
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT dbo.users(id,test_key,is_test) VALUES(CONVERT(uniqueidentifier,@p1),@p2,1)`, userID.String(), "bo05-"+userID.String())
	exec(`INSERT dbo.conversations(id,user_id,title,created_at) VALUES(CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),N'Test',@p3)`, conversationID.String(), userID.String(), recorded)
	exec(`INSERT dbo.turns(id,conversation_id,user_id,input_hash,status,lifecycle_version,user_message_id,assistant_message_id,completed_at,created_at)
        VALUES(CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),CONVERT(uniqueidentifier,@p3),@p4,N'completed',1,
        CONVERT(uniqueidentifier,@p5),CONVERT(uniqueidentifier,@p6),@p7,@p8)`,
		completedID.String(), conversationID.String(), userID.String(), make([]byte, 32), uuid.NewString(), uuid.NewString(), recorded.Add(1500*time.Millisecond), recorded)
	exec(`INSERT dbo.turns(id,conversation_id,user_id,input_hash,status,lifecycle_version,user_message_id,created_at)
        VALUES(CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),CONVERT(uniqueidentifier,@p3),@p4,N'failed',1,CONVERT(uniqueidentifier,@p5),@p6)`,
		failedID.String(), conversationID.String(), userID.String(), make([]byte, 32), uuid.NewString(), recorded)
	exec(`INSERT dbo.ai_llm_attempts(id,turn_id,kind,tier,provider,model,started_at,finished_at,status,duration_ms,
        input_tokens,output_tokens,cached_tokens,input_rate_micro_usd_per_m,output_rate_micro_usd_per_m,cached_rate_micro_usd_per_m,cost_micro_usd,cost_status)
        VALUES(CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),'structured','balanced','test-provider','test-model',@p3,@p4,'succeeded',100,
        20,10,0,1000000,1000000,0,30,'estimated')`, attemptID.String(), completedID.String(), recorded, recorded.Add(time.Second))
	exec(`INSERT dbo.ai_llm_attempts(id,kind,tier,provider,model,started_at,
        input_rate_micro_usd_per_m,output_rate_micro_usd_per_m,cached_rate_micro_usd_per_m)
        VALUES(CONVERT(uniqueidentifier,@p1),'embed','balanced','test-provider','test-model',@p2,0,0,0)`, unfinishedID.String(), recorded)
	exec(`INSERT dbo.ai_stage_events(id,turn_id,stage,status,duration_ms,recorded_at)
        VALUES(CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),'brain','success',42,@p3)`, stageID.String(), completedID.String(), recorded)
	exec(`INSERT dbo.ai_worker_heartbeats(instance_id,worker_kind,status,last_seen_at)
        VALUES(CONVERT(uniqueidentifier,@p1),'memory','running',@p2)`, workerID.String(), at.Add(-time.Second))
	exec(`INSERT dbo.ai_jobs(id,user_id,kind,status,run_after,created_at)
        VALUES(CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),N'memory',N'queued',@p3,@p4)`, jobID.String(), recorded, recorded)
	overview, err := NewOverviewRepository(db).ReadOverview(ctx, domain.OverviewFilter{From: from, To: at, AsOf: at})
	if err != nil {
		t.Fatal(err)
	}
	if overview.Turns.Completed < 1 || overview.Turns.Failed < 1 || overview.Turns.LatencySamples < 1 ||
		overview.Turns.P95ElapsedMs == nil || overview.LLM.Attempts < 2 || overview.LLM.EstimatedMicroUSD == nil ||
		overview.LLM.UnknownCostCount < 1 || overview.Jobs.DueNow < 1 || overview.Stages.DurationSamples < 1 {
		t.Fatalf("overview evidence incomplete: %+v", overview)
	}
	if len(overview.Workers) != 3 || overview.Workers[0].Status != "running" {
		t.Fatalf("worker freshness: %+v", overview.Workers)
	}
}
