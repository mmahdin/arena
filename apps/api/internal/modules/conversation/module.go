package conversation

import (
	"database/sql"
	httpadapter "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/conversation/adapters/http"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/conversation/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/conversation/application"
)

func NewModule(db *sql.DB, profiles application.ProfileReader, generator application.ReplyGenerator, maxContext int) *httpadapter.Handler {
	return httpadapter.New(application.New(mssql.New(db), profiles, generator, maxContext))
}
