package auth

import (
	"net/url"
	"strings"
)

// SafeReturnPath accepts only an app-local path. It must never accept an
// absolute URL, protocol-relative URL, encoded slash/backslash, control
// character, or dot-segment that a proxy/browser could interpret differently.
// The validated value is sealed into the one-time OIDC login transaction;
// callback query parameters never select the post-login destination.
func SafeReturnPath(raw string) (string, error) {
	if raw == "" {
		return "/", nil
	}
	if len(raw) > 1024 || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.Contains(raw, "#") {
		return "", ErrInvalid
	}
	for _, r := range raw {
		if r < 0x20 || r == 0x7f || r == '\\' {
			return "", ErrInvalid
		}
	}
	lower := strings.ToLower(raw)
	for _, encoded := range []string{"%2f", "%5c", "%0a", "%0d", "%00", "%09"} {
		if strings.Contains(lower, encoded) {
			return "", ErrInvalid
		}
	}
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.Opaque != "" || parsed.Fragment != "" ||
		!strings.HasPrefix(parsed.Path, "/") || strings.HasPrefix(parsed.Path, "//") {
		return "", ErrInvalid
	}
	for _, segment := range strings.Split(parsed.Path, "/") {
		if segment == "." || segment == ".." {
			return "", ErrInvalid
		}
	}
	return raw, nil
}
