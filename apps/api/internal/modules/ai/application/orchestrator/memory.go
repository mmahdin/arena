package orchestrator

import (
	"context"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

func (o *Service) memoryJobContext(ctx context.Context, a c.Admission, d c.StateDelta, w c.WorkingContext, extracted c.Extracted) context.Context {
	thread := d.Session.FocusThread
	if thread == nil && w.FocusThread != nil {
		id := w.FocusThread.ID
		thread = &id
	}
	feedback := memoryFeedback(d, w, extracted)
	var signals []c.HypothesisSignal
	if d.Thread != nil {
		signals = d.Thread.HypothesisSignals
	}
	return mem.WithMeta(ctx, mem.Meta{UserID: a.UserID, TurnID: a.TurnID, OperationID: a.TurnID, Bundle: o.bundleReference(ctx), ThreadID: thread, Feedback: feedback, Signals: signals})
}

func memoryFeedback(d c.StateDelta, w c.WorkingContext, extracted c.Extracted) mem.Feedback {
	feedback := mem.Feedback{Referenced: extracted.ReferencedContext}
	if d.SchemaVersion == "state_delta.v2" {
		// Feedback updates only retrieved memory rows. Unified references also
		// include messages, state and persistent facts, which must not decay.
		feedback = mem.Feedback{}
		referenced := make(map[c.ID]bool, len(extracted.ReferencedContext))
		for _, id := range extracted.ReferencedContext {
			referenced[id] = true
		}
		seen := map[c.ID]bool{}
		for _, item := range w.Retrieved {
			if item.Source != "memory" || seen[item.ID] || len(feedback.Retrieved) >= 7 {
				continue
			}
			seen[item.ID] = true
			feedback.Retrieved = append(feedback.Retrieved, item.ID)
			if referenced[item.ID] {
				feedback.Referenced = append(feedback.Referenced, item.ID)
			}
		}

	}
	return feedback
}
