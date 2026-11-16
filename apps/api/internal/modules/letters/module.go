package letters

import (
	"database/sql"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/openai"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/letters/adapters/generator"
	httpadapter "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/letters/adapters/http"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/letters/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/letters/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/config"
)

func NewModule(db *sql.DB, cfg config.Config) *httpadapter.Handler {
	var gen application.Generator
	// No synthetic 'personalized' letters from the mock provider.
	if cfg.LLMProvider == "openai-compatible" {
		gen = generator.New(openai.New(cfg.ModelAPIBaseURL, cfg.ModelAPIKey, cfg.ModelName, cfg.LLMTimeout, cfg.ModelMaxOutputTokens))
	}
	return httpadapter.New(application.New(mssql.New(db), gen))
}
