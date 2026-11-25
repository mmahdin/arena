package application

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
)

var ErrOverviewUnavailable = errors.New("admin overview unavailable")

// OverviewStore is a dedicated metadata-only reader, never a query interface
// that accepts arbitrary SQL or projects user or message content.
type OverviewStore interface {
	ReadOverview(context.Context, domain.OverviewFilter) (domain.Overview, error)
}

func (s *Service) ReadOverview(ctx context.Context, principal domain.Principal, requestID uuid.UUID, filter domain.OverviewFilter) (domain.Overview, error) {
	if err := filter.Validate(); err != nil {
		return domain.Overview{}, err
	}
	if err := s.RecordAuthorizedRead(ctx, principal, requestID, domain.ActionOverviewRead, uuid.Nil); err != nil {
		if errors.Is(err, ErrUnverifiedStaff) {
			return domain.Overview{}, err
		}
		return domain.Overview{}, ErrAuditUnavailable
	}
	if s.overview == nil {
		return domain.Overview{}, ErrOverviewUnavailable
	}
	report, err := s.overview.ReadOverview(ctx, filter)
	if err != nil {
		return domain.Overview{}, ErrOverviewUnavailable // never expose database errors
	}
	report.From, report.To, report.AsOf = filter.From.UTC(), filter.To.UTC(), filter.AsOf.UTC()
	report.Complete()
	return report, nil
}
