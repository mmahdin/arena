// Package orchestrator owns durable admission, the unified turn pipeline and atomic finalization.
package orchestrator

import (
	"context"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"
	"golang.org/x/text/unicode/norm"
	"log/slog"
	"time"
)

type Dependencies struct {
	Knowledge         KnowledgeSelector
	Unified           ports.UnifiedModel
	UnifiedMemory     UnifiedMemory
	UnifiedReflection UnifiedReflection
	Inspector         *trace.Inspector
	Gateway           ports.Gateway
	Store             ports.ConversationStore
	Finalizer         ports.TurnFinalizer
	Recovery          ports.TurnRecovery
	Assembler         ports.ContextAssembler
	Retriever         ports.Retriever
	Protocol          ports.ProtocolEngine
	Tracer            ports.Tracer
}
type Options struct {
	ContextBudgets                              bundle.ContextTokenBudgets
	RecentLimit                                 int
	TotalTimeout, RenewInterval, CleanupTimeout time.Duration
	MaxLLMCalls                                 int
}
type Service struct {
	deps        Dependencies
	opts        Options
	integration *integration
}

var _ ports.Orchestrator = (*Service)(nil)

func newService(d Dependencies, opt Options) (*Service, error) {
	if d.Gateway == nil || d.Store == nil || d.Finalizer == nil || d.Assembler == nil || d.Unified == nil || d.Knowledge == nil {
		return nil, errors.New("unified turn dependencies required")
	}
	if opt.RecentLimit < 1 || opt.TotalTimeout <= 0 || opt.CleanupTimeout <= 0 || opt.MaxLLMCalls < 1 || opt.RenewInterval < 0 || opt.RenewInterval > 0 && d.Recovery == nil {
		return nil, errors.New("invalid unified turn budgets")
	}
	return &Service{deps: d, opts: opt}, nil
}

// ProcessTurn does NOT expose any draft token. Admission happens before all
// model work; one finalizer transaction makes the assistant message visible.
// Replaying a completed/failed/cancelled key never calls the model again.
func (o *Service) ProcessTurn(ctx context.Context, req contract.TurnRequest) (result contract.TurnResult, resultErr error) {
	ctx, inspectionSpan := o.deps.Inspector.StartTurn(ctx, req)
	defer func() { inspectionSpan.Complete(func() any { return result }, resultErr, recover()) }()

	// Wake-up events are internal outbox metadata, not user-authored messages.
	// Until a reviewed synthetic-turn admission/finalization contract exists,
	// reject BEFORE Gateway.Admit: a substituted gateway must not fabricate a
	// user message or lease an ordinary turn for a proactive event.
	if req.Origin == contract.OriginWakeUp || req.WakeUp != nil {
		return contract.TurnResult{}, coreError(contract.ErrForbidden, "wakeup_contract", nil)
	}
	req.Text = norm.NFC.String(req.Text)
	admitCtx, admitSpan := trace.Start(ctx, "gateway.Admit", func() any { return req })
	trace.Bind(ctx, trace.Ref(ctx, "/input/request/text"), admitSpan.Ref("/input/text"), "transform")
	a, err := o.deps.Gateway.Admit(admitCtx, req)
	admitSpan.End(func() any { return a }, err)
	trace.Associate(ctx, a.TurnID)
	if err != nil {
		slog.Log(ctx, slog.LevelError, "ai.turn.admission_failed",
			append(append([]any{"stage", "admit"}, observability.CorrelationFields(ctx)...), observability.DiagnosticFields(err, observability.Source(0))...)...)
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
	// Persist a bounded metadata-only reference to this actual OTel root.
	// Failures never change lease admission, provider calls or finalization.
	trace.RecordAdmittedTurn(ctx, a.TurnID)

	turnTimeout := o.opts.TotalTimeout
	deadline := time.Now().Add(turnTimeout)
	if b, ok := contract.BudgetFromContext(ctx); ok && !b.TotalDeadline.IsZero() && b.TotalDeadline.Before(deadline) {
		deadline = b.TotalDeadline
	}
	turnCtx, cancel := context.WithDeadline(observability.WithTurnRef(ctx, observability.TurnRef(a.TurnID)), deadline)
	defer cancel()
	var tr ports.TurnTrace
	if o.deps.Tracer != nil {
		meta := contract.CallMeta{TurnID: a.TurnID, UserID: a.UserID}
		meta.Bundle = o.deps.Unified.Reference()
		if contextual, ok := o.deps.Tracer.(interface {
			BeginContext(context.Context, contract.ID, contract.CallMeta) ports.TurnTrace
		}); ok {
			tr = contextual.BeginContext(ctx, a.TurnID, meta)
		} else {
			tr = o.deps.Tracer.Begin(a.TurnID, meta)
		}
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
			stageStarted(tr, "finalizer", "finalize")
			finalCtx, finalSpan := trace.Start(turnCtx, "finalizer.Complete", func() any { return map[string]any{"admission": a, "prepared": p} })
			trace.Bind(ctx, admitSpan.Ref("/output"), finalSpan.Ref("/input/admission"), "copy")
			trace.Bind(ctx, trace.ChildRef(turnCtx, "application/orchestrator/orchestrator.prepare", "/output/preparedResult"), finalSpan.Ref("/input/prepared"), "copy")
			result, completeErr := o.deps.Finalizer.Complete(finalCtx, a, p)
			finalSpan.End(func() any { return result }, completeErr)
			recordDiagnostic(tr, "finalizer", "finalize", started, completeErr, 0, false)
			if completeErr == nil {
				trace.Bind(ctx, finalSpan.Ref("/output"), inspectionSpan.Ref("/output"), "copy")
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
		tr.Fail(stage, runErr)
	}
	return o.failTurn(observability.WithTurnRef(ctx, observability.TurnRef(a.TurnID)), a, req.ClientTurnID, runErr)
}

func (o *Service) failTurn(ctx context.Context, a contract.Admission, client contract.ID, reason error) (result0 contract.TurnResult, result1 error) {
	ctx, inspectionSpan := trace.Start(ctx, "application/orchestrator/orchestrator.failTurn", func() any { return map[string]any{"a": a, "client": client, "reason": trace.ErrorInfo(reason)} })
	defer func() {
		inspectionSpan.Complete(func() any { return map[string]any{"result0": result0} }, result1, recover())
	}()

	code := failureCode(reason)
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), o.opts.CleanupTimeout)
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
		slog.Log(ctx, slog.LevelError, "ai.turn.cleanup_failed",
			append([]any{"stage", "recovery", "turn_ref", observability.TurnRef(a.TurnID)}, observability.ErrorFields(cleanupErr)...)...)
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
	return &contract.Error{Code: code, Step: step, Cause: cause, Origin: observability.Source(1), Retryable: code == contract.ErrStateCommit || code == contract.ErrStateConflict || code == contract.ErrProvider}
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
		// MarshalJSON/ToPublic sanitize display. Keep diagnostics across the
		// internal HTTP boundary; the adapter maps this wrapper's code first.
		return &turnFailure{core: &contract.Error{Code: e.Code, Step: e.Step, Retryable: e.Retryable, Cause: err, Origin: e.Origin}}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &turnFailure{core: &contract.Error{Code: contract.ErrBudget, Step: "deadline", Cause: err}}
	}
	return &turnFailure{core: &contract.Error{Code: contract.ErrInternal, Step: "turn", Cause: err}}
}
func stageStarted(tr ports.TurnTrace, stage, operation string) {
	if tr != nil {
		tr.Record(stage, trace.Observation{Started: true, Operation: operation})
	}
}
func errorStage(err error, fallback string) string {
	var stage interface{ DiagnosticStage() string }
	if errors.As(err, &stage) && stage.DiagnosticStage() != "" {
		return stage.DiagnosticStage()
	}
	return fallback
}
func record(tr ports.TurnTrace, name string, at time.Time, degraded bool) {
	if tr != nil {
		tr.Record(name, trace.Observation{Duration: time.Since(at), Degraded: degraded})
	}
}

// recordDiagnostic marks the precise stage boundary where a failure was
// observed, retaining the original typed cause for safe metadata extraction.
// Never pass message bodies or generated text to the tracer.
func recordDiagnostic(tr ports.TurnTrace, stage, operation string, started time.Time, err error, attempt int, fallback bool) {
	if tr == nil {
		return
	}
	terminal := err != nil && (stage == "unified" || stage == "state_validate" || stage == "finalizer" || stage == "recent" || stage == "assembler" || stage == "state_read")
	tr.Record(stage, trace.Observation{Duration: time.Since(started), Degraded: err != nil, Terminal: terminal,
		Error: err, Operation: operation, Origin: observability.Source(1),
		Attempt: attempt, FallbackUsed: fallback})
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

func (o *Service) prepare(ctx context.Context, req contract.TurnRequest, a contract.Admission, tr ports.TurnTrace) (preparedResult contract.PreparedTurn, failedStage string, prepareErr error) {
	ctx, span := trace.Start(ctx, "application/orchestrator/orchestrator.prepare", func() any { return map[string]any{"request": req, "admission": a} })
	defer func() {
		span.Complete(func() any { return map[string]any{"preparedResult": preparedResult, "failedStage": failedStage} }, prepareErr, recover())
	}()
	return o.prepareUnified(ctx, req, a, tr)
}

type turnFailure struct{ core *contract.Error }

func (e *turnFailure) Error() string                { return e.core.Error() }
func (e *turnFailure) Unwrap() error                { return e.core }
func (e *turnFailure) MarshalJSON() ([]byte, error) { return e.core.MarshalJSON() }

func (*turnFailure) PreservePublicCode() bool { return true }
