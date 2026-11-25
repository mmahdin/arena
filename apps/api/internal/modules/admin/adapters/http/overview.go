package httpadapter

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
	authhttp "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth/adapters/http"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

// parseOverviewWindow forbids unknown/duplicate/partial query parameters.
// Both timestamps must be RFC3339 with timezone; the upper bound is exclusive.
func parseOverviewWindow(c *gin.Context, now time.Time) (domain.OverviewFilter, error) {
	filter := domain.NewOverviewFilter(now)
	query := c.Request.URL.Query()
	for key, values := range query {
		if key != "from" && key != "to" || len(values) != 1 || values[0] == "" {
			return filter, domain.ErrInvalidOverview
		}
	}
	from, hasFrom := query["from"]
	to, hasTo := query["to"]
	if hasFrom != hasTo {
		return filter, domain.ErrInvalidOverview
	}
	if hasFrom {
		var err error
		filter.From, err = time.Parse(time.RFC3339, from[0])
		if err != nil {
			return filter, domain.ErrInvalidOverview
		}
		filter.To, err = time.Parse(time.RFC3339, to[0])
		if err != nil {
			return filter, domain.ErrInvalidOverview
		}
	}
	return filter, filter.Validate()
}

func (h *Handler) Overview(c *gin.Context) {
	verified, ok := authhttp.VerifiedStaff(c)
	if !ok {
		httpx.Failure(c, http.StatusUnauthorized, "STAFF_SESSION_REQUIRED")
		return
	}
	filter, err := parseOverviewWindow(c, time.Now().UTC())
	if err != nil {
		httpx.Failure(c, http.StatusBadRequest, "OVERVIEW_FILTER_INVALID")
		return
	}
	requestID, err := uuid.Parse(c.GetString("requestId"))
	if err != nil {
		httpx.Failure(c, http.StatusServiceUnavailable, "ADMIN_AUDIT_UNAVAILABLE")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	report, err := h.service.ReadOverview(ctx, domain.Principal{StaffID: verified.ID, ExpiresAt: verified.ExpiresAt}, requestID, filter)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidOverview):
			httpx.Failure(c, http.StatusBadRequest, "OVERVIEW_FILTER_INVALID")
		case errors.Is(err, application.ErrUnverifiedStaff):
			httpx.Failure(c, http.StatusUnauthorized, "STAFF_SESSION_REQUIRED")
		case errors.Is(err, application.ErrAuditUnavailable):
			httpx.Failure(c, http.StatusServiceUnavailable, "ADMIN_AUDIT_UNAVAILABLE")
		default:
			httpx.Failure(c, http.StatusServiceUnavailable, "ADMIN_OVERVIEW_UNAVAILABLE")
		}
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, report)
}
