package auth

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth/oidc"
)

type lifecycleOIDC struct {
	fakeOIDC
	event oidc.LogoutEvent
	fail  bool
}

func (p lifecycleOIDC) VerifyBackchannel(_ context.Context, token string) (oidc.LogoutEvent, error) {
	if p.fail || token != "signed-by-Keycloak" {
		return oidc.LogoutEvent{}, oidc.ErrInvalidLogoutToken
	}
	return p.event, nil
}

type lifecycleMemory struct {
	*fakeStore
	mu     sync.Mutex
	seen   map[string]bool
	fences map[string]bool
}

func (m *lifecycleMemory) RevokeBackchannel(_ context.Context, kind Kind, issuer, subject, sid string, eventHash []byte, _ time.Time) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.seen == nil {
		m.seen = map[string]bool{}
	}
	if m.fences == nil {
		m.fences = map[string]bool{}
	}
	if m.seen[string(eventHash)] {
		return false, nil
	}
	m.seen[string(eventHash)] = true
	scope, value := "sid", sid
	if sid == "" {
		scope, value = "subject", subject
	}
	m.fences[string(LogoutFenceHash(kind, issuer, scope, value))] = true
	m.fakeStore.mu.Lock()
	defer m.fakeStore.mu.Unlock()
	for key, s := range m.fakeStore.sessions {
		if s.Kind != kind || s.Issuer != issuer {
			continue
		}
		if sid != "" && s.KeycloakSID != sid {
			continue
		}
		if subject != "" && s.Subject != subject {
			continue
		}
		delete(m.fakeStore.sessions, key)
	}
	return true, nil
}
func (m *lifecycleMemory) RevokeAllForPrincipal(_ context.Context, kind Kind, p uuid.UUID) error {
	m.fakeStore.mu.Lock()
	defer m.fakeStore.mu.Unlock()
	for key, s := range m.fakeStore.sessions {
		if s.Kind == kind && s.PrincipalID == p {
			delete(m.fakeStore.sessions, key)
		}
	}
	return nil
}
func TestAUTH08BackchannelRealmIsolationAndReplay(t *testing.T) {
	ctx := context.Background()
	store := &lifecycleMemory{fakeStore: &fakeStore{userID: uuid.New()}}
	event := oidc.LogoutEvent{Issuer: "https://keycloak.example/realms/users", Subject: "opaque-sub", SessionID: "provider-sid", TokenID: "event-1", ExpiresAt: time.Now().Add(time.Minute)}
	provider := lifecycleOIDC{event: event}
	s, err := New(store, []byte(strings.Repeat("k", 32)), map[Kind]Realm{
		User:  {Issuer: event.Issuer, Origin: "http://app.localhost:8080", RedirectURI: "http://app.localhost:8080/auth/user/callback", Provider: provider},
		Staff: {Issuer: "https://keycloak.example/realms/staff", Origin: "http://admin.localhost:8080", RedirectURI: "http://admin.localhost:8080/auth/staff/callback", Provider: provider},
	})
	if err != nil {
		t.Fatal(err)
	}
	state, _, err := s.Begin(ctx, User)
	if err != nil {
		t.Fatal(err)
	}
	token, session, err := s.Complete(ctx, User, state, "valid-code")
	if err != nil {
		t.Fatal(err)
	}
	if session.LoginStartedAt.IsZero() {
		t.Fatal("callback start time not retained for fence")
	}
	if _, err := s.BackchannelLogout(ctx, Staff, "signed-by-Keycloak"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("cross-realm event accepted: %v", err)
	}
	if _, err := s.Authenticate(ctx, User, token); err != nil {
		t.Fatal("wrong realm revoked user")
	}
	applied, err := s.BackchannelLogout(ctx, User, "signed-by-Keycloak")
	if err != nil || !applied {
		t.Fatalf("first logout not applied: %v", err)
	}
	if _, err = s.Authenticate(ctx, User, token); !errors.Is(err, ErrNotFound) {
		t.Fatalf("revoked user accepted: %v", err)
	}
	applied, err = s.BackchannelLogout(ctx, User, "signed-by-Keycloak")
	if err != nil || applied {
		t.Fatalf("replayed event was not a no-op: %v", err)
	}
	if len(store.fences) != 1 || len(store.seen) != 1 {
		t.Fatal("durable event/fence not recorded")
	}
	if _, err = s.BackchannelLogout(ctx, User, "forgery"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("forged token accepted: %v", err)
	}
}
func TestAUTH08AllSessionsAndSubjectOnlyLogout(t *testing.T) {
	ctx := context.Background()
	store := &lifecycleMemory{fakeStore: &fakeStore{userID: uuid.New()}}
	event := oidc.LogoutEvent{Issuer: "https://keycloak.example/realms/users", Subject: "opaque-sub", TokenID: "subject-only", ExpiresAt: time.Now().Add(time.Minute)}
	s, err := New(store, []byte(strings.Repeat("k", 32)), map[Kind]Realm{
		User:  {Issuer: event.Issuer, Origin: "http://app.localhost:8080", RedirectURI: "http://app.localhost:8080/auth/user/callback", Provider: lifecycleOIDC{event: event}},
		Staff: {Issuer: "https://keycloak.example/realms/staff", Origin: "http://admin.localhost:8080", RedirectURI: "http://admin.localhost:8080/auth/staff/callback", Provider: fakeOIDC{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	create := func() string {
		state, _, err := s.Begin(ctx, User)
		if err != nil {
			t.Fatal(err)
		}
		token, _, err := s.Complete(ctx, User, state, "valid-code")
		if err != nil {
			t.Fatal(err)
		}
		return token
	}
	first, second := create(), create()
	if err := s.RevokeAll(ctx, User, first); err != nil {
		t.Fatal(err)
	}
	for _, tok := range []string{first, second} {
		if _, err := s.Authenticate(ctx, User, tok); !errors.Is(err, ErrNotFound) {
			t.Fatal("all-device Theia revocation failed")
		}
	}
	third := create()
	if _, err := s.BackchannelLogout(ctx, User, "signed-by-Keycloak"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Authenticate(ctx, User, third); !errors.Is(err, ErrNotFound) {
		t.Fatal("subject-only Keycloak logout did not revoke session")
	}
}
