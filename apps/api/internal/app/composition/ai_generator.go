package composition

import (
	"context"
	"database/sql"

	aidb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/generator"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// NewAIResponseGenerator composes WP-11 with the SAME release-verified bundle
// used by WP-06/07/09/10 and NewIntegrated. It does not activate the legacy HTTP
// path, an SSE endpoint, a Protocol Engine or a Memory Writer.
func NewAIResponseGenerator(ctx context.Context, db *sql.DB, client ports.Client, pinned ports.Bundle, use retrieval.UsePolicy) (*generator.Service, error) {
	a, err := generator.Load(ctx, pinned, generator.LoadOptions{})
	if err != nil {
		return nil, err
	}
	v, err := aidb.NewGeneratorContextVerifier(db, use)
	if err != nil {
		return nil, err
	}
	return generator.New(client, a, generator.Options{Verifier: v})
}
