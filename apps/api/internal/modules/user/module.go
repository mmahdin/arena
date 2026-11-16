package user

import (
	"database/sql"
	httpadapter "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/adapters/http"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/application"
)

func NewProgressModule(db *sql.DB) *httpadapter.ProgressHandler {
	return httpadapter.NewProgressHandler(application.NewProgressService(mssql.NewProgressRepository(db)))
}

func NewModule(db *sql.DB) *httpadapter.Handler {
	return httpadapter.New(application.New(mssql.New(db)))
}

// Privacy settings are metadata only; no conversation sharing occurs here.
func NewPrivacyModule(db *sql.DB) *httpadapter.PrivacyHandler {
	return httpadapter.NewPrivacyHandler(application.NewPrivacyService(mssql.NewPrivacyRepository(db)))
}
