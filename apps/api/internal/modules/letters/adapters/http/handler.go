package httpadapter

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/letters/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/principal"
)

type Handler struct{ service *application.Service }

func New(service *application.Service) *Handler { return &Handler{service: service} }
func fail(c *gin.Context, err error) {
	status, code := http.StatusServiceUnavailable, "DB_UNAVAILABLE"
	switch {
	case errors.Is(err, application.ErrInsufficientContext):
		status, code = http.StatusConflict, "INSUFFICIENT_CONVERSATION_CONTEXT"
	case errors.Is(err, application.ErrGenerationUnavailable):
		status, code = http.StatusServiceUnavailable, "LETTER_GENERATION_UNAVAILABLE"
	case errors.Is(err, httpx.ErrInvalid):
		status, code = http.StatusBadRequest, "VALIDATION_ERROR"
	case errors.Is(err, httpx.ErrNotFound):
		status, code = http.StatusNotFound, "RESOURCE_NOT_FOUND"
	}
	httpx.FailureWithError(c, status, code, err)
}
func (h *Handler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context(), principal.UserID(c))
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
	item, err := h.service.Generate(c.Request.Context(), principal.UserID(c), *body.Consent)
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
	if err = h.service.Delete(c.Request.Context(), principal.UserID(c), id); err != nil {
		fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
