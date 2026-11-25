package ports

import (
	"context"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

type BackgroundResearch struct {
	Request c.BackgroundRequest `json:"request"`
	Items   []c.MemoryView      `json:"items"`
	Meta    []c.RetrievalMeta   `json:"retrieval_meta"`
}
type ResearchPreparer interface {
	PrepareResearch(context.Context, c.ID, c.ID, c.ID, c.BackgroundRequest) (BackgroundResearch, error)
	VerifyResearch(context.Context, c.ID, c.ID, BackgroundResearch) error
}
