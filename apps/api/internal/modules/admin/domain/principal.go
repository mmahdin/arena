// Package domain defines the authenticated staff-facing model used by Admin
// application services. Staff authentication and roles remain owned by Auth.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// AdminRole is the single fixed MVP capability level. All active staff admins
// can access every implemented Admin API; there is no grants/roles table.
const AdminRole = "admin"

// Principal is constructed from an already authenticated staff BFF session,
// not a request header, query parameter, or browser-supplied user ID.
type Principal struct {
	StaffID   uuid.UUID
	ExpiresAt time.Time
}

type CurrentStaff struct {
	StaffID   uuid.UUID
	Role      string
	Active    bool
	ExpiresAt time.Time
}
