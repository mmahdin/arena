package mssql

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	driver "github.com/microsoft/go-mssqldb"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"
)

func TestClaimSQLDiagnosticSurvivesContractWrapping(t *testing.T) {
	underlying := driver.Error{Number: 207, Class: 16, State: 1, Message: "SECRET invalid column and SQL"}
	wrapped := memorySQLError(sqlAt(fmt.Errorf("private statement: %w", underlying), "ai_jobs.claim_ready"))
	var core *c.Error
	if !errors.As(wrapped, &core) || core.Code != c.ErrInternal {
		t.Fatalf("lost application error classification: %T", wrapped)
	}
	var sqlErr driver.Error
	if !errors.As(wrapped, &sqlErr) || sqlErr.Number != 207 {
		t.Fatal("lost original driver cause")
	}
	fields := observability.ErrorFields(wrapped)
	found := map[string]any{}
	for i := 0; i+1 < len(fields); i += 2 {
		found[fields[i].(string)] = fields[i+1]
	}
	if found["reason"] != "db_invalid_column" || found["operation"] != "ai_jobs.claim_ready" || found["db_error_number"] != int32(207) || found["diagnostic_hint"] != "verify_schema_migrations" {
		t.Fatalf("missing diagnostics: %+v", found)
	}
	if origin, ok := found["error_origin"].(string); !ok || !strings.Contains(origin, "error_diagnostics_test.go:") {
		t.Fatalf("SQL observation source not preserved: %+v", found)
	}
	for _, field := range fields {
		if s, ok := field.(string); ok && strings.Contains(s, "SECRET") {
			t.Fatalf("private SQL entered diagnostics: %+v", fields)
		}
	}
}
