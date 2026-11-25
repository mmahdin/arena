package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestAUTH07ReturnPathSealedAndOneTime(t *testing.T) {
	s, store := serviceFixture(t)
	if _, _, err := s.BeginWithReturn(context.Background(), User, "//evil.invalid"); !errors.Is(err, ErrInvalid) {
		t.Fatal("open redirect should fail before creating login state")
	}
	if len(store.login) != 0 {
		t.Fatal("bad return path must not persist transaction")
	}
	state, _, err := s.BeginWithReturn(context.Background(), User, "/conversations/abc?tab=messages")
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range store.login {
		if strings.Contains(string(item.SealedPayload), "conversations") {
			t.Fatal("return path must be encrypted with PKCE material")
		}
	}
	_, _, path, err := s.CompleteWithReturn(context.Background(), User, state, "valid-code")
	if err != nil || path != "/conversations/abc?tab=messages" {
		t.Fatalf("sealed return path was not restored: %s, %v", path, err)
	}
	if _, _, _, err := s.CompleteWithReturn(context.Background(), User, state, "valid-code"); !errors.Is(err, ErrNotFound) {
		t.Fatal("state replay accepted")
	}
	state, _, err = s.Begin(context.Background(), User)
	if err != nil {
		t.Fatal(err)
	}
	_, _, path, err = s.CompleteWithReturn(context.Background(), User, state, "valid-code")
	if err != nil || path != "/" {
		t.Fatalf("missing return path should default to root: %s %v", path, err)
	}
}
