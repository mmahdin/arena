package orchestrator

import (
	"context"
	"encoding/json"

	rf "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/reflection"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// ProtocolFacts is an approved owner-scoped contraindication inventory. Missing
// facts are unknown, never an implicit absence of contraindications.
type ProtocolFacts interface {
	Flags(context.Context, c.ID, c.ID) ([]string, bool, error)
}
type TurnReflection interface {
	Reference() c.BundleRef
	PlanJob(c.Admission, c.BrainDecision, c.StateDelta) *c.AsyncJobProposal
}
type ReflectionPlanner struct{ Bundle c.BundleRef }

func (p ReflectionPlanner) Reference() c.BundleRef { return p.Bundle }
func (p ReflectionPlanner) PlanJob(a c.Admission, b c.BrainDecision, d c.StateDelta) *c.AsyncJobProposal {
	if !b.Needs.Formulation {
		return nil
	}
	thread := d.Session.FocusThread
	if thread == nil && d.Thread != nil {
		thread = &d.Thread.ThreadID
	}
	if thread == nil {
		return nil
	}
	task := rf.Task{Schema: "reflection_job.v1", Bundle: p.Bundle, UserID: a.UserID, TurnID: a.TurnID, ThreadID: *thread, SessionID: a.SessionID, Trigger: "needs_formulation"}
	payload, _ := json.Marshal(task)
	return &c.AsyncJobProposal{Kind: "reflection", DedupeKey: a.TurnID.String() + ":needs_formulation", Payload: payload}
}
