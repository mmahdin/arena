package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
)

type inMemoryAudit struct {
	events    []domain.AuditEvent
	appendErr error
	listErr   error
	listCalls int
}

func (s *inMemoryAudit) Append(_ context.Context, e domain.AuditEvent) error {
	if s.appendErr != nil {
		return s.appendErr
	}
	s.events = append(s.events, e)
	return nil
}
func (s *inMemoryAudit) List(_ context.Context, _ domain.AuditFilter) ([]domain.AuditEvent, string, error) {
	s.listCalls++
	return s.events, "", s.listErr
}

func TestMeWithAuditFailsClosed(t *testing.T) {
	principal := domain.Principal{StaffID: uuid.New(), ExpiresAt: time.Now().Add(time.Hour)}
	req := uuid.New()
	ctx := context.Background()
	if _, err := New().MeWithAudit(ctx, principal, req); !errors.Is(err, ErrAuditUnavailable) {
		t.Fatalf("missing repository exposed data: %v", err)
	}
	repo := &inMemoryAudit{appendErr: errors.New("SQL down")}
	if _, err := NewWithAudit(repo).MeWithAudit(ctx, principal, req); err == nil {
		t.Fatal("write failure exposed data")
	}
	repo.appendErr = nil
	staff, err := NewWithAudit(repo).MeWithAudit(ctx, principal, req)
	if err != nil || staff.StaffID != principal.StaffID || len(repo.events) != 1 || repo.events[0].RequestID != req {
		t.Fatalf("invalid audited me: %+v %+v %v", staff, repo.events, err)
	}
	if repo.events[0].ActionCode != domain.ActionAdminMeRead || repo.events[0].ResourceID != uuid.Nil {
		t.Fatal("me audit payload differs from allowlist")
	}
}

func TestAuditReadBeforeListAndPrincipalValidation(t *testing.T) {
	repo := &inMemoryAudit{}
	svc := NewWithAudit(repo)
	p := domain.Principal{StaffID: uuid.New(), ExpiresAt: time.Now().Add(time.Hour)}
	filter := domain.NewAuditFilter(time.Now())
	if _, _, err := svc.ReadAudit(context.Background(), p, uuid.New(), filter); err != nil || len(repo.events) != 1 || repo.listCalls != 1 {
		t.Fatalf("audit read was not recorded first: %v", err)
	}
	p.ExpiresAt = time.Now().Add(-time.Minute)
	if _, _, err := svc.ReadAudit(context.Background(), p, uuid.New(), filter); !errors.Is(err, ErrUnverifiedStaff) || repo.listCalls != 1 {
		t.Fatal("expired staff read audit")
	}
	p.ExpiresAt = time.Now().Add(time.Hour)
	filter.Limit = 1001
	if _, _, err := svc.ReadAudit(context.Background(), p, uuid.New(), filter); !errors.Is(err, domain.ErrInvalidAudit) || len(repo.events) != 1 {
		t.Fatal("unbounded filter recorded or queried")
	}
}
