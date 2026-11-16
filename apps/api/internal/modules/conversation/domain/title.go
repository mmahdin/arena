package domain

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const DefaultTitle = "گفت‌وگوی جدید"
const MaxTitleRunes = 60

var (
	contactPattern = regexp.MustCompile(`(?i)\b(?:https?://|www\.)\S+|\b[\w.+-]+@[\w.-]+\.[a-z]{2,}\b|\b\d{8,}\b`)
)

// SuggestedTitle is deliberately local and deterministic: no provider calls,
// profiling or cross-chat context. Avoid including common contact identifiers
// in the history title. The original message itself is never modified.
func SuggestedTitle(message string) string {
	clean := contactPattern.ReplaceAllString(message, " ")
	clean = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return ' '
		}
		return r
	}, clean)
	clean = strings.Join(strings.Fields(clean), " ")
	clean = strings.Trim(clean, " \"'«».,،؛:!?؟-–—#*")
	for _, separator := range []string{". ", "؟ ", "! ", "? ", "؛ "} {
		if pos := strings.Index(clean, separator); pos > 9 {
			clean = clean[:pos]
		}
	}
	chars := []rune(clean)
	if len(chars) > MaxTitleRunes {
		chars = chars[:MaxTitleRunes-1]
		short := strings.TrimSpace(string(chars))
		// Avoid breaking a final word when a readable word boundary exists.
		if idx := strings.LastIndex(short, " "); idx >= 18 {
			short = short[:idx]
		}
		clean = strings.TrimSpace(short) + "…"
	}
	clean = strings.TrimSpace(clean)
	if clean == "" {
		return DefaultTitle
	}
	return clean
}

func ValidateTitle(title string) (string, error) {
	clean := strings.TrimSpace(title)
	if clean == "" || utf8.RuneCountInString(clean) > MaxTitleRunes {
		return "", errors.New("title must contain 1..60 characters")
	}
	if strings.ContainsFunc(title, unicode.IsControl) {
		return "", errors.New("title must not contain control characters")
	}
	return clean, nil
}
