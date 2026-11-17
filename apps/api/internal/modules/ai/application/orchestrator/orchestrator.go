// Package orchestrator executes WP-05's deterministic, non-streaming turn DAG.
// It deliberately delegates all psychological reasoning, safety, validation,
// retrieval and generation to injectable ports: none is implemented here.
package orchestrator

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
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
	Gateway   ports.Gateway
	Store     ports.ConversationStore
	Finalizer ports.TurnFinalizer
	Recovery  ports.TurnRecovery // optional: explicit retry, cancellation and heartbeat
	Router    ports.Router
	Safety    ports.SafetyGate
	Assembler ports.ContextAssembler
	Retriever ports.Retriever // absent until WP-09; memory is unavailable
	Brain     ports.Brain
	Generator ports.Generator
	Validator ports.Validator
	Crisis    ports.CrisisPlanner // required only for a crisis route
	Tracer    ports.Tracer
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

	turnCtx, cancel := context.WithTimeout(ctx, o.opts.TotalTimeout)
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
		ret, err := o.deps.Retriever.Retrieve(ctx, contract.RetrievalRequest{UserID: a.UserID, Turn: turn, MaxItems: 5})
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
		// WP-02 rejects wake-up admission; a future WP-14-approved synthetic
		// pathway must explicitly enable it. No unconsented proactive output.
		return contract.PreparedTurn{}, "route", coreError(contract.ErrValidation, "proactive", nil)
	}

	objective := contract.ObjReflect
	plan := restrictedPlan()
	degraded := safe.signal.Degraded || retr.result.Meta.Degraded
	var crisis *contract.CrisisPlan
	var banner *contract.SafetyBanner
	var vettedBrain *contract.BrainDecision
	switch decision.Route {
	case contract.RouteFast:
		objective = contract.ObjAnswerDirect
	case contract.RouteNormal:
		started = time.Now()
		brainCtx, cancel := context.WithTimeout(ctx, o.opts.BrainTimeout)
		output, e := o.deps.Brain.Decide(brainCtx, ports.BrainInput{WCtx: wc, Safety: safe.signal, Bundle: o.bundleReference(ctx)})
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
	reply, wasFallback, finalObjective, err := o.generateValidated(ctx, wc, plan, objective, safe.signal.Level, crisis, decision.Route, req.Locale, tr)
	objective = finalObjective
	if err != nil {
		return contract.PreparedTurn{}, "validator", err
	}
	degraded = degraded || wasFallback
	delta := contract.StateDelta{TurnID: a.TurnID, UserID: a.UserID, ExpectedVersions: a.Versions,
		Session: contract.SessionPatch{LastObjective: objective, TurnCountInc: 1}}
	if o.integration != nil {
		delta.ExpectedVersions = stateSnapshot.Versions
		delta.Session.ConsecutiveQuestions = wc.Session.ConsecutiveQuestions
		// Only conservative, evidence-bearing updates for the current focus
		// thread are admitted. No new thread, protocol, memory, or hypothesis
		// is invented while WP-07/WP-10/WP-13 remain unimplemented.
		if vettedBrain != nil && decision.Route == contract.RouteNormal && riskRank(safe.signal.Level) < riskRank(contract.RiskElevated) {
			o.integrateBrainDelta(&delta, *vettedBrain, wc, stateSnapshot)
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
	return contract.PreparedTurn{
		TurnID: a.TurnID, UserMessageID: a.UserMessageID, Text: reply,
		Route: decision.Route, Objective: objective, Degraded: degraded,
		State:  delta,
		Trace:  contract.TurnTraceSummary{TurnID: a.TurnID, BundleVersion: o.bundleReference(ctx).Version, Route: decision.Route, Objective: objective},
		Safety: banner,
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
func (o *Service) generateValidated(ctx context.Context, wc contract.WorkingContext, plan contract.ResponsePlan, objective contract.Objective, risk contract.RiskLevel, crisis *contract.CrisisPlan, route contract.Route, locale string, tr ports.TurnTrace) (string, bool, contract.Objective, error) {
	var regenNote *string
	var lastErr error
	lastAction := contract.VARegenerate
	// No generation work can escape the turn deadline. The existing WP-04
	// client independently enforces provider/stream timeouts and no retry
	// after a token is exposed to its own provider stream.
	for attempt := 0; attempt < 2 && ctx.Err() == nil; attempt++ {
		started := time.Now()
		genCtx, cancel := context.WithTimeout(ctx, o.opts.GeneratorTimeout)
		guard := o.deps.Validator.StreamGuard(plan, risk)
		sink := &bufferedSink{guard: guard, guardFactory: func() ports.StreamGuard { return o.deps.Validator.StreamGuard(plan, risk) }}
		out, err := o.deps.Generator.Generate(genCtx, ports.GenInput{
			Turn: wc.Turn, Plan: plan, Objective: objective, Recent: wc.Recent, Permitted: nil,
			Prefs: wc.Prefs, Risk: risk, Crisis: crisis, Bundle: o.bundleReference(ctx), RegenNote: regenNote,
		}, sink)
		cancel()
		if tr != nil {
			tr.AddUsage(out.Usage)
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
		if out.Sentinel != "" || out.Aborted || sink.blocked {
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
			if sink.violation != nil && tr != nil {
				tr.Record("validator", trace.Observation{RuleID: sink.violation.Code, Count: 1, Degraded: true})
			}
			if sink.violation != nil && sink.violation.Code == "OV-030" {
				lastAction = contract.VACrisis
				break // escape hatch: do not regenerate the unsafe plan
			}
			if errors.Is(ctx.Err(), context.Canceled) || ctx.Err() != nil {
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
		report, valErr := o.deps.Validator.Finalize(validator.WithFinishReason(ctx, out.Usage.FinishReason), text, plan, wc, risk, crisis)
		record(tr, "validator", started, valErr != nil || !report.Passed || report.Action != contract.VAAccept)
		if tr != nil && len(report.Violations) > 0 {
			tr.Record("validator", trace.Observation{RuleID: report.Violations[0].Code, Count: len(report.Violations), Degraded: true})
		}
		if valErr == nil && report.Passed && report.Action == contract.VAAccept {
			// Even successful regeneration is a degraded turn, and is eligible
			// for clinician sampling under the approved release workflow.
			return text, attempt > 0, objective, nil
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
		return "", false, objective, err
	}
	// A sentinel asks for rerouting. Use the existing approved reflective
	// fallback on non-crisis turns; crisis keeps its mandatory static script.
	fallbackObjective := objective
	fallbackPlan := plan
	if lastAction == contract.VACrisis && route != contract.RouteCrisis {
		fallbackObjective = contract.ObjReflect
		fallbackPlan = restrictedPlan()
	}
	fallback := ""
	found := false
	if route == contract.RouteCrisis && crisis != nil {
		fallback, found = crisis.TemplateText, true
	} else if route != contract.RouteCrisis && o.opts.Fallback != nil {
		fallback, found = o.opts.Fallback(route, fallbackObjective, locale)
	}
	if !found || strings.TrimSpace(fallback) == "" {
		return "", false, fallbackObjective, coreError(contract.ErrGenFailed, "fallback", lastErr)
	}
	fallback = strings.TrimSpace(fallback)
	if len(utf16.Encode([]rune(fallback))) > 20000 {
		return "", false, fallbackObjective, coreError(contract.ErrValidation, "fallback", nil)
	}
	started := time.Now()
	report, checkErr := o.deps.Validator.Finalize(ctx, fallback, fallbackPlan, wc, risk, crisis)
	record(tr, "fallback", started, checkErr != nil || !report.Passed || report.Action != contract.VAAccept)
	if tr != nil && len(report.Violations) > 0 {
		tr.Record("fallback", trace.Observation{RuleID: report.Violations[0].Code, Count: len(report.Violations), Degraded: true})
	}
	if checkErr != nil || !report.Passed || report.Action != contract.VAAccept {
		return "", false, fallbackObjective, coreError(contract.ErrValidation, "fallback", checkErr)
	}
	return fallback, true, fallbackObjective, nil
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
