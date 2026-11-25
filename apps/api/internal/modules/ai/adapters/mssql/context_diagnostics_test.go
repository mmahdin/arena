package mssql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	mssql "github.com/microsoft/go-mssqldb"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"
	"io"
	"strings"
	"testing"
)

// A connection-local scripted driver exercises the real adapter without network
// access or a new SQL mocking dependency. No shared/global script state.
type diagnosticQuery struct {
	values     []driver.Value
	err        error
	requireSQL string
}
type diagnosticConnector struct {
	queries []diagnosticQuery
	index   int
}

func (c *diagnosticConnector) Connect(context.Context) (driver.Conn, error) {
	return &diagnosticConn{c}, nil
}
func (*diagnosticConnector) Driver() driver.Driver { return diagnosticDriver{} }

type diagnosticDriver struct{}

func (diagnosticDriver) Open(string) (driver.Conn, error) { return nil, errors.New("unsupported") }

type diagnosticConn struct{ script *diagnosticConnector }

func (*diagnosticConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("unsupported") }
func (*diagnosticConn) Begin() (driver.Tx, error)           { return nil, errors.New("unsupported") }
func (*diagnosticConn) Close() error                        { return nil }
func (c *diagnosticConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	if c.script.index >= len(c.script.queries) {
		return nil, errors.New("unexpected query")
	}
	q := c.script.queries[c.script.index]
	c.script.index++
	if q.requireSQL != "" && !strings.Contains(query, q.requireSQL) {
		return nil, errors.New("missing required owner-scoped SQL constraint")
	}
	if q.err != nil {
		return nil, q.err
	}
	return &diagnosticRows{values: q.values}, nil
}

type diagnosticRows struct {
	values   []driver.Value
	consumed bool
}

func (r *diagnosticRows) Columns() []string {
	if r.values == nil {
		return []string{"value"}
	}
	names := make([]string, len(r.values))
	for i := range names {
		names[i] = "value"
	}
	return names
}
func (*diagnosticRows) Close() error { return nil }
func (r *diagnosticRows) Next(out []driver.Value) error {
	if r.consumed || r.values == nil {
		return io.EOF
	}
	copy(out, r.values)
	r.consumed = true
	return nil
}
func diagnosticDB(t *testing.T, queries ...diagnosticQuery) *sql.DB {
	t.Helper()
	db := sql.OpenDB(&diagnosticConnector{queries: queries})
	t.Cleanup(func() { _ = db.Close() })
	return db
}
func diagnosticMap(err error) map[string]any {
	out := map[string]any{}
	fields := observability.ErrorFields(err)
	for i := 0; i+1 < len(fields); i += 2 {
		out[fields[i].(string)] = fields[i+1]
	}
	return out
}

type diagnosticPolicy struct {
	allowed bool
	reason  string
	err     error
}

func (p diagnosticPolicy) CanSearch(context.Context, c.ID, c.ID) (bool, error) { return true, nil }
func (p diagnosticPolicy) CanUse(context.Context, c.ID, retrieval.SourceKind, c.ID) (bool, error) {
	return p.allowed, p.err
}
func (p diagnosticPolicy) ExplainUse(context.Context, c.ID, retrieval.SourceKind, c.ID) (bool, string, error) {
	return p.allowed, p.reason, p.err
}
func TestContextVerifierDenialsKeepSpecificCheckAndDriverCause(t *testing.T) {
	sqlCause := mssql.Error{Number: 229, State: 2, Class: 14, Message: "PRIVATE SQL parameters"}
	for _, tc := range []struct {
		name, reason, check string
		queries             []diagnosticQuery
		cause               error
	}{
		{"missing_turn", "resource_not_visible", "turn_admission", []diagnosticQuery{{}}, sql.ErrNoRows},
		{"db_permission", "db_permission_denied", "turn_admission", []diagnosticQuery{{err: sqlCause}}, sqlCause},
		{"role", "source_role_mismatch", "message_role", []diagnosticQuery{{values: []driver.Value{int64(10)}}, {values: []driver.Value{"assistant", "current", int64(10)}}}, nil},
		{"content", "source_content_mismatch", "message_content", []diagnosticQuery{{values: []driver.Value{int64(10)}}, {values: []driver.Value{"user", "different", int64(10)}}}, nil},
		{"chronology", "source_chronology_invalid", "message_chronology", []diagnosticQuery{{values: []driver.Value{int64(10)}}, {values: []driver.Value{"user", "current", int64(10)}}, {values: []driver.Value{"user", "recent", int64(11)}}}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			user, turn := c.NewID(), c.NewID()
			v, _ := NewSourceVerifier(diagnosticDB(t, tc.queries...), diagnosticPolicy{allowed: true})
			w := c.WorkingContext{Turn: c.TurnView{TurnID: turn, MessageID: c.NewID(), Text: "current"}}
			if tc.name == "chronology" {
				w.Recent = []c.MsgView{{ID: c.NewID(), Role: c.RoleUser, Text: "recent"}}
			}
			err := v.VerifyContext(context.Background(), user, turn, w)
			fields := diagnosticMap(err)
			if fields["reason"] != tc.reason || fields["failed_check"] != tc.check {
				t.Fatal(fields)
			}
			if tc.cause != nil && tc.name != "db_permission" && !errors.Is(err, tc.cause) {
				t.Fatal("cause lost", err)
			}
			var driverCause mssql.Error
			if tc.name == "db_permission" && (!errors.As(err, &driverCause) || driverCause.Number != 229 || fields["db_error_number"] != int32(229)) {
				t.Fatal(fields)
			}
		})
	}
}
func TestPersistedConsentDecisionDoesNotInventRevocation(t *testing.T) {
	for _, tc := range []struct {
		raw     sql.NullString
		reason  string
		allowed bool
		denied  bool
	}{
		{sql.NullString{}, "consent_missing", false, false},
		{sql.NullString{String: `{}`, Valid: true}, "consent_missing", false, false},
		{sql.NullString{String: `{"use":false}`, Valid: true}, "consent_not_granted", false, false},
		{sql.NullString{String: `{"use":true}`, Valid: true}, "", true, false},
		{sql.NullString{String: `PRIVATE malformed`, Valid: true}, "consent_invalid", false, true},
	} {
		allowed, reason, err := consentDecision(tc.raw, "use")
		if allowed != tc.allowed || reason != tc.reason || (err != nil) != tc.denied {
			t.Fatal(allowed, reason, err)
		}
	}
}
