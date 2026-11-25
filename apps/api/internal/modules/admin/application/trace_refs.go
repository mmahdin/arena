package application

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
)

type TraceReferenceStore interface {
	ListTraceReferences(context.Context, domain.TraceReferenceFilter) (domain.TraceReferencePage, error)
}

// Separate optional port avoids changing existing inspector test stubs.
func (s *Service) SetTraceReferenceStore(store TraceReferenceStore) { s.traceRefs = store }

func (s *Service) GetTraceReferences(ctx context.Context, p domain.Principal, requestID uuid.UUID, f domain.TraceReferenceFilter) (domain.TraceReferencePage, error) {
	if err := f.Validate(); err != nil {
		return domain.TraceReferencePage{}, err
	}
	if err := s.inspectorAudit(ctx, p, requestID, domain.ActionTurnTraceRefsRead, f.TurnID); err != nil {
		return domain.TraceReferencePage{}, err
	}
	if s.traceRefs == nil {
		return domain.TraceReferencePage{}, domain.ErrTraceRefUnavailable
	}
	page, err := s.traceRefs.ListTraceReferences(ctx, f)
	if errors.Is(err, domain.ErrTurnNotFound) {
		return domain.TraceReferencePage{}, err
	}
	if err != nil {
		return domain.TraceReferencePage{}, domain.ErrTraceRefUnavailable
	}
	if page.Items == nil {
		page.Items = []domain.TraceReference{}
	}
	return page, nil
}
