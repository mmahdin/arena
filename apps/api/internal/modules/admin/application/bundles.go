package application

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
)

// Bundle metadata is captured at runtime startup; this read cannot invoke the
// release CLI, rescan arbitrary directories, write files, or activate bundles.
func (s *Service) ListBundles(ctx context.Context, p domain.Principal, requestID uuid.UUID) (domain.BundleInventory, error) {
	if err := s.RecordAuthorizedRead(ctx, p, requestID, domain.ActionBundlesRead, uuid.Nil); err != nil {
		if errors.Is(err, ErrUnverifiedStaff) {
			return domain.BundleInventory{}, err
		}
		return domain.BundleInventory{}, ErrAuditUnavailable
	}
	return domain.CloneBundleInventory(s.bundles), nil
}

func (s *Service) SetBundleInventory(inventory domain.BundleInventory) {
	// Only composition sets this once while assembling the Admin module.
	s.bundles = domain.CloneBundleInventory(inventory)
	if s.bundles.Scope == "" {
		s.bundles.Scope = "not_configured"
	}
	if s.bundles.AsOf.IsZero() {
		s.bundles.AsOf = time.Now().UTC()
	}
}
