package application

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
)

var ErrInspectorUnavailable = errors.New("admin inspector unavailable")

// InspectorStore exposes only pre-approved, bounded operational projections.
// No arbitrary SQL, transcript, job payload, local inspection file or user
// impersonation capability belongs in this interface.
type InspectorStore interface {
	ListTurns(context.Context, domain.InspectorFilter) (domain.TurnPage, error)
	GetTurn(context.Context, uuid.UUID) (domain.TurnDetail, error)
	ListJobs(context.Context, domain.InspectorFilter) (domain.JobPage, error)
	ListWorkers(context.Context, time.Time) (domain.WorkerPage, error)
}

func NewWithOperationalRepositories(audit AuditStore, overview OverviewStore, inspector InspectorStore) *Service {
	return &Service{audit: audit, overview: overview, inspector: inspector}
}

// inspectorAudit maps storage failures to the same stable fail-closed code as
// the existing Overview endpoint; raw database errors never reach HTTP.
func (s *Service) inspectorAudit(ctx context.Context, p domain.Principal, requestID uuid.UUID, action string, id uuid.UUID) error {
	if err := s.RecordAuthorizedRead(ctx, p, requestID, action, id); err != nil {
		if errors.Is(err, ErrUnverifiedStaff) {
			return err
		}
		return ErrAuditUnavailable
	}
	return nil
}

func (s *Service) ListTurns(ctx context.Context, p domain.Principal, requestID uuid.UUID, f domain.InspectorFilter) (domain.TurnPage, error) {
	if err := f.Validate(); err != nil {
		return domain.TurnPage{}, err
	}
	if f.Kind != "" {
		return domain.TurnPage{}, domain.ErrInvalidInspector
	}
	if err := s.inspectorAudit(ctx, p, requestID, domain.ActionTurnsRead, uuid.Nil); err != nil {
		return domain.TurnPage{}, err
	}
	if s.inspector == nil {
		return domain.TurnPage{}, ErrInspectorUnavailable
	}
	page, err := s.inspector.ListTurns(ctx, f)
	if err != nil {
		return domain.TurnPage{}, ErrInspectorUnavailable
	}
	if page.Items == nil {
		page.Items = []domain.TurnRecord{}
	}
	return page, nil
}

func (s *Service) GetTurn(ctx context.Context, p domain.Principal, requestID, id uuid.UUID) (domain.TurnDetail, error) {
	if id == uuid.Nil {
		return domain.TurnDetail{}, domain.ErrInvalidInspector
	}
	if err := s.inspectorAudit(ctx, p, requestID, domain.ActionTurnDetailRead, id); err != nil {
		return domain.TurnDetail{}, err
	}
	if s.inspector == nil {
		return domain.TurnDetail{}, ErrInspectorUnavailable
	}
	detail, err := s.inspector.GetTurn(ctx, id)
	if errors.Is(err, domain.ErrTurnNotFound) {
		return domain.TurnDetail{}, err
	}
	if err != nil {
		return domain.TurnDetail{}, ErrInspectorUnavailable
	}
	return detail, nil
}

func (s *Service) ListJobs(ctx context.Context, p domain.Principal, requestID uuid.UUID, f domain.InspectorFilter) (domain.JobPage, error) {
	if err := f.Validate(); err != nil {
		return domain.JobPage{}, err
	}
	if err := s.inspectorAudit(ctx, p, requestID, domain.ActionJobsRead, uuid.Nil); err != nil {
		return domain.JobPage{}, err
	}
	if s.inspector == nil {
		return domain.JobPage{}, ErrInspectorUnavailable
	}
	page, err := s.inspector.ListJobs(ctx, f)
	if err != nil {
		return domain.JobPage{}, ErrInspectorUnavailable
	}
	if page.Items == nil {
		page.Items = []domain.JobRecord{}
	}
	return page, nil
}

func (s *Service) ListWorkers(ctx context.Context, p domain.Principal, requestID uuid.UUID, asOf time.Time) (domain.WorkerPage, error) {
	if err := s.inspectorAudit(ctx, p, requestID, domain.ActionWorkersRead, uuid.Nil); err != nil {
		return domain.WorkerPage{}, err
	}
	if s.inspector == nil {
		return domain.WorkerPage{}, ErrInspectorUnavailable
	}
	page, err := s.inspector.ListWorkers(ctx, asOf)
	if err != nil {
		return domain.WorkerPage{}, ErrInspectorUnavailable
	}
	return page, nil
}
