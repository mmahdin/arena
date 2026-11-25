package auth

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeStore struct {
	mu         sync.Mutex
	login      map[string]LoginRecord
	sessions   map[string]Session
	userID     uuid.UUID
	allowStaff bool
	audit      []AuditEvent
}

func (f *fakeStore) SaveLogin(_ context.Context, r LoginRecord) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.login == nil {
		f.login = map[string]LoginRecord{}
	}
	f.login[string(r.StateHash)] = r
	return nil
}
func (f *fakeStore) ConsumeLogin(_ context.Context, state []byte) (LoginRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.login[string(state)]
	if !ok {
		return LoginRecord{}, ErrNotFound
	}
	delete(f.login, string(state))
	return r, nil
}
func (f *fakeStore) ResolveIdentity(_ context.Context, kind Kind, issuer, subject string) (uuid.UUID, error) {
	if issuer == "" || subject == "" {
		return uuid.Nil, ErrInvalid
	}
	if kind == Staff && !f.allowStaff {
		return uuid.Nil, ErrInactive
	}
	return f.userID, nil
}
func (f *fakeStore) SaveSession(_ context.Context, s Session) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.sessions == nil {
		f.sessions = map[string]Session{}
	}
	f.sessions[string(s.TokenHash)] = s
	f.audit = append(f.audit, AuditEvent{Kind: s.Kind, PrincipalID: s.PrincipalID, Action: "login", Outcome: "success"})
	return nil
}
func (f *fakeStore) ActiveSession(_ context.Context, kind Kind, token []byte) (Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.sessions[string(token)]
	if !ok || s.Kind != kind || s.ExpiresAt.Before(time.Now()) {
		return Session{}, ErrNotFound
	}
	return s, nil
}
func (f *fakeStore) RevokeSession(_ context.Context, kind Kind, token []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.sessions, string(token))
	return nil
}
func (f *fakeStore) AppendAudit(_ context.Context, e AuditEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.audit = append(f.audit, e)
	return nil
}

type fakeOIDC struct{}

func (fakeOIDC) AuthorizationURL(state, nonce, verifier string) string {
	return "https://keycloak.example/authorize?state=" + state
}
func (fakeOIDC) Exchange(_ context.Context, code, verifier, nonce string) (string, string, error) {
	if code != "valid-code" || len(verifier) < 40 || len(nonce) < 40 {
		return "", "", ErrInvalid
	}
	return "opaque-sub", "provider-sid", nil
}

func serviceFixture(t *testing.T) (*Service, *fakeStore) {
	t.Helper()
	store := &fakeStore{userID: uuid.New()}
	provider := fakeOIDC{}
	s, err := New(store, []byte(strings.Repeat("k", 32)), map[Kind]Realm{
		User:  {Issuer: "https://keycloak.example/realms/users", Origin: "https://app.example", RedirectURI: "https://app.example/auth/user/callback", Provider: provider},
		Staff: {Issuer: "https://keycloak.example/realms/staff", Origin: "https://admin.example", RedirectURI: "https://admin.example/auth/staff/callback", Provider: provider},
	})
	if err != nil {
		t.Fatal(err)
	}
	return s, store
}
func TestOneTimeOIDCStateAndRevocableSession(t *testing.T) {
	ctx := context.Background()
	service, store := serviceFixture(t)
	state, redirect, err := service.Begin(ctx, User)
	if err != nil || len(state) != 43 || !strings.Contains(redirect, state) {
		t.Fatalf("begin failed: %v", err)
	}
	if len(store.login) != 1 {
		t.Fatal("missing stored state")
	}
	token, sess, err := service.Complete(ctx, User, state, "valid-code")
	if err != nil || len(token) != 43 || sess.Kind != User || sess.KeycloakSID != "provider-sid" {
		t.Fatalf("complete failed: %v", err)
	}
	if _, err := service.Authenticate(ctx, User, token); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Authenticate(ctx, Staff, token); !errors.Is(err, ErrNotFound) {
		t.Fatal("cross-realm cookie accepted")
	}
	if _, _, err := service.Complete(ctx, User, state, "valid-code"); !errors.Is(err, ErrNotFound) {
		t.Fatal("state replay accepted")
	}
	if !service.VerifyCSRF(token, service.CSRFToken(token)) || service.VerifyCSRF(token, service.CSRFToken("wrong")) {
		t.Fatal("CSRF comparison failed")
	}
	if err := service.Revoke(ctx, User, token); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Authenticate(ctx, User, token); !errors.Is(err, ErrNotFound) {
		t.Fatal("revoked session accepted")
	}
	if len(store.audit) != 1 || store.audit[0].Action != "login" {
		t.Fatal("auth audit missing")
	}
}
func TestWrongRealmAndInactiveStaffFail(t *testing.T) {
	ctx := context.Background()
	service, _ := serviceFixture(t)
	state, _, err := service.Begin(ctx, User)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.Complete(ctx, Staff, state, "valid-code"); !errors.Is(err, ErrInvalid) {
		t.Fatal("realm mixing accepted")
	}
	// State is consumed even on the wrong-kind callback; cannot be retried.
	if _, _, err := service.Complete(ctx, User, state, "valid-code"); !errors.Is(err, ErrNotFound) {
		t.Fatal("mismatched state replay accepted")
	}
	state, _, err = service.Begin(ctx, Staff)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.Complete(ctx, Staff, state, "valid-code"); !errors.Is(err, ErrInactive) {
		t.Fatal("unprovisioned staff admitted")
	}
}
func TestExpiredLoginAndEncryptedVerifier(t *testing.T) {
	service, store := serviceFixture(t)
	now := time.Now().UTC()
	service.now = func() time.Time { return now }
	state, _, err := service.Begin(context.Background(), User)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range store.login {
		if strings.Contains(string(item.SealedPayload), "verifier") || strings.Contains(string(item.SealedPayload), "nonce") {
			t.Fatal("login verifier not encrypted")
		}
	}
	service.now = func() time.Time { return now.Add(LoginTTL + time.Second) }
	if _, _, err := service.Complete(context.Background(), User, state, "valid-code"); !errors.Is(err, ErrInvalid) {
		t.Fatal("expired state accepted")
	}
}
