package config

import "testing"

func TestMetricsConfigurationLocalAndOptInOnly(t *testing.T) {
	cleanConfigEnv(t)
	setSplitDatabaseEnv(t)
	t.Setenv("APP_ENV", "local")
	t.Setenv("THEIA_METRICS_ENABLED", "true")
	t.Setenv("THEIA_METRICS_ADDR", "127.0.0.1:9100")
	cfg, err := Load()
	if err != nil || !cfg.MetricsOptions.Enabled {
		t.Fatalf("valid metrics config failed: %v", err)
	}
	for _, tc := range []struct{ key, value string }{{"THEIA_METRICS_ENABLED", "yes"}, {"APP_ENV", "prod"}, {"THEIA_METRICS_ADDR", "0.0.0.0:9100"}, {"THEIA_METRICS_ADDR", ":9100"}} {
		t.Run(tc.key+tc.value, func(t *testing.T) {
			t.Setenv(tc.key, tc.value)
			if _, err := Load(); err == nil {
				t.Fatal("unsafe metrics config accepted")
			}
		})
	}
}
