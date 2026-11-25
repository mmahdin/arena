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

type PrivacyHandler struct{ service *application.PrivacyService }

func NewPrivacyHandler(service *application.PrivacyService) *PrivacyHandler {
	return &PrivacyHandler{service: service}
}
func privacyFailure(c *gin.Context, err error) {
	if errors.Is(err, httpx.ErrNotFound) {
		httpx.FailureWithError(c, 404, "RESOURCE_NOT_FOUND", err)
		return
	}
	httpx.FailureWithError(c, 503, "DB_UNAVAILABLE", err)
}
func (h *PrivacyHandler) Get(c *gin.Context) {
	value, err := h.service.Get(c.Request.Context(), principal.UserID(c))
	if err != nil {
		privacyFailure(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}
func (h *PrivacyHandler) Set(c *gin.Context) {
	var input map[string]json.RawMessage
	if !httpx.StrictJSON(c, &input) {
		return
	}
	raw, ok := input["conversationSharingEnabled"]
	var enabled *bool
	if !ok || len(input) != 1 || json.Unmarshal(raw, &enabled) != nil || enabled == nil {
		httpx.Failure(c, 400, "VALIDATION_ERROR")
		return
	}
	value, err := h.service.Set(c.Request.Context(), principal.UserID(c), *enabled)
	if err != nil {
		privacyFailure(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}
