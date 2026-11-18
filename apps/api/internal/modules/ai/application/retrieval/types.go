// Package retrieval implements WP-09's deterministic, owner-scoped read path.
// It never creates memories, decides clinical policy, or changes retained data.
package retrieval

import (
	"context"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

type SourceKind string

const (
	SourceMessage SourceKind = "message"
	SourceMemory  SourceKind = "memory"
)

// Candidate is a read projection with provenance; text from protected/encrypted
// records is not returned by the SQL adapter without a separately approved
// decryptor. A missing evidence link is not a valid memory citation.
type Candidate struct {
	ID          contract.ID
	UserID      contract.ID
	Source      SourceKind
	Kind        contract.MemoryKind
	Text        string
	Epistemic   contract.Epistemic
	Status      contract.MemoryStatus
	Sensitivity contract.Sensitivity
	Importance  float64
	CreatedAt   time.Time
	ValidFrom   *time.Time
	EvidenceN   int
	ThreadIDs   []contract.ID
	Entities    []contract.EntityRef
	ClusterID   *contract.ID
	DeletedAt   *time.Time
	Vector      []float32 // v1, validated by the SQL Server embedding codec
}

type State struct {
	TZ           string
	MemoryPaused bool
}

type MemoryQuery struct {
	UserID      contract.ID
	TriggerID   contract.ID
	FocusThread *contract.ID
	ThreadOnly  bool
	EntityRole  string // canonical role, not an inferred person alias
	EntityLabel string // only an explicitly provided label
	Since       *time.Time
	LexicalTerm string
	Model       string
	Dimension   int
	Limit       int
}

type MessageQuery struct {
	UserID    contract.ID
	TriggerID contract.ID
	From      time.Time
	To        time.Time // exclusive, UTC
	Limit     int
}

// Repository must scope EVERY query to a verified owner and to message order
// strictly before the admitted trigger. Deleted messages/parents are invisible
// to raw reads; derived memory visibility is independent of parent deletion.
type Repository interface {
	UserState(context.Context, contract.ID) (State, error)
	Memories(context.Context, MemoryQuery) ([]Candidate, error)
	Messages(context.Context, MessageQuery) ([]Candidate, error)
	RecentBefore(context.Context, contract.ID, contract.ID, int) ([]string, error)
}

// UsePolicy is a required, externally approved AI-use authorization dependency.
// It is intentionally NOT implemented from deleted_at, memory_paused, or a
// locally invented consent rule. Neither retention nor visibility grants use.
// Both methods must be owner-aware and return deny on ambiguous permission.
type UsePolicy interface {
	CanSearch(ctx context.Context, owner contract.ID, turn contract.ID) (bool, error)
	CanUse(ctx context.Context, owner contract.ID, kind SourceKind, id contract.ID) (bool, error)
}
