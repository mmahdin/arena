package composition

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/config"
)

func TestAIStatusReportsOnlyConfiguredProviderAndModel(t *testing.T) {
	for _, provider := range []string{"mock", "openai-compatible"} {
		r := gin.New()
		r.GET("/status", aiStatus(config.Config{LLMProvider: provider, ModelName: "gpt-5.6-sol", ModelAPIKey: "synthetic-secret-never-return"}))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/status", nil))
		var data map[string]any
		if json.Unmarshal(w.Body.Bytes(), &data) != nil || len(data) != 4 || data["provider"] != provider || data["mock"] != (provider == "mock") || strings.Contains(w.Body.String(), "synthetic-secret") {
			t.Fatal("status missing configuration or exposing private configuration")
		}
		if provider == "mock" && data["model"] != nil {
			t.Fatal("mock status falsely claims a configured external model")
		}
		if provider != "mock" && data["model"] != "gpt-5.6-sol" {
			t.Fatal("status changed the selected model")
		}
	}
}

func TestDevEndpointsNotMountedWhenDisabled(t *testing.T) {
	handler, err := NewAPI(config.Config{Environment: "staging", AllowDevLogin: false}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range []string{"/api/v1/dev/users", "/api/v1/me", "/api/v1/conversations", "/api/v1/me/letters", "/api/v1/me/ai-status"} {
		req := httptest.NewRequest(http.MethodGet, route, nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("%s: expected disabled dev endpoint, got %d", route, w.Code)
		}
	}
}

func TestStagingDevSessionsMountProtectedRoutes(t *testing.T) {
	for _, environment := range []string{"local", "stage", "staging", "staging-v2"} {
		t.Run(environment, func(t *testing.T) {
			handler, err := NewAPI(config.Config{
				Environment: environment, AllowDevLogin: true,
				DevSessionSecret: strings.Repeat("s", 64), CookieSecure: true,
			}, nil)
			if err != nil {
				t.Fatal(err)
			}
			for _, route := range []string{"/api/v1/me", "/api/v1/conversations", "/api/v1/me/bookmarks", "/api/v1/me/letters", "/api/v1/me/ai-status"} {
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, route, nil))
				if w.Code != http.StatusUnauthorized {
					t.Fatalf("%s: expected a mounted route protected by the session guard, got %d", route, w.Code)
				}
			}
		})
	}
}

func TestProductionRejectsDevSessions(t *testing.T) {
	_, err := NewAPI(config.Config{
		Environment: "production", AllowDevLogin: true,
		DevSessionSecret: strings.Repeat("s", 64),
	}, nil)
	if err == nil {
		t.Fatal("production must not mount test identity routes")
	}
}
