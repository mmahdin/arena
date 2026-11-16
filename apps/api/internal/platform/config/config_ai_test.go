package config

import (
	"strings"
	"testing"
	"time"
)

func setAIEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "local")
	t.Setenv("ALLOW_DEV_LOGIN", "false")
	t.Setenv("DATABASE_URL", "sqlserver://test:password@example.invalid:1433?database=wellbeing")
	t.Setenv("LLM_PROVIDER", "openai-compatible")
	t.Setenv("MODEL_API_KEY", "synthetic-test-key")
	t.Setenv("MODEL_NAME", "synthetic-model")
	t.Setenv("MODEL_API_BASE_URL", "https://example.invalid/v1")
}
func TestLLMConfigurationBounds(t *testing.T) {
	setAIEnv(t)
	cfg, err := Load()
	if err != nil || cfg.LLMTimeout != 35*time.Second || cfg.MaxContextChars != 16000 || cfg.ModelMaxOutputTokens != 700 {
		t.Fatalf("default config=%+v err=%v", cfg, err)
	}
	for _, tc := range []struct{ key, value string }{
		{"LLM_TIMEOUT_SECONDS", "121"}, {"LLM_TIMEOUT_SECONDS", "0"},
		{"MAX_CONTEXT_CHARS", "50000"}, {"MAX_CONTEXT_CHARS", "abc"},
		{"MODEL_MAX_OUTPUT_TOKENS", "99999"}, {"MODEL_MAX_OUTPUT_TOKENS", "0"},
	} {
		t.Run(tc.key+tc.value, func(t *testing.T) {
			t.Setenv(tc.key, tc.value)
			if _, err := Load(); err == nil || !strings.Contains(err.Error(), tc.key) {
				t.Fatalf("invalid config not rejected: %v", err)
			}
		})
	}
}
func TestProviderURLRejectsCredentialOrQuery(t *testing.T) {
	for _, raw := range []string{"http://example.invalid/v1", "https://me:pass@example.invalid/v1", "https://example.invalid/v1?token=secret", "https://example.invalid/v1#fragment", "https:///v1"} {
		t.Run(raw, func(t *testing.T) {
			setAIEnv(t)
			t.Setenv("MODEL_API_BASE_URL", raw)
			if _, err := Load(); err == nil {
				t.Fatalf("accepted forbidden base URL")
			}
		})
	}
}
