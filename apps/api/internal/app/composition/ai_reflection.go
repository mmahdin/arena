package composition

import (
	"context"
	"database/sql"

	aidb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/reflection"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/scheduler"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type AIReflection struct {
	Worker    *reflection.Worker
	Scheduler *scheduler.Service
}

// NewAIReflection builds internal WP-14 workers. The host explicitly starts Run
// under cancellation and joins both goroutines on shutdown. It never activates
// HTTP endpoints, notifications, or the experimental daily-reflection feature.
func NewAIReflection(ctx context.Context, db *sql.DB, client ports.Client, embed ports.EmbeddingClient, pinned ports.Bundle, policy memory.Policy, cipher memory.Cipher, model string, dimension int, workerID string, metrics memory.Metrics, feedback memory.FeedbackPolicy) (*AIReflection, error) {
	a, e := reflection.Load(ctx, pinned, reflection.LoadOptions{})
	if e != nil {
		return nil, e
	}
	ma, e := memory.Load(ctx, pinned, memory.LoadOptions{})
	if e != nil {
		return nil, e
	}
	pol, e := contextstate.LoadPolicy(ctx, oneBundle{pinned}, [16]byte{})
	if e != nil {
		return nil, e
	}
	mr, e := aidb.NewMemoryRepository(db, cipher, policy, model, dimension, feedback)
	if e != nil {
		return nil, e
	}
	repo, e := aidb.NewReflectionRepository(mr, pol)
	if e != nil {
		return nil, e
	}
	writer, e := memory.NewWriter(client, ma, repo, metrics)
	if e != nil {
		return nil, e
	}
	committer, e := memory.New(repo, ma, memory.Options{Policy: policy, Cipher: cipher, Embedder: embed, EmbeddingModel: model, Dimension: dimension, Metrics: metrics})
	if e != nil {
		return nil, e
	}
	reflector, e := reflection.NewModel(client, a, repo, policy)
	if e != nil {
		return nil, e
	}
	worker, e := reflection.NewWorker(repo, reflector, writer, committer, a, policy, cipher, workerID)
	if e != nil {
		return nil, e
	}
	timers, e := scheduler.New(repo, a.Reference())
	if e != nil {
		return nil, e
	}
	return &AIReflection{worker, timers}, nil
}

type oneBundle struct{ ports.Bundle }

func (b oneBundle) Active(context.Context, c.ID) (ports.Bundle, error) { return b.Bundle, nil }
