package safety

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type testClassifier struct {
	reply Assessment
	err   error
	calls int
}

func (c *testClassifier) Classify(_ context.Context, _ ports.SafetyInput) (Assessment, error) {
	c.calls++
	return c.reply, c.err
}
func lowClassifier() *testClassifier {
	return &testClassifier{reply: Assessment{Output: contract.SafetyClassifierOutput{Level: contract.RiskLow, Confidence: 0.91}, Version: "synthetic-classifier-v1"}}
}
func testGate(t *testing.T, c *testClassifier, look SecondLook) *Gate {
	t.Helper()
	g, err := NewGate(loadFixture(t), c, look)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestGateL0WinsUnicodeOffsetsAndNoRawTextInTriggers(t *testing.T) {
	g := testGate(t, lowClassifier(), nil)
	sig, err := g.Assess(context.Background(), ports.SafetyInput{Text: "🙂 TEST_HIGH_SIGNAL", Lang: "en"})
	if err != nil || sig.Level != contract.RiskHigh || sig.Source != "l0" || sig.Degraded || sig.LexiconVersion != "synthetic-0.9.0" {
		t.Fatalf("L0 priority: %+v %v", sig, err)
	}
	if len(sig.Triggers) != 1 || sig.Triggers[0].RuleID != "synthetic.high.001" || sig.Triggers[0].Span == nil || sig.Triggers[0].Span.Start != 2 || sig.Triggers[0].Span.End != 18 {
		t.Fatalf("incorrect unicode span: %+v", sig.Triggers)
	}
	if strings.Contains(sig.Triggers[0].RuleID, "TEST_HIGH_SIGNAL") {
		t.Fatal("raw content leaked into trace")
	}
	if sig.ClassifierVersion != "synthetic-classifier-v1" {
		t.Fatal("classifier version missing")
	}
}
func TestGateL1EscalatesNeverLowersL0(t *testing.T) {
	c := lowClassifier()
	c.reply.Output = contract.SafetyClassifierOutput{Level: contract.RiskImminent, Categories: []contract.RiskCategory{contract.RiskOther}, Confidence: 0.81}
	g := testGate(t, c, nil)
	result, err := g.Assess(context.Background(), ports.SafetyInput{Text: "neutral synthetic", Lang: "en-US"})
	if err != nil || result.Level != contract.RiskImminent || result.Source != "l1" || len(result.Triggers) != 1 {
		t.Fatalf("L1 escalate: %+v %v", result, err)
	}
	high, err := g.Assess(context.Background(), ports.SafetyInput{Text: "TEST_HIGH_SIGNAL", Lang: "en"})
	if err != nil || high.Level != contract.RiskImminent {
		t.Fatalf("no L0 downgrade: %+v %v", high, err)
	}
}
func TestGateFailClosedMatrix(t *testing.T) {
	cases := []struct {
		name         string
		c            *testClassifier
		text, locale string
		want         contract.RiskLevel
	}{
		{"classifier unavailable", &testClassifier{err: errors.New("provider down")}, "ordinary", "en", contract.RiskElevated},
		{"classifier absent high stays high", &testClassifier{err: context.DeadlineExceeded}, "TEST_HIGH_SIGNAL", "en", contract.RiskHigh},
		{"bad output", &testClassifier{reply: Assessment{Output: contract.SafetyClassifierOutput{Level: "unknown", Confidence: 0.5}}}, "ordinary", "en", contract.RiskElevated},
		{"confidence out of range", &testClassifier{reply: Assessment{Output: contract.SafetyClassifierOutput{Level: contract.RiskNone, Confidence: 1.2}}}, "ordinary", "en", contract.RiskElevated},
		{"unreviewed locale", lowClassifier(), "ordinary", "de", contract.RiskElevated},
		{"oversized input", lowClassifier(), strings.Repeat("X", 65537), "en", contract.RiskElevated},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := testGate(t, tc.c, nil)
			sig, err := g.Assess(context.Background(), ports.SafetyInput{Text: tc.text, Lang: tc.locale})
			if err == nil || !sig.Degraded || sig.Level != tc.want || sig.Source != "timeout_default" {
				t.Fatalf("expected safe degraded: %+v %v", sig, err)
			}
		})
	}
}
func TestGateRecentUserSignalAndNeverAssistantText(t *testing.T) {
	g := testGate(t, lowClassifier(), nil)
	sig, err := g.Assess(context.Background(), ports.SafetyInput{Lang: "en", Text: "current", Recent: []contract.MsgView{{Role: contract.RoleAssistant, Text: "TEST_HIGH_SIGNAL"}, {Role: contract.RoleUser, Text: "TEST_HIGH_SIGNAL"}}})
	if err != nil || sig.Level != contract.RiskHigh || len(sig.Triggers) != 1 || sig.Triggers[0].Span != nil {
		t.Fatalf("recent safety signal: %+v %v", sig, err)
	}
}

type fixedLook struct {
	review SecondLookReview
	calls  int
}

func (r *fixedLook) Review(context.Context, ports.SafetyInput, []contract.SafetyTrigger, []contract.SafetyContextFlag) (SecondLookReview, error) {
	r.calls++
	return r.review, nil
}
func TestSecondLookRequiresVerifiedPerTriggerAndNoFirstPersonPresent(t *testing.T) {
	c := lowClassifier()
	c.reply.Output.ContextFlags = []contract.SafetyContextFlag{contract.ContextQuoted}
	look := &fixedLook{review: SecondLookReview{Flags: []contract.SafetyContextFlag{contract.ContextQuoted}, ContextualTriggers: []bool{true}, FirstPersonChecked: true}}
	g := testGate(t, c, look)
	in := ports.SafetyInput{Lang: "en", Text: "TEST_HIGH_SIGNAL"}
	sig, err := g.Assess(context.Background(), in)
	if err != nil || sig.Level != contract.RiskElevated {
		t.Fatalf("synthetic validated second look: %+v %v", sig, err)
	}
	look.review.FirstPersonPresent = true
	sig, _ = g.Assess(context.Background(), in)
	if sig.Level != contract.RiskHigh {
		t.Fatal("first-person-present downgraded")
	}
	look.review.FirstPersonPresent = false
	look.review.ContextualTriggers = nil
	sig, _ = g.Assess(context.Background(), in)
	if sig.Level != contract.RiskHigh {
		t.Fatal("missing per-trigger proof downgraded")
	}
	look.review.ContextualTriggers = []bool{true}
	look.review.FirstPersonChecked = false
	sig, _ = g.Assess(context.Background(), in)
	if sig.Level != contract.RiskHigh {
		t.Fatal("unverified first-person check downgraded")
	}
	look.review.FirstPersonChecked = true
	imminent, _ := g.Assess(context.Background(), ports.SafetyInput{Lang: "en", Text: "TEST_IMMINENT_SIGNAL"})
	if imminent.Level != contract.RiskImminent {
		t.Fatal("imminent signal downgraded")
	}
}
func TestGateCancellationDoesNotReturnNone(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sig, err := testGate(t, lowClassifier(), nil).Assess(ctx, ports.SafetyInput{Lang: "en"})
	if !errors.Is(err, context.Canceled) || !sig.Degraded || sig.Level != contract.RiskElevated {
		t.Fatalf("cancelled: %+v %v", sig, err)
	}
}
func TestLoadAssetsStrictAndDefaultRefusesSynthetic(t *testing.T) {
	b := syntheticBundle()
	if _, err := Load(context.Background(), b, LoadOptions{}); !errors.Is(err, ErrAssets) {
		t.Fatalf("synthetic authorized: %v", err)
	}
	a := loadFixture(t)
	ref := a.Reference()
	ref.PromptSHAs["prompts/safety_l1.system.tmpl"] = "mutated"
	if a.Reference().PromptSHAs["prompts/safety_l1.system.tmpl"] == "mutated" {
		t.Fatal("mutable reference")
	}
	for _, bad := range []struct{ path, value string }{
		{"lexicons/safety_l0.en.yaml", `{"schema":"lexicon.v1","version":"test","locale":"en","rules":[{"id":"bad","category":"other","level":"high","match":{"type":"regex","pattern":"("}}]}`},
		{"policy/safety_policy.yaml", `{"schema":"safety_policy.v1","risk_fsm":{"follow_up_hours":0}}`},
		{"resources/crisis_resources.yaml", `{"schema":"crisis_resources.v1","fallback_locale":"","stale_after_days":-1}`},
	} {
		b := syntheticBundle()
		b.files[bad.path] = bad.value
		if _, err := Load(context.Background(), b, LoadOptions{AllowSyntheticForTests: true}); err == nil {
			t.Fatalf("invalid asset loaded: %s", bad.path)
		}
	}
}
