// Package reflection implements WP-14 psychological reflection. It never emits
// user-facing text and never uses the unrelated daily-reflection feature.
package reflection

import (
	"context"
	"time"

	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type Task struct {
	BackgroundData []byte      `json:"background_data,omitempty"`
	TurnID         c.ID        `json:"turn_id,omitempty"`
	Schema         string      `json:"schema"`
	Bundle         c.BundleRef `json:"bundle"`
	UserID         c.ID        `json:"user_id"`
	ThreadID       c.ID        `json:"thread_id"`
	SessionID      c.ID        `json:"session_id"`
	Trigger        string      `json:"trigger"`
}

// Snapshot is bounded and owner-scoped. Sources include session membership and
// independent visibility metadata; neither implies permission for AI use.
type Snapshot struct {
	User                                  c.UserState
	Thread                                c.ThreadRecord
	Session                               c.SessionRecord
	Items                                 []c.MemoryItem
	Sources                               []mem.Source
	Previous                              *c.FormulationDoc
	Cooldowns                             map[string]time.Time
	Revision                              int64
	ThreadTurns, NewItems, Contradictions int
}
type Commit struct {
	Task     Task
	Snapshot Snapshot
	Result   c.ReflectionResult
	Usage    c.LLMUsage
	Rejected []c.DeltaRejection
}
type Repository interface {
	Load(context.Context, Task) (Snapshot, error)
	Verify(context.Context, Task, Snapshot) error
	ClaimReflection(context.Context, string, time.Duration, c.BundleRef, int) (*mem.Lease, error)
	Checkpoint(context.Context, mem.Lease, []byte) error
	Fail(context.Context, mem.Lease, string, time.Duration) error
	Complete(context.Context, mem.Lease) error
	EnqueueReflection(context.Context, Task) error
	WithCommit(context.Context, Commit) context.Context
}
type metaKey struct{}
type call struct {
	Task     Task
	Snapshot Snapshot
}

func withInput(ctx context.Context, t Task, s Snapshot) context.Context {
	return context.WithValue(ctx, metaKey{}, call{t, s})
}
func validTrigger(t string) bool {
	switch t {
	case "session_closed", "needs_formulation", "thread_growth", "contradictions", "nightly":
		return true
	}
	return false
}
func (a *Assets) Eligible(t Task, s Snapshot) bool {
	if s.User.UserID != t.UserID || s.Thread.UserID != t.UserID || s.Thread.ID != t.ThreadID || s.Session.UserID != t.UserID || s.Session.ID != t.SessionID || !validTrigger(t.Trigger) {
		return false
	}
	p := a.thresholds.Reflection
	switch t.Trigger {
	case "session_closed":
		return p.Triggers.SessionClosed && s.Session.ClosedAt != nil
	case "needs_formulation":
		return sufficient(a, s)
	case "thread_growth":
		return s.NewItems >= p.Triggers.MinNewItems || s.ThreadTurns >= p.Triggers.ThreadTurnsGTE
	case "contradictions":
		return s.Contradictions >= p.Triggers.ContradictionEventsGTE
	case "nightly":
		return true
	}
	return false
}
func sufficient(a *Assets, s Snapshot) bool {
	sessions := map[c.ID]bool{}
	situations := map[c.ID]bool{}
	// Count concrete reported situations/episodes, never assistant/model text.
	for _, it := range s.Items {
		if it.Kind != c.MemEpisode || it.Epistemic != c.EpReported {
			continue
		}
		for _, ev := range it.Evidence {
			for _, src := range s.Sources {
				if ev.Kind == "message" && ev.Role == "supports" && ev.Ref == src.Message.ID && src.Message.Role == c.RoleUser && c.ValidID(src.SessionID) {
					sessions[src.SessionID] = true
					situations[ev.Ref] = true
				}
			}
		}
	}
	if slot, ok := s.Thread.Slots["situation"]; ok && slot.Status == c.EpReported {
		for _, id := range slot.Evidence {
			for _, src := range s.Sources {
				if src.Message.ID == id && src.Message.Role == c.RoleUser && c.ValidID(src.SessionID) {
					sessions[src.SessionID] = true
					situations[id] = true
				}
			}
		}
	}
	return len(sessions) >= a.thresholds.Reflection.MinEvidenceForFormulation.Sessions && len(situations) >= a.thresholds.Reflection.MinEvidenceForFormulation.Situations
}
func permitted(ctx context.Context, p mem.Policy, s Snapshot, a mem.Action, kind string, id c.ID) bool {
	ok, e := p.Allow(ctx, mem.Permission{Action: a, User: s.User, SourceKind: kind, SourceID: id, Origin: c.ProvenanceSrc{Origin: "reflection"}})
	return e == nil && ok
}
func authorize(ctx context.Context, p mem.Policy, t Task, s Snapshot) bool {
	if p == nil || !s.User.Risk.Level.Valid() || s.User.Risk.Level.Rank() >= c.RiskElevated.Rank() || !permitted(ctx, p, s, mem.Reflect, "thread", t.ThreadID) {
		return false
	}
	return true
}

var _ ports.Reflector = (*Model)(nil)
