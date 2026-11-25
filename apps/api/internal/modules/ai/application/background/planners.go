package background

import (
	"context"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

type MemoryPlanner struct{ *Service }

func (p MemoryPlanner) PlanTurnJob(context.Context, c.Admission, c.UnifiedTurn) (*c.AsyncJobProposal, error) {
	return nil, nil
}
func (p MemoryPlanner) PlanAnalysisJob(ctx context.Context, a c.Admission, r c.BackgroundRequest) (*c.AsyncJobProposal, error) {
	return p.Plan(ctx, a, r, Scope{Kind: "user", ThreadIDs: []c.ID{}})
}

type ReflectionPlanner struct{ *Service }

func (p ReflectionPlanner) PlanTurnJob(c.Admission, c.UnifiedTurn, c.StateDelta) *c.AsyncJobProposal {
	return nil
}
func (p ReflectionPlanner) PlanAnalysisJob(ctx context.Context, a c.Admission, r c.BackgroundRequest, d c.StateDelta) (*c.AsyncJobProposal, error) {
	scope := Scope{Kind: "user", ThreadIDs: []c.ID{}}
	if d.Session.FocusThread != nil {
		scope = Scope{Kind: "threads", ThreadIDs: []c.ID{*d.Session.FocusThread}}
	}
	return p.Plan(ctx, a, r, scope)
}

func (p MemoryPlanner) BackgroundOnly() bool { return true }
