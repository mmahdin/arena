package application

import (
	"context"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
	"time"
)

type SavedMessage struct {
	ID             uuid.UUID `json:"id"`
	ConversationID uuid.UUID `json:"conversationId"`
	Role           string    `json:"role"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"createdAt"`
	SavedAt        time.Time `json:"savedAt"`
}
type Repository interface {
	List(context.Context, uuid.UUID) ([]SavedMessage, error)
	Save(context.Context, uuid.UUID, uuid.UUID) error
	Remove(context.Context, uuid.UUID, uuid.UUID) error
}
type Service struct{ repo Repository }

func New(repo Repository) *Service { return &Service{repo: repo} }
func (s *Service) List(ctx context.Context, uid uuid.UUID) ([]SavedMessage, error) {
	return s.repo.List(ctx, uid)
}
func (s *Service) Save(ctx context.Context, uid, id uuid.UUID) error {
	if uid == uuid.Nil || id == uuid.Nil {
		return httpx.ErrInvalid
	}
	return s.repo.Save(ctx, uid, id)
}
func (s *Service) Remove(ctx context.Context, uid, id uuid.UUID) error {
	if uid == uuid.Nil || id == uuid.Nil {
		return httpx.ErrInvalid
	}
	return s.repo.Remove(ctx, uid, id)
}
