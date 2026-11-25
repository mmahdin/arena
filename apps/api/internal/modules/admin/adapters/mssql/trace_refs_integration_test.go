package mssql

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/microsoft/go-mssqldb"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
	aidb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
)

// This test leaves fixtures in the database and MUST be run only on a fresh, explicitly
// acknowledged disposable SQL Server 2022 database migrated through 0034.
func TestOBS03TraceReferenceDisposableSQL(t *testing.T) {
	dsn := os.Getenv("THEIA_DISPOSABLE_TRACE_SQL_DSN")
	if dsn == "" || os.Getenv("THEIA_DISPOSABLE_TRACE_SQL_ACK") != "YES_DISPOSABLE_ONLY" {
		t.Skip("requires disposable SQL Server with migrations through 0034 and explicit acknowledgement")
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
	user, conv, turn1, turn2, turn3, job := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT dbo.users(id,test_key,is_test) VALUES(CONVERT(uniqueidentifier,@p1),@p2,1)`, user.String(), "obs03-"+user.String())
	exec(`INSERT dbo.conversations(id,user_id,title) VALUES(CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),N'PRIVATE DO NOT LEAK')`, conv.String(), user.String())
	for _, turn := range []uuid.UUID{turn1, turn2, turn3} {
		message := uuid.NewString()
		exec(`INSERT dbo.turns(id,conversation_id,user_id,input_hash,status,lifecycle_version,attempt_count,user_message_id)
              VALUES(CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),CONVERT(uniqueidentifier,@p3),@p4,N'failed',1,1,CONVERT(uniqueidentifier,@p5))`,
			turn.String(), conv.String(), user.String(), make([]byte, 32), message)
		exec(`INSERT dbo.messages(id,conversation_id,turn_id,role,content)
              VALUES(CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),CONVERT(uniqueidentifier,@p3),N'user',N'Synthetic OBS-03 fixture')`,
			message, conv.String(), turn.String())
	}
	exec(`INSERT dbo.ai_jobs(id,user_id,turn_id,kind,status,attempts,payload_json)
          VALUES(CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),CONVERT(uniqueidentifier,@p3),N'memory_write',N'running',2,N'{"secret":"PRIVATE DO NOT LEAK"}')`, job.String(), user.String(), turn1.String())
	writer := aidb.NewTraceReferenceRepository(db)
	traces := []trace.RootReference{
		{TurnID: turn1, Kind: "turn", TraceID: "00000000000000000000000000000001", RootSpanID: "0000000000000001"},
		{TurnID: turn1, Kind: "memory", JobID: job, JobAttempt: 2, TraceID: "00000000000000000000000000000002", RootSpanID: "0000000000000002"},
		{TurnID: turn1, Kind: "turn", TraceID: "00000000000000000000000000000003", RootSpanID: "0000000000000003"},
		{TurnID: turn2, Kind: "turn", TraceID: "00000000000000000000000000000004", RootSpanID: "0000000000000004"},
	}
	for _, r := range traces {
		if err := writer.WriteTraceReference(ctx, r); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.WriteTraceReference(ctx, trace.RootReference{TurnID: turn2, Kind: "memory", JobID: job, JobAttempt: 2, TraceID: "00000000000000000000000000000005", RootSpanID: "0000000000000005"}); err == nil {
		t.Fatal("cross-Turn job reference accepted")
	}
	if err := writer.WriteTraceReference(ctx, trace.RootReference{TurnID: turn1, Kind: "turn", TraceID: "not_w3c", RootSpanID: "0000000000000005"}); err == nil {
		t.Fatal("malformed trace id accepted")
	}
	reader := NewTraceReferenceReader(db)
	f := domain.TraceReferenceFilter{TurnID: turn1, Limit: 1}
	seen := map[string]bool{}
	for n := 0; n < 3; n++ {
		page, err := reader.ListTraceReferences(ctx, f)
		if err != nil || len(page.Items) != 1 {
			t.Fatalf("page %d %+v: %v", n, page, err)
		}
		item := page.Items[0]
		if item.JobID != nil && (*item.JobID != job || item.JobAttempt == nil || *item.JobAttempt != 2) {
			t.Fatal("job reference mismatch")
		}
		if seen[item.TraceID] {
			t.Fatal("duplicate cursor page")
		}
		seen[item.TraceID] = true
		if n < 2 {
			if page.NextCursor == nil {
				t.Fatal("early end of references")
			}
			f.Cursor, err = domain.DecodeInspectorCursor(*page.NextCursor)
			if err != nil {
				t.Fatal(err)
			}
		} else if page.NextCursor != nil {
			t.Fatal("unexpected next cursor")
		}
	}
	if len(seen) != 3 {
		t.Fatal("missing admitted Turn/worker attempts")
	}
	empty, err := reader.ListTraceReferences(ctx, domain.TraceReferenceFilter{TurnID: turn3, Limit: 25})
	if err != nil || empty.Items == nil || len(empty.Items) != 0 {
		t.Fatalf("empty Turn: %+v %v", empty, err)
	}
	_, err = reader.ListTraceReferences(ctx, domain.TraceReferenceFilter{TurnID: uuid.New(), Limit: 25})
	if !errors.Is(err, domain.ErrTurnNotFound) {
		t.Fatalf("missing Turn: %v", err)
	}
}
