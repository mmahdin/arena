// Package retrieval implements a single owner-scoped hybrid search pipeline.
package retrieval

import (
	"context"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"time"
)

type SourceKind string

const (
	SourceMessage     SourceKind = "message"
	SourceMemory      SourceKind = "memory"
	SourceSession     SourceKind = "session"
	SourceThread      SourceKind = "thread"
	SourceFormulation SourceKind = "formulation"
	SourceProtocol    SourceKind = "protocol"
	SourceOpenLoop    SourceKind = "open_loop"
)

func (s SourceKind) Valid() bool {
	switch s {
	case SourceMessage, SourceMemory, SourceSession, SourceThread, SourceFormulation, SourceProtocol, SourceOpenLoop:
		return true
	}
	return false
}

// Candidate retains the original source identity and immutable index revision.
// Vector is absent while indexing is pending; such sources remain keyword searchable.
type Candidate struct {
	Lexical        *LexicalDocument
	ThreadID       *c.ID
	Evidence       []c.EvidenceRef
	ID, UserID     c.ID
	Source         SourceKind
	SourceVersion  string
	ConversationID string
	ValidFrom      *time.Time
	Kind           c.MemoryKind
	Text           string
	Epistemic      c.Epistemic
	Status         c.MemoryStatus
	Sensitivity    c.Sensitivity
	CreatedAt      time.Time
	EvidenceN      int
	Vector         []float32
}
type State struct {
	TZ           string
	MemoryPaused bool
}
type Query struct {
	TermHashes        []string
	UserID, TriggerID c.ID
	Model             string
	Dimension         int
}

// Scan visits the COMPLETE visible owner corpus before the trigger, without
// importance, recency, type or lexical preselection. Adapters may stream rows.
// Authorization is checked before decrypting protected content.
type Repository interface {
	UserState(context.Context, c.ID) (State, error)
	Scan(context.Context, Query, func(Candidate) error) error
}

// Visibility is independent of this mandatory deployment-reviewed AI-use gate.
type UsePolicy interface {
	CanSearch(context.Context, c.ID, c.ID) (bool, error)
	CanUse(context.Context, c.ID, SourceKind, c.ID) (bool, error)
}

// Relationships permits only bounded direct expansion, never another search.
type Relationships interface {
	Evidence(context.Context, Query, Candidate) ([]c.EvidenceRef, *c.ID, error)
	Lookup(context.Context, Query, SourceKind, c.ID) (Candidate, bool, error)
	Baseline(context.Context, c.ID, c.ID) ([]Candidate, error)
}
