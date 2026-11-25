package httpadapter

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

func TestBookmarkFailuresAreSanitized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{httpx.ErrNotFound, 404, "RESOURCE_NOT_FOUND"},
		{httpx.ErrInvalid, 400, "VALIDATION_ERROR"},
		{errors.New("SQL secret private fixture content"), 503, "DB_UNAVAILABLE"},
	} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		failure(c, tc.err)
		if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.code) || strings.Contains(w.Body.String(), "secret") {
			t.Fatalf("unsafe bookmark error response, status %d", w.Code)
		}
	}
}
