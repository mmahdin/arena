package composition

import (
	"database/sql"
	aidb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	convdb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/conversation/adapters/mssql"
	"time"
)

// NewAIPersistence wires the WP-02 SQL unit of work. WP-05 will inject it into
// the new Gateway/Orchestrator; it does not invoke the experimental AI pipeline.
func NewAIPersistence(db *sql.DB, lease ...time.Duration) *convdb.TurnStore {
	var ttl time.Duration
	if len(lease) > 0 {
		ttl = lease[0]
	}
	return convdb.NewTurnStore(db, aidb.NewStateStore(), ttl)
}
