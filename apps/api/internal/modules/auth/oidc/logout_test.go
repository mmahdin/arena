package oidc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAUTH08LogoutTokenStrictVerification(t *testing.T) {
	key, jwk := rsaFixture(t)
	now := time.Now().UTC().Truncate(time.Second)
	const issuer = "https://auth.example.invalid/realms/theia-users"
	good := func() map[string]interface{} {
		return map[string]interface{}{
			"iss": issuer, "sub": "opaque-1", "sid": "sid-1", "aud": "theia-users-bff",
			"iat": now.Unix(), "exp": now.Add(3 * time.Minute).Unix(), "jti": "nonce-1",
			"events": map[string]interface{}{backchannelEventURI: map[string]interface{}{}},
		}
	}
	if e, err := VerifyLogoutToken(jwtFixture(t, key, good()), issuer, "theia-users-bff", []publicKey{jwk}, now); err != nil || e.SessionID != "sid-1" {
		t.Fatalf("valid logout rejected: %+v %v", e, err)
	}
	testCases := []struct {
		name   string
		change func(map[string]interface{})
	}{
		{"wrong issuer", func(m map[string]interface{}) { m["iss"] = "https://evil.invalid/realms/other" }},
		{"wrong client", func(m map[string]interface{}) { m["aud"] = "theia-staff-bff" }},
		{"multiple aud without azp", func(m map[string]interface{}) { m["aud"] = []string{"theia-users-bff", "evil"} }},
		{"no jti", func(m map[string]interface{}) { delete(m, "jti") }},
		{"expired", func(m map[string]interface{}) { m["exp"] = now.Add(-time.Second).Unix() }},
		{"future iat", func(m map[string]interface{}) { m["iat"] = now.Add(2 * time.Minute).Unix() }},
		{"old iat", func(m map[string]interface{}) { m["iat"] = now.Add(-6 * time.Minute).Unix() }},
		{"missing exp", func(m map[string]interface{}) { delete(m, "exp") }},
		{"missing subject and sid", func(m map[string]interface{}) { delete(m, "sub"); delete(m, "sid") }},
		{"nonce forbidden", func(m map[string]interface{}) { m["nonce"] = "login-nonce" }},
		{"null nonce forbidden", func(m map[string]interface{}) { m["nonce"] = nil }},
		{"wrong event", func(m map[string]interface{}) { m["events"] = map[string]interface{}{"event": "{}"} }},
		{"event null", func(m map[string]interface{}) { m["events"] = map[string]interface{}{backchannelEventURI: nil} }},
		{"event array", func(m map[string]interface{}) { m["events"] = map[string]interface{}{backchannelEventURI: []string{}} }},
		{"unbounded sid", func(m map[string]interface{}) { m["sid"] = strings.Repeat("a", 201) }},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := good()
			tc.change(m)
			_, err := VerifyLogoutToken(jwtFixture(t, key, m), issuer, "theia-users-bff", []publicKey{jwk}, now)
			if err == nil {
				t.Fatal("malformed logout accepted")
			}
		})
	}
	raw := jwtFixture(t, key, good())
	if _, err := VerifyLogoutToken(raw, issuer, "theia-users-bff", nil, now); err == nil {
		t.Fatal("untrusted key accepted")
	}
	tampered := raw[:len(raw)-4] + "abcd"
	if _, err := VerifyLogoutToken(tampered, issuer, "theia-users-bff", []publicKey{jwk}, now); err == nil {
		t.Fatal("forged signature accepted")
	}
	// Subject-only notifications are required for global subject invalidation.
	m := good()
	delete(m, "sid")
	if _, err := VerifyLogoutToken(jwtFixture(t, key, m), issuer, "theia-users-bff", []publicKey{jwk}, now); err != nil {
		t.Fatalf("subject-only logout invalid: %v", err)
	}
}

func TestAUTH08ProviderRefreshesPinnedKeycloakJWKS(t *testing.T) {
	key, jwk := rsaFixture(t)
	var issuer string
	doc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/realms/theia-users/.well-known/openid-configuration":
			json.NewEncoder(w).Encode(discovery{Issuer: issuer, AuthorizationEndpoint: issuer + "/protocol/openid-connect/auth", TokenEndpoint: issuer + "/protocol/openid-connect/token", JWKSURI: issuer + "/protocol/openid-connect/certs"})
		case "/realms/theia-users/protocol/openid-connect/certs":
			json.NewEncoder(w).Encode(jwks{Keys: []publicKey{jwk}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer doc.Close()
	issuer = doc.URL + "/realms/theia-users"
	p, err := New(context.Background(), Settings{Issuer: issuer, ClientID: "theia-users-bff", ClientSecret: "secret", RedirectURI: "http://app.localhost:8080/auth/user/callback"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	jwt := jwtFixture(t, key, map[string]interface{}{"iss": issuer, "sub": "subject", "sid": "sid", "aud": "theia-users-bff", "iat": now.Unix(), "exp": now.Add(time.Minute).Unix(), "jti": "jti-1", "events": map[string]interface{}{backchannelEventURI: map[string]interface{}{}}})
	event, err := p.VerifyBackchannel(context.Background(), jwt)
	if err != nil || event.Subject != "subject" {
		t.Fatalf("Keycloak JWT validation failed: %v", err)
	}
	if _, err = p.VerifyBackchannel(context.Background(), "attacker"); err == nil {
		t.Fatal("unsigned JWT accepted")
	}
}

func TestAUTH08ProviderRequiresSignedSessionIDForSSO(t *testing.T) {
	key, jwk := rsaFixture(t)
	var issuer string
	doc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/realms/staff/.well-known/openid-configuration":
			json.NewEncoder(w).Encode(discovery{Issuer: issuer, AuthorizationEndpoint: issuer + "/protocol/openid-connect/auth", TokenEndpoint: issuer + "/protocol/openid-connect/token", JWKSURI: issuer + "/protocol/openid-connect/certs"})
		case "/realms/staff/protocol/openid-connect/certs":
			json.NewEncoder(w).Encode(jwks{Keys: []publicKey{jwk}})
		case "/realms/staff/protocol/openid-connect/token":
			now := time.Now().UTC()
			json.NewEncoder(w).Encode(map[string]string{"token_type": "Bearer", "id_token": jwtFixture(t, key, map[string]interface{}{
				"iss": issuer, "sub": "staff-id", "aud": "theia-staff-bff", "iat": now.Unix(), "exp": now.Add(time.Minute).Unix(), "nonce": "login-nonce",
			})})
		default:
			http.NotFound(w, r)
		}
	}))
	defer doc.Close()
	issuer = doc.URL + "/realms/staff"
	p, err := New(context.Background(), Settings{Issuer: issuer, ClientID: "theia-staff-bff", ClientSecret: "secret", RedirectURI: "http://admin.localhost:8080/auth/staff/callback", RequireSID: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = p.Exchange(context.Background(), "valid", "verifier", "login-nonce"); err == nil {
		t.Fatal("ID token without signed sid accepted despite lifecycle requirement")
	}
}
