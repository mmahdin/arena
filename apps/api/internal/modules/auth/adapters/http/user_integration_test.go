package httpadapter

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/principal"
)

// AUTH-06 exercises the real HTTP session/guard path using an in-memory Store
// and test-only OIDC provider. The production issuer cryptographic proof is
// exercised separately in oidc/client_test.go.
func TestAUTH06UserSessionOwnsProductPrincipalAndRequiresCSRF(t *testing.T) {
	router, handler := setupHandler(t)
	var expectedUser uuid.UUID
	router.GET("/api/v1/me/owner-check", handler.RequireUser(), func(c *gin.Context) {
		id := principal.UserID(c)
		if id == uuid.Nil || (expectedUser != uuid.Nil && id != expectedUser) {
			c.Status(http.StatusInternalServerError)
			return
		}
		expectedUser = id
		c.JSON(http.StatusOK, gin.H{"userId": id.String()})
	})
	router.POST("/api/v1/me/owner-check", handler.RequireUser(), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	start := request(router, http.MethodGet, "/auth/user/login", "app.localhost:8080")
	if start.Code != http.StatusFound {
		t.Fatalf("login: %d", start.Code)
	}
	login := start.Result().Cookies()[0]
	callback := request(router, http.MethodGet, "/auth/user/callback?state="+url.QueryEscape(login.Value)+"&code=valid", "app.localhost:8080", login)
	if callback.Code != http.StatusSeeOther {
		t.Fatalf("callback: %d %s", callback.Code, callback.Body.String())
	}
	var sessionCookie *http.Cookie
	for _, item := range callback.Result().Cookies() {
		if strings.Contains(item.Name, "user_session") {
			sessionCookie = item
		}
	}
	if sessionCookie == nil {
		t.Fatal("User session cookie missing")
	}
	if request(router, http.MethodGet, "/api/v1/me/owner-check", "app.localhost:8080").Code != http.StatusUnauthorized {
		t.Fatal("unauthenticated product request succeeded")
	}
	if request(router, http.MethodGet, "/api/v1/me/owner-check", "admin.localhost:8080", sessionCookie).Code != http.StatusBadRequest {
		t.Fatal("User session accepted on Staff host")
	}
	if got := request(router, http.MethodGet, "/api/v1/me/owner-check", "app.localhost:8080", sessionCookie); got.Code != http.StatusOK {
		t.Fatalf("verified product owner unavailable: %d", got.Code)
	}
	info := request(router, http.MethodGet, "/auth/user/session", "app.localhost:8080", sessionCookie)
	var payload struct {
		PrincipalID string `json:"principalId"`
		CSRFToken   string `json:"csrfToken"`
	}
	if err := json.Unmarshal(info.Body.Bytes(), &payload); err != nil || payload.PrincipalID != expectedUser.String() || payload.CSRFToken == "" {
		t.Fatalf("session principal mismatch: %s %v", info.Body.String(), err)
	}
	if request(router, http.MethodPost, "/api/v1/me/owner-check", "app.localhost:8080", sessionCookie).Code != http.StatusForbidden {
		t.Fatal("mutation bypassed CSRF")
	}
	post := httptest.NewRequest(http.MethodPost, "/api/v1/me/owner-check", nil)
	post.Host = "app.localhost:8080"
	post.AddCookie(sessionCookie)
	post.Header.Set("Origin", "http://app.localhost:8080")
	post.Header.Set("X-CSRF-Token", payload.CSRFToken)
	out := httptest.NewRecorder()
	router.ServeHTTP(out, post)
	if out.Code != http.StatusNoContent {
		t.Fatalf("valid User mutation failed: %d", out.Code)
	}
	// Never accept a user identifier supplied by the client instead of the
	// verified session principal. The route ignores this forged header.
	forged := httptest.NewRequest(http.MethodGet, "/api/v1/me/owner-check", nil)
	forged.Host = "app.localhost:8080"
	forged.AddCookie(sessionCookie)
	forged.Header.Set("X-User-ID", uuid.NewString())
	router.ServeHTTP(httptest.NewRecorder(), forged)
	if expectedUser.String() != payload.PrincipalID {
		t.Fatal("client ID replaced verified principal")
	}
}
