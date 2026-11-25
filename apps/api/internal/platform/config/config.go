package config

import (
	"errors"
	"log/slog"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/metrics"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/telemetry"
	_ "time/tzdata"
)

type Config struct {
	MetricsOptions    metrics.Options
	MetricsRuntime    *metrics.Runtime // initialized in cmd/api, nil when disabled
	OTelOptions       telemetry.Options
	OTelRuntime       *telemetry.Runtime // injected by cmd/api after validation; never from caller input
	Auth              AuthConfig
	LocalInspector    bool
	LocalInspectorDir string
	AICoreConfigFile  string
	// LocalKBDraft permits an unsigned, synthetic-only Knowledge Bundle on a loopback-only developer server.
	LocalKBDraft bool

	Environment          string
	Address              string
	DatabaseURL          string
	AllowedOrigin        string
	CookieSecure         bool
	BusinessTimezone     *time.Location
	LogLevel             slog.Level
	SafeLogEnabled       bool
	LLMProvider          string
	ModelAPIBaseURL      string
	ModelAPIKey          string
	ModelName            string
	LLMTimeout           time.Duration
	ModelMaxOutputTokens int
	MaxContextChars      int
	MaxContextMessages   int
}

func Load() (Config, error) {
	cfg := Config{
		SafeLogEnabled: os.Getenv("THEIA_SAFE_LOG_EXPORT_ENABLED") == "true",
		MetricsOptions: metrics.Options{Enabled: os.Getenv("THEIA_METRICS_ENABLED") == "true", Address: getenv("THEIA_METRICS_ADDR", "127.0.0.1:9100"), Environment: strings.ToLower(strings.TrimSpace(getenv("APP_ENV", "local")))},
		OTelOptions: telemetry.Options{
			Enabled:          os.Getenv("THEIA_OTEL_ENABLED") == "true",
			CaptureContent:   os.Getenv("THEIA_OTEL_CAPTURE_CONTENT") == "true",
			PrivateIngestKey: os.Getenv("THEIA_OTEL_CONTENT_API_KEY"),
			Endpoint:         strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")),
			ServiceName:      getenv("OTEL_SERVICE_NAME", "theia-api"),
			ServiceVersion:   getenv("OTEL_SERVICE_VERSION", "dev"),
			Environment:      strings.ToLower(strings.TrimSpace(getenv("APP_ENV", "local"))),
		},
		LocalInspector:       os.Getenv("AI_LOCAL_INSPECTOR") == "true",
		LocalInspectorDir:    getenv("AI_LOCAL_INSPECTOR_DIR", ".local/ai-inspector"),
		AICoreConfigFile:     os.Getenv("AI_CORE_CONFIG_FILE"),
		LocalKBDraft:         os.Getenv("AI_CORE_LOCAL_DRAFT") == "true",
		Environment:          strings.ToLower(strings.TrimSpace(getenv("APP_ENV", "local"))),
		Address:              getenv("HTTP_ADDR", "127.0.0.1:8080"),
		AllowedOrigin:        getenv("FRONTEND_ORIGIN", getenv("APP_ALLOWED_ORIGIN", "http://localhost:5173")),
		LLMProvider:          getenv("LLM_PROVIDER", "mock"),
		ModelAPIBaseURL:      getenv("MODEL_API_BASE_URL", "https://api.openai.com/v1"),
		ModelAPIKey:          os.Getenv("MODEL_API_KEY"),
		ModelName:            getenv("MODEL_NAME", "configure-me"),
		LLMTimeout:           35 * time.Second,
		ModelMaxOutputTokens: 700,
		MaxContextChars:      16000,
		MaxContextMessages:   20,
	}
	var err error
	if v := os.Getenv("THEIA_METRICS_ENABLED"); v != "" && v != "true" && v != "false" {
		return cfg, errors.New("THEIA_METRICS_ENABLED must be true or false")
	}
	if raw := os.Getenv("THEIA_SAFE_LOG_EXPORT_ENABLED"); raw != "" && raw != "true" && raw != "false" {
		return cfg, errors.New("THEIA_SAFE_LOG_EXPORT_ENABLED must be true or false")
	}
	if cfg.SafeLogEnabled && cfg.Environment != "local" {
		return cfg, errors.New("THEIA_SAFE_LOG_EXPORT_ENABLED requires APP_ENV=local")
	}
	if cfg.SafeLogEnabled {
		path := os.Getenv("THEIA_SAFE_LOG_EXPORT_PATH")
		if !filepath.IsAbs(path) || filepath.Base(path) != "theia-safe.jsonl" {
			return cfg, errors.New("THEIA_SAFE_LOG_EXPORT_PATH must be an absolute dedicated JSONL path")
		}
	}
	if !cfg.SafeLogEnabled && os.Getenv("THEIA_SAFE_LOG_EXPORT_PATH") != "" {
		return cfg, errors.New("THEIA_SAFE_LOG_EXPORT_PATH requires THEIA_SAFE_LOG_EXPORT_ENABLED=true")
	}
	if err = metrics.Validate(cfg.MetricsOptions); err != nil {
		return cfg, err
	}
	if cfg.Auth, err = readAuthConfig(cfg.Environment); err != nil {
		return cfg, err
	}
	if cfg.DatabaseURL, err = LoadDatabaseURL(); err != nil {
		return cfg, err
	}
	if cfg.BusinessTimezone, err = time.LoadLocation(getenv("BUSINESS_TIMEZONE", "Asia/Tehran")); err != nil {
		return cfg, errors.New("BUSINESS_TIMEZONE must be a valid IANA timezone")
	}
	if err = cfg.LogLevel.UnmarshalText([]byte(getenv("LOG_LEVEL", "info"))); err != nil {
		return cfg, errors.New("LOG_LEVEL must be debug, info, warn or error")
	}
	cfg.CookieSecure = getenv("COOKIE_SECURE", strconv.FormatBool(strings.HasPrefix(cfg.AllowedOrigin, "https://"))) == "true"
	if s := os.Getenv("MAX_CONTEXT_MESSAGES"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 || n > 40 {
			return cfg, errors.New("MAX_CONTEXT_MESSAGES must be in 1..40")
		}
		cfg.MaxContextMessages = n
	}
	for _, item := range []struct {
		key      string
		dest     *int
		min, max int
	}{
		{"LLM_TIMEOUT_SECONDS", nil, 5, 90},
		{"MODEL_MAX_OUTPUT_TOKENS", &cfg.ModelMaxOutputTokens, 128, 2048},
		{"MAX_CONTEXT_CHARS", &cfg.MaxContextChars, 2000, 30000},
	} {
		if raw := os.Getenv(item.key); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil || n < item.min || n > item.max {
				return cfg, errors.New(item.key + " is out of range")
			}
			if item.dest != nil {
				*item.dest = n
			} else {
				cfg.LLMTimeout = time.Duration(n) * time.Second
			}
		}
	}
	// No partial or mistyped opt-in values are silently treated as approval.
	for _, key := range []string{"THEIA_OTEL_ENABLED", "THEIA_OTEL_CAPTURE_CONTENT"} {
		if value := os.Getenv(key); value != "" && value != "true" && value != "false" {
			return cfg, errors.New(key + " must be true or false")
		}
	}
	// Content capture is a server-side opt-in. It does not restore Dev Session
	// or require enabling the separate offline Inspector. Full capture remains
	// local-loopback only; telemetry.Validate enforces the private receiver/key.
	cfg.OTelOptions.LoopbackListener = LocalDraftListenerAllowed(cfg.Address)
	if err := telemetry.Validate(cfg.OTelOptions); err != nil {
		return cfg, err
	}
	if cfg.LocalInspector && (cfg.Environment != "local" || !LocalDraftListenerAllowed(cfg.Address)) {
		return cfg, errors.New("AI_LOCAL_INSPECTOR requires APP_ENV=local and a loopback HTTP_ADDR")
	}
	if cfg.LocalKBDraft && (cfg.Environment != "local" || !LocalDraftListenerAllowed(cfg.Address) || cfg.AICoreConfigFile == "") {
		return cfg, errors.New("AI_CORE_LOCAL_DRAFT requires APP_ENV=local, a loopback HTTP_ADDR and AI_CORE_CONFIG_FILE")
	}
	// AUTH-09 rejects stale opt-in switches instead of silently ignoring a
	// previously configured synthetic identity bypass.
	if strings.EqualFold(strings.TrimSpace(os.Getenv("ALLOW_DEV_LOGIN")), "true") {
		return cfg, errors.New("ALLOW_DEV_LOGIN was retired in AUTH-09; use Keycloak User BFF")
	}
	if cfg.LLMProvider != "mock" && cfg.LLMProvider != "openai-compatible" {
		return cfg, errors.New("unsupported LLM_PROVIDER")
	}
	if cfg.LLMProvider == "openai-compatible" {
		if cfg.ModelAPIKey == "" {
			return cfg, errors.New("MODEL_API_KEY required")
		}
		if cfg.ModelName == "configure-me" {
			return cfg, errors.New("MODEL_NAME must be set for external provider")
		}
		endpoint, err := url.Parse(cfg.ModelAPIBaseURL)
		if err != nil || endpoint.Scheme != "https" || endpoint.Hostname() == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
			return cfg, errors.New("MODEL_API_BASE_URL must be an https origin/path without credentials, query, or fragment")
		}
	}
	return cfg, nil
}

func getenv(key, fallback string) string {
	if s := os.Getenv(key); s != "" {
		return s
	}
	return fallback
}

// loopbackListener prevents unreviewed knowledge from being served on shared networks.
func LocalDraftListenerAllowed(address string) bool {
	host, _, err := net.SplitHostPort(address)
	if err != nil || host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
