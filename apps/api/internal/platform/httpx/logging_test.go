package httpx

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"
)

func TestRequestLoggerCorrelatesFailureWithoutPrivateInput(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var logs bytes.Buffer
	logger := observability.NewLogger(&logs, slog.LevelDebug, time.UTC)
	r := gin.New()
	r.Use(RequestLog(logger), Recovery(logger), Middleware("http://localhost:5173"))
	r.GET("/items/:id", func(c *gin.Context) {
		FailureWithError(c, 503, "DB_UNAVAILABLE", errors.New("SECRET health text and SQL password"))
	})
	request := httptest.NewRequest(http.MethodGet, "/items/secret-id?token=VERY_SECRET", nil)
	request.Header.Set("Authorization", "Bearer SECRET")
	response := httptest.NewRecorder()
	r.ServeHTTP(response, request)
	if response.Code != 503 {
		t.Fatal(response.Code)
	}
	var problem Problem
	if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
		t.Fatal(err)
	}
	if problem.RequestID == "" || response.Header().Get("X-Request-ID") != problem.RequestID {
		t.Fatal("missing matching request id")
	}
	if strings.Contains(logs.String(), "SECRET") || strings.Contains(logs.String(), "secret-id") || strings.Contains(logs.String(), "VERY_SECRET") {
		t.Fatalf("private values leaked: %s", logs.String())
	}
	var event map[string]any
	if err := json.Unmarshal(logs.Bytes(), &event); err != nil {
		t.Fatal(err)
	}
	for k, value := range map[string]any{"route": "/items/:id", "request_id": problem.RequestID, "status": float64(503), "failure_code": "DB_UNAVAILABLE", "error_type": "*errors.errorString"} {
		if event[k] != value {
			t.Fatalf("%s = %v (wanted %v)", k, event[k], value)
		}
	}
	if !strings.Contains(event["error_site"].(string), "logging_test.go:") {
		t.Fatalf("bad error site: %v", event)
	}
}

func TestRecoveryLogsCodeOnlyStackAndPreservesErrorResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var logs bytes.Buffer
	logger := observability.NewLogger(&logs, slog.LevelDebug, time.UTC)
	r := gin.New()
	r.Use(RequestLog(logger), Recovery(logger), Middleware(""))
	r.GET("/panic", func(*gin.Context) { panic("SECRET patient message") })
	response := httptest.NewRecorder()
	r.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/panic?private=SECRET", nil))
	if response.Code != 500 {
		t.Fatalf("status %d", response.Code)
	}
	var p Problem
	if err := json.Unmarshal(response.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.Code != "INTERNAL_ERROR" || p.RequestID == "" {
		t.Fatalf("unexpected body %+v", p)
	}
	if strings.Contains(logs.String(), "SECRET") || !strings.Contains(logs.String(), "panic_recovered") || !strings.Contains(logs.String(), "logging_test.go:") {
		t.Fatalf("unsafe or insufficient panic diagnostic: %s", logs.String())
	}
}

func TestRequestLoggerAvoidsUnknownPathsAndSuccessfulHealthNoise(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var logs bytes.Buffer
	logger := observability.NewLogger(&logs, slog.LevelInfo, time.UTC)
	r := gin.New()
	r.Use(RequestLog(logger), Recovery(logger), Middleware(""))
	r.GET("/health/live", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	r.NoRoute(func(c *gin.Context) { Failure(c, 404, "RESOURCE_NOT_FOUND") })
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/health/live", nil))
	if logs.Len() != 0 {
		t.Fatalf("health logged at info level: %s", logs.String())
	}
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/unknown/secret?body=sensitive", nil))
	if strings.Contains(logs.String(), "secret") || !strings.Contains(logs.String(), "unmatched") {
		t.Fatalf("invalid unmatched route diagnostic: %s", logs.String())
	}
}

// This fake has the same numeric-only interface as the MSSQL driver's error.
type requestSQLTestError struct{}

func (requestSQLTestError) Error() string         { return "SECRET SQL parameters and credentials" }
func (requestSQLTestError) SQLErrorNumber() int32 { return 229 }
func (requestSQLTestError) SQLErrorState() uint8  { return 5 }
func (requestSQLTestError) SQLErrorClass() uint8  { return 14 }

func TestHTTPFailureIncludesSafeDatabaseReason(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var logs bytes.Buffer
	r := gin.New()
	r.Use(RequestLog(observability.NewLogger(&logs, slog.LevelDebug, time.UTC)), Middleware(""))
	r.POST("/api/v1/private/:id", func(c *gin.Context) {
		FailureWithError(c, 503, "DB_UNAVAILABLE", requestSQLTestError{})
	})
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/api/v1/private/private-value", nil))
	var event map[string]any
	if err := json.Unmarshal(logs.Bytes(), &event); err != nil {
		t.Fatal(err)
	}
	if event["reason"] != "db_permission_denied" || event["db_error_number"] != float64(229) ||
		event["db_error_state"] != float64(5) || event["diagnostic_hint"] != "check_database_grants" {
		t.Fatalf("missing DB classification in request event: %+v", event)
	}
	if strings.Contains(logs.String(), "SECRET") || strings.Contains(logs.String(), "private-value") {
		t.Fatalf("unsafe request log: %s", logs.String())
	}
}
