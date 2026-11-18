package orchestrator_test

import (
	"context"
	"errors"
	"testing"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/orchestrator"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

func TestWakeUpCannotEnterUserAdmissionOrProduceMessages(t *testing.T) {
	h := newHarness(t, orchestrator.Options{})
	for _, variant := range []struct {
		name    string
		origin  c.TurnOrigin
		payload bool
	}{
		{"wakeup_origin", c.OriginWakeUp, false},
		{"wakeup_payload_disguised_as_user", c.OriginUser, true},
	} {
		t.Run(variant.name, func(t *testing.T) {
			req := h.request(h.conv)
			req.Origin = variant.origin
			if variant.payload {
				req.WakeUp = &c.WakeUpInfo{Reason: "check_in", FollowUpID: c.NewID()}
			}
			_, err := h.o.ProcessTurn(context.Background(), req)
			var ce *c.Error
			if !errors.As(err, &ce) || ce.Code != c.ErrForbidden {
				t.Fatalf("unapproved proactive admission: %v", err)
			}
			if len(h.s.order) != 0 || len(h.client.history()) != 0 {
				t.Fatal("a proactive event fabricated user state or called the provider")
			}
		})
	}
}
