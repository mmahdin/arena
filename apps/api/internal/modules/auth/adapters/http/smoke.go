package httpadapter

import (
	_ "embed"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	auth "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

//go:embed smoke_page.html
var smokePage []byte

//go:embed smoke.js
var smokeScript []byte

const smokeCSP = "default-src 'none'; script-src 'self'; connect-src 'self'; style-src 'none'; img-src 'none'; object-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

func smokeHeaders(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Content-Security-Policy", smokeCSP)
	c.Header("X-Robots-Tag", "noindex, nofollow")
}

// SmokePage is explicitly opt-in, local-only, and subject to the same strict
// host/scheme boundary as a normal User or Staff session. It never embeds
// secrets or accesses local AI inspector/private data.
func (h *Handler) SmokePage(kind auth.Kind) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.settings.BrowserSmokeEnabled {
			httpx.Failure(c, http.StatusNotFound, "RESOURCE_NOT_FOUND")
			return
		}
		if !h.enabledOrFail(c, kind) {
			return
		}
		smokeHeaders(c)
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(strings.ReplaceAll(string(smokePage), "__KIND__", string(kind))))
	}
}

// SmokeJavaScript is a shared static asset served only from the configured
// User or Staff origin, never from a wildcard host.
func (h *Handler) SmokeJavaScript() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.settings.BrowserSmokeEnabled {
			httpx.Failure(c, http.StatusNotFound, "RESOURCE_NOT_FOUND")
			return
		}
		if !h.expectedRequest(c, auth.User) && !h.expectedRequest(c, auth.Staff) {
			httpx.Failure(c, http.StatusBadRequest, "AUTH_ORIGIN_MISMATCH")
			return
		}
		smokeHeaders(c)
		c.Data(http.StatusOK, "text/javascript; charset=utf-8", smokeScript)
	}
}
