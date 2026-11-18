package brain

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// Facts supplies read-only policy evidence unavailable in WorkingContext.
// Implementations MUST scope by authenticated owner. Missing facts fail closed:
// no new hypotheses, offers or hypothesis voicing. No later module is invoked.
type Facts interface {
	HypothesisAllowed(ctx context.Context, user, turn c.ID, thread *c.ID, tag string) (bool, error)
	OfferAllowed(ctx context.Context, user, turn c.ID, minTurnGap, maxPerSession int) (bool, error)
	VoiceAllowed(ctx context.Context, user, turn, hypothesis c.ID, minGap time.Duration, maxPerSession int) (bool, error)
}
type Metrics interface {
	Observe(elapsed time.Duration, usage c.LLMUsage, repairs []c.DeltaRejection)
}
type ContextVerifier interface {
	VerifyContext(context.Context, c.ID, c.ID, c.WorkingContext) error
}

type Options struct {
	Verifier ContextVerifier
	Facts    Facts
	Metrics  Metrics
}
type Service struct {
	client ports.Client
	assets *Assets
	reader contextstate.Reader
	opts   Options
}

var _ ports.Brain = (*Service)(nil)

func New(client ports.Client, assets *Assets, reader contextstate.Reader, opts Options) (*Service, error) {
	if client == nil || assets == nil || reader == nil || !assets.synthetic && opts.Verifier == nil {
		return nil, ErrAssets
	}
	return &Service{client, assets, reader, opts}, nil
}
func (s *Service) Reference() c.BundleRef { return s.assets.Reference() }

// WP10 identifies a schema-validating, gated Brain to the existing orchestrator.
// No protocol execution or response generation is implemented here.
func (*Service) WP10() {}
func (s *Service) MaxHops() int {
	if s.assets.budgets.Limits.MaxBrainHops > 0 {
		return 1
	}
	return 0
}
func (s *Service) MaxRetrieved() int {
	n := s.assets.retrieval.Defaults.HardCapWithHop
	if n > 7 {
		n = 7
	}
	return n
}
func (s *Service) BrainTimeout() time.Duration {
	return time.Duration(s.assets.budgets.Turn.BrainMS) * time.Millisecond
}
func (s *Service) RetrievalTimeout() time.Duration {
	return time.Duration(s.assets.budgets.Turn.RetrievalMS) * time.Millisecond
}
func (s *Service) Fallback(wc c.WorkingContext) c.BrainDecision { return s.assets.fallback(wc) }

type metaKey struct{}

func WithCallMeta(ctx context.Context, meta c.CallMeta) context.Context {
	return context.WithValue(ctx, metaKey{}, meta)
}

func (s *Service) Decide(ctx context.Context, in ports.BrainInput) (out ports.BrainOutput, err error) {
	start := time.Now()
	defer func() {
		if s.opts.Metrics != nil {
			s.opts.Metrics.Observe(time.Since(start), out.Usage, out.Repairs)
		}
	}()
	meta, ok := ctx.Value(metaKey{}).(c.CallMeta)
	if !ok || !c.ValidID(meta.UserID) || meta.TurnID != in.WCtx.Turn.TurnID || !sameBundle(s.assets.ref, in.Bundle) || !sameBundle(meta.Bundle, in.Bundle) || in.Hop < 0 || in.Hop > 1 {
		return out, &c.Error{Code: c.ErrForbidden, Step: "brain"}
	}
	if ctx.Err() != nil {
		return out, ctx.Err()
	}
	snap, e := s.reader.ForTurn(ctx, meta.UserID, meta.TurnID)
	if e != nil || snap.User.UserID != meta.UserID || snap.Session.UserID != meta.UserID || snap.Session.ID != in.WCtx.Session.SessionID || snap.Session.ClosedAt != nil {
		return out, &c.Error{Code: c.ErrForbidden, Step: "brain_context"}
	}
	// Validate the server-assembled state handles before any model/provider call.
	if !ownedContext(in.WCtx, snap) {
		return out, &c.Error{Code: c.ErrForbidden, Step: "brain_context"}
	}
	if s.opts.Verifier != nil {
		if e := s.opts.Verifier.VerifyContext(ctx, meta.UserID, meta.TurnID, in.WCtx); e != nil {
			return out, &c.Error{Code: c.ErrForbidden, Step: "brain_context"}
		}
	}
	var raise *c.BrainRiskRead
	fallback := func(code string) (ports.BrainOutput, error) {
		out.Decision = s.assets.fallback(in.WCtx)
		out.Decision.RiskRead = raise
		out.Repairs = append(out.Repairs, c.DeltaRejection{Path: "brain", RuleCode: code}, c.DeltaRejection{Path: "brain", RuleCode: "FB-001"})
		return out, nil
	}
	system, dynamic, rendered, e := s.assets.render(in)
	if e != nil {
		return fallback("BR-001")
	}
	timeout := time.Duration(s.assets.budgets.Turn.BrainMS) * time.Millisecond
	if b, ok := c.BudgetFromContext(ctx); ok && b.BrainTimeout > 0 && b.BrainTimeout < timeout {
		timeout = b.BrainTimeout
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	resolved, _ := s.assets.schemas.JSON(c.BrainSchemaVersion)
	// Reserve one call for the later Generator. A repair consumes the same
	// two-call allowance as a second reasoning pass, so both cannot occur.
	budget, _ := c.BudgetFromContext(callCtx)
	allowance := s.assets.budgets.Limits.MaxSyncLLMCalls - 1 - in.Hop
	if allowance > 2-in.Hop {
		allowance = 2 - in.Hop
	}
	if budget.MaxLLMCalls > 0 && budget.MaxLLMCalls-1 < allowance {
		allowance = budget.MaxLLMCalls - 1
	}
	if allowance < 1 {
		return fallback("BR-001")
	}
	budget.MaxLLMCalls = allowance
	callCtx = c.WithBudget(callCtx, budget)
	resp, e := s.client.Structured(callCtx, ports.StructuredRequest{Tier: ports.TierSmall, System: system, Messages: []ports.ChatMessage{{Role: c.RoleUser, Content: dynamic}}, Schema: resolved, SchemaName: c.BrainSchemaVersion, MaxTokens: s.assets.budgets.Tokens.BrainOut, Temperature: .2, Timeout: timeout, Meta: meta, IdempotencyKey: meta.TurnID.String() + ".brain." + boolText(in.Hop == 1)})
	out.Usage = resp.Usage
	if ctx.Err() != nil {
		return out, ctx.Err()
	}
	// WP-04 alone owns its single JSON-mode repair and provider failover. Never
	// wrap Structured in another retry loop or re-reason on a timeout.
	if e != nil || callCtx.Err() != nil {
		return fallback("BR-001")
	}
	decision, e := s.assets.schemas.DecodeBrain(resp.JSON)
	if e != nil {
		return fallback("BR-001")
	}
	if decision.RiskRead != nil && decision.RiskRead.Level.Rank() > c.MaxRisk(in.WCtx.Risk.Current, in.Safety.Level).Rank() {
		raise = decision.RiskRead
	}
	out.Raw = nil // encrypted trace capture is opt-in elsewhere; no private raw JSON
	evidence := contextMessages(rendered)
	ids := make([]c.ID, 0, len(evidence))
	for id := range evidence {
		ids = append(ids, id)
	}
	owned, e := s.reader.UserEvidence(ctx, meta.UserID, ids)
	if e != nil {
		return fallback("BR-005")
	}
	// All in-context hypotheses still need owner/status verification at read time.
	hypIDs := hypotheses(rendered)
	list := []c.ID{}
	for id := range hypIDs {
		list = append(list, id)
	}
	validHyp, e := s.reader.ExistingHypotheses(ctx, meta.UserID, list)
	if e != nil {
		return fallback("ST-007")
	}
	facts := policyFacts{owned: owned, hypotheses: validHyp}
	if s.opts.Facts != nil {
		for _, h := range decision.Understanding.Hypotheses {
			if h.Op == c.HypPropose && h.Tag != nil {
				thread := decision.Understanding.ThreadID
				if thread == nil && rendered.FocusThread != nil && decision.TurnAnalysis.ThreadRelation == c.RelContinues {
					id := rendered.FocusThread.ID
					thread = &id
				}
				allowed, e := s.opts.Facts.HypothesisAllowed(ctx, meta.UserID, meta.TurnID, thread, *h.Tag)
				if e == nil && allowed {
					if facts.tags == nil {
						facts.tags = map[string]bool{}
					}
					facts.tags[*h.Tag] = true
				}
			}
		}
		if decision.Plan.Protocol != nil && decision.Plan.Protocol.Action == c.PAOffer {
			facts.offer, _ = s.opts.Facts.OfferAllowed(ctx, meta.UserID, meta.TurnID, s.assets.gate.Protocols.OfferMinTurnsSinceLastOffer, s.assets.gate.Protocols.MaxOffersPerSession)
		}
		if decision.Plan.VoiceHypothesisID != nil {
			facts.voice, _ = s.opts.Facts.VoiceAllowed(ctx, meta.UserID, meta.TurnID, *decision.Plan.VoiceHypothesisID, time.Duration(s.assets.gate.Hypotheses.VoiceMinGapHours)*time.Hour, s.assets.gate.Hypotheses.VoiceMaxPerSession)
		}
	}
	in.WCtx = rendered
	out.Decision, out.Repairs = s.assets.validate(decision, in, facts)
	if ctx.Err() != nil {
		return ports.BrainOutput{}, ctx.Err()
	}
	// Verify deterministic repairs also conform to exactly the pinned schema.
	checked, e := json.Marshal(out.Decision)
	if e != nil || s.assets.schemas.Validate(c.BrainSchemaVersion, checked) != nil {
		return fallback("BR-001")
	}
	return out, nil
}
func sameBundle(a, b c.BundleRef) bool {
	return a.Version != "" && a.Version == b.Version && reflect.DeepEqual(a.PromptSHAs, b.PromptSHAs)
}
func ownedContext(w c.WorkingContext, s contextstate.Snapshot) bool {
	if w.FocusThread != nil {
		t, ok := contextstate.FindThread(s, w.FocusThread.ID)
		if !ok || t.UserID != s.User.UserID {
			return false
		}
	}
	for _, v := range w.OtherThreads {
		if _, ok := contextstate.FindThread(s, v.ID); !ok {
			return false
		}
	}
	if w.Protocol != nil && (s.Protocol == nil || w.Protocol.InstanceID != s.Protocol.ID || s.Protocol.UserID != s.User.UserID) {
		return false
	}
	return true
}
