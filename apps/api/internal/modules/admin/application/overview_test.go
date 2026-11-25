package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
)

type fakeOverview struct {
	calls  int
	report domain.Overview
	err    error
}

func (f *fakeOverview) ReadOverview(_ context.Context, _ domain.OverviewFilter) (domain.Overview, error) {
	f.calls++
	return f.report, f.err
}

func TestReadOverviewAuditsBeforeQuery(t *testing.T) {
	audit := &inMemoryAudit{}
	store := &fakeOverview{report: domain.Overview{Turns: domain.TurnSummary{Completed: 2, Failed: 1}}}
	svc := NewWithRepositories(audit, store)
	principal := domain.Principal{StaffID: uuid.New(), ExpiresAt: time.Now().Add(time.Hour)}
	id := uuid.New()
	f := domain.NewOverviewFilter(time.Now())
	report, err := svc.ReadOverview(context.Background(), principal, id, f)
	if err != nil || store.calls != 1 || report.Turns.FailureRate == nil || len(audit.events) != 1 || audit.events[0].ActionCode != domain.ActionOverviewRead || audit.events[0].RequestID != id {
		t.Fatalf("invalid overview read: %+v %v", report, err)
	}
	expired := principal
	expired.ExpiresAt = time.Now().Add(-time.Minute)
	if _, err := svc.ReadOverview(context.Background(), expired, id, f); !errors.Is(err, ErrUnverifiedStaff) || store.calls != 1 || len(audit.events) != 1 {
		t.Fatal("expired staff queried overview")
	}
	audit.appendErr = errors.New("database down")
	if _, err := svc.ReadOverview(context.Background(), principal, id, f); err == nil || store.calls != 1 {
		t.Fatal("audit write failure did not fail closed")
	}
	audit.appendErr = nil
	store.err = errors.New("PRIVATE SQL DETAIL")
	if _, err := svc.ReadOverview(context.Background(), principal, id, f); !errors.Is(err, ErrOverviewUnavailable) || err.Error() == store.err.Error() {
		t.Fatalf("SQL detail leak: %v", err)
	}
	f.From = f.From.Add(-32 * 24 * time.Hour)
	if _, err := svc.ReadOverview(context.Background(), principal, id, f); !errors.Is(err, domain.ErrInvalidOverview) {
		t.Fatal("unbounded request accepted")
	}
}
