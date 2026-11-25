package httpadapter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
	auth "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth"
	authhttp "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth/adapters/http"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/config"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

// This is a test-only in-memory Auth Store, not a production shortcut. It
// exercises the real AUTH-03/04 BFF guard with the actual Admin handler.
type fakeStore struct {
	logins     map[string]auth.LoginRecord
	sessions   map[string]auth.Session
	staffID    uuid.UUID
	userID     uuid.UUID
	allowStaff bool
}

func (s *fakeStore) SaveLogin(_ context.Context, record auth.LoginRecord) error {
	if s.logins == nil {
		s.logins = make(map[string]auth.LoginRecord)
	}
	s.logins[string(record.StateHash)] = record
	return nil
}
func (s *fakeStore) ConsumeLogin(_ context.Context, hash []byte) (auth.LoginRecord, error) {
	record, ok := s.logins[string(hash)]
	if !ok {
		return record, auth.ErrNotFound
	}
	delete(s.logins, string(hash))
	return record, nil
}
func (s *fakeStore) ResolveIdentity(_ context.Context, kind auth.Kind, _, _ string) (uuid.UUID, error) {
	if kind == auth.Staff {
		if !s.allowStaff {
			return uuid.Nil, auth.ErrInactive
		}
		return s.staffID, nil
	}
	return s.userID, nil
}
func (s *fakeStore) SaveSession(_ context.Context, item auth.Session) error {
	if s.sessions == nil {
		s.sessions = make(map[string]auth.Session)
	}
	s.sessions[string(item.TokenHash)] = item
	return nil
}
func (s *fakeStore) ActiveSession(_ context.Context, kind auth.Kind, hash []byte) (auth.Session, error) {
	item, ok := s.sessions[string(hash)]
	if !ok || item.Kind != kind || (kind == auth.Staff && !s.allowStaff) || !item.ExpiresAt.After(time.Now()) {
		return auth.Session{}, auth.ErrNotFound
	}
	return item, nil
}
func (s *fakeStore) RevokeSession(_ context.Context, kind auth.Kind, hash []byte) error {
	item, exists := s.sessions[string(hash)]
	if !exists || item.Kind != kind {
		return auth.ErrNotFound
	}
	delete(s.sessions, string(hash))
	return nil
}
func (s *fakeStore) AppendAudit(_ context.Context, _ auth.AuditEvent) error { return nil }

type fakeAuditStore struct {
	mu        sync.Mutex
	events    []domain.AuditEvent
	appendErr error
	listErr   error
}

func (s *fakeAuditStore) Append(_ context.Context, e domain.AuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.appendErr != nil {
		return s.appendErr
	}
	e.RecordedAt = time.Now().UTC().Truncate(time.Millisecond)
	s.events = append(s.events, e)
	return nil
}
func (s *fakeAuditStore) List(_ context.Context, f domain.AuditFilter) ([]domain.AuditEvent, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listErr != nil {
		return nil, "", s.listErr
	}
	items := make([]domain.AuditEvent, 0, len(s.events))
	for _, e := range s.events {
		if e.RecordedAt.Before(f.From) || !e.RecordedAt.Before(f.To) {
			continue
		}
		if f.ActorStaffID != uuid.Nil && e.ActorStaffID != f.ActorStaffID {
			continue
		}
		if f.ActionCode != "" && e.ActionCode != f.ActionCode {
			continue
		}
		if f.Cursor != nil && !e.RecordedAt.Before(f.Cursor.RecordedAt) {
			continue
		}
		items = append(items, e)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].RecordedAt.After(items[j].RecordedAt) })
	var next string
	if len(items) > f.Limit {
		items = items[:f.Limit]
		last := items[len(items)-1]
		next = (domain.AuditCursor{RecordedAt: last.RecordedAt, ID: last.ID}).Encode()
	}
	return items, next, nil
}

type fakeOIDC struct{}

func (fakeOIDC) AuthorizationURL(state, _, _ string) string {
	return "https://issuer.test/auth?state=" + url.QueryEscape(state)
}
func (fakeOIDC) Exchange(_ context.Context, code, verifier, nonce string) (string, string, error) {
	if code != "valid" || len(verifier) < 40 || len(nonce) < 40 {
		return "", "", auth.ErrInvalid
	}
	return "test-subject", "test-keycloak-sid", nil
}

func setup(t *testing.T, auditStores ...*fakeAuditStore) (*gin.Engine, *fakeStore) {
	return setupWithOverview(t, &fakeOverviewStore{}, auditStores...)
}

func setupWithOverview(t *testing.T, overview application.OverviewStore, auditStores ...*fakeAuditStore) (*gin.Engine, *fakeStore) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	store := &fakeStore{staffID: uuid.New(), userID: uuid.New(), allowStaff: true}
	provider := fakeOIDC{}
	svc, err := auth.New(store, []byte(strings.Repeat("s", 32)), map[auth.Kind]auth.Realm{
		auth.User:  {Origin: "http://app.localhost:8080", RedirectURI: "http://app.localhost:8080/auth/user/callback", Issuer: "https://issuer.test/users", Provider: provider},
		auth.Staff: {Origin: "http://admin.localhost:8080", RedirectURI: "http://admin.localhost:8080/auth/staff/callback", Issuer: "https://issuer.test/staff", Provider: provider},
	})
	if err != nil {
		t.Fatal(err)
	}
	bff := authhttp.NewForTests(svc, config.AuthConfig{
		Enabled: true, UserLoginEnabled: true, StaffLoginEnabled: true,
		UserOrigin: "http://app.localhost:8080", StaffOrigin: "http://admin.localhost:8080",
	})
	r := gin.New()
	r.Use(httpx.Middleware("http://admin.localhost:8080"))
	r.GET("/auth/user/login", bff.Login(auth.User))
	r.GET("/auth/user/callback", bff.Callback(auth.User))
	r.GET("/auth/staff/login", bff.Login(auth.Staff))
	r.GET("/auth/staff/callback", bff.Callback(auth.Staff))
	r.GET("/auth/staff/session", bff.Session(auth.Staff))
	r.POST("/auth/staff/logout", bff.Logout(auth.Staff))
	adminAPI := r.Group("/api/v1/admin")
	adminAPI.Use(bff.RequireStaff())
	repo := &fakeAuditStore{}
	if len(auditStores) > 0 {
		repo = auditStores[0]
	}
	handler := New(application.NewWithRepositories(repo, overview))
	adminAPI.GET("/me", handler.Me)
	adminAPI.GET("/audit", handler.Audit)
	adminAPI.GET("/overview", handler.Overview)
	return r, store
}

func call(r *gin.Engine, method, host, path string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	req.Host = host
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	r.ServeHTTP(w, req)
	return w
}
func login(t *testing.T, r *gin.Engine, kind auth.Kind) *http.Cookie {
	t.Helper()
	host := "admin.localhost:8080"
	if kind == auth.User {
		host = "app.localhost:8080"
	}
	base := "/auth/" + string(kind)
	start := call(r, http.MethodGet, host, base+"/login")
	if start.Code != http.StatusFound || len(start.Result().Cookies()) != 1 {
		t.Fatalf("begin login: %d %s", start.Code, start.Body.String())
	}
	stateCookie := start.Result().Cookies()[0]
	cb := call(r, http.MethodGet, host, base+"/callback?code=valid&state="+url.QueryEscape(stateCookie.Value), stateCookie)
	if cb.Code != http.StatusSeeOther {
		t.Fatalf("complete login: %d %s", cb.Code, cb.Body.String())
	}
	for _, cookie := range cb.Result().Cookies() {
		if strings.Contains(cookie.Name, "session") && cookie.MaxAge > 0 {
			return cookie
		}
	}
	t.Fatal("missing application session cookie")
	return nil
}

func TestAdminMeGuardAndRevocation(t *testing.T) {
	r, store := setup(t)
	adminPath := "/api/v1/admin/me"
	// Neither headers nor existing Dev Session cookies are Staff identities.
	w := call(r, http.MethodGet, "admin.localhost:8080", adminPath,
		&http.Cookie{Name: "wellbeing_dev_session", Value: "synthetic"})
	if w.Code != 401 {
		t.Fatalf("dev cookie accepted: %d", w.Code)
	}
	forged := httptest.NewRequest(http.MethodGet, adminPath+"?role=admin&staffId="+store.staffID.String(), nil)
	forged.Host = "admin.localhost:8080"
	forged.Header.Set("X-Staff-ID", store.staffID.String())
	forged.Header.Set("X-Role", "admin")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, forged)
	if w.Code != 401 {
		t.Fatalf("forged identity accepted: %d", w.Code)
	}
	userCookie := login(t, r, auth.User)
	if w = call(r, http.MethodGet, "admin.localhost:8080", adminPath, userCookie); w.Code != 401 {
		t.Fatalf("user realm accepted as admin: %d", w.Code)
	}
	staffCookie := login(t, r, auth.Staff)
	if w = call(r, http.MethodGet, "app.localhost:8080", adminPath, staffCookie); w.Code != 400 {
		t.Fatalf("wrong origin accepted: %d", w.Code)
	}
	w = call(r, http.MethodGet, "admin.localhost:8080", adminPath, staffCookie)
	if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("active staff request: %d %s", w.Code, w.Body.String())
	}
	var body struct {
		StaffID   string `json:"staffId"`
		Role      string `json:"role"`
		Active    bool   `json:"active"`
		ExpiresAt string `json:"expiresAt"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.StaffID != store.staffID.String() || body.Role != "admin" || !body.Active || body.ExpiresAt == "" {
		t.Fatalf("unexpected /me response: %s %v", w.Body.String(), err)
	}
	if strings.Contains(w.Body.String(), "test-subject") || strings.Contains(w.Body.String(), "keycloak") {
		t.Fatal("OIDC secrets in Admin response")
	}
	store.allowStaff = false
	if w = call(r, http.MethodGet, "admin.localhost:8080", adminPath, staffCookie); w.Code != 401 {
		t.Fatalf("deactivated staff accepted: %d", w.Code)
	}
	store.allowStaff = true
	info := call(r, http.MethodGet, "admin.localhost:8080", "/auth/staff/session", staffCookie)
	if info.Code != 200 {
		t.Fatalf("session lookup: %d", info.Code)
	}
	var sess struct {
		CSRFToken string `json:"csrfToken"`
	}
	if err := json.Unmarshal(info.Body.Bytes(), &sess); err != nil || sess.CSRFToken == "" {
		t.Fatalf("missing CSRF token: %v", err)
	}
	logout := httptest.NewRequest(http.MethodPost, "/auth/staff/logout", nil)
	logout.Host = "admin.localhost:8080"
	logout.AddCookie(staffCookie)
	logout.Header.Set("Origin", "http://admin.localhost:8080")
	logout.Header.Set("X-CSRF-Token", sess.CSRFToken)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, logout)
	if w.Code != 204 {
		t.Fatalf("logout: %d %s", w.Code, w.Body.String())
	}
	if w = call(r, http.MethodGet, "admin.localhost:8080", adminPath, staffCookie); w.Code != 401 {
		t.Fatalf("revoked session accepted: %d", w.Code)
	}
}

func TestDirectAdminHandlerRequiresVerifiedGuard(t *testing.T) {
	r := gin.New()
	r.GET("/admin/me", New(application.New()).Me) // deliberate missing middleware
	w := call(r, http.MethodGet, "admin.localhost:8080", "/admin/me")
	if w.Code != 401 || !strings.Contains(w.Body.String(), "STAFF_SESSION_REQUIRED") {
		t.Fatalf("direct handler should fail closed: %d %s", w.Code, w.Body.String())
	}
}
