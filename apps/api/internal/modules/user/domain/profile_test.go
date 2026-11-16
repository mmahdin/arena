package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestFirstSaveCompletesOnboardingAndLaterEditsPreserveTimestamp(t *testing.T) {
	var p Profile
	first := time.Date(2026, 11, 2, 9, 30, 0, 0, time.FixedZone("test", 3*3600))
	if err := p.Update("  نیلو ", Balanced, first); err != nil {
		t.Fatal(err)
	}
	if p.DisplayName != "نیلو" || p.OnboardingCompletedAt == nil || !p.OnboardingCompletedAt.Equal(first) || p.OnboardingCompletedAt.Location() != time.UTC {
		t.Fatalf("wrong profile state: %+v", p)
	}
	original := *p.OnboardingCompletedAt
	if err := p.Update("مریم", Listening, first.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if p.DisplayName != "مریم" || p.ConversationStyle != Listening || !p.OnboardingCompletedAt.Equal(original) {
		t.Fatalf("update reset onboarding: %+v", p)
	}
}

func TestInvalidProfileUpdateIsAtomic(t *testing.T) {
	p := Profile{DisplayName: "Safe", ConversationStyle: Balanced}
	for _, tc := range []struct {
		name, input string
		style       Style
		want        error
	}{
		{"blank", "  ", Balanced, ErrInvalidDisplayName},
		{"too long", strings.Repeat("ن", 61), Balanced, ErrInvalidDisplayName},
		{"line break", "Name\nInjected", Balanced, ErrInvalidDisplayName},
		{"direction override", "Name\u202eInjected", Balanced, ErrInvalidDisplayName},
		{"invalid style", "New", Style("unknown"), ErrInvalidConversationStyle},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := p.Update(tc.input, tc.style, time.Now()); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
			if p.DisplayName != "Safe" || p.ConversationStyle != Balanced || p.OnboardingCompletedAt != nil {
				t.Fatalf("partial invalid write: %+v", p)
			}
		})
	}
	if err := p.Update(strings.Repeat("ن", 60), Guidance, time.Now()); err != nil {
		t.Fatalf("60 runes rejected: %v", err)
	}
	if err := p.Update("می\u200cنا", Guidance, time.Now()); err != nil {
		t.Fatalf("Persian ZWNJ rejected: %v", err)
	}
}

func TestFiveIndependentStylesAreAcceptedAndInvalidRejected(t *testing.T) {
	for _, style := range []Style{Balanced, Listening, Teaching, GentleGuidance, Collaborative, Guidance} {
		p := Profile{}
		if err := p.Update("Synthetic User", style, time.Now()); err != nil || p.ConversationStyle != style {
			t.Fatalf("style %s not retained: %v", style, err)
		}
	}
	p := Profile{DisplayName: "Still safe", ConversationStyle: Balanced}
	if err := p.Update("Changed", "inject-instructions", time.Now()); !errors.Is(err, ErrInvalidConversationStyle) || p.DisplayName != "Still safe" {
		t.Fatal("untrusted style must be rejected without changing profile")
	}
}

func TestOnboardingNameDoesNotCompleteOnboardingOrChangeStyle(t *testing.T) {
	p := Profile{DisplayName: "", ConversationStyle: Listening}
	if err := p.SetOnboardingName("  مهسا  "); err != nil {
		t.Fatal(err)
	}
	if p.DisplayName != "مهسا" || p.ConversationStyle != Listening || p.OnboardingCompletedAt != nil {
		t.Fatalf("draft name must not complete onboarding or alter style: %+v", p)
	}
	if err := p.SetOnboardingName("  "); !errors.Is(err, ErrInvalidDisplayName) || p.DisplayName != "مهسا" {
		t.Fatal("invalid draft must not change existing name")
	}
	if err := p.Update("مهسا", Balanced, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := p.SetOnboardingName("دیگر"); !errors.Is(err, ErrOnboardingAlreadyComplete) || p.DisplayName != "مهسا" {
		t.Fatal("name-only onboarding endpoint must not edit completed profiles")
	}
}
