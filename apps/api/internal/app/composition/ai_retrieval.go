package composition

import (
	"database/sql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"

	aidb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// NewAIRetrieval assembles WP-09 for explicit Orchestrator injection. The
// caller MUST supply an approved AI-use policy; WP-02's visibility and consent
// storage do not specify one. No synthetic, permissive, or production-default
// AI-use grant is introduced by this composition helper.
func NewAIRetrieval(db *sql.DB, embeddings ports.EmbeddingClient, bundles ports.BundleRegistry, use retrieval.UsePolicy, cipher memory.Cipher) (*retrieval.Service, error) {
	store, err := aidb.NewRetrievalRepository(db, use, cipher)
	if err != nil {
		return nil, err
	}
	return retrieval.New(store, embeddings, bundles, use)
}
