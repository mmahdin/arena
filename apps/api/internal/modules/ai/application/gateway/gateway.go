// Package gateway is the in-process WP-05 admission boundary. The Conversation
// store owns the one durable message/turn record; its WP-02 admission transaction
// also obtains the fenced, per-user SQL lease across all conversations.
package gateway

import (
	"context"
	"errors"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type Service struct {
	store    ports.ConversationStore
	recovery ports.TurnRecovery
}

func New(store ports.ConversationStore, recovery ports.TurnRecovery) (*Service, error) {
	if store == nil {
		return nil, errors.New("gateway requires a durable ConversationStore")
	}
	return &Service{store: store, recovery: recovery}, nil
}

var _ ports.Gateway = (*Service)(nil)

func (g *Service) Admit(ctx context.Context, req contract.TurnRequest) (contract.Admission, error) {
	if err := ctx.Err(); err != nil {
		return contract.Admission{}, err
	}
	// Authorization uses the authenticated UserID supplied by the composition
	// boundary. The store repeats the owner/visibility check inside its txn.
	if err := g.store.ValidateOwner(ctx, req.UserID, req.ConversationID); err != nil {
		return contract.Admission{}, err
	}
	return g.store.AdmitUserMessage(ctx, req)
}

// Complete/Fail release their lease ATOMICALLY in the WP-02 SQL transaction,
// not here. A second explicit Release would be an erroneous stale-token write.
// Cancellation is a separate durable status transition (never a DELETE).
func (g *Service) Release(ctx context.Context, a contract.Admission, outcome ports.TurnOutcome) error {
	if !outcome.Valid() {
		return &contract.Error{Code: contract.ErrValidation, Step: "gateway.release"}
	}
	if outcome != ports.OutcomeCancelled {
		return nil
	}
	if g.recovery == nil {
		return &contract.Error{Code: contract.ErrInternal, Step: "gateway.cancel", Retryable: true}
	}
	return g.recovery.Cancel(ctx, a)
}
