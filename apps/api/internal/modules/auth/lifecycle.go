package auth

import (
	"context"
	"crypto/sha256"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth/oidc"
)

// LifecycleStore extends the existing auth persistence boundary without
// requiring transitional synthetic Dev Session stores to implement it.
// Production BFF initialization requires this capability when using AUTH-08.
type LifecycleStore interface {
	// RevokeBackchannel atomically consumes an event ID and revokes only the
	// issuer/kind + SID/subject-matched sessions. False means already applied.
	RevokeBackchannel(ctx context.Context, kind Kind, issuer, subject, sid string, eventHash []byte, expiresAt time.Time) (bool, error)
	RevokeAllForPrincipal(ctx context.Context, kind Kind, principal uuid.UUID) error
}

type backchannelVerifier interface {
	VerifyBackchannel(context.Context, string) (oidc.LogoutEvent, error)
}

// BackchannelLogout accepts a signed OP notification, NOT a browser request.
// It never trusts claims until the fixed realm provider verifies its signature.
// LogoutFenceHash binds the fence to a fixed provider realm, principal kind
// and OP session/subject. Only the digest is stored in SQL Server.
func LogoutFenceHash(kind Kind, issuer, scope, value string) []byte {
	digest := sha256.Sum256([]byte(string(kind) + "\x00" + issuer + "\x00" + scope + "\x00" + value))
	return digest[:]
}

func (s *Service) BackchannelLogout(ctx context.Context, kind Kind, token string) (bool, error) {
	realm, ok := s.realms[kind]
	if !ok {
		return false, ErrInvalid
	}
	verifier, ok := realm.Provider.(backchannelVerifier)
	if !ok {
		return false, errors.New("backchannel verifier unavailable")
	}
	repository, ok := s.store.(LifecycleStore)
	if !ok {
		return false, errors.New("session lifecycle store unavailable")
	}
	event, err := verifier.VerifyBackchannel(ctx, token)
	if err != nil {
		if errors.Is(err, oidc.ErrInvalidLogoutToken) {
			return false, ErrInvalid
		}
		return false, err
	}
	if event.Issuer != realm.Issuer || event.TokenID == "" ||
		event.Subject == "" && event.SessionID == "" || !event.ExpiresAt.After(s.now()) {
		return false, ErrInvalid
	}
	// Hash issuer/kind/JTI to avoid storing an identifying JWT or its JTI.
	id := sha256.Sum256([]byte(string(kind) + "\x00" + event.Issuer + "\x00" + event.TokenID))
	return repository.RevokeBackchannel(ctx, kind, event.Issuer, event.Subject, event.SessionID, id[:], event.ExpiresAt)
}

// RevokeAll invalidates every existing Theia application session for the
// authenticated identity (across browsers/devices). It does not revoke other
// Keycloak SSO sessions; Keycloak owns that separate lifecycle.
func (s *Service) RevokeAll(ctx context.Context, kind Kind, sessionToken string) error {
	repository, ok := s.store.(LifecycleStore)
	if !ok {
		return errors.New("session lifecycle store unavailable")
	}
	session, err := s.Authenticate(ctx, kind, sessionToken)
	if err != nil {
		return err
	}
	return repository.RevokeAllForPrincipal(ctx, kind, session.PrincipalID)
}
