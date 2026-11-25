package domain

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestInspectorFiltersAndCursor(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	valid := NewInspectorFilter(now)
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	cursor := InspectorCursor{At: now.Add(-time.Minute), ID: uuid.New()}
	parsed, err := DecodeInspectorCursor(cursor.Encode())
	if err != nil || !parsed.At.Equal(cursor.At) || parsed.ID != cursor.ID {
		t.Fatalf("cursor roundtrip: %+v %v", parsed, err)
	}
	valid.Cursor = parsed
	if err := valid.Validate(); err != nil {
		t.Fatal(err)
	}
	bad := []InspectorFilter{
		{From: now, To: now, Limit: 1},
		{From: now.Add(-32 * 24 * time.Hour), To: now, Limit: 1},
		{From: now.Add(-time.Hour), To: now, Limit: 101},
		{From: now.Add(-time.Hour), To: now, Limit: 1, Status: "deleted"},
		{From: now.Add(-time.Hour), To: now, Limit: 1, Kind: "; DROP TABLE jobs"},
		{From: now.Add(-time.Hour), To: now, Limit: 1, Cursor: &InspectorCursor{At: now.Add(10 * time.Minute), ID: uuid.New()}},
	}
	for _, f := range bad {
		if !errors.Is(f.Validate(), ErrInvalidInspector) {
			t.Fatalf("accepted %+v", f)
		}
	}
	for _, s := range []string{"", "invalid", "bad", strings.Repeat("x", 500), "YWI", "YQ=="} {
		if _, err := DecodeInspectorCursor(s); !errors.Is(err, ErrInvalidInspector) {
			t.Fatalf("accepted cursor %q", s)
		}
	}
}
func TestInspectorSanitizesCodes(t *testing.T) {
	for _, v := range []string{"conversation text", "password=secret", "\nEVAL", strings.Repeat("X", 81), ""} {
		if SafeCode(v) != nil {
			t.Fatalf("unsafe code accepted: %q", v)
		}
	}
	for _, v := range []string{"TIMEOUT", "memory_attempts_exhausted", "llm-429", "service.failed"} {
		if p := SafeCode(v); p == nil || *p != v {
			t.Fatalf("lost stable code %q", v)
		}
	}
}
