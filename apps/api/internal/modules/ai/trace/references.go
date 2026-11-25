package trace

import (
	"context"
	"time"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/telemetry"
)

// RootReference is a metadata-only link to the actual W3C OpenTelemetry root.
// It must never contain the private local Inspector's graph UUID or disk path.
type RootReference struct {
	TurnID     c.ID
	Kind       string // turn, memory, reflection
	JobID      c.ID
	JobAttempt int
	TraceID    string
	RootSpanID string
}

// ReferenceWriter is best-effort; failures never change AI inference or jobs.
type ReferenceWriter interface {
	WriteTraceReference(context.Context, RootReference) error
}
type referenceWriterKey struct{}
type timeoutsKey struct{}
type Timeouts struct{ ReferenceWrite, JobRefresh time.Duration }

func WithTimeouts(ctx context.Context, limits Timeouts) context.Context {
	return context.WithValue(ctx, timeoutsKey{}, limits)
}
func timeoutPolicy(ctx context.Context) Timeouts {
	limits, _ := ctx.Value(timeoutsKey{}).(Timeouts)
	return limits
}

func WithReferenceWriter(ctx context.Context, writer ReferenceWriter) context.Context {
	if ctx == nil || writer == nil {
		return ctx
	}
	return context.WithValue(ctx, referenceWriterKey{}, writer)
}

// A short independent write prevents an unavailable SQL journal from changing
// the AI result. Dropped mappings remain visibly absent, never fabricated.
func recordRoot(ctx context.Context, r RootReference) {
	if ctx == nil || !telemetry.Enabled(ctx) || !c.ValidID(r.TurnID) {
		return
	}
	writer, _ := ctx.Value(referenceWriterKey{}).(ReferenceWriter)
	if writer == nil {
		return
	}
	traceID, rootSpanID, ok := telemetry.ActiveRootIDs(ctx)
	if !ok {
		return
	}
	r.TraceID, r.RootSpanID = traceID, rootSpanID
	limit := timeoutPolicy(ctx).ReferenceWrite
	if limit <= 0 {
		return
	}
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), limit)
	defer cancel()
	_ = writer.WriteTraceReference(writeCtx, r) // never log potentially private DB errors
}

// RecordAdmittedTurn is invoked only after successful lease-owner admission;
// duplicates, invalid admissions and queued turns must not create trace refs.
func RecordAdmittedTurn(ctx context.Context, turn c.ID) {
	recordRoot(ctx, RootReference{TurnID: turn, Kind: "turn"})
}

// RecordClaimedJob is invoked after a real durable worker claim and root start.
func RecordClaimedJob(ctx context.Context, turn, job c.ID, kind string, attempt int) {
	if !c.ValidID(job) || (kind != "memory" && kind != "reflection") || attempt < 1 || attempt > 1000000 {
		return
	}
	recordRoot(ctx, RootReference{TurnID: turn, JobID: job, Kind: kind, JobAttempt: attempt})
}
