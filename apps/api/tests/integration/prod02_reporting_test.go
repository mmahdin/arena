package integration

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
)

// Disposable local SQL Server 2022 only: the existing aiDatabase harness
// refuses nonlocal or shared databases and deletes its test database.
func TestPROD02AggregateViewsAndReaderPermissions(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	f := aiSetup(t, db)
	ctx := context.Background()
	// A one-person synthetic cohort: registration 10 days ago; exact chat
	// re-engagement on D1 and D7, with D30 intentionally immature.
	aiExec(t, db, `INSERT dbo.product_events(id,user_id,event_name,subject_id,occurred_at)
      VALUES(NEWID(),@p1,'user_registered',@p1,DATEADD(day,-10,CONVERT(date,SYSUTCDATETIME())))`, f.a)
	for _, event := range []struct {
		name    string
		day     int
		subject uuid.UUID
	}{
		{"onboarding_started", -9, f.a},
		{"onboarding_completed", -8, f.a},
		{"ai_turn_admitted", -10, uuid.New()},
		{"ai_turn_admitted", -9, uuid.New()},
		{"ai_turn_admitted", -3, uuid.New()},
	} {
		aiExec(t, db, `INSERT dbo.product_events(id,user_id,event_name,subject_id,occurred_at)
          VALUES(NEWID(),@p1,@p2,@p3,DATEADD(day,@p4,CONVERT(date,SYSUTCDATETIME())))`, f.a, event.name, event.subject, event.day)
	}
	var registrations, started, completed int64
	if err := db.QueryRowContext(ctx, `SELECT registered_test_users,started_test_users,completed_test_users
      FROM dbo.v_product_onboarding_cohorts_synthetic`).Scan(&registrations, &started, &completed); err != nil ||
		registrations != 1 || started != 1 || completed != 1 {
		t.Fatalf("bad onboarding cohort: %v / %d %d %d", err, registrations, started, completed)
	}
	var cohortSize, d1, d7 int64
	var d30 sql.NullInt64
	if err := db.QueryRowContext(ctx, `SELECT first_chat_test_users,d1_returning_test_users,d7_returning_test_users,d30_returning_test_users
      FROM dbo.v_product_chat_retention_synthetic`).Scan(&cohortSize, &d1, &d7, &d30); err != nil ||
		cohortSize != 1 || d1 != 1 || d7 != 1 || d30.Valid {
		t.Fatalf("bad retention: %v / %d %d %d %v", err, cohortSize, d1, d7, d30)
	}
	var active int
	if err := db.QueryRowContext(ctx, `SELECT chat_active_test_users FROM dbo.v_product_daily_engagement_synthetic
      WHERE activity_date_utc=DATEADD(day,-9,CONVERT(date,SYSUTCDATETIME()))`).Scan(&active); err != nil || active != 1 {
		t.Fatalf("bad chat-active metric: %v / %d", err, active)
	}
	// Privacy invariant: no identity, chat text, or attempt-level metadata
	// are returned from any reporting view.
	for _, name := range []string{
		"v_product_daily_events_synthetic", "v_product_onboarding_cohorts_synthetic",
		"v_product_daily_engagement_synthetic", "v_product_chat_retention_synthetic",
		"v_product_ai_usage_daily_synthetic",
	} {
		var unsafe int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sys.columns WHERE object_id=OBJECT_ID(@p1)
          AND name IN ('user_id','subject_id','turn_id','phone','email','content','prompt','model','provider','test_key','request_id')`, "dbo."+name).Scan(&unsafe); err != nil || unsafe != 0 {
			t.Fatalf("sensitive reporting column in %s: %v / %d", name, err, unsafe)
		}
	}
	// Emulate a true role member without creating a server-level SQL login.
	aiExec(t, db, `CREATE USER [prod02_readonly_test] WITHOUT LOGIN`)
	aiExec(t, db, `ALTER ROLE theia_analytics_reader ADD MEMBER [prod02_readonly_test]`)
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `EXECUTE AS USER=N'prod02_readonly_test'`); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = conn.ExecContext(context.Background(), `REVERT`) }()
	for _, sqlQuery := range []string{
		`SELECT COUNT(*) FROM dbo.v_product_daily_events_synthetic`,
		`SELECT COUNT(*) FROM dbo.v_product_onboarding_cohorts_synthetic`,
		`SELECT COUNT(*) FROM dbo.v_product_daily_engagement_synthetic`,
		`SELECT COUNT(*) FROM dbo.v_product_chat_retention_synthetic`,
		`SELECT COUNT(*) FROM dbo.v_product_ai_usage_daily_synthetic`,
	} {
		var count int
		if err := conn.QueryRowContext(ctx, sqlQuery).Scan(&count); err != nil {
			t.Fatalf("reader cannot query approved view: %v", err)
		}
	}
	for _, sqlQuery := range []string{
		`SELECT COUNT(*) FROM dbo.product_events`,
		`SELECT COUNT(*) FROM dbo.users`,
		`SELECT COUNT(*) FROM dbo.messages`,
		`SELECT COUNT(*) FROM dbo.ai_llm_attempts`,
		`SELECT COUNT(*) FROM dbo.turns`,
	} {
		var count int
		if err := conn.QueryRowContext(ctx, sqlQuery).Scan(&count); err == nil {
			t.Fatalf("reader unexpectedly accessed restricted base table")
		}
	}
}
