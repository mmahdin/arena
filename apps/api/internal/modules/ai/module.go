// Package ai exposes the new AI Core product adapter. There is no experimental reply path.
package ai

import (
	httpadapter "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/http"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/product"
)

func NewProductHTTP(s *product.Service) *httpadapter.Handler { return httpadapter.New(s) }
