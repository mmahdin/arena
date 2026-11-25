package httpadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

func TestOnlyFinalResultsExposeText(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, status := range []c.TurnStatus{c.TurnQueued, c.TurnRunning, c.TurnFailed, c.TurnCancelled, c.TurnCompleted} {
		t.Run(string(status), func(t *testing.T) {
			r := gin.New()
			r.GET("/", func(ctx *gin.Context) {
				assistant := c.NewID()
				result(ctx, c.TurnResult{TurnID: c.NewID(), UserMessageID: c.NewID(), AssistantMessageID: &assistant, Text: "SECRET_UNVALIDATED_DRAFT", Status: status}, false)
			})
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
			var got map[string]any
			if json.Unmarshal(w.Body.Bytes(), &got) != nil {
				t.Fatal("json")
			}
			if status != c.TurnCompleted && (got["text"] != nil || got["assistantMessageId"] != nil) {
				t.Fatal("draft exposed")
			}
			expected := 200
			if status == c.TurnQueued || status == c.TurnRunning {
				expected = 202
			}
			if w.Code != expected {
				t.Fatal(w.Code)
			}
		})
	}
}
func TestPublicErrorsNeverExposeCause(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{&c.Error{Code: c.ErrProvider, Cause: errors.New("SECRET SQL PROMPT")}, 503, "AI_CORE_UNAVAILABLE"},
		{&c.Error{Code: c.ErrDuplicate}, 409, "IDEMPOTENCY_CONFLICT"},
		{&c.Error{Code: c.ErrRateLimited}, 429, "RATE_LIMITED"},
		{&c.Error{Code: c.ErrValidation}, 400, "VALIDATION_ERROR"},
		{mem.ErrDenied, 404, "RESOURCE_NOT_FOUND"}, {mem.ErrConflict, 409, "STATE_CONFLICT"},
		{httpx.ErrNotFound, 404, "RESOURCE_NOT_FOUND"}, {context.DeadlineExceeded, 503, "TURN_INTERRUPTED"},
	} {
		r := gin.New()
		r.GET("/", func(ctx *gin.Context) { failure(ctx, tc.err) })
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		var p httpx.Problem
		_ = json.Unmarshal(w.Body.Bytes(), &p)
		if w.Code != tc.status || p.Code != tc.code || strings.Contains(w.Body.String(), "SECRET") {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}
func TestUnavailableCoreNeverFallsBackToPrototype(t *testing.T) {
	h := New(nil)
	r := gin.New()
	r.POST("/conversations/:id/turns", h.Turn)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/conversations/"+c.NewID().String()+"/turns", strings.NewReader(`{"message":"synthetic"}`)))
	if w.Code != 503 {
		t.Fatal(w.Code)
	}
}

func TestGeneratorDenialHTTP404IsSanitizedAndDiagnosed(t *testing.T) {
	var b bytes.Buffer
	r := gin.New()
	r.Use(httpx.RequestLog(observability.NewLogger(&b, slog.LevelDebug, time.UTC)), httpx.Middleware("http://localhost"))
	r.GET("/turn", func(ctx *gin.Context) {
		ctx.Set("turn_ref", "0123456789abcdef")
		failure(ctx, &c.Error{Code: c.ErrForbidden, Step: "generator_sources", Cause: &c.Error{Code: c.ErrForbidden, Step: "generator_context", Reason: "consent_missing", Check: "recent_source_use", Origin: "apps/api/internal/modules/ai/adapters/mssql/generator_context.go:66", Cause: errors.New("PRIVATE token and conversation")}})
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/turn", nil))
	if w.Code != 404 || strings.Contains(w.Body.String(), "consent") || strings.Contains(w.Body.String(), "PRIVATE") {
		t.Fatal(w.Code, w.Body.String())
	}
	var record map[string]any
	if err := json.Unmarshal(b.Bytes(), &record); err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]any{"status": float64(404), "http_error_kind": "authorization_concealed", "failure_stage": "generator", "failed_check": "recent_source_use", "reason": "consent_missing", "failure_code": "RESOURCE_NOT_FOUND"} {
		if record[k] != want {
			t.Fatalf("%s: %v", k, record)
		}
	}
	if record["request_id"] == "" || strings.Contains(b.String(), "PRIVATE") {
		t.Fatal(b.String())
	}
}

type diagnosticTurnBoundary struct{ cause error }

func (e diagnosticTurnBoundary) Error() string          { return "turn failed" }
func (e diagnosticTurnBoundary) Unwrap() error          { return e.cause }
func (diagnosticTurnBoundary) PreservePublicCode() bool { return true }
func TestPreservedInternalCauseDoesNotChangePublicMapping(t *testing.T) {
	for _, cause := range []error{context.DeadlineExceeded, httpx.ErrNotFound, mem.ErrDenied} {
		r := gin.New()
		r.GET("/", func(ctx *gin.Context) {
			failure(ctx, diagnosticTurnBoundary{&c.Error{Code: c.ErrValidation, Step: "generator", Cause: cause}})
		})
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		if w.Code != 400 || !strings.Contains(w.Body.String(), "VALIDATION_ERROR") {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}
