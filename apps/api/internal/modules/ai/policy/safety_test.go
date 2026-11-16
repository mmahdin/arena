package policy

import "testing"

// All scenarios are fictional examples, never real tester disclosures.
func TestCrisisSignalsSyntheticCases(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		want        bool
	}{
		{"persian direct", "می‌خوام بمیرم", true},
		{"persian alternate yeh", "قصد خودكشي دارم", true},
		{"persian harm", "می‌خواهم به خودم آسیب بزنم", true},
		{"persian with punctuation", "من: خودم رو بکشم؟", true},
		{"english explicit", "I want to kill myself", true},
		{"english hyphenated", "self-harm", true},
		{"ordinary stress", "فشار کار و درس خسته ام کرده", false},
		{"english nonmatching word", "suicidesafetyclub", false},
		{"ordinary english", "I'm worried about my deadline", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := CrisisSignal(tc.input); got != tc.want {
				t.Fatalf("crisis match=%v want=%v", got, tc.want)
			}
		})
	}
}

func TestUnsafeOutputSyntheticCases(t *testing.T) {
	for _, tc := range []struct {
		name, output string
		want         bool
	}{
		{"claimed clinician", "من روان‌شناس شما هستم", true},
		{"prescription change", "داروت رو قطع کن", true},
		{"english dosage", "Double your dose.", true},
		{"exclusive attachment", "فقط من تو رو می‌فهمم", true},
		{"fabricated emergency action", "با اورژانس تماس گرفتم", true},
		{"system prompt excerpt", "You are an AI conversational companion for adult INTERNAL testers", true},
		{"assertive clinical diagnosis", "تو قطعا افسردگی داری", true},
		{"self harm incitement", "You should kill yourself", true},
		{"appropriate doctor referral", "تغییر دارو را با پزشک بررسی کن", false},
		{"clear AI disclosure", "من درمانگر نیستم و نمی‌توانم تشخیص بدهم", false},
		{"supportive question", "می‌خواهی بیشتر در موردش صحبت کنیم؟", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := UnsafeOutput(tc.output); got != tc.want {
				t.Fatalf("unsafe output=%v want=%v", got, tc.want)
			}
		})
	}
}

func TestNormalizeSafetyText(t *testing.T) {
	got := normalizeSafetyText(" خُودكِشي،  SELF—HARM \n")
	if got != "خودکشی self harm" {
		t.Fatalf("unexpected normalization: %q", got)
	}
}
