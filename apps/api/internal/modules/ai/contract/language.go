package contract

import "strings"

const KnowledgeLanguage = "en"
const DefaultResponseLanguage = "fa"

// ConversationLanguage canonicalizes supported BCP 47 tags, including regional
// variants. It does not detect the language of text or grant safety coverage.
func ConversationLanguage(tag string) (string, bool) {
	tag = strings.ToLower(strings.TrimSpace(tag))
	parts := strings.Split(tag, "-")
	for _, part := range parts {
		if len(part) < 2 || len(part) > 8 {
			return "", false
		}
		for _, r := range part {
			if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9') {
				return "", false
			}
		}
	}

	switch parts[0] {
	case "en", "fa":
		return parts[0], true
	default:
		return "", false
	}
}

// ResponseLanguage is independent of knowledge and input language. A saved
// unsupported preference falls back to Persian; only a missing preference uses
// the input hint. Bundle coverage is checked separately and never falls back.
func ResponseLanguage(preference, input string) string {
	if strings.TrimSpace(preference) != "" {
		if lang, ok := ConversationLanguage(preference); ok {
			return lang
		}
		return DefaultResponseLanguage
	}
	if lang, ok := ConversationLanguage(input); ok {
		return lang
	}
	return DefaultResponseLanguage
}

// NormalizeSafetyText folds common Persian/Arabic character variants and digit
// scripts for matchers only. One rune stays one rune, preserving L0 span offsets.
// Stored messages and generated prose are never rewritten.
func NormalizeSafetyText(text string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == 'ي' || r == 'ى':
			return 'ی'
		case r == 'ك':
			return 'ک'
		case r == '\u200c' || r == '\u200d':
			return ' '
		case r >= '۰' && r <= '۹':
			return '0' + r - '۰'
		case r >= '٠' && r <= '٩':
			return '0' + r - '٠'
		default:
			return r
		}
	}, text)
}
