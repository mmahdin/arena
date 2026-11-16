package httpadapter

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/devsession/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/devsecurity"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

type Handler struct {
	s      *application.Service
	signer *devsecurity.Signer
}

func New(s *application.Service, signer *devsecurity.Signer) *Handler {
	return &Handler{s: s, signer: signer}
}

func (h *Handler) List(c *gin.Context) {
	items, err := h.s.List(c.Request.Context())
	if err != nil {
		httpx.Failure(c, 500, "INTERNAL_ERROR")
		return
	}
	c.JSON(200, gin.H{"items": items})
}

func (h *Handler) Login(c *gin.Context) {
	var body struct {
		UserID uuid.UUID `json:"userId"`
	}
	if !httpx.StrictJSON(c, &body) {
		return
	}
	if body.UserID == uuid.Nil {
		httpx.Failure(c, 400, "INVALID_ID")
		return
	}
	now := time.Now()
	sessionID := uuid.New()
	exists, err := h.s.StartSession(c.Request.Context(), body.UserID, sessionID, now.Add(devsecurity.SessionTTL))
	if err != nil {
		httpx.Failure(c, 500, "INTERNAL_ERROR")
		return
	}
	if !exists {
		httpx.Failure(c, 404, "RESOURCE_NOT_FOUND")
		return
	}

	// Switching testers revokes the previous session even when the old cookie was copied.
	if previous, err := c.Request.Cookie(devsecurity.CookieName); err == nil {
		if claims, err := h.signer.Verify(previous.Value, now); err == nil {
			if err := h.s.EndSession(c.Request.Context(), claims.UserID, claims.SessionID); err != nil {
				_ = h.s.EndSession(c.Request.Context(), body.UserID, sessionID)
				httpx.Failure(c, 500, "INTERNAL_ERROR")
				return
			}
		}
	}
	http.SetCookie(c.Writer, h.signer.Cookie(h.signer.Issue(body.UserID, sessionID, now)))
	c.JSON(200, gin.H{"ok": true})
}

func (h *Handler) Logout(c *gin.Context) {
	if previous, err := c.Request.Cookie(devsecurity.CookieName); err == nil {
		if claims, err := h.signer.Verify(previous.Value, time.Now()); err == nil {
			if err := h.s.EndSession(c.Request.Context(), claims.UserID, claims.SessionID); err != nil {
				httpx.Failure(c, 500, "INTERNAL_ERROR")
				return
			}
		}
	}
	http.SetCookie(c.Writer, h.signer.ExpireCookie())
	c.Status(http.StatusNoContent)
}

// IdentityLogin is a local-only synthetic find-or-create flow, NOT mobile auth.
func (h *Handler) IdentityLogin(c *gin.Context) {
	var body struct {
		Phone string `json:"phone"`
	}
	if !httpx.StrictJSON(c, &body) {
		return
	}
	now := time.Now()
	sid := uuid.New()
	uid, created, err := h.s.StartPhoneSession(c.Request.Context(), body.Phone, sid, now.Add(devsecurity.SessionTTL))
	if errors.Is(err, application.ErrInvalidPhone) {
		httpx.Failure(c, 400, "INVALID_DEV_PHONE")
		return
	}
	if err != nil {
		httpx.Failure(c, 503, "SESSION_UNAVAILABLE")
		return
	}
	if previous, err := c.Request.Cookie(devsecurity.CookieName); err == nil {
		if claims, err := h.signer.Verify(previous.Value, now); err == nil {
			if err := h.s.EndSession(c.Request.Context(), claims.UserID, claims.SessionID); err != nil {
				_ = h.s.EndSession(c.Request.Context(), uid, sid)
				httpx.Failure(c, 503, "SESSION_UNAVAILABLE")
				return
			}
		}
	}
	http.SetCookie(c.Writer, h.signer.Cookie(h.signer.Issue(uid, sid, now)))
	c.JSON(200, gin.H{"ok": true, "created": created})
}
