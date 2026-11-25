package httpx

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"
)

// RequestLog records ONE bounded event per request. Paths are Gin route
// templates: never the raw URL, query, headers, IP, body, or session identity.
func RequestLog(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		status := c.Writer.Status()
		level := slog.LevelInfo
		switch {
		case status >= 500:
			level = slog.LevelError
		case status >= 400:
			level = slog.LevelWarn
		case c.FullPath() == "/health/live" || c.FullPath() == "/health/ready":
			level = slog.LevelDebug
		}
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		args := []any{
			"request_id", c.GetString("requestId"), "method", c.Request.Method,
			"route", route, "status", status, "duration_ms", time.Since(started).Milliseconds(),
		}
		for _, name := range []string{"failure_code", "error_type", "cause_type", "root_cause_type", "error_kind", "error_site", "error_origin", "reason", "error_code", "retryable", "provider_status", "ai_step", "ai_code", "turn_ref", "operation", "diagnostic_hint", "db_error_number", "db_error_state", "db_error_class", "failed_check", "failure_stage", "http_error_kind", "provider_error_code", "timeout_category"} {
			if v, ok := c.Get(name); ok {
				args = append(args, name, v)
			}
		}
		if c.GetBool("panic_recovered") {
			args = append(args, "panic_recovered", true)
		}
		logger.Log(c.Request.Context(), level, "http.request", args...)
	}
}

// Recovery intentionally does not log the panic value: it may contain private
// model output, SQL parameters or secrets. A code-only stack trace is safe.
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if value := recover(); value != nil {
				route := c.FullPath()
				if route == "" {
					route = "unmatched"
				}
				c.Set("panic_recovered", true)
				logger.ErrorContext(c.Request.Context(), "http.panic", "request_id", c.GetString("requestId"),
					"route", route, "method", c.Request.Method, "panic_type", observability.Type(value),
					"stack", observability.SafeStack(1))
				if !c.Writer.Written() {
					Failure(c, http.StatusInternalServerError, "INTERNAL_ERROR")
				}
				c.Abort()
			}
		}()
		c.Next()
	}
}

// FailureWithError preserves the existing sanitized HTTP contract while making
// the failure site and the underlying error *types* searchable in server logs.
func FailureWithError(c *gin.Context, status int, code string, err error) {
	if err != nil {
		c.Set("error_site", observability.Source(1))
		details := observability.ErrorFields(err)
		for i := 0; i+1 < len(details); i += 2 {
			c.Set(details[i].(string), details[i+1])
		}
	}
	Failure(c, status, code)
}
