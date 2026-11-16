package httpadapter

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/reflection/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/devsecurity"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

type Handler struct{ service *application.Service }

func New(service *application.Service) *Handler { return &Handler{service: service} }
func fail(c *gin.Context, err error) {
	switch {
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
func (h *Handler) Get(c *gin.Context) {
	item, err := h.service.Get(c.Request.Context(), devsecurity.UserID(c), c.Param("date"))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}
func (h *Handler) Save(c *gin.Context) {
	var body struct {
		Text *string `json:"text"`
	}
	if !httpx.StrictJSON(c, &body) {
		return
	}
	if body.Text == nil {
		httpx.Failure(c, http.StatusBadRequest, "VALIDATION_ERROR")
		return
	}
	item, err := h.service.Save(c.Request.Context(), devsecurity.UserID(c), c.Param("date"), *body.Text)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}
func (h *Handler) Delete(c *gin.Context) {
	err := h.service.Delete(c.Request.Context(), devsecurity.UserID(c), c.Param("date"))
	if err != nil {
		fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
