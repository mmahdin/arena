package composition

import (
	"context"
	"database/sql"

	aidb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type AIMemory struct {
	Writer    *memory.Writer
	Committer *memory.Service
	Worker    *memory.Worker
}

// NewAIMemory requires a release-verified WP-03 snapshot, WP-04 client,
// explicit reviewed consent/use policy and deployment-managed encryption keys.
// The caller starts Worker.Run under graceful shutdown and injects Committer
// into the integrated orchestrator. No legacy AI or HTTP cutover is activated.
func NewAIMemory(ctx context.Context, db *sql.DB, client ports.Client, embed ports.EmbeddingClient, pinned ports.Bundle, policy memory.Policy, cipher memory.Cipher, model string, dimension int, workerID string, metrics memory.Metrics, feedback memory.FeedbackPolicy) (*AIMemory, error) {
	assets, e := memory.Load(ctx, pinned, memory.LoadOptions{})
	if e != nil {
		return nil, e
	}
	repo, e := aidb.NewMemoryRepository(db, cipher, policy, model, dimension, feedback)
	if e != nil {
		return nil, e
	}
	writer, e := memory.NewWriter(client, assets, repo, metrics)
	if e != nil {
		return nil, e
	}
	committer, e := memory.New(repo, assets, memory.Options{Policy: policy, Cipher: cipher, Embedder: embed, EmbeddingModel: model, Dimension: dimension, Metrics: metrics})
	if e != nil {
		return nil, e
	}
	worker, e := memory.NewWorker(repo, repo, writer, committer, workerID)
	if e != nil {
		return nil, e
	}
	return &AIMemory{writer, committer, worker}, nil
}
