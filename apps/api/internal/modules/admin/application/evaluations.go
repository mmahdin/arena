package application

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/eval"
)

var ErrEvaluationsUnavailable = errors.New("admin evaluations unavailable")

type EvaluationStore interface {
	Import(context.Context, domain.EvalRunDetail) (domain.EvalRun, bool, error)
	List(context.Context, domain.EvalFilter) (domain.EvalPage, error)
	Get(context.Context, uuid.UUID) (domain.EvalRunDetail, error)
}

func NewWithEvaluationRepositories(audit AuditStore, overview OverviewStore, inspector InspectorStore, evaluations EvaluationStore) *Service {
	return &Service{audit: audit, overview: overview, inspector: inspector, evaluations: evaluations}
}

func (s *Service) evaluationAudit(ctx context.Context, p domain.Principal, requestID uuid.UUID, action string, id uuid.UUID) error {
	if err := s.RecordAuthorizedRead(ctx, p, requestID, action, id); err != nil {
		if errors.Is(err, ErrUnverifiedStaff) {
			return err
		}
		return ErrAuditUnavailable
	}
	return nil
}

// The importer never executes tests, opens a trace file, or claims the origin
// of a report was attested. SHA256 proves only byte integrity of the supplied
// artifact, not who produced it. Baseline differences are recomputed server-side
// using the existing WP-15 comparison policy (zero tolerance by default).
func (s *Service) ImportEvaluation(ctx context.Context, p domain.Principal, requestID uuid.UUID, raw []byte, expectedHash string, baselineID *uuid.UUID) (domain.EvalRun, bool, error) {
	report, _, err := domain.ParseEvalReport(raw)
	if err != nil {
		return domain.EvalRun{}, false, err
	}
	hash, err := domain.ValidateArtifactHash(raw, expectedHash)
	if err != nil {
		return domain.EvalRun{}, false, err
	}
	if baselineID != nil && *baselineID == uuid.Nil {
		return domain.EvalRun{}, false, domain.ErrInvalidEval
	}
	if err = s.evaluationAudit(ctx, p, requestID, domain.ActionEvaluationImport, uuid.Nil); err != nil {
		return domain.EvalRun{}, false, err
	}
	if s.evaluations == nil {
		return domain.EvalRun{}, false, ErrEvaluationsUnavailable
	}
	regressions := []domain.EvalRegression{}
	if baselineID != nil {
		baseline, readErr := s.evaluations.Get(ctx, *baselineID)
		if readErr != nil {
			if errors.Is(readErr, domain.ErrEvalNotFound) {
				return domain.EvalRun{}, false, readErr
			}
			return domain.EvalRun{}, false, ErrEvaluationsUnavailable
		}
		baseBytes, e := json.Marshal(baseline.Report)
		if e != nil {
			return domain.EvalRun{}, false, ErrEvaluationsUnavailable
		}
		newBytes, e := json.Marshal(report)
		if e != nil {
			return domain.EvalRun{}, false, ErrEvaluationsUnavailable
		}
		var baselineWP15, nowWP15 eval.Report
		if json.Unmarshal(baseBytes, &baselineWP15) != nil || json.Unmarshal(newBytes, &nowWP15) != nil {
			return domain.EvalRun{}, false, ErrEvaluationsUnavailable
		}
		for _, r := range eval.CompareReports(baselineWP15, nowWP15, eval.RegressionPolicy{}) {
			regressions = append(regressions, domain.EvalRegression{Scenario: r.Scenario, Assertion: r.Assertion, Metric: r.Metric, Code: r.Code})
		}
	}
	at := time.Now().UTC()
	run := domain.EvalRun{
		ID: uuid.New(), ReportSHA256: hash, ImportedAt: at, ImportedByStaffID: p.StaffID, BaselineRunID: baselineID,
		Method: report.Method, EvidenceType: domain.ReportEvidence(report.Method), Status: domain.EvalStatus(report, len(regressions)),
		Totals: report.Totals, Coverage: domain.ReportCoverage(report), ArtifactHashVerified: true,
		SourceAttested: false, ClinicallyApproved: false, RegressionCount: len(regressions),
	}
	saved, created, err := s.evaluations.Import(ctx, domain.EvalRunDetail{Run: run, Report: report, BaselineRegressions: regressions})
	if err != nil {
		if errors.Is(err, domain.ErrEvalConflict) {
			return domain.EvalRun{}, false, err
		}
		return domain.EvalRun{}, false, ErrEvaluationsUnavailable
	}
	return saved, created, nil
}

func (s *Service) ListEvaluations(ctx context.Context, p domain.Principal, requestID uuid.UUID, f domain.EvalFilter) (domain.EvalPage, error) {
	if err := f.Validate(); err != nil {
		return domain.EvalPage{}, err
	}
	if err := s.evaluationAudit(ctx, p, requestID, domain.ActionEvaluationsRead, uuid.Nil); err != nil {
		return domain.EvalPage{}, err
	}
	if s.evaluations == nil {
		return domain.EvalPage{}, ErrEvaluationsUnavailable
	}
	page, err := s.evaluations.List(ctx, f)
	if err != nil {
		return domain.EvalPage{}, ErrEvaluationsUnavailable
	}
	if page.Items == nil {
		page.Items = []domain.EvalRun{}
	}
	return page, nil
}

func (s *Service) GetEvaluation(ctx context.Context, p domain.Principal, requestID, id uuid.UUID) (domain.EvalRunDetail, error) {
	if id == uuid.Nil {
		return domain.EvalRunDetail{}, domain.ErrInvalidEval
	}
	if err := s.evaluationAudit(ctx, p, requestID, domain.ActionEvaluationDetailRead, id); err != nil {
		return domain.EvalRunDetail{}, err
	}
	if s.evaluations == nil {
		return domain.EvalRunDetail{}, ErrEvaluationsUnavailable
	}
	detail, err := s.evaluations.Get(ctx, id)
	if errors.Is(err, domain.ErrEvalNotFound) {
		return domain.EvalRunDetail{}, err
	}
	if err != nil {
		return domain.EvalRunDetail{}, ErrEvaluationsUnavailable
	}
	return detail, nil
}
