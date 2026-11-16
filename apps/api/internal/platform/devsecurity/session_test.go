package devsecurity

import (
	"github.com/google/uuid"
	"net/http"
	"testing"
	"time"
)

func TestSessionTamperingAndExpiry(t *testing.T) {
	s, err := NewSigner("0123456789abcdef0123456789abcdef", false)
	if err != nil {
		t.Fatal(err)
	}
	uid, sid := uuid.New(), uuid.New()
	now := time.Now().UTC()
	token := s.Issue(uid, sid, now)
	got, err := s.Verify(token, now)
	if err != nil || got.UserID != uid || got.SessionID != sid {
		t.Fatalf("valid token failed: %+v %v", got, err)
	}
	if _, err := s.Verify(token+"a", now); err == nil {
		t.Fatal("tampering accepted")
	}
	if _, err := s.Verify(token, now.Add(SessionTTL)); err == nil {
		t.Fatal("expired accepted")
	}
	if _, err := s.Verify(token, now.Add(-time.Minute)); err != nil {
		t.Fatal("premature expiry", err)
	}
	cookie := s.Cookie(token)
	if !cookie.HttpOnly || cookie.Path != "/api/v1" || cookie.SameSite != http.SameSiteLaxMode || cookie.MaxAge != int(SessionTTL.Seconds()) {
		t.Fatal("cookie policy mismatch")
	}
	if s.ExpireCookie().MaxAge >= 0 {
		t.Fatal("logout cookie must expire")
	}
}

func TestSignedSessionRequiresSessionID(t *testing.T) {
	s, _ := NewSigner("0123456789abcdef0123456789abcdef", false)
	token := s.Issue(uuid.New(), uuid.Nil, time.Now())
	if _, err := s.Verify(token, time.Now()); err == nil {
		t.Fatal("missing session id accepted")
	}
}
