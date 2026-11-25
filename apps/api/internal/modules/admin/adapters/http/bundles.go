package httpadapter

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

// Bundles reports startup-verified metadata. Client-supplied paths, names,
// version selectors and other query parameters are never accepted.
func (h *Handler) Bundles(c *gin.Context) {
	p, req, ok := verifiedAdmin(c)
	if !ok {
		return
	}
	if len(c.Request.URL.Query()) != 0 {
		httpx.Failure(c, http.StatusBadRequest, "ADMIN_BUNDLE_QUERY_INVALID")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	inventory, err := h.service.ListBundles(ctx, p, req)
	if err != nil {
		switch {
		case errors.Is(err, application.ErrUnverifiedStaff):
			httpx.Failure(c, http.StatusUnauthorized, "STAFF_SESSION_REQUIRED")
		case errors.Is(err, application.ErrAuditUnavailable):
			httpx.Failure(c, http.StatusServiceUnavailable, "ADMIN_AUDIT_UNAVAILABLE")
		default:
			httpx.Failure(c, http.StatusServiceUnavailable, "ADMIN_BUNDLES_UNAVAILABLE")
		}
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, inventory)
}
