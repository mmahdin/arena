package application

import (
	"context"
	"strings"
	"time"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
)

// Reflection is an explicitly authored note. It is not an AI-generated summary or memory.
type Reflection struct {
	Date      string    `json:"date"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Repository interface {
	List(context.Context, uuid.UUID) ([]Reflection, error)
	Get(context.Context, uuid.UUID, string) (Reflection, error)
	Save(context.Context, uuid.UUID, string, string) (Reflection, error)
	Delete(context.Context, uuid.UUID, string) error
}

type Service struct{ repo Repository }

func New(repo Repository) *Service { return &Service{repo: repo} }

func ValidDate(date string) bool {
	if len(date) != len("2006-01-02") {
		return false
	}
	parsed, err := time.Parse("2006-01-02", date)
	return err == nil && parsed.Format("2006-01-02") == date
}

func CleanText(input string) (string, bool) {
	clean := strings.TrimSpace(input)
	if clean == "" || !utf8.ValidString(clean) {
		return "", false
	}
	// NVARCHAR(4000) counts UTF-16 units; enforce the same limit in Go and JS.
	if len(utf16.Encode([]rune(clean))) > 4000 {
		return "", false
	}
	for _, r := range clean {
		// Persian half-spaces and script joiners are valid authored text.
		if (unicode.IsControl(r) && r != '\n' && r != '\t') || (unicode.Is(unicode.Cf, r) && r != '\u200c' && r != '\u200d') {
			return "", false
		}
	}
	return clean, true
}

func (s *Service) List(ctx context.Context, uid uuid.UUID) ([]Reflection, error) {
	if uid == uuid.Nil {
		return nil, httpx.ErrInvalid
	}
	return s.repo.List(ctx, uid)
}
func (s *Service) Get(ctx context.Context, uid uuid.UUID, date string) (Reflection, error) {
	if uid == uuid.Nil || !ValidDate(date) {
		return Reflection{}, httpx.ErrInvalid
	}
	return s.repo.Get(ctx, uid, date)
}
func (s *Service) Save(ctx context.Context, uid uuid.UUID, date, input string) (Reflection, error) {
	if uid == uuid.Nil || !ValidDate(date) {
		return Reflection{}, httpx.ErrInvalid
	}
	clean, valid := CleanText(input)
	if !valid {
		return Reflection{}, httpx.ErrInvalid
	}
	return s.repo.Save(ctx, uid, date, clean)
}
func (s *Service) Delete(ctx context.Context, uid uuid.UUID, date string) error {
	if uid == uuid.Nil || !ValidDate(date) {
		return httpx.ErrInvalid
	}
	return s.repo.Delete(ctx, uid, date)
}
