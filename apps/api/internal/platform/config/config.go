package config

import (
	"errors"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"
)

type Config struct {
	Environment          string
	Address              string
	DatabaseURL          string
	AllowDevLogin        bool
	DevSessionSecret     string
	AllowedOrigin        string
	CookieSecure         bool
	BusinessTimezone     *time.Location
	LogLevel             slog.Level
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
		Environment:          strings.ToLower(strings.TrimSpace(getenv("APP_ENV", "local"))),
		Address:              getenv("HTTP_ADDR", "127.0.0.1:8080"),
		AllowDevLogin:        os.Getenv("ALLOW_DEV_LOGIN") == "true",
		DevSessionSecret:     os.Getenv("DEV_SESSION_SECRET"),
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
	if cfg.AllowDevLogin && !SupportsDevLogin(cfg.Environment) {
		return cfg, errors.New("ALLOW_DEV_LOGIN is allowed only in local or staging environments")
	}
	if cfg.AllowDevLogin && (len(cfg.DevSessionSecret) < 32 || cfg.DevSessionSecret == "replace_with_random_64_hex_characters_before_use") {
		return cfg, errors.New("DEV_SESSION_SECRET must be at least 32 bytes when development login is on")
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

// SupportsDevLogin permits explicit synthetic tester sessions in development
// and staging; production still requires a separate authentication mechanism.
func SupportsDevLogin(environment string) bool {
	switch environment {
	case "local", "stage", "staging", "staging-v2":
		return true
	default:
		return false
	}
}

func getenv(key, fallback string) string {
	if s := os.Getenv(key); s != "" {
		return s
	}
	return fallback
}
