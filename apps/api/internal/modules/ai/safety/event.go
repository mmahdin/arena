package safety

import (
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// RiskEventPayload is deliberately metadata-only (SF-009). It contains no
// user message, classifier rationale, trigger spans or narrative.
// Persist it ONLY through the existing turn finalization transaction.
type RiskEventPayload struct {
	From          contract.RiskLevel      `json:"from"`
	To            contract.RiskLevel      `json:"to"`
	Categories    []contract.RiskCategory `json:"categories"`
	TurnID        contract.ID             `json:"turn_id"`
	FollowUpUntil *time.Time              `json:"follow_up_until,omitempty"`
}

// AddToDelta attaches an FSM transition and its minimal outbox event to the
// caller's PreparedTurn.State. It never writes directly to persistence. The
// WP-05 integration owner must call this before TurnFinalizer.Complete.
// Returns false for an absent transition, mismatched turn or a previously
// prepared risk transition/event, guarding against duplicate proposals.
func (r RiskChange) AddToDelta(d *contract.StateDelta, categories []contract.RiskCategory) bool {
	if d == nil || r.Transition == nil || d.TurnID != r.Transition.TurnID || !contract.ValidID(d.UserID) || d.User.Risk != nil {
		return false
	}
	for _, event := range d.Events {
		if event.Type == contract.EvRiskLevelChanged {
			return false
		}
	}
	clean := make([]contract.RiskCategory, 0, 3)
	for _, c := range categories {
		if !c.Valid() {
			continue
		}
		found := false
		for _, already := range clean {
			if already == c {
				found = true
				break
			}
		}
		if !found && len(clean) < 3 {
			clean = append(clean, c)
		}
	}
	transition := *r.Transition
	d.User.Risk = &transition
	var until *time.Time
	if r.State.FollowUpUntil != nil {
		value := *r.State.FollowUpUntil
		until = &value
	}
	d.Events = append(d.Events, contract.DomainEvent{
		Type: contract.EvRiskLevelChanged,
		Payload: RiskEventPayload{
			From: transition.From, To: transition.To, TurnID: transition.TurnID,
			Categories: clean, FollowUpUntil: until,
		},
	})
	return true
}
