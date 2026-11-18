package protocol

import "strings"

// ExplicitReadiness is only a conservative opt-in cue for shortlisting; it is
// NOT consent to begin an exercise. Consent requires a separate response to an
// already-persisted offer and verified ownership of that turn's message.
func ExplicitReadiness(text string) Readiness {
	s := strings.ToLower(strings.Join(strings.Fields(text), " "))
	for _, v := range []string{"i want to try an exercise", "can we do an exercise", "please guide me through an exercise", "i would like to try a technique", "i would like to do an exercise", "could we try an exercise"} {
		if s == v || s == v+"?" || s == v+"." {
			return ReadinessHigh
		}
	}
	return ReadinessUnknown
}

// AffirmedOffer accepts ONLY a short standalone unambiguous affirmative in
// response to the current offered intervention. Free-form interpretation by
// the Brain cannot itself supply PR-003 consent.
func AffirmedOffer(text string) bool {
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "yes", "yes.", "yes, let's try", "yes, let's try.", "yes, i'd like to try", "yes, i'd like to try.", "i agree to try this exercise", "i agree to try this exercise.":
		return true
	default:
		return false
	}
}
