package domain

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAuditEventValidationAndCursor(t *testing.T) {
	good := AuditEvent{ID: uuid.New(), ActorStaffID: uuid.New(), ActionCode: ActionAuditRead,
		ResourceKind: "audit", OutcomeCode: OutcomeAuthorized, RequestID: uuid.New()}
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	bad := good
	bad.ActionCode = "../private/raw-text"
	if !errors.Is(bad.Validate(), ErrInvalidAudit) {
		t.Fatal("untrusted action code accepted")
	}
	bad = good
	bad.ResourceKind = "turn"
	if !errors.Is(bad.Validate(), ErrInvalidAudit) {
		t.Fatal("mismatched resource type accepted")
	}
	bad = good
	bad.RequestID = uuid.Nil
	if !errors.Is(bad.Validate(), ErrInvalidAudit) {
		t.Fatal("missing correlation accepted")
	}
	bad = good
	bad.OutcomeCode = "anything"
	if !errors.Is(bad.Validate(), ErrInvalidAudit) {
		t.Fatal("unbounded outcome accepted")
	}
	when := time.Date(2026, 11, 19, 0, 0, 0, 123000000, time.UTC)
	cursor := AuditCursor{RecordedAt: when, ID: uuid.New()}
	parsed, err := DecodeAuditCursor(cursor.Encode())
	if err != nil || parsed.ID != cursor.ID || !parsed.RecordedAt.Equal(cursor.RecordedAt) {
		t.Fatalf("cursor roundtrip: %+v %v", parsed, err)
	}
	for _, invalid := range []string{"@", strings.Repeat("A", MaxAuditCursorSize+1), "", "c2VjcmV0"} {
		result, err := DecodeAuditCursor(invalid)
		if invalid == "" {
			if result != nil || err != nil {
				t.Fatalf("empty cursor: %v", err)
			}
		} else if !errors.Is(err, ErrInvalidAudit) {
			t.Fatalf("bad cursor accepted: %s", invalid)
		}
	}
}

func TestAuditFilterBounds(t *testing.T) {
	f := NewAuditFilter(time.Now())
	if err := f.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []AuditFilter{
		{From: f.From, To: f.To, Limit: 0},
		{From: f.From, To: f.To, Limit: 101},
		{From: f.To, To: f.From, Limit: 10},
		{From: f.From.Add(-MaxAuditWindow), To: f.To, Limit: 10},
		{From: f.From, To: f.To, Limit: 10, ActionCode: "free-form-text"},
		{From: f.From, To: f.To, Limit: 10, Cursor: &AuditCursor{}},
	} {
		if !errors.Is(bad.Validate(), ErrInvalidAudit) {
			t.Fatalf("unbounded audit query accepted: %+v", bad)
		}
	}
}
