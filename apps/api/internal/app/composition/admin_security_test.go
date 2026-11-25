package composition

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/config"
)

// This checks the actual composition routing (not a second handcrafted Admin
// router). A new handler accidentally mounted outside the Staff guard is a
// release-blocking regression even when its individual unit tests pass.
func TestBO10EveryComposedAdminRouteFailsClosedWithoutStaffBFF(t *testing.T) {
	expected := map[string]string{
		"GET /api/v1/admin/me":                             "/api/v1/admin/me",
		"GET /api/v1/admin/audit":                          "/api/v1/admin/audit",
		"GET /api/v1/admin/overview":                       "/api/v1/admin/overview",
		"GET /api/v1/admin/turns":                          "/api/v1/admin/turns",
		"GET /api/v1/admin/turns/:turnId":                  "/api/v1/admin/turns/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		"GET /api/v1/admin/turns/:turnId/trace-references": "/api/v1/admin/turns/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa/trace-references",
		"GET /api/v1/admin/jobs":                           "/api/v1/admin/jobs",
		"GET /api/v1/admin/workers":                        "/api/v1/admin/workers",
		"GET /api/v1/admin/bundles":                        "/api/v1/admin/bundles",
		"GET /api/v1/admin/evaluations":                    "/api/v1/admin/evaluations",
		"GET /api/v1/admin/evaluations/:runId":             "/api/v1/admin/evaluations/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		"POST /api/v1/admin/evaluations/import":            "/api/v1/admin/evaluations/import",
	}
	for _, environment := range []string{"local", "production"} {
		cfg := config.Config{Environment: environment}
		api, err := NewAPI(cfg, nil)
		if err != nil {
			t.Fatal(err)
		}
		engine, ok := api.Handler.(*gin.Engine)
		if !ok {
			t.Fatalf("expected Gin engine, got %T", api.Handler)
		}
		routes := make(map[string]struct{})
		for _, route := range engine.Routes() {
			if strings.HasPrefix(route.Path, "/api/v1/admin/") {
				routes[route.Method+" "+route.Path] = struct{}{}
			}
		}
		if len(routes) != len(expected) {
			t.Fatalf("env=%s: expected exactly %d guarded Admin routes, got %v", environment, len(expected), routes)
		}
		for methodPath, requestPath := range expected {
			if _, exists := routes[methodPath]; !exists {
				t.Fatalf("env=%s: missing Admin route %s", environment, methodPath)
			}
			method := strings.SplitN(methodPath, " ", 2)[0]
			req := httptest.NewRequest(method, requestPath+"?role=admin&staffId=spoof", strings.NewReader(`{"role":"admin"}`))
			req.Host = "admin.localhost:8080"
			req.AddCookie(&http.Cookie{Name: "wellbeing_dev_session", Value: "synthetic"})
			req.Header.Set("X-Staff-ID", "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")
			req.Header.Set("Authorization", "Bearer forged")
			w := httptest.NewRecorder()
			api.ServeHTTP(w, req)
			if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), "AUTH_NOT_CONFIGURED") || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("env=%s: %s exposed without BFF: %d %s", environment, methodPath, w.Code, w.Body.String())
			}
		}
	}
}
