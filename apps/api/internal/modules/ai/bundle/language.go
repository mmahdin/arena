package bundle

import (
	"fmt"
	"slices"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// LanguageResources describes safety/resource review independently of English
// knowledge. Every conversation language still requires L0, Validator, crisis,
// boundary and fallback assets. ReviewStatus is not a substitute for hash approval.
type LanguageResources struct {
	ReviewStatus string `yaml:"review_status"`
	Protocols    bool   `yaml:"protocols"`
}

func (m Manifest) Conversations() []string {
	if len(m.ConversationLanguages) > 0 {
		return slices.Clone(m.ConversationLanguages)
	}
	return slices.Clone(m.Locales)
}
func (m Manifest) Knowledge() string {
	if m.KnowledgeLanguage == "" {
		return contract.KnowledgeLanguage
	}
	return m.KnowledgeLanguage
}

// ValidateLanguages is shared by runtime loaders as well as the release linter.
// Omitted new fields preserve legacy English-only signed bundle semantics.
func (m Manifest) ValidateLanguages() error {
	if m.Knowledge() != contract.KnowledgeLanguage || len(m.Conversations()) == 0 {
		return fmt.Errorf("invalid knowledge/conversation language")
	}
	// Historical manifests retain their existing coverage validation.
	if len(m.ConversationLanguages) == 0 && len(m.LanguageResources) == 0 {
		seen := map[string]bool{}
		for _, lang := range m.Locales {
			if !localeRE.MatchString(lang) || seen[lang] {
				return fmt.Errorf("invalid legacy locale")
			}
			seen[lang] = true
		}
		return nil
	}
	seen := map[string]bool{}
	for _, lang := range m.Conversations() {
		canonical, ok := contract.ConversationLanguage(lang)
		if !ok || canonical != lang || seen[lang] {
			return fmt.Errorf("invalid conversation language")
		}
		seen[lang] = true
		resource, exists := m.LanguageResources[lang]
		if !exists {
			if lang != "en" || len(m.ConversationLanguages) > 0 {
				return fmt.Errorf("missing language resource declaration")
			}
			continue
		}
		if resource.ReviewStatus != "reviewed" && resource.ReviewStatus != "development" {
			return fmt.Errorf("invalid language review status")
		}
		if !m.SyntheticOnly && resource.ReviewStatus != "reviewed" {
			return fmt.Errorf("unreviewed language resources")
		}
	}
	for lang := range m.LanguageResources {
		if !seen[lang] {
			return fmt.Errorf("undeclared resource language")
		}
	}
	if len(m.ConversationLanguages) > 0 && len(m.Locales) > 0 && !slices.Equal(m.Locales, m.ConversationLanguages) {
		return fmt.Errorf("conflicting legacy locales")
	}
	return nil
}
func (m Manifest) ProtocolLanguage(lang string) bool {
	if len(m.Conversations()) > 0 && !slices.Contains(m.Conversations(), lang) {
		return false
	}
	r, exists := m.LanguageResources[lang]
	return !exists || r.Protocols
}
