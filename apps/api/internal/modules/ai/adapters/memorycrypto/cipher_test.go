package memorycrypto

import (
	"bytes"
	"context"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"testing"
)

func TestOwnerPurposeTamperAndRotation(t *testing.T) {
	old := bytes.Repeat([]byte{1}, 32)
	current := bytes.Repeat([]byte{2}, 32)
	k, e := New("old", map[string][]byte{"old": old})
	if e != nil {
		t.Fatal(e)
	}
	ctx := context.Background()
	user := c.NewID()
	plain := []byte("synthetic private memory")
	raw, e := k.Seal(ctx, user, "memory-item:1", plain)
	if e != nil || bytes.Contains(raw, plain) {
		t.Fatal("encryption failed")
	}
	rotated, _ := New("new", map[string][]byte{"old": old, "new": current})
	opened, e := rotated.Open(ctx, user, "memory-item:1", raw)
	if e != nil || !bytes.Equal(opened, plain) {
		t.Fatal("rotation lost retained data")
	}
	for _, tc := range []struct {
		user    c.ID
		purpose string
		raw     []byte
	}{{c.NewID(), "memory-item:1", raw}, {user, "memory-task:1", raw}, {user, "memory-item:1", append(append([]byte(nil), raw[:len(raw)-1]...), raw[len(raw)-1]^1)}, {user, "memory-item:1", []byte("m1:old:x")}} {
		if _, e = rotated.Open(ctx, tc.user, tc.purpose, tc.raw); e == nil {
			t.Fatal("unauthenticated ciphertext opened")
		}
	}
	if _, e = New("bad", map[string][]byte{"bad": []byte("short")}); e == nil {
		t.Fatal("accepted weak key")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, e = k.Open(cancelled, user, "memory-item:1", raw); e == nil {
		t.Fatal("ignored cancellation")
	}
}
