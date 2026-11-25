package safety

import (
	"context"
	"errors"
	"testing"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// Safety-side L0/L1 signal invariants. The cross-package WP-05 routing
// assertions live in the integration test package to avoid a Go import cycle.
func TestWP06SignalsForIntegratedRouter(t *testing.T) {
	// Route assertions have moved to the WP-05/WP-06 integrated orchestration suite.
	for _, tc := range []struct {
		name       string
		classifier *testClassifier
		text       string
		degraded   bool
	}{
		{"l0 high", lowClassifier(), "TEST_HIGH_SIGNAL", false},
		{"l1 imminent", &testClassifier{reply: Assessment{Output: contract.SafetyClassifierOutput{Level: contract.RiskImminent, Categories: []contract.RiskCategory{contract.RiskOther}, Confidence: 0.5}, Version: "test"}}, "ordinary", false},
		{"l0 high l1 down", &testClassifier{err: errors.New("provider unavailable")}, "TEST_HIGH_SIGNAL", true},
		{"l1 down safe ordinary", lowClassifier(), "ordinary", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gate := testGate(t, tc.classifier, nil)
			sig, err := gate.Assess(context.Background(), ports.SafetyInput{Text: tc.text, Lang: "en"})
			if tc.degraded != (err != nil) || sig.Degraded != tc.degraded {
				t.Fatalf("gate returned invalid degraded state: %+v, %v", sig, err)
			}
			if tc.degraded && sig.Level == contract.RiskHigh && (len(sig.Categories) != 1 || sig.Categories[0] != contract.RiskOther) {
				t.Fatal("l0 trigger category lost during l1 failure")
			}
			if (tc.text == "TEST_HIGH_SIGNAL" || tc.classifier.reply.Output.Level == contract.RiskImminent) && sig.Level.Rank() < contract.RiskHigh.Rank() {
				t.Fatalf("L0/L1 high risk lost: %+v", sig)
			}
		})
	}
}
