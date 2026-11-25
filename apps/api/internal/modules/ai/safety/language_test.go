package safety

import (
	"context"
	"strings"
	"testing"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

func TestMixedLanguageRulesCannotOverrideSecondLookVeto(t *testing.T) {
	b := syntheticBundle()
	// Reuse an ID across locales; English forbids downgrade, Persian permits it.
	// Both match this one synthetic input, and fa is visited after en.
	b.files["lexicons/safety_l0.en.yaml"] = strings.Replace(b.files["lexicons/safety_l0.en.yaml"], `"second_look":true`, `"second_look":false`, 1)
	b.files["lexicons/safety_l0.fa.yaml"] = strings.Replace(b.files["lexicons/safety_l0.fa.yaml"], "synthetic.fa.high.001", "synthetic.high.001", 1)
	a, err := Load(context.Background(), b, LoadOptions{AllowSyntheticForTests: true})
	if err != nil {
		t.Fatal(err)
	}
	classifier := lowClassifier()
	classifier.reply.Output.ContextFlags = []contract.SafetyContextFlag{contract.ContextQuoted}
	look := &fixedLook{review: SecondLookReview{Flags: []contract.SafetyContextFlag{contract.ContextQuoted}, ContextualTriggers: []bool{true, true}, FirstPersonChecked: true}}
	gate, err := NewGate(a, classifier, look)
	if err != nil {
		t.Fatal(err)
	}
	signal, err := gate.Assess(context.Background(), ports.SafetyInput{Lang: "en", Text: "TEST_HIGH_SIGNAL TEST_FA_HIGH"})
	if err != nil || signal.Level != contract.RiskHigh {
		t.Fatal("cross-locale duplicate rule erased downgrade veto", signal, err)
	}
}
