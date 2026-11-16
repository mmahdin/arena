package reflection

import (
	"database/sql"
	httpadapter "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/reflection/adapters/http"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/reflection/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/reflection/application"
)

func NewModule(db *sql.DB) *httpadapter.Handler {
	return httpadapter.New(application.New(mssql.New(db)))
}
