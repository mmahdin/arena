package bookmark

import (
	"database/sql"
	httpadapter "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/bookmark/adapters/http"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/bookmark/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/bookmark/application"
)

func NewModule(db *sql.DB) *httpadapter.Handler {
	return httpadapter.New(application.New(mssql.New(db)))
}
