package mssql

import (
	"testing"
	"time"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/safety"
)

func TestRiskFollowupForAtomicCommit(t *testing.T) {
	at := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	turn, user := c.NewID(), c.NewID()
	for _, tc := range []struct {
		name     string
		level    c.RiskLevel
		deadline bool
		valid    bool
	}{
		{"high with approved fsm deadline", c.RiskHigh, true, true},
		{"elevated requires deadline", c.RiskElevated, false, false},
		{"low without deadline", c.RiskLow, false, true},
		{"none without deadline", c.RiskNone, false, true},
		{"high missing deadline", c.RiskHigh, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			transition := c.RiskTransition{At: at, From: c.RiskNone, To: tc.level, TurnID: turn, Reason: "SF-001"}
			d := c.StateDelta{UserID: user, TurnID: turn, User: c.UserStatePatch{Risk: &transition}}
			var until *time.Time
			if tc.deadline {
				val := at.Add(24 * time.Hour)
				until = &val
			}
			d.Events = []c.DomainEvent{{Type: c.EvRiskLevelChanged, Payload: safety.RiskEventPayload{From: transition.From, To: transition.To, TurnID: turn, FollowUpUntil: until}}}
			got, err := riskFollowupForCommit(d)
			if (err == nil) != tc.valid {
				t.Fatalf("unexpected validation result: %v", err)
			}
			if err == nil && tc.deadline && (got == nil || !got.Equal(*until)) {
				t.Fatalf("deadline lost: %v", got)
			}
			if err == nil && !tc.deadline && got != nil {
				t.Fatalf("invented follow-up deadline %v", got)
			}
		})
	}
}

func TestRiskFollowupRejectsDuplicateOrMismatchedEvents(t *testing.T) {
	at := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	turn, user := c.NewID(), c.NewID()
	until := at.Add(time.Hour)
	transition := c.RiskTransition{At: at, From: c.RiskNone, To: c.RiskHigh, TurnID: turn, Reason: "SF-001"}
	base := func() c.StateDelta {
		return c.StateDelta{UserID: user, TurnID: turn, User: c.UserStatePatch{Risk: &transition}, Events: []c.DomainEvent{{Type: c.EvRiskLevelChanged, Payload: safety.RiskEventPayload{From: c.RiskNone, To: c.RiskHigh, TurnID: turn, FollowUpUntil: &until}}}}
	}
	cases := []struct {
		name   string
		mutate func(*c.StateDelta)
	}{
		{"missing", func(d *c.StateDelta) { d.Events = nil }},
		{"duplicate", func(d *c.StateDelta) { d.Events = append(d.Events, d.Events[0]) }},
		{"different turn", func(d *c.StateDelta) { d.TurnID = c.NewID() }},
		{"different to", func(d *c.StateDelta) { d.User.Risk.To = c.RiskElevated }},
		{"deadline before transition", func(d *c.StateDelta) {
			past := at.Add(-time.Hour)
			d.Events[0].Payload = safety.RiskEventPayload{From: c.RiskNone, To: c.RiskHigh, TurnID: turn, FollowUpUntil: &past}
		}},
		{"no risk update", func(d *c.StateDelta) { d.User.Risk = nil }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := base()
			tc.mutate(&d)
			if _, err := riskFollowupForCommit(d); err == nil {
				t.Fatal("invalid event was accepted")
			}
		})
	}
	noChange := c.StateDelta{UserID: user, TurnID: turn}
	if v, err := riskFollowupForCommit(noChange); err != nil || v != nil {
		t.Fatalf("unchanged risk cannot require fake event: %v %v", v, err)
	}
}
