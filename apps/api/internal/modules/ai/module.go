package ai

import (
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mock"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/openai"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/config"
)

func NewModule(cfg config.Config) *application.Service {
	if cfg.LLMProvider == "openai-compatible" {
		return application.New(openai.New(cfg.ModelAPIBaseURL, cfg.ModelAPIKey, cfg.ModelName, cfg.LLMTimeout, cfg.ModelMaxOutputTokens), cfg.MaxContextChars)
	}
	return application.New(mock.Gateway{}, cfg.MaxContextChars)
}
