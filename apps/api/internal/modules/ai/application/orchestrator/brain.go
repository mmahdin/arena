package orchestrator

import (
	"context"
	"errors"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/brain"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
)

// Opt-in capability of the real WP-10 module. WP-05 fixtures remain isolated;
// runtime use requires NewIntegrated's WP-06/07/08 and pinned bundle checks.
type turnBrain interface {
	ports.Brain
	WP10()
	Reference() c.BundleRef
	MaxHops() int
	MaxRetrieved() int
	RetrievalTimeout() time.Duration
	BrainTimeout() time.Duration
	Fallback(c.WorkingContext) c.BrainDecision
}

func (o *Service) decideWP10(ctx context.Context, a c.Admission, wc c.WorkingContext, sig c.SafetySignal, tr ports.TurnTrace) (ports.BrainOutput, c.WorkingContext, bool, error) {
	b := o.deps.Brain.(turnBrain)
	if o.integration == nil {
		return ports.BrainOutput{}, wc, false, coreError(c.ErrValidation, "brain_integration", nil)
	}
	ctx = brain.WithCallMeta(ctx, c.CallMeta{UserID: a.UserID, TurnID: a.TurnID, Bundle: o.bundleReference(ctx), Route: c.RouteNormal})
	// One stage deadline covers BOTH reasoning invocations and their hop. It
	// cannot double the configured Brain budget or outlive the turn deadline.
	timeout := b.BrainTimeout()
	if o.opts.BrainTimeout < timeout {
		timeout = o.opts.BrainTimeout
	}
	if budget, ok := c.BudgetFromContext(ctx); ok && budget.BrainTimeout > 0 && budget.BrainTimeout < timeout {
		timeout = budget.BrainTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	call := func(w c.WorkingContext, hop int) (ports.BrainOutput, error) {
		started := time.Now()
		out, e := b.Decide(ctx, ports.BrainInput{WCtx: w, Safety: sig, Bundle: o.bundleReference(ctx), Hop: hop})
		if tr != nil {
			tr.AddUsage(out.Usage)
			for _, r := range out.Repairs {
				tr.Record("policy_gate", trace.Observation{RuleID: r.RuleCode, Count: 1, Degraded: true})
			}
		}
		record(tr, "brain", started, e != nil)
		return out, e
	}
	out, err := call(wc, 0)
	if err != nil {
		var ce *c.Error
		if errors.As(err, &ce) && (ce.Code == c.ErrForbidden || ce.Code == c.ErrBundle || ce.Code == c.ErrStateConflict) {
			return out, wc, false, err
		}
		if errors.Is(err, context.Canceled) {
			return out, wc, false, err
		}
		out.Decision = b.Fallback(wc)
		out.Repairs = append(out.Repairs, c.DeltaRejection{RuleCode: "FB-001"})
		return out, wc, true, nil
	}
	degraded := len(out.Repairs) > 0
	need := out.Decision.Needs.Context
	// Safety escalation is handled BEFORE retrieval, including after the first
	// Brain. No extra source is read during crisis routing.
	raised := out.Decision.RiskRead != nil && out.Decision.RiskRead.Level.Rank() >= c.RiskElevated.Rank()
	if need == nil || raised {
		return out, wc, degraded, nil
	}
	if b.MaxHops() == 0 || o.deps.Retriever == nil || wc.RetrievalMeta.Hops != 0 || out.Usage.Attempts > 1 || len(wc.Retrieved) >= b.MaxRetrieved() {
		out.Decision = b.Fallback(wc)
		out.Repairs = append(out.Repairs, c.DeltaRejection{RuleCode: "BR-019"})
		return out, wc, true, nil
	}
	exclude := []c.ID{wc.Turn.MessageID}
	for _, m := range wc.Recent {
		exclude = append(exclude, m.ID)
	}
	for _, m := range wc.Retrieved {
		exclude = append(exclude, m.ID)
	}
	var focus *c.ID
	if wc.FocusThread != nil {
		id := wc.FocusThread.ID
		focus = &id
	}
	// Keep the original turn/date semantics. A request for background is never
	// permission to replace an exact miss with a semantic substitute.
	strategies := []c.RetrievalStrategy(nil)
	if wc.RetrievalMeta.ExactMatchFound != nil {
		strategies = []c.RetrievalStrategy{c.StratTemporal}
	}
	hopCtx, stop := context.WithTimeout(ctx, b.RetrievalTimeout())
	ret, e := o.deps.Retriever.Retrieve(hopCtx, c.RetrievalRequest{UserID: a.UserID, Turn: wc.Turn, FocusThread: focus, Extra: need, Exclude: exclude, MaxItems: 1, Strategies: strategies, IncludeSensitivity: []c.Sensitivity{c.SensNormal, c.SensSensitive}})
	timedOut := hopCtx.Err() != nil
	stop()
	wc.RetrievalMeta.Hops = 1
	if e != nil || ret.Meta.Degraded || timedOut {
		wc.Retrieved = nil
		wc.RetrievalMeta.Degraded = true
		wc.RetrievalMeta.ExactMatchFound = nil
		wc.Eligibility.MemoryAvailable = false
		degraded = true
	} else {
		seen := map[c.ID]bool{}
		for _, id := range exclude {
			seen[id] = true
		}
		for _, item := range ret.Items {
			if len(wc.Retrieved) >= b.MaxRetrieved() {
				break
			}
			if !seen[item.ID] && c.ValidID(item.ID) && item.Sensitivity != c.SensHighly && item.EvidenceN > 0 {
				wc.Retrieved = append(wc.Retrieved, item)
				break
			}
		}
		wc.RetrievalMeta.Strategies = append(wc.RetrievalMeta.Strategies, ret.Meta.Strategies...)
		wc.RetrievalMeta.AmbiguousReference = ret.Meta.AmbiguousReference
		// An earlier exact failure stays a failure. WP-09 owns date matching.
		if wc.RetrievalMeta.ExactMatchFound == nil {
			wc.RetrievalMeta.ExactMatchFound = ret.Meta.ExactMatchFound
		}
	}
	if tr != nil {
		tr.Record("retrieval_hop", trace.Observation{Count: 1, Degraded: degraded})
	}
	// Always re-evaluate after a hop, including a failed hop, so the model knows
	// memory is unavailable. Never run an unvalidated first plan with new data.
	second, e := call(wc, 1)
	if e != nil {
		var ce *c.Error
		if errors.As(e, &ce) && (ce.Code == c.ErrForbidden || ce.Code == c.ErrBundle || ce.Code == c.ErrStateConflict) {
			return second, wc, true, e
		}
		if errors.Is(e, context.Canceled) {
			return second, wc, true, e
		}
		second.Decision = b.Fallback(wc)
		second.Repairs = append(second.Repairs, c.DeltaRejection{RuleCode: "FB-001"})
		degraded = true
	}
	// If the first pass detected a higher risk, a second pass cannot erase it.
	if out.Decision.RiskRead != nil && (second.Decision.RiskRead == nil || second.Decision.RiskRead.Level.Rank() < out.Decision.RiskRead.Level.Rank()) {
		second.Decision.RiskRead = out.Decision.RiskRead
	}
	second.Decision.Needs.Context = nil
	return second, wc, degraded || len(second.Repairs) > 0, nil
}

func permittedContext(w c.WorkingContext, plan c.ResponsePlan) []c.MemoryView {
	out := []c.MemoryView{}
	seen := map[c.ID]bool{}
	for _, id := range plan.PermittedContext {
		if seen[id] {
			continue
		}
		seen[id] = true
		for _, m := range w.Retrieved {
			if m.ID == id && !w.RetrievalMeta.Degraded {
				out = append(out, m)
			}
		}
		for _, m := range w.Recent {
			if m.ID == id {
				out = append(out, c.MemoryView{ID: id, Text: m.Text, Epistemic: messageEpistemic(m.Role), Verbatim: true, Age: m.Age})
			}
		}
		if w.Turn.MessageID == id {
			out = append(out, c.MemoryView{ID: id, Text: w.Turn.Text, Epistemic: c.EpReported, Verbatim: true})
		}
	}
	return out
}

func messageEpistemic(role c.Role) c.Epistemic {
	if role == c.RoleUser {
		return c.EpReported
	}
	return c.EpSystemObserved
}
