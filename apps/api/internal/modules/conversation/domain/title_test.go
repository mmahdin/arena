package domain

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSuggestedTitle(t *testing.T) {
	cases := []struct{ input, expected string }{
		{"  سلام، امروز چطوری؟  ", "سلام، امروز چطوری"},
		{"برنامه‌ریزی برای آخر هفته. موضوع دوم", "برنامه‌ریزی برای آخر هفته"},
		{"   \n   ", DefaultTitle},
		{"https://example.com  test@example.com  09123456789", DefaultTitle},
		{"  برنامه‌ریزی \n برای   فردا ", "برنامه‌ریزی برای فردا"},
	}
	for _, tc := range cases {
		if got := SuggestedTitle(tc.input); got != tc.expected {
			t.Errorf("SuggestedTitle(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
	long := strings.Repeat("فردا ", 30)
	if got := SuggestedTitle(long); utf8.RuneCountInString(got) > MaxTitleRunes {
		t.Errorf("overlong title %q", got)
	}
}
func TestValidateTitle(t *testing.T) {
	for _, bad := range []string{"", " \n ", "line\nfeed", "عنوان\n", "\tعنوان", strings.Repeat("آ", 61)} {
		if _, err := ValidateTitle(bad); err == nil {
			t.Errorf("accepted invalid title %q", bad)
		}
	}
	if got, err := ValidateTitle("  اسم دلخواه  "); err != nil || got != "اسم دلخواه" {
		t.Errorf("unexpected validated title %q, %v", got, err)
	}
}
