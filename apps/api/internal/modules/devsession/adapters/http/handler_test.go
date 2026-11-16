package httpadapter

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/devsession/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/devsecurity"
)

type testSession struct {
	user    uuid.UUID
	expires time.Time
	revoked bool
}
type fakeSessionRepository struct {
	users      map[uuid.UUID]bool
	sessions   map[uuid.UUID]*testSession
	phoneUsers map[string]uuid.UUID
	checkError error
}

func (r *fakeSessionRepository) List(context.Context) ([]application.TestUser, error) {
	var result []application.TestUser
	for id := range r.users {
		result = append(result, application.TestUser{ID: id, TestKey: "synthetic"})
	}
	return result, nil
}
func (r *fakeSessionRepository) CreateSession(_ context.Context, userID, sessionID uuid.UUID, expiresAt time.Time) (bool, error) {
	if !r.users[userID] {
		return false, nil
	}
	r.sessions[sessionID] = &testSession{user: userID, expires: expiresAt}
	return true, nil
}
func (r *fakeSessionRepository) FindOrCreatePhoneSession(_ context.Context, phone string, sid uuid.UUID, expires time.Time) (uuid.UUID, bool, error) {
	if r.phoneUsers == nil {
		r.phoneUsers = make(map[string]uuid.UUID)
	}
	uid, ok := r.phoneUsers[phone]
	if !ok {
		uid = uuid.New()
		r.phoneUsers[phone] = uid
		r.users[uid] = true
	}
	r.sessions[sid] = &testSession{user: uid, expires: expires}
	return uid, !ok, nil
}

func (r *fakeSessionRepository) RevokeSession(_ context.Context, userID, sessionID uuid.UUID) error {
	if session, ok := r.sessions[sessionID]; ok && session.user == userID {
		session.revoked = true
	}
	return nil
}
func (r *fakeSessionRepository) IsActive(_ context.Context, userID, sessionID uuid.UUID) (bool, error) {
	if r.checkError != nil {
		return false, r.checkError
	}
	session, ok := r.sessions[sessionID]
	return ok && session.user == userID && r.users[userID] && !session.revoked && time.Now().Before(session.expires), nil
}

func TestDevSessionLoginSwitchLogoutAndGuards(t *testing.T) {
	gin.SetMode(gin.TestMode)
	a, b := uuid.New(), uuid.New()
	repo := &fakeSessionRepository{users: map[uuid.UUID]bool{a: true, b: true}, sessions: map[uuid.UUID]*testSession{}}
	signer, err := devsecurity.NewSigner("0123456789abcdef0123456789abcdef", false)
	if err != nil {
		t.Fatal(err)
	}
	handler := New(application.New(repo), signer)
	router := gin.New()
	group := router.Group("/api/v1")
	group.GET("/dev/users", handler.List)
	group.POST("/dev/session", handler.Login)
	group.DELETE("/dev/session", handler.Logout)
	private := group.Group("")
	private.Use(devsecurity.Guard(signer, repo))
	private.GET("/me", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"userId": devsecurity.UserID(c).String()}) })

	call := func(method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		if cookie != nil {
			req.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	check := func(w *httptest.ResponseRecorder, status int) {
		t.Helper()
		if w.Code != status {
			t.Fatalf("expected %d, got %d (body=%s)", status, w.Code, w.Body.String())
		}
	}
	check(call("GET", "/api/v1/me", "", nil), 401)
	check(call("POST", "/api/v1/dev/session", `{"userId":"`+uuid.NewString()+`"}`, nil), 404)
	loginA := call("POST", "/api/v1/dev/session", `{"userId":"`+a.String()+`"}`, nil)
	check(loginA, 200)
	if len(loginA.Result().Cookies()) != 1 {
		t.Fatal("no session cookie")
	}
	cookieA := loginA.Result().Cookies()[0]
	if !cookieA.HttpOnly || cookieA.Path != "/api/v1" {
		t.Fatal("unsafe cookie flags")
	}
	check(call("GET", "/api/v1/me", "", cookieA), 200)
	fakeCookie := &http.Cookie{Name: cookieA.Name, Value: cookieA.Value + "tampered"}
	check(call("GET", "/api/v1/me", "", fakeCookie), 401)
	loginB := call("POST", "/api/v1/dev/session", `{"userId":"`+b.String()+`"}`, cookieA)
	check(loginB, 200)
	cookieB := loginB.Result().Cookies()[0]
	check(call("GET", "/api/v1/me", "", cookieA), 401)
	check(call("GET", "/api/v1/me", "", cookieB), 200)
	check(call("DELETE", "/api/v1/dev/session", "", cookieB), 204)
	check(call("GET", "/api/v1/me", "", cookieB), 401)
	expired := signer.Issue(a, uuid.New(), time.Now().Add(-13*time.Hour))
	check(call("GET", "/api/v1/me", "", &http.Cookie{Name: cookieA.Name, Value: expired}), 401)
	// Revocation also works if a tester is removed from the eligible set.
	fresh := call("POST", "/api/v1/dev/session", `{"userId":"`+a.String()+`"}`, nil)
	check(fresh, 200)
	repo.users[a] = false
	check(call("GET", "/api/v1/me", "", fresh.Result().Cookies()[0]), 401)
	repo.checkError = errors.New("database offline")
	check(call("GET", "/api/v1/me", "", cookieB), 503) // valid signature; storage unavailable
}

func TestSyntheticPhoneIdentitySession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &fakeSessionRepository{users: map[uuid.UUID]bool{}, sessions: map[uuid.UUID]*testSession{}}
	signer, err := devsecurity.NewSigner("0123456789abcdef0123456789abcdef", false)
	if err != nil {
		t.Fatal(err)
	}
	handler := New(application.New(repo), signer)
	router := gin.New()
	router.POST("/api/v1/dev/identity-session", handler.IdentityLogin)
	private := router.Group("/api/v1")
	private.Use(devsecurity.Guard(signer, repo))
	private.GET("/me", func(c *gin.Context) { c.JSON(200, gin.H{"userId": devsecurity.UserID(c).String()}) })
	call := func(phone string, cookie *http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/api/v1/dev/identity-session", bytes.NewBufferString(`{"phone":"`+phone+`"}`))
		req.Header.Set("Content-Type", "application/json")
		if cookie != nil {
			req.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}
	if w := call("0900000000", nil); w.Code != 400 {
		t.Fatalf("invalid-length identifier allowed: %d", w.Code)
	}
	first := call("۰۹۰۰۰۰۰۰۰۰۱", nil)
	if first.Code != 200 || !bytes.Contains(first.Body.Bytes(), []byte(`"created":true`)) {
		t.Fatalf("new tester: %d %s", first.Code, first.Body.String())
	}
	cookieA := first.Result().Cookies()[0]
	same := call("09000000001", cookieA)
	if same.Code != 200 || !bytes.Contains(same.Body.Bytes(), []byte(`"created":false`)) {
		t.Fatalf("existing tester: %d %s", same.Code, same.Body.String())
	}
	cookieA2 := same.Result().Cookies()[0]
	me := func(cookie *http.Cookie) int {
		req := httptest.NewRequest("GET", "/api/v1/me", nil)
		req.AddCookie(cookie)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w.Code
	}
	if me(cookieA) != 401 || me(cookieA2) != 200 {
		t.Fatal("old cookie still active or new cookie rejected")
	}
	second := call("09010000002", cookieA2)
	if second.Code != 200 || me(cookieA2) != 401 || me(second.Result().Cookies()[0]) != 200 {
		t.Fatal("phone identity switch failed")
	}
	if len(repo.phoneUsers) != 2 {
		t.Fatal("duplicate identity created")
	}
}
