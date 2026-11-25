package application

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
)

var ErrAuditUnavailable = errors.New("admin audit repository unavailable")

// AuditStore is the only persistent audit port. It intentionally offers no
// update/delete operation or generic SQL/payload access.
type AuditStore interface {
	Append(context.Context, domain.AuditEvent) error
	List(context.Context, domain.AuditFilter) ([]domain.AuditEvent, string, error)
}

// RecordAuthorizedRead must complete BEFORE returning privileged read data.
// It fails closed when the audit repository is unavailable.
func (s *Service) RecordAuthorizedRead(ctx context.Context, principal domain.Principal, requestID uuid.UUID, action string, resourceID uuid.UUID) error {
	if _, err := s.Me(principal); err != nil {
		return err
	}
	if s.audit == nil {
		return ErrAuditUnavailable
	}
	kind, ok := domain.ResourceKindForAction(action)
	if !ok {
		return domain.ErrInvalidAudit
	}
	event := domain.AuditEvent{
		ID: uuid.New(), ActorStaffID: principal.StaffID, RequestID: requestID,
		ActionCode: action, ResourceKind: kind, ResourceID: resourceID,
		OutcomeCode: domain.OutcomeAuthorized,
	}
	if err := event.Validate(); err != nil {
		return err
	}
	return s.audit.Append(ctx, event)
}

// ReadAudit writes an authorized read event before executing the page query.
// Both read and write are scoped to verified Staff on the admin host by HTTP.
func (s *Service) ReadAudit(ctx context.Context, principal domain.Principal, requestID uuid.UUID, filter domain.AuditFilter) ([]domain.AuditEvent, string, error) {
	if err := filter.Validate(); err != nil {
		return nil, "", err
	}
	if err := s.RecordAuthorizedRead(ctx, principal, requestID, domain.ActionAuditRead, uuid.Nil); err != nil {
		return nil, "", err
	}
	return s.audit.List(ctx, filter)
}

// CurrentStaff is intentionally not stored in audit. The staff UUID comes
// from a verified and currently active SQL-backed BFF session only.
func (s *Service) MeWithAudit(ctx context.Context, principal domain.Principal, requestID uuid.UUID) (domain.CurrentStaff, error) {
	staff, err := s.Me(principal)
	if err != nil {
		return staff, err
	}
	if err := s.RecordAuthorizedRead(ctx, principal, requestID, domain.ActionAdminMeRead, uuid.Nil); err != nil {
		return domain.CurrentStaff{}, err
	}
	return staff, nil
}
