package domain

import (
	"encoding/json"
	"errors"
	"unicode"
	"unicode/utf8"
)

var ErrInvalidOnboardingAnswers = errors.New("invalid onboarding answers")

// OnboardingAnswers contains only explicitly submitted questionnaire selections.
// No generated summaries, inferred traits, or cross-conversation memory.
type OnboardingAnswers struct {
	Age            string   `json:"age,omitempty"`
	Identity       string   `json:"identity,omitempty"`
	Therapist      string   `json:"therapist,omitempty"`
	Faith          string   `json:"faith,omitempty"`
	Neurodivergent string   `json:"neurodivergent,omitempty"`
	Mood           *int     `json:"mood,omitempty"`
	Concerns       []string `json:"concerns,omitempty"`
	Focus          []string `json:"focus,omitempty"`
	Feelings       []string `json:"feelings,omitempty"`
	Relationships  []string `json:"relationships,omitempty"`
	Interests      []string `json:"interests,omitempty"`
	Strengths      []string `json:"strengths,omitempty"`
	Barriers       []string `json:"barriers,omitempty"`
	Recharge       []string `json:"recharge,omitempty"`
}

func (a OnboardingAnswers) Validate() error {
	valid := func(s string) bool {
		if !utf8.ValidString(s) || utf8.RuneCountInString(s) > 200 {
			return false
		}
		for _, r := range s {
			if unicode.IsControl(r) || (r >= 0x202A && r <= 0x202E) || (r >= 0x2066 && r <= 0x2069) {
				return false
			}
		}
		return true
	}
	for _, s := range []string{a.Age, a.Identity, a.Therapist, a.Faith, a.Neurodivergent} {
		if !valid(s) {
			return ErrInvalidOnboardingAnswers
		}
	}
	if a.Identity != "" && a.Identity != "زن" && a.Identity != "مرد" && a.Identity != "تمایل ندارم بگم" {
		return ErrInvalidOnboardingAnswers
	}
	if a.Mood != nil && (*a.Mood < 0 || *a.Mood > 4) {
		return ErrInvalidOnboardingAnswers
	}
	for _, items := range [][]string{a.Concerns, a.Focus, a.Feelings, a.Relationships, a.Interests, a.Strengths, a.Barriers, a.Recharge} {
		if len(items) > 8 {
			return ErrInvalidOnboardingAnswers
		}
		seen := map[string]bool{}
		for _, s := range items {
			if s == "" || !valid(s) || seen[s] {
				return ErrInvalidOnboardingAnswers
			}
			seen[s] = true
		}
	}
	if len(a.Focus) > 3 {
		return ErrInvalidOnboardingAnswers
	}
	data, err := json.Marshal(a)
	if err != nil || utf8.RuneCount(data) > 4000 {
		return ErrInvalidOnboardingAnswers
	}
	return nil
}

// ForModel leaves declined disclosures out of the provider context. The stored
// selection is preserved so a decline never becomes an inferred identity.
func (a OnboardingAnswers) ForModel() OnboardingAnswers {
	for _, field := range []*string{&a.Identity, &a.Therapist, &a.Faith, &a.Neurodivergent} {
		if *field == "تمایل ندارم بگم" || *field == "ترجیح می‌دم نگم" {
			*field = ""
		}
	}
	return a
}
