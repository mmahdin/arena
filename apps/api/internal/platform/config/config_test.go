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
		"AI_CORE_LOCAL_DRAFT", "AI_CORE_CONFIG_FILE", "AI_LOCAL_INSPECTOR", "APP_ENV", "HTTP_ADDR", "DATABASE_URL", "DATABASE_HOST", "DATABASE_PORT",
		"DATABASE_USERNAME", "DATABASE_PASSWORD", "DATABASE_NAME", "DATABASE_ENCRYPT",
		"ALLOW_DEV_LOGIN", "DEV_SESSION_SECRET", "FRONTEND_ORIGIN", "APP_ALLOWED_ORIGIN",
		"AUTH_BFF_ENABLED", "AUTH_USER_LOGIN_ENABLED", "AUTH_USER_SMS_FLOW_VERIFIED", "AUTH_STAFF_LOGIN_ENABLED", "AUTH_BFF_KEY_BASE64",
		"AUTH_KEYCLOAK_BASE_URL", "AUTH_USER_ORIGIN", "AUTH_STAFF_ORIGIN", "AUTH_TRUSTED_PROXY_CIDRS", "AUTH_BROWSER_SMOKE_ENABLED",
		"THEIA_USERS_BFF_CLIENT_SECRET", "THEIA_STAFF_BFF_CLIENT_SECRET",
		"COOKIE_SECURE", "BUSINESS_TIMEZONE", "LOG_LEVEL", "LLM_PROVIDER",
		"THEIA_SAFE_LOG_EXPORT_ENABLED", "THEIA_SAFE_LOG_EXPORT_PATH",
		"MAX_CONTEXT_MESSAGES", "MAX_CONTEXT_CHARS", "LLM_TIMEOUT_SECONDS", "MODEL_MAX_OUTPUT_TOKENS",
		"THEIA_OTEL_ENABLED", "THEIA_OTEL_CAPTURE_CONTENT", "THEIA_OTEL_SYNTHETIC_DATA_CONFIRMED", "THEIA_OTEL_CONTENT_API_KEY", "OTEL_SERVICE_NAME", "OTEL_SERVICE_VERSION", "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
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
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Environment != "staging" || cfg.Address != ":8080" {
		t.Fatal("staging application settings were not applied")
	}
	if cfg.AllowedOrigin != "https://frontend.example.invalid" || !cfg.CookieSecure {
		t.Fatal("frontend origin must override the legacy value and default to secure cookies on HTTPS")
	}
	if cfg.BusinessTimezone.String() != "Asia/Tehran" || cfg.LogLevel != slog.LevelWarn {
		t.Fatal("business timezone or log level was not loaded")
	}
}

func TestAUTH09OldDevLoginFlagCannotReactivateTestIdentity(t *testing.T) {
	for _, env := range []string{"local", "staging", "production"} {
		t.Run(env, func(t *testing.T) {
			cleanConfigEnv(t)
			setSplitDatabaseEnv(t)
			t.Setenv("APP_ENV", env)
			t.Setenv("ALLOW_DEV_LOGIN", "true")
			if _, err := Load(); err == nil || !strings.Contains(err.Error(), "retired") {
				t.Fatalf("old Dev Login flag accepted: %v", err)
			}
		})
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

func TestLocalKnowledgeDraftMustBeExplicitAndLoopbackOnly(t *testing.T) {
	cleanConfigEnv(t)
	setSplitDatabaseEnv(t)
	t.Setenv("AI_CORE_LOCAL_DRAFT", "true")
	t.Setenv("AI_CORE_CONFIG_FILE", "/tmp/dev-runtime.json")
	t.Setenv("APP_ENV", "local")
	for _, address := range []string{"127.0.0.1:8080", "localhost:8080", "[::1]:8080"} {
		t.Setenv("HTTP_ADDR", address)
		cfg, err := Load()
		if err != nil || !cfg.LocalKBDraft {
			t.Fatalf("local draft rejected for %s: %v", address, err)
		}
	}
	for _, address := range []string{":8080", "0.0.0.0:8080", "[::]:8080", "192.0.2.10:8080"} {
		t.Setenv("HTTP_ADDR", address)
		if _, err := Load(); err == nil || !strings.Contains(err.Error(), "AI_CORE_LOCAL_DRAFT") {
			t.Fatalf("draft must reject public listener %q", address)
		}
	}
	t.Setenv("HTTP_ADDR", "127.0.0.1:8080")
	for _, environment := range []string{"staging", "stage", "production", "prod"} {
		t.Setenv("APP_ENV", environment)
		if _, err := Load(); err == nil || !strings.Contains(err.Error(), "AI_CORE_LOCAL_DRAFT") {
			t.Fatalf("draft must reject %s", environment)
		}
	}
	t.Setenv("APP_ENV", "local")
	if cfg, err := Load(); err != nil || !cfg.LocalKBDraft {
		t.Fatalf("safe local draft must not require developer login: %+v %v", cfg, err)
	}
	t.Setenv("AI_CORE_CONFIG_FILE", "")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "AI_CORE_LOCAL_DRAFT") {
		t.Fatal("draft must require runtime configuration")
	}
}
