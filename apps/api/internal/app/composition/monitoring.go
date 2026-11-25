package composition

import (
	"context"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/metrics"
)

// meteredAttemptJournal wraps, never replaces, the existing SQL attempt
// accounting. Only a successfully committed terminal attempt increments a
// best-effort operational counter. No IDs, models or user fields are labels.
type meteredAttemptJournal struct {
	next  llm.AttemptJournal
	meter *metrics.Runtime
}

func (j meteredAttemptJournal) BeginAttempt(ctx context.Context, a llm.AttemptStart) error {
	return j.next.BeginAttempt(ctx, a)
}

func (j meteredAttemptJournal) FinishAttempt(ctx context.Context, a llm.AttemptFinish) error {
	if err := j.next.FinishAttempt(ctx, a); err != nil {
		return err
	}
	j.meter.ObserveLLM(a.Status, a.CostStatus, a.Duration, a.UsageKnown,
		a.InTokens, a.OutTokens, a.CachedTokens, a.CostMicroUSD)
	return nil
}

// Heartbeat gauges update ONLY after SQL acknowledged the write. Missing
// heartbeats are detected through the freshness timestamp, not assumed healthy.
type meteredWorkerPulse struct {
	next  WorkerPulse
	meter *metrics.Runtime
}

func (p meteredWorkerPulse) Heartbeat(ctx context.Context, instance contract.ID, kind, status string) error {
	if p.next == nil {
		return nil
	}
	if err := p.next.Heartbeat(ctx, instance, kind, status); err != nil {
		return err
	}
	p.meter.WorkerHeartbeat(kind, status, time.Now())
	return nil
}
