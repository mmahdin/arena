package httpadapter

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
)

func (h *Handler) TraceReferences(c *gin.Context) {
	p, req, ok := verifiedAdmin(c)
	if !ok {
		return
	}
	turnID, err := uuid.Parse(c.Param("turnId"))
	if err != nil || turnID == uuid.Nil {
		inspectorFailure(c, domain.ErrInvalidInspector)
		return
	}
	f := domain.TraceReferenceFilter{TurnID: turnID, Limit: domain.DefaultTraceReferencePage}
	query := c.Request.URL.Query()
	for key, values := range query {
		if (key != "pageSize" && key != "cursor") || len(values) != 1 || values[0] == "" {
			inspectorFailure(c, domain.ErrInvalidInspector)
			return
		}
	}
	if raw := query.Get("pageSize"); raw != "" {
		f.Limit, err = strconv.Atoi(raw)
		if err != nil {
			inspectorFailure(c, domain.ErrInvalidInspector)
			return
		}
	}
	if raw := query.Get("cursor"); raw != "" {
		f.Cursor, err = domain.DecodeInspectorCursor(raw)
		if err != nil {
			inspectorFailure(c, domain.ErrInvalidInspector)
			return
		}
	}
	if err = f.Validate(); err != nil {
		inspectorFailure(c, err)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	page, err := h.service.GetTraceReferences(ctx, p, req, f)
	if err != nil {
		inspectorFailure(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.JSON(http.StatusOK, page)
}
