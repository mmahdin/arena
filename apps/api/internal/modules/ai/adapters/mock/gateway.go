package mock

import (
	"context"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application"
)

// Only enable the mock adapter for synthetic tests in APP_ENV=local.
type Gateway struct{}

func (Gateway) Generate(_ context.Context, _ []application.ChatMessage) (string, error) {
	return "شنیدم. دوست داری بیشتر از چیزی که امروز درگیرت کرده برام بگی؟ (پاسخ آزمایشی Mock)", nil
}
