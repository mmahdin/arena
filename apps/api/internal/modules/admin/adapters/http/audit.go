package httpadapter

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
	authhttp "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth/adapters/http"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

var auditQueryKeys = map[string]bool{
	"from": true, "to": true, "cursor": true, "pageSize": true,
	"actorStaffId": true, "actionCode": true,
}

// parseAuditFilter rejects unbounded, repeated and unexpected query parameters;
// SQL only ever sees typed/validated filter values.
func parseAuditFilter(c *gin.Context, now time.Time) (domain.AuditFilter, error) {
	filter := domain.NewAuditFilter(now)
	query := c.Request.URL.Query()
	for key, values := range query {
		if !auditQueryKeys[key] || len(values) != 1 {
			return filter, domain.ErrInvalidAudit
		}
	}
	from, hasFrom := query["from"]
	to, hasTo := query["to"]
	if hasFrom != hasTo {
		return filter, domain.ErrInvalidAudit
	}
	if hasFrom {
		var err error
		filter.From, err = time.Parse(time.RFC3339, from[0])
		if err != nil {
			return filter, domain.ErrInvalidAudit
		}
		filter.To, err = time.Parse(time.RFC3339, to[0])
		if err != nil {
			return filter, domain.ErrInvalidAudit
		}
	}
	if raw := query.Get("pageSize"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return filter, domain.ErrInvalidAudit
		}
		filter.Limit = n
	} else if _, exists := query["pageSize"]; exists {
		return filter, domain.ErrInvalidAudit
	}
	if raw := query.Get("actorStaffId"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil || id == uuid.Nil {
			return filter, domain.ErrInvalidAudit
		}
		filter.ActorStaffID = id
	} else if _, exists := query["actorStaffId"]; exists {
		return filter, domain.ErrInvalidAudit
	}
	if raw := query.Get("actionCode"); raw != "" {
		filter.ActionCode = raw
	} else if _, exists := query["actionCode"]; exists {
		return filter, domain.ErrInvalidAudit
	}
	if raw := query.Get("cursor"); raw != "" {
		cursor, err := domain.DecodeAuditCursor(raw)
		if err != nil {
			return filter, err
		}
		filter.Cursor = cursor
	} else if _, exists := query["cursor"]; exists {
		return filter, domain.ErrInvalidAudit
	}
	return filter, filter.Validate()
}

type auditItem struct {
	ID           string  `json:"id"`
	ActorStaffID string  `json:"actorStaffId"`
	ActionCode   string  `json:"actionCode"`
	ResourceKind string  `json:"resourceKind"`
	ResourceID   *string `json:"resourceId,omitempty"`
	OutcomeCode  string  `json:"outcomeCode"`
	RequestID    string  `json:"requestId"`
	RecordedAt   string  `json:"recordedAt"`
}

type auditResponse struct {
	Items      []auditItem `json:"items"`
	NextCursor *string     `json:"nextCursor"`
}

func (h *Handler) Audit(c *gin.Context) {
	verified, ok := authhttp.VerifiedStaff(c)
	if !ok {
		httpx.Failure(c, http.StatusUnauthorized, "STAFF_SESSION_REQUIRED")
		return
	}
	filter, err := parseAuditFilter(c, time.Now())
	if err != nil {
		httpx.Failure(c, http.StatusBadRequest, "AUDIT_FILTER_INVALID")
		return
	}
	requestID, err := uuid.Parse(c.GetString("requestId"))
	if err != nil {
		httpx.Failure(c, http.StatusServiceUnavailable, "ADMIN_AUDIT_UNAVAILABLE")
		return
	}
	rows, next, err := h.service.ReadAudit(c.Request.Context(), domain.Principal{StaffID: verified.ID, ExpiresAt: verified.ExpiresAt}, requestID, filter)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidAudit) {
			httpx.Failure(c, http.StatusBadRequest, "AUDIT_FILTER_INVALID")
		} else {
			httpx.Failure(c, http.StatusServiceUnavailable, "ADMIN_AUDIT_UNAVAILABLE")
		}
		return
	}
	items := make([]auditItem, 0, len(rows))
	for _, e := range rows {
		var resourceID *string
		if e.ResourceID != uuid.Nil {
			id := e.ResourceID.String()
			resourceID = &id
		}
		items = append(items, auditItem{ID: e.ID.String(), ActorStaffID: e.ActorStaffID.String(),
			ActionCode: e.ActionCode, ResourceKind: e.ResourceKind, ResourceID: resourceID,
			OutcomeCode: e.OutcomeCode, RequestID: e.RequestID.String(),
			RecordedAt: e.RecordedAt.UTC().Format(time.RFC3339Nano)})
	}
	var nextCursor *string
	if next != "" {
		nextCursor = &next
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, auditResponse{Items: items, NextCursor: nextCursor})
}
