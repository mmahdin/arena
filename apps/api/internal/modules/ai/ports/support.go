package ports

import (
	"context"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

const (
	OutcomeCompleted           TurnOutcome    = "completed"
	OutcomeFailed              TurnOutcome    = "failed"
	OutcomeCancelled           TurnOutcome    = "cancelled"
	OutcomeQueued              TurnOutcome    = "queued"
	CorrectionContest          CorrectionKind = "contest"
	CorrectionReject           CorrectionKind = "reject"
	CorrectionSupersedePending CorrectionKind = "supersede_pending"
	ComponentUnified           ModelTier      = "unified"
	ComponentJEV               ModelTier      = "jev"
	ComponentSafety            ModelTier      = "safety"
	ComponentRouter            ModelTier      = "router"
	ComponentBrain             ModelTier      = "brain"
	ComponentGenerator         ModelTier      = "generator"
	ComponentValidator         ModelTier      = "validator"
	ComponentMemoryWriter      ModelTier      = "memory_writer"
	ComponentReflection        ModelTier      = "reflection"
	ComponentEmbedding         ModelTier      = "embedding"
	// Legacy tier identifiers remain only for isolated tests and compatibility
	// with older in-process clients. Runtime v2 never accepts them.
	TierSmall      ModelTier = "small"
	TierStrong     ModelTier = "strong"
	TierClassifier ModelTier = "classifier"
	TierEmbedding  ModelTier = "embedding"
)

func (v TurnOutcome) Valid() bool {
	return v == OutcomeCompleted || v == OutcomeFailed || v == OutcomeCancelled || v == OutcomeQueued
}
func (v CorrectionKind) Valid() bool {
	return v == CorrectionContest || v == CorrectionReject || v == CorrectionSupersedePending
}
func (v ModelTier) Valid() bool {
	return v == ComponentJEV || v == ComponentUnified || v == ComponentSafety || v == ComponentBrain || v == ComponentGenerator ||
		v == ComponentValidator || v == ComponentRouter || v == ComponentMemoryWriter || v == ComponentReflection ||
		v == ComponentEmbedding || v == TierSmall || v == TierStrong || v == TierClassifier
}

// Bundle is an immutable, pinned content handle. Implementations must return
// copies; loading, review checks, and activation belong to WP-03.
type Bundle interface {
	Reference() contract.BundleRef
	Artifact(ctx context.Context, name string) ([]byte, error)
}

// These reports contain identifiers and rule codes, never source content.
type ReconcileReport struct {
	Repaired   bool
	Versions   contract.Versions
	Rejections []contract.DeltaRejection
}
type MemoryApplyReport struct {
	Applied  []contract.ID
	Rejected []contract.DeltaRejection
}

// DeletionReport counts records newly hidden by an explicit soft-delete scope.
// No content is erased and derived records or AI consent are not changed implicitly.
type DeletionReport struct {
	DeletedItems, DeletedMessages, DeletedEmbeddings, DeletedTraces int
	AffectedThreads                                                 []contract.ID
}
type MaintenanceReport struct {
	Staled, Merged, Suppressed int
}

type Block struct {
	Text  string
	Cache bool
	Name  string
}
type ChatMessage struct {
	Role    contract.Role
	Content string
}

// Recv returns io.EOF after the final chunk. Usage may be present only on the
// final chunk. The stream retains the context supplied to Client.Stream;
// cancellation must unblock Recv, and Close must release provider resources.
type Chunk struct {
	Text         string
	Usage        *contract.LLMUsage
	FinishReason string
}
type LLMUsage = contract.LLMUsage
