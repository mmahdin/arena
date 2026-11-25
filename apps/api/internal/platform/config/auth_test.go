package config

import (
	"encoding/base64"
	"strings"
	"testing"
)

func setAuthTestEnv(t *testing.T) {
	t.Helper()
	cleanConfigEnv(t)
	setSplitDatabaseEnv(t)
	t.Setenv("APP_ENV", "local")
	t.Setenv("AUTH_BFF_ENABLED", "true")
	t.Setenv("AUTH_BFF_KEY_BASE64", base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 32))))
	t.Setenv("THEIA_USERS_BFF_CLIENT_SECRET", "synthetic-user-secret")
	t.Setenv("THEIA_STAFF_BFF_CLIENT_SECRET", "synthetic-staff-secret")
	t.Setenv("AUTH_USER_ORIGIN", "")
	t.Setenv("AUTH_STAFF_ORIGIN", "")
	t.Setenv("AUTH_KEYCLOAK_BASE_URL", "")
	t.Setenv("AUTH_USER_LOGIN_ENABLED", "false")
	t.Setenv("AUTH_USER_SMS_FLOW_VERIFIED", "false")
	t.Setenv("AUTH_STAFF_LOGIN_ENABLED", "false")
	t.Setenv("AUTH_TRUSTED_PROXY_CIDRS", "")
	t.Setenv("AUTH_BROWSER_SMOKE_ENABLED", "")
}
func TestBFFLocalConfigDefaultsAndExplicitFlags(t *testing.T) {
	setAuthTestEnv(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Auth.Enabled || cfg.Auth.UserLoginEnabled || cfg.Auth.StaffLoginEnabled {
		t.Fatal("login enabled without per-realm flag")
	}
	if cfg.Auth.UserOrigin != "http://app.localhost:8080" || cfg.Auth.StaffOrigin != "http://admin.localhost:8080" {
		t.Fatal("local host binding changed")
	}
	if cfg.Auth.UserIssuer.ClientID != "theia-users-bff" || cfg.Auth.StaffIssuer.ClientID != "theia-staff-bff" {
		t.Fatal("realm client IDs changed")
	}
}
func TestBFFRejectsWeakKeySharedHostAndInsecureNonlocalOrigins(t *testing.T) {
	for _, tc := range []struct {
		name       string
		key, value string
	}{
		{"weak key", "AUTH_BFF_KEY_BASE64", "too-short"},
		{"same host", "AUTH_STAFF_ORIGIN", "http://app.localhost:8080"},
		{"public HTTP origin", "AUTH_USER_ORIGIN", "http://example.com:8080"},
		{"public HTTP issuer", "AUTH_KEYCLOAK_BASE_URL", "http://example.com:8180"},
		{"malformed CIDR", "AUTH_TRUSTED_PROXY_CIDRS", "not-a-cidr"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setAuthTestEnv(t)
			t.Setenv(tc.key, tc.value)
			if _, err := Load(); err == nil {
				t.Fatal("unsafe Auth configuration accepted")
			}
		})
	}
	setAuthTestEnv(t)
	t.Setenv("APP_ENV", "staging")
	if _, err := Load(); err == nil {
		t.Fatal("HTTP local defaults accepted outside local environment")
	}
}

func TestAUTH09RejectsRetiredDevLoginWhileStaffConfigRemainsIndependent(t *testing.T) {
	setAuthTestEnv(t)
	t.Setenv("ALLOW_DEV_LOGIN", "true")
	if _, err := Load(); err == nil {
		t.Fatal("deprecated Dev Login accepted")
	}
	t.Setenv("ALLOW_DEV_LOGIN", "false")
	t.Setenv("AUTH_STAFF_LOGIN_ENABLED", "true")
	if _, err := Load(); err != nil {
		t.Fatal("Staff BFF disabled by legacy false value:", err)
	}
}

func TestAUTH06UserBFFNeedsExplicitSMSAttestation(t *testing.T) {
	setAuthTestEnv(t)
	t.Setenv("AUTH_USER_LOGIN_ENABLED", "true")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "AUTH_USER_SMS_FLOW_VERIFIED") {
		t.Fatal("User BFF enabled without explicit verified-SMS attestation")
	}
	t.Setenv("AUTH_USER_SMS_FLOW_VERIFIED", "true")
	cfg, err := Load()
	if err != nil || !cfg.Auth.UserLoginEnabled || !cfg.Auth.UserSMSFlowVerified || !cfg.Auth.UserIssuer.RequireSMSProof || cfg.Auth.StaffIssuer.RequireSMSProof {
		t.Fatalf("User SMS proof must be mandatory and Staff login independent: %v", err)
	}
	t.Setenv("ALLOW_DEV_LOGIN", "true")
	if _, err := Load(); err == nil {
		t.Fatal("real User login permitted alongside synthetic Dev Login")
	}
}

func TestAUTH07LocalBrowserSmokeConfigGuard(t *testing.T) {
	setAuthTestEnv(t)
	t.Setenv("AUTH_BROWSER_SMOKE_ENABLED", "true")
	cfg, err := Load()
	if err != nil || !cfg.Auth.BrowserSmokeEnabled || !BrowserSmokeAllowed(cfg.Auth, cfg.Environment) {
		t.Fatalf("explicit local smoke should be permitted: %v", err)
	}
	for _, tc := range []struct{ name, key, value string }{
		{"public user host", "AUTH_USER_ORIGIN", "https://app.example.invalid"},
		{"public staff host", "AUTH_STAFF_ORIGIN", "https://admin.example.invalid"},
		{"staging", "APP_ENV", "staging"},
		{"BFF disabled", "AUTH_BFF_ENABLED", "false"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setAuthTestEnv(t)
			t.Setenv("AUTH_BROWSER_SMOKE_ENABLED", "true")
			t.Setenv(tc.key, tc.value)
			if _, err := Load(); err == nil {
				t.Fatal("unsafe browser smoke config accepted")
			}
		})
	}
}

func TestAUTH07TrustedProxyCannotBeOpenInternet(t *testing.T) {
	for _, prefix := range []string{"0.0.0.0/0", "::/0"} {
		setAuthTestEnv(t)
		t.Setenv("AUTH_TRUSTED_PROXY_CIDRS", prefix)
		if _, err := Load(); err == nil {
			t.Fatalf("trusted proxy %s must not include the entire internet", prefix)
		}
	}
	setAuthTestEnv(t)
	t.Setenv("AUTH_TRUSTED_PROXY_CIDRS", "172.18.0.0/16,127.0.0.1/32")
	if cfg, err := Load(); err != nil || len(cfg.Auth.TrustedProxyCIDRs) != 2 {
		t.Fatalf("explicit local proxies rejected: %v", err)
	}
}
