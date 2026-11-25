package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
)

type evalMemoryStore struct {
	audit         *inMemoryAudit
	entries       map[uuid.UUID]domain.EvalRunDetail
	reads         int
	errorOnImport error
}

func (s *evalMemoryStore) Import(_ context.Context, d domain.EvalRunDetail) (domain.EvalRun, bool, error) {
	if len(s.audit.events) == 0 {
		panic("import before audit")
	}
	if s.errorOnImport != nil {
		return domain.EvalRun{}, false, s.errorOnImport
	}
	if s.entries == nil {
		s.entries = map[uuid.UUID]domain.EvalRunDetail{}
	}
	for _, v := range s.entries {
		if v.Run.ReportSHA256 == d.Run.ReportSHA256 && (v.Run.BaselineRunID == nil) == (d.Run.BaselineRunID == nil) {
			if v.Run.BaselineRunID == nil || *v.Run.BaselineRunID == *d.Run.BaselineRunID {
				return v.Run, false, nil
			}
		}
	}
	s.entries[d.Run.ID] = d
	return d.Run, true, nil
}
func (s *evalMemoryStore) List(_ context.Context, _ domain.EvalFilter) (domain.EvalPage, error) {
	if len(s.audit.events) == 0 {
		panic("list before audit")
	}
	s.reads++
	items := []domain.EvalRun{}
	for _, entry := range s.entries {
		items = append(items, entry.Run)
	}
	return domain.EvalPage{Items: items}, nil
}
func (s *evalMemoryStore) Get(_ context.Context, id uuid.UUID) (domain.EvalRunDetail, error) {
	if len(s.audit.events) == 0 {
		panic("detail before audit")
	}
	s.reads++
	entry, ok := s.entries[id]
	if !ok {
		return domain.EvalRunDetail{}, domain.ErrEvalNotFound
	}
	return entry, nil
}
func exampleWP15(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("../../../../../../docs/quality/test-results/ai-core-eval.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func artifactDigest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func TestEvaluationImportAndAuditedReads(t *testing.T) {
	audit := &inMemoryAudit{}
	store := &evalMemoryStore{audit: audit}
	svc := NewWithEvaluationRepositories(audit, nil, nil, store)
	p := domain.Principal{StaffID: uuid.New(), ExpiresAt: time.Now().Add(time.Hour)}
	req := uuid.New()
	ctx := context.Background()
	raw := exampleWP15(t)
	if _, _, err := svc.ImportEvaluation(ctx, p, req, raw, "0", nil); !errors.Is(err, domain.ErrInvalidEval) || len(audit.events) != 0 {
		t.Fatal("bad hash recorded or accepted")
	}
	imported, created, err := svc.ImportEvaluation(ctx, p, req, raw, artifactDigest(raw), nil)
	if err != nil || !created || imported.Status != "passed" || !imported.ArtifactHashVerified || imported.SourceAttested || imported.ClinicallyApproved {
		t.Fatalf("bad import %+v %v", imported, err)
	}
	if len(audit.events) != 1 || audit.events[0].ActionCode != domain.ActionEvaluationImport {
		t.Fatal("import audit missing")
	}
	duplicate, created, err := svc.ImportEvaluation(ctx, p, req, raw, artifactDigest(raw), nil)
	if err != nil || created || duplicate.ID != imported.ID {
		t.Fatal("idempotency failed")
	}
	page, err := svc.ListEvaluations(ctx, p, req, domain.NewEvalFilter(time.Now()))
	if err != nil || len(page.Items) != 1 || page.Items[0].ID != imported.ID {
		t.Fatal("list", err)
	}
	detail, err := svc.GetEvaluation(ctx, p, req, imported.ID)
	if err != nil || detail.Report.Schema != "eval_report.v1" || len(detail.Report.Scenarios) != 3 {
		t.Fatal("detail", err)
	}
	if len(audit.events) != 4 || audit.events[2].ActionCode != domain.ActionEvaluationsRead || audit.events[3].ResourceID != imported.ID {
		t.Fatal("read audit incomplete")
	}
	p.ExpiresAt = time.Now().Add(-time.Minute)
	if _, err := svc.GetEvaluation(ctx, p, req, imported.ID); !errors.Is(err, ErrUnverifiedStaff) {
		t.Fatal("expired staff read")
	}
	p.ExpiresAt = time.Now().Add(time.Hour)
	audit.appendErr = errors.New("PRIVATE SQL password")
	before := store.reads
	if _, err := svc.ListEvaluations(ctx, p, req, domain.NewEvalFilter(time.Now())); !errors.Is(err, ErrAuditUnavailable) || store.reads != before {
		t.Fatal("audit did not fail closed")
	}
}
func TestEvaluationRecomputesBaselineFailures(t *testing.T) {
	audit := &inMemoryAudit{}
	store := &evalMemoryStore{audit: audit}
	svc := NewWithEvaluationRepositories(audit, nil, nil, store)
	p := domain.Principal{StaffID: uuid.New(), ExpiresAt: time.Now().Add(time.Hour)}
	ctx := context.Background()
	req := uuid.New()
	raw := exampleWP15(t)
	baseline, _, err := svc.ImportEvaluation(ctx, p, req, raw, artifactDigest(raw), nil)
	if err != nil {
		t.Fatal(err)
	}
	var r domain.EvalReport
	if json.Unmarshal(raw, &r) != nil {
		t.Fatal("repo fixture invalid")
	}
	r.Passed = false
	r.Scenarios[0].Passed = false
	r.Scenarios[0].Turns[0].Passed = false
	r.Scenarios[0].Turns[0].Diagnostics = []domain.EvalDiagnostic{{Code: "EVAL_TEST_FAILURE", Assertion: "route", Severity: "failure"}}
	r.Totals.PassedScenarios--
	r.Totals.FailedScenarios++
	r.Totals.PassedTurns--
	r.Totals.Failures++
	newer, _ := json.Marshal(r)
	imported, _, err := svc.ImportEvaluation(ctx, p, req, newer, artifactDigest(newer), &baseline.ID)
	if err != nil {
		t.Fatal(err)
	}
	if imported.Status != "failed" || imported.RegressionCount == 0 || imported.BaselineRunID == nil || *imported.BaselineRunID != baseline.ID {
		t.Fatalf("no regression %+v", imported)
	}
	if _, _, err = svc.ImportEvaluation(ctx, p, req, newer, artifactDigest(newer), func() *uuid.UUID { id := uuid.New(); return &id }()); !errors.Is(err, domain.ErrEvalNotFound) {
		t.Fatalf("missing baseline accepted %v", err)
	}
}
