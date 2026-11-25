package background

import (
	"encoding/json"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// Optional context can shrink; messages selected for coverage cannot disappear.
func (s *Service) fit(p *Prepared, retrieved []c.MemoryView) error {
	mandatory := map[c.ID]bool{}
	for _, id := range p.Input.Request.MessageIDs {
		mandatory[id] = true
	}
	targets := map[c.ID]bool{}
	for _, r := range retrieved {
		if r.Source == "memory" {
			targets[r.ID] = true
		}
	}
	for {
		raw, _ := json.Marshal(p.Input)
		if (len(raw)+len(s.prompt)+len(outputSchema)+3)/4 <= s.Config.InputTokens {
			return nil
		}
		if n := len(p.Input.Records); n > 0 {
			p.Input.Records = p.Input.Records[:n-1]
			continue
		}
		removed := false
		for i := len(p.Input.Memories) - 1; i >= 0; i-- {
			if !targets[p.Input.Memories[i].Item.ID] {
				p.Input.Memories = append(p.Input.Memories[:i], p.Input.Memories[i+1:]...)
				removed = true
				break
			}
		}
		if removed {
			continue
		}
		for i := len(p.Input.Messages) - 1; i >= 0; i-- {
			if !mandatory[p.Input.Messages[i].ID] {
				p.Input.Messages = append(p.Input.Messages[:i], p.Input.Messages[i+1:]...)
				removed = true
				break
			}
		}
		if removed {
			continue
		}
		return &c.Error{Code: c.ErrBudget, Step: "intelligence_primary_input"}
	}
}
