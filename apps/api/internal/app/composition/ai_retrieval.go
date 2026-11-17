package composition

import (
	"database/sql"
	"errors"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/embedding"
	aidb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// RetrievalOptions are the WP-09 integration inputs. They are explicit so that
// a missing Knowledge Bundle registry, model client or database fails at
// construction rather than silently disabling recall at runtime.
type RetrievalOptions struct {
	DB *sql.DB
	// Bundles must be the SAME registry that pins WP-03/WP-08 per turn.
	Bundles ports.BundleRegistry
	// Embedding is optional: without it retrieval runs on temporal, thread,
	// entity, lexical and recency signals and reports no similarity.
	Embedding ports.EmbeddingClient
	// EmbeddingModel is the storage identity (model id + dimensions). It must
	// match the vectors already stored in dbo.ai_embeddings.
	EmbeddingModel embedding.ModelID
}

// NewRetriever wires the WP-09 SQL Server read adapter and, when an embedding
// client is supplied, the WP-04-backed embedding adapter.
//
// This composes existing contracts only: the returned value satisfies WP-01's
// ports.Retriever, which WP-05's orchestrator already consumes (an absent
// retriever is treated as degraded memory, so wiring this is additive and no
// turn-path behavior is replaced here).
//
// Runtime activation is intentionally NOT performed by this function. A caller
// must construct it in the composition root and inject it as
// orchestrator Dependencies.Retriever. Do not enable the synthetic WP-03 bundle
// for real traffic; see the WP-03 registry's allowSynthetic gate.
func NewRetriever(opts RetrievalOptions) (ports.Retriever, error) {
	if opts.DB == nil {
		return nil, errors.New("WP-09 retrieval requires the shared SQL Server pool")
	}
	if opts.Bundles == nil {
		return nil, errors.New("WP-09 retrieval requires the active Knowledge Bundle registry")
	}
	store, err := aidb.NewRetrievalStore(opts.DB)
	if err != nil {
		return nil, err
	}
	var embedder retrieval.Embedder
	if opts.Embedding != nil {
		if opts.EmbeddingModel.Model == "" || opts.EmbeddingModel.Dimensions < 1 {
			return nil, errors.New("WP-09 embedding model id and dimensions are required when a client is configured")
		}
		adapter, aerr := embedding.New(embedding.Options{
			Client: opts.Embedding,
			Model:  opts.EmbeddingModel,
		})
		if aerr != nil {
			return nil, aerr
		}
		embedder = adapter
	}
	return retrieval.New(store, opts.Bundles, embedder)
}
