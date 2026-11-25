package observability

// Detail annotates a cause without copying its possibly private message. Callers
// supply code-owned labels; the logging boundary validates each vocabulary.
// Use only at the source of a check, never to guess why a dependency failed.
func Detail(cause error, reason, check, operation string) error {
	return &detail{cause: cause, reason: reason, check: check, operation: operation, origin: Source(1)}
}

type detail struct {
	cause                            error
	reason, check, operation, origin string
}

func (e *detail) Error() string               { return "internal operation failed" }
func (e *detail) Unwrap() error               { return e.cause }
func (e *detail) DiagnosticReason() string    { return e.reason }
func (e *detail) DiagnosticCheck() string     { return e.check }
func (e *detail) DiagnosticOperation() string { return e.operation }
func (e *detail) DiagnosticOrigin() string    { return e.origin }

// DiagnosticValue walks inward, including joined causes, preferring the deepest
// nonempty value. Translation wrappers must not hide source diagnostics.
func diagnosticValue(err error, value func(error) string) string {
	var result string
	bestDepth, visited := -1, 0
	var visit func(error, int)
	visit = func(e error, depth int) {
		if e == nil || depth > 16 || visited >= 64 {
			return
		}
		visited++
		if v := value(e); v != "" && depth > bestDepth {
			result, bestDepth = v, depth
		}
		switch e := e.(type) {
		case interface{ Unwrap() []error }:
			for _, child := range e.Unwrap() {
				visit(child, depth+1)
			}
		case interface{ Unwrap() error }:
			visit(e.Unwrap(), depth+1)
		}
	}
	visit(err, 0)
	return result
}

// DatabaseError adds a stable operation at the SQL adapter boundary. The
// underlying driver remains intact and only its numeric metadata is logged.
func DatabaseError(err error, operation string) error {
	if err == nil {
		return nil
	}
	return &detail{cause: err, operation: operation, origin: Source(1)}
}
