package mssql

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/microsoft/go-mssqldb"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
)

// This test writes IMMUTABLE audit-style eval rows and a synthetic staff
// account. It must ONLY run against a throwaway, isolated SQL Server DB;
// the inserted rows cannot be deleted without dropping the disposable DB.
// Migrations through 0030 must already be applied.
func TestBO08EvaluationsDisposableSQL(t *testing.T) {
	dsn := os.Getenv("THEIA_DISPOSABLE_EVAL_SQL_DSN")
	if dsn == "" || os.Getenv("THEIA_DISPOSABLE_EVAL_SQL_ACK") != "YES_DESTROY_AFTER_TEST" {
		t.Skip("requires explicitly acknowledged disposable SQL Server, migration 0030")
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
	staffID := uuid.New()
	if _, err = db.ExecContext(ctx, `INSERT dbo.staff_accounts(id,active) VALUES(CONVERT(uniqueidentifier,@p1),1)`, staffID.String()); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("../../../../../../../docs/quality/test-results/ai-core-eval.json")
	if err != nil {
		t.Fatal(err)
	}
	r, _, err := domain.ParseEvalReport(raw)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(raw)
	run := domain.EvalRun{
		ID: uuid.New(), ReportSHA256: hex.EncodeToString(hash[:]), ImportedByStaffID: staffID,
		Method: r.Method, EvidenceType: domain.ReportEvidence(r.Method), Status: domain.EvalStatus(r, 0),
		Totals: r.Totals, Coverage: domain.ReportCoverage(r), ArtifactHashVerified: true,
	}
	adapter := NewEvaluationRepository(db)
	stored, created, err := adapter.Import(ctx, domain.EvalRunDetail{Run: run, Report: r, BaselineRegressions: []domain.EvalRegression{}})
	if err != nil || !created || stored.ID != run.ID {
		t.Fatalf("insert %+v %v", stored, err)
	}
	rerun, created, err := adapter.Import(ctx, domain.EvalRunDetail{Run: run, Report: r, BaselineRegressions: []domain.EvalRegression{}})
	if err != nil || created || rerun.ID != run.ID {
		t.Fatalf("duplicate %+v %v", rerun, err)
	}
	read, err := adapter.Get(ctx, run.ID)
	if err != nil || read.Run.Status != "passed" || read.Run.SourceAttested || read.Run.ClinicallyApproved || read.Report.Totals.Scenarios != 3 {
		t.Fatalf("read %+v %v", read.Run, err)
	}
	// Time window around the SQL Server-generated imported_at timestamp.
	f := domain.EvalFilter{From: stored.ImportedAt.Add(-time.Second), To: stored.ImportedAt.Add(time.Second), Limit: 25}
	page, err := adapter.List(ctx, f)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range page.Items {
		if item.ID == run.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("missing eval in paginated page")
	}
	if _, err = db.ExecContext(ctx, `UPDATE dbo.admin_eval_runs SET status='failed' WHERE id=CONVERT(uniqueidentifier,@p1)`, run.ID.String()); err == nil {
		t.Fatal("immutable eval update succeeded")
	}
	if _, err = db.ExecContext(ctx, `DELETE FROM dbo.admin_eval_runs WHERE id=CONVERT(uniqueidentifier,@p1)`, run.ID.String()); err == nil {
		t.Fatal("immutable eval deletion succeeded")
	}
}
