package contract

import "testing"

func TestResponseLanguage(t *testing.T) {
	for _, tc := range []struct{ preference, input, want string }{
		{"fa", "en", "fa"}, {"en", "fa", "en"}, {"FA-ir", "en", "fa"}, {"en-US", "fa", "en"},
		{"", "fa-IR", "fa"}, {"", "en", "en"}, {"", "", "fa"}, {"", "de", "fa"},
		{"de", "en", "fa"}, {"not_a_locale!", "en", "fa"}, {"  ", "en", "en"},
	} {
		if got := ResponseLanguage(tc.preference, tc.input); got != tc.want {
			t.Errorf("%q/%q: %q != %q", tc.preference, tc.input, got, tc.want)
		}
	}
	if got := NormalizeSafetyText("مي‌خواهم داروي ۱۲٣ كسی"); got != "می خواهم داروی 123 کسی" {
		t.Fatal(got)
	}
}
