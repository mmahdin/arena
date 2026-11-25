package validator_test

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/validator"
	"gopkg.in/yaml.v3"
)

type syntheticBundle struct {
	files     map[string][]byte
	version   string
	synthetic bool
}

func (b syntheticBundle) Artifact(ctx context.Context, name string) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	raw, ok := b.files[name]
	if !ok {
		return nil, errors.New("not found")
	}
	return append([]byte(nil), raw...), nil
}
func (b syntheticBundle) Reference() c.BundleRef {
	return c.BundleRef{Version: b.version, PromptSHAs: map[string]string{"prompts/generator.system.tmpl": "fixture"}}
}
func (b syntheticBundle) SyntheticOnly() bool { return b.synthetic }

func fixtureBundle(t *testing.T) syntheticBundle {
	t.Helper()
	lex, err := os.ReadFile("testdata/validator.en.yaml")
	if err != nil {
		t.Fatal(err)
	}
	return syntheticBundle{version: "kb@0.0.1", synthetic: true, files: map[string][]byte{
		"bundle.yaml":                     []byte("version: 0.0.1\nlocales: [en]\nsynthetic_only: true\n"),
		"lexicons/validator.en.yaml":      lex,
		"resources/crisis_resources.yaml": []byte("schema: crisis_resources.v1\nfallback_locale: en\nlocales:\n  en:\n    - {id: test_988, name: reviewed_fixture, kind: hotline, contact: '988', verified_on: '2026-11-16'}\n    - {id: approved_url, name: fixture_web, kind: chat, contact: 'https://approved.example/help', verified_on: '2026-11-16'}\n"),
		"config/thresholds.yaml":          []byte("schema: thresholds.v1\nvalidator:\n  directive_advice: 0.8\n  certainty_about_inference: 0.8\n  dependency_encouraging: 0.8\n  invalidating: 0.8\n"),
	}}
}
func svc(t *testing.T, opt validator.Options) *validator.Service {
	t.Helper()
	opt.AllowSyntheticForTests = true
	s, err := validator.New(context.Background(), fixtureBundle(t), opt)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func base() (c.WorkingContext, c.ResponsePlan) {
	wc := c.WorkingContext{Turn: c.TurnView{Lang: "en"}, Prefs: c.PrefsView{Language: "en"}, Eligibility: c.Eligibility{QuestionAllowed: true, MemoryAvailable: true}}
	plan := c.ResponsePlan{Length: c.LengthShort, Question: c.QuestionPlan{Allowed: false, MaxCount: 0}}
	return wc, plan
}
func hasCode(r c.ValidationReport, code string) bool {
	for _, v := range r.Violations {
		if v.Code == code {
			return true
		}
	}
	return false
}

type golden struct {
	Name              string `yaml:"name"`
	Advisory          bool   `yaml:"advisory"`
	Text              string `yaml:"text"`
	Code              string `yaml:"code"`
	QuestionAllowed   bool   `yaml:"question_allowed"`
	AvoidAdvice       bool   `yaml:"avoid_advice"`
	Inferred          bool   `yaml:"inferred"`
	MemoryUnpermitted bool   `yaml:"memory_unpermitted"`
	Protocol          bool   `yaml:"protocol"`
	Length            string `yaml:"length"`
	Risk              string `yaml:"risk"`
}

func TestRuleRegistryGoldens(t *testing.T) {
	raw, err := os.ReadFile("testdata/golden/rules.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []golden
	if err = yaml.Unmarshal(raw, &fixtures); err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	v := svc(t, validator.Options{})
	for _, tc := range fixtures {
		t.Run(tc.Name, func(t *testing.T) {
			wc, plan := base()
			plan.Question.Allowed = tc.QuestionAllowed
			if tc.QuestionAllowed {
				plan.Question.MaxCount = 1
			}
			if tc.AvoidAdvice {
				plan.MustAvoid = []string{"advice"}
			}
			if tc.Length != "" {
				plan.Length = c.Length(tc.Length)
			}
			if tc.Protocol {
				plan.Protocol = &c.ProtocolDirective{}
			}
			if tc.Inferred || tc.MemoryUnpermitted {
				id := c.NewID()
				wc.Retrieved = []c.MemoryView{{ID: id, Epistemic: c.EpInferred, Text: "avoid social events"}}
				if !tc.MemoryUnpermitted {
					plan.PermittedContext = []c.ID{id}
				}
			}
			risk := c.RiskLow
			if tc.Risk == "high" {
				risk = c.RiskHigh
			}
			ctx := context.Background()
			if tc.Protocol {
				ctx = validator.WithProtocolRequirements(ctx, &c.ProtocolStepSpec{MustSay: []string{"fixed_disclosure"}}, map[string]string{"fixed_disclosure": "EXPLICIT FIXED DISCLOSURE"})
			}
			if tc.Code == "OV-015" {
				plan.MustAvoid = append(plan.MustAvoid, "intervention")
			}
			got, err := v.Finalize(ctx, tc.Text, plan, wc, risk, nil)
			if err != nil {
				t.Fatal(err)
			}
			if !hasCode(got, tc.Code) {
				t.Fatalf("want %s, got %+v", tc.Code, got)
			}
			if got.Passed != tc.Advisory {
				t.Fatal("incorrect safety/style handling")
			}
			if tc.Code == "OV-030" && (got.Action != c.VACrisis || got.Extracted.Sentinel != "PLAN_UNSAFE") {
				t.Fatalf("sentinel handling: %+v", got)
			}
			found[tc.Code] = true
		})
	}
	for _, code := range []string{"OV-001", "OV-002", "OV-003", "OV-004", "OV-005", "OV-006", "OV-007", "OV-008", "OV-009", "OV-010", "OV-011", "OV-012", "OV-013", "OV-014", "OV-015", "OV-016", "OV-017", "OV-020", "OV-021", "OV-030", "OV-031"} {
		if !found[code] {
			t.Fatalf("missing golden for %s", code)
		}
	}
}
func TestPassingAndWarningCases(t *testing.T) {
	v := svc(t, validator.Options{})
	wc, plan := base()
	for _, text := range []string{"That sounds difficult, and I'm listening.", "Thank you for telling me."} {
		report, err := v.Finalize(context.Background(), text, plan, wc, c.RiskLow, nil)
		if err != nil || !report.Passed || report.Action != c.VAAccept {
			t.Fatalf("unexpected failure: %+v %v", report, err)
		}
	}
	plan.Question = c.QuestionPlan{Allowed: true, MaxCount: 1}
	report, err := v.Finalize(context.Background(), "Would talking help?", plan, wc, c.RiskLow, nil)
	if err != nil || !report.Passed || report.Extracted.QuestionCount != 1 || *report.Extracted.QuestionText != "Would talking help?" {
		t.Fatalf("question metadata incorrect: %+v %v", report, err)
	}
	report, err = v.Finalize(context.Background(), `You wrote "Why am I sad?".`, plan, wc, c.RiskLow, nil)
	if err != nil || !report.Passed || report.Extracted.QuestionCount != 0 {
		t.Fatalf("quoted question should not count: %+v %v", report, err)
	}
	report, err = v.Finalize(validator.WithFinishReason(context.Background(), "length"), "This was cut off", plan, wc, c.RiskLow, nil)
	if err != nil || !hasCode(report, "OV-031") {
		t.Fatalf("truncation missing: %+v %v", report, err)
	}
}
func TestInferenceAndProvenance(t *testing.T) {
	v := svc(t, validator.Options{})
	wc, plan := base()
	id := c.NewID()
	wc.Retrieved = []c.MemoryView{{ID: id, Text: "avoids social events", Epistemic: c.EpInferred}}
	plan.PermittedContext = []c.ID{id}
	r, err := v.Finalize(context.Background(), "It may be that you avoid social events.", plan, wc, c.RiskLow, nil)
	if err != nil || !r.Passed || len(r.Extracted.ReferencedContext) != 1 || r.Extracted.ReferencedContext[0] != id || len(r.Extracted.ClaimsAboutUser) != 1 || !r.Extracted.ClaimsAboutUser[0].Hedged {
		t.Fatalf("hedged source missed: %+v %v", r, err)
	}
	wc.RetrievalMeta.Degraded = true
	r, err = v.Finalize(context.Background(), "I remember you told me before about this.", plan, wc, c.RiskLow, nil)
	if err != nil || !hasCode(r, "OV-009") {
		t.Fatalf("degraded recall not blocked: %+v %v", r, err)
	}
}
func TestCrisisTemplateAndContacts(t *testing.T) {
	v := svc(t, validator.Options{})
	wc, plan := base()
	cp := c.CrisisPlan{Buffered: true, TemplateText: "I hear you and care about your safety. Are you safe? Please call 988. I'm here if you want to continue.", RequiredElements: []string{"acknowledge", "care", "ask_safety", "resources", "invite_continue"}, Resources: []c.CrisisResource{{ID: "test_988", Contact: "988", Locale: "en"}}}
	plan.Question.Allowed = true
	plan.Question.MaxCount = 1
	r, err := v.Finalize(context.Background(), cp.TemplateText, plan, wc, c.RiskHigh, &cp)
	if err != nil || !r.Passed {
		t.Fatalf("reviewed static crisis template failed: %+v %v", r, err)
	}
	cp.Resources[0].Contact = "911"
	r, err = v.Finalize(context.Background(), cp.TemplateText, plan, wc, c.RiskHigh, &cp)
	if err != nil || !hasCode(r, "OV-020") {
		t.Fatalf("unreviewed resource permitted: %+v %v", r, err)
	}
	cp.Resources[0].Contact = "988"
	cp.TemplateText = "I hear you. See https://unsafe.example instead."
	r, err = v.Finalize(context.Background(), cp.TemplateText, plan, wc, c.RiskHigh, &cp)
	if err != nil || !hasCode(r, "OV-011") || !hasCode(r, "OV-020") {
		t.Fatalf("unsafe resource or required elements missed: %+v %v", r, err)
	}
}
func TestStreamGuardAndContext(t *testing.T) {
	v := svc(t, validator.Options{})
	_, plan := base()
	g := v.StreamGuard(plan, c.RiskLow)
	for _, s := range []string{"You ", "have ", "depression."} {
		ok, violation := g.Push(s)
		if strings.HasSuffix(s, "depression.") {
			if ok || violation == nil || violation.Code != "OV-003" {
				t.Fatalf("unsafe stream not stopped: %v %+v", ok, violation)
			}
		} else if !ok {
			t.Fatal("premature stream reject")
		}
	}
	g = v.StreamGuard(plan, c.RiskLow)
	if ok, _ := g.Push("<<PLAN_UNSAFE>>"); ok {
		t.Fatal("sentinel escaped")
	}
}
func TestFailClosedAssetAndLocale(t *testing.T) {
	b := fixtureBundle(t)
	if _, err := validator.New(context.Background(), b, validator.Options{}); !errors.Is(err, validator.ErrAssets) {
		t.Fatalf("synthetic bundle loaded in production: %v", err)
	}
	v := svc(t, validator.Options{})
	wc, plan := base()
	wc.Turn.Lang = "fa"
	wc.Prefs.Language = "fa"
	if _, err := v.Finalize(context.Background(), "سلام", plan, wc, c.RiskLow, nil); !errors.Is(err, validator.ErrCoverage) {
		t.Fatalf("unsupported language accepted: %v", err)
	}
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := v.Finalize(cancelCtx, "Hello.", plan, wc, c.RiskLow, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation not respected: %v", err)
	}
	b.files["lexicons/validator.en.yaml"] = []byte("schema: validator_lexicon.v1\nlocale: en\n")
	if _, err := validator.New(context.Background(), b, validator.Options{AllowSyntheticForTests: true}); !errors.Is(err, validator.ErrAssets) {
		t.Fatalf("empty clinical rules accepted: %v", err)
	}
}

type scoreClassifier struct {
	scores map[string]float32
	err    error
}

func (s scoreClassifier) Scores(context.Context, string, []c.MsgView, string) (map[string]float32, error) {
	return s.scores, s.err
}
func TestClassifierThresholdsAndFailure(t *testing.T) {
	scores := map[string]float32{"directive_advice": 0.99, "certainty_about_inference": 0.03, "dependency_encouraging": 0.03, "invalidating": 0.03}
	v := svc(t, validator.Options{Classifier: scoreClassifier{scores: scores}})
	wc, plan := base()
	plan.MustAvoid = []string{"advice"}
	report, err := v.Finalize(context.Background(), "I hear you.", plan, wc, c.RiskLow, nil)
	if err != nil || !hasCode(report, "OV-007") || report.Passed {
		t.Fatalf("classifier not applied: %+v %v", report, err)
	}
	plan.MustAvoid = nil
	report, err = v.Finalize(context.Background(), "I hear you.", plan, wc, c.RiskLow, nil)
	if err != nil || !report.Passed || !reflect.DeepEqual(report.Scores, scores) {
		t.Fatalf("unexpected classifier result: %+v %v", report, err)
	}
	v = svc(t, validator.Options{Classifier: scoreClassifier{err: errors.New("unavailable")}})
	if _, err = v.Finalize(context.Background(), "I hear you.", plan, wc, c.RiskLow, nil); !errors.Is(err, validator.ErrClassification) {
		t.Fatalf("classifier failure ignored: %v", err)
	}
}
func TestProtocolRequirementBinding(t *testing.T) {
	v := svc(t, validator.Options{})
	wc, plan := base()
	plan.Protocol = &c.ProtocolDirective{}
	spec := &c.ProtocolStepSpec{MustSay: []string{"opt_out"}}
	ctx := validator.WithProtocolRequirements(context.Background(), spec, map[string]string{"opt_out": "You can opt out."})
	r, err := v.Finalize(ctx, "You can opt out.", plan, wc, c.RiskLow, nil)
	if err != nil || !r.Passed {
		t.Fatalf("reviewed mandatory line missing: %+v %v", r, err)
	}
}

func TestStreamGuardIncrementalApprovedContact(t *testing.T) {
	v := svc(t, validator.Options{})
	_, plan := base()
	g := v.StreamGuard(plan, c.RiskLow)
	for _, token := range []string{"Please visit https://approved.ex", "ample/help", " when needed."} {
		if ok, violation := g.Push(token); !ok {
			t.Fatalf("approved contact was rejected on a partial stream: %+v", violation)
		}
	}
	g = v.StreamGuard(plan, c.RiskLow)
	if ok, _ := g.Push("Please visit https://unapproved.ex"); ok {
		// This contact cannot be a prefix of any approved resource and can
		// safely be rejected even before receiving trailing whitespace.
		if ok, _ = g.Push("ample/help "); ok {
			t.Fatal("unapproved completed contact passed streaming guard")
		}
	}
	wc, plan := base()
	if got, err := v.Finalize(context.Background(), "Approved: https://approved.example/help", plan, wc, c.RiskLow, nil); err != nil || !got.Passed {
		t.Fatalf("approved final contact was blocked: %+v %v", got, err)
	}
}

func TestExplicitResponseLanguageOverridesTurnLanguage(t *testing.T) {
	v := svc(t, validator.Options{})
	wc, plan := base()
	wc.Turn.Lang = "fa"
	wc.Prefs.Language = "en"
	accepted, err := v.Finalize(context.Background(), "Thank you for telling me.", plan, wc, c.RiskLow, nil)
	if err != nil || !accepted.Passed || hasCode(accepted, "OV-013") {
		t.Fatalf("explicit English response preference rejected: %+v %v", accepted, err)
	}
	rejected, err := v.Finalize(context.Background(), "سلام امروز اینجا هستم", plan, wc, c.RiskLow, nil)
	if err != nil || !hasCode(rejected, "OV-013") || rejected.Passed {
		t.Fatalf("reply ignoring explicit response language accepted: %+v %v", rejected, err)
	}
}

func TestLengthWithinToleranceAndLowWarning(t *testing.T) {
	v := svc(t, validator.Options{})
	wc, plan := base()
	plan.Length = c.LengthMinimal
	short, err := v.Finalize(context.Background(), "Thank you for telling me.", plan, wc, c.RiskLow, nil)
	if err != nil || !short.Passed || !hasCode(short, "OV-012") || short.Action != c.VAAccept {
		t.Fatalf("underlength warning should pass: %+v %v", short, err)
	}
}
