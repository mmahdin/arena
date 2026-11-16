package application

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/conversation/domain"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
	"time"
)

type ClaimedTurn struct {
	LeaseID            uuid.UUID
	Existing           bool
	Pending            bool
	Reply              string
	PromptVersion      string
	UserMessageID      uuid.UUID
	AssistantMessageID uuid.UUID
}
type TurnResult struct {
	UserMessageID      uuid.UUID `json:"userMessageId"`
	AssistantMessageID uuid.UUID `json:"assistantMessageId"`
	Reply              string    `json:"reply"`
	PromptVersion      string    `json:"promptVersion"`
}
type Repository interface {
	List(ctx context.Context, userID uuid.UUID) ([]domain.Conversation, error)
	Create(ctx context.Context, userID uuid.UUID) (domain.Conversation, error)
	Rename(ctx context.Context, userID, conversationID uuid.UUID, title string) (domain.Conversation, error)
	Messages(ctx context.Context, userID, conversationID uuid.UUID, limit int) ([]domain.Message, error)
	Delete(ctx context.Context, userID, conversationID uuid.UUID) error
	ClaimTurn(ctx context.Context, userID, conversationID, clientTurnID uuid.UUID, input string) (ClaimedTurn, error)
	CompleteTurn(ctx context.Context, userID, conversationID, clientTurnID, leaseID uuid.UUID, input, reply, version string) (TurnResult, error)
	ReleaseTurn(ctx context.Context, userID, conversationID, clientTurnID, leaseID uuid.UUID) error
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
type ReplyGenerator interface {
	Reply(ctx context.Context, profile Profile, history []domain.Message, userMessage string) (reply, version string, err error)
}
type Service struct {
	repo       Repository
	profiles   ProfileReader
	generator  ReplyGenerator
	maxContext int
}

func New(repo Repository, profiles ProfileReader, generator ReplyGenerator, maxContext int) *Service {
	return &Service{repo: repo, profiles: profiles, generator: generator, maxContext: maxContext}
}
func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]domain.Conversation, error) {
	return s.repo.List(ctx, userID)
}
func (s *Service) Create(ctx context.Context, userID uuid.UUID) (domain.Conversation, error) {
	profile, err := s.profiles.ForUser(ctx, userID)
	if err != nil {
		return domain.Conversation{}, err
	}
	if !profile.OnboardingComplete {
		return domain.Conversation{}, httpx.ErrOnboardingRequired
	}
	return s.repo.Create(ctx, userID)
}
func (s *Service) Rename(ctx context.Context, userID, convID uuid.UUID, title string) (domain.Conversation, error) {
	clean, err := domain.ValidateTitle(title)
	if err != nil {
		return domain.Conversation{}, httpx.ErrInvalid
	}
	return s.repo.Rename(ctx, userID, convID, clean)
}
func (s *Service) Messages(ctx context.Context, userID, convID uuid.UUID) ([]domain.Message, error) {
	return s.repo.Messages(ctx, userID, convID, 100)
}
func (s *Service) Delete(ctx context.Context, userID, convID uuid.UUID) error {
	return s.repo.Delete(ctx, userID, convID)
}
func (s *Service) Turn(ctx context.Context, userID, convID, clientTurnID uuid.UUID, message string) (TurnResult, error) {
	var result TurnResult
	if err := domain.ValidateMessage(message); err != nil || clientTurnID == uuid.Nil {
		return result, httpx.ErrInvalid
	}
	profile, err := s.profiles.ForUser(ctx, userID)
	if err != nil {
		return result, err
	}
	if !profile.OnboardingComplete {
		// Check ownership before returning the onboarding error, so a foreign
		// conversation is still indistinguishable from a missing resource.
		if _, err := s.repo.Messages(ctx, userID, convID, 1); err != nil {
			return result, err
		}
		return result, httpx.ErrOnboardingRequired
	}
	claim, err := s.repo.ClaimTurn(ctx, userID, convID, clientTurnID, message)
	if err != nil {
		return result, err
	}
	if claim.Existing {
		return TurnResult{UserMessageID: claim.UserMessageID, AssistantMessageID: claim.AssistantMessageID, Reply: claim.Reply, PromptVersion: claim.PromptVersion}, nil
	}
	if claim.Pending {
		return result, httpx.ErrConflict
	}
	if claim.LeaseID == uuid.Nil {
		return result, httpx.ErrUnavailable
	}
	completed := false
	defer func() {
		if !completed {
			retryCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
			defer cancel()
			_ = s.repo.ReleaseTurn(retryCtx, userID, convID, clientTurnID, claim.LeaseID)
		}
	}()
	history, err := s.repo.Messages(ctx, userID, convID, s.maxContext)
	if err != nil {
		return result, err
	}
	reply, version, err := s.generator.Reply(ctx, profile, history, message)
	if err != nil {
		return result, errors.Join(httpx.ErrUnavailable, err)
	}
	if reply == "" {
		return result, httpx.ErrUnavailable
	}
	result, err = s.repo.CompleteTurn(ctx, userID, convID, clientTurnID, claim.LeaseID, message, reply, version)
	if err != nil {
		return TurnResult{}, err
	}
	completed = true
	return result, nil
}
