package httpadapter

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/conversation/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/devsecurity"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
	"net/http"
)

type Handler struct{ s *application.Service }

func New(s *application.Service) *Handler { return &Handler{s: s} }
func (h *Handler) List(c *gin.Context) {
	data, err := h.s.List(c.Request.Context(), devsecurity.UserID(c))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(200, gin.H{"items": data})
}
func (h *Handler) Create(c *gin.Context) {
	data, err := h.s.Create(c.Request.Context(), devsecurity.UserID(c))
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(201, data)
}
func parseID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil || id == uuid.Nil {
		httpx.Failure(c, 400, "INVALID_ID")
		return uuid.Nil, false
	}
	return id, true
}
func (h *Handler) Messages(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	data, err := h.s.Messages(c.Request.Context(), devsecurity.UserID(c), id)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(200, gin.H{"items": data})
}
func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.s.Delete(c.Request.Context(), devsecurity.UserID(c), id); err != nil {
		fail(c, err)
		return
	}
	c.Status(204)
}
func (h *Handler) Rename(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var body struct {
		Title string `json:"title"`
	}
	if !httpx.StrictJSON(c, &body) {
		return
	}
	data, err := h.s.Rename(c.Request.Context(), devsecurity.UserID(c), id, body.Title)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}
func (h *Handler) Turn(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var body struct {
		ClientTurnID uuid.UUID `json:"clientTurnId"`
		Message      string    `json:"message"`
	}
	if !httpx.StrictJSON(c, &body) {
		return
	}
	data, err := h.s.Turn(c.Request.Context(), devsecurity.UserID(c), id, body.ClientTurnID, body.Message)
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}
func fail(c *gin.Context, err error) {
	status, code := httpx.ErrStatus(err)
	httpx.Failure(c, status, code)
}
