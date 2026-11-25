package config

import (
	"strings"
	"testing"
)

func TestSafeLogExportExplicitLocalOnly(t *testing.T) {
	for _, tc := range []struct {
		name, env, flag, path string
		allowed               bool
	}{
		{"off", "local", "false", "", true},
		{"local-valid", "local", "true", "/var/log/theia/theia-safe.jsonl", true},
		{"staging-denied", "staging", "true", "/var/log/theia/theia-safe.jsonl", false},
		{"production-denied", "production", "true", "/var/log/theia/theia-safe.jsonl", false},
		{"remote-path-denied", "local", "true", "/tmp/logs/arbitrary.jsonl", false},
		{"relative-path-denied", "local", "true", "theia-safe.jsonl", false},
		{"disabled-path-denied", "local", "false", "/var/log/theia/theia-safe.jsonl", false},
		{"invalid-boolean", "local", "TRUE", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cleanConfigEnv(t)
			setSplitDatabaseEnv(t)
			t.Setenv("APP_ENV", tc.env)
			t.Setenv("THEIA_SAFE_LOG_EXPORT_ENABLED", tc.flag)
			t.Setenv("THEIA_SAFE_LOG_EXPORT_PATH", tc.path)
			_, err := Load()
			if tc.allowed && err != nil {
				t.Fatal(err)
			}
			if !tc.allowed && (err == nil || !strings.Contains(err.Error(), "THEIA_SAFE_LOG")) {
				t.Fatalf("expected safe-log validation error, got %v", err)
			}
		})
	}
}
