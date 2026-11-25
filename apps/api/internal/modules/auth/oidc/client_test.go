package oidc

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func rsaFixture(t *testing.T) (*rsa.PrivateKey, publicKey) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	e := big.NewInt(int64(key.PublicKey.E)).Bytes()
	return key, publicKey{KeyID: "key-1", KeyType: "RSA", Algorithm: "RS256", Use: "sig", N: base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes()), E: base64.RawURLEncoding.EncodeToString(e)}
}
func jwtFixture(t *testing.T, key *rsa.PrivateKey, payload map[string]interface{}) string {
	t.Helper()
	encode := func(v interface{}) string {
		raw, e := json.Marshal(v)
		if e != nil {
			t.Fatal(e)
		}
		return base64.RawURLEncoding.EncodeToString(raw)
	}
	input := encode(map[string]string{"alg": "RS256", "kid": "key-1", "typ": "JWT"}) + "." + encode(payload)
	d := sha256.Sum256([]byte(input))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, d[:])
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(sig)
}
func TestStrictIDTokenChecks(t *testing.T) {
	key, jwk := rsaFixture(t)
	now := time.Now().UTC()
	basic := func() map[string]interface{} {
		return map[string]interface{}{
			"iss": "https://login.example.invalid/realms/staff", "sub": "staff-subject",
			"aud": "theia-staff-bff", "nonce": "expected-nonce", "iat": now.Unix(), "exp": now.Add(2 * time.Minute).Unix(),
		}
	}
	const issuer = "https://login.example.invalid/realms/staff"
	if sub, _, err := VerifyIDToken(jwtFixture(t, key, basic()), issuer, "theia-staff-bff", "expected-nonce", []publicKey{jwk}, now); err != nil || sub != "staff-subject" {
		t.Fatalf("valid token rejected: %v", err)
	}
	for _, test := range []struct {
		name   string
		modify func(map[string]interface{})
	}{
		{"issuer", func(m map[string]interface{}) { m["iss"] = "https://evil.invalid" }},
		{"audience", func(m map[string]interface{}) { m["aud"] = "theia-users-bff" }},
		{"nonce", func(m map[string]interface{}) { m["nonce"] = "attacker" }},
		{"expired", func(m map[string]interface{}) { m["exp"] = now.Add(-time.Second).Unix() }},
		{"future issued", func(m map[string]interface{}) { m["iat"] = now.Add(2 * time.Minute).Unix() }},
		{"multiple audiences without azp", func(m map[string]interface{}) { m["aud"] = []string{"theia-staff-bff", "other"} }},
		{"wrong azp", func(m map[string]interface{}) { m["azp"] = "evil" }},
		{"missing subject", func(m map[string]interface{}) { m["sub"] = "" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			m := basic()
			test.modify(m)
			_, _, err := VerifyIDToken(jwtFixture(t, key, m), issuer, "theia-staff-bff", "expected-nonce", []publicKey{jwk}, now)
			if err == nil {
				t.Fatal("malformed claims accepted")
			}
		})
	}
	raw := jwtFixture(t, key, basic())
	corrupt := raw[:len(raw)-3] + "abc"
	if _, _, err := VerifyIDToken(corrupt, issuer, "theia-staff-bff", "expected-nonce", []publicKey{jwk}, now); err == nil {
		t.Fatal("signature tampering accepted")
	}
	if _, _, err := VerifyIDToken(raw, issuer, "theia-staff-bff", "expected-nonce", nil, now); err == nil {
		t.Fatal("missing signing key accepted")
	}
}

func TestOIDCDiscoveryExchangeAndPKCE(t *testing.T) {
	key, jwk := rsaFixture(t)
	var issuer string
	seenToken := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/realms/staff/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(discovery{Issuer: issuer, AuthorizationEndpoint: issuer + "/protocol/openid-connect/auth", TokenEndpoint: issuer + "/protocol/openid-connect/token", JWKSURI: issuer + "/protocol/openid-connect/certs"})
		case "/realms/staff/protocol/openid-connect/certs":
			_ = json.NewEncoder(w).Encode(jwks{Keys: []publicKey{jwk}})
		case "/realms/staff/protocol/openid-connect/token":
			id, secret, ok := r.BasicAuth()
			if !ok || id != "staff-bff" || secret != "fake-secret" || r.FormValue("code_verifier") != "test-verifier" || r.FormValue("redirect_uri") != "http://admin.localhost:8080/auth/staff/callback" {
				w.WriteHeader(401)
				return
			}
			seenToken = true
			_ = json.NewEncoder(w).Encode(map[string]string{"token_type": "Bearer", "id_token": jwtFixture(t, key, map[string]interface{}{
				"iss": issuer, "sub": "staff-1", "aud": "staff-bff", "nonce": "test-nonce", "iat": time.Now().Unix(), "exp": time.Now().Add(time.Minute).Unix(),
			})})
		default:
			http.NotFound(w, r)
		}
	})
	server := httptest.NewServer(handler)
	defer server.Close()
	issuer = server.URL + "/realms/staff"
	provider, err := New(context.Background(), Settings{Issuer: issuer, ClientID: "staff-bff", ClientSecret: "fake-secret", RedirectURI: "http://admin.localhost:8080/auth/staff/callback"})
	if err != nil {
		t.Fatal(err)
	}
	authorizationURL := provider.AuthorizationURL("test-state", "test-nonce", "test-verifier")
	u, err := url.Parse(authorizationURL)
	if err != nil {
		t.Fatal(err)
	}
	vals := u.Query()
	if vals.Get("response_type") != "code" || vals.Get("code_challenge_method") != "S256" || vals.Get("state") != "test-state" || vals.Get("code_verifier") != "" {
		t.Fatal("OIDC parameters incorrect")
	}
	expected := sha256.Sum256([]byte("test-verifier"))
	if vals.Get("code_challenge") != base64.RawURLEncoding.EncodeToString(expected[:]) {
		t.Fatal("PKCE wrong")
	}
	subject, _, err := provider.Exchange(context.Background(), "test-code", "test-verifier", "test-nonce")
	if err != nil || subject != "staff-1" || !seenToken {
		t.Fatalf("token exchange failed: %v", err)
	}
	_, _, err = provider.Exchange(context.Background(), "test-code", "wrong-verifier", "test-nonce")
	if err == nil {
		t.Fatal("invalid verifier accepted")
	}
	if !sameOrigin(issuer+"/endpoint", issuer) || sameOrigin("https://evil.invalid", issuer) {
		t.Fatal("endpoint origin error")
	}
	if !strings.Contains(fmt.Sprint(err), "denied") {
		t.Fatal("expected sanitized error")
	}
}

func TestAUTH06UserIDTokenRequiresSignedSMSProof(t *testing.T) {
	key, jwk := rsaFixture(t)
	now := time.Now().UTC()
	const issuer = "https://login.example.invalid/realms/theia-users"
	makeToken := func(proof interface{}) string {
		payload := map[string]interface{}{
			"iss": issuer, "sub": "stable-opaque-user-id", "aud": "theia-users-bff",
			"nonce": "one-time-nonce", "iat": now.Unix(), "exp": now.Add(time.Minute).Unix(),
		}
		if proof != nil {
			payload["theia_sms_verified"] = proof
		}
		return jwtFixture(t, key, payload)
	}
	for _, tc := range []struct {
		name  string
		proof interface{}
		allow bool
	}{
		{"missing", nil, false},
		{"static boolean", true, false},
		{"unverified", "false", false},
		{"wrong version", "sms-otp-v0", false},
		{"valid", "sms-otp-v1", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			token := makeToken(tc.proof)
			sub, _, err := VerifyIDToken(token, issuer, "theia-users-bff", "one-time-nonce", []publicKey{jwk}, now, true)
			if (err == nil) != tc.allow {
				t.Fatalf("proof %q: accepted=%t wanted=%t", tc.name, err == nil, tc.allow)
			}
			if tc.allow && sub != "stable-opaque-user-id" {
				t.Fatal("subject mapping changed")
			}
		})
	}
	// A raw string in an attacker-supplied unsigned payload is never sufficient.
	valid := makeToken("sms-otp-v1")
	parts := strings.Split(valid, ".")
	body, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	body = []byte(strings.Replace(string(body), "sms-otp-v1", "sms-otp-v0", 1))
	tampered := parts[0] + "." + base64.RawURLEncoding.EncodeToString(body) + "." + parts[2]
	if _, _, err := VerifyIDToken(tampered, issuer, "theia-users-bff", "one-time-nonce", []publicKey{jwk}, now, true); err == nil {
		t.Fatal("tampered proof accepted")
	}
}

func TestAUTH06ProviderEnforcesSMSProofOnExchange(t *testing.T) {
	key, jwk := rsaFixture(t)
	var issuer string
	proof := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/realms/theia-users/.well-known/openid-configuration":
			_ = json.NewEncoder(w).Encode(discovery{Issuer: issuer, AuthorizationEndpoint: issuer + "/protocol/openid-connect/auth", TokenEndpoint: issuer + "/protocol/openid-connect/token", JWKSURI: issuer + "/protocol/openid-connect/certs"})
		case "/realms/theia-users/protocol/openid-connect/certs":
			_ = json.NewEncoder(w).Encode(jwks{Keys: []publicKey{jwk}})
		case "/realms/theia-users/protocol/openid-connect/token":
			if r.FormValue("code_verifier") != "my-pkce-verifier" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			claims := map[string]interface{}{"iss": issuer, "sub": "keycloak-uuid", "aud": "theia-users-bff", "nonce": "my-nonce", "iat": time.Now().Unix(), "exp": time.Now().Add(time.Minute).Unix()}
			if proof != "" {
				claims["theia_sms_verified"] = proof
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"token_type": "Bearer", "id_token": jwtFixture(t, key, claims)})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	issuer = server.URL + "/realms/theia-users"
	provider, err := New(context.Background(), Settings{Issuer: issuer, ClientID: "theia-users-bff", ClientSecret: "synthetic", RedirectURI: "http://app.localhost:8080/auth/user/callback", RequireSMSProof: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := provider.Exchange(context.Background(), "code", "my-pkce-verifier", "my-nonce"); err == nil {
		t.Fatal("missing proof permitted user login")
	}
	proof = "sms-otp-v1"
	if sub, _, err := provider.Exchange(context.Background(), "code", "my-pkce-verifier", "my-nonce"); err != nil || sub != "keycloak-uuid" {
		t.Fatalf("SMS proof rejected: %v", err)
	}
}
