package mssql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
)

// TelemetryStore implements durable, metadata-only writes. Do not attach user
// messages or the local Inspector to these DTOs. It is NOT a quota enforcer.
type TelemetryStore struct{ db *sql.DB }

func NewTelemetryStore(db *sql.DB) (*TelemetryStore, error) {
	if db == nil {
		return nil, errors.New("telemetry SQL unavailable")
	}
	return &TelemetryStore{db: db}, nil
}

func (r *TelemetryStore) BeginAttempt(ctx context.Context, e llm.AttemptStart) error {
	if !contract.ValidID(e.ID) || e.StartedAt.IsZero() || e.Kind != "structured" && e.Kind != "stream" && e.Kind != "embed" ||
		!e.Tier.Valid() || !safeTelemetryLabel(e.Provider) || !safeTelemetryLabel(e.Model) ||
		e.InputRate < 0 || e.OutputRate < 0 || e.CachedRate < 0 {
		return errors.New("invalid attempt metadata")
	}
	var turn any
	if contract.ValidID(e.TurnID) {
		turn = e.TurnID.String()
	}
	_, err := r.db.ExecContext(ctx, `INSERT dbo.ai_llm_attempts
      (id,turn_id,kind,tier,provider,model,started_at,input_rate_micro_usd_per_m,
       output_rate_micro_usd_per_m,cached_rate_micro_usd_per_m)
      VALUES(CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),@p3,@p4,@p5,@p6,@p7,@p8,@p9,@p10)`,
		e.ID.String(), turn, e.Kind, string(e.Tier), e.Provider, e.Model, e.StartedAt.UTC(),
		e.InputRate, e.OutputRate, e.CachedRate)
	return err
}

func (r *TelemetryStore) FinishAttempt(ctx context.Context, e llm.AttemptFinish) error {
	if !contract.ValidID(e.ID) || e.FinishedAt.IsZero() || e.Duration < 0 ||
		e.Status != "succeeded" && e.Status != "failed" && e.Status != "incomplete" ||
		e.CostStatus != "unknown" && e.CostStatus != "unpriced" && e.CostStatus != "estimated" ||
		e.CostStatus == "estimated" && (!e.UsageKnown || e.CostMicroUSD < 0) ||
		!e.UsageKnown && e.CostStatus != "unknown" ||
		!safeFailure(e.Failure) {
		return errors.New("invalid attempt result")
	}
	var in, out, cached, cost, failure any
	if e.UsageKnown {
		if e.InTokens < 0 || e.OutTokens < 0 || e.CachedTokens < 0 || e.CachedTokens > e.InTokens {
			return errors.New("invalid attempt counters")
		}
		in, out, cached = e.InTokens, e.OutTokens, e.CachedTokens
	}
	if e.CostStatus == "estimated" {
		cost = e.CostMicroUSD
	}
	if e.Failure != "" {
		failure = string(e.Failure)
	}
	// State change is guarded so replays/concurrent completion cannot charge an
	// attempt twice. A second Finish is a no-op if a terminal row already exists.
	result, err := r.db.ExecContext(ctx, `UPDATE dbo.ai_llm_attempts SET
      finished_at=@p2,status=@p3,failure_kind=@p4,duration_ms=@p5,
      input_tokens=@p6,output_tokens=@p7,cached_tokens=@p8,cost_micro_usd=@p9,cost_status=@p10
      WHERE id=CONVERT(uniqueidentifier,@p1) AND status='started'`,
		e.ID.String(), e.FinishedAt.UTC(), e.Status, failure, e.Duration.Milliseconds(),
		in, out, cached, cost, e.CostStatus)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 1 {
		return nil
	}
	// An already-terminal row is idempotent. Unknown attempt IDs are errors.
	var status string
	err = r.db.QueryRowContext(ctx, `SELECT status FROM dbo.ai_llm_attempts WHERE id=CONVERT(uniqueidentifier,@p1)`, e.ID.String()).Scan(&status)
	if err != nil {
		return err
	}
	if status != e.Status {
		return errors.New("conflicting attempt completion")
	}
	return nil
}

func safeFailure(k llm.FailureKind) bool {
	switch k {
	case "", llm.FailureUnknown, llm.FailureTimeout, llm.FailureCanceled, llm.FailureTransport,
		llm.FailureServer, llm.FailureRateLimit, llm.FailurePermanent, llm.FailureAuth, llm.FailureInvalid:
		return true
	}
	return false
}

func safeTelemetryLabel(s string) bool {
	if len(s) == 0 || len(s) > 80 {
		return false
	}
	for _, c := range s {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == '-' || c == ':' {
			continue
		}
		return false
	}
	return true
}

type StageEvent struct {
	TurnID         contract.ID
	Stage, Status  string
	Duration       time.Duration // Only measured complete events; zero may be unavailable.
	Count, Attempt int
	Fallback       bool
	RuleID         string
}

func (r *TelemetryStore) RecordStage(ctx context.Context, e StageEvent) error {
	if !contract.ValidID(e.TurnID) || !validStage(e.Stage) ||
		e.Status != "success" && e.Status != "failed" && e.Status != "degraded" && e.Status != "skipped" ||
		e.Duration < 0 || e.Count < 0 || e.Attempt < 0 || e.Attempt > 20 ||
		e.RuleID != "" && !trace.ApprovedRule(e.RuleID) {
		return errors.New("invalid stage metadata")
	}
	var duration, attempt, count, rule any
	if e.Duration > 0 {
		duration = e.Duration.Milliseconds()
	}
	if e.Attempt > 0 {
		attempt = e.Attempt
	}
	if e.Count > 0 {
		count = e.Count
	}
	if e.RuleID != "" {
		rule = e.RuleID
	}
	_, err := r.db.ExecContext(ctx, `INSERT dbo.ai_stage_events
      (id,turn_id,stage,status,duration_ms,attempt,event_count,fallback_used,rule_id)
      VALUES(CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),@p3,@p4,@p5,@p6,@p7,@p8,@p9)`,
		contract.NewID().String(), e.TurnID.String(), e.Stage, e.Status,
		duration, attempt, count, e.Fallback, rule)
	return err
}

// A strict allowlist prevents arbitrary trace step names being stored as labels.
func validStage(s string) bool {
	switch s {
	case "jev", "output_resolution", "background", "unified", "persistent", "protocol_prepare", "protocol", "unified_validate", "parallel_dependencies", "policy_gate", "retrieval_hop", "state_validate", "admit", "recent", "safety", "retrieval", "assembler", "route", "brain", "generator", "validator", "fallback", "finalizer", "recovery", "cancel", "bundle", "state_read", "protocol_shortlist", "brain_dependency", "memory_schedule", "memory_correction", "protocol_integration", "protocol_requirements", "protocol_safety", "risk_delta", "protocol_commit", "generator_integration", "brain_integration", "crisis", "proactive", "outage_fallback", "stream_guard", "stream_size", "validator_bundle", "generator_bundle", "protocol_bundle", "reflection_bundle", "memory_bundle", "wakeup_contract", "deadline", "turn", "persistence":
		return true
	}
	return false
}

// Heartbeat stores ONLY host liveness. Freshness, not the latest status alone,
// determines whether a worker can be considered live in BO-05/06 dashboards.
func (r *TelemetryStore) Heartbeat(ctx context.Context, instanceID contract.ID, kind, status string) error {
	if !contract.ValidID(instanceID) || kind != "memory" && kind != "reflection" && kind != "scheduler" ||
		status != "running" && status != "stopped" && status != "failed" {
		return errors.New("invalid heartbeat")
	}
	// Every runtime instance ID is random and each worker has a single writer.
	// Lock the key range for inserts; a duplicate-key failure must be retried
	// by the caller at the next heartbeat rather than losing an old state.
	_, err := r.db.ExecContext(ctx, `UPDATE dbo.ai_worker_heartbeats WITH (UPDLOCK,SERIALIZABLE)
       SET status=@p3,last_seen_at=SYSUTCDATETIME()
       WHERE instance_id=CONVERT(uniqueidentifier,@p1) AND worker_kind=@p2;
       IF @@ROWCOUNT = 0
       INSERT dbo.ai_worker_heartbeats(instance_id,worker_kind,status,last_seen_at)
       VALUES(CONVERT(uniqueidentifier,@p1),@p2,@p3,SYSUTCDATETIME());`, instanceID.String(), kind, status)
	return err
}

var _ llm.AttemptJournal = (*TelemetryStore)(nil)
