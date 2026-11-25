package letters

import (
	"database/sql"

	httpadapter "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/letters/adapters/http"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/letters/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/letters/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/config"
)

func NewModule(db *sql.DB, cfg config.Config) *httpadapter.Handler {
	var gen application.Generator
	// Generation is unavailable pending its own approved AI Core contract.
	return httpadapter.New(application.New(mssql.New(db), gen))
}
