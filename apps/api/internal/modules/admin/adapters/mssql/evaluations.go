package mssql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
)

// EvaluationRepository owns only allowlisted synthetic eval_report.v1 rows.
// There is no arbitrary SQL, fixture execution or content retrieval path.
type EvaluationRepository struct{ db *sql.DB }

func NewEvaluationRepository(db *sql.DB) *EvaluationRepository { return &EvaluationRepository{db: db} }

func (r *EvaluationRepository) Import(ctx context.Context, detail domain.EvalRunDetail) (domain.EvalRun, bool, error) {
	run := detail.Run
	if r == nil || r.db == nil || run.ID == uuid.Nil || run.ImportedByStaffID == uuid.Nil || !run.ArtifactHashVerified || run.SourceAttested || run.ClinicallyApproved {
		return domain.EvalRun{}, false, domain.ErrInvalidEval
	}
	reportJSON, err := json.Marshal(detail.Report)
	if err != nil {
		return domain.EvalRun{}, false, err
	}
	report, _, err := domain.ParseEvalReport(reportJSON)
	if err != nil || report.Method != run.Method || run.Coverage != domain.ReportCoverage(report) || run.Totals != report.Totals || run.Status != domain.EvalStatus(report, len(detail.BaselineRegressions)) {
		return domain.EvalRun{}, false, domain.ErrInvalidEval
	}
	if len(run.ReportSHA256) != 64 {
		return domain.EvalRun{}, false, domain.ErrInvalidEval
	}
	totals, err := json.Marshal(run.Totals)
	if err != nil {
		return domain.EvalRun{}, false, err
	}
	coverage, err := json.Marshal(run.Coverage)
	if err != nil {
		return domain.EvalRun{}, false, err
	}
	diffs, err := json.Marshal(detail.BaselineRegressions)
	if err != nil {
		return domain.EvalRun{}, false, err
	}
	if run.RegressionCount != len(detail.BaselineRegressions) {
		return domain.EvalRun{}, false, domain.ErrInvalidEval
	}
	var baseline any
	if run.BaselineRunID != nil {
		baseline = run.BaselineRunID.String()
	}
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return domain.EvalRun{}, false, err
	}
	defer tx.Rollback()
	var existing string
	err = tx.QueryRowContext(ctx, `SELECT CONVERT(varchar(36),id) FROM dbo.admin_eval_runs WITH (UPDLOCK,HOLDLOCK)
         WHERE report_sha256=@p1 AND ((baseline_run_id IS NULL AND @p2 IS NULL)
           OR baseline_run_id=CONVERT(uniqueidentifier,@p2))`, run.ReportSHA256, baseline).Scan(&existing)
	if err == nil {
		id, e := uuid.Parse(existing)
		if e != nil {
			return domain.EvalRun{}, false, e
		}
		// Return the stored row: duplicate imports never mutate actor/time/status.
		if e = tx.Commit(); e != nil {
			return domain.EvalRun{}, false, e
		}
		prior, e := r.Get(ctx, id)
		return prior.Run, false, e
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return domain.EvalRun{}, false, err
	}
	_, err = tx.ExecContext(ctx, `INSERT dbo.admin_eval_runs
        (id,report_sha256,imported_by_staff_id,baseline_run_id,method,evidence_type,status,
         scenario_count,failed_scenarios,blockers,regression_count,totals_json,coverage_json,
         report_json,baseline_regressions_json,artifact_hash_verified,source_attested,clinically_approved)
        VALUES(CONVERT(uniqueidentifier,@p1),@p2,CONVERT(uniqueidentifier,@p3),CONVERT(uniqueidentifier,@p4),
          @p5,@p6,@p7,@p8,@p9,@p10,@p11,@p12,@p13,@p14,@p15,1,0,0)`,
		run.ID.String(), run.ReportSHA256, run.ImportedByStaffID.String(), baseline, run.Method, run.EvidenceType, run.Status,
		run.Totals.Scenarios, run.Totals.FailedScenarios, run.Totals.Blockers, run.RegressionCount, string(totals), string(coverage), string(reportJSON), string(diffs))
	if err != nil {
		return domain.EvalRun{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return domain.EvalRun{}, false, err
	}
	// Let SQL Server assign its authoritative imported_at timestamp.
	inserted, err := r.Get(ctx, run.ID)
	if err != nil {
		return domain.EvalRun{}, false, err
	}
	return inserted.Run, true, nil
}

func (r *EvaluationRepository) Get(ctx context.Context, id uuid.UUID) (domain.EvalRunDetail, error) {
	if id == uuid.Nil {
		return domain.EvalRunDetail{}, domain.ErrInvalidEval
	}
	if r == nil || r.db == nil {
		return domain.EvalRunDetail{}, errors.New("evaluation store unavailable")
	}
	var record domain.EvalRunDetail
	var runID, staffID, baseline sql.NullString
	var totalsJSON, coverageJSON, reportJSON, regsJSON string
	err := r.db.QueryRowContext(ctx, `SELECT CONVERT(varchar(36),id),report_sha256,imported_at,
      CONVERT(varchar(36),imported_by_staff_id),CONVERT(varchar(36),baseline_run_id),
      method,evidence_type,status,totals_json,coverage_json,report_json,baseline_regressions_json,
      artifact_hash_verified,source_attested,clinically_approved,regression_count
      FROM dbo.admin_eval_runs WHERE id=CONVERT(uniqueidentifier,@p1)`, id.String()).Scan(
		&runID, &record.Run.ReportSHA256, &record.Run.ImportedAt, &staffID, &baseline,
		&record.Run.Method, &record.Run.EvidenceType, &record.Run.Status, &totalsJSON, &coverageJSON, &reportJSON, &regsJSON,
		&record.Run.ArtifactHashVerified, &record.Run.SourceAttested, &record.Run.ClinicallyApproved, &record.Run.RegressionCount)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.EvalRunDetail{}, domain.ErrEvalNotFound
	}
	if err != nil {
		return domain.EvalRunDetail{}, err
	}
	record.Run.ID, err = uuid.Parse(runID.String)
	if err != nil {
		return domain.EvalRunDetail{}, err
	}
	record.Run.ImportedByStaffID, err = uuid.Parse(staffID.String)
	if err != nil {
		return domain.EvalRunDetail{}, err
	}
	if baseline.Valid {
		uid, e := uuid.Parse(baseline.String)
		if e != nil {
			return domain.EvalRunDetail{}, e
		}
		record.Run.BaselineRunID = &uid
	}
	rReport, _, err := domain.ParseEvalReport([]byte(reportJSON))
	if err != nil {
		return domain.EvalRunDetail{}, err
	}
	record.Report = rReport
	record.Run.Totals = rReport.Totals
	record.Run.Coverage = domain.ReportCoverage(rReport)
	if json.Unmarshal([]byte(regsJSON), &record.BaselineRegressions) != nil || record.Run.RegressionCount != len(record.BaselineRegressions) ||
		record.Run.Status != domain.EvalStatus(rReport, len(record.BaselineRegressions)) ||
		record.Run.Method != rReport.Method || record.Run.EvidenceType != domain.ReportEvidence(rReport.Method) ||
		!record.Run.ArtifactHashVerified || record.Run.SourceAttested || record.Run.ClinicallyApproved {
		return domain.EvalRunDetail{}, domain.ErrInvalidEval
	}
	if err = domain.ValidateEvalRegressions(record.BaselineRegressions); err != nil {
		return domain.EvalRunDetail{}, err
	}
	record.Run.ImportedAt = record.Run.ImportedAt.UTC()
	return record, nil
}

func (r *EvaluationRepository) List(ctx context.Context, f domain.EvalFilter) (domain.EvalPage, error) {
	if err := f.Validate(); err != nil {
		return domain.EvalPage{}, err
	}
	if r == nil || r.db == nil {
		return domain.EvalPage{}, errors.New("evaluation store unavailable")
	}
	var cursorAt, cursorID any
	if f.Cursor != nil {
		cursorAt = f.Cursor.At.UTC()
		cursorID = f.Cursor.ID.String()
	}
	rows, err := r.db.QueryContext(ctx, `SELECT TOP (@p1)
      CONVERT(varchar(36),id),report_sha256,imported_at,CONVERT(varchar(36),imported_by_staff_id),
      CONVERT(varchar(36),baseline_run_id),method,evidence_type,status,totals_json,coverage_json,
      artifact_hash_verified,source_attested,clinically_approved,regression_count
      FROM dbo.admin_eval_runs
      WHERE imported_at >= @p2 AND imported_at < @p3
      AND (@p4='' OR status=@p4)
      AND (@p5 IS NULL OR imported_at < @p5 OR (imported_at=@p5 AND id<CONVERT(uniqueidentifier,@p6)))
      ORDER BY imported_at DESC,id DESC`, f.Limit+1, f.From.UTC(), f.To.UTC(), f.Status, cursorAt, cursorID)
	if err != nil {
		return domain.EvalPage{}, err
	}
	defer rows.Close()
	items := make([]domain.EvalRun, 0, f.Limit+1)
	for rows.Next() {
		var run domain.EvalRun
		var runID, staffID string
		var baseline sql.NullString
		var totals, coverage string
		if err = rows.Scan(&runID, &run.ReportSHA256, &run.ImportedAt, &staffID, &baseline, &run.Method, &run.EvidenceType, &run.Status,
			&totals, &coverage, &run.ArtifactHashVerified, &run.SourceAttested, &run.ClinicallyApproved, &run.RegressionCount); err != nil {
			return domain.EvalPage{}, err
		}
		run.ID, err = uuid.Parse(runID)
		if err != nil {
			return domain.EvalPage{}, err
		}
		run.ImportedByStaffID, err = uuid.Parse(staffID)
		if err != nil {
			return domain.EvalPage{}, err
		}
		if baseline.Valid {
			uid, e := uuid.Parse(baseline.String)
			if e != nil {
				return domain.EvalPage{}, e
			}
			run.BaselineRunID = &uid
		}
		if json.Unmarshal([]byte(totals), &run.Totals) != nil || json.Unmarshal([]byte(coverage), &run.Coverage) != nil ||
			run.SourceAttested || run.ClinicallyApproved || !run.ArtifactHashVerified {
			return domain.EvalPage{}, domain.ErrInvalidEval
		}
		run.ImportedAt = run.ImportedAt.UTC().Truncate(time.Millisecond)
		items = append(items, run)
	}
	if err = rows.Err(); err != nil {
		return domain.EvalPage{}, err
	}
	var next *string
	if len(items) > f.Limit {
		items = items[:f.Limit]
		last := items[len(items)-1]
		s := (domain.InspectorCursor{At: last.ImportedAt, ID: last.ID}).Encode()
		next = &s
	}
	return domain.EvalPage{Items: items, NextCursor: next}, nil
}
