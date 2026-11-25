package httpadapter

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/bookmark/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/principal"
	"net/http"
)

type Handler struct{ s *application.Service }

func New(s *application.Service) *Handler { return &Handler{s: s} }
func parse(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("messageId"))
	if err != nil || id == uuid.Nil {
		httpx.Failure(c, 400, "INVALID_ID")
		return uuid.Nil, false
	}
	return id, true
}
func failure(c *gin.Context, err error) {
	status, code := 503, "DB_UNAVAILABLE"
	switch {
	case errors.Is(err, httpx.ErrNotFound):
		status, code = 404, "RESOURCE_NOT_FOUND"
	case errors.Is(err, httpx.ErrInvalid):
		status, code = 400, "VALIDATION_ERROR"
	}
	httpx.FailureWithError(c, status, code, err)
}
func (h *Handler) List(c *gin.Context) {
	items, err := h.s.List(c.Request.Context(), principal.UserID(c))
	if err != nil {
		failure(c, err)
		return
	}
	c.JSON(200, gin.H{"items": items})
}
func (h *Handler) Save(c *gin.Context) {
	id, ok := parse(c)
	if !ok {
		return
	}
	if err := h.s.Save(c.Request.Context(), principal.UserID(c), id); err != nil {
		failure(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"saved": true})
}
func (h *Handler) Remove(c *gin.Context) {
	id, ok := parse(c)
	if !ok {
		return
	}
	if err := h.s.Remove(c.Request.Context(), principal.UserID(c), id); err != nil {
		failure(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
