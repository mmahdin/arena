package httpadapter

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	auth "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth/oidc"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/config"
)

type auth08Store struct {
	*httpStore
	eventSeen   bool
	unavailable bool
}

func (s *auth08Store) RevokeBackchannel(_ context.Context, kind auth.Kind, issuer, subject, sid string, _ []byte, _ time.Time) (bool, error) {
	if s.unavailable {
		return false, errors.New("SQL unavailable")
	}
	if s.eventSeen {
		return false, nil
	}
	s.eventSeen = true
	for hash, session := range s.session {
		if session.Kind == kind && session.Issuer == issuer && (sid == "" || session.KeycloakSID == sid) &&
			(subject == "" || session.Subject == subject) {
			delete(s.session, hash)
		}
	}
	return true, nil
}
func (s *auth08Store) RevokeAllForPrincipal(_ context.Context, kind auth.Kind, id uuid.UUID) error {
	if s.unavailable {
		return errors.New("SQL unavailable")
	}
	for hash, session := range s.session {
		if session.Kind == kind && session.PrincipalID == id {
			delete(s.session, hash)
		}
	}
	return nil
}

type auth08Provider struct {
	fakeProvider
	event oidc.LogoutEvent
}

func (p auth08Provider) VerifyBackchannel(_ context.Context, raw string) (oidc.LogoutEvent, error) {
	if raw != "verified-jwt" {
		return oidc.LogoutEvent{}, oidc.ErrInvalidLogoutToken
	}
	return p.event, nil
}
func auth08Fixture(t *testing.T) (*gin.Engine, *Handler, *auth08Store) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	store := &auth08Store{httpStore: &httpStore{principal: uuid.New()}}
	issuer := "https://issuer.example/realms/theia-users"
	provider := auth08Provider{event: oidc.LogoutEvent{Issuer: issuer, Subject: "owner-sub", SessionID: "sid-1", TokenID: "event-1", ExpiresAt: time.Now().Add(time.Minute)}}
	svc, err := auth.New(store, []byte(strings.Repeat("k", 32)), map[auth.Kind]auth.Realm{
		auth.User:  {Issuer: issuer, Origin: "http://app.localhost:8080", RedirectURI: "http://app.localhost:8080/auth/user/callback", Provider: provider},
		auth.Staff: {Issuer: "https://issuer.example/realms/theia-staff", Origin: "http://admin.localhost:8080", RedirectURI: "http://admin.localhost:8080/auth/staff/callback", Provider: provider},
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := NewForTests(svc, config.AuthConfig{Enabled: true, UserLoginEnabled: true, StaffLoginEnabled: true,
		UserOrigin: "http://app.localhost:8080", StaffOrigin: "http://admin.localhost:8080",
		UserIssuer:  oidc.Settings{Issuer: issuer, ClientID: "theia-users-bff"},
		StaffIssuer: oidc.Settings{Issuer: "https://issuer.example/realms/theia-staff", ClientID: "theia-staff-bff"}})
	router := gin.New()
	router.GET("/auth/user/session", handler.Session(auth.User))
	router.POST("/auth/user/logout/sso", handler.LogoutWithSSO(auth.User, false))
	router.POST("/auth/user/logout-all", handler.LogoutWithSSO(auth.User, true))
	router.POST("/auth/user/backchannel-logout", handler.Backchannel(auth.User))
	router.POST("/auth/staff/backchannel-logout", handler.Backchannel(auth.Staff))
	return router, handler, store
}
func addAuth08Session(t *testing.T, store *auth08Store, token string) {
	t.Helper()
	d := sha256.Sum256([]byte(token))
	if store.session == nil {
		store.session = map[string]auth.Session{}
	}
	store.session[string(d[:])] = auth.Session{Kind: auth.User, PrincipalID: store.principal, TokenHash: d[:], Issuer: "https://issuer.example/realms/theia-users", Subject: "owner-sub", KeycloakSID: "sid-1", ExpiresAt: time.Now().Add(time.Hour)}
}
func auth08Post(router *gin.Engine, path, host string, cookie *http.Cookie, csrf string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, path, nil)
	r.Host = host
	if cookie != nil {
		r.AddCookie(cookie)
	}
	if csrf != "" {
		r.Header.Set("Origin", "http://"+host)
		r.Header.Set("X-CSRF-Token", csrf)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	return w
}
func TestAUTH08SSOLogoutCSRFAndFixedBrowserURL(t *testing.T) {
	router, h, store := auth08Fixture(t)
	first, second := strings.Repeat("a", 43), strings.Repeat("b", 43)
	addAuth08Session(t, store, first)
	addAuth08Session(t, store, second)
	cookie := &http.Cookie{Name: h.cookieName(auth.User, false), Value: first}
	if got := auth08Post(router, "/auth/user/logout/sso", "app.localhost:8080", cookie, ""); got.Code != 403 {
		t.Fatalf("missing CSRF: %d", got.Code)
	}
	if got := auth08Post(router, "/auth/user/logout/sso", "admin.localhost:8080", cookie, "bad"); got.Code != 400 {
		t.Fatalf("wrong origin: %d", got.Code)
	}
	csrf := h.service.CSRFToken(first)
	got := auth08Post(router, "/auth/user/logout/sso", "app.localhost:8080", cookie, csrf)
	if got.Code != 200 {
		t.Fatalf("SSO handoff: %d %s", got.Code, got.Body.String())
	}
	var body struct{ LogoutURL, TheiaSessionsRevoked, KeycloakSSOStatus string }
	if err := json.Unmarshal(got.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(body.LogoutURL)
	if err != nil {
		t.Fatal(err)
	}
	if u.Host != "issuer.example" || u.Query().Get("client_id") != "theia-users-bff" ||
		u.Query().Get("post_logout_redirect_uri") != "http://app.localhost:8080/" ||
		body.TheiaSessionsRevoked != "current" || body.KeycloakSSOStatus != "browser_action_required" {
		t.Fatalf("unsafe browser SSO response: %+v", body)
	}
	if request(router, "GET", "/auth/user/session", "app.localhost:8080", cookie).Code != 401 {
		t.Fatal("SSO handoff left current session alive")
	}
	secondCookie := &http.Cookie{Name: h.cookieName(auth.User, false), Value: second}
	if request(router, "GET", "/auth/user/session", "app.localhost:8080", secondCookie).Code != 200 {
		t.Fatal("current-only logout revoked other device")
	}
	got = auth08Post(router, "/auth/user/logout-all", "app.localhost:8080", secondCookie, h.service.CSRFToken(second))
	if got.Code != 200 || request(router, "GET", "/auth/user/session", "app.localhost:8080", secondCookie).Code != 401 {
		t.Fatal("all-device revocation failed")
	}
}
func TestAUTH08BackchannelJWTErrorHandlingReplayAndRealm(t *testing.T) {
	router, h, store := auth08Fixture(t)
	token := strings.Repeat("c", 43)
	addAuth08Session(t, store, token)
	cookie := &http.Cookie{Name: h.cookieName(auth.User, false), Value: token}
	backchannel := func(path, ct, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		r.Host = "api:8080"
		r.Header.Set("Content-Type", ct)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	endpoint := "/auth/user/backchannel-logout"
	if got := backchannel(endpoint, "application/json", "{}"); got.Code != 415 {
		t.Fatalf("bad content type %d", got.Code)
	}
	if got := backchannel(endpoint, "application/x-www-form-urlencoded", "logout_token=fake"); got.Code != 400 {
		t.Fatalf("unsigned JWT accepted: %d", got.Code)
	}
	if request(router, "GET", "/auth/user/session", "app.localhost:8080", cookie).Code != 200 {
		t.Fatal("invalid notification revoked session")
	}
	if got := backchannel("/auth/staff/backchannel-logout", "application/x-www-form-urlencoded", "logout_token=verified-jwt"); got.Code != 400 {
		t.Fatalf("cross-realm JWT accepted: %d", got.Code)
	}
	for i := 0; i < 2; i++ {
		if got := backchannel(endpoint, "application/x-www-form-urlencoded", "logout_token=verified-jwt"); got.Code != 200 {
			t.Fatalf("Keycloak signed logout refused %d", got.Code)
		}
	}
	if request(router, "GET", "/auth/user/session", "app.localhost:8080", cookie).Code != 401 {
		t.Fatal("backchannel failed to revoke Theia")
	}
	store.unavailable = true
	if got := backchannel(endpoint, "application/x-www-form-urlencoded", "logout_token=verified-jwt"); got.Code != 503 && got.Code != 200 {
		t.Fatalf("unhandled storage error: %d", got.Code)
	}
}
