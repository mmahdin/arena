package migrations

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestSequentialMigrationNames(t *testing.T) {
	for _, name := range []string{"0001_initial_schema.sql", "0002_synthetic_dev_users.sql", "0040_add_index.sql"} {
		if !migrationName.MatchString(name) {
			t.Fatalf("expected valid sequential migration: %s", name)
		}
	}
	for _, name := range []string{"abc_test.sql", "1_test.sql", "0003_bad-name.sql", "0004_test.sql.bak", "0005_Test.sql"} {
		if migrationName.MatchString(name) {
			t.Fatalf("expected invalid migration name: %s", name)
		}
	}
}

// Regression: the former 0014 contained its own transaction delimiters, which
// the real runner rejects. Rewriting already-saved v2 cursors also reset users.
func TestV2CursorMigrationIsRunnableAndNonDestructive(t *testing.T) {
	raw, err := os.ReadFile("../../../db/migrations/0014_onboarding_v2_cursor.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToUpper(string(raw))
	if regexp.MustCompile(`(?m)^\s*GO\s*$`).MatchString(sql) ||
		regexp.MustCompile(`(?i)\b(BEGIN\s+TRAN(SACTION)?|COMMIT\s*(TRAN(SACTION)?)?|ROLLBACK\s*(TRAN(SACTION)?)?)\b`).MatchString(sql) {
		t.Fatal("0014 must let the migration runner manage transactions")
	}
	if strings.Contains(sql, "UPDATE DBO.USER_PROFILES") {
		t.Fatal("0014 must preserve previously saved v2 progress")
	}
	if !strings.Contains(sql, "CHECK (ONBOARDING_STEP BETWEEN 0 AND 25)") {
		t.Fatal("0014 must widen the cursor constraint to the new 26-page flow")
	}
}
