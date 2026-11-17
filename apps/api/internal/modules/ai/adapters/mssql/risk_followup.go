package mssql

import (
	"encoding/json"
	"time"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// riskFollowupForCommit validates WP-06's metadata-only risk event against the
// *same* transition being applied in the WP-02 Unit of Work. The deadline
// comes from the pinned Safety FSM, never an invented SQL time window. A
// mismatched event fails the entire assistant/state/trace transaction.
func riskFollowupForCommit(d c.StateDelta) (*time.Time, error) {
	n := 0
	var until *time.Time
	for _, ev := range d.Events {
		if ev.Type != c.EvRiskLevelChanged {
			continue
		}
		n++
		if d.User.Risk == nil || n != 1 {
			return nil, invalid()
		}
		payload, err := json.Marshal(ev.Payload)
		if err != nil {
			return nil, invalid()
		}
		var record struct {
			From          c.RiskLevel `json:"from"`
			To            c.RiskLevel `json:"to"`
			TurnID        c.ID        `json:"turn_id"`
			FollowUpUntil *time.Time  `json:"follow_up_until"`
		}
		if err = json.Unmarshal(payload, &record); err != nil || record.TurnID != d.TurnID ||
			record.From != d.User.Risk.From || record.To != d.User.Risk.To || !record.To.Valid() {
			return nil, invalid()
		}
		if record.To.Rank() >= c.RiskElevated.Rank() && record.FollowUpUntil == nil {
			return nil, invalid()
		}
		if record.FollowUpUntil != nil && !record.FollowUpUntil.After(d.User.Risk.At) {
			return nil, invalid()
		}
		if record.FollowUpUntil != nil {
			v := record.FollowUpUntil.UTC()
			until = &v
		}
	}
	if (d.User.Risk == nil && n != 0) || (d.User.Risk != nil && n != 1) {
		return nil, invalid()
	}
	return until, nil
}
