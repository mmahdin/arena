package contract

import (
	"github.com/google/uuid"
)

// Alias keeps all §4 contracts compatible with the existing backend's UUIDs.
// Typed ID wrappers may be introduced inside individual modules if justified,
// but no prefix/ULID encoding is added to storage or network contracts.
type ID = uuid.UUID

func NewID() ID          { return uuid.New() }
func ValidID(id ID) bool { return id != uuid.Nil }
