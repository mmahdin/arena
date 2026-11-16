package devsession

import (
	"database/sql"

	httpadapter "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/devsession/adapters/http"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/devsession/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/devsession/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/devsecurity"
)

type Module struct {
	Handler  *httpadapter.Handler
	Sessions devsecurity.SessionStore
}

func NewModule(db *sql.DB, signer *devsecurity.Signer, identitySecret string) *Module {
	repo := mssql.New(db, identitySecret)
	return &Module{Handler: httpadapter.New(application.New(repo), signer), Sessions: repo}
}
