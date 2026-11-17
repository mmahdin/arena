package mssql

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// parseUUIDStrict converts the canonical varchar(36) form SELECTed from
// SQL Server back into the contract UUID type. Any other representation is an
// error: identifiers must never be guessed or partially trusted.
func parseUUIDStrict(raw string) (contract.ID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, fmt.Errorf("mssql: invalid uuid %q", raw)
	}
	return id, nil
}
