package productevents

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type captured struct {
	calls int
	query string
	args  []any
	err   error
}

func (x *captured) ExecContext(_ context.Context, q string, args ...any) (sql.Result, error) {
	x.calls++
	x.query = q
	x.args = args
	return nil, x.err
}
func TestEventVocabularyAndUserSubject(t *testing.T) {
	valid := []Name{UserRegistered, OnboardingStarted, OnboardingCompleted, ConversationStarted,
		AITurnAdmitted, AITurnCompleted, AITurnFailed, BookmarkSaved}
	for _, n := range valid {
		if !n.Valid() {
			t.Fatalf("%q rejected", n)
		}
	}
	for _, n := range []Name{"", "conversation_deleted", "user_id", "user_registered; DROP TABLE x"} {
		if n.Valid() {
			t.Fatalf("unsafe event %q accepted", n)
		}
	}
	user, other := uuid.New(), uuid.New()
	for _, n := range []Name{UserRegistered, OnboardingStarted, OnboardingCompleted} {
		c := &captured{}
		if err := record(context.Background(), c, user, n, other); !errors.Is(err, ErrInvalidEvent) || c.calls != 0 {
			t.Fatalf("%q allowed wrong subject: %v", n, err)
		}
	}
}
func TestRecordUsesOnlyServerEnumsAndSyntheticEligibleSQL(t *testing.T) {
	user, subject := uuid.New(), uuid.New()
	sink := &captured{}
	if err := record(context.Background(), sink, user, AITurnCompleted, subject); err != nil {
		t.Fatal(err)
	}
	if sink.calls != 1 || len(sink.args) != 4 {
		t.Fatalf("unexpected arguments: %+v", sink.args)
	}
	if sink.args[1] != user.String() || sink.args[2] != string(AITurnCompleted) || sink.args[3] != subject.String() {
		t.Fatalf("unsafe event values: %+v", sink.args)
	}
	for _, want := range []string{"u.is_test=1", "u.deleted_at IS NULL", "NOT EXISTS", "UPDLOCK,HOLDLOCK", "dbo.product_events"} {
		if !strings.Contains(sink.query, want) {
			t.Fatalf("missing SQL guard %q", want)
		}
	}
	for _, prohibited := range []string{"content", "request_body", "properties_json", "phone", "prompt", "response", "user_agent"} {
		if strings.Contains(strings.ToLower(sink.query), prohibited) {
			t.Fatalf("payload-bearing SQL %q", prohibited)
		}
	}
}
func TestRecordRejectsInvalidAndPropagatesSQLFailure(t *testing.T) {
	sink := &captured{}
	if err := record(context.Background(), sink, uuid.Nil, AITurnAdmitted, uuid.New()); !errors.Is(err, ErrInvalidEvent) {
		t.Fatal(err)
	}
	if err := record(context.Background(), sink, uuid.New(), Name("injected"), uuid.New()); !errors.Is(err, ErrInvalidEvent) {
		t.Fatal(err)
	}
	if sink.calls != 0 {
		t.Fatal("invalid payload reached SQL")
	}
	expected := errors.New("sql unavailable")
	sink.err = expected
	if err := record(context.Background(), sink, uuid.New(), BookmarkSaved, uuid.New()); !errors.Is(err, expected) {
		t.Fatal("SQL failures must abort enclosing business transaction", err)
	}
	if err := Record(context.Background(), nil, uuid.New(), BookmarkSaved, uuid.New()); !errors.Is(err, ErrInvalidEvent) {
		t.Fatal(err)
	}
}
