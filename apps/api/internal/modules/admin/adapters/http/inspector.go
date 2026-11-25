package httpadapter

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
	authhttp "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth/adapters/http"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

// parseInspectorFilter rejects unknown/duplicate params, unbounded windows,
// noncanonical cursors and invalid status/kind before any audit or SQL work.
func parseInspectorFilter(c *gin.Context, now time.Time, allowKind bool) (domain.InspectorFilter, error) {
	f := domain.NewInspectorFilter(now)
	query := c.Request.URL.Query()
	for key, vals := range query {
		switch key {
		case "from", "to", "cursor", "pageSize", "status":
		case "kind":
			if !allowKind {
				return f, domain.ErrInvalidInspector
			}
		default:
			return f, domain.ErrInvalidInspector
		}
		if len(vals) != 1 || vals[0] == "" {
			return f, domain.ErrInvalidInspector
		}
	}
	from, hasFrom := query["from"]
	to, hasTo := query["to"]
	if hasFrom != hasTo {
		return f, domain.ErrInvalidInspector
	}
	if hasFrom {
		var err error
		f.From, err = time.Parse(time.RFC3339, from[0])
		if err != nil {
			return f, domain.ErrInvalidInspector
		}
		f.To, err = time.Parse(time.RFC3339, to[0])
		if err != nil {
			return f, domain.ErrInvalidInspector
		}
	}
	if f.To.After(now.Add(time.Second)) {
		return f, domain.ErrInvalidInspector
	}
	if raw := query.Get("status"); raw != "" {
		f.Status = raw
	}
	if allowKind {
		f.Kind = query.Get("kind")
	}
	if raw := query.Get("pageSize"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return f, domain.ErrInvalidInspector
		}
		f.Limit = n
	}
	if raw := query.Get("cursor"); raw != "" {
		cursor, err := domain.DecodeInspectorCursor(raw)
		if err != nil {
			return f, err
		}
		f.Cursor = cursor
	}
	return f, f.Validate()
}
func verifiedAdmin(c *gin.Context) (domain.Principal, uuid.UUID, bool) {
	verified, ok := authhttp.VerifiedStaff(c)
	if !ok {
		httpx.Failure(c, http.StatusUnauthorized, "STAFF_SESSION_REQUIRED")
		return domain.Principal{}, uuid.Nil, false
	}
	requestID, err := uuid.Parse(c.GetString("requestId"))
	if err != nil {
		httpx.Failure(c, http.StatusServiceUnavailable, "ADMIN_AUDIT_UNAVAILABLE")
		return domain.Principal{}, uuid.Nil, false
	}
	return domain.Principal{StaffID: verified.ID, ExpiresAt: verified.ExpiresAt}, requestID, true
}
func inspectorFailure(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInspector):
		httpx.Failure(c, http.StatusBadRequest, "ADMIN_INSPECTOR_FILTER_INVALID")
	case errors.Is(err, domain.ErrTraceRefUnavailable):
		httpx.Failure(c, http.StatusServiceUnavailable, "ADMIN_TRACE_REFS_UNAVAILABLE")
	case errors.Is(err, domain.ErrTurnNotFound):
		httpx.Failure(c, http.StatusNotFound, "ADMIN_TURN_NOT_FOUND")
	case errors.Is(err, application.ErrUnverifiedStaff):
		httpx.Failure(c, http.StatusUnauthorized, "STAFF_SESSION_REQUIRED")
	case errors.Is(err, application.ErrAuditUnavailable):
		httpx.Failure(c, http.StatusServiceUnavailable, "ADMIN_AUDIT_UNAVAILABLE")
	default:
		httpx.Failure(c, http.StatusServiceUnavailable, "ADMIN_INSPECTOR_UNAVAILABLE")
	}
}
func (h *Handler) Turns(c *gin.Context) {
	p, req, ok := verifiedAdmin(c)
	if !ok {
		return
	}
	f, err := parseInspectorFilter(c, time.Now().UTC(), false)
	if err != nil {
		inspectorFailure(c, err)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	page, err := h.service.ListTurns(ctx, p, req, f)
	if err != nil {
		inspectorFailure(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, page)
}
func (h *Handler) TurnDetail(c *gin.Context) {
	p, req, ok := verifiedAdmin(c)
	if !ok {
		return
	}
	if len(c.Request.URL.Query()) != 0 {
		inspectorFailure(c, domain.ErrInvalidInspector)
		return
	}
	id, err := uuid.Parse(c.Param("turnId"))
	if err != nil || id == uuid.Nil {
		inspectorFailure(c, domain.ErrInvalidInspector)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	detail, err := h.service.GetTurn(ctx, p, req, id)
	if err != nil {
		inspectorFailure(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, detail)
}
func (h *Handler) Jobs(c *gin.Context) {
	p, req, ok := verifiedAdmin(c)
	if !ok {
		return
	}
	f, err := parseInspectorFilter(c, time.Now().UTC(), true)
	if err != nil {
		inspectorFailure(c, err)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	page, err := h.service.ListJobs(ctx, p, req, f)
	if err != nil {
		inspectorFailure(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, page)
}
func (h *Handler) Workers(c *gin.Context) {
	p, req, ok := verifiedAdmin(c)
	if !ok {
		return
	}
	if len(c.Request.URL.Query()) != 0 {
		inspectorFailure(c, domain.ErrInvalidInspector)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	page, err := h.service.ListWorkers(ctx, p, req, time.Now().UTC())
	if err != nil {
		inspectorFailure(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, page)
}
