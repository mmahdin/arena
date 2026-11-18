package state

import (
	"context"
	"testing"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

func TestReflectionMergeSplitFailClosedWithoutApprovedTargets(t *testing.T) {
	owner, thread, other := c.NewID(), c.NewID(), c.NewID()
	snap := contextstate.Snapshot{
		User: c.UserState{UserID: owner},
		Threads: []c.ThreadRecord{
			{ID: thread, UserID: owner, Phase: c.PhaseMaintaining, Version: 1},
			{ID: other, UserID: owner, Phase: c.PhaseExploring, Version: 1},
		},
	}
	policy := contextstate.Policy{}
	policy.Thresholds.Thread.MaxActive = 3
	// A reason string, a phase, and even a second candidate thread are not
	// target or partition contracts. Do not infer a destination from them.
	proposals := []c.ThreadProposal{
		{ThreadID: thread, Op: "merge", Reason: "synthetic suggestion mentioning other thread " + other.String()},
		{ThreadID: other, Op: "split", Reason: "synthetic topic labels"},
		{ThreadID: thread, Op: "resolve", Reason: "synthetic resolution"},
	}
	result, rejected, err := ValidateReflection(context.Background(), snap, policy, proposals)
	if err != nil || len(result.ThreadOps) != 1 || result.ThreadOps[0].Op != "resolve" || result.ThreadOps[0].ThreadID != thread {
		t.Fatalf("unsupported operations applied or supported operation lost: %+v %v", result.ThreadOps, err)
	}
	if len(rejected) != 2 || rejected[0].RuleCode != "RF-005" || rejected[1].RuleCode != "RF-005" {
		t.Fatalf("merge/split accepted without precise targets: %+v", rejected)
	}
	if len(result.Events) != 0 || result.Thread != nil {
		t.Fatal("unsupported reflection proposal invented state changes")
	}
}
