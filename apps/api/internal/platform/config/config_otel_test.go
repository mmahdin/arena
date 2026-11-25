package config

import (
	"strings"
	"testing"
)

func setupLocalCaptureEnv(t *testing.T) {
	t.Helper()
	cleanConfigEnv(t)
	setSplitDatabaseEnv(t)
	t.Setenv("APP_ENV", "local")
	t.Setenv("HTTP_ADDR", "127.0.0.1:8080")
	t.Setenv("THEIA_OTEL_ENABLED", "true")
	t.Setenv("THEIA_OTEL_CAPTURE_CONTENT", "true")
	t.Setenv("THEIA_OTEL_CONTENT_API_KEY", strings.Repeat("a", 48))
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "http://127.0.0.1:6006/v1/traces")
}

func TestMetadataTracingConfigFailClosed(t *testing.T) {
	cleanConfigEnv(t)
	setSplitDatabaseEnv(t)
	t.Setenv("APP_ENV", "local")
	t.Setenv("THEIA_OTEL_ENABLED", "true")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "http://127.0.0.1:6006/v1/traces")
	cfg, err := Load()
	if err != nil || !cfg.OTelOptions.Enabled || cfg.OTelOptions.CaptureContent {
		t.Fatalf("metadata tracing invalid: %+v, %v", cfg.OTelOptions, err)
	}
	t.Setenv("THEIA_OTEL_CAPTURE_CONTENT", "true")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "ingestion key") {
		t.Fatalf("content export without receiver key accepted: %v", err)
	}
	t.Setenv("THEIA_OTEL_CAPTURE_CONTENT", "false")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "http://invalid.example/v1/traces")
	if _, err := Load(); err == nil {
		t.Fatal("nonloopback plaintext telemetry accepted")
	}
	t.Setenv("THEIA_OTEL_ENABLED", "false")
	if _, err := Load(); err != nil {
		t.Fatalf("disabled tracer must ignore exporter URL: %v", err)
	}
}

func TestFullContentEnabledOnLocalBFFWithoutDevSessionOrInspector(t *testing.T) {
	setupLocalCaptureEnv(t)
	t.Setenv("AI_LOCAL_INSPECTOR", "false")
	t.Setenv("ALLOW_DEV_LOGIN", "false")
	cfg, err := Load()
	if err != nil || !cfg.OTelOptions.CaptureContent {
		t.Fatalf("opt-in local content capture failed: %v", err)
	}
	t.Setenv("ALLOW_DEV_LOGIN", "true")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "retired") {
		t.Fatalf("Dev Session bypass restored by capture: %v", err)
	}
}

func TestContentCaptureRejectsPublicOrSharedRuntime(t *testing.T) {
	setupLocalCaptureEnv(t)
	for _, address := range []string{":8080", "0.0.0.0:8080", "192.0.2.10:8080"} {
		t.Run(address, func(t *testing.T) {
			t.Setenv("HTTP_ADDR", address)
			if _, err := Load(); err == nil || !strings.Contains(err.Error(), "loopback") {
				t.Fatalf("public listener accepted: %v", err)
			}
		})
	}
	t.Setenv("HTTP_ADDR", "127.0.0.1:8080")
	for _, env := range []string{"stage", "production"} {
		t.Run(env, func(t *testing.T) {
			t.Setenv("APP_ENV", env)
			if _, err := Load(); err == nil {
				t.Fatal("shared capture environment accepted")
			}
		})
	}
}

func TestOTelBooleanEnvironmentValuesMustBeExplicit(t *testing.T) {
	setupLocalCaptureEnv(t)
	for _, key := range []string{"THEIA_OTEL_ENABLED", "THEIA_OTEL_CAPTURE_CONTENT"} {
		t.Run(key, func(t *testing.T) {
			t.Setenv(key, "yes")
			if _, err := Load(); err == nil {
				t.Fatal("invalid boolean accepted")
			}
		})
	}
}

func TestLocalAuthenticatedDualCollectorContentOptIn(t *testing.T) {
	setupLocalCaptureEnv(t)
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "http://127.0.0.1:4318/v1/traces")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("valid local dual collector rejected: %v", err)
	}
	if !cfg.OTelOptions.CaptureContent || cfg.OTelOptions.PrivateIngestKey == "" {
		t.Fatal("private capture unexpectedly disabled")
	}
	t.Setenv("THEIA_OTEL_CONTENT_API_KEY", "")
	if _, err := Load(); err == nil {
		t.Fatal("local dual collector without ingestion key accepted")
	}
}
