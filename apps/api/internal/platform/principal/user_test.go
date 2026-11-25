package principal

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestOnlyTrustedMiddlewareCanSetUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	if got := UserID(c); got != uuid.Nil {
		t.Fatal("unauthenticated user ID must be empty")
	}
	SetUserID(c, uuid.Nil)
	if got := UserID(c); got != uuid.Nil {
		t.Fatal("nil user principal accepted")
	}
	id := uuid.New()
	SetUserID(c, id)
	if got := UserID(c); got != id {
		t.Fatal("verified internal UUID lost")
	}
}
