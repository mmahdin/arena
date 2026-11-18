package orchestrator

import (
	"context"

	"github.com/google/uuid"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// TurnMemory is the WP-12 critical-path handoff only. Memory extraction and
// reconciliation run after the accepted turn is atomically finalized.
type TurnMemory interface {
	Correct(context.Context, c.ID, c.ID, ports.CorrectionKind) error
	PlanJob(context.Context, c.Admission, c.BrainDecision, []c.UserClaim) (*c.AsyncJobProposal, error)
	Reference() c.BundleRef
}

func (o *Service) memoryJobContext(ctx context.Context, a c.Admission, d c.StateDelta, w c.WorkingContext, extracted c.Extracted) context.Context {
	thread := d.Session.FocusThread
	if thread == nil && w.FocusThread != nil {
		id := w.FocusThread.ID
		thread = &id
	}
	feedback := mem.Feedback{Referenced: extracted.ReferencedContext}
	for _, item := range w.Retrieved {
		feedback.Retrieved = append(feedback.Retrieved, item.ID)
	}
	var signals []c.HypothesisSignal
	if d.Thread != nil {
		signals = d.Thread.HypothesisSignals
	}
	return mem.WithMeta(ctx, mem.Meta{UserID: a.UserID, TurnID: a.TurnID, OperationID: a.TurnID, Bundle: o.bundleReference(ctx), ThreadID: thread, Feedback: feedback, Signals: signals})
}

func (o *Service) correctMemory(ctx context.Context, a c.Admission, target c.ID) error {
	if o.deps.Memory == nil {
		return coreError(c.ErrValidation, "brain_dependency", nil)
	}
	operation := uuid.NewSHA1(a.TurnID, []byte("memory-correction:"+target.String()))
	ctx = mem.WithMeta(ctx, mem.Meta{UserID: a.UserID, TurnID: a.TurnID, OperationID: operation, Bundle: o.bundleReference(ctx)})
	if e := o.deps.Memory.Correct(ctx, a.UserID, target, ports.CorrectionContest); e != nil {
		return coreError(c.ErrInternal, "memory_correction", nil)
	}
	return nil
}
func withoutCorrectedMemory(w c.WorkingContext, p c.ResponsePlan, target c.ID) (c.WorkingContext, c.ResponsePlan) {
	memories := w.Retrieved[:0]
	for _, v := range w.Retrieved {
		if v.ID != target {
			memories = append(memories, v)
		}
	}
	w.Retrieved = memories
	ids := p.PermittedContext[:0]
	for _, id := range p.PermittedContext {
		if id != target {
			ids = append(ids, id)
		}
	}
	p.PermittedContext = ids
	// A psychological formulation can contain the disputed statement without a
	// direct source handle. Rebuild it on a later turn after fresh state reads.
	w.Formulation = nil
	if w.FocusThread != nil {
		copy := *w.FocusThread
		copy.Hypotheses = nil
		for _, h := range w.FocusThread.Hypotheses {
			if h.ID != target {
				copy.Hypotheses = append(copy.Hypotheses, h)
			}
		}
		w.FocusThread = &copy
	}
	return w, p
}
