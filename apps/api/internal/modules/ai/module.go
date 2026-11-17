package ai

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/embed"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mock"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/openai"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	aidomain "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/domain"
	domainRetrieval "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/domain/retrieval"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/config"
)

func NewModule(cfg config.Config) *application.Service {
	if cfg.LLMProvider == "openai-compatible" {
		return application.New(openai.New(cfg.ModelAPIBaseURL, cfg.ModelAPIKey, cfg.ModelName, cfg.LLMTimeout, cfg.ModelMaxOutputTokens), cfg.MaxContextChars)
	}
	return application.New(mock.Gateway{}, cfg.MaxContextChars)
}

// NewRetrievalModule wires the WP-09 Retrieval Service: the SQL Server read
// model, the embedding tier and the deterministic domain rules.
//
// The offline deterministic embedder is the default so no billable network call
// is ever made implicitly. A provider-backed embedder must be requested through
// configuration and carries the same key/URL validation as the LLM client.
func NewRetrievalModule(cfg config.Config, db *sql.DB) (*retrieval.Service, error) {
	if db == nil {
		return nil, errors.New("retrieval requires the shared SQL Server pool")
	}
	var embedder retrieval.Embedder
	switch cfg.EmbeddingProvider {
	case "openai-compatible":
		embedder = embed.NewOpenAICompatible(cfg.ModelAPIBaseURL, cfg.ModelAPIKey, cfg.EmbeddingModel, cfg.EmbeddingDimensions, 5*time.Second)
	default:
		embedder = embed.NewDeterministic(cfg.EmbeddingDimensions)
	}
	return retrieval.New(retrieval.Config{
		Store:              mssql.New(db),
		Embedder:           embedder,
		Tunables:           domainRetrieval.Defaults(),
		Location:           aidomain.BusinessLocation(cfg.BusinessTimezone),
		WeekStartsOnMonday: true,
	})
}

// Retrieve exposes the WP-09 service for the orchestrator (WP-05/WP-10) and for
// composition-level smoke tests. A nil database yields a service whose store
// fails closed with a degraded, empty result rather than panicking.
func Retrieve(svc *retrieval.Service, ctx context.Context, req contract.RetrievalRequest) (contract.RetrievalResult, error) {
	if svc == nil {
		return contract.RetrievalResult{Meta: contract.RetrievalMeta{Degraded: true, Strategies: []string{}}}, retrieval.ErrNoStore
	}
	return svc.Retrieve(ctx, req)
}
