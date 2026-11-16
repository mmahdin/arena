package httpadapter

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/letters/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/devsecurity"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

type Handler struct{ service *application.Service }

func New(service *application.Service) *Handler { return &Handler{service: service} }
func fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, application.ErrInsufficientContext):
		httpx.Failure(c, http.StatusConflict, "INSUFFICIENT_CONVERSATION_CONTEXT")
	case errors.Is(err, application.ErrGenerationUnavailable):
		httpx.Failure(c, http.StatusServiceUnavailable, "LETTER_GENERATION_UNAVAILABLE")
	case errors.Is(err, httpx.ErrInvalid):
		httpx.Failure(c, http.StatusBadRequest, "VALIDATION_ERROR")
	case errors.Is(err, httpx.ErrNotFound):
		httpx.Failure(c, http.StatusNotFound, "RESOURCE_NOT_FOUND")
	default:
		httpx.Failure(c, http.StatusServiceUnavailable, "DB_UNAVAILABLE")
	}
}
func (h *Handler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context(), devsecurity.UserID(c))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}
func (h *Handler) Generate(c *gin.Context) {
	var body struct {
		Consent *bool `json:"consent"`
	}
	if !httpx.StrictJSON(c, &body) {
		return
	}
	if body.Consent == nil || !*body.Consent {
		httpx.Failure(c, http.StatusBadRequest, "CONSENT_REQUIRED")
		return
	}
	item, err := h.service.Generate(c.Request.Context(), devsecurity.UserID(c), *body.Consent)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}
func (h *Handler) Delete(c *gin.Context) {
	raw := strings.TrimSpace(c.Param("id"))
	id, err := uuid.Parse(raw)
	if err != nil || id == uuid.Nil {
		httpx.Failure(c, http.StatusBadRequest, "VALIDATION_ERROR")
		return
	}
	if err = h.service.Delete(c.Request.Context(), devsecurity.UserID(c), id); err != nil {
		fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
