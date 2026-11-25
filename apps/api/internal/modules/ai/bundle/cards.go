package bundle

import (
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"io/fs"
	"strings"
)

// Catalog descriptions are versioned with their full source artifacts. They
// never replace the clinical review or integrity gates on those artifacts.
func validateCards(files map[string][]byte, r *Report) {
	var catalog struct {
		Schema string                   `yaml:"schema"`
		Cards  []contract.KnowledgeCard `yaml:"cards"`
	}
	if strictYAML(files["cards.yaml"], &catalog) != nil || catalog.Schema != "knowledge_cards.v1" || len(catalog.Cards) == 0 || len(catalog.Cards) > 128 {
		r.fail("cards.yaml", "knowledge_catalog", "invalid card catalog")
		return
	}
	ids, paths := map[string]bool{}, map[string]bool{}
	for _, card := range catalog.Cards {
		valid := card.ID != "" && !ids[card.ID] && !paths[card.ContentID] && fs.ValidPath(card.ContentID) && len(files[card.ContentID]) > 0
		valid = valid && oneOf(card.Category, "skill", "framework", "protocol", "exercise", "other")
		for _, text := range []string{card.Name, card.Description, card.Purpose, card.UseWhen, card.Limitations, card.Prerequisites} {
			if strings.TrimSpace(text) == "" || len(text) > 1200 {
				valid = false
			}
		}
		if strings.HasPrefix(card.ContentID, "policy/") || strings.HasPrefix(card.ContentID, "prompts/") || strings.HasPrefix(card.ContentID, "resources/crisis") {
			valid = false
		}
		if !valid {
			r.fail("cards.yaml", "knowledge_card", "invalid card metadata or content reference")
		}
		ids[card.ID] = true
		paths[card.ContentID] = true
	}
	for name := range files {
		if (strings.HasPrefix(name, "skills/") || strings.HasPrefix(name, "protocols/") || strings.HasPrefix(name, "schemas/formulation_")) && strings.HasSuffix(name, ".yaml") && !paths[name] {
			r.fail("cards.yaml", "knowledge_coverage", "a psychological resource has no card")
		}
	}
}
