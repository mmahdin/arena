package httpadapter

import (
	"github.com/google/uuid"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
	authhttp "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth/adapters/http"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

type Handler struct{ service *application.Service }

func New(service *application.Service) *Handler { return &Handler{service: service} }

type meResponse struct {
	StaffID   string `json:"staffId"`
	Role      string `json:"role"`
	Active    bool   `json:"active"`
	ExpiresAt string `json:"expiresAt"`
}

// Me MUST be mounted only under RequireStaff. Even when accidentally mounted
// without the guard, it refuses to trust user-supplied identity information.
func (h *Handler) Me(c *gin.Context) {
	verified, ok := authhttp.VerifiedStaff(c)
	if !ok {
		httpx.Failure(c, http.StatusUnauthorized, "STAFF_SESSION_REQUIRED")
		return
	}
	requestID, err := uuid.Parse(c.GetString("requestId"))
	if err != nil {
		httpx.Failure(c, http.StatusServiceUnavailable, "ADMIN_AUDIT_UNAVAILABLE")
		return
	}
	staff, err := h.service.MeWithAudit(c.Request.Context(), domain.Principal{StaffID: verified.ID, ExpiresAt: verified.ExpiresAt}, requestID)
	if err != nil {
		if err == application.ErrUnverifiedStaff {
			httpx.Failure(c, http.StatusUnauthorized, "STAFF_SESSION_REQUIRED")
		} else {
			httpx.Failure(c, http.StatusServiceUnavailable, "ADMIN_AUDIT_UNAVAILABLE")
		}
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, meResponse{
		StaffID: staff.StaffID.String(), Role: staff.Role, Active: staff.Active,
		ExpiresAt: staff.ExpiresAt.Format(time.RFC3339),
	})
}
