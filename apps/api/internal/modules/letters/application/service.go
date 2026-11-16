package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

// A letter is created ONLY after the user explicitly requests generation.
// Provenance identifies the single user-owned conversation supporting its content.
type Letter struct {
	ID                    uuid.UUID   `json:"id"`
	Kind                  string      `json:"kind"`
	Title                 string      `json:"title"`
	Salutation            *string     `json:"salutation"`
	Paragraphs            []string    `json:"paragraphs"`
	Question              *string     `json:"question"`
	CreatedAt             time.Time   `json:"createdAt"`
	SourceConversationIDs []uuid.UUID `json:"sourceConversationIds"`
}

type Draft struct {
	Title      string   `json:"title"`
	Salutation *string  `json:"salutation"`
	Paragraphs []string `json:"paragraphs"`
	Question   *string  `json:"question"`
}

type Turn struct{ Role, Content string }
type Source struct {
	ConversationID uuid.UUID
	Turns          []Turn
}

type Repository interface {
	List(ctx context.Context, userID uuid.UUID) ([]Letter, error)
	LatestEligible(ctx context.Context, userID uuid.UUID) (Source, error)
	BySource(ctx context.Context, userID, conversationID uuid.UUID) (Letter, error)
	Save(ctx context.Context, userID, conversationID uuid.UUID, draft Draft) (Letter, error)
	Delete(ctx context.Context, userID, letterID uuid.UUID) error
}
type Generator interface {
	Generate(ctx context.Context, turns []Turn) (Draft, error)
}

type Service struct {
	repo      Repository
	generator Generator
}

var ErrInsufficientContext = errors.New("not enough user conversation context")
var ErrGenerationUnavailable = errors.New("letter generator unavailable")

func New(repo Repository, generator Generator) *Service {
	return &Service{repo: repo, generator: generator}
}
func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Letter, error) {
	if userID == uuid.Nil {
		return nil, httpx.ErrInvalid
	}
	return s.repo.List(ctx, userID)
}
func (s *Service) Delete(ctx context.Context, userID, id uuid.UUID) error {
	if userID == uuid.Nil || id == uuid.Nil {
		return httpx.ErrInvalid
	}
	return s.repo.Delete(ctx, userID, id)
}

// Generate never infers preferences from other users or generates automatically.
// A repeated request for the same conversation returns its previously stored letter.
func (s *Service) Generate(ctx context.Context, userID uuid.UUID, consent bool) (Letter, error) {
	if userID == uuid.Nil || !consent {
		return Letter{}, httpx.ErrInvalid
	}
	source, err := s.repo.LatestEligible(ctx, userID)
	if err != nil {
		return Letter{}, err
	}
	if prior, err := s.repo.BySource(ctx, userID, source.ConversationID); err == nil {
		return prior, nil
	} else if !errors.Is(err, httpx.ErrNotFound) {
		return Letter{}, err
	}
	if s.generator == nil {
		return Letter{}, ErrGenerationUnavailable
	}
	draft, err := s.generator.Generate(ctx, source.Turns)
	if err != nil || ctx.Err() != nil {
		return Letter{}, ErrGenerationUnavailable
	}
	if !ValidDraft(draft) {
		return Letter{}, ErrGenerationUnavailable
	}
	return s.repo.Save(ctx, userID, source.ConversationID, draft)
}

func validText(v string, max int) bool {
	v = strings.TrimSpace(v)
	if v == "" || !utf8.ValidString(v) || len(utf16.Encode([]rune(v))) > max {
		return false
	}
	for _, r := range v {
		if (unicode.IsControl(r) && r != '\n' && r != '\t') ||
			(unicode.Is(unicode.Cf, r) && r != '\u200c' && r != '\u200d') {
			return false
		}
	}
	return true
}
func ValidDraft(d Draft) bool {
	if !validText(d.Title, 160) || len(d.Paragraphs) < 2 || len(d.Paragraphs) > 6 {
		return false
	}
	if d.Salutation != nil && !validText(*d.Salutation, 120) {
		return false
	}
	if d.Question != nil && !validText(*d.Question, 600) {
		return false
	}
	for _, p := range d.Paragraphs {
		if !validText(p, 2000) {
			return false
		}
	}
	return true
}
func EncodeParagraphs(d Draft) (string, error) {
	if !ValidDraft(d) {
		return "", httpx.ErrInvalid
	}
	data, err := json.Marshal(d.Paragraphs)
	return string(data), err
}
