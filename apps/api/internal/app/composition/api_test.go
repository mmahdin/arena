package composition

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/config"
)

func TestDevEndpointsAbsentButProductRoutesFailClosed(t *testing.T) {
	api, err := NewAPI(config.Config{Environment: "staging"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ method, route string }{
		{http.MethodGet, "/api/v1/dev/users"},
		{http.MethodPost, "/api/v1/dev/identity-session"},
		{http.MethodPost, "/api/v1/dev/session"},
		{http.MethodDelete, "/api/v1/dev/session"},
	} {
		w := httptest.NewRecorder()
		api.ServeHTTP(w, httptest.NewRequest(tc.method, tc.route, nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("%s %s: disabled dev route must be absent, got %d", tc.method, tc.route, w.Code)
		}
	}
	for _, route := range []string{
		"/api/v1/me", "/api/v1/conversations", "/api/v1/me/letters", "/api/v1/me/ai/controls",
		"/api/v1/me/memories", "/api/v1/me/privacy-preferences",
	} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, route, nil)
		// An old synthetic cookie cannot grant access without Dev Login.
		req.AddCookie(&http.Cookie{Name: "wellbeing_dev_session", Value: "synthetic"})
		api.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "SESSION_REQUIRED") {
			t.Fatalf("%s: mounted user API must fail closed, got %d %s", route, w.Code, w.Body.String())
		}
	}
	// Admin route is always mounted, but without Staff Auth it fails closed.
	w := httptest.NewRecorder()
	api.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/admin/me", nil))
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "AUTH_NOT_CONFIGURED") {
		t.Fatalf("admin route must fail closed without configured Staff Auth: %d %s", w.Code, w.Body.String())
	}
}

func TestAUTH09NoLegacyDevRouteOrCookieCanAuthenticate(t *testing.T) {
	for _, environment := range []string{"local", "stage", "staging", "staging-v2", "production"} {
		api, err := NewAPI(config.Config{Environment: environment}, nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, route := range []struct{ method, path string }{
			{"GET", "/api/v1/dev/users"}, {"POST", "/api/v1/dev/session"},
			{"DELETE", "/api/v1/dev/session"}, {"POST", "/api/v1/dev/identity-session"},
		} {
			w := httptest.NewRecorder()
			api.ServeHTTP(w, httptest.NewRequest(route.method, route.path, nil))
			if w.Code != 404 {
				t.Fatalf("%s %s accepted Dev route: %d", environment, route.path, w.Code)
			}
		}
		for _, path := range []string{"/api/v1/me", "/api/v1/conversations", "/api/v1/admin/me"} {
			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", path, nil)
			req.AddCookie(&http.Cookie{Name: "wellbeing_dev_session", Value: "synthetic"})
			api.ServeHTTP(w, req)
			want := 401
			if path == "/api/v1/admin/me" {
				want = 503
			}
			if w.Code != want {
				t.Fatalf("%s %s accepted old cookie: %d", environment, path, w.Code)
			}
		}
	}
}

func TestReservedOIDCEntrypointsFailClosed(t *testing.T) {
	api, err := NewAPI(config.Config{Environment: "local"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, actor := range []string{"user", "staff"} {
		for _, suffix := range []string{"login", "callback", "session"} {
			w := httptest.NewRecorder()
			api.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/"+actor+"/"+suffix, nil))
			if w.Code != 503 || !strings.Contains(w.Body.String(), "AUTH_NOT_CONFIGURED") {
				t.Fatalf("%s %s: expected fail-closed BFF: %d", actor, suffix, w.Code)
			}
		}
	}
}

func TestProductRoutesRequireServerResolvedSession(t *testing.T) {
	api, e := NewAPI(config.Config{Environment: "local"}, nil)
	if e != nil {
		t.Fatal(e)
	}
	const conv = "11111111-1111-4111-8111-111111111111"
	for _, tc := range []struct{ method, path string }{
		{"POST", "/api/v1/conversations/" + conv + "/turns"}, {"GET", "/api/v1/conversations/" + conv + "/turns/" + conv}, {"POST", "/api/v1/conversations/" + conv + "/turns/" + conv + "/retry"},
		{"POST", "/api/v1/me/ai/recover"}, {"GET", "/api/v1/me/ai/controls"}, {"PATCH", "/api/v1/me/ai/controls"}, {"GET", "/api/v1/me/memories"}, {"DELETE", "/api/v1/me/memories"}, {"PATCH", "/api/v1/me/memories/" + conv}, {"DELETE", "/api/v1/me/memories/" + conv}, {"POST", "/api/v1/me/memories/" + conv + "/corrections"}, {"GET", "/api/v1/me/ai/export"}, {"DELETE", "/api/v1/me"}, {"DELETE", "/api/v1/messages/" + conv},
	} {
		w := httptest.NewRecorder()
		api.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		if w.Code != 401 {
			t.Fatal(tc.method, tc.path, w.Code)
		}
	}
	for _, path := range []string{"/api/v1/me/daily-reflections", "/api/v1/me/ai-status", "/api/v1/me/ai/wakeup"} {
		w := httptest.NewRecorder()
		api.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 404 {
			t.Fatal(path, w.Code)
		}
	}
}
