// Package application contains Back-Office use cases. It does not implement
// OIDC, sessions, Keycloak account management or product-user impersonation.
package application

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
)

var ErrUnverifiedStaff = errors.New("verified active staff session required")

type Service struct {
	audit       AuditStore
	overview    OverviewStore
	inspector   InspectorStore
	evaluations EvaluationStore
	traceRefs   TraceReferenceStore
	bundles     domain.BundleInventory
}

func New() *Service { return &Service{} }

// NewWithAudit wires a durable audit store for the running Admin module.
func NewWithAudit(store AuditStore) *Service { return &Service{audit: store} }

// Me projects only the minimal safe metadata from a verified Staff session.
// The SQL-backed RequireStaff guard is responsible for checking whether the
// staff account remains active and the session is not revoked or expired.
func (s *Service) Me(principal domain.Principal) (domain.CurrentStaff, error) {
	if principal.StaffID == uuid.Nil || principal.ExpiresAt.IsZero() || !principal.ExpiresAt.After(time.Now()) {
		return domain.CurrentStaff{}, ErrUnverifiedStaff
	}
	return domain.CurrentStaff{
		StaffID:   principal.StaffID,
		Role:      domain.AdminRole,
		Active:    true,
		ExpiresAt: principal.ExpiresAt.UTC(),
	}, nil
}

// NewWithRepositories shares the existing mandatory admin audit boundary.
func NewWithRepositories(audit AuditStore, overview OverviewStore) *Service {
	return &Service{audit: audit, overview: overview}
}
