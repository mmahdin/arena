package config

import (
	"log/slog"
	"net/url"
	"strings"
	"testing"
)

func cleanConfigEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"APP_ENV", "HTTP_ADDR", "DATABASE_URL", "DATABASE_HOST", "DATABASE_PORT",
		"DATABASE_USERNAME", "DATABASE_PASSWORD", "DATABASE_NAME", "DATABASE_ENCRYPT",
		"ALLOW_DEV_LOGIN", "DEV_SESSION_SECRET", "FRONTEND_ORIGIN", "APP_ALLOWED_ORIGIN",
		"COOKIE_SECURE", "BUSINESS_TIMEZONE", "LOG_LEVEL", "LLM_PROVIDER",
		"MAX_CONTEXT_MESSAGES", "MAX_CONTEXT_CHARS", "LLM_TIMEOUT_SECONDS", "MODEL_MAX_OUTPUT_TOKENS",
	} {
		t.Setenv(key, "")
	}
}

func setSplitDatabaseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_HOST", "db.example.invalid")
	t.Setenv("DATABASE_PORT", "1433")
	t.Setenv("DATABASE_USERNAME", "app@tenant")
	t.Setenv("DATABASE_PASSWORD", "synthetic:p@ss/?#&+% word")
	t.Setenv("DATABASE_NAME", "Synthetic Stage & Tests")
	t.Setenv("DATABASE_ENCRYPT", "disable")
}

func TestLoadStagingConfigMap(t *testing.T) {
	cleanConfigEnv(t)
	setSplitDatabaseEnv(t)
	t.Setenv("APP_ENV", "staging")
	t.Setenv("HTTP_ADDR", ":8080")
	t.Setenv("FRONTEND_ORIGIN", "https://frontend.example.invalid")
	t.Setenv("APP_ALLOWED_ORIGIN", "https://old.example.invalid")
	t.Setenv("BUSINESS_TIMEZONE", "Asia/Tehran")
	t.Setenv("LOG_LEVEL", "warn")
	t.Setenv("ALLOW_DEV_LOGIN", "true")
	t.Setenv("DEV_SESSION_SECRET", strings.Repeat("s", 64))
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Environment != "staging" || cfg.Address != ":8080" || !cfg.AllowDevLogin {
		t.Fatal("staging application settings were not applied")
	}
	if cfg.AllowedOrigin != "https://frontend.example.invalid" || !cfg.CookieSecure {
		t.Fatal("frontend origin must override the legacy value and default to secure cookies on HTTPS")
	}
	if cfg.BusinessTimezone.String() != "Asia/Tehran" || cfg.LogLevel != slog.LevelWarn {
		t.Fatal("business timezone or log level was not loaded")
	}
}

func TestDevLoginEnvironmentGuard(t *testing.T) {
	for _, environment := range []string{"local", "stage", "staging", "staging-v2", "production", "prod", "unknown"} {
		t.Run(environment, func(t *testing.T) {
			cleanConfigEnv(t)
			setSplitDatabaseEnv(t)
			t.Setenv("APP_ENV", environment)
			t.Setenv("ALLOW_DEV_LOGIN", "true")
			t.Setenv("DEV_SESSION_SECRET", strings.Repeat("s", 64))
			_, err := Load()
			allowed := environment == "local" || environment == "stage" || environment == "staging" || environment == "staging-v2"
			if allowed && err != nil {
				t.Fatal(err)
			}
			if !allowed && (err == nil || !strings.Contains(err.Error(), "ALLOW_DEV_LOGIN")) {
				t.Fatal("dev login must be rejected outside local and staging")
			}
		})
	}
}

func TestDevLoginIsExplicitAndRequiresSecret(t *testing.T) {
	cleanConfigEnv(t)
	setSplitDatabaseEnv(t)
	t.Setenv("APP_ENV", "staging")
	cfg, err := Load()
	if err != nil || cfg.AllowDevLogin {
		t.Fatal("staging must not automatically enable tester login")
	}
	t.Setenv("ALLOW_DEV_LOGIN", "true")
	t.Setenv("DEV_SESSION_SECRET", "too-short")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "DEV_SESSION_SECRET") {
		t.Fatal("staging tester login must require a signing secret")
	}
}

func TestLocalConfigCompatibility(t *testing.T) {
	cleanConfigEnv(t)
	raw := "sqlserver://sa:synthetic@127.0.0.1:14333?database=wellbeing&encrypt=disable"
	t.Setenv("DATABASE_URL", raw)
	t.Setenv("APP_ALLOWED_ORIGIN", "http://localhost:5173")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	dsn, _ := url.Parse(cfg.DatabaseURL)
	if cfg.Environment != "local" || dsn.Hostname() != "127.0.0.1" || dsn.Query().Get("database") != "wellbeing" {
		t.Fatal("local configuration must retain its independent database")
	}
	if cfg.AllowedOrigin != "http://localhost:5173" || cfg.CookieSecure {
		t.Fatal("legacy local origin and HTTP cookie defaults changed")
	}
	t.Setenv("FRONTEND_ORIGIN", "https://frontend.example.invalid")
	t.Setenv("COOKIE_SECURE", "false")
	cfg, err = Load()
	if err != nil || cfg.CookieSecure {
		t.Fatal("explicit COOKIE_SECURE override must be respected")
	}
}

func TestInvalidRuntimeConfigIsSanitized(t *testing.T) {
	for _, tc := range []struct{ key, value string }{
		{"BUSINESS_TIMEZONE", "not/a/timezone"},
		{"LOG_LEVEL", "not-a-level"},
	} {
		t.Run(tc.key, func(t *testing.T) {
			cleanConfigEnv(t)
			setSplitDatabaseEnv(t)
			t.Setenv(tc.key, tc.value)
			if _, err := Load(); err == nil || !strings.Contains(err.Error(), tc.key) || strings.Contains(err.Error(), tc.value) {
				t.Fatal("invalid runtime setting must fail without exposing its value")
			}
		})
	}
}
