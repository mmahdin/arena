package auth

import (
	"strings"
	"testing"
)

func TestSafeReturnPath(t *testing.T) {
	for _, tc := range []struct {
		in, want string
	}{
		{"", "/"},
		{"/", "/"},
		{"/onboarding", "/onboarding"},
		{"/conversations/abc?tab=messages", "/conversations/abc?tab=messages"},
		{"/settings?lang=fa&mode=dark", "/settings?lang=fa&mode=dark"},
		{"/__auth-smoke/user", "/__auth-smoke/user"},
	} {
		got, err := SafeReturnPath(tc.in)
		if err != nil || got != tc.want {
			t.Errorf("SafeReturnPath(%q) = %q, %v; want %q", tc.in, got, err, tc.want)
		}
	}
	for _, bad := range []string{
		"https://evil.invalid/path", "http://evil.invalid", "//evil.invalid", "///evil.invalid",
		"/\\\\evil.invalid", "/%2f%2fevil.invalid", "/%2Fevil.invalid",
		"/%5c%5cevil.invalid", "/%2e%2e/secret", "/a/../secret", "/a/./secret",
		"/path#fragment", "/\r\nLocation: https://evil.invalid", "/a%0db", "/a%0Ab",
		"/a%00b", "/a%09b", "/a\\b", "javascript:alert(1)",
		strings.Repeat("a", 1025), "/" + strings.Repeat("a", 1024),
	} {
		if _, err := SafeReturnPath(bad); err == nil {
			t.Errorf("unsafe return_to accepted: %q", bad)
		}
	}
}
