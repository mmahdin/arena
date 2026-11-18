// Package memory implements WP-12's extraction, deterministic reconciliation,
// explicit user controls and leased background processing. It owns no HTTP API.
package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"golang.org/x/text/unicode/norm"
)

var ErrConflict = errors.New("memory: concurrent change or stale lease")
var ErrAssets = errors.New("memory: approved assets and explicit dependencies required")
var ErrDenied = errors.New("memory: operation unavailable")

type Action string

const (
	Reflect           Action = "reflect"   // independently approved psychological reflection
	Proactive         Action = "proactive" // independently authorized scheduled interaction
	Extract           Action = "extract"
	UseSource         Action = "use_source"
	UseRetainedSource Action = "use_retained_source" // separate, explicit eligibility for hidden raw sources
	Store             Action = "store"
	Sensitive         Action = "sensitive"
	ExplicitChange    Action = "explicit_change"
	Endorse           Action = "endorse"
)

// Policy MUST be an explicitly approved, deterministic authorization policy.
// It receives current consent; visibility and retention never imply permission.
// ExplicitChange/Endorse must be attested user actions, not model reason tags.
// Implementations must not perform network/model work or independent DB writes.
type Policy interface {
	Allow(context.Context, Permission) (bool, error)
}
type Permission struct {
	Action           Action
	User             c.UserState
	TurnID, SourceID c.ID
	SourceKind       string
	Proposal         *c.MemoryOpProposal
	Origin           c.ProvenanceSrc
	Evidence         []Source
	Memory           *c.MemoryItem
}
type Cipher interface {
	Seal(context.Context, c.ID, string, []byte) ([]byte, error)
	Open(context.Context, c.ID, string, []byte) ([]byte, error)
}
type Meta struct {
	SourceModels                map[string]string // authenticated async proposal hash -> actual extraction model; no authority override
	UserID, TurnID, OperationID c.ID
	Bundle                      c.BundleRef
	Job                         *Lease
	Consolidate                 bool
	MemoryTags                  []string
	Targets                     []c.ID
	RequireCompleted            bool
	IncludeInactive             bool
	ThreadID                    *c.ID
	Feedback                    Feedback
	Signals                     []c.HypothesisSignal
}
type metaKey struct{}

func WithMeta(ctx context.Context, m Meta) context.Context {
	raw, _ := json.Marshal(m)
	var copy Meta
	_ = json.Unmarshal(raw, &copy)
	return context.WithValue(ctx, metaKey{}, copy)
}
func Metadata(ctx context.Context) (Meta, bool) { m, ok := ctx.Value(metaKey{}).(Meta); return m, ok }
func Auth(ctx context.Context, user c.ID) (Meta, error) {
	m, ok := Metadata(ctx)
	if !ok || !c.ValidID(user) || m.UserID != user || !c.ValidID(m.OperationID) {
		return m, ErrDenied
	}
	return m, nil
}

type Record struct {
	Item           c.MemoryItem
	Version        int64
	Pinned         bool
	DeletedAt      *time.Time
	DeletedBy      *c.ID
	DeletionReason *string
	Vector         []float32
	VectorModel    string
}
type Source struct {
	Message   c.Message
	SessionID c.ID
	Hidden    bool
	Future    bool
}
type EventSource struct {
	Actor, Type string
	At          time.Time
	SessionID   c.ID
}
type Receipt struct {
	Hash     string
	Applied  []c.ID
	Rejected []c.DeltaRejection
}
type Snapshot struct {
	ThreadID   *c.ID
	User       c.UserState
	Revision   int64
	Records    []Record
	Sources    map[c.ID]Source
	Events     map[c.ID]EventSource
	Suppressed map[string]bool
	Cooldowns  map[string]time.Time
	Ledger     map[string]Receipt
}
type Mutation struct {
	Before *Record
	After  Record
}
type Cooldown struct {
	Tag    string
	Item   c.ID
	Thread *c.ID
	Until  time.Time
}
type Plan struct {
	Changes   []Mutation
	Receipts  []Receipt
	Loops     []c.ID
	Cooldowns []Cooldown
	Report    ports.MemoryApplyReport
}

// Repository owns the short SQL transactions. No transaction is held while
// calling a model, embedding provider, encryption backend or policy service.
type Repository interface {
	Selection(context.Context, c.ID, c.ID, c.ID) (Snapshot, error)
	Snapshot(context.Context, c.ID, Meta, []c.ID) (Snapshot, error)
	Receipt(context.Context, c.ID, c.ID, string) (ports.MemoryApplyReport, bool, error)
	Save(context.Context, Meta, Snapshot, Plan, string) error
	Control(context.Context, Meta, Control) (ControlResult, error)
	WriterInput(context.Context, Meta, Task) (ports.MemWriterInput, error)
	VerifyWriter(context.Context, Meta, ports.MemWriterInput) error
}
type Control struct {
	Kind          string
	Target        c.ID
	Correction    ports.CorrectionKind
	Scope         ports.DeletionScope
	Pause         *bool
	Consent       map[string]bool
	Prefs         *c.PrefsView
	Pinned        *bool
	ClaimHash     string
	SuppressUntil *time.Time
	CooldownDays  int
}
type ControlResult struct{ Deleted ports.DeletionReport }
type Queue interface {
	Claim(context.Context, string, time.Duration, c.BundleRef, int) (*Lease, error)
	Checkpoint(context.Context, Lease, []byte) error
	Fail(context.Context, Lease, string, time.Duration) error
	Complete(context.Context, Lease) error
	Enqueue(context.Context, c.ID, c.ID, c.AsyncJobProposal) error
	PublishEmbedding(context.Context, Lease) error
	RecordFeedback(context.Context, Lease) error
}
type Lease struct {
	ID, UserID, TurnID, Token c.ID
	Kind                      string
	Attempts                  int
	Payload                   []byte
	Checkpoint                []byte
}
type Task struct {
	Schema      string         `json:"schema"`
	Bundle      c.BundleRef    `json:"bundle"`
	OperationID c.ID           `json:"operation_id"`
	Data        []byte         `json:"data_enc"`
	Analysis    c.TurnAnalysis `json:"-"`
	Claims      []c.UserClaim  `json:"-"`
	MemoryTags  []string       `json:"-"`
	SourceIDs   []c.ID         `json:"-"`
	Consolidate bool           `json:"consolidate,omitempty"`
}
type taskData struct {
	Signals    []c.HypothesisSignal
	Feedback   Feedback
	ThreadID   *c.ID
	Analysis   c.TurnAnalysis
	Claims     []c.UserClaim
	MemoryTags []string
	SourceIDs  []c.ID
}
type Checkpoint struct {
	Output c.MemoryWriterOutput
	Usage  c.LLMUsage
}
type Metrics interface {
	Observe(string, time.Duration, c.LLMUsage, []c.DeltaRejection)
}

// FeedbackPolicy is a required technical maintenance configuration supplied at
// composition, not psychological knowledge or model-selected policy.
type FeedbackPolicy struct {
	MinUnusedRetrievals  int
	ImportanceMultiplier float32
}
type Feedback struct{ Retrieved, Referenced []c.ID }
type FeedbackTask struct {
	Schema   string      `json:"schema"`
	Bundle   c.BundleRef `json:"bundle"`
	Feedback Feedback    `json:"feedback"`
}
type MaintenanceTask struct {
	Schema string      `json:"schema"`
	Bundle c.BundleRef `json:"bundle"`
	At     time.Time   `json:"at"`
}

func Canonical(text string) string {
	return strings.ToLower(strings.Join(strings.Fields(norm.NFC.String(text)), " "))
}
func Hash(v any) string {
	raw, _ := json.Marshal(v)
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
}
func ClaimHash(text string) string { return Hash(Canonical(text)) }
func allowed(ctx context.Context, p Policy, s Snapshot, m Meta, a Action, kind string, id c.ID, op *c.MemoryOpProposal, origin c.ProvenanceSrc) bool {
	permission := Permission{Action: a, User: s.User, TurnID: m.TurnID, SourceID: id, SourceKind: kind, Proposal: op, Origin: origin}
	if op != nil {
		for _, ref := range op.EvidenceMsg {
			if source, ok := s.Sources[ref]; ok && source.Message.UserID == m.UserID {
				permission.Evidence = append(permission.Evidence, source)
			}
		}
	}
	for _, record := range s.Records {
		if record.Item.ID == id {
			copy := record.Item
			permission.Memory = &copy
			break
		}
	}
	ok, err := p.Allow(ctx, permission)
	return err == nil && ok
}
