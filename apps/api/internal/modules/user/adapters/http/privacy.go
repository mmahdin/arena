package httpadapter

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/devsecurity"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

type PrivacyHandler struct{ service *application.PrivacyService }

func NewPrivacyHandler(service *application.PrivacyService) *PrivacyHandler {
	return &PrivacyHandler{service: service}
}
func privacyFailure(c *gin.Context, err error) {
	if errors.Is(err, httpx.ErrNotFound) {
		httpx.Failure(c, 404, "RESOURCE_NOT_FOUND")
		return
	}
	httpx.Failure(c, 503, "DB_UNAVAILABLE")
}
func (h *PrivacyHandler) Get(c *gin.Context) {
	value, err := h.service.Get(c.Request.Context(), devsecurity.UserID(c))
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
	value, err := h.service.Set(c.Request.Context(), devsecurity.UserID(c), *enabled)
	if err != nil {
		privacyFailure(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}
