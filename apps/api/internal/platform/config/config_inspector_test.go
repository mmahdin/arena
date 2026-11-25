package config

import (
	"testing"
)

func TestLocalInspectorConfigurationGuard(t *testing.T) {
	setAIEnv(t)
	for _, tc := range []struct {
		env, address string
		allowed      bool
	}{{"local", "127.0.0.1:8080", true}, {"local", "[::1]:8080", true}, {"local", "0.0.0.0:8080", false}, {"local", ":8080", false}, {"staging", "127.0.0.1:8080", false}, {"production", "127.0.0.1:8080", false}, {"unknown", "127.0.0.1:8080", false}} {
		t.Run(tc.env+tc.address, func(t *testing.T) {
			t.Setenv("APP_ENV", tc.env)
			t.Setenv("HTTP_ADDR", tc.address)
			t.Setenv("AI_LOCAL_INSPECTOR", "true")
			cfg, err := Load()
			if (err == nil) != tc.allowed {
				t.Fatal(cfg.LocalInspector, err)
			}
		})
	}
}
