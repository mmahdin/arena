package memory

import (
	"context"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// ReviewRecent supplements per-turn scheduling without any Unified request.
func (s *Service) ReviewRecent(ctx context.Context) error {
	pending, ok := s.repo.(interface {
		PendingMemoryReviews(context.Context, int) ([]c.Admission, error)
	})
	if !ok {
		return nil
	}
	queue, ok := s.repo.(Queue)
	if !ok {
		return ErrAssets
	}
	turns, e := pending.PendingMemoryReviews(ctx, 8)
	if e != nil {
		return e
	}
	for _, a := range turns {
		job, e := s.PlanTurnJob(ctx, a, c.UnifiedTurn{Conversation: c.ConversationIntent{Intent: c.IntentOther, Objective: c.ObjReflect}})
		if e != nil {
			return e
		}
		if job != nil {
			if e = queue.Enqueue(ctx, a.UserID, a.TurnID, *job); e != nil {
				return e
			}
		}
	}
	return nil
}
