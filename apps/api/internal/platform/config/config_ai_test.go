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

func TestEmbeddingConfigurationDefaultsAreOffline(t *testing.T) {
	setAIEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.EmbeddingProvider != "deterministic" || cfg.EmbeddingDimensions != 128 {
		t.Fatalf("embedding tier must default to the offline deterministic model, got %q/%d", cfg.EmbeddingProvider, cfg.EmbeddingDimensions)
	}
}

func TestEmbeddingConfigurationValidation(t *testing.T) {
	setAIEnv(t)
	t.Setenv("EMBEDDING_PROVIDER", "pgvector")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "EMBEDDING_PROVIDER") {
		t.Fatalf("unknown embedding provider not rejected: %v", err)
	}
	t.Setenv("EMBEDDING_PROVIDER", "openai-compatible")
	t.Setenv("EMBEDDING_MODEL", "text-embedding-synthetic")
	t.Setenv("EMBEDDING_DIMENSIONS", "4")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "EMBEDDING_DIMENSIONS") {
		t.Fatalf("out-of-range dimensions not rejected: %v", err)
	}
	t.Setenv("EMBEDDING_DIMENSIONS", "1024")
	cfg, err := Load()
	if err != nil || cfg.EmbeddingDimensions != 1024 {
		t.Fatalf("valid embedding config rejected: %v %+v", err, cfg)
	}
	// The network embedding provider needs the same key as the LLM client.
	t.Setenv("MODEL_API_KEY", "")
	if _, err := Load(); err == nil {
		t.Fatal("network embeddings without a key must be rejected")
	}
}
