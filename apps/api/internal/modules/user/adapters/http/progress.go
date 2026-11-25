package httpadapter

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/principal"
)

type ProgressHandler struct{ service *application.ProgressService }

func NewProgressHandler(s *application.ProgressService) *ProgressHandler {
	return &ProgressHandler{service: s}
}
func progressError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, application.ErrInvalidProgress):
		httpx.Failure(c, 400, "INVALID_ONBOARDING_STEP")
	case errors.Is(err, application.ErrProgressConflict):
		httpx.Failure(c, 409, "ONBOARDING_STEP_CONFLICT")
	case errors.Is(err, application.ErrProgressCompleted):
		httpx.Failure(c, 409, "ONBOARDING_ALREADY_COMPLETED")
	case errors.Is(err, httpx.ErrNotFound):
		httpx.Failure(c, 404, "RESOURCE_NOT_FOUND")
	default:
		httpx.Failure(c, 503, "DB_UNAVAILABLE")
	}
}
func (h *ProgressHandler) Get(c *gin.Context) {
	data, err := h.service.Get(c.Request.Context(), principal.UserID(c))
	if err != nil {
		progressError(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}
func (h *ProgressHandler) Advance(c *gin.Context) {
	// Reject every extra property, especially questionnaire answers. This endpoint
	// has a deliberately tiny privacy-preserving contract.
	var body map[string]json.RawMessage
	if !httpx.StrictJSON(c, &body) {
		return
	}
	raw, ok := body["furthestStep"]
	if !ok || len(body) != 1 {
		httpx.Failure(c, 400, "INVALID_ONBOARDING_STEP")
		return
	}
	var step *int
	if json.Unmarshal(raw, &step) != nil || step == nil {
		httpx.Failure(c, 400, "INVALID_ONBOARDING_STEP")
		return
	}
	data, err := h.service.Advance(c.Request.Context(), principal.UserID(c), *step)
	if err != nil {
		progressError(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}
