package ports

import (
	"context"
	"time"

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
	TierSmall                  ModelTier      = "small"
	TierStrong                 ModelTier      = "strong"
	TierClassifier             ModelTier      = "classifier"
	TierEmbedding              ModelTier      = "embedding"
)

func (v TurnOutcome) Valid() bool {
	return v == OutcomeCompleted || v == OutcomeFailed || v == OutcomeCancelled || v == OutcomeQueued
}
func (v CorrectionKind) Valid() bool {
	return v == CorrectionContest || v == CorrectionReject || v == CorrectionSupersedePending
}
func (v ModelTier) Valid() bool {
	return v == TierSmall || v == TierStrong || v == TierClassifier || v == TierEmbedding
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

// RouterConfig contains policy inputs, not an executable expression language.
// Ordered predicate interpretation belongs to WP-05 and bundle loading to WP-03.
type RouterConfig struct {
	Rules             []RouterRule
	TrivialPatterns   map[string][]string
	TrivialMaxChars   int
	ShortlistMaxItems int
	DeclineCooldown   time.Duration
}
type RouterRule struct {
	ID         string
	Predicates []Predicate
	Decision   RouteDecision
}
type Predicate struct {
	Name  string
	Value string
}
type PolicyConfig struct {
	MaxConsecutiveQuestions, MaxQuestionsPerReply            int
	VoiceRequiresStatus                                      contract.MemoryStatus
	VoiceMinGap                                              time.Duration
	VoiceMaxPerSession, OfferMinTurnGap, MaxOffersPerSession int
}
type RouteOverride struct {
	Route               contract.Route
	Reason              string // rule code only
	ForceBufferedOutput bool
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
