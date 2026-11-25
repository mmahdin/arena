package composition

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"time"

	aidb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"
)

// loggingTracer is deliberately metadata-only. No prompt, user, model, or
// conversation identifiers or arbitrary Record payload may enter these logs.
// It is NOT a full execution inspector or a replacement for encrypted traces.
type loggingTracer struct {
	writeTimeout time.Duration
	logger       *slog.Logger
	stages       interface {
		RecordStage(context.Context, aidb.StageEvent) error
	}
}
type loggingTurn struct {
	writeTimeout time.Duration
	logger       *slog.Logger
	stages       interface {
		RecordStage(context.Context, aidb.StageEvent) error
	}
	turnID   c.ID
	ref      string
	started  time.Time
	failed   atomic.Bool
	degraded atomic.Bool
}

var _ ports.Tracer = loggingTracer{}
var _ ports.TurnTrace = (*loggingTurn)(nil)

func (t loggingTracer) Begin(id c.ID, meta c.CallMeta) ports.TurnTrace {
	return t.BeginContext(context.Background(), id, meta)
}
func (t loggingTracer) BeginContext(ctx context.Context, id c.ID, _ c.CallMeta) ports.TurnTrace {
	ref := observability.TurnRef(id)
	logger := t.logger.With(observability.CorrelationFields(ctx)...)
	logger.Info("ai.turn.started", "component", "ai_core", "turn_ref", ref)
	return &loggingTurn{writeTimeout: t.writeTimeout, logger: logger, stages: t.stages, turnID: id, ref: ref, started: time.Now()}
}

// Persist only allowlisted operational metadata. A failed stage write is
// observable in logs and freshness reports; it never leaks the trace payload.
func (t *loggingTurn) persistStage(e aidb.StageEvent) {
	if t.stages == nil || !c.ValidID(t.turnID) {
		return
	}
	e.TurnID = t.turnID
	ctx, cancel := context.WithCancel(context.Background())
	if t.writeTimeout > 0 {
		cancel()
		ctx, cancel = context.WithTimeout(context.Background(), t.writeTimeout)
	}
	defer cancel()
	if err := t.stages.RecordStage(ctx, e); err != nil {
		t.logger.Warn("ai.telemetry.stage_write_failed", "component", "ai_core", "stage", e.Stage)
	}
}

func approvedStep(step string) bool {
	switch step {
	case "jev", "unified", "persistent", "protocol_prepare", "protocol", "unified_validate", "output_resolution", "background", "parallel_dependencies", "policy_gate", "retrieval_hop", "state_validate", "admit", "recent", "safety", "retrieval", "assembler", "route", "brain", "generator", "validator", "fallback", "finalizer", "recovery", "cancel", "bundle", "state_read", "protocol_shortlist", "brain_dependency", "memory_schedule", "memory_correction", "protocol_integration", "protocol_requirements", "protocol_safety", "risk_delta", "protocol_commit", "generator_integration", "brain_integration", "crisis", "proactive", "outage_fallback", "stream_guard", "stream_size", "validator_bundle", "generator_bundle", "protocol_bundle", "reflection_bundle", "memory_bundle", "wakeup_contract", "deadline", "turn", "persistence":
		return true
	default:
		return false
	}
}
func (t *loggingTurn) Record(step string, value any) {
	o, ok := value.(trace.Observation)
	if !ok || !approvedStep(step) {
		return
	}
	if o.Duration < 0 {
		o.Duration = 0
	}
	if o.Count < 0 {
		o.Count = 0
	}
	level := slog.LevelDebug
	event := "ai.stage.completed"
	status := "success"
	if o.Degraded && !o.Started && !o.Skipped {
		level, event, status = slog.LevelWarn, "ai.stage.degraded", "degraded"
		t.degraded.Store(true)
	}
	if o.Terminal {
		level, event, status = slog.LevelError, "ai.stage.failed", "failed"
	}
	if o.Started {
		event, status = "ai.stage.started", "started"
	}
	if o.Skipped {
		event, status = "ai.stage.skipped", "skipped"
	}
	args := []any{"component", "ai_core", "turn_ref", t.ref, "stage", step,
		"status", status, "duration_ms", o.Duration.Milliseconds(), "count", o.Count}
	if trace.ApprovedRule(o.RuleID) {
		args = append(args, "rule_id", o.RuleID)
	}
	if o.Attempt > 0 && o.Attempt <= 20 {
		args = append(args, "attempt", o.Attempt)
	}
	args = append(args, "fallback_used", o.FallbackUsed)
	if o.Operation != "" {
		switch o.Operation {
		case "select", "infer", "assess", "retrieve", "assemble", "route", "llm_inference", "generate", "validate", "finalize", "persist", "load", "policy", "fallback":
			args = append(args, "operation", o.Operation)
		}
	}
	if o.Error != nil {
		args = append(args, observability.DiagnosticFields(o.Error, o.Origin)...)
	} else if o.Degraded || o.Skipped {
		reason := o.Reason
		switch reason {
		case "optional_output_rejected", "jev_unavailable", "jev_invalid_output", "invalid_signal", "dependency_unavailable", "degraded_result", "policy_fallback", "validation_failed", "stream_guard", "schema_invalid", "budget_exceeded", "state_conflict", "fallback_unavailable", "no_provider", "fast_route", "crisis_route":
		default:
			reason = "degraded_result"
		}
		args = append(args, "reason", reason)
		args = append(args, observability.DiagnosticFields(nil, o.Origin)...)
	}
	t.logger.Log(context.Background(), level, event, observability.UniqueFields(args)...)
	if status != "started" {
		e := aidb.StageEvent{Stage: step, Status: status, Duration: o.Duration,
			Count: o.Count, Attempt: o.Attempt, Fallback: o.FallbackUsed}
		if trace.ApprovedRule(o.RuleID) {
			e.RuleID = o.RuleID
		}
		t.persistStage(e)
	}
}
func (t *loggingTurn) AddUsage(c.LLMUsage) {} // no model/provider/token payloads
func (t *loggingTurn) SetRoute(route c.Route) {
	if route.Valid() {
		t.logger.Debug("ai.route", "turn_ref", t.ref, "route", string(route))
	}
}
func (t *loggingTurn) Fail(step string, err error) {
	if !approvedStep(step) {
		step = "unknown"
	}
	if !t.failed.CompareAndSwap(false, true) {
		return
	}
	code := c.ErrInternal
	var ce *c.Error
	if errors.As(err, &ce) && ce != nil && ce.Code.Valid() {
		code = ce.Code
	}
	args := []any{"component", "ai_core", "turn_ref", t.ref, "stage", step, "status", "failed", "error_site", observability.Source(1)}
	if ce == nil || !ce.Code.Valid() {
		args = append(args, "error_code", string(code))
	}
	args = append(args, observability.ErrorFields(err)...)
	t.logger.Error("ai.turn.failed", observability.UniqueFields(args)...)
	persistedStep := step
	if !approvedStep(step) {
		persistedStep = "turn"
	}
	t.persistStage(aidb.StageEvent{Stage: persistedStep, Status: "failed"})
}
func (t *loggingTurn) Finish(_ context.Context) {
	status := "success"
	if t.degraded.Load() {
		status = "degraded"
	}
	if t.failed.Load() {
		status = "failed"
	}
	t.logger.Info("ai.turn.finished", "component", "ai_core", "status", status, "turn_ref", t.ref,
		"duration_ms", time.Since(t.started).Milliseconds(), "failed", t.failed.Load(), "degraded", t.degraded.Load())
	t.persistStage(aidb.StageEvent{Stage: "turn", Status: status, Duration: time.Since(t.started)})
}
