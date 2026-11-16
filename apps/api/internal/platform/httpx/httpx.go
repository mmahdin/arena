package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Problem struct {
	Type      string `json:"type"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Code      string `json:"code"`
	RequestID string `json:"requestId,omitempty"`
}

func Failure(c *gin.Context, status int, code string) {
	title := http.StatusText(status)
	if title == "" {
		title = "Error"
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Content-Type", "application/problem+json")
	c.JSON(status, Problem{Type: "about:blank", Title: title, Status: status, Code: code})
}

func StrictJSON(c *gin.Context, dst interface{}) bool {
	if c.ContentType() != "application/json" {
		Failure(c, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE")
		return false
	}
	if c.Request.ContentLength > 16*1024 {
		Failure(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE")
		return false
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16*1024)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		Failure(c, 400, "INVALID_JSON")
		return false
	}
	// A request contains one JSON value; reject appended objects or garbage.
	if err := decoder.Decode(new(interface{})); err != io.EOF {
		Failure(c, 400, "INVALID_JSON")
		return false
	}
	return true
}

func ErrStatus(err error) (int, string) {
	if err == nil {
		return http.StatusOK, ""
	}
	switch {
	case errors.Is(err, ErrNotFound):
		return 404, "RESOURCE_NOT_FOUND"
	case errors.Is(err, ErrOnboardingRequired):
		return 409, "ONBOARDING_REQUIRED"
	case errors.Is(err, ErrConflict):
		return 409, "TURN_ALREADY_PROCESSING"
	case errors.Is(err, ErrInvalid):
		return 400, "VALIDATION_ERROR"
	case errors.Is(err, ErrUnavailable):
		return 503, "AI_UNAVAILABLE"
	default:
		return 500, "INTERNAL_ERROR"
	}
}

var (
	ErrNotFound           = errors.New("not found")
	ErrOnboardingRequired = errors.New("onboarding required")
	ErrConflict           = errors.New("conflict")
	ErrInvalid            = errors.New("invalid")
	ErrUnavailable        = errors.New("unavailable")
)

func Middleware(allowedOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Cache-Control", "no-store")
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead && c.Request.Method != http.MethodOptions {
			origin := c.GetHeader("Origin")
			if origin != "" && origin != allowedOrigin && !sameOrigin(origin, c.Request.Host) {
				Failure(c, 403, "CSRF_ORIGIN_DENIED")
				c.Abort()
				return
			}
		}
		c.Next()
	}
}
func sameOrigin(origin, host string) bool {
	return strings.EqualFold(origin, "http://"+host) || strings.EqualFold(origin, "https://"+host)
}
