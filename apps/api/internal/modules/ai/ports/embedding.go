package ports

import (
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"time"
)

// EmbeddingModel identifies a compatible vector space. Provider IDs refer to
// immutable configured adapters/endpoints for the lifetime of one process.
type EmbeddingModel struct {
	Provider   string
	Model      string
	Dimensions int
}

// EmbeddingProfile is resolved by the existing configured client BEFORE lookup.
// Models preserves the actual fallback order; Revision distinguishes effective
// adapter configurations not already represented by model/provider/dimensions.
// Caller wait limits are intentionally separate from ExecutionTimeout.
type EmbeddingProfile struct {
	Models           []EmbeddingModel
	Revision         string
	ExecutionTimeout time.Duration
}

type EmbeddingProfileResolver interface {
	EmbeddingProfile(ModelTier) (EmbeddingProfile, error)
}

// EmbeddingInvalidator clears process-local optimizations, never durable data.
type EmbeddingInvalidator interface{ InvalidateEmbeddingUser(userID c.ID) }
