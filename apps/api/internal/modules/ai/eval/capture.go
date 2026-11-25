package eval

import (
	"context"
	"sync"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// Capture holds the ACTUAL typed results from a synthetic integrated run. The
// wrappers below are opt-in at the test composition root; no Orchestrator,
// protocol, gateway, persistence, or shared contract changes are necessary.
// The capture is ephemeral and never writes draft text to a trace or log.
type Capture struct {
	mu    sync.Mutex
	turns map[c.ID]Observation
}

func NewCapture() *Capture { return &Capture{turns: map[c.ID]Observation{}} }
func (s *Capture) begin(user, turn c.ID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.turns[turn] = Observation{UserID: user.String(), TurnID: turn.String()}
}
func (s *Capture) record(turn c.ID, f func(*Observation)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.turns[turn]
	if !ok {
		return
	}
	f(&o)
	s.turns[turn] = o
}
func (s *Capture) Snapshot(turn c.ID) (Observation, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.turns[turn]
	return o, ok
}

// SetSafety is for an explicitly synthetic, separately instrumented WP-06
// gate. If omitted, the safety assertion correctly blocks for missing evidence.
func (s *Capture) SetSafety(turn c.ID, signal c.SafetySignal) {
	s.record(turn, func(o *Observation) { copy := signal; o.Safety = &copy })
}

// CaptureGateway acquires owner identity exclusively from the trusted WP-02
// admission, never from fixture-supplied query/body fields.
type CaptureGateway struct {
	Next    ports.Gateway
	Capture *Capture
}

var _ ports.Gateway = (*CaptureGateway)(nil)

func (w *CaptureGateway) Admit(ctx context.Context, req c.TurnRequest) (c.Admission, error) {
	a, err := w.Next.Admit(ctx, req)
	if err == nil && a.UserID == req.UserID && c.ValidID(a.TurnID) && c.ValidID(a.LeaseToken) && !a.Queued && w.Capture != nil {
		w.Capture.begin(a.UserID, a.TurnID)
	}
	return a, err
}
func (w *CaptureGateway) Release(ctx context.Context, a c.Admission, outcome ports.TurnOutcome) error {
	return w.Next.Release(ctx, a, outcome)
}

// CaptureValidator never upgrades a failed/erroring Finalize into a pass.
type CaptureValidator struct {
	Next    ports.Validator
	Capture *Capture
}

var _ ports.Validator = (*CaptureValidator)(nil)

func (w *CaptureValidator) StreamGuard(plan c.ResponsePlan, risk c.RiskLevel) ports.StreamGuard {
	return w.Next.StreamGuard(plan, risk)
}
func (w *CaptureValidator) Finalize(ctx context.Context, text string, plan c.ResponsePlan, wc c.WorkingContext, risk c.RiskLevel, crisis *c.CrisisPlan) (c.ValidationReport, error) {
	out, err := w.Next.Finalize(ctx, text, plan, wc, risk, crisis)
	if w.Capture != nil {
		w.Capture.record(wc.Turn.TurnID, func(o *Observation) {
			copy := out
			if err != nil {
				copy.Passed = false
			}
			o.Validator = &copy
		})
	}
	return out, err
}
func (w *CaptureValidator) Reference() c.BundleRef {
	if ref, ok := w.Next.(interface{ Reference() c.BundleRef }); ok {
		return ref.Reference()
	}
	return c.BundleRef{}
}

// CaptureFinalizer captures ONLY successful, transactional prepared output. It
// never changes the finalizer result and never parses sensitive async payloads.
type CaptureFinalizer struct {
	Next    ports.TurnFinalizer
	Capture *Capture
}

var _ ports.TurnFinalizer = (*CaptureFinalizer)(nil)

func (w *CaptureFinalizer) Complete(ctx context.Context, a c.Admission, p c.PreparedTurn) (c.TurnResult, error) {
	out, err := w.Next.Complete(ctx, a, p)
	if err == nil && w.Capture != nil {
		w.Capture.record(a.TurnID, func(o *Observation) {
			o.Route = string(p.Route)
			degraded := p.Degraded
			o.Degraded = &degraded
			o.State = &StateObserved{Session: &SessionObserved{LastObjective: string(p.State.Session.LastObjective), PendingQuestionSet: p.State.Session.PendingQuestion != nil}}
			if p.State.Thread != nil {
				o.State.Thread = &ThreadObserved{}
				for _, slot := range p.State.Thread.Slots {
					o.State.Thread.SlotsUpdated = append(o.State.Thread.SlotsUpdated, slot.Slot)
				}
				if p.State.Thread.Phase != nil {
					o.State.Thread.Phase = string(*p.State.Thread.Phase)
				}
			}
			if p.State.User.Risk != nil {
				o.State.Risk = &RiskObserved{Level: string(p.State.User.Risk.To)}
			}
		})
	}
	return out, err
}
func (w *CaptureFinalizer) Fail(ctx context.Context, a c.Admission, code c.Code) error {
	return w.Next.Fail(ctx, a, code)
}

// VerifiedOwner is a read-only, explicit ownership check for WP-09 results.
// Without it, retrieved items are intentionally untrusted and the Runner
// refuses to score them as owner-isolated evidence.
type VerifiedOwner func(context.Context, c.ID, []c.ID) error
type CaptureRetriever struct {
	Next        ports.Retriever
	Capture     *Capture
	VerifyOwner VerifiedOwner
}

var _ ports.Retriever = (*CaptureRetriever)(nil)

func (w *CaptureRetriever) Retrieve(ctx context.Context, req c.RetrievalRequest) (c.RetrievalResult, error) {
	out, err := w.Next.Retrieve(ctx, req)
	if err != nil || w.Capture == nil {
		return out, err
	}
	ids := make([]c.ID, 0, len(out.Items))
	for _, item := range out.Items {
		ids = append(ids, item.ID)
	}
	ownerVerified := w.VerifyOwner != nil && w.VerifyOwner(ctx, req.UserID, ids) == nil
	w.Capture.record(req.Turn.TurnID, func(o *Observation) {
		o.RetrievalStatusesObserved = ownerVerified && (!o.RetrievalObserved || o.RetrievalStatusesObserved)
		o.RetrievalObserved = true
		for _, item := range out.Items {
			owned := ""
			if ownerVerified {
				owned = req.UserID.String()
			}
			o.Retrieval = append(o.Retrieval, Retrieved{ID: item.ID.String(), UserID: owned, Status: string(item.Status)})
		}
	})
	return out, nil
}

// CaptureExecutor calls a real injected ports.Orchestrator against a fresh
// synthetic test store assembled by the caller; no DB credentials or prod
// sessions are read by this package.
type CaptureExecutor struct {
	Orchestrator ports.Orchestrator
	Capture      *Capture
	NewRequest   func(*Environment, SequenceTurn) (c.TurnRequest, error)
}

func (w CaptureExecutor) Execute(ctx context.Context, env *Environment, step SequenceTurn) (Observation, error) {
	if w.Orchestrator == nil || w.Capture == nil || w.NewRequest == nil {
		return Observation{}, ErrExecution
	}
	req, err := w.NewRequest(env, step)
	if err != nil || req.UserID.String() != env.UserID {
		return Observation{}, ErrExecution
	}
	result, err := w.Orchestrator.ProcessTurn(ctx, req)
	if err != nil {
		return Observation{}, ErrExecution
	}
	got, ok := w.Capture.Snapshot(result.TurnID)
	if !ok || got.UserID != env.UserID {
		return Observation{}, ErrExecution
	}
	return got, nil
}

// CaptureUnified retains schema-decoded synthetic evidence. It does not fabricate
// safety/validator observations or label mocked prose clinically safe.
type CaptureUnified struct {
	Next    ports.UnifiedModel
	Capture *Capture
}

func (w *CaptureUnified) Reference() c.BundleRef { return w.Next.Reference() }
func (w *CaptureUnified) Infer(ctx context.Context, in ports.UnifiedInput) (ports.UnifiedOutput, error) {
	out, err := w.Next.Infer(ctx, in)
	if err == nil && w.Capture != nil {
		w.Capture.record(in.Context.Turn.TurnID, func(o *Observation) { value := out.Turn; o.Unified = &value })
	}
	return out, err
}
