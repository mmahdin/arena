package integration

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	aidb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/database"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/migrations"
)

// Uses its own disposable database. Never runs against production, staging or
// an existing developer database, and never inserts real chat/user content.
func TestBO04DurableTelemetrySQL(t *testing.T) {
	if os.Getenv("WELLBEING_INTEGRATION") != "true" {
		t.Skip("requires opt-in SQL Server integration")
	}
	if os.Getenv("APP_ENV") != "local" {
		t.Fatal("requires APP_ENV=local")
	}
	u, err := url.Parse(os.Getenv("DATABASE_URL"))
	if err != nil || u.Scheme != "sqlserver" || (u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1" && u.Hostname() != "mssql") {
		t.Fatal("disposable local SQL Server URL required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	name := "theia_telemetry_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	q := u.Query()
	q.Set("database", name)
	u.RawQuery = q.Encode()
	if err = database.EnsureLocalDatabase(ctx, u.String()); err != nil {
		t.Fatal(err)
	}
	q.Set("database", "master")
	masterURL := *u
	masterURL.RawQuery = q.Encode()
	master, err := database.Connect(ctx, masterURL.String())
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer func() {
		cleanup, done := context.WithTimeout(context.Background(), 30*time.Second)
		defer done()
		if _, e := master.ExecContext(cleanup, "ALTER DATABASE ["+name+"] SET SINGLE_USER WITH ROLLBACK IMMEDIATE; DROP DATABASE ["+name+"]"); e != nil {
			t.Errorf("cleanup: %v", e)
		}
	}()
	db, err := database.Connect(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = migrations.Apply(ctx, db, "../../db/migrations"); err != nil {
		t.Fatal(err)
	}
	store, err := aidb.NewTelemetryStore(db)
	if err != nil {
		t.Fatal(err)
	}
	id, turn, instance := uuid.New(), uuid.New(), uuid.New()
	start := time.Now().UTC().Add(-time.Second).Truncate(time.Millisecond)
	if err = store.BeginAttempt(ctx, llm.AttemptStart{ID: id, TurnID: turn, Kind: "structured", Tier: ports.ComponentBrain, Provider: "provider", Model: "model-1", StartedAt: start, InputRate: 1000000}); err != nil {
		t.Fatal(err)
	}
	// Duplicate begin must not produce another charge or a second row.
	if err = store.BeginAttempt(ctx, llm.AttemptStart{ID: id, TurnID: turn, Kind: "structured", Tier: ports.ComponentBrain, Provider: "provider", Model: "model-1", StartedAt: start, InputRate: 1000000}); err == nil {
		t.Fatal("duplicate begin accepted")
	}
	finish := llm.AttemptFinish{ID: id, FinishedAt: start.Add(time.Second), Duration: time.Second, Status: "succeeded", UsageKnown: true, InTokens: 40, OutTokens: 5, CostStatus: "estimated", CostMicroUSD: 40}
	if err = store.FinishAttempt(ctx, finish); err != nil {
		t.Fatal(err)
	}
	if err = store.FinishAttempt(ctx, finish); err != nil {
		t.Fatal("completion replay should be idempotent", err)
	}
	if err = store.FinishAttempt(ctx, llm.AttemptFinish{ID: uuid.New(), FinishedAt: time.Now(), Status: "failed", CostStatus: "unknown"}); err == nil {
		t.Fatal("unknown completion accepted")
	}
	unknownID := uuid.New()
	if err = store.BeginAttempt(ctx, llm.AttemptStart{ID: unknownID, Kind: "stream", Tier: ports.ComponentGenerator, Provider: "provider", Model: "model-1", StartedAt: start}); err != nil {
		t.Fatal(err)
	}
	if err = store.RecordStage(ctx, aidb.StageEvent{TurnID: turn, Stage: "brain", Status: "degraded", Duration: 10 * time.Millisecond, Attempt: 1}); err != nil {
		t.Fatal(err)
	}
	if err = store.RecordStage(ctx, aidb.StageEvent{TurnID: turn, Stage: "raw_sensitive_payload", Status: "success"}); err == nil {
		t.Fatal("unsanitized stage accepted")
	}
	for _, status := range []string{"running", "running", "stopped"} {
		if err = store.Heartbeat(ctx, instance, "memory", status); err != nil {
			t.Fatal(err)
		}
	}
	var attempts, unknown, cost, tokens int
	if err = db.QueryRowContext(ctx, `SELECT COUNT(*),SUM(CASE WHEN status='started' THEN 1 ELSE 0 END),
      SUM(CASE WHEN cost_status='estimated' THEN CAST(cost_micro_usd AS int) ELSE 0 END),
      SUM(CASE WHEN input_tokens IS NOT NULL THEN CAST(input_tokens AS int) ELSE 0 END)
      FROM dbo.ai_llm_attempts`).Scan(&attempts, &unknown, &cost, &tokens); err != nil {
		t.Fatal(err)
	}
	if attempts != 2 || unknown != 1 || cost != 40 || tokens != 40 {
		t.Fatalf("unexpected persisted attempts %d, started %d, cost %d, tokens %d", attempts, unknown, cost, tokens)
	}
	var stageCount int
	if err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM dbo.ai_stage_events").Scan(&stageCount); err != nil || stageCount != 1 {
		t.Fatalf("stages %d err %v", stageCount, err)
	}
	var status string
	if err = db.QueryRowContext(ctx, "SELECT status FROM dbo.ai_worker_heartbeats WHERE instance_id=@p1 AND worker_kind='memory'", instance.String()).Scan(&status); err != nil || status != "stopped" {
		t.Fatalf("heartbeat %q err %v", status, err)
	}
}
