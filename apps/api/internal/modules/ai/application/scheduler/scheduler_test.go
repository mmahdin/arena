package scheduler

import (
	"context"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"testing"
	"time"
)

type p struct{}

func (p) Allow(_ context.Context, in mem.Permission) (bool, error) {
	return in.User.Consent["proactive"], nil
}
func TestAuthorizationQuietHoursRiskAndExplicitConsent(t *testing.T) {
	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC)
	u := c.UserState{ProactiveOptIn: true, TZ: "UTC", Risk: c.RiskState{Level: c.RiskNone}, Consent: map[string]bool{"proactive": true}}
	if !Authorized(context.Background(), p{}, u, now) {
		t.Fatal("explicit authorized")
	}
	for _, mode := range []string{"optout", "missing_consent", "risk", "followup", "timezone", "quiet", "invalidquiet"} {
		t.Run(mode, func(t *testing.T) {
			v := u
			switch mode {
			case "optout":
				v.ProactiveOptIn = false
			case "missing_consent":
				v.Consent = nil
			case "risk":
				v.Risk.Level = c.RiskElevated
			case "followup":
				until := now.Add(time.Hour)
				v.Risk.FollowUpUntil = &until
			case "timezone":
				v.TZ = "bad"
			case "quiet":
				v.QuietHours = &c.QuietHours{Start: "11:00", End: "13:00"}
			case "invalidquiet":
				v.QuietHours = &c.QuietHours{Start: "bad", End: "13:00"}
			}
			if Authorized(context.Background(), p{}, v, now) {
				t.Fatal("unauthorized wakeup")
			}
		})
	}
	q := &c.QuietHours{Start: "22:00", End: "07:00"}
	if !Quiet(now.Add(12*time.Hour), q) || Quiet(now, q) {
		t.Fatal("overnight quiet hours")
	}
}
