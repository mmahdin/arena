package ports

import (
	"context"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

type UnifiedInput struct {
	Knowledge []c.KnowledgeResource `json:"knowledge"`
	UserID    c.ID
	Context   c.WorkingContext
	Bundle    c.BundleRef
}
type UnifiedOutput struct {
	Turn  c.UnifiedTurn
	Usage c.LLMUsage
}
type UnifiedModel interface {
	Infer(context.Context, UnifiedInput) (UnifiedOutput, error)
	Reference() c.BundleRef
}
