// Package orchestrator executes WP-05's deterministic, non-streaming turn DAG.
// It deliberately delegates all psychological reasoning, safety, validation,
// retrieval and generation to injectable ports: none is implemented here.
package orchestrator

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"time"
	"unicode/utf16"

	"golang.org/x/text/unicode/norm"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/brain"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/generator"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/protocol"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/safety"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/validator"
)

// Fallback selects externally reviewed, route- and locale-specific text. An
// absent fallback deliberately leaves a durable failed USER-ONLY turn. WP-03
// will supply approved content; WP-05 must not invent clinical templates.
type Fallback func(route contract.Route, objective contract.Objective, locale string) (string, bool)

type Dependencies struct {
	Reflection    TurnReflection
	ProtocolFacts ProtocolFacts
	Gateway       ports.Gateway
	Store         ports.ConversationStore
	Finalizer     ports.TurnFinalizer
	Recovery      ports.TurnRecovery // optional: explicit retry, cancellation and heartbeat
	Router        ports.Router
	Safety        ports.SafetyGate
	Assembler     ports.ContextAssembler
	Retriever     ports.Retriever // WP-09, requiring an explicit AI-use policy
	Brain         ports.Brain
	Generator     ports.Generator
	Validator     ports.Validator
	Crisis        ports.CrisisPlanner  // required only for a crisis route
	Memory        TurnMemory           // WP-12, explicitly composed with current consent policy
	Protocol      ports.ProtocolEngine // WP-13, approved and pinned deterministic interventions
	Tracer        ports.Tracer
}
type Options struct {
	RouterConfig     ports.RouterConfig
	Fallback         Fallback
	RecentLimit      int
	TotalTimeout     time.Duration
	BrainTimeout     time.Duration
	GeneratorTimeout time.Duration
	RenewInterval    time.Duration // 0 disables renewal; total must fit the SQL lease
}
type Service struct {
	deps        Dependencies
	opts        Options
	integration *integration // nil retains the isolated WP-05 walking skeleton
}

var _ ports.Orchestrator = (*Service)(nil)

func New(d Dependencies, opts Options) (*Service, error) {
	if d.Gateway == nil || d.Store == nil || d.Finalizer == nil || d.Safety == nil || d.Assembler == nil || d.Brain == nil || d.Generator == nil || d.Validator == nil {
		return nil, errors.New("WP-05 requires durable persistence, safety, context, brain, generator and validator ports")
	}
	if d.Router == nil {
		d.Router = Router{}
	}
	if opts.RecentLimit == 0 {
		opts.RecentLimit = 16
	}
	if opts.TotalTimeout == 0 {
		opts.TotalTimeout = 45 * time.Second
	}
	if opts.BrainTimeout == 0 {
		opts.BrainTimeout = 15 * time.Second
	}
	if opts.GeneratorTimeout == 0 {
		opts.GeneratorTimeout = 20 * time.Second
	}
	if opts.RecentLimit < 1 || opts.RecentLimit > 200 || opts.TotalTimeout <= 0 || opts.TotalTimeout > time.Minute || opts.BrainTimeout <= 0 || opts.GeneratorTimeout <= 0 || opts.RenewInterval < 0 || (opts.RenewInterval != 0 && d.Recovery == nil) {
		return nil, errors.New("invalid WP-05 budgets/recovery configuration")
	}
	return &Service{deps: d, opts: opts}, nil
}

// ProcessTurn does NOT expose any draft token. Admission happens before all
// model work; one finalizer transaction makes the assistant message visible.
// Replaying a completed/failed/cancelled key never calls the model again.
func (o *Service) ProcessTurn(ctx context.Context, req contract.TurnRequest) (contract.TurnResult, error) {
	// Wake-up events are internal outbox metadata, not user-authored messages.
	// Until a reviewed synthetic-turn admission/finalization contract exists,
	// reject BEFORE Gateway.Admit: a substituted gateway must not fabricate a
	// user message or lease an ordinary turn for a proactive event.
	if req.Origin == contract.OriginWakeUp || req.WakeUp != nil {
		return contract.TurnResult{}, coreError(contract.ErrForbidden, "wakeup_contract", nil)
	}
	req.Text = norm.NFC.String(req.Text)
	a, err := o.deps.Gateway.Admit(ctx, req)
	if err != nil {
		return contract.TurnResult{}, err
	}
	if a.UserID != req.UserID || a.ConversationID != req.ConversationID || !contract.ValidID(a.TurnID) || !contract.ValidID(a.UserMessageID) {
		return contract.TurnResult{}, coreError(contract.ErrInternal, "admit", nil)
	}
	if a.Queued || !contract.ValidID(a.LeaseToken) {
		// Includes already-finalized and live in-flight duplicate admission.
		// Only the lease owner can begin any side-effecting inference.
		return o.deps.Store.GetTurn(ctx, req.UserID, req.ClientTurnID)
	}
	if a.DuplicateOf != nil && *a.DuplicateOf != a.TurnID {
		return contract.TurnResult{}, coreError(contract.ErrStateConflict, "admit", nil)
	}

	turnTimeout := o.opts.TotalTimeout
	if g, ok := o.deps.Generator.(interface {
		WP11()
		TurnTimeout() time.Duration
	}); ok && g.TurnTimeout() < turnTimeout {
		turnTimeout = g.TurnTimeout()
	}
	deadline := time.Now().Add(turnTimeout)
	if b, ok := contract.BudgetFromContext(ctx); ok && !b.TotalDeadline.IsZero() && b.TotalDeadline.Before(deadline) {
		deadline = b.TotalDeadline
	}
	turnCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	var tr ports.TurnTrace
	if o.deps.Tracer != nil {
		meta := contract.CallMeta{TurnID: a.TurnID, UserID: a.UserID}
		if o.integration != nil {
			meta.Bundle = o.integration.assets.Reference()
		}
		tr = o.deps.Tracer.Begin(a.TurnID, meta)
		defer tr.Finish(context.WithoutCancel(ctx))
		tr.Record("admit", trace.Observation{Count: 1})
	}
	stopLease := o.watchLease(turnCtx, cancel, a)
	var p contract.PreparedTurn
	var stage string
	var runErr error
	if o.integration != nil {
		turnCtx, runErr = o.integration.pin(turnCtx, a)
		if runErr != nil {
			stage = "bundle"
		}
	}
	if runErr == nil {
		p, stage, runErr = o.prepare(turnCtx, req, a, tr)
	}
	// Stop the renewer BEFORE entering the short finalization transaction.
	// A fenced writer whose lease was lost may never complete or fail a turn.
	if leaseErr := stopLease(); leaseErr != nil {
		runErr = coreError(contract.ErrStateConflict, "recovery", leaseErr)
		stage = "recovery"
	}
	if runErr == nil {
		if err = turnCtx.Err(); err == nil {
			started := time.Now()
			result, completeErr := o.deps.Finalizer.Complete(turnCtx, a, p)
			record(tr, "finalizer", started, completeErr != nil)
			if completeErr == nil {
				return result, nil
			}
			runErr = coreError(contract.ErrStateCommit, "finalizer", completeErr)
			stage = "finalizer"
		} else {
			runErr = err
			stage = "finalizer"
		}
	}
	if tr != nil {
		tr.Fail(stage, publicFailure(runErr))
	}
	return o.failTurn(ctx, a, req.ClientTurnID, runErr)
}

func (o *Service) failTurn(ctx context.Context, a contract.Admission, client contract.ID, reason error) (contract.TurnResult, error) {
	code := failureCode(reason)
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	var cleanupErr error
	if errors.Is(ctx.Err(), context.Canceled) {
		cleanupErr = o.deps.Gateway.Release(cleanup, a, ports.OutcomeCancelled)
	} else {
		cleanupErr = o.deps.Finalizer.Fail(cleanup, a, code)
	}
	// A failed cleanup may mean SQL is down or the lease was fenced out. In
	// either case never return success or mutate the message independently.
	if cleanupErr != nil {
		return contract.TurnResult{TurnID: a.TurnID, UserMessageID: a.UserMessageID, Status: contract.TurnRunning}, coreError(contract.ErrStateCommit, "recovery", cleanupErr)
	}
	r, err := o.deps.Store.GetTurn(cleanup, a.UserID, client)
	if err != nil {
		return contract.TurnResult{TurnID: a.TurnID, UserMessageID: a.UserMessageID, Status: contract.TurnFailed}, publicFailure(reason)
	}
	return r, publicFailure(reason)
}

// RecoverNext is an explicit, authenticated-worker entrypoint: it reuses the
// original persisted request and competes for the expired per-user lease. No
// user-message row is inserted twice, including after process death.
func (o *Service) RecoverNext(ctx context.Context, user contract.ID) (contract.TurnResult, error) {
	if o.deps.Recovery == nil {
		return contract.TurnResult{}, coreError(contract.ErrInternal, "recovery", nil)
	}
	if !contract.ValidID(user) {
		return contract.TurnResult{}, coreError(contract.ErrValidation, "recovery", nil)
	}
	req, err := o.deps.Recovery.Next(ctx, user)
	if err != nil {
		return contract.TurnResult{}, err
	}
	if req.UserID != user {
		return contract.TurnResult{}, coreError(contract.ErrForbidden, "recovery", nil)
	}
	return o.ProcessTurn(ctx, req)
}

// RetryFailed is an explicit owner-scoped status change, NOT a replay.
// After queueing, RecoverNext loads the ORIGINAL persisted request metadata;
// passing a caller-edited body into ProcessTurn would strand the queued turn.
// An HTTP retry endpoint is reserved for WP-16 after authorization review.
func (o *Service) RetryFailed(ctx context.Context, user, client contract.ID) error {
	if o.deps.Recovery == nil {
		return coreError(contract.ErrInternal, "recovery", nil)
	}
	if !contract.ValidID(user) || !contract.ValidID(client) {
		return coreError(contract.ErrValidation, "recovery", nil)
	}
	prev, err := o.deps.Store.GetTurn(ctx, user, client)
	if err != nil {
		return err
	} // GetTurn enforces owner + soft-delete visibility
	if prev.Status != contract.TurnFailed && prev.Status != contract.TurnCancelled {
		return coreError(contract.ErrStateConflict, "recovery", nil)
	}
	return o.deps.Recovery.Retry(ctx, user, client)
}

func coreError(code contract.Code, step string, cause error) error {
	return &contract.Error{Code: code, Step: step, Cause: cause, Retryable: code == contract.ErrStateCommit || code == contract.ErrStateConflict || code == contract.ErrProvider}
}
func failureCode(err error) contract.Code {
	var e *contract.Error
	if errors.As(err, &e) && e.Code.Valid() {
		return e.Code
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return contract.ErrBudget
	}
	return contract.ErrInternal
}
func publicFailure(err error) error {
	var e *contract.Error
	if errors.As(err, &e) && e.Code.Valid() {
		return &contract.Error{Code: e.Code, Step: e.Step, Retryable: e.Retryable}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return coreError(contract.ErrBudget, "deadline", nil)
	}
	return coreError(contract.ErrInternal, "turn", nil)
}
func record(tr ports.TurnTrace, name string, at time.Time, degraded bool) {
	if tr != nil {
		tr.Record(name, trace.Observation{Duration: time.Since(at), Degraded: degraded})
	}
}

// watchLease renews only the fenced SQL lease, never holds a SQL transaction
// while awaiting an AI stage. Any renewal conflict cancels model work.
func (o *Service) watchLease(ctx context.Context, cancel context.CancelFunc, a contract.Admission) func() error {
	if o.deps.Recovery == nil || o.opts.RenewInterval == 0 {
		return func() error { return nil }
	}
	stop := make(chan struct{})
	finished := make(chan struct{})
	lost := make(chan error, 1)
	go func() {
		defer close(finished)
		ticker := time.NewTicker(o.opts.RenewInterval)
		defer ticker.Stop()
		h := ports.LeaseHandle{UserID: a.UserID, TurnID: a.TurnID, Token: a.LeaseToken}
		for {
			select {
			case <-stop:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				next, err := o.deps.Recovery.Renew(ctx, h)
				if err != nil {
					lost <- err
					cancel()
					return
				}
				h = next
			}
		}
	}()
	return func() error {
		close(stop)
		<-finished
		select {
		case err := <-lost:
			return err
		default:
			return nil
		}
	}
}

func (o *Service) prepare(ctx context.Context, req contract.TurnRequest, a contract.Admission, tr ports.TurnTrace) (contract.PreparedTurn, string, error) {
	started := time.Now()
	messages, err := o.deps.Store.Recent(ctx, a.UserID, a.ConversationID, o.opts.RecentLimit)
	record(tr, "recent", started, err != nil)
	if err != nil {
		return contract.PreparedTurn{}, "recent", err
	}
	var prior contextstate.Snapshot
	if o.integration != nil {
		prior, err = o.integration.reader.ForTurn(ctx, a.UserID, a.TurnID)
		if err != nil {
			return contract.PreparedTurn{}, "state_read", err
		}
		versions := contextstate.Versions(prior)
		if prior.User.UserID != a.UserID || prior.Session.UserID != a.UserID ||
			prior.Session.ConversationID != a.ConversationID || prior.Session.ID != a.SessionID ||
			prior.Session.ClosedAt != nil || versions.Session != a.Versions.Session || versions.User != a.Versions.User ||
			!prior.User.Risk.Level.Valid() {
			return contract.PreparedTurn{}, "state_read", coreError(contract.ErrStateConflict, "state_read", nil)
		}
		// WP-02 may already have admitted later queued messages: their text
		// must never contaminate the current turn's safety classification.
		trigger := -1
		for n, m := range messages {
			if m.ID == a.UserMessageID && m.Role == contract.RoleUser {
				trigger = n
				break
			}
		}
		if trigger < 0 {
			return contract.PreparedTurn{}, "recent", coreError(contract.ErrStateConflict, "recent", nil)
		}
		messages = messages[:trigger]
	}
	history := make([]contract.MsgView, 0, len(messages))
	for _, m := range messages {
		if m.ID == a.UserMessageID {
			continue
		} // triggering message appears in TurnView, not twice
		if m.Role == contract.RoleUser || m.Role == contract.RoleAssistant {
			history = append(history, contract.MsgView{ID: m.ID, Role: m.Role, Text: m.Content})
		}
	}
	// Safety and speculative retrieval are independent. The assembler's port
	// requires both outputs, so it runs after this bounded parallel fanout.
	type safetyResult struct {
		signal contract.SafetySignal
		err    error
	}
	type retrieveResult struct {
		result contract.RetrievalResult
		err    error
	}
	safetyCh := make(chan safetyResult, 1)
	retrieveCh := make(chan retrieveResult, 1)
	started = time.Now()
	go func() {
		previousRisk := contract.RiskState{Level: contract.RiskElevated}
		if o.integration != nil {
			previousRisk = prior.User.Risk
		}
		sig, err := o.deps.Safety.Assess(ctx, ports.SafetyInput{Text: req.Text, Recent: history, Lang: req.Locale, Risk: previousRisk})
		safetyCh <- safetyResult{sig, err}
	}()
	go func() {
		if o.deps.Retriever == nil {
			retrieveCh <- retrieveResult{result: contract.RetrievalResult{Meta: contract.RetrievalMeta{Degraded: true}}}
			return
		}
		turn := contract.TurnView{TurnID: a.TurnID, MessageID: a.UserMessageID, Text: req.Text, Lang: req.Locale, Origin: req.Origin}
		var focus *contract.ID
		if o.integration != nil {
			focus = prior.Session.FocusThreadID
		}
		exclude := make([]contract.ID, 0, len(history))
		for _, m := range history {
			exclude = append(exclude, m.ID)
		}
		ret, err := o.deps.Retriever.Retrieve(ctx, contract.RetrievalRequest{UserID: a.UserID, Turn: turn, FocusThread: focus, Exclude: exclude, MaxItems: 5})
		retrieveCh <- retrieveResult{ret, err}
	}()
	var safe safetyResult
	var retr retrieveResult
	for i := 0; i < 2; i++ {
		select {
		case safe = <-safetyCh:
			safetyCh = nil
		case retr = <-retrieveCh:
			retrieveCh = nil
		case <-ctx.Done():
			return contract.PreparedTurn{}, "safety", ctx.Err()
		}
	}
	// Without WP-06's full classifier/L0, adapters MUST fail conservatively.
	// An error never lowers risk; the clinician-owned Safety implementation
	// will supply real L0/L1 and the persisted previous RiskState later.
	if safe.err != nil || !safe.signal.Level.Valid() {
		if !safe.signal.Level.Valid() || riskRank(safe.signal.Level) < riskRank(contract.RiskElevated) {
			safe.signal.Level = contract.RiskElevated
		}
		safe.signal.Degraded = true
		safe.signal.Source = "timeout_default"
	}
	record(tr, "safety", started, safe.signal.Degraded)
	if retr.err != nil {
		retr.result = contract.RetrievalResult{Meta: contract.RetrievalMeta{Degraded: true}}
	}
	if o.deps.Retriever == nil {
		retr.result.Meta.Degraded = true
	}
	record(tr, "retrieval", started, retr.result.Meta.Degraded)

	started = time.Now()
	wc, stateSnapshot, err := o.deps.Assembler.Build(ctx, req, safe.signal, retr.result)
	record(tr, "assembler", started, err != nil)
	if err != nil {
		return contract.PreparedTurn{}, "assembler", err
	}
	if o.integration != nil {
		initial := contextstate.Versions(prior)
		if stateSnapshot.Session.ID != a.SessionID || stateSnapshot.User.UserID != a.UserID ||
			stateSnapshot.Session.UserID != a.UserID || stateSnapshot.Session.ConversationID != a.ConversationID ||
			stateSnapshot.Versions != initial {
			return contract.PreparedTurn{}, "assembler", coreError(contract.ErrStateConflict, "assembler", nil)
		}
	}
	if o.integration == nil {
		wc.Turn = contract.TurnView{TurnID: a.TurnID, MessageID: a.UserMessageID, Text: req.Text, Lang: req.Locale, Origin: req.Origin}
	} else if wc.Turn.TurnID != a.TurnID || wc.Turn.MessageID != a.UserMessageID {
		return contract.PreparedTurn{}, "assembler", coreError(contract.ErrStateConflict, "assembler", nil)
	}
	if wc.Recent == nil {
		wc.Recent = history
	}
	if retr.result.Meta.Degraded {
		wc.RetrievalMeta.Degraded = true
		wc.Eligibility.MemoryAvailable = false
		wc.Retrieved = nil
	}
	if !wc.Risk.Current.Valid() {
		wc.Risk.Current = contract.RiskElevated
	}
	if riskRank(wc.Risk.Current) > riskRank(safe.signal.Level) {
		safe.signal.Level = wc.Risk.Current
	}
	var riskChange safety.RiskChange
	if o.integration != nil {
		riskChange = o.integration.fsm.Advance(o.integration.now(), a.TurnID, prior.User.Risk, safe.signal, safety.ClearanceEvidence{})
		if riskRank(riskChange.State.Level) > riskRank(safe.signal.Level) {
			safe.signal.Level = riskChange.State.Level
		}
		wc.Risk.Current = safe.signal.Level
		wc.Risk.FollowUpActive = riskChange.FollowUpActive
		if riskRank(safe.signal.Level) >= riskRank(contract.RiskElevated) || riskChange.FollowUpActive {
			wc.Eligibility.ProtocolsPossible = false
			wc.Eligibility.MayVoiceHypothesis = false
		}
	}

	decision := o.deps.Router.Select(ports.RouteInput{Safety: safe.signal, WCtx: wc, Origin: req.Origin, Config: o.opts.RouterConfig})
	if !decision.Route.Valid() || (riskRank(safe.signal.Level) >= 3 && decision.Route != contract.RouteCrisis) || (req.Origin == contract.OriginWakeUp && decision.Route != contract.RouteProactive && decision.Route != contract.RouteCrisis) {
		return contract.PreparedTurn{}, "route", coreError(contract.ErrValidation, "route", nil)
	}
	if tr != nil {
		tr.SetRoute(decision.Route)
	}
	if tr != nil {
		tr.Record("route", trace.Observation{RuleID: decision.Reason, Degraded: safe.signal.Degraded})
	}
	if decision.Route == contract.RouteProactive {
		// Defense in depth: no PROACTIVE turn is authorized by the current
		// admission/finalization contract. No unconsented proactive output.
		return contract.PreparedTurn{}, "route", coreError(contract.ErrValidation, "proactive", nil)
	}

	// Pre-Brain WP-13 shortlist. Never show the model candidates without
	// explicit readiness and a verifiable owner-scoped snapshot. Candidate
	// scoring and contraindications are deterministic and bundle-defined.
	if o.integration != nil && o.deps.Protocol != nil && decision.Route == contract.RouteNormal && wc.Protocol == nil {
		if p, ok := o.deps.Protocol.(interface {
			ShortlistWithContext(context.Context, protocol.ShortlistContext) ([]contract.ProtocolBrief, error)
		}); ok && wc.Eligibility.ProtocolsPossible {
			readiness := protocol.ExplicitReadiness(wc.Turn.Text)
			if readiness != protocol.ReadinessUnknown {
				flags := []string{}
				verified := false
				if o.deps.ProtocolFacts != nil && stateSnapshot.Focus != nil {
					var e error
					flags, verified, e = o.deps.ProtocolFacts.Flags(ctx, a.UserID, stateSnapshot.Focus.ID)
					if e != nil {
						verified = false
						flags = nil
					}
				}
				candidates, e := p.ShortlistWithContext(ctx, protocol.ShortlistContext{WCtx: wc, Thread: stateSnapshot.Focus, User: stateSnapshot.User, Readiness: readiness, Flags: flags, FlagsVerified: verified, Now: o.integration.now()})
				if e != nil {
					return contract.PreparedTurn{}, "protocol_shortlist", e
				}
				wc.Shortlist = candidates
			}
		}
		if len(wc.Shortlist) == 0 {
			wc.Eligibility.ProtocolsPossible = false
			wc.Eligibility.ReasonsBlocked = append(wc.Eligibility.ReasonsBlocked, "no_eligible_reviewed_protocol")
		}
	}
	objective := contract.ObjReflect
	plan := restrictedPlan()
	degraded := safe.signal.Degraded || retr.result.Meta.Degraded
	var crisis *contract.CrisisPlan
	var banner *contract.SafetyBanner
	var vettedBrain *contract.BrainDecision
	if g, ok := o.deps.Generator.(interface {
		WP11()
		MaxSyncCalls() int
	}); ok {
		limit := g.MaxSyncCalls()
		if b, ok := contract.BudgetFromContext(ctx); ok && b.MaxLLMCalls > 0 && b.MaxLLMCalls < limit {
			limit = b.MaxLLMCalls
		}
		ctx = contract.WithCallAllowance(ctx, limit)
	}
	switch decision.Route {
	case contract.RouteFast:
		objective = contract.ObjAnswerDirect
	case contract.RouteNormal:
		var output ports.BrainOutput
		if _, wp10 := o.deps.Brain.(turnBrain); wp10 {
			var brainDegraded bool
			output, wc, brainDegraded, err = o.decideWP10(ctx, a, wc, safe.signal, tr)
			if err != nil {
				return contract.PreparedTurn{}, "brain", err
			}
			degraded = degraded || brainDegraded
			objective, plan = output.Decision.Decision.Primary, output.Decision.Plan
			vettedBrain = &output.Decision
		} else {
			started = time.Now()
			brainCtx, cancel := context.WithTimeout(ctx, o.opts.BrainTimeout)
			var e error
			output, e = o.deps.Brain.Decide(brainCtx, ports.BrainInput{WCtx: wc, Safety: safe.signal, Bundle: o.bundleReference(ctx)})
			cancel()
			if tr != nil {
				tr.AddUsage(output.Usage)
			}
			record(tr, "brain", started, e != nil)
			if e != nil || output.Decision.SchemaVersion != "brain.v1" || !output.Decision.Decision.Primary.Valid() || output.Decision.Plan.Protocol != nil || output.Decision.Plan.VoiceHypothesisID != nil {
				degraded = true // FB-001: restricted reflect plan, never unvetted intervention
			} else {
				// No Policy Gate, clinical Skills or Protocol Engine in WP-05.
				// Only conservative, non-interventional objectives are admissible.
				switch output.Decision.Decision.Primary {
				case contract.ObjReflect, contract.ObjValidateSupport, contract.ObjClarify:
					objective = output.Decision.Decision.Primary
					if o.integration != nil {
						vettedBrain = &output.Decision
					}
				default:
					degraded = true
				}
			}
		}
		// The Brain may only RAISE risk; re-route before generation.
		if output.Decision.RiskRead != nil && output.Decision.RiskRead.Level.Valid() && riskRank(output.Decision.RiskRead.Level) > riskRank(safe.signal.Level) {
			safe.signal.Level = output.Decision.RiskRead.Level
			for _, cat := range output.Decision.RiskRead.Categories {
				if !cat.Valid() || len(safe.signal.Categories) >= 3 {
					continue
				}
				found := false
				for _, present := range safe.signal.Categories {
					if present == cat {
						found = true
						break
					}
				}
				if !found {
					safe.signal.Categories = append(safe.signal.Categories, cat)
				}
			}
			if o.integration != nil {
				riskChange = o.integration.fsm.Advance(o.integration.now(), a.TurnID, prior.User.Risk, safe.signal, safety.ClearanceEvidence{})
				wc.Risk.Current = riskChange.State.Level
				wc.Risk.FollowUpActive = riskChange.FollowUpActive
			}
			if riskRank(safe.signal.Level) >= 3 {
				decision.Route = contract.RouteCrisis
				if tr != nil {
					tr.SetRoute(contract.RouteCrisis)
				}
			}
		}
	case contract.RouteCrisis:
		// No free-form Brain call on a high-risk signal.
	default:
		return contract.PreparedTurn{}, "route", coreError(contract.ErrValidation, "route", nil)
	}
	if vettedBrain != nil && decision.Route == contract.RouteNormal {
		if vettedBrain.Plan.Protocol != nil && o.deps.Protocol == nil {
			return contract.PreparedTurn{}, "brain_dependency", coreError(contract.ErrValidation, "brain_dependency", nil)
		}
		if target := vettedBrain.TurnAnalysis.CorrectionTarget; target != nil {
			if err := o.correctMemory(ctx, a, *target); err != nil {
				return contract.PreparedTurn{}, "memory_correction", err
			}
			// The already-built context is stale after the synchronous correction.
			// Exclude the target from every generation surface, including hypotheses.
			wc, plan = withoutCorrectedMemory(wc, plan, *target)
			signals := vettedBrain.Understanding.Hypotheses[:0]
			for _, signal := range vettedBrain.Understanding.Hypotheses {
				if signal.ID == nil || *signal.ID != *target {
					signals = append(signals, signal)
				}
			}
			vettedBrain.Understanding.Hypotheses = signals
		}
	}

	// Resolve from the persisted offered/active instance, never from the
	// model's assumed state. ResolveTurn verifies ownership and consent evidence.
	var resolvedSpec *contract.ProtocolStepSpec
	var resolvedPatch *contract.ProtocolPatch
	if decision.Route == contract.RouteNormal && plan.Protocol != nil && o.integration != nil && o.deps.Protocol != nil {
		engine, ok := o.deps.Protocol.(interface {
			ResolveTurn(context.Context, protocol.ResolveInput) (contract.ProtocolStepSpec, contract.ProtocolPatch, error)
		})
		if !ok {
			return contract.PreparedTurn{}, "protocol_integration", coreError(contract.ErrValidation, "protocol_integration", nil)
		}
		spec, patch, resolveErr := engine.ResolveTurn(ctx, protocol.ResolveInput{
			UserID: a.UserID, TurnID: a.TurnID, UserMessageID: a.UserMessageID, Bundle: o.bundleReference(ctx),
			Plan: plan, Objective: objective, Active: stateSnapshot.Protocol, WCtx: wc, Shortlist: wc.Shortlist,
			UserAffirmed: wc.Protocol != nil && (wc.Protocol.Status == contract.PIOffered || wc.Protocol.Status == contract.PIPaused) && protocol.AffirmedOffer(wc.Turn.Text), Risk: safe.signal.Level,
		})
		if resolveErr != nil {
			// Invalid protocol decisions are never sent to the generator or committed.
			degraded = true
			plan = restrictedPlan()
			objective = contract.ObjReflect
		} else {
			resolvedPatch = &patch
			if spec.ProtocolID != "" {
				resolvedSpec = &spec
			}
			if resolvedSpec == nil {
				plan = restrictedPlan()
				objective = contract.ObjCloseOrPause
			}
		}
	}
	if resolvedSpec != nil && plan.Protocol != nil {
		resolver, ok := o.deps.Protocol.(interface {
			RequiredText(contract.ProtocolStepSpec) (map[string]string, error)
		})
		if !ok {
			return contract.PreparedTurn{}, "protocol_requirements", coreError(contract.ErrValidation, "protocol_requirements", nil)
		}
		required, e := resolver.RequiredText(*resolvedSpec)
		if e != nil {
			return contract.PreparedTurn{}, "protocol_requirements", e
		}
		ctx = validator.WithProtocolRequirements(ctx, resolvedSpec, required)
	}
	if decision.Route == contract.RouteCrisis {
		if o.deps.Crisis == nil {
			return contract.PreparedTurn{}, "route", coreError(contract.ErrSafetyDegraded, "crisis", nil)
		}
		currentRisk := contract.RiskState{Level: safe.signal.Level}
		if o.integration != nil {
			currentRisk = riskChange.State
		}
		cp, err := o.deps.Crisis.Plan(ctx, safe.signal, req.Locale, currentRisk)
		if err != nil || !cp.Buffered || strings.TrimSpace(cp.TemplateText) == "" {
			return contract.PreparedTurn{}, "route", coreError(contract.ErrSafetyDegraded, "crisis", err)
		}
		crisis = &cp
		banner = &contract.SafetyBanner{Level: safe.signal.Level, Resources: cp.Resources}
		objective = contract.ObjSafetySupport
		plan = restrictedPlan()
	}
	// An elevated-risk turn must terminate an outstanding offer even when the
	// Brain is bypassed (crisis route) or its protocol directive is removed by
	// the safety gate. This status-only patch is independent of reply delivery.
	safetyInterruption := false
	if o.integration != nil && o.deps.Protocol != nil && stateSnapshot.Protocol != nil &&
		stateSnapshot.Protocol.Status == contract.PIOffered &&
		(riskRank(safe.signal.Level) >= riskRank(contract.RiskElevated) || wc.Risk.FollowUpActive) {
		engine, ok := o.deps.Protocol.(interface {
			ResolveTurn(context.Context, protocol.ResolveInput) (contract.ProtocolStepSpec, contract.ProtocolPatch, error)
		})
		if !ok {
			return contract.PreparedTurn{}, "protocol_safety", coreError(contract.ErrValidation, "protocol_safety", nil)
		}
		_, patch, e := engine.ResolveTurn(ctx, protocol.ResolveInput{
			UserID: a.UserID, TurnID: a.TurnID, UserMessageID: a.UserMessageID,
			Bundle: o.bundleReference(ctx), Active: stateSnapshot.Protocol, WCtx: wc,
			Plan: contract.ResponsePlan{Protocol: &contract.ProtocolDirective{Action: contract.PAStop}}, Risk: safe.signal.Level,
		})
		if e != nil || patch.NewStatus == nil || *patch.NewStatus != contract.PIStopped {
			return contract.PreparedTurn{}, "protocol_safety", coreError(contract.ErrStateConflict, "protocol_safety", e)
		}
		resolvedPatch = &patch
		resolvedSpec = nil
		safetyInterruption = true
	}
	reply, wasFallback, finalObjective, validationReport, usedTemplate, outageCrisis, err := o.generateValidated(ctx, wc, plan, objective, resolvedSpec, safe.signal.Level, crisis, decision.Route, req.Locale, tr)
	objective = finalObjective
	if err != nil {
		return contract.PreparedTurn{}, "validator", err
	}
	degraded = degraded || wasFallback
	if outageCrisis != nil {
		crisis = outageCrisis
		banner = &contract.SafetyBanner{Level: safe.signal.Level, Resources: crisis.Resources}
		decision.Route = contract.RouteCrisis
		plan = restrictedPlan()
		if tr != nil {
			tr.SetRoute(decision.Route)
		}
	}
	if usedTemplate || wasFallback || outageCrisis != nil {
		vettedBrain = nil
		if !safetyInterruption {
			resolvedPatch = nil // never commit an offer/step the user did not receive
		}
	}
	delta := contract.StateDelta{TurnID: a.TurnID, UserID: a.UserID, ExpectedVersions: a.Versions,
		Session: contract.SessionPatch{LastObjective: objective, TurnCountInc: 1}}
	if o.integration != nil {
		delta.ExpectedVersions = stateSnapshot.Versions
		delta.Session.ConsecutiveQuestions = wc.Session.ConsecutiveQuestions
		if validationReport.Extracted.QuestionText != nil && validationReport.Extracted.QuestionCount == 1 {
			delta.Session.PendingQuestion = &contract.PendingQuestion{Text: *validationReport.Extracted.QuestionText, Slot: plan.Question.Slot, AskedAt: o.integration.now().UTC()}
			delta.Session.ConsecutiveQuestions++
		} else {
			delta.Session.ConsecutiveQuestions = 0
		}
		// WP-10 proposes creation/shift/extraction only after decision gates.
		// WP-08 validates ownership, readiness and transitions again; the
		// isolated WP-05 path retains its original restricted state bridge.
		if vettedBrain != nil && decision.Route == contract.RouteNormal {
			if _, wp10 := o.deps.Brain.(turnBrain); wp10 {
				brain.ProposeDelta(&delta, *vettedBrain, wc, stateSnapshot)
			} else if riskRank(safe.signal.Level) < riskRank(contract.RiskElevated) {
				o.integrateBrainDelta(&delta, *vettedBrain, wc, stateSnapshot)
			}
		}
		if resolvedPatch != nil {
			delta.Protocol = resolvedPatch
			if resolvedPatch.NewStatus != nil {
				switch *resolvedPatch.NewStatus {
				case contract.PICompleted, contract.PIStopped, contract.PIDeclined:
					if stateSnapshot.Protocol != nil {
						outcome := string(*resolvedPatch.NewStatus)
						delta.User.ProtocolHistoryAdd = &contract.ProtocolHistoryBrief{ProtocolID: stateSnapshot.Protocol.ProtocolID, Outcome: outcome, At: o.integration.now().UTC()}
					}
				}
			}
		}
		if riskChange.Transition != nil && !riskChange.AddToDelta(&delta, safe.signal.Categories) {
			return contract.PreparedTurn{}, "risk_delta", coreError(contract.ErrValidation, "risk_delta", nil)
		}
		started = time.Now()
		checked, rejected, validationErr := o.integration.committer.ValidateDelta(ctx, delta)
		record(tr, "state_validate", started, validationErr != nil || len(rejected) != 0)
		if validationErr != nil {
			return contract.PreparedTurn{}, "state_validate", validationErr
		}
		if resolvedPatch != nil && (checked.Protocol == nil || !reflect.DeepEqual(*checked.Protocol, *resolvedPatch)) {
			return contract.PreparedTurn{}, "protocol_commit", coreError(contract.ErrStateConflict, "protocol_commit", nil)
		}
		if riskChange.Transition != nil && (checked.User.Risk == nil || *checked.User.Risk != *riskChange.Transition || !hasRiskEvent(checked.Events)) {
			// Never complete a turn while silently dropping an escalation.
			return contract.PreparedTurn{}, "risk_delta", coreError(contract.ErrStateConflict, "risk_delta", nil)
		}
		// WP-08 can remove every proposed slot/phase after rejecting
		// unverifiable evidence. Do not turn that no-op into a real SQL
		// thread update (which would increment its version and touch time).
		if checked.Thread != nil && checked.Thread.Create == nil && checked.Thread.Phase == nil &&
			len(checked.Thread.Slots) == 0 && len(checked.Thread.HypothesisSignals) == 0 {
			checked.Thread = nil
		}
		delta = checked
		if len(rejected) > 0 {
			degraded = true
		}
	}
	var jobs []contract.AsyncJobProposal
	if o.deps.Memory != nil && vettedBrain != nil && decision.Route == contract.RouteNormal {
		jobCtx := o.memoryJobContext(ctx, a, delta, wc, validationReport.Extracted)
		job, err := o.deps.Memory.PlanJob(jobCtx, a, *vettedBrain, validationReport.Extracted.ClaimsAboutUser)
		if err != nil {
			// Extraction is off the reply's critical path. An approved response
			// can complete; a later explicit sweep may recover the persisted turn.
			degraded = true
			record(tr, "memory_schedule", time.Now(), true)
			job = nil
		}
		if job != nil {
			jobs = append(jobs, *job)
		}
	}
	if o.deps.Reflection != nil && vettedBrain != nil && decision.Route == contract.RouteNormal {
		if job := o.deps.Reflection.PlanJob(a, *vettedBrain, delta); job != nil {
			jobs = append(jobs, *job)
		}
	}
	return contract.PreparedTurn{
		TurnID: a.TurnID, UserMessageID: a.UserMessageID, Text: reply,
		Route: decision.Route, Objective: objective, Degraded: degraded,
		State:  delta,
		Trace:  contract.TurnTraceSummary{TurnID: a.TurnID, BundleVersion: o.bundleReference(ctx).Version, Route: decision.Route, Objective: objective},
		Safety: banner,
		Jobs:   jobs,
	}, "finalizer", nil
}

func restrictedPlan() contract.ResponsePlan {
	// No psychological policy, diagnosis, protocol selection, remembered facts,
	// or question is synthesized by the skeleton.
	return contract.ResponsePlan{Question: contract.QuestionPlan{Allowed: false, MaxCount: 0}}
}

// generateValidated implements FB-003/004/005: one bounded regeneration at
// most, then a separately reviewed static fallback which is revalidated.
// The provider stream is never exposed to HTTP or persisted before acceptance.
// A PLAN_UNSAFE sentinel chooses the approved reflect fallback rather than
// executing a plan which the Generator has flagged unsafe.
func (o *Service) generateValidated(ctx context.Context, wc contract.WorkingContext, plan contract.ResponsePlan, objective contract.Objective, spec *contract.ProtocolStepSpec, risk contract.RiskLevel, crisis *contract.CrisisPlan, route contract.Route, locale string, tr ports.TurnTrace) (string, bool, contract.Objective, contract.ValidationReport, bool, *contract.CrisisPlan, error) {
	var regenNote *string
	var lastErr error
	var outageCrisis *contract.CrisisPlan
	providersUnavailable := false
	lastAction := contract.VARegenerate
	maxAttempts := 2
	stageTimeout := o.opts.GeneratorTimeout
	if g, ok := o.deps.Generator.(interface {
		WP11()
		GenerationTimeout() time.Duration
		MaxRegenerations() int
	}); ok {
		maxAttempts = 1 + g.MaxRegenerations()
		if g.GenerationTimeout() < stageTimeout {
			stageTimeout = g.GenerationTimeout()
		}
		pinned, exists := ctx.Value(pinnedBundleKey{}).(pinnedBundle)
		if o.integration == nil || !exists {
			return "", false, objective, contract.ValidationReport{}, false, nil, coreError(contract.ErrBundle, "generator_integration", nil)
		}
		ctx = generator.WithCallMeta(ctx, contract.CallMeta{UserID: pinned.user, TurnID: wc.Turn.TurnID, Bundle: o.bundleReference(ctx), Route: route})
	}
	// ONE stage deadline covers both attempts. Fallback validation retains the
	// outer turn deadline, so an expired provider cannot consume its reserve.
	stageCtx, stageCancel := context.WithTimeout(ctx, stageTimeout)
	defer stageCancel()
	// No generation work can escape the turn deadline. The existing WP-04
	// client independently enforces provider/stream timeouts and no retry
	// after a token is exposed to its own provider stream.
	for attempt := 0; attempt < maxAttempts && stageCtx.Err() == nil; attempt++ {
		started := time.Now()
		genCtx, cancel := context.WithCancel(stageCtx)
		guard := o.deps.Validator.StreamGuard(plan, risk)
		sink := &bufferedSink{guard: guard, guardFactory: func() ports.StreamGuard { return o.deps.Validator.StreamGuard(plan, risk) }}
		out, err := o.deps.Generator.Generate(genCtx, ports.GenInput{
			Turn: wc.Turn, Plan: plan, Objective: objective, Spec: spec, Recent: wc.Recent, Permitted: permittedContext(wc, plan),
			Prefs: wc.Prefs, Risk: risk, Crisis: crisis, Bundle: o.bundleReference(ctx), RegenNote: regenNote,
		}, sink)
		cancel()
		if tr != nil {
			tr.AddUsage(out.Usage)
		}
		if generator.Fatal(err) {
			return "", false, objective, contract.ValidationReport{}, false, nil, err
		}
		text := strings.TrimSpace(out.Text)
		// A structured sentinel has exactly the same escape-hatch semantics as
		// the streamed sentinel. Neither the marker nor its draft is ever saved.
		if out.Sentinel == "PLAN_UNSAFE" {
			lastAction = contract.VACrisis
			if tr != nil {
				tr.Record("validator", trace.Observation{RuleID: "OV-030", Count: 1, Degraded: true})
			}
			break
		}
		if err == nil && (out.Sentinel != "" || out.Aborted || sink.blocked) {
			err = coreError(contract.ErrValidation, "generator", nil)
		}
		if err == nil && sink.used {
			if text == "" {
				text = sink.String()
			} else if text != sink.String() {
				err = coreError(contract.ErrValidation, "generator", nil)
			}
		}
		if err == nil && (text == "" || len(utf16.Encode([]rune(text))) > 20000) {
			err = coreError(contract.ErrValidation, "generator", nil)
		}
		record(tr, "generator", started, err != nil)
		if err != nil {
			lastErr = err
			var providerError *contract.Error
			if errors.As(err, &providerError) && providerError.Code == contract.ErrProvider {
				providersUnavailable = true
			} else if !errors.As(err, &providerError) || providerError.Code != contract.ErrBudget {
				providersUnavailable = false
			}
			if sink.violation != nil && tr != nil {
				tr.Record("validator", trace.Observation{RuleID: sink.violation.Code, Count: 1, Degraded: true})
			}
			if sink.violation != nil && sink.violation.Code == "OV-030" {
				lastAction = contract.VACrisis
				break // escape hatch: do not regenerate the unsafe plan
			}
			if stageCtx.Err() != nil {
				break
			}
			if attempt == 0 {
				note := "OV-031: provide a complete response within the approved plan"
				if sink.violation != nil {
					note = sink.violation.Code + ": follow the approved safety and response plan"
				}
				regenNote = &note
				continue
			}
			break
		}
		started = time.Now()
		providersUnavailable = false
		report, valErr := o.deps.Validator.Finalize(validator.WithFinishReason(stageCtx, out.Usage.FinishReason), text, plan, wc, risk, crisis)
		record(tr, "validator", started, valErr != nil || !report.Passed || report.Action != contract.VAAccept)
		if tr != nil && len(report.Violations) > 0 {
			tr.Record("validator", trace.Observation{RuleID: report.Violations[0].Code, Count: len(report.Violations), Degraded: true})
		}
		if valErr == nil && report.Passed && report.Action == contract.VAAccept {
			// Even successful regeneration is a degraded turn, and is eligible
			// for clinician sampling under the approved release workflow.
			return text, attempt > 0, objective, report, false, nil, nil
		}
		lastErr, lastAction = valErr, report.Action
		if valErr != nil || report.Action == contract.VAFallback || report.Action == contract.VACrisis || attempt != 0 || ctx.Err() != nil {
			break
		}
		// Only rule codes enter the correction note. Do not include a draft,
		// classifier details, context, user data or unreviewed clinical advice.
		codes := make([]string, 0, len(report.Violations))
		for _, v := range report.Violations {
			if strings.HasPrefix(v.Code, "OV-") && len(codes) < 4 {
				codes = append(codes, v.Code)
			}
		}
		if len(codes) == 0 {
			codes = []string{"OV-031"}
		}
		note := strings.Join(codes, ",") + ": revise to satisfy the approved plan and safety rules"
		regenNote = &note
	}
	if err := ctx.Err(); err != nil {
		return "", false, objective, contract.ValidationReport{}, false, nil, err
	}
	// A sentinel asks for rerouting. Use the existing approved reflective
	// fallback on non-crisis turns; crisis keeps its mandatory static script.
	fallbackObjective := objective
	fallbackPlan := plan
	if lastAction == contract.VACrisis && route != contract.RouteCrisis {
		fallbackObjective = contract.ObjReflect
		fallbackPlan = restrictedPlan()
	}
	// FB-006: after all generation providers fail, elevated/follow-up turns
	// use a reviewed crisis script with curated resources. This changes the
	// response route, never the persisted risk level. No approved generic
	// family means fail closed rather than inventing clinical wording.
	if route != contract.RouteCrisis && (risk.Rank() >= contract.RiskElevated.Rank() || wc.Risk.FollowUpActive) && providersUnavailable {
		if g, ok := o.deps.Generator.(interface {
			OutagePlan(context.Context, string) (contract.CrisisPlan, error)
		}); ok {
			cp, e := g.OutagePlan(ctx, locale)
			if e != nil {
				return "", false, objective, contract.ValidationReport{}, false, nil, coreError(contract.ErrSafetyDegraded, "outage_fallback", nil)
			}
			outageCrisis = &cp
			crisis = &cp
			route = contract.RouteCrisis
			fallbackObjective = contract.ObjSafetySupport
			fallbackPlan = restrictedPlan()
			if tr != nil {
				tr.Record("fallback", trace.Observation{RuleID: "FB-006", Count: 1, Degraded: true})
			}
		}
	}
	fallback := ""
	found := false
	if route == contract.RouteCrisis && crisis != nil {
		fallback, found = crisis.TemplateText, true
	} else if route != contract.RouteCrisis && o.opts.Fallback != nil {
		fallback, found = o.opts.Fallback(route, fallbackObjective, locale)
	}
	if !found || strings.TrimSpace(fallback) == "" {
		return "", false, fallbackObjective, contract.ValidationReport{}, false, nil, coreError(contract.ErrGenFailed, "fallback", lastErr)
	}
	fallback = strings.TrimSpace(fallback)
	if len(utf16.Encode([]rune(fallback))) > 20000 {
		return "", false, fallbackObjective, contract.ValidationReport{}, false, nil, coreError(contract.ErrValidation, "fallback", nil)
	}
	started := time.Now()
	report, checkErr := o.deps.Validator.Finalize(ctx, fallback, fallbackPlan, wc, risk, crisis)
	record(tr, "fallback", started, checkErr != nil || !report.Passed || report.Action != contract.VAAccept)
	if tr != nil && len(report.Violations) > 0 {
		tr.Record("fallback", trace.Observation{RuleID: report.Violations[0].Code, Count: len(report.Violations), Degraded: true})
	}
	if checkErr != nil || !report.Passed || report.Action != contract.VAAccept {
		return "", false, fallbackObjective, contract.ValidationReport{}, false, nil, coreError(contract.ErrValidation, "fallback", checkErr)
	}
	return fallback, true, fallbackObjective, report, true, outageCrisis, nil
}

type bufferedSink struct {
	value        strings.Builder
	used         bool
	blocked      bool
	violation    *contract.Violation
	guard        ports.StreamGuard
	guardFactory func() ports.StreamGuard
}

func (s *bufferedSink) Send(token string) error {
	if s.blocked {
		return coreError(contract.ErrValidation, "stream_guard", nil)
	}
	if s.guard != nil {
		ok, v := s.guard.Push(token)
		if !ok {
			s.violation = v
			s.blocked = true
			return coreError(contract.ErrValidation, "stream_guard", nil)
		}
	}
	if s.value.Len()+len(token) > 80000 {
		s.blocked = true
		return coreError(contract.ErrValidation, "stream_size", nil)
	}
	s.used = true
	_, _ = s.value.WriteString(token)
	return nil
}
func (s *bufferedSink) Reset() error {
	// In-process model retries may retract an unexposed buffer. Resetting the
	// validator guard is mandatory: no earlier tokens have escaped to HTTP.
	s.value.Reset()
	s.used = false
	s.blocked = false
	s.violation = nil
	if s.guardFactory != nil {
		s.guard = s.guardFactory()
	}
	return nil
}
func (s *bufferedSink) String() string { return strings.TrimSpace(s.value.String()) }
