package mssql

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/microsoft/go-mssqldb"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
)

// This test INSERTS/DELETES rows. Never run on staging, production or any
// shared database. Migrations through 0029 must already be applied.
func TestBO06InspectorDisposableSQL(t *testing.T) {
	dsn := os.Getenv("THEIA_DISPOSABLE_INSPECTOR_SQL_DSN")
	if dsn == "" || os.Getenv("THEIA_DISPOSABLE_INSPECTOR_SQL_ACK") != "YES_DISPOSABLE_ONLY" {
		t.Skip("requires explicitly acknowledged disposable SQL Server with migration 0029")
	}
	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Millisecond)
	at := now.Add(-2 * time.Minute)
	userID, convID, turn1, turn2, jobID, stageID, attemptID, workerID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	execute := func(q string, args ...any) {
		t.Helper()
		if _, e := db.ExecContext(ctx, q, args...); e != nil {
			t.Fatal(e)
		}
	}
	defer func() {
		cleanup := context.Background()
		for _, entry := range []struct {
			query string
			args  []any
		}{
			{`DELETE FROM dbo.ai_worker_heartbeats WHERE instance_id=CONVERT(uniqueidentifier,@p1)`, []any{workerID.String()}},
			{`DELETE FROM dbo.ai_stage_events WHERE id=CONVERT(uniqueidentifier,@p1)`, []any{stageID.String()}},
			{`DELETE FROM dbo.ai_llm_attempts WHERE id=CONVERT(uniqueidentifier,@p1)`, []any{attemptID.String()}},
			{`DELETE FROM dbo.ai_jobs WHERE id=CONVERT(uniqueidentifier,@p1)`, []any{jobID.String()}},
			{`DELETE FROM dbo.turns WHERE id IN (CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2))`, []any{turn1.String(), turn2.String()}},
			{`DELETE FROM dbo.conversations WHERE id=CONVERT(uniqueidentifier,@p1)`, []any{convID.String()}},
			{`DELETE FROM dbo.users WHERE id=CONVERT(uniqueidentifier,@p1)`, []any{userID.String()}},
		} {
			if _, e := db.ExecContext(cleanup, entry.query, entry.args...); e != nil {
				t.Logf("cleanup: %v", e)
			}
		}
	}()
	execute(`INSERT dbo.users(id,test_key,is_test) VALUES(CONVERT(uniqueidentifier,@p1),@p2,1)`, userID.String(), "bo06-"+userID.String())
	execute(`INSERT dbo.conversations(id,user_id,title,created_at) VALUES(CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),N'PRIVATE DO NOT LEAK',@p3)`, convID.String(), userID.String(), at)
	for _, id := range []uuid.UUID{turn1, turn2} {
		execute(`INSERT dbo.turns(id,conversation_id,user_id,input_hash,status,lifecycle_version,attempt_count,error_code,user_message_id,created_at)
          VALUES(CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),CONVERT(uniqueidentifier,@p3),@p4,N'failed',1,2,N'private text must not leak',CONVERT(uniqueidentifier,@p5),@p6)`, id.String(), convID.String(), userID.String(), make([]byte, 32), uuid.NewString(), at)
	}
	execute(`INSERT dbo.ai_jobs(id,user_id,turn_id,kind,status,attempts,max_attempts,last_error_code,run_after,created_at,payload_json)
      VALUES(CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),CONVERT(uniqueidentifier,@p3),N'memory_write',N'failed',2,6,N'job_error',@p4,@p4,N'{"secret":"PRIVATE DO NOT LEAK"}')`, jobID.String(), userID.String(), turn1.String(), at)
	execute(`INSERT dbo.ai_stage_events(id,turn_id,stage,status,duration_ms,attempt,rule_id,recorded_at)
      VALUES(CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),'brain','failed',51,2,'rule_critical',@p3)`, stageID.String(), turn1.String(), at)
	execute(`INSERT dbo.ai_llm_attempts(id,turn_id,kind,tier,provider,model,started_at,finished_at,status,duration_ms,
      input_tokens,output_tokens,cached_tokens,input_rate_micro_usd_per_m,output_rate_micro_usd_per_m,cached_rate_micro_usd_per_m,cost_status)
      VALUES(CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),'structured','balanced','test-provider','model-v1',@p3,@p4,'failed',100,
      NULL,NULL,NULL,0,0,0,'unknown')`, attemptID.String(), turn1.String(), at, at.Add(100*time.Millisecond))
	execute(`INSERT dbo.ai_worker_heartbeats(instance_id,worker_kind,status,last_seen_at) VALUES(CONVERT(uniqueidentifier,@p1),'memory','running',@p2)`, workerID.String(), now.Add(-90*time.Second))
	// Existing rows and privileged SQL writes may predate telemetry label
	// validation. The Admin reader must never reflect their free-text values.
	execute(`UPDATE dbo.ai_stage_events SET stage=N'PRIVATE DO NOT LEAK' WHERE id=CONVERT(uniqueidentifier,@p1)`, stageID.String())
	execute(`UPDATE dbo.ai_llm_attempts SET model=N'PRIVATE DO NOT LEAK' WHERE id=CONVERT(uniqueidentifier,@p1)`, attemptID.String())
	r := NewInspectorRepository(db)
	filter := domain.InspectorFilter{From: now.Add(-time.Hour), To: now, Limit: 1, Status: "failed"}
	page, err := r.ListTurns(ctx, filter)
	if err != nil || len(page.Items) != 1 || page.NextCursor == nil || page.Items[0].ErrorCode != nil {
		t.Fatalf("turn page %+v: %v", page, err)
	}
	filter.Cursor, err = domain.DecodeInspectorCursor(*page.NextCursor)
	if err != nil {
		t.Fatal(err)
	}
	second, err := r.ListTurns(ctx, filter)
	if err != nil || len(second.Items) < 1 || second.Items[0].ID == page.Items[0].ID {
		t.Fatalf("cursor lost tie %+v %v", second, err)
	}
	detail, err := r.GetTurn(ctx, turn1)
	if err != nil || len(detail.Stages) != 1 || len(detail.LLMAttempts) != 1 || len(detail.Jobs) != 1 || detail.Turn.ErrorCode != nil {
		t.Fatalf("detail incomplete %+v %v", detail, err)
	}
	if detail.Stages[0].Stage != "unclassified" || detail.LLMAttempts[0].Model != "unclassified" {
		t.Fatalf("unsafe label was not redacted: %+v", detail)
	}
	for _, text := range []string{detail.Stages[0].Stage, detail.Jobs[0].Kind, detail.LLMAttempts[0].Provider, detail.LLMAttempts[0].Model} {
		if strings.Contains(text, "PRIVATE") {
			t.Fatal("private payload leaked")
		}
	}
	if detail.LLMAttempts[0].InputTokens != nil || detail.LLMAttempts[0].EstimatedMicroUsd != nil {
		t.Fatal("unknown usage became zero")
	}
	missing, err := r.GetTurn(ctx, uuid.New())
	_ = missing
	if !errors.Is(err, domain.ErrTurnNotFound) {
		t.Fatalf("missing turn: %v", err)
	}
	jobFilter := domain.InspectorFilter{From: now.Add(-time.Hour), To: now, Limit: 5, Kind: "memory_write", Status: "failed"}
	jobs, err := r.ListJobs(ctx, jobFilter)
	if err != nil || len(jobs.Items) < 1 || jobs.Items[0].ID != jobID || jobs.Items[0].LastErrorCode == nil {
		t.Fatalf("jobs %+v %v", jobs, err)
	}
	workers, err := r.ListWorkers(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, w := range workers.Instances {
		if w.InstanceID == workerID {
			found = true
			if w.Status != "stale" {
				t.Fatalf("stale heartbeat marked healthy: %+v", w)
			}
		}
	}
	if !found {
		t.Fatal("test worker not visible")
	}
}
