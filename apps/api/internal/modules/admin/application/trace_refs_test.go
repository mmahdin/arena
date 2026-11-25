package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
)

type testTraceReferenceStore struct {
	audit *inMemoryAudit
	calls int
	fail  error
}

func (r *testTraceReferenceStore) ListTraceReferences(_ context.Context, f domain.TraceReferenceFilter) (domain.TraceReferencePage, error) {
	if len(r.audit.events) == 0 {
		panic("trace references read before durable audit")
	}
	r.calls++
	return domain.TraceReferencePage{TurnID: f.TurnID, Items: []domain.TraceReference{}}, r.fail
}
func TestTraceRefsRequireStaffAuditBeforeRead(t *testing.T) {
	audit := &inMemoryAudit{}
	reader := &testTraceReferenceStore{audit: audit}
	s := NewWithAudit(audit)
	s.SetTraceReferenceStore(reader)
	id := uuid.New()
	f := domain.TraceReferenceFilter{TurnID: id, Limit: 25}
	p := domain.Principal{StaffID: uuid.New(), ExpiresAt: time.Now().Add(time.Hour)}
	req := uuid.New()
	page, err := s.GetTraceReferences(context.Background(), p, req, f)
	if err != nil || page.TurnID != id || page.Items == nil || reader.calls != 1 || len(audit.events) != 1 ||
		audit.events[0].ActionCode != domain.ActionTurnTraceRefsRead || audit.events[0].ResourceID != id || audit.events[0].RequestID != req {
		t.Fatalf("bad result or audit: %+v, %v", page, err)
	}
	audit.appendErr = errors.New("SECRET audit SQL")
	if _, err := s.GetTraceReferences(context.Background(), p, req, f); !errors.Is(err, ErrAuditUnavailable) || reader.calls != 1 {
		t.Fatal("audit failure did not fail closed")
	}
	audit.appendErr = nil
	p.ExpiresAt = time.Now().Add(-time.Second)
	if _, err := s.GetTraceReferences(context.Background(), p, req, f); !errors.Is(err, ErrUnverifiedStaff) || reader.calls != 1 {
		t.Fatal("expired staff accepted")
	}
	p.ExpiresAt = time.Now().Add(time.Hour)
	f.Limit = 51
	if _, err := s.GetTraceReferences(context.Background(), p, req, f); !errors.Is(err, domain.ErrInvalidInspector) || reader.calls != 1 {
		t.Fatal("invalid page reached storage")
	}
	f.Limit = 25
	reader.fail = errors.New("SECRET SQL credential")
	if _, err := s.GetTraceReferences(context.Background(), p, req, f); !errors.Is(err, domain.ErrTraceRefUnavailable) || err.Error() != "trace references unavailable" {
		t.Fatal("raw SQL error leaked")
	}
	reader.fail = domain.ErrTurnNotFound
	if _, err := s.GetTraceReferences(context.Background(), p, req, f); !errors.Is(err, domain.ErrTurnNotFound) {
		t.Fatal("missing turn not preserved")
	}
}
