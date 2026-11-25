package oidc

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// ErrInvalidLogoutToken is never accompanied by an untrusted token or claim.
var ErrInvalidLogoutToken = errors.New("OIDC back-channel logout token invalid")

const backchannelEventURI = "http://schemas.openid.net/event/backchannel-logout"

// LogoutEvent is available only after issuer-bound RS256 verification. The
// credential itself, raw subject and Keycloak session ID must never be logged.
type LogoutEvent struct {
	Issuer    string
	Subject   string
	SessionID string
	TokenID   string
	ExpiresAt time.Time
}

type logoutClaims struct {
	Issuer          string                     `json:"iss"`
	Subject         string                     `json:"sub"`
	Audience        json.RawMessage            `json:"aud"`
	AuthorizedParty string                     `json:"azp"`
	IssuedAt        int64                      `json:"iat"`
	ExpiresAt       int64                      `json:"exp"`
	TokenID         string                     `json:"jti"`
	SessionID       string                     `json:"sid"`
	Nonce           json.RawMessage            `json:"nonce"`
	Events          map[string]json.RawMessage `json:"events"`
}

// VerifyLogoutToken validates an OIDC Back-Channel Logout 1.0 token. Unlike an
// ID token it MUST carry the logout event and MUST NOT have a nonce. Replay
// handling is in the SQL transaction, not in this stateless verifier.
func VerifyLogoutToken(token, issuer, clientID string, keys []publicKey, now time.Time) (LogoutEvent, error) {
	invalid := LogoutEvent{}
	if len(token) > 16*1024 || len(token) < 40 || strings.Count(token, ".") != 2 {
		return invalid, ErrInvalidLogoutToken
	}
	parts := strings.Split(token, ".")
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return invalid, ErrInvalidLogoutToken
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return invalid, ErrInvalidLogoutToken
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return invalid, ErrInvalidLogoutToken
	}
	var h tokenHeader
	var c logoutClaims
	if json.Unmarshal(headerBytes, &h) != nil || json.Unmarshal(payloadBytes, &c) != nil ||
		h.Algorithm != "RS256" || h.KeyID == "" ||
		(h.Type != "" && h.Type != "JWT" && h.Type != "logout+jwt") {
		return invalid, ErrInvalidLogoutToken
	}
	if c.Issuer != issuer || c.Subject == "" && c.SessionID == "" ||
		len(c.Subject) > 200 || len(c.SessionID) > 200 ||
		c.TokenID == "" || len(c.TokenID) > 200 || len(c.Nonce) != 0 ||
		c.IssuedAt <= 0 || c.IssuedAt > now.Add(time.Minute).Unix() ||
		c.IssuedAt < now.Add(-5*time.Minute).Unix() || c.ExpiresAt <= now.Unix() ||
		c.ExpiresAt < c.IssuedAt {
		return invalid, ErrInvalidLogoutToken
	}
	var audiences []string
	if json.Unmarshal(c.Audience, &audiences) != nil {
		var single string
		if json.Unmarshal(c.Audience, &single) != nil {
			return invalid, ErrInvalidLogoutToken
		}
		audiences = []string{single}
	}
	matched := false
	for _, a := range audiences {
		if a == clientID {
			matched = true
		}
	}
	if !matched || (len(audiences) > 1 && c.AuthorizedParty != clientID) ||
		(c.AuthorizedParty != "" && c.AuthorizedParty != clientID) {
		return invalid, ErrInvalidLogoutToken
	}
	evt, ok := c.Events[backchannelEventURI]
	if !ok || len(evt) == 0 {
		return invalid, ErrInvalidLogoutToken
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(evt, &object) != nil || object == nil {
		return invalid, ErrInvalidLogoutToken
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	verified := false
	for _, jwk := range keys {
		if jwk.KeyID != h.KeyID || jwk.KeyType != "RSA" ||
			(jwk.Algorithm != "" && jwk.Algorithm != "RS256") ||
			(jwk.Use != "" && jwk.Use != "sig") {
			continue
		}
		key, err := rsaKey(jwk)
		if err == nil && rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature) == nil {
			verified = true
			break
		}
	}
	if !verified {
		return invalid, ErrInvalidLogoutToken
	}
	return LogoutEvent{Issuer: issuer, Subject: c.Subject, SessionID: c.SessionID,
		TokenID: c.TokenID, ExpiresAt: time.Unix(c.ExpiresAt, 0).UTC()}, nil
}
