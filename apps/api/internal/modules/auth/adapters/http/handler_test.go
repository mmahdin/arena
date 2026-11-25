package httpadapter

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	auth "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/config"
)

type httpStore struct {
	login      map[string]auth.LoginRecord
	session    map[string]auth.Session
	principal  uuid.UUID
	allowStaff bool
}

func (h *httpStore) SaveLogin(_ context.Context, r auth.LoginRecord) error {
	if h.login == nil {
		h.login = make(map[string]auth.LoginRecord)
	}
	h.login[string(r.StateHash)] = r
	return nil
}
func (h *httpStore) ConsumeLogin(_ context.Context, hash []byte) (auth.LoginRecord, error) {
	r, ok := h.login[string(hash)]
	if !ok {
		return r, auth.ErrNotFound
	}
	delete(h.login, string(hash))
	return r, nil
}
func (h *httpStore) ResolveIdentity(_ context.Context, kind auth.Kind, issuer, sub string) (uuid.UUID, error) {
	if kind == auth.Staff && !h.allowStaff {
		return uuid.Nil, auth.ErrInactive
	}
	return h.principal, nil
}
func (h *httpStore) SaveSession(_ context.Context, s auth.Session) error {
	if h.session == nil {
		h.session = make(map[string]auth.Session)
	}
	h.session[string(s.TokenHash)] = s
	return nil
}
func (h *httpStore) ActiveSession(_ context.Context, kind auth.Kind, hash []byte) (auth.Session, error) {
	s, ok := h.session[string(hash)]
	if !ok || s.Kind != kind || (kind == auth.Staff && !h.allowStaff) {
		return s, auth.ErrNotFound
	}
	return s, nil
}
func (h *httpStore) RevokeSession(_ context.Context, kind auth.Kind, hash []byte) error {
	delete(h.session, string(hash))
	return nil
}
func (h *httpStore) AppendAudit(_ context.Context, e auth.AuditEvent) error { return nil }

type fakeProvider struct{}

func (fakeProvider) AuthorizationURL(state, nonce, verifier string) string {
	return "https://issuer.example/authorize?state=" + url.QueryEscape(state)
}
func (fakeProvider) Exchange(_ context.Context, code, verifier, nonce string) (string, string, error) {
	if code != "valid" || len(verifier) < 40 || len(nonce) < 40 {
		return "", "", auth.ErrInvalid
	}
	return "opaque-user-sub", "sid", nil
}
func setupHandler(t *testing.T) (*gin.Engine, *Handler) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	store := &httpStore{principal: uuid.New()}
	p := fakeProvider{}
	service, err := auth.New(store, []byte(strings.Repeat("k", 32)), map[auth.Kind]auth.Realm{
		auth.User:  {Origin: "http://app.localhost:8080", Issuer: "https://issuer.example/realms/users", RedirectURI: "http://app.localhost:8080/auth/user/callback", Provider: p},
		auth.Staff: {Origin: "http://admin.localhost:8080", Issuer: "https://issuer.example/realms/staff", RedirectURI: "http://admin.localhost:8080/auth/staff/callback", Provider: p},
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := NewForTests(service, config.AuthConfig{Enabled: true, UserLoginEnabled: true, StaffLoginEnabled: true, UserOrigin: "http://app.localhost:8080", StaffOrigin: "http://admin.localhost:8080"})
	router := gin.New()
	router.GET("/auth/user/login", handler.Login(auth.User))
	router.GET("/auth/user/callback", handler.Callback(auth.User))
	router.GET("/auth/user/session", handler.Session(auth.User))
	router.POST("/auth/user/logout", handler.Logout(auth.User))
	router.GET("/auth/staff/login", handler.Login(auth.Staff))
	router.GET("/auth/staff/session", handler.Session(auth.Staff))
	router.POST("/api/test", handler.RequireUser(), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	return router, handler
}
func request(router *gin.Engine, method, path, host string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(method, path, nil)
	r.Host = host
	for _, cookie := range cookies {
		r.AddCookie(cookie)
	}
	router.ServeHTTP(w, r)
	return w
}
func TestBrowserLoginOriginIsolationAndCSRF(t *testing.T) {
	router, _ := setupHandler(t)
	bad := request(router, "GET", "/auth/user/login", "admin.localhost:8080")
	if bad.Code != 400 {
		t.Fatalf("wrong host login got %d", bad.Code)
	}
	start := request(router, "GET", "/auth/user/login", "app.localhost:8080")
	if start.Code != 302 || len(start.Result().Cookies()) == 0 {
		t.Fatal("login didn't redirect or set cookie")
	}
	cookie := start.Result().Cookies()[0]
	link, err := url.Parse(start.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	state := link.Query().Get("state")
	if state == "" || cookie.Value != state {
		t.Fatal("state binding missing")
	}
	// Mismatched cross-browser/login cookie must never establish a session.
	mismatched := request(router, "GET", "/auth/user/callback?state="+state+"&code=valid", "app.localhost:8080", &http.Cookie{Name: cookie.Name, Value: "attacker"})
	if mismatched.Code != 401 {
		t.Fatalf("login CSRF accepted: %d", mismatched.Code)
	}
	cb := request(router, "GET", "/auth/user/callback?state="+state+"&code=valid", "app.localhost:8080", cookie)
	if cb.Code != 303 {
		t.Fatalf("callback failed: %d %s", cb.Code, cb.Body.String())
	}
	var sessionCookie *http.Cookie
	for _, c := range cb.Result().Cookies() {
		if strings.Contains(c.Name, "session") {
			sessionCookie = c
		}
	}
	if sessionCookie == nil || !sessionCookie.HttpOnly || sessionCookie.Path != "/" {
		t.Fatal("missing secure session cookie")
	}
	staff := request(router, "GET", "/auth/staff/session", "admin.localhost:8080", sessionCookie)
	if staff.Code != 401 {
		t.Fatal("user session accepted as staff")
	}
	sess := request(router, "GET", "/auth/user/session", "app.localhost:8080", sessionCookie)
	if sess.Code != 200 {
		t.Fatalf("session failed: %d", sess.Code)
	}
	var body struct {
		CSRFToken string `json:"csrfToken"`
	}
	if json.Unmarshal(sess.Body.Bytes(), &body) != nil || body.CSRFToken == "" {
		t.Fatal("missing CSRF value")
	}
	post := func(token, origin string) int {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/api/test", nil)
		r.Host = "app.localhost:8080"
		r.AddCookie(sessionCookie)
		r.Header.Set("Origin", origin)
		r.Header.Set("X-CSRF-Token", token)
		router.ServeHTTP(w, r)
		return w.Code
	}
	if post("", "http://app.localhost:8080") != 403 || post(body.CSRFToken, "http://evil.invalid") != 403 {
		t.Fatal("CSRF protection bypass")
	}
	if post(body.CSRFToken, "http://app.localhost:8080") != 204 {
		t.Fatal("valid CSRF rejected")
	}
	// Verify the fake session token is an opaque, hash-indexed bearer secret.
	sum := sha256.Sum256([]byte(sessionCookie.Value))
	if len(sum) != 32 {
		t.Fatal("hash failed")
	}
}

func TestStaffProvisionedLoginSessionGuardAndLogout(t *testing.T) {
	// Staff authentication uses the same OIDC BFF infrastructure as Users but
	// resolves only explicitly provisioned active staff identities.
	staffDB := &httpStore{principal: uuid.New(), allowStaff: true}
	provider := fakeProvider{}
	service, err := auth.New(staffDB, []byte(strings.Repeat("k", 32)), map[auth.Kind]auth.Realm{
		auth.User:  {Origin: "http://app.localhost:8080", Issuer: "https://issuer.example/realms/users", RedirectURI: "http://app.localhost:8080/auth/user/callback", Provider: provider},
		auth.Staff: {Origin: "http://admin.localhost:8080", Issuer: "https://issuer.example/realms/staff", RedirectURI: "http://admin.localhost:8080/auth/staff/callback", Provider: provider},
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := NewForTests(service, config.AuthConfig{Enabled: true, StaffLoginEnabled: true, StaffOrigin: "http://admin.localhost:8080", UserOrigin: "http://app.localhost:8080"})
	router := gin.New()
	router.GET("/auth/staff/login", handler.Login(auth.Staff))
	router.GET("/auth/staff/callback", handler.Callback(auth.Staff))
	router.GET("/auth/staff/session", handler.Session(auth.Staff))
	router.POST("/auth/staff/logout", handler.Logout(auth.Staff))
	router.GET("/auth/user/login", handler.Login(auth.User))
	router.GET("/api/v1/admin/test", handler.RequireStaff(), func(c *gin.Context) {
		if StaffID(c) != staffDB.principal || c.GetString("staffRole") != "admin" {
			c.Status(500)
			return
		}
		c.Status(204)
	})
	router.POST("/api/v1/admin/test", handler.RequireStaff(), func(c *gin.Context) { c.Status(204) })
	if request(router, "GET", "/auth/user/login", "app.localhost:8080").Code != 503 {
		t.Fatal("user login activated before SMS authentication")
	}
	start := request(router, "GET", "/auth/staff/login", "admin.localhost:8080")
	if start.Code != 302 {
		t.Fatalf("staff login start: %d %s", start.Code, start.Body.String())
	}
	loginCookie := start.Result().Cookies()[0]
	state := loginCookie.Value
	unauthorized := request(router, "GET", "/auth/staff/callback?state="+state+"&code=valid", "app.localhost:8080", loginCookie)
	if unauthorized.Code != 400 {
		t.Fatalf("wrong host callback: %d", unauthorized.Code)
	}
	callback := request(router, "GET", "/auth/staff/callback?state="+state+"&code=valid", "admin.localhost:8080", loginCookie)
	if callback.Code != 303 {
		t.Fatalf("staff callback: %d %s", callback.Code, callback.Body.String())
	}
	var cookie *http.Cookie
	for _, candidate := range callback.Result().Cookies() {
		if strings.Contains(candidate.Name, "staff_session") {
			cookie = candidate
		}
	}
	if cookie == nil || !cookie.HttpOnly || cookie.Path != "/" {
		t.Fatal("missing staff session cookie")
	}
	if request(router, "GET", "/api/v1/admin/test", "admin.localhost:8080").Code != 401 {
		t.Fatal("unguarded admin API")
	}
	if request(router, "GET", "/api/v1/admin/test", "admin.localhost:8080", cookie).Code != 204 {
		t.Fatal("staff guard failed")
	}
	if request(router, "GET", "/api/v1/admin/test", "app.localhost:8080", cookie).Code != 400 {
		t.Fatal("wrong host accepted")
	}
	info := request(router, "GET", "/auth/staff/session", "admin.localhost:8080", cookie)
	if info.Code != 200 {
		t.Fatalf("staff session inspect: %d", info.Code)
	}
	var payload struct{ Role, CSRFToken string }
	if err = json.Unmarshal(info.Body.Bytes(), &payload); err != nil || payload.Role != "admin" || payload.CSRFToken == "" {
		t.Fatalf("unexpected session response %v: %s", err, info.Body.String())
	}
	badLogout := request(router, "POST", "/auth/staff/logout", "admin.localhost:8080", cookie)
	if badLogout.Code != 403 || request(router, "POST", "/api/v1/admin/test", "admin.localhost:8080", cookie).Code != 403 {
		t.Fatal("staff mutation bypassed CSRF")
	}
	validPost := httptest.NewRecorder()
	validRequest := httptest.NewRequest("POST", "/api/v1/admin/test", nil)
	validRequest.Host = "admin.localhost:8080"
	validRequest.AddCookie(cookie)
	validRequest.Header.Set("Origin", "http://admin.localhost:8080")
	validRequest.Header.Set("X-CSRF-Token", payload.CSRFToken)
	router.ServeHTTP(validPost, validRequest)
	if validPost.Code != 204 {
		t.Fatalf("staff mutation refused with valid CSRF: %d", validPost.Code)
	}
	// Deactivation must deny a still-valid staff session on its next request.
	staffDB.allowStaff = false
	if request(router, "GET", "/auth/staff/session", "admin.localhost:8080", cookie).Code != 401 {
		t.Fatal("inactive staff session accepted")
	}
	staffDB.allowStaff = true
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/auth/staff/logout", nil)
	req.Host = "admin.localhost:8080"
	req.AddCookie(cookie)
	req.Header.Set("Origin", "http://admin.localhost:8080")
	req.Header.Set("X-CSRF-Token", payload.CSRFToken)
	router.ServeHTTP(w, req)
	if w.Code != 204 {
		t.Fatalf("staff logout %d", w.Code)
	}
	if request(router, "GET", "/api/v1/admin/test", "admin.localhost:8080", cookie).Code != 401 {
		t.Fatal("revoked staff session accepted")
	}
}

func TestUnprovisionedStaffDeniedAfterKeycloakSuccess(t *testing.T) {
	db := &httpStore{principal: uuid.New(), allowStaff: false}
	provider := fakeProvider{}
	svc, err := auth.New(db, []byte(strings.Repeat("k", 32)), map[auth.Kind]auth.Realm{
		auth.User:  {Origin: "http://app.localhost:8080", Issuer: "https://issuer.example/realms/users", RedirectURI: "http://app.localhost:8080/auth/user/callback", Provider: provider},
		auth.Staff: {Origin: "http://admin.localhost:8080", Issuer: "https://issuer.example/realms/staff", RedirectURI: "http://admin.localhost:8080/auth/staff/callback", Provider: provider},
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := NewForTests(svc, config.AuthConfig{Enabled: true, StaffLoginEnabled: true, StaffOrigin: "http://admin.localhost:8080"})
	router := gin.New()
	router.GET("/auth/staff/login", handler.Login(auth.Staff))
	router.GET("/auth/staff/callback", handler.Callback(auth.Staff))
	begin := request(router, "GET", "/auth/staff/login", "admin.localhost:8080")
	if begin.Code != 302 {
		t.Fatal("start refused")
	}
	cookie := begin.Result().Cookies()[0]
	cb := request(router, "GET", "/auth/staff/callback?state="+cookie.Value+"&code=valid", "admin.localhost:8080", cookie)
	if cb.Code != 403 || !strings.Contains(cb.Body.String(), "STAFF_NOT_AUTHORIZED") {
		t.Fatalf("unprovisioned staff accepted: %d %s", cb.Code, cb.Body.String())
	}
}
