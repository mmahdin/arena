package application

import (
	"context"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/conversation/domain"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

type Repository interface {
	List(context.Context, uuid.UUID) ([]domain.Conversation, error)
	Create(context.Context, uuid.UUID) (domain.Conversation, error)
	Rename(context.Context, uuid.UUID, uuid.UUID, string) (domain.Conversation, error)
	Messages(context.Context, uuid.UUID, uuid.UUID, int) ([]domain.Message, error)
	Delete(context.Context, uuid.UUID, uuid.UUID) error
}
type Profile struct {
	DisplayName        string
	ConversationStyle  string
	OnboardingComplete bool
	OnboardingAnswers  *OnboardingAnswers
}
type ProfileReader interface {
	ForUser(ctx context.Context, id uuid.UUID) (Profile, error)
}

type Service struct {
	repo     Repository
	profiles ProfileReader
}

func New(repo Repository, profiles ProfileReader) *Service { return &Service{repo, profiles} }
func (s *Service) List(ctx context.Context, u uuid.UUID) ([]domain.Conversation, error) {
	return s.repo.List(ctx, u)
}
func (s *Service) Create(ctx context.Context, u uuid.UUID) (domain.Conversation, error) {
	p, e := s.profiles.ForUser(ctx, u)
	if e != nil {
		return domain.Conversation{}, e
	}
	if !p.OnboardingComplete {
		return domain.Conversation{}, httpx.ErrOnboardingRequired
	}
	return s.repo.Create(ctx, u)
}
func (s *Service) Rename(ctx context.Context, u, id uuid.UUID, title string) (domain.Conversation, error) {
	v, e := domain.ValidateTitle(title)
	if e != nil {
		return domain.Conversation{}, httpx.ErrInvalid
	}
	return s.repo.Rename(ctx, u, id, v)
}
func (s *Service) Messages(ctx context.Context, u, id uuid.UUID) ([]domain.Message, error) {
	return s.repo.Messages(ctx, u, id, 100)
}
func (s *Service) Delete(ctx context.Context, u, id uuid.UUID) error {
	return s.repo.Delete(ctx, u, id)
}
