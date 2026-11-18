package composition

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"time"

	aidb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/brain"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/gateway"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/generator"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/orchestrator"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/protocol"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/reflection"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/scheduler"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/safety"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/validator"
)

// AIInternal is the in-process core. It has no HTTP routes or notification
// transport. Product adapters remain WP-16 and cannot bypass these modules.
type AIInternal struct {
	Turns        *orchestrator.Service
	Memory       *memory.Service
	MemoryWorker *memory.Worker
	Reflection   *reflection.Worker
	Scheduler    *scheduler.Service
}
type AIInternalOptions struct {
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
	ProtocolFacts  orchestrator.ProtocolFacts
	// Explicitly for isolated synthetic fixtures; false in every deployment.
	AllowSyntheticForTests bool
	TestOutputClassifier   validator.Classifier
}

func NewAIInternal(ctx context.Context, opt AIInternalOptions) (*AIInternal, error) {
	if opt.DB == nil || opt.Bundle == nil || opt.Client == nil || opt.Embeddings == nil || opt.UsePolicy == nil || opt.MemoryPolicy == nil || opt.Cipher == nil || opt.WorkerID == "" || opt.Tracer == nil || !opt.AllowSyntheticForTests && (opt.TestOutputClassifier != nil || opt.TestDecorate != nil) {
		return nil, errors.New("AI Core explicit internal dependencies required")
	}
	synthetic := opt.AllowSyntheticForTests
	reader, e := aidb.NewContextReader(opt.DB)
	if e != nil {
		return nil, e
	}
	reg := oneBundle{opt.Bundle}
	ba, e := brain.Load(ctx, opt.Bundle, brain.LoadOptions{AllowSyntheticForTests: synthetic})
	if e != nil {
		return nil, e
	}
	facts, e := aidb.NewBrainFacts(opt.DB)
	if e != nil {
		return nil, e
	}
	bv, e := aidb.NewBrainContextVerifier(opt.DB, opt.UsePolicy)
	if e != nil {
		return nil, e
	}
	br, e := brain.New(opt.Client, ba, reader, brain.Options{Facts: facts, Verifier: bv})
	if e != nil {
		return nil, e
	}
	ga, e := generator.Load(ctx, opt.Bundle, generator.LoadOptions{AllowSyntheticForTests: synthetic})
	if e != nil {
		return nil, e
	}
	gv, e := aidb.NewGeneratorContextVerifier(opt.DB, opt.UsePolicy)
	if e != nil {
		return nil, e
	}
	gen, e := generator.New(opt.Client, ga, generator.Options{Verifier: gv})
	if e != nil {
		return nil, e
	}
	sa, e := safety.Load(ctx, opt.Bundle, safety.LoadOptions{AllowSyntheticForTests: synthetic})
	if e != nil {
		return nil, e
	}
	l1, e := safety.NewLLMClassifier(opt.Client, ba.Schemas(), sa, 800*time.Millisecond)
	if e != nil {
		return nil, e
	}
	classifier := opt.TestOutputClassifier
	if !synthetic {
		classifier, e = validator.NewLLMClassifier(ctx, opt.Client, opt.Bundle, time.Second)
		if e != nil {
			return nil, e
		}
	}
	val, e := validator.New(ctx, opt.Bundle, validator.Options{AllowSyntheticForTests: synthetic, Classifier: classifier})
	if e != nil {
		return nil, e
	}
	pa, e := protocol.Load(ctx, opt.Bundle, protocol.LoadOptions{AllowSyntheticForTests: synthetic})
	if e != nil {
		return nil, e
	}
	protocols, e := protocol.New(pa, reader, time.Now)
	if e != nil {
		return nil, e
	}
	ret, e := NewAIRetrieval(opt.DB, opt.Embeddings, reg, opt.UsePolicy)
	if e != nil {
		return nil, e
	}
	ma, e := memory.Load(ctx, opt.Bundle, memory.LoadOptions{AllowSyntheticForTests: synthetic})
	if e != nil {
		return nil, e
	}
	mr, e := aidb.NewMemoryRepository(opt.DB, opt.Cipher, opt.MemoryPolicy, opt.EmbeddingModel, opt.Dimension, opt.Feedback)
	if e != nil {
		return nil, e
	}
	pol, e := contextstate.LoadPolicy(ctx, reg, c.ID{})
	if e != nil {
		return nil, e
	}
	rr, e := aidb.NewReflectionRepository(mr, pol)
	if e != nil {
		return nil, e
	}
	writer, e := memory.NewWriter(opt.Client, ma, rr, nil)
	if e != nil {
		return nil, e
	}
	committer, e := memory.New(rr, ma, memory.Options{Policy: opt.MemoryPolicy, Cipher: opt.Cipher, Embedder: opt.Embeddings, EmbeddingModel: opt.EmbeddingModel, Dimension: opt.Dimension})
	if e != nil {
		return nil, e
	}
	mw, e := memory.NewWorker(rr, rr, writer, committer, opt.WorkerID+"/memory")
	if e != nil {
		return nil, e
	}
	ra, e := reflection.Load(ctx, opt.Bundle, reflection.LoadOptions{AllowSyntheticForTests: synthetic})
	if e != nil {
		return nil, e
	}
	reflector, e := reflection.NewModel(opt.Client, ra, rr, opt.MemoryPolicy)
	if e != nil {
		return nil, e
	}
	rw, e := reflection.NewWorker(rr, reflector, writer, committer, ra, opt.MemoryPolicy, opt.Cipher, opt.WorkerID+"/reflection")
	if e != nil {
		return nil, e
	}
	timers, e := scheduler.New(rr, opt.Bundle.Reference())
	if e != nil {
		return nil, e
	}
	store := NewAIPersistence(opt.DB)
	gate, e := gateway.New(store, store)
	if e != nil {
		return nil, e
	}
	deps := orchestrator.Dependencies{Gateway: gate, Store: store, Finalizer: store, Recovery: store, Brain: br, Generator: gen, Validator: val, Retriever: ret, Memory: committer, Protocol: protocols, Tracer: opt.Tracer, Reflection: orchestrator.ReflectionPlanner{Bundle: opt.Bundle.Reference()}, ProtocolFacts: opt.ProtocolFacts}
	if opt.TestDecorate != nil {
		deps = opt.TestDecorate(deps)
	}
	turns, e := orchestrator.NewIntegrated(deps, orchestrator.Options{}, orchestrator.IntegratedOptions{Reader: reader, Bundles: reg, Assets: sa, Classifier: l1, AllowSyntheticForTests: synthetic})
	if e != nil {
		return nil, e
	}
	return &AIInternal{turns, committer, mw, rw, timers}, nil
}

// Run owns bounded in-process workers and joins every goroutine on shutdown.
// It intentionally does not consume notification outbox records or call users.
func (core *AIInternal) Run(ctx context.Context) error {
	work, cancel := context.WithCancel(ctx)
	defer cancel()
	var wg sync.WaitGroup
	errs := make(chan error, 3)
	for _, run := range []func(context.Context) error{core.MemoryWorker.Run, core.Reflection.Run, core.Scheduler.Run} {
		wg.Add(1)
		go func(f func(context.Context) error) { defer wg.Done(); e := f(work); errs <- e; cancel() }(run)
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
