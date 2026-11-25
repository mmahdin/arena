// Package principal exposes verified request-scoped identities without coupling
// product modules to the temporary Dev Session authentication mechanism.
package principal

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const userKey = "resolvedUserID"

// SetUserID is called only by trusted middleware after successful verification.
// It must never receive a user-provided header, path parameter or JSON field.
func SetUserID(c *gin.Context, id uuid.UUID) {
	if id != uuid.Nil {
		c.Set(userKey, id)
	}
}

func UserID(c *gin.Context) uuid.UUID {
	value, ok := c.Get(userKey)
	if !ok {
		return uuid.Nil
	}
	id, ok := value.(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}
