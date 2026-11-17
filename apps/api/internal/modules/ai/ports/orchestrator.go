package ports

import (
	"context"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// Main AI Core entrypoint; no RPC or second deployment.
type Orchestrator interface {
	ProcessTurn(ctx context.Context, req contract.TurnRequest) (contract.TurnResult, error)
}
type Router interface {
	Select(in RouteInput) RouteDecision
}
type RouteInput struct {
	Safety contract.SafetySignal
	WCtx   contract.WorkingContext
	Origin contract.TurnOrigin
	Config RouterConfig
}
type RouteDecision struct {
	Route               contract.Route
	Reason              string
	BrainShortlist      bool
	RiskConstraints     bool
	ForceBufferedOutput bool
}
type PolicyGate interface {
	Gate(in GateInput) (contract.BrainDecision, []contract.DeltaRejection, *RouteOverride)
}
type GateInput struct {
	Decision contract.BrainDecision
	WCtx     contract.WorkingContext
	Safety   contract.SafetySignal
	Config   PolicyConfig
}
