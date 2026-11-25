package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"

	rf "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/reflection"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
)

func (o *Service) planBackgroundRequests(ctx context.Context, a c.Admission, u c.UnifiedTurn, w c.WorkingContext, d c.StateDelta, stateIDs []string) (jobsResult []c.AsyncJobProposal, acceptedResult []string, rejectedResult []c.DeltaRejection) {
	ctx, span := trace.Start(ctx, "orchestrator.planBackgroundRequests", func() any {
		return map[string]any{"admission": a, "turn": u, "context": w, "state": d, "accepted_state_ids": stateIDs}
	})
	defer func() {
		span.Complete(func() any {
			return map[string]any{"jobs": jobsResult, "accepted_ids": acceptedResult, "rejected": rejectedResult}
		}, nil, recover())
	}()
	jobs := []c.AsyncJobProposal{}
	accepted := []string{}
	rejected := []c.DeltaRejection{}
	ids := map[string]bool{}
	for _, id := range stateIDs {
		ids[id] = true
	}
	evidence := map[c.ID]bool{a.UserMessageID: true}
	for _, m := range w.Recent {
		if m.Role == c.RoleUser {
			evidence[m.ID] = true
		}
	}
	for _, m := range w.Retrieved {
		if m.Source == "message" && m.Epistemic == c.EpReported {
			evidence[m.ID] = true
		}
	}
	for i, request := range u.BackgroundRequests {
		valid := i < 2 && len(request.Queries) > 0 && len(request.Queries) <= 3 && len(request.Objective) > 0 && len(request.Objective) <= 300 && len(request.Topic) > 0 && len(request.Topic) <= 160
		for _, query := range request.Queries {
			if len(query) == 0 || len(query) > 1200 {
				valid = false
			}
		}
		for _, id := range request.DependsOn {
			if !ids[id] {
				valid = false
			}
		}
		for _, id := range request.EvidenceIDs {
			if !evidence[id] {
				valid = false
			}
		}
		var job *c.AsyncJobProposal
		var err error
		if valid {
			switch request.Kind {
			case "memory_intelligence":
				if planner, ok := o.deps.UnifiedMemory.(interface {
					PlanAnalysisJob(context.Context, c.Admission, c.BackgroundRequest) (*c.AsyncJobProposal, error)
				}); ok {
					job, err = planner.PlanAnalysisJob(ctx, a, request)
				}
			case "reflection":
				if planner, ok := o.deps.UnifiedReflection.(interface {
					PlanAnalysisJob(context.Context, c.Admission, c.BackgroundRequest, c.StateDelta) (*c.AsyncJobProposal, error)
				}); ok {
					reflectionDelta := d
					if reflectionDelta.Session.FocusThread == nil && !reflectionDelta.Session.ClearFocus && w.FocusThread != nil {
						id := w.FocusThread.ID
						reflectionDelta.Session.FocusThread = &id
					}
					job, err = planner.PlanAnalysisJob(ctx, a, request, reflectionDelta)
				}
			}
		}
		if !valid || err != nil || job == nil {
			rejected = append(rejected, c.DeltaRejection{Path: fmt.Sprintf("background_requests.%d", i), RuleCode: "BG-001", Detail: "invalid, dependent or unavailable background request"})
			continue
		}
		jobs = append(jobs, *job)
		accepted = append(accepted, request.ID)
	}
	return jobs, accepted, rejected
}

func (p ReflectionPlanner) PlanAnalysisJob(ctx context.Context, a c.Admission, request c.BackgroundRequest, d c.StateDelta) (*c.AsyncJobProposal, error) {
	if p.Cipher == nil {
		return nil, nil
	}
	thread := d.Session.FocusThread
	if thread == nil && d.Thread != nil {
		thread = &d.Thread.ThreadID
	}
	if thread == nil {
		return nil, nil
	}
	raw, _ := json.Marshal(request)
	encrypted, err := p.Cipher.Seal(ctx, a.UserID, "reflection-request:"+a.TurnID.String(), raw)
	if err != nil {
		return nil, err
	}
	task := rf.Task{Schema: "reflection_job.v1", Bundle: p.Bundle, UserID: a.UserID, TurnID: a.TurnID, ThreadID: *thread, SessionID: a.SessionID, Trigger: "needs_formulation", BackgroundData: encrypted}
	payload, _ := json.Marshal(task)
	return &c.AsyncJobProposal{Kind: "reflection", DedupeKey: a.TurnID.String() + ":analysis:" + request.ID, Payload: payload}, nil
}
