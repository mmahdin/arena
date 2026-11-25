package state

import (
	"context"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"testing"
)

func TestReflectionUsesExistingThreadFSMAndProtectsFocus(t *testing.T) {
	user, id := c.NewID(), c.NewID()
	snap := contextstate.Snapshot{User: c.UserState{UserID: user}, Threads: []c.ThreadRecord{{ID: id, UserID: user, Phase: c.PhaseExploring, Version: 1}}}
	pol := contextstate.Policy{}
	pol.Thresholds.Thread.MaxActive = 3
	proposal := []c.ThreadProposal{{ThreadID: id, Op: "resolve", Reason: "synthetic"}}
	d, r, e := ValidateReflection(context.Background(), snap, pol, proposal)
	if e != nil || len(d.ThreadOps) != 0 || len(r) == 0 {
		t.Fatal("skipped FSM")
	}
	snap.Threads[0].Phase = c.PhaseMaintaining
	d, r, e = ValidateReflection(context.Background(), snap, pol, proposal)
	if e != nil || len(d.ThreadOps) != 1 || len(r) != 0 {
		t.Fatal("valid proposal rejected")
	}
	snap.Session.FocusThreadID = &id
	d, _, _ = ValidateReflection(context.Background(), snap, pol, proposal)
	if len(d.ThreadOps) != 0 {
		t.Fatal("focus orphaned")
	}
	d, r, _ = ValidateReflection(context.Background(), snap, pol, []c.ThreadProposal{{ThreadID: id, Op: "merge"}})
	if len(d.ThreadOps) != 0 || len(r) != 1 || r[0].RuleCode != "RF-005" {
		t.Fatal("guessed merge target")
	}
}
