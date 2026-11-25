package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"
)

type diagnosticTestError struct {
	cause  error
	secret string
	reason string
	origin string
	status int
}

func (e *diagnosticTestError) Error() string             { return "PRIVATE: " + e.secret }
func (e *diagnosticTestError) Unwrap() error             { return e.cause }
func (e *diagnosticTestError) DiagnosticReason() string  { return e.reason }
func (e *diagnosticTestError) DiagnosticCode() string    { return "provider_unavailable" }
func (e *diagnosticTestError) DiagnosticOrigin() string  { return e.origin }
func (e *diagnosticTestError) DiagnosticRetryable() bool { return true }
func (e *diagnosticTestError) DiagnosticStatus() int     { return e.status }

func fieldsMap(fields []any) map[string]any {
	m := make(map[string]any)
	for i := 0; i+1 < len(fields); i += 2 {
		m[fields[i].(string)] = fields[i+1]
	}
	return m
}

func TestSafeDiagnosticClassificationAndOrigin(t *testing.T) {
	err := fmt.Errorf("PRIVATE wrapped: %w", &diagnosticTestError{
		cause: errors.New("PRIVATE raw model output"), secret: "PRIVATE token",
		reason: "rate_limit", origin: "apps/api/internal/modules/ai/adapters/llmhttp/provider.go:123", status: 429,
	})
	got := fieldsMap(ErrorFields(err))
	if got["reason"] != "rate_limit" || got["retryable"] != true || got["provider_status"] != 429 ||
		got["error_code"] != "provider_unavailable" || got["error_origin"] != "apps/api/internal/modules/ai/adapters/llmhttp/provider.go:123" {
		t.Fatalf("incomplete diagnostics: %+v", got)
	}
	var output bytes.Buffer
	NewLogger(&output, slog.LevelDebug, time.UTC).Warn("probe", ErrorFields(err)...)
	if strings.Contains(output.String(), "PRIVATE") || strings.Contains(output.String(), "raw model") {
		t.Fatalf("private error data leaked: %s", output.String())
	}
}

func TestUnknownAndUntrustedDiagnosticsAreRedacted(t *testing.T) {
	err := &diagnosticTestError{secret: "secret", reason: "SECRET patient disclosure", origin: "https://a.invalid/private", status: 200}
	fields := fieldsMap(ErrorFields(err))
	if fields["reason"] != "unknown" {
		t.Fatal(fields)
	}
	if _, ok := fields["error_origin"]; ok {
		t.Fatal(fields)
	}
	if _, ok := fields["provider_status"]; ok {
		t.Fatal(fields)
	}
	if ErrorReason(errors.Join(err, context.DeadlineExceeded)) != "timeout" {
		t.Fatal("timeout hidden by wrapping")
	}
	if ErrorReason(context.Canceled) != "canceled" {
		t.Fatal("cancellation misclassified")
	}
	if got := SafeLabel("${SECRET}"); got != "" {
		t.Fatalf("unsafe label %s", got)
	}
}

func TestCorrelationContextNeverAcceptsPrivateStrings(t *testing.T) {
	const requestID = "abcdef12-1234-4abc-9abc-1234567890ab"
	ctx := WithRequestID(context.Background(), requestID)
	ctx = WithTurnRef(ctx, "0123456789abcdef")
	fields := fieldsMap(CorrelationFields(ctx))
	if fields["request_id"] != requestID || fields["turn_ref"] != "0123456789abcdef" {
		t.Fatal(fields)
	}
	if RequestID(WithRequestID(ctx, "SECRET patient")) != requestID {
		t.Fatal("malformed request id replaced trusted id")
	}
	if TurnRefFromContext(WithTurnRef(ctx, "SECRET")) != "0123456789abcdef" {
		t.Fatal("unsafe turn ref accepted")
	}
}

func TestSafeStackHasLocationsButNotArguments(t *testing.T) {
	secret := "SECRET private conversation"
	_ = secret
	stack := SafeStack(0)
	if strings.Contains(stack, "SECRET") || !strings.Contains(stack, "diagnostics_test.go:") {
		t.Fatal(stack)
	}
}

func TestConcurrentLoggerProducesIndependentJSONLines(t *testing.T) {
	var output bytes.Buffer
	logger := NewLogger(&output, slog.LevelDebug, time.UTC)
	const count = 48
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(index int) { defer wg.Done(); logger.Info("concurrent", "attempt", index) }(i)
	}
	wg.Wait()
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != count {
		t.Fatalf("want %d records, got %d", count, len(lines))
	}
	for _, line := range lines {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil || record["msg"] != "concurrent" {
			t.Fatalf("invalid concurrent record: %s %v", line, err)
		}
	}
}

// Mimics the stable numeric SQLError* interface exported by go-mssqldb.
// The strings are deliberately private; they must NEVER appear in a log.
type sqlDiagnosticTestError struct {
	number int32
	state  uint8
	class  uint8
}

func (e sqlDiagnosticTestError) Error() string {
	return "SECRET private SQL statement, user input and password"
}
func (e sqlDiagnosticTestError) SQLErrorNumber() int32 { return e.number }
func (e sqlDiagnosticTestError) SQLErrorState() uint8  { return e.state }
func (e sqlDiagnosticTestError) SQLErrorClass() uint8  { return e.class }

type operationDiagnosticTestError struct {
	cause     error
	operation string
	origin    string
}

func (e operationDiagnosticTestError) Error() string               { return "SECRET private operation data" }
func (e operationDiagnosticTestError) Unwrap() error               { return e.cause }
func (e operationDiagnosticTestError) DiagnosticOperation() string { return e.operation }
func (e operationDiagnosticTestError) DiagnosticOrigin() string    { return e.origin }

func TestSQLDiagnosticsArePreciseWithoutLeakingDriverMessage(t *testing.T) {
	cases := []struct {
		number    int32
		reason    string
		hint      string
		retryable any
	}{
		{207, "db_invalid_column", "verify_schema_migrations", false},
		{208, "db_missing_object", "verify_schema_migrations", false},
		{229, "db_permission_denied", "check_database_grants", false},
		{1205, "db_transaction_conflict", "retry_transaction_with_backoff", true},
		{2627, "db_duplicate_key", "check_unique_constraint", false},
		{50123, "db_error", "", nil},
	}
	for _, tc := range cases {
		// A real worker wraps errors at the SQL site and again as a
		// contract.Error; the classification must survive those layers.
		err := fmt.Errorf("SECRET wrapper: %w", operationDiagnosticTestError{
			cause:     sqlDiagnosticTestError{number: tc.number, state: 2, class: 16},
			operation: "ai_jobs.claim_ready",
			origin:    "apps/api/internal/modules/ai/adapters/mssql/memory_jobs.go:71",
		})
		fields := fieldsMap(DiagnosticFields(err, "apps/api/internal/modules/ai/application/memory/worker.go:52"))
		if fields["reason"] != tc.reason || fields["db_error_number"] != tc.number ||
			fields["db_error_state"] != uint8(2) || fields["db_error_class"] != uint8(16) ||
			fields["operation"] != "ai_jobs.claim_ready" || (tc.hint != "" && fields["diagnostic_hint"] != tc.hint) ||
			fields["retryable"] != tc.retryable || fields["error_origin"] != "apps/api/internal/modules/ai/adapters/mssql/memory_jobs.go:71" ||
			fields["error_site"] != "apps/api/internal/modules/ai/application/memory/worker.go:52" {
			t.Fatalf("sql number %d produced unexpected fields: %+v", tc.number, fields)
		}
		if tc.hint == "" {
			if _, ok := fields["diagnostic_hint"]; ok {
				t.Fatalf("unknown SQL number invents hint: %+v", fields)
			}
		}
		var output bytes.Buffer
		NewLogger(&output, slog.LevelDebug, time.UTC).Warn("ai.worker.retry", DiagnosticFields(err, "apps/api/internal/modules/ai/application/memory/worker.go:52")...)
		if strings.Contains(output.String(), "SECRET") || strings.Contains(output.String(), "password") {
			t.Fatalf("sql number %d leaked sensitive fields: %s", tc.number, output.String())
		}
	}
}

func TestUntrustedSQLOperationAndInvalidNumberAreNotLogged(t *testing.T) {
	err := operationDiagnosticTestError{
		cause:     sqlDiagnosticTestError{number: -99, state: 1, class: 1},
		operation: "SECRET-private-chat",
		origin:    "C:/private/error:123",
	}
	got := fieldsMap(ErrorFields(err))
	for _, name := range []string{"operation", "error_origin", "db_error_number", "db_error_state", "db_error_class", "diagnostic_hint"} {
		if _, ok := got[name]; ok {
			t.Errorf("untrusted diagnostic field %q appeared: %+v", name, got)
		}
	}
}

func TestSourceDetailsSurviveTranslationAndJoin(t *testing.T) {
	inner := Detail(errors.New("PRIVATE disclosure"), "consent_missing", "recent_source_use", "source_use")
	outer := Detail(errors.Join(errors.New("PRIVATE wrapper"), inner), "policy_query_failed", "context_state", "verify_generation")
	got := fieldsMap(ErrorFields(outer))
	if got["reason"] != "consent_missing" || got["failed_check"] != "recent_source_use" || got["diagnostic_hint"] != "check_source_use_consent_and_mapping" {
		t.Fatal(got)
	}
	var b bytes.Buffer
	NewLogger(&b, slog.LevelDebug, time.UTC).Warn("test", ErrorFields(outer)...)
	if strings.Contains(b.String(), "PRIVATE") {
		t.Fatal(b.String())
	}
	sqlErr := sqlDiagnosticTestError{number: 229, state: 1, class: 16}
	if got := fieldsMap(ErrorFields(Detail(sqlErr, "context_query_failed", "turn_admission", "verify_context"))); got["reason"] != "db_permission_denied" || got["failed_check"] != "turn_admission" {
		t.Fatal(got)
	}
}

func TestDeepestSourceWinsOverSiblingTranslation(t *testing.T) {
	original := Detail(Detail(errors.New("PRIVATE"), "consent_missing", "recent_source_use", "source_use"), "context_query_failed", "context_state", "verify_generation")
	sibling := Detail(errors.New("PRIVATE"), "policy_denied", "job_validation", "memory_access")
	fields := fieldsMap(ErrorFields(errors.Join(original, sibling)))
	if fields["failed_check"] != "recent_source_use" || fields["reason"] != "consent_missing" {
		t.Fatal(fields)
	}
}
