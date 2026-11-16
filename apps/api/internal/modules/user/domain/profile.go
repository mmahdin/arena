package domain

import (
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

var (
	ErrInvalidDisplayName        = errors.New("invalid display name")
	ErrInvalidConversationStyle  = errors.New("invalid conversation style")
	ErrOnboardingAlreadyComplete = errors.New("onboarding already complete")
)

type Style string

const (
	Balanced       Style = "balanced"
	Listening      Style = "listening"
	Teaching       Style = "teaching"
	GentleGuidance Style = "gentle_guidance"
	Collaborative  Style = "collaborative"
	Guidance       Style = "guidance" // legacy value: same behavior as GentleGuidance
)

type Profile struct {
	UserID                uuid.UUID          `json:"userId"`
	DisplayName           string             `json:"displayName"`
	ConversationStyle     Style              `json:"conversationStyle"`
	OnboardingCompletedAt *time.Time         `json:"onboardingCompletedAt"`
	OnboardingAnswers     *OnboardingAnswers `json:"onboardingAnswers,omitempty"`
}

func validDisplayName(name string) bool {
	if n := utf8.RuneCountInString(name); n < 1 || n > 60 {
		return false
	}
	for _, r := range name {
		// Permit Persian ZWNJ (U+200C) while rejecting control chars and
		// direction overrides that can disguise text in logs or the UI.
		if unicode.IsControl(r) || (r >= 0x202A && r <= 0x202E) || (r >= 0x2066 && r <= 0x2069) {
			return false
		}
	}
	return true
}

// SetOnboardingName persists the user's chosen name before onboarding ends.
// Unlike Update, it must not complete onboarding or alter the conversation style.
func (p *Profile) SetOnboardingName(name string) error {
	if p.OnboardingCompletedAt != nil {
		return ErrOnboardingAlreadyComplete
	}
	name = strings.TrimSpace(name)
	if !validDisplayName(name) {
		return ErrInvalidDisplayName
	}
	p.DisplayName = name
	return nil
}

func (p *Profile) Update(name string, style Style, now time.Time) error {
	name = strings.TrimSpace(name)
	if !validDisplayName(name) {
		return ErrInvalidDisplayName
	}
	if style != Listening && style != Guidance && style != Balanced && style != Teaching && style != GentleGuidance && style != Collaborative {
		return ErrInvalidConversationStyle
	}
	p.DisplayName = name
	p.ConversationStyle = style
	if p.OnboardingCompletedAt == nil {
		completed := now.UTC()
		p.OnboardingCompletedAt = &completed
	}
	return nil
}
