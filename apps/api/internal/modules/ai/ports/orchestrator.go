package ports

import (
	"context"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// Main AI Core entrypoint; no RPC or second deployment.
type Orchestrator interface {
	ProcessTurn(ctx context.Context, req contract.TurnRequest) (contract.TurnResult, error)
}
