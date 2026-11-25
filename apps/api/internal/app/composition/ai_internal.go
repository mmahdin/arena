package composition

import (
	"context"
	"database/sql"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"
	"log/slog"
	"sync"
	"time"

	aidb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/background"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/embedding"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/gateway"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/knowledge"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/orchestrator"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/protocol"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/reflection"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/scheduler"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/unified"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// AIInternal is the in-process core. It has no HTTP routes or notification
// transport. Product adapters remain WP-16 and cannot bypass these modules.
type AIInternal struct {
	Budgets      bundle.Budgets
	Embeddings   *embedding.Service
	Inspector    *trace.Inspector
	Turns        *orchestrator.Service
	Memory       *memory.Service
	Intelligence *background.Service
	MemoryWorker *memory.Worker
	Reflection   *reflection.Worker
	Scheduler    *scheduler.Service
	Pulse        WorkerPulse
	InstanceID   c.ID
}

// WorkerPulse contains liveness only, never job payload or model output.
type WorkerPulse interface {
	Heartbeat(context.Context, c.ID, string, string) error
}
type AIInternalOptions struct {
	Budgets                                 bundle.Budgets
	EmbeddingCache                          embedding.Config
	EmbeddingProfile                        ports.EmbeddingProfile // Explicit identity for substituted test clients.
	JEVTimeout                              time.Duration
	UnifiedTimeout                          time.Duration
	TurnTimeout                             time.Duration
	UnifiedInputTokens, UnifiedOutputTokens int
	Inspector                               *trace.Inspector
	AdmissionLimit                          int

	DisableProactive bool // Product boundary never enables wake-ups

	TestDecorate   func(orchestrator.Dependencies) orchestrator.Dependencies // isolated WP-15 instrumentation only
	DB             *sql.DB
	Bundle         ports.Bundle
	Client         ports.Client
	Embeddings     ports.EmbeddingClient
	UsePolicy      retrieval.UsePolicy
	MemoryPolicy   memory.Policy
	Cipher         memory.Cipher
	EmbeddingModel string
	Dimension      int
	WorkerID       string
	Feedback       memory.FeedbackPolicy
	Tracer         ports.Tracer
	WorkerPulse    WorkerPulse
	// Explicit for isolated synthetic testing and guarded local drafts; false in staging/production.
	AllowSyntheticForTests bool
}

// bootstrapFailure logs an explicit constructor stage while preserving the
// original error for callers. Never log err.Error() or private parameters.
func bootstrapFailure(stage string, err error) error {
	slog.Log(context.Background(), slog.LevelError, "ai.bootstrap.failed",
		append([]any{"stage", stage}, observability.ErrorFields(err)...)...)
	return err
}

func NewAIInternal(ctx context.Context, opt AIInternalOptions) (*AIInternal, error) {
	if opt.AdmissionLimit < 0 || opt.DB == nil || opt.Bundle == nil || opt.Client == nil || opt.Embeddings == nil || opt.UsePolicy == nil || opt.MemoryPolicy == nil || opt.Cipher == nil || opt.WorkerID == "" || opt.Tracer == nil || !opt.AllowSyntheticForTests && opt.TestDecorate != nil {
		return nil, errors.New("AI Core explicit internal dependencies required")
	}
	profile := opt.EmbeddingProfile
	if len(profile.Models) == 0 && opt.AllowSyntheticForTests {
		profile = ports.EmbeddingProfile{Models: []ports.EmbeddingModel{{Provider: "synthetic", Model: opt.EmbeddingModel, Dimensions: opt.Dimension}}, ExecutionTimeout: opt.EmbeddingProfile.ExecutionTimeout}
	}
	shared, e := embedding.New(ctx, opt.Embeddings, embedding.Options{Config: opt.EmbeddingCache, Profile: profile})
	if e != nil {
		return nil, bootstrapFailure("shared_embedding", e)
	}
	initialized := false
	defer func() {
		if !initialized {
			shared.Close()
		}
	}()
	opt.Embeddings = shared
	synthetic := opt.AllowSyntheticForTests
	reader, e := aidb.NewContextReader(opt.DB)
	if e != nil {
		return nil, bootstrapFailure("context_reader", e)
	}
	reg := oneBundle{opt.Bundle}
	verifier, e := aidb.NewUnifiedContextVerifier(opt.DB, opt.UsePolicy, opt.Cipher, time.Duration(opt.Budgets.Operations.HintVerificationMS)*time.Millisecond)
	if e != nil {
		return nil, bootstrapFailure("unified_context_verifier", e)
	}
	main, e := unified.New(ctx, opt.Client, opt.Bundle, verifier, unified.Options{AllowSyntheticForTests: synthetic, Timeout: opt.UnifiedTimeout, MaxInputTokens: opt.UnifiedInputTokens, MaxOutputTokens: opt.UnifiedOutputTokens, MaxContentTokens: opt.Budgets.Tokens.KnowledgeContent})
	if e != nil {
		return nil, bootstrapFailure("unified_model", e)
	}
	selector, e := knowledge.New(ctx, opt.Client, opt.Bundle, knowledge.Options{Timeout: opt.JEVTimeout, Verifier: verifier, MaxInputTokens: opt.Budgets.Tokens.JEVIn, MaxContentTokens: opt.Budgets.Tokens.KnowledgeContent})
	if e != nil {
		return nil, bootstrapFailure("jev_selector", e)
	}
	pa, e := protocol.Load(ctx, opt.Bundle, protocol.LoadOptions{AllowSyntheticForTests: synthetic})
	if e != nil {
		return nil, bootstrapFailure("protocol_assets", e)
	}
	protocols, e := protocol.New(pa, reader, time.Now)
	if e != nil {
		return nil, bootstrapFailure("protocol_service", e)
	}
	ret, e := NewAIRetrieval(opt.DB, opt.Embeddings, reg, opt.UsePolicy, opt.Cipher)
	if e != nil {
		return nil, bootstrapFailure("retrieval", e)
	}
	ma, e := memory.Load(ctx, opt.Bundle, memory.LoadOptions{AllowSyntheticForTests: synthetic})
	if e != nil {
		return nil, bootstrapFailure("memory_assets", e)
	}
	mr, e := aidb.NewMemoryRepository(opt.DB, opt.Cipher, opt.MemoryPolicy, opt.EmbeddingModel, opt.Dimension, opt.Feedback)
	if e != nil {
		return nil, bootstrapFailure("memory_repository", e)
	}
	mr.WithSearchPolicy(opt.UsePolicy)
	pol, e := contextstate.LoadPolicy(ctx, reg, c.ID{})
	if e != nil {
		return nil, bootstrapFailure("context_policy", e)
	}
	rr, e := aidb.NewReflectionRepository(mr, pol)
	if e != nil {
		return nil, bootstrapFailure("reflection_repository", e)
	}
	committer, e := memory.New(rr, ma, memory.Options{Policy: opt.MemoryPolicy, Cipher: opt.Cipher, Embedder: opt.Embeddings, EmbeddingModel: opt.EmbeddingModel, Dimension: opt.Dimension, EmbeddingInvalidator: shared})
	if e != nil {
		return nil, bootstrapFailure("memory_committer", e)
	}
	intelligence, e := background.New(ctx, aidb.NewIntelligenceRepository(mr), mr, opt.Cipher, opt.Client, opt.Bundle, ret, opt.MemoryPolicy)
	if e != nil {
		return nil, bootstrapFailure("background_intelligence", e)
	}
	mw, e := memory.NewIntelligenceWorker(rr, rr, committer, intelligence, opt.WorkerID+"/memory")
	if e != nil {
		return nil, bootstrapFailure("memory_worker", e)
	}
	ra, e := reflection.Load(ctx, opt.Bundle, reflection.LoadOptions{AllowSyntheticForTests: synthetic})
	if e != nil {
		return nil, bootstrapFailure("reflection_assets", e)
	}
	rw, e := reflection.NewIntelligenceWorker(rr, ra, intelligence, opt.WorkerID+"/reflection")
	if e != nil {
		return nil, bootstrapFailure("reflection_worker", e)
	}
	timers, e := scheduler.NewWithOptions(rr, opt.Bundle.Reference(), scheduler.RunOptions{DisableDispatch: opt.DisableProactive, Interval: time.Duration(opt.Budgets.Operations.SchedulerIntervalMS) * time.Millisecond, RetryBase: time.Duration(opt.Budgets.Operations.SchedulerRetryBaseMS) * time.Millisecond, RetryMax: time.Duration(opt.Budgets.Operations.SchedulerRetryMaxMS) * time.Millisecond, MaxConsecutiveFailures: opt.Budgets.Operations.SchedulerMaxFailures})
	if e != nil {
		return nil, bootstrapFailure("scheduler", e)
	}
	store := NewAIPersistence(opt.DB, time.Duration(opt.Budgets.Operations.TurnLeaseMS)*time.Millisecond)
	if opt.AdmissionLimit > 0 {
		store = store.WithAdmissionLimit(opt.AdmissionLimit)
	}
	gate, e := gateway.New(store, store)
	if e != nil {
		return nil, bootstrapFailure("gateway", e)
	}
	reader.WithBackgroundHints(opt.Cipher, time.Duration(opt.Budgets.Operations.HintReadMS)*time.Millisecond)
	deps := orchestrator.Dependencies{Knowledge: selector, Unified: main, UnifiedMemory: background.MemoryPlanner{Service: intelligence}, Inspector: opt.Inspector, Gateway: gate, Store: store, Finalizer: store, Recovery: store, Retriever: ret, Protocol: protocols, Tracer: opt.Tracer, UnifiedReflection: background.ReflectionPlanner{Service: intelligence}}
	if opt.TestDecorate != nil {
		deps = opt.TestDecorate(deps)
	}
	turns, e := orchestrator.NewUnified(deps, orchestrator.Options{RenewInterval: time.Duration(opt.Budgets.Operations.TurnRenewMS) * time.Millisecond, TotalTimeout: opt.TurnTimeout, CleanupTimeout: time.Duration(opt.Budgets.Operations.TurnCleanupMS) * time.Millisecond, RecentLimit: opt.Budgets.Operations.RecentMessages, MaxLLMCalls: opt.Budgets.Limits.MaxSyncLLMCalls, ContextBudgets: opt.Budgets.Context}, orchestrator.UnifiedOptions{Reader: reader, Bundles: reg, Persistent: newPersistentProfileReader(ret, opt.DB), AllowSyntheticForTests: synthetic})
	if e != nil {
		return nil, bootstrapFailure("orchestrator", e)
	}
	initialized = true
	return &AIInternal{Budgets: opt.Budgets, Embeddings: shared, Intelligence: intelligence, Turns: turns, Memory: committer, MemoryWorker: mw, Reflection: rw, Scheduler: timers, Inspector: opt.Inspector, Pulse: opt.WorkerPulse, InstanceID: c.NewID()}, nil
}

// Run owns bounded in-process workers and joins every goroutine on shutdown.
// It intentionally does not consume notification outbox records or call users.
func (core *AIInternal) Run(ctx context.Context) error {
	if core.Embeddings != nil {
		defer core.Embeddings.Close()
	}
	ctx = trace.WithTimeouts(ctx, trace.Timeouts{ReferenceWrite: time.Duration(core.Budgets.Operations.TraceReferenceMS) * time.Millisecond, JobRefresh: time.Duration(core.Budgets.Operations.TraceJobsMS) * time.Millisecond})
	work, cancel := context.WithCancel(trace.WithInspector(ctx, core.Inspector))
	defer cancel()
	var wg sync.WaitGroup
	errs := make(chan error, 3)
	workers := []struct {
		name string
		run  func(context.Context) error
	}{
		{"memory", core.MemoryWorker.Run},
		{"reflection", core.Reflection.Run},
		{"scheduler", core.Scheduler.Run},
	}
	for _, worker := range workers {
		wg.Add(1)
		go func(name string, run func(context.Context) error) {
			defer wg.Done()
			terminal := "stopped"
			if core.Pulse != nil && c.ValidID(core.InstanceID) {
				finished, stopped := make(chan struct{}), make(chan struct{})
				go func() {
					defer close(stopped)
					ticker := time.NewTicker(time.Duration(core.Budgets.Operations.HeartbeatIntervalMS) * time.Millisecond)
					defer ticker.Stop()
					core.writeHeartbeat(name, "running")
					for {
						select {
						case <-finished:
							return
						case <-ticker.C:
							core.writeHeartbeat(name, "running")
						}
					}
				}()
				// Stop and join the heartbeat loop BEFORE recording terminal
				// state, so a late tick cannot resurrect a stopped worker.
				defer func() { close(finished); <-stopped; core.writeHeartbeat(name, terminal) }()
			}
			// Panics in a goroutine would otherwise bypass Gin recovery and
			// terminate the process without a structured diagnostic.
			defer func() {
				if value := recover(); value != nil {
					terminal = "failed"
					slog.ErrorContext(work, "ai.worker.panic", "worker", name,
						"panic_type", observability.Type(value), "stack", observability.SafeStack(1))
					errs <- errors.New("AI worker panic")
					cancel()
				}
			}()
			slog.InfoContext(work, "ai.worker.started", "worker", name)
			err := run(work)
			if err != nil && !errors.Is(err, context.Canceled) && work.Err() == nil {
				terminal = "failed"
				args := append([]any{"worker", name}, observability.ErrorFields(err)...)
				if name == "scheduler" {
					h := core.Scheduler.Health()
					args = append(args, "scheduler_stage", h.LastFailureStage,
						"scheduler_code", string(h.LastFailureCode), "consecutive_failures", h.ConsecutiveFailures)
				}
				slog.Log(work, slog.LevelError, "ai.worker.failed", args...)
			}
			errs <- err
			cancel()
		}(worker.name, worker.run)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil && !errors.Is(e, context.Canceled) {
			return e
		}
	}
	return ctx.Err()
}

// Worker heartbeats reflect host process liveness, not job throughput. Error
// details stay in sanitized logs and the existing durable ai_jobs table.
func (core *AIInternal) writeHeartbeat(name, status string) {
	if core.Pulse == nil || !c.ValidID(core.InstanceID) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(core.Budgets.Operations.HeartbeatMS)*time.Millisecond)
	defer cancel()
	if err := core.Pulse.Heartbeat(ctx, core.InstanceID, name, status); err != nil {
		slog.Warn("ai.telemetry.heartbeat_write_failed", "worker", name)
	}
}

// oneBundle pins all synchronous/background consumers to the same immutable release.
type oneBundle struct{ ports.Bundle }

func (b oneBundle) Active(context.Context, c.ID) (ports.Bundle, error) { return b.Bundle, nil }
