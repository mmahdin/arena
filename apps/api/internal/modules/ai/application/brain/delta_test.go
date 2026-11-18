package brain

import (
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"testing"
)

func TestStateDeltaCreationShiftAndResume(t *testing.T) {
	a := testAssets(t)
	in := input()
	user := c.NewID()
	old, target := c.NewID(), c.NewID()
	focus := c.ThreadRecord{ID: old, UserID: user, Version: 4, Phase: c.PhaseExploring}
	snap := ports.StateSnapshot{Focus: &focus, Threads: []c.ThreadRecord{focus, {ID: target, UserID: user, Version: 8, Phase: c.PhasePaused, PrevPhase: c.PhaseWorking}}}
	for _, relation := range []c.ThreadRelation{c.RelNew, c.RelShifts, c.RelReturns, c.RelContinues} {
		t.Run(string(relation), func(t *testing.T) {
			b := decision(a, in)
			b.TurnAnalysis.ThreadRelation = relation
			topic := "Synthetic topic"
			b.TurnAnalysis.NewThreadTopic = &topic
			if relation == c.RelReturns || relation == c.RelShifts {
				b.TurnAnalysis.TargetThreadID = &target
			}
			b.Understanding.Slots = []c.SlotUpdate{{Slot: "situation", Status: c.EpReported, Value: "Synthetic event", Evidence: []c.ID{in.WCtx.Turn.MessageID}}}
			d := c.StateDelta{UserID: user, TurnID: in.WCtx.Turn.TurnID, ExpectedVersions: c.Versions{Thread: 4}}
			ProposeDelta(&d, b, in.WCtx, snap)
			if d.Thread == nil || len(d.Thread.Slots) != 1 {
				t.Fatal(d)
			}
			switch relation {
			case c.RelNew:
				if d.Thread.Create == nil || d.ExpectedVersions.Thread != 0 || d.Session.FocusThread == nil || len(d.ThreadOps) != 1 {
					t.Fatal(d)
				}
			case c.RelReturns, c.RelShifts:
				if d.Thread.ThreadID != target || d.ExpectedVersions.Thread != 8 || d.Thread.Phase == nil || *d.Thread.Phase != c.PhaseWorking || len(d.ThreadOps) != 1 {
					t.Fatal(d)
				}
			default:
				if d.Thread.ThreadID != old || len(d.ThreadOps) != 0 {
					t.Fatal(d)
				}
			}
			if d.Protocol != nil || d.User.Risk != nil || len(d.FollowUps) != 0 {
				t.Fatal("invented later-module state")
			}
		})
	}
}
func TestNoModelQuestionOrConsentBecomesState(t *testing.T) {
	a := testAssets(t)
	in := input()
	b := decision(a, in)
	b.Plan.Question.Allowed = true
	b.Plan.Protocol = &c.ProtocolDirective{Action: c.PAStart}
	d := c.StateDelta{UserID: c.NewID(), TurnID: in.WCtx.Turn.TurnID}
	ProposeDelta(&d, b, in.WCtx, ports.StateSnapshot{})
	if d.Session.PendingQuestion != nil || d.Session.ConsecutiveQuestions != 0 || d.Protocol != nil {
		t.Fatal("model created unvalidated question or consent", d)
	}
}
