package state

import (
	"context"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// ValidateReflection applies the existing ST-002/003/005/006 thread gates to
// asynchronous proposals without fabricating a user turn or incrementing it.
// Merge/split lack target/link specifications in reflection.v1; retain them for
// review instead of guessing a destructive merge or creating invented threads.
func ValidateReflection(ctx context.Context, snap contextstate.Snapshot, pol contextstate.Policy, proposals []c.ThreadProposal) (c.StateDelta, []c.DeltaRejection, error) {
	d := c.StateDelta{UserID: snap.User.UserID}
	rejected := []c.DeltaRejection{}
	reject := func(path, rule, detail string) {
		rejected = append(rejected, c.DeltaRejection{Path: path, RuleCode: rule, Detail: detail})
	}
	for _, p := range proposals {
		if p.Op != "resolve" && p.Op != "dormant" {
			reject("thread_proposals", "RF-005", "proposal needs explicit targets")
			continue
		}
		if p.Phase != nil && (*p.Phase != c.PhaseResolved && p.Op == "resolve" || *p.Phase != c.PhaseDormant && p.Op == "dormant") {
			reject("thread_proposals", "ST-002", "operation/phase mismatch")
			continue
		}
		d.ThreadOps = append(d.ThreadOps, c.ThreadOp{ThreadID: p.ThreadID, Op: p.Op, Reason: "reflection"})
	}
	s := &Service{}
	e := s.validateThread(ctx, snap, pol, &d, reject)
	return d, rejected, e
}
