package httpadapter

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/principal"
)

// verifiedUserForTests simulates a trusted BFF guard in *unit tests only*.
// It is not exported, registered in application composition or compiled into
// the production binary. A client-supplied header never authenticates users.
func verifiedUserForTests(userID uuid.UUID) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("X-Verified-User-For-Tests") != userID.String() {
			httpx.Failure(c, http.StatusUnauthorized, "SESSION_REQUIRED")
			c.Abort()
			return
		}
		principal.SetUserID(c, userID)
		c.Next()
	}
}
