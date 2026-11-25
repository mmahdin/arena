package llm

import (
	"context"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"
	"log/slog"
	"time"
)

func (c *Client) decision(ctx context.Context, tier ports.ModelTier, model Model, attempt int, reason string, retry, fallback bool) {
	args := append([]any{"component", "llm", "tier", string(tier), "provider", observability.SafeLabel(model.Provider), "model", observability.SafeLabel(model.Name), "attempt", attempt, "reason", reason, "retry_planned", retry, "fallback_planned", fallback}, observability.CorrelationFields(ctx)...)
	slog.DebugContext(ctx, "llm.chain.decision", args...)
}
func exhausted(ctx context.Context, attempts int, cause error) error {
	if cause == nil {
		cause = observability.Detail(nil, "circuit_open", "", "")
	}
	args := append([]any{"component", "llm", "attempts", attempts, "outcome", "exhausted"}, observability.CorrelationFields(ctx)...)
	args = append(args, observability.ErrorFields(cause)...)
	slog.WarnContext(ctx, "llm.chain.exhausted", args...)
	return errors.Join(ErrUnavailable, cause)
}

func callOutcome(ctx context.Context, operation string, tier ports.ModelTier, started time.Time, err error) {
	if err == nil {
		return
	}
	args := append([]any{"component", "llm", "operation", operation, "tier", string(tier), "duration_ms", time.Since(started).Milliseconds(), "outcome", "failed"}, observability.CorrelationFields(ctx)...)
	args = append(args, observability.DiagnosticFields(localFailure(err), observability.Source(0))...)
	slog.WarnContext(ctx, "llm.call.failed", args...)
}

func localFailure(err error) error {
	if errors.Is(err, ErrQuota) {
		return observability.Detail(err, "local_quota_exhausted", "", "")
	}
	if errors.Is(err, ErrConfig) {
		return observability.Detail(err, "invalid_request", "", "")
	}
	if errors.Is(err, ErrInvalid) {
		return observability.Detail(err, "schema_invalid", "", "")
	}
	if errors.Is(err, ErrUsage) {
		return observability.Detail(err, "usage_invalid", "", "")
	}
	return err
}
