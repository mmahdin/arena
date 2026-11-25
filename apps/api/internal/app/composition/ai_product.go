package composition

import (
	"context"
	admindomain "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/scheduler"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"net/http"
	"sync/atomic"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/telemetry"
)

// ProductOptions is explicit injection for host composition and isolated tests.
// Callers must construct AIInternal with DisableProactive=true; the product host
// additionally runs a maintenance-only scheduler regardless of test options.
type ProductOptions struct {
	Core            *AIInternal
	BundleInventory admindomain.BundleInventory
	ConsentKeys     []string
	// Deprecated: accepted for compatibility; never gates AI chat.
	ChatConsentKey string
}

// API owns worker lifetime; cmd/api starts Run once and joins it before DB close.
type API struct {
	http.Handler
	core           *AIInternal
	otel           *telemetry.Runtime
	traceRefWriter trace.ReferenceWriter
	running        atomic.Bool
	healthy        atomic.Bool
}

func (a *API) Run(ctx context.Context) error {
	if !a.running.CompareAndSwap(false, true) {
		return ErrRuntime
	}
	if a.core == nil {
		<-ctx.Done()
		return ctx.Err()
	}
	a.healthy.Store(true)
	defer a.healthy.Store(false)
	return a.core.Run(trace.WithReferenceWriter(telemetry.WithRuntime(ctx, a.otel), a.traceRefWriter))
}

// NullTracer retains no process-local user data. The finalizer still persists
// safe trace metadata transactionally. Full trace replay remains unavailable.
type nullTracer struct{}
type nullTrace struct{}

func (nullTracer) Begin(c.ID, c.CallMeta) ports.TurnTrace { return nullTrace{} }
func (nullTrace) Record(string, any)                      {}
func (nullTrace) AddUsage(c.LLMUsage)                     {}
func (nullTrace) SetRoute(c.Route)                        {}
func (nullTrace) Fail(string, error)                      {}
func (nullTrace) Finish(context.Context)                  {}

// WorkerHealth is safe host telemetry; it contains no owner IDs or payloads.
func (a *API) WorkerHealth() scheduler.Health {
	if a.core == nil {
		return scheduler.Health{}
	}
	return a.core.Scheduler.Health()
}
