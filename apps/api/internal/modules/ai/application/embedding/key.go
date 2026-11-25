// Package embedding shares only vector computations. Authorization, source
// versions, lexical/vector persistence and retrieval selection remain consumers' work.
package embedding

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"strings"
	"unicode/utf8"
)

var (
	ErrConfig      = errors.New("shared embedding configuration invalid")
	ErrInvalid     = errors.New("shared embedding result invalid")
	ErrCapacity    = errors.New("shared embedding computation capacity reached")
	ErrClosed      = errors.New("shared embedding service stopped")
	ErrInvalidated = errors.New("shared embedding user cache invalidated")
)

type key struct {
	user                 c.ID
	provider, model      string
	dimensions           int
	input, configuration [32]byte
}

func makeKey(req ports.EmbeddingRequest, p ports.EmbeddingProfile) (key, error) {
	if !c.ValidID(req.Meta.UserID) || (req.Tier != ports.ComponentEmbedding && req.Tier != ports.TierEmbedding) || !utf8.ValidString(req.Input) || strings.TrimSpace(req.Input) == "" || len(req.Input) > 128*1024 || len(p.Models) == 0 || len(p.Models) > 8 || p.ExecutionTimeout <= 0 {
		return key{}, ErrConfig
	}
	for _, m := range p.Models {
		if m.Provider == "" || m.Model == "" || m.Dimensions < 1 || m.Dimensions > 8192 {
			return key{}, ErrConfig
		}
	}
	raw, _ := json.Marshal(struct {
		Models   []ports.EmbeddingModel
		Revision string
	}{p.Models, p.Revision})
	m := p.Models[0]
	return key{user: req.Meta.UserID, provider: m.Provider, model: m.Model, dimensions: m.Dimensions, input: sha256.Sum256([]byte(req.Input)), configuration: sha256.Sum256(raw)}, nil
}
