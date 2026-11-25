package config

import (
	"net/url"
	"os"
	"strings"
	"testing"
)

func TestSplitDatabaseSettingsEscapeCredentials(t *testing.T) {
	cleanConfigEnv(t)
	setSplitDatabaseEnv(t)
	t.Setenv("DATABASE_URL", "sqlserver://old:synthetic@old.example.invalid?database=old")
	raw, err := LoadDatabaseURL()
	if err != nil {
		t.Fatal(err)
	}
	dsn, err := url.Parse(raw)
	if err != nil {
		t.Fatal("could not parse generated SQL Server URL")
	}
	password, _ := dsn.User.Password()
	if dsn.Scheme != "sqlserver" || dsn.Hostname() != os.Getenv("DATABASE_HOST") || dsn.Port() != "1433" ||
		dsn.User.Username() != os.Getenv("DATABASE_USERNAME") || password != os.Getenv("DATABASE_PASSWORD") ||
		dsn.Query().Get("database") != os.Getenv("DATABASE_NAME") || dsn.Query().Get("encrypt") != "disable" {
		t.Fatal("split database configuration did not round-trip")
	}
}

func TestSplitDatabaseDefaultsAndIPv6(t *testing.T) {
	cleanConfigEnv(t)
	setSplitDatabaseEnv(t)
	t.Setenv("DATABASE_HOST", "::1")
	t.Setenv("DATABASE_PORT", "")
	t.Setenv("DATABASE_ENCRYPT", "")
	raw, err := LoadDatabaseURL()
	if err != nil {
		t.Fatal(err)
	}
	dsn, _ := url.Parse(raw)
	if dsn.Host != "[::1]:1433" || dsn.Query().Get("encrypt") != "disable" {
		t.Fatal("SQL Server port/encryption defaults or IPv6 handling changed")
	}
}

func TestSplitDatabaseRejectsInvalidSettingsWithoutLeakingSecrets(t *testing.T) {
	for _, tc := range []struct{ key, value string }{
		{"DATABASE_HOST", ""}, {"DATABASE_USERNAME", ""}, {"DATABASE_PASSWORD", ""}, {"DATABASE_NAME", ""},
		{"DATABASE_PORT", "wrong"}, {"DATABASE_PORT", "0"}, {"DATABASE_PORT", "65536"},
		{"DATABASE_HOST", "https://db.example.invalid"}, {"DATABASE_HOST", "db.example.invalid:1433"},
		{"DATABASE_ENCRYPT", "unsupported"},
	} {
		t.Run(tc.key+tc.value, func(t *testing.T) {
			cleanConfigEnv(t)
			setSplitDatabaseEnv(t)
			t.Setenv("DATABASE_URL", "sqlserver://old:synthetic@old.example.invalid?database=old")
			t.Setenv(tc.key, tc.value)
			_, err := LoadDatabaseURL()
			if err == nil || !strings.Contains(err.Error(), tc.key) {
				t.Fatal("invalid split setting must fail instead of falling back to a stale URL")
			}
			if secret := os.Getenv("DATABASE_PASSWORD"); secret != "" && strings.Contains(err.Error(), secret) {
				t.Fatal("database password leaked in configuration error")
			}
		})
	}
}

func TestLegacyDatabaseURLValidation(t *testing.T) {
	for _, raw := range []string{"", "postgres://user:synthetic@db.example.invalid/database", "sqlserver://", "sqlserver://user:synthetic@db.example.invalid/#fragment", "sqlserver://%invalid"} {
		t.Run(raw, func(t *testing.T) {
			cleanConfigEnv(t)
			t.Setenv("DATABASE_URL", raw)
			if _, err := LoadDatabaseURL(); err == nil || strings.Contains(err.Error(), "synthetic") {
				t.Fatal("invalid legacy database URL must fail with a sanitized error")
			}
		})
	}
}
