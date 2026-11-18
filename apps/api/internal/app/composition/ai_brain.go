package composition

import (
	"context"
	"database/sql"

	aidb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/brain"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// NewAITurnBrain assembles WP-10 for a previously release-verified, pinned
// WP-03 snapshot. No HTTP/runtime cutover, Generator, Memory or Protocol Engine
// is activated. Use this SAME bundle reference with NewIntegrated and WP-09.
func NewAITurnBrain(ctx context.Context, db *sql.DB, client ports.Client, pinned ports.Bundle, use retrieval.UsePolicy, metrics brain.Metrics) (*brain.Service, error) {
	assets, err := brain.Load(ctx, pinned, brain.LoadOptions{})
	if err != nil {
		return nil, err
	}
	reader, err := aidb.NewContextReader(db)
	if err != nil {
		return nil, err
	}
	facts, err := aidb.NewBrainFacts(db)
	if err != nil {
		return nil, err
	}
	verifier, err := aidb.NewBrainContextVerifier(db, use)
	if err != nil {
		return nil, err
	}
	return brain.New(client, assets, reader, brain.Options{Facts: facts, Verifier: verifier, Metrics: metrics})
}
