package validator_test

import (
	"context"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/validator"
	"strings"
	"testing"
)

func TestConversationalVariationDoesNotDiscardHelpfulAnswers(t *testing.T) {
	v := svc(t, validator.Options{})
	wc, plan := base()
	for _, text := range []string{
		"What matters to you? What feels manageable?",
		"How thoughts and feelings influence one another is useful to explore.",
		"Try a breathing exercise if that sounds useful; you can stop whenever you want.",
		strings.Repeat("Here is a concrete explanation with useful examples. ", 24),
	} {
		report, err := v.Finalize(context.Background(), text, plan, wc, c.RiskLow, nil)
		if err != nil || !report.Passed {
			t.Fatal("style discarded helpful text", report, err)
		}
	}
	plan.Protocol = &c.ProtocolDirective{Action: c.PAAdvance}
	report, err := v.Finalize(context.Background(), "We can take this slowly, and you can leave any part out.", plan, wc, c.RiskLow, nil)
	if err != nil || !report.Passed || hasCode(report, "OV-014") {
		t.Fatal("ordinary protocol required a script", report, err)
	}
	guard := v.StreamGuard(plan, c.RiskLow)
	if ok, _ := guard.Push("What matters to you? What feels manageable?"); !ok {
		t.Fatal("private stream rejected question punctuation")
	}
}
