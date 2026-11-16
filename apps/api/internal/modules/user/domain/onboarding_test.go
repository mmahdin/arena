package domain

import (
	"strings"
	"testing"
)

func TestOnboardingAnswerLimitsAndDeclinedDisclosures(t *testing.T) {
	zero, invalidMood := 0, 5
	valid := OnboardingAnswers{Mood: &zero, Identity: "تمایل ندارم بگم", Therapist: "ترجیح می‌دم نگم", Faith: "ترجیح می‌دم نگم", Neurodivergent: "ترجیح می‌دم نگم"}
	if valid.Validate() != nil || (OnboardingAnswers{}).Validate() != nil {
		t.Fatal("valid empty or zero-mood answers rejected")
	}
	model := valid.ForModel()
	if model.Identity != "" || model.Therapist != "" || model.Faith != "" || model.Neurodivergent != "" || valid.Identity == "" || model.Mood == nil {
		t.Fatal("declines were sent to the model, mutated in storage, or mood zero was lost")
	}
	for _, a := range []OnboardingAnswers{
		{Mood: &invalidMood}, {Identity: "invalid"}, {Faith: strings.Repeat("ن", 201)},
		{Interests: []string{""}}, {Interests: []string{"same", "same"}}, {Faith: "bad\u202e"},
		{Focus: []string{"a", "b", "c", "d"}}, {Recharge: make([]string, 9)},
		{Concerns: []string{strings.Repeat("ن", 200), strings.Repeat("م", 200), strings.Repeat("س", 200), strings.Repeat("ا", 200), strings.Repeat("ب", 200), strings.Repeat("ج", 200), strings.Repeat("د", 200), strings.Repeat("ه", 200)}, Feelings: []string{strings.Repeat("ن", 200), strings.Repeat("م", 200), strings.Repeat("س", 200), strings.Repeat("ا", 200), strings.Repeat("ب", 200), strings.Repeat("ج", 200), strings.Repeat("د", 200), strings.Repeat("ه", 200)}, Age: strings.Repeat("ن", 200), Faith: strings.Repeat("ن", 200), Therapist: strings.Repeat("ن", 200), Neurodivergent: strings.Repeat("ن", 200)},
	} {
		if a.Validate() == nil {
			t.Fatal("invalid or unbounded onboarding answers accepted")
		}
	}
}
