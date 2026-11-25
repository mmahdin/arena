package oidc

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"time"
)

var errInvalidToken = errors.New("OIDC identity token invalid")

type claims struct {
	Issuer          string          `json:"iss"`
	Subject         string          `json:"sub"`
	Audience        json.RawMessage `json:"aud"`
	AuthorizedParty string          `json:"azp"`
	Nonce           string          `json:"nonce"`
	ExpiresAt       int64           `json:"exp"`
	IssuedAt        int64           `json:"iat"`
	NotBefore       int64           `json:"nbf"`
	SID             string          `json:"sid"`
	SMSVerified     string          `json:"theia_sms_verified"`
}

type tokenHeader struct {
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid"`
	Type      string `json:"typ"`
}

type publicKey struct {
	KeyID     string   `json:"kid"`
	KeyType   string   `json:"kty"`
	Algorithm string   `json:"alg"`
	Use       string   `json:"use"`
	N         string   `json:"n"`
	E         string   `json:"e"`
	X5c       []string `json:"x5c"`
}

type jwks struct {
	Keys []publicKey `json:"keys"`
}

// VerifyIDToken accepts only RSA/SHA-256 signed ID tokens from a preconfigured
// issuer. It never trusts algorithm or key URLs supplied inside a JWT.
// requireSMS is set only by the fixed User realm settings, never by browser input.
func VerifyIDToken(token, issuer, clientID, nonce string, keys []publicKey, now time.Time, requireSMS ...bool) (string, string, error) {
	if len(token) > 24*1024 || strings.Count(token, ".") != 2 {
		return "", "", errInvalidToken
	}
	parts := strings.Split(token, ".")
	hb, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", "", errInvalidToken
	}
	pb, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", errInvalidToken
	}
	sb, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", "", errInvalidToken
	}
	var h tokenHeader
	if json.Unmarshal(hb, &h) != nil || h.Algorithm != "RS256" || h.KeyID == "" {
		return "", "", errInvalidToken
	}
	var c claims
	if json.Unmarshal(pb, &c) != nil {
		return "", "", errInvalidToken
	}
	if len(requireSMS) > 0 && requireSMS[0] && c.SMSVerified != "sms-otp-v1" {
		return "", "", errInvalidToken
	}
	if c.Issuer != issuer || c.Subject == "" || len(c.Subject) > 200 || c.Nonce == "" || c.Nonce != nonce || c.ExpiresAt <= now.Unix() || c.IssuedAt == 0 || c.IssuedAt > now.Add(60*time.Second).Unix() || (c.NotBefore != 0 && c.NotBefore > now.Add(60*time.Second).Unix()) {
		return "", "", errInvalidToken
	}
	var audiences []string
	if json.Unmarshal(c.Audience, &audiences) != nil {
		var single string
		if json.Unmarshal(c.Audience, &single) != nil {
			return "", "", errInvalidToken
		}
		audiences = []string{single}
	}
	found := false
	for _, aud := range audiences {
		if aud == clientID {
			found = true
		}
	}
	if !found || (len(audiences) > 1 && c.AuthorizedParty != clientID) || (c.AuthorizedParty != "" && c.AuthorizedParty != clientID) {
		return "", "", errInvalidToken
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	for _, jwk := range keys {
		if jwk.KeyID != h.KeyID || jwk.KeyType != "RSA" || (jwk.Algorithm != "" && jwk.Algorithm != "RS256") || (jwk.Use != "" && jwk.Use != "sig") {
			continue
		}
		key, err := rsaKey(jwk)
		if err != nil {
			continue
		}
		if rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], sb) == nil {
			return c.Subject, c.SID, nil
		}
	}
	return "", "", errInvalidToken
}

func rsaKey(key publicKey) (*rsa.PublicKey, error) {
	nb, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil || len(nb) < 256 || len(nb) > 1024 {
		return nil, errInvalidToken
	}
	eb, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil || len(eb) < 1 || len(eb) > 4 {
		return nil, errInvalidToken
	}
	n := new(big.Int).SetBytes(nb)
	e := new(big.Int).SetBytes(eb)
	if !e.IsInt64() || e.Int64() < 3 || e.Int64() > 1<<31-1 || e.Bit(0) == 0 || n.Sign() <= 0 {
		return nil, errInvalidToken
	}
	return &rsa.PublicKey{N: n, E: int(e.Int64())}, nil
}
