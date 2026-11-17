package orchestrator_test

import (
	"context"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/orchestrator"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

func TestDeterministicRoutePrecedence(t *testing.T) {
	config := ports.RouterConfig{TrivialPatterns: map[string][]string{"en": {"hello", "thanks"}}, TrivialMaxChars: 24}
	baseline := func() ports.RouteInput {
		return ports.RouteInput{Origin: c.OriginUser, Safety: c.SafetySignal{Level: c.RiskLow}, Config: config, WCtx: c.WorkingContext{Turn: c.TurnView{Text: "hello", Lang: "en"}, Risk: c.RiskView{Current: c.RiskLow}}}
	}
	tests := []struct {
		name   string
		edit   func(*ports.RouteInput)
		want   c.Route
		reason string
		buffer bool
	}{
		{"trivial", func(*ports.RouteInput) {}, c.RouteFast, "RT-004", false},
		{"normal", func(i *ports.RouteInput) { i.WCtx.Turn.Text = "Need to discuss something" }, c.RouteNormal, "RT-099", false},
		{"safety_high_priority", func(i *ports.RouteInput) { i.Safety.Level = c.RiskHigh; i.Origin = c.OriginWakeUp }, c.RouteCrisis, "RT-001", true},
		{"context_high_priority", func(i *ports.RouteInput) { i.WCtx.Risk.Current = c.RiskImminent }, c.RouteCrisis, "RT-001", true},
		{"unknown_risk_restricts", func(i *ports.RouteInput) { i.WCtx.Risk.Current = "unknown" }, c.RouteNormal, "RT-099", true},
		{"active_followup", func(i *ports.RouteInput) { i.WCtx.Risk.FollowUpActive = true }, c.RouteNormal, "RT-002", true},
		{"wakeup", func(i *ports.RouteInput) { i.Origin = c.OriginWakeUp }, c.RouteProactive, "RT-003", false},
		{"pending_question", func(i *ports.RouteInput) { i.WCtx.Session.PendingQuestion = &c.PendingQuestion{AskedAt: time.Now()} }, c.RouteNormal, "RT-099", false},
		{"active_protocol", func(i *ports.RouteInput) { i.WCtx.Protocol = &c.ProtocolView{} }, c.RouteNormal, "RT-099", false},
		{"elevated_even_if_trivial", func(i *ports.RouteInput) { i.Safety.Level = c.RiskElevated }, c.RouteNormal, "RT-099", true},
		{"substring_is_not_trivial", func(i *ports.RouteInput) { i.WCtx.Turn.Text = "hello, I am in trouble" }, c.RouteNormal, "RT-099", false},
		{"different_locale", func(i *ports.RouteInput) { i.WCtx.Turn.Lang = "fa" }, c.RouteNormal, "RT-099", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := baseline()
			tt.edit(&in)
			d := orchestrator.Router{}.Select(in)
			if d.Route != tt.want || d.Reason != tt.reason || d.ForceBufferedOutput != tt.buffer {
				t.Fatalf("got %+v, want %s/%s buffered=%v", d, tt.want, tt.reason, tt.buffer)
			}
		})
	}
}

func TestNoClinicalClaimsFromTrivialRuleAndNoSilentRecovery(t *testing.T) {
	h := newHarness(t, orchestrator.Options{RouterConfig: ports.RouterConfig{TrivialPatterns: map[string][]string{"en": {"hello"}}}})
	req := h.request(h.conv)
	req.Text = "hello"
	r, err := h.o.ProcessTurn(context.Background(), req)
	if err != nil || r.Route != c.RouteFast || h.b.count() != 0 || h.g.count() != 1 {
		t.Fatalf("FAST should bypass Brain, not Generator/Validator: %+v %v", r, err)
	}
	// In-flight duplicates must not get a live lease token or second model call.
	h2 := newHarness(t, orchestrator.Options{})
	h2.b.enter = make(chan struct{}, 1)
	h2.b.proceed = make(chan struct{})
	req = h2.request(h2.conv)
	done := make(chan error, 1)
	go func() { _, err := h2.o.ProcessTurn(context.Background(), req); done <- err }()
	<-h2.b.enter
	second, e := h2.o.ProcessTurn(context.Background(), req)
	if e != nil || second.Status != c.TurnRunning || h2.b.count() != 1 {
		t.Fatalf("active turn replay executed twice: %+v %v", second, e)
	}
	close(h2.b.proceed)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
