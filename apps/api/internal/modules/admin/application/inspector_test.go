package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
)

type testInspector struct {
	calls int
	audit *inMemoryAudit
	fail  error
}

func (s *testInspector) ListTurns(_ context.Context, _ domain.InspectorFilter) (domain.TurnPage, error) {
	if len(s.audit.events) == 0 {
		panic("read before audit")
	}
	s.calls++
	return domain.TurnPage{Items: []domain.TurnRecord{{ID: uuid.New()}}}, s.fail
}
func (s *testInspector) GetTurn(_ context.Context, id uuid.UUID) (domain.TurnDetail, error) {
	if len(s.audit.events) == 0 {
		panic("read before audit")
	}
	s.calls++
	if s.fail != nil {
		return domain.TurnDetail{}, s.fail
	}
	return domain.TurnDetail{Turn: domain.TurnRecord{ID: id}}, nil
}
func (s *testInspector) ListJobs(_ context.Context, _ domain.InspectorFilter) (domain.JobPage, error) {
	if len(s.audit.events) == 0 {
		panic("read before audit")
	}
	s.calls++
	return domain.JobPage{Items: []domain.JobRecord{}}, s.fail
}
func (s *testInspector) ListWorkers(_ context.Context, at time.Time) (domain.WorkerPage, error) {
	if len(s.audit.events) == 0 {
		panic("read before audit")
	}
	s.calls++
	return domain.WorkerPage{AsOf: at, Instances: []domain.WorkerInstance{}}, s.fail
}
func TestInspectorReadsRequireAudit(t *testing.T) {
	audit := &inMemoryAudit{}
	reader := &testInspector{audit: audit}
	svc := NewWithOperationalRepositories(audit, nil, reader)
	p := domain.Principal{StaffID: uuid.New(), ExpiresAt: time.Now().Add(time.Hour)}
	req := uuid.New()
	ctx := context.Background()
	f := domain.NewInspectorFilter(time.Now())
	if _, err := svc.ListTurns(ctx, p, req, f); err != nil {
		t.Fatal(err)
	}
	if audit.events[0].ActionCode != domain.ActionTurnsRead || audit.events[0].ResourceKind != "turn" {
		t.Fatal("turn audit")
	}
	id := uuid.New()
	if _, err := svc.GetTurn(ctx, p, req, id); err != nil {
		t.Fatal(err)
	}
	if audit.events[1].ResourceID != id || audit.events[1].ActionCode != domain.ActionTurnDetailRead {
		t.Fatal("detail audit")
	}
	if _, err := svc.ListJobs(ctx, p, req, f); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ListWorkers(ctx, p, req, time.Now()); err != nil {
		t.Fatal(err)
	}
	if reader.calls != 4 || len(audit.events) != 4 || audit.events[2].ActionCode != domain.ActionJobsRead || audit.events[3].ActionCode != domain.ActionWorkersRead {
		t.Fatal("missing audit/reader")
	}
	audit.appendErr = errors.New("audit sql offline")
	if _, err := svc.ListTurns(ctx, p, req, f); !errors.Is(err, ErrAuditUnavailable) || reader.calls != 4 {
		t.Fatal("auditing did not fail closed")
	}
	audit.appendErr = nil
	p.ExpiresAt = time.Now().Add(-time.Second)
	if _, err := svc.ListTurns(ctx, p, req, f); !errors.Is(err, ErrUnverifiedStaff) || reader.calls != 4 {
		t.Fatal("expired session accepted")
	}
}
func TestInspectorErrorIsolation(t *testing.T) {
	audit := &inMemoryAudit{}
	reader := &testInspector{audit: audit, fail: errors.New("sql private connection secret")}
	svc := NewWithOperationalRepositories(audit, nil, reader)
	p := domain.Principal{StaffID: uuid.New(), ExpiresAt: time.Now().Add(time.Hour)}
	f := domain.NewInspectorFilter(time.Now())
	if _, err := svc.ListTurns(context.Background(), p, uuid.New(), f); !errors.Is(err, ErrInspectorUnavailable) || err.Error() != "admin inspector unavailable" {
		t.Fatalf("leaked SQL: %v", err)
	}
	reader.fail = domain.ErrTurnNotFound
	if _, err := svc.GetTurn(context.Background(), p, uuid.New(), uuid.New()); !errors.Is(err, domain.ErrTurnNotFound) {
		t.Fatalf("lost not found: %v", err)
	}
	f.Limit = 1000
	before := reader.calls
	if _, err := svc.ListJobs(context.Background(), p, uuid.New(), f); !errors.Is(err, domain.ErrInvalidInspector) || reader.calls != before {
		t.Fatal("invalid filter read SQL")
	}
}
