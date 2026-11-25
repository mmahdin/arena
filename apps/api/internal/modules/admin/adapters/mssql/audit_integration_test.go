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

// This test deliberately leaves immutable rows behind. Run ONLY against a
// disposable SQL Server database with all migrations through 0026 applied.
// Neither the CI nor a developer machine should accidentally run it against
// shared/prod data.
func TestAuditSQLDisposableDatabase(t *testing.T) {
	dsn := os.Getenv("THEIA_DISPOSABLE_AUDIT_SQL_DSN")
	if dsn == "" || os.Getenv("THEIA_DISPOSABLE_AUDIT_SQL_ACK") != "YES_DISPOSABLE_ONLY" {
		t.Skip("requires dedicated disposable SQL Server and explicit destructive-test acknowledgement")
	}
	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	staffID := uuid.New()
	if _, err := db.ExecContext(ctx, `INSERT INTO dbo.staff_accounts(id,active) VALUES(CONVERT(uniqueidentifier,@p1),1)`, staffID.String()); err != nil {
		t.Fatal("migrations through 0026 required:", err)
	}
	repo := NewAuditRepository(db)
	event := domain.AuditEvent{ID: uuid.New(), ActorStaffID: staffID, RequestID: uuid.New(),
		ActionCode: domain.ActionAdminMeRead, ResourceKind: "admin", OutcomeCode: domain.OutcomeAuthorized}
	if err := repo.Append(ctx, event); err != nil {
		t.Fatal(err)
	}
	filter := domain.NewAuditFilter(time.Now().Add(time.Hour))
	filter.From = time.Now().Add(-time.Hour)
	filter.To = time.Now().Add(time.Hour)
	filter.Limit = 1
	filter.ActorStaffID = staffID
	rows, _, err := repo.List(ctx, filter)
	if err != nil || len(rows) != 1 || rows[0].ID != event.ID || rows[0].RequestID != event.RequestID {
		t.Fatalf("audit read did not reconcile: %+v %v", rows, err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE dbo.admin_audit_events SET action_code='tampered' WHERE id=CONVERT(uniqueidentifier,@p1)`, event.ID.String()); err == nil {
		t.Fatal("audit UPDATE unexpectedly succeeded")
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM dbo.admin_audit_events WHERE id=CONVERT(uniqueidentifier,@p1)`, event.ID.String()); err == nil {
		t.Fatal("audit DELETE unexpectedly succeeded")
	}
	if _, err := db.ExecContext(ctx, `UPDATE dbo.staff_accounts SET active=0,deactivated_at=SYSUTCDATETIME() WHERE id=CONVERT(uniqueidentifier,@p1)`, staffID.String()); err != nil {
		t.Fatal(err)
	}
	event.ID = uuid.New()
	if err := repo.Append(ctx, event); err == nil {
		t.Fatal("deactivated staff should not append an authorized audit event")
	}
}
