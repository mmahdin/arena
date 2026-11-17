// Package contextstate defines WP-08's read-only state projection. SQL writes
// remain exclusively with the WP-02 Conversation-owned TurnFinalizer UoW.
package contextstate

import (
	"context"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// Snapshot is a per-turn, owned read model. Every record is scoped by the
// authenticated user; loaders must never return state from another user.
// ThreadUserTurns is deliberately zero when a reliable per-thread count is
// unavailable: readiness checks must fail closed, never substitute session turns.
type Snapshot struct {
	Session         c.SessionRecord
	User            c.UserState
	Threads         []c.ThreadRecord
	Protocol        *c.ProtocolInstance
	OpenLoops       []c.OpenLoop
	FollowUps       []c.FollowUp
	Formulation     *c.FormulationBrief
	ThreadUserTurns map[c.ID]int
}

// Reader is implemented by a WP-08-owned SQL read adapter. It never reads
// prototype AI state, creates sessions, acquires leases or writes to SQL.
type Reader interface {
	Read(ctx context.Context, userID, conversationID c.ID) (Snapshot, error)
	ForTurn(ctx context.Context, userID, turnID c.ID) (Snapshot, error)
	// UserEvidence returns IDs of existing, user-owned messages with role=user.
	// Soft-deleted messages remain stored but are not eligible as new evidence.
	// Broader memory-use consent rules belong to WP-12.
	UserEvidence(ctx context.Context, userID c.ID, ids []c.ID) (map[c.ID]bool, error)
	// ExistingHypotheses verifies that IDs reference active, user-owned
	// hypothesis rows; no psychological content is read or modified.
	ExistingHypotheses(ctx context.Context, userID c.ID, ids []c.ID) (map[c.ID]bool, error)
	Completed(ctx context.Context, userID, turnID c.ID) (bool, error)
	// MissingFinalizations counts completed turns missing their atomic state
	// context record. It is an audit signal, not permission to rewrite history.
	MissingFinalizations(ctx context.Context, userID c.ID) (int, error)
}

func Versions(s Snapshot) c.Versions {
	v := c.Versions{Session: s.Session.Version, User: s.User.Version}
	if s.Session.FocusThreadID != nil {
		for _, t := range s.Threads {
			if t.ID == *s.Session.FocusThreadID {
				v.Thread = t.Version
				break
			}
		}
	}
	if s.Protocol != nil {
		v.Protocol = s.Protocol.Version
	}
	return v
}

func FindThread(s Snapshot, id c.ID) (c.ThreadRecord, bool) {
	for _, t := range s.Threads {
		if t.ID == id && t.UserID == s.User.UserID {
			return t, true
		}
	}
	return c.ThreadRecord{}, false
}
