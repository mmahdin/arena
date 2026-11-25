package mssql

import "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"

// sqlOperationFailure adds an internal operation and exact error construction
// site without exposing SQL text, parameters or the driver's error message.
// The underlying driver error remains available through errors.Is/As, and
// observability reads its SQLErrorNumber/State/Class methods directly.
type sqlOperationFailure struct {
	cause     error
	operation string
	origin    string
}

func (e *sqlOperationFailure) Error() string               { return "database operation failed" }
func (e *sqlOperationFailure) Unwrap() error               { return e.cause }
func (e *sqlOperationFailure) DiagnosticOperation() string { return e.operation }
func (e *sqlOperationFailure) DiagnosticOrigin() string    { return e.origin }

func sqlAt(err error, operation string) error {
	if err == nil {
		return nil
	}
	return &sqlOperationFailure{cause: err, operation: operation, origin: observability.Source(1)}
}
