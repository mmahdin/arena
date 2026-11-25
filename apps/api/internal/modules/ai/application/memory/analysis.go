package memory

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

func (s *Service) PlanAnalysisJob(ctx context.Context, a c.Admission, request c.BackgroundRequest) (*c.AsyncJobProposal, error) {
	job, err := s.PlanTurnJob(ctx, a, c.UnifiedTurn{Conversation: c.ConversationIntent{Intent: c.IntentDiscloseEvent, Objective: c.ObjReflect}})
	if err != nil || job == nil {
		return job, err
	}
	var task Task
	if json.Unmarshal(job.Payload, &task) != nil {
		return nil, ErrDenied
	}
	plain, err := s.opt.Cipher.Open(ctx, a.UserID, "memory-task:"+a.TurnID.String(), task.Data)
	if err != nil {
		return nil, err
	}
	var data taskData
	if json.Unmarshal(plain, &data) != nil {
		return nil, ErrDenied
	}
	data.Request = &request
	plain, _ = json.Marshal(data)
	task.OperationID = uuid.NewSHA1(a.TurnID, []byte("analysis:"+request.ID))
	task.Data, err = s.opt.Cipher.Seal(ctx, a.UserID, "memory-task:"+a.TurnID.String(), plain)
	if err != nil {
		return nil, err
	}
	job.Payload, _ = json.Marshal(task)
	job.DedupeKey = task.OperationID.String()
	return job, nil
}
