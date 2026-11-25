package orchestrator

import (
	"encoding/json"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	rf "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/reflection"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

type ReflectionPlanner struct {
	Bundle c.BundleRef
	Cipher mem.Cipher
}

func (p ReflectionPlanner) Reference() c.BundleRef { return p.Bundle }
func (p ReflectionPlanner) PlanTurnJob(a c.Admission, u c.UnifiedTurn, d c.StateDelta) *c.AsyncJobProposal {
	// Periodic coverage supplements an omitted model signal; the independent
	// worker still enforces longitudinal eligibility, quotas and ownership.
	thread := d.Session.FocusThread
	if thread == nil && d.Thread != nil {
		thread = &d.Thread.ThreadID
	}
	if thread == nil || !u.Reflection {
		return nil
	}
	task := rf.Task{Schema: "reflection_job.v1", Bundle: p.Bundle, UserID: a.UserID, TurnID: a.TurnID, ThreadID: *thread, SessionID: a.SessionID, Trigger: "needs_formulation"}
	raw, _ := json.Marshal(task)
	return &c.AsyncJobProposal{Kind: "reflection", DedupeKey: a.TurnID.String() + ":needs_formulation", Payload: raw}
}
