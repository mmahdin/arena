// Package auth owns browser authentication and opaque application sessions.
// It deliberately does not know about AI Core or administrative operations.
package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Kind string

const (
	User  Kind = "user"
	Staff Kind = "staff"
)

var (
	ErrNotFound = errors.New("auth record not found")
	ErrInactive = errors.New("identity inactive or not provisioned")
	ErrInvalid  = errors.New("invalid authentication data")
)

// LoginRecord is one-time state. Payload is AES-GCM encrypted by the application,
// never persisted as an unencrypted code verifier or nonce.
type LoginRecord struct {
	StateHash      []byte
	Kind           Kind
	ExpectedOrigin string
	RedirectURI    string
	SealedPayload  []byte
	ExpiresAt      time.Time
}

type Session struct {
	TokenHash      []byte
	Kind           Kind
	PrincipalID    uuid.UUID
	Issuer         string
	Subject        string
	KeycloakSID    string
	CreatedAt      time.Time
	LoginStartedAt time.Time // OIDC transaction start; protects delayed callback against logout fences
	IdleExpiresAt  time.Time
	ExpiresAt      time.Time
	IdleTTL        time.Duration
}

type AuditEvent struct {
	Kind        Kind
	PrincipalID uuid.UUID // uuid.Nil for unsuccessful or unauthenticated attempts
	Action      string    // fixed allowlisted codes; no user input or sensitive identifiers
	Outcome     string
	RequestID   string
}

// Store is intentionally narrow, making the OIDC and HTTP flows independently
// testable without SQL Server, and enforcing all eligibility checks in the SQL adapter.
type Store interface {
	SaveLogin(ctx context.Context, item LoginRecord) error
	ConsumeLogin(ctx context.Context, stateHash []byte) (LoginRecord, error)
	ResolveIdentity(ctx context.Context, kind Kind, issuer, subject string) (uuid.UUID, error)
	SaveSession(ctx context.Context, item Session) error
	ActiveSession(ctx context.Context, kind Kind, tokenHash []byte) (Session, error)
	RevokeSession(ctx context.Context, kind Kind, tokenHash []byte) error
	AppendAudit(ctx context.Context, event AuditEvent) error
}
