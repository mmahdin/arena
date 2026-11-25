package config

import (
	"encoding/base64"
	"errors"
	"net"
	"net/netip"
	"net/url"
	"os"
	"strings"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth/oidc"
)

// AuthConfig configures only the real, issuer-bound User and Staff BFF sessions.
type AuthConfig struct {
	Enabled             bool
	UserLoginEnabled    bool
	UserSMSFlowVerified bool // operator attestation; ID-token SMS proof remains mandatory
	StaffLoginEnabled   bool
	Key                 []byte
	UserIssuer          oidc.Settings
	StaffIssuer         oidc.Settings
	UserOrigin          string
	StaffOrigin         string
	TrustedProxyCIDRs   []netip.Prefix
	BrowserSmokeEnabled bool // local-only, opt-in same-origin smoke harness
}

func readAuthConfig(environment string) (AuthConfig, error) {
	var a AuthConfig
	a.Enabled = os.Getenv("AUTH_BFF_ENABLED") == "true"
	a.BrowserSmokeEnabled = os.Getenv("AUTH_BROWSER_SMOKE_ENABLED") == "true"
	if a.BrowserSmokeEnabled && (!a.Enabled || environment != "local") {
		return a, errors.New("AUTH_BROWSER_SMOKE_ENABLED requires local BFF")
	}
	if !a.Enabled {
		return a, nil
	}
	raw := os.Getenv("AUTH_BFF_KEY_BASE64")
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil || len(key) != 32 {
		return a, errors.New("AUTH_BFF_KEY_BASE64 must encode 32 bytes")
	}
	a.Key = key
	a.UserLoginEnabled = os.Getenv("AUTH_USER_LOGIN_ENABLED") == "true"
	a.UserSMSFlowVerified = os.Getenv("AUTH_USER_SMS_FLOW_VERIFIED") == "true"
	if a.UserLoginEnabled && !a.UserSMSFlowVerified {
		return a, errors.New("AUTH_USER_SMS_FLOW_VERIFIED required before User login")
	}
	a.StaffLoginEnabled = os.Getenv("AUTH_STAFF_LOGIN_ENABLED") == "true"
	base := os.Getenv("AUTH_KEYCLOAK_BASE_URL")
	a.UserOrigin = os.Getenv("AUTH_USER_ORIGIN")
	a.StaffOrigin = os.Getenv("AUTH_STAFF_ORIGIN")
	if environment == "local" {
		if base == "" {
			base = "http://auth.localhost:8180"
		}
		if a.UserOrigin == "" {
			a.UserOrigin = "http://app.localhost:8080"
		}
		if a.StaffOrigin == "" {
			a.StaffOrigin = "http://admin.localhost:8080"
		}
	}
	if err = checkAuthURL(base, environment, true); err != nil {
		return a, errors.New("AUTH_KEYCLOAK_BASE_URL invalid")
	}
	if err = checkAuthURL(a.UserOrigin, environment, false); err != nil {
		return a, errors.New("AUTH_USER_ORIGIN invalid")
	}
	if err = checkAuthURL(a.StaffOrigin, environment, false); err != nil {
		return a, errors.New("AUTH_STAFF_ORIGIN invalid")
	}
	if strings.EqualFold(a.UserOrigin, a.StaffOrigin) {
		return a, errors.New("Auth user and staff origins must differ")
	}
	if a.BrowserSmokeEnabled && (!browserSmokeOrigin(a.UserOrigin) || !browserSmokeOrigin(a.StaffOrigin)) {
		return a, errors.New("AUTH_BROWSER_SMOKE_ENABLED requires loopback User and Staff origins")
	}
	a.UserIssuer = oidc.Settings{Issuer: base + "/realms/theia-users", ClientID: "theia-users-bff", ClientSecret: os.Getenv("THEIA_USERS_BFF_CLIENT_SECRET"), RedirectURI: a.UserOrigin + "/auth/user/callback", RequireSMSProof: true, RequireSID: true}
	a.StaffIssuer = oidc.Settings{Issuer: base + "/realms/theia-staff", ClientID: "theia-staff-bff", ClientSecret: os.Getenv("THEIA_STAFF_BFF_CLIENT_SECRET"), RedirectURI: a.StaffOrigin + "/auth/staff/callback", RequireSID: true}
	if a.UserIssuer.ClientSecret == "" || a.StaffIssuer.ClientSecret == "" {
		return a, errors.New("BFF client secrets are required")
	}
	if proxyList := strings.TrimSpace(os.Getenv("AUTH_TRUSTED_PROXY_CIDRS")); proxyList != "" {
		for _, value := range strings.Split(proxyList, ",") {
			prefix, e := netip.ParsePrefix(strings.TrimSpace(value))
			if e != nil {
				return a, errors.New("AUTH_TRUSTED_PROXY_CIDRS invalid")
			}
			if prefix.Bits() == 0 {
				return a, errors.New("AUTH_TRUSTED_PROXY_CIDRS must not trust all peers")
			}
			a.TrustedProxyCIDRs = append(a.TrustedProxyCIDRs, prefix.Masked())
		}
	}
	return a, nil
}

func checkAuthURL(value, environment string, allowPath bool) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Opaque != "" || parsed.Path != "" && (!allowPath || strings.HasSuffix(parsed.Path, "/")) {
		return errors.New("invalid URL")
	}
	if parsed.Scheme == "https" {
		return nil
	}
	if environment != "local" || parsed.Scheme != "http" || parsed.Path != "" {
		return errors.New("HTTPS required")
	}
	host := parsed.Hostname()
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return nil
	}
	ip := net.ParseIP(host)
	if ip != nil && ip.IsLoopback() {
		return nil
	}
	return errors.New("loopback origin required")
}

// browserSmokeOrigin prevents the opt-in smoke page from being exposed on a
// public host even if APP_ENV is incorrectly set to local.
func browserSmokeOrigin(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Hostname() == "" {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// BrowserSmokeAllowed is also used when tests or embedding callers construct
// Config directly, bypassing the environment parser.
func BrowserSmokeAllowed(cfg AuthConfig, environment string) bool {
	return environment == "local" && cfg.Enabled && browserSmokeOrigin(cfg.UserOrigin) && browserSmokeOrigin(cfg.StaffOrigin)
}
