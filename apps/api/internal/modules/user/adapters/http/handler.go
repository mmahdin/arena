package httpadapter

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/domain"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/principal"
	"net/http"
)

type Handler struct{ s *application.Service }

func New(s *application.Service) *Handler { return &Handler{s: s} }
func (h *Handler) Get(c *gin.Context) {
	p, err := h.s.Get(c.Request.Context(), principal.UserID(c))
	if err != nil {
		status, code := httpx.ErrStatus(err)
		httpx.FailureWithError(c, status, code, err)
		return
	}
	c.JSON(http.StatusOK, p)
}
func (h *Handler) Update(c *gin.Context) {
	var body struct {
		DisplayName       string                    `json:"displayName"`
		ConversationStyle domain.Style              `json:"conversationStyle"`
		OnboardingAnswers *domain.OnboardingAnswers `json:"onboardingAnswers"`
	}
	if !httpx.StrictJSON(c, &body) {
		return
	}
	p, err := h.s.UpdateWithOnboarding(c.Request.Context(), principal.UserID(c), body.DisplayName, body.ConversationStyle, body.OnboardingAnswers)
	if err != nil {
		if errors.Is(err, httpx.ErrNotFound) {
			httpx.FailureWithError(c, 404, "RESOURCE_NOT_FOUND", err)
		} else if errors.Is(err, domain.ErrInvalidConversationStyle) || errors.Is(err, domain.ErrInvalidDisplayName) || errors.Is(err, domain.ErrInvalidOnboardingAnswers) {
			httpx.FailureWithError(c, 400, "VALIDATION_ERROR", err)
		} else {
			httpx.FailureWithError(c, 500, "INTERNAL_ERROR", err)
		}
		return
	}
	c.JSON(http.StatusOK, p)
}

// SaveOnboardingName stores the selected name on screen four without ending onboarding.
func (h *Handler) SaveOnboardingName(c *gin.Context) {
	var body struct {
		DisplayName string `json:"displayName"`
	}
	if !httpx.StrictJSON(c, &body) {
		return
	}
	p, err := h.s.SaveOnboardingName(c.Request.Context(), principal.UserID(c), body.DisplayName)
	if err != nil {
		switch {
		case errors.Is(err, httpx.ErrNotFound):
			httpx.FailureWithError(c, http.StatusNotFound, "RESOURCE_NOT_FOUND", err)
		case errors.Is(err, domain.ErrInvalidDisplayName):
			httpx.FailureWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", err)
		case errors.Is(err, domain.ErrOnboardingAlreadyComplete):
			httpx.FailureWithError(c, http.StatusConflict, "ONBOARDING_ALREADY_COMPLETED", err)
		default:
			httpx.FailureWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err)
		}
		return
	}
	c.JSON(http.StatusOK, p)
}
