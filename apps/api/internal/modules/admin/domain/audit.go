package domain

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	// Audit access is a separate admin operation, not Auth's login audit.
	ActionAdminMeRead          = "admin.me.read"
	ActionAuditRead            = "admin.audit.read"
	ActionOverviewRead         = "admin.overview.read"
	ActionTurnsRead            = "admin.turns.read"
	ActionTurnDetailRead       = "admin.turn.detail.read"
	ActionTurnTraceRefsRead    = "admin.turn.trace_refs.read"
	ActionJobsRead             = "admin.jobs.read"
	ActionWorkersRead          = "admin.workers.read"
	ActionEvaluationsRead      = "admin.evaluations.read"
	ActionEvaluationDetailRead = "admin.evaluation.detail.read"
	ActionEvaluationImport     = "admin.evaluation.import"
	ActionBundlesRead          = "admin.bundles.read"
	OutcomeAuthorized          = "authorized"
	DefaultAuditLimit          = 25
	MaxAuditLimit              = 100
	MaxAuditWindow             = 31 * 24 * time.Hour
	MaxAuditCursorSize         = 180
)

var ErrInvalidAudit = errors.New("invalid admin audit request")

// No arbitrary free-text fields: event content is restricted to stable codes,
// an optional opaque resource UUID, and the server-issued request correlation ID.
// Other Admin use cases must add their own explicit action allowlist entries.
var allowedActions = map[string]string{
	ActionAdminMeRead:          "admin",
	ActionAuditRead:            "audit",
	ActionOverviewRead:         "overview",
	ActionTurnsRead:            "turn",
	ActionTurnDetailRead:       "turn",
	ActionTurnTraceRefsRead:    "turn",
	ActionJobsRead:             "job",
	ActionWorkersRead:          "worker",
	ActionEvaluationsRead:      "evaluation",
	ActionEvaluationDetailRead: "evaluation",
	ActionEvaluationImport:     "evaluation",
	ActionBundlesRead:          "bundle",
}

func AllowedAction(action string) bool { _, ok := allowedActions[action]; return ok }

func ResourceKindForAction(action string) (string, bool) {
	kind, ok := allowedActions[action]
	return kind, ok
}

// Event deliberately excludes request path/query, message content, reason text,
// phone/email, session cookie, JWT, and Keycloak subjects.
type AuditEvent struct {
	ID           uuid.UUID
	ActorStaffID uuid.UUID
	ActionCode   string
	ResourceKind string
	ResourceID   uuid.UUID // uuid.Nil means no specific resource
	OutcomeCode  string
	RequestID    uuid.UUID
	RecordedAt   time.Time
}

func (e AuditEvent) Validate() error {
	if e.ID == uuid.Nil || e.ActorStaffID == uuid.Nil || e.RequestID == uuid.Nil {
		return ErrInvalidAudit
	}
	kind, ok := allowedActions[e.ActionCode]
	if !ok || kind != e.ResourceKind {
		return ErrInvalidAudit
	}
	switch e.OutcomeCode {
	case OutcomeAuthorized, "success", "denied", "failed":
		return nil
	default:
		return ErrInvalidAudit
	}
}

// AuditCursor uses the same server-side ordering as SQL Server:
// (recorded_at DESC, id DESC). It is an opaque paging marker, not a credential.
type AuditCursor struct {
	RecordedAt time.Time
	ID         uuid.UUID
}

func (c AuditCursor) Encode() string {
	return base64.RawURLEncoding.EncodeToString([]byte(c.RecordedAt.UTC().Format(time.RFC3339Nano) + "|" + c.ID.String()))
}

func DecodeAuditCursor(value string) (*AuditCursor, error) {
	if value == "" {
		return nil, nil
	}
	if len(value) > MaxAuditCursorSize {
		return nil, ErrInvalidAudit
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(decoded) > 120 {
		return nil, ErrInvalidAudit
	}
	parts := strings.Split(string(decoded), "|")
	if len(parts) != 2 {
		return nil, ErrInvalidAudit
	}
	at, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil || at.IsZero() {
		return nil, ErrInvalidAudit
	}
	id, err := uuid.Parse(parts[1])
	if err != nil || id == uuid.Nil || id.String() != parts[1] {
		return nil, ErrInvalidAudit
	}
	return &AuditCursor{RecordedAt: at.UTC(), ID: id}, nil
}

type AuditFilter struct {
	From         time.Time
	To           time.Time
	Limit        int
	ActorStaffID uuid.UUID
	ActionCode   string
	Cursor       *AuditCursor
}

func (f AuditFilter) Validate() error {
	if f.From.IsZero() || f.To.IsZero() || !f.To.After(f.From) || f.To.Sub(f.From) > MaxAuditWindow || f.Limit < 1 || f.Limit > MaxAuditLimit {
		return ErrInvalidAudit
	}
	if f.ActionCode != "" && !AllowedAction(f.ActionCode) {
		return ErrInvalidAudit
	}
	if f.Cursor != nil && (f.Cursor.ID == uuid.Nil || f.Cursor.RecordedAt.IsZero()) {
		return ErrInvalidAudit
	}
	return nil
}

func NewAuditFilter(now time.Time) AuditFilter {
	end := now.UTC()
	return AuditFilter{From: end.Add(-24 * time.Hour), To: end, Limit: DefaultAuditLimit}
}
