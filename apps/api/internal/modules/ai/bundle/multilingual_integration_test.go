package bundle_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/safety"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/validator"
)

type multilingualClassifier struct{ fail bool }

func (m multilingualClassifier) Classify(context.Context, ports.SafetyInput) (safety.Assessment, error) {
	if m.fail {
		return safety.Assessment{}, errors.New("synthetic outage")
	}
	return safety.Assessment{Version: "synthetic", Output: c.SafetyClassifierOutput{Level: c.RiskNone, Confidence: 1}}, nil
}
func multilingualBundle(t *testing.T) *bundle.Snapshot {
	t.Helper()
	b, report, err := bundle.Read(context.Background(), filepath.Join("..", "..", "..", "..", "..", "..", "knowledge"), bundle.Options{})
	if err != nil || !report.OK() {
		t.Fatalf("bundle: %v %+v", err, report.Errors)
	}
	return b
}
func TestPersianSafetyCrisisAndValidationWithEnglishKnowledge(t *testing.T) {
	ctx := context.Background()
	b := multilingualBundle(t)
	a, err := safety.Load(ctx, b, safety.LoadOptions{AllowSyntheticForTests: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, fail := range []bool{false, true} {
		gate, err := safety.NewGate(a, multilingualClassifier{fail}, nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct {
			text, category string
			level          c.RiskLevel
		}{
			{"می‌خواهم خودم را بکشم", "suicidal_ideation", c.RiskHigh},
			{"مي‌خواهم بميرم", "suicidal_ideation", c.RiskHigh},
			{"کاش نبودم", "suicidal_ideation", c.RiskHigh},
			{"به خودم آسیب بزنم", "self_harm", c.RiskHigh},
			{"می‌خوام اونو بکشم", "harm_to_others", c.RiskHigh},
			{"منو می‌زنه", "abuse_disclosure", c.RiskHigh},
			{"نمی‌توانم نفس بکشم", "medical_emergency", c.RiskImminent},
			{"همه قرص‌ها را خوردم", "substance_crisis", c.RiskImminent},
		} {
			// The English input hint must never suppress Persian L0 evidence.
			sig, err := gate.Assess(ctx, ports.SafetyInput{Text: tc.text, Lang: "en"})
			if sig.Level.Rank() < tc.level.Rank() || len(sig.Categories) == 0 || string(sig.Categories[0]) != tc.category || fail && err == nil {
				t.Fatalf("%q: %+v %v", tc.text, sig, err)
			}
			if len(sig.Triggers) == 0 || sig.Triggers[0].Span == nil || sig.Triggers[0].Span.End > len([]rune(tc.text)) {
				t.Fatal("invalid normalized rune span")
			}
		}
	}
	gate, _ := safety.NewGate(a, multilingualClassifier{}, nil)
	for _, text := range []string{"خودکشی یعنی چه؟ برای یک متن آموزشی می پرسم.", "اوردوز یعنی چه؟", "خودزنی چه معنایی دارد؟"} {
		sig, err := gate.Assess(ctx, ports.SafetyInput{Text: text, Lang: "fa"})
		if err != nil || sig.Level != c.RiskNone {
			t.Fatal("informational term forced crisis", sig, err)
		}
	}
	sig, err := gate.Assess(ctx, ports.SafetyInput{Text: "امروز روز سختی بود", Lang: "fa"})
	if err != nil || sig.Level != c.RiskNone {
		t.Fatal("neutral text escalated", sig, err)
	}
	sig, err = gate.Assess(ctx, ports.SafetyInput{Text: "hello", Lang: "xx"})
	if !errors.Is(err, safety.ErrLocale) || !sig.Degraded || sig.Level.Rank() < c.RiskElevated.Rank() {
		t.Fatal("missing input coverage bypassed")
	}
	planner, _ := safety.NewPlanner(a, func() time.Time { return time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC) })
	v, err := validator.New(ctx, b, validator.Options{AllowSyntheticForTests: true})
	if err != nil {
		t.Fatal(err)
	}
	wc := c.WorkingContext{Turn: c.TurnView{Lang: "en"}, Prefs: c.PrefsView{Language: "fa"}, Eligibility: c.Eligibility{QuestionAllowed: true}}
	plan := c.ResponsePlan{Question: c.QuestionPlan{Allowed: true, MaxCount: 1}}
	for _, category := range []c.RiskCategory{c.RiskSuicidalIdeation, c.RiskAbuseDisclosure, c.RiskMedicalEmergency, c.RiskSubstanceCrisis} {
		cp, err := planner.Plan(ctx, c.SafetySignal{Level: c.RiskHigh, Categories: []c.RiskCategory{category}}, "fa", c.RiskState{Level: c.RiskHigh})
		if err != nil || !strings.Contains(cp.TemplateText, "امن") {
			t.Fatal("Persian crisis unavailable", cp, err)
		}
		text, ok := safety.StaticCrisisFallback(cp)
		if !ok {
			t.Fatal("missing static crisis")
		}
		report, err := v.Finalize(ctx, text, plan, wc, c.RiskHigh, &cp)
		if err != nil || !report.Passed {
			t.Fatalf("Persian crisis rejected: %v %+v", err, report)
		}
	}
}

func TestPersianOutputRulesAndMixedLanguageBackstops(t *testing.T) {
	ctx := context.Background()
	b := multilingualBundle(t)
	v, err := validator.New(ctx, b, validator.Options{AllowSyntheticForTests: true})
	if err != nil {
		t.Fatal(err)
	}
	wc := c.WorkingContext{Turn: c.TurnView{Lang: "en"}, Prefs: c.PrefsView{Language: "fa"}, Eligibility: c.Eligibility{QuestionAllowed: true, MemoryAvailable: true}}
	plan := c.ResponsePlan{Question: c.QuestionPlan{Allowed: true, MaxCount: 1}, MustAvoid: []string{"advice", "intervention"}}
	for _, tc := range []struct{ text, code string }{
		{"تو افسردگی داری.", "OV-003"}, {"دارويت را قطع كن.", "OV-004"}, {"۱۲ قرص بخور.", "OV-004"},
		{"تضمین می‌کنم خوب می‌شوی.", "OV-005"}, {"من درمانگر تو هستم.", "OV-006"},
		{"باید استعفا بدهی.", "OV-007"}, {"شناسه حافظه: چیزی", "OV-010"},
		{"با ۰۹۱۲۳۴۵۶۷۸۹ تماس بگیر.", "OV-011"}, {"با ٠٩١٢٣٤٥٦٧٨٩ تماس بگیر.", "OV-011"},
		{"بیا تمرین تنفس انجام بده.", "OV-015"}, {"فقط من تو را می‌فهمم.", "OV-016"},
		{"فقط فراموشش کن.", "OV-017"}, {"چطوری؟ چه شده؟", "OV-001"},
		{"این جمله است. You have depression.", "OV-003"},
		{"This is an English response with one word: سلام.", "OV-013"},
	} {
		report, err := v.Finalize(ctx, tc.text, plan, wc, c.RiskLow, nil)
		found := false
		for _, violation := range report.Violations {
			if violation.Code == tc.code {
				found = true
			}
		}
		if err != nil || report.Passed && tc.code != "OV-001" || !found {
			t.Errorf("%s: %q %v %+v", tc.code, tc.text, err, report)
		}
	}
	report, err := v.Finalize(ctx, "به نظر می‌رسد امروز فشار زیادی را تحمل کرده‌ای. اگر بخواهی، می‌توانیم درباره‌اش حرف بزنیم.", plan, wc, c.RiskLow, nil)
	if err != nil || !report.Passed {
		t.Fatal("natural Persian rejected", report, err)
	}
	guard := v.StreamGuard(plan, c.RiskLow)
	if ok, _ := guard.Push("تو افسردگی "); !ok {
		t.Fatal("premature block")
	}
	if ok, violation := guard.Push("داری."); ok || violation == nil || violation.Code != "OV-003" {
		t.Fatal("Persian stream escaped", ok, violation)
	}
}

type missingLanguageArtifact struct {
	*bundle.Snapshot
	missing string
}

func (b missingLanguageArtifact) Artifact(ctx context.Context, name string) ([]byte, error) {
	if name == b.missing {
		return nil, errors.New("synthetic missing asset")
	}
	return b.Snapshot.Artifact(ctx, name)
}
func TestRuntimeLanguageCoverageNeverSilentlyBypassesGates(t *testing.T) {
	ctx := context.Background()
	b := multilingualBundle(t)
	for _, path := range []string{"lexicons/safety_l0.fa.yaml", "lexicons/validator.fa.yaml"} {
		missing := missingLanguageArtifact{b, path}
		if strings.Contains(path, "safety") {
			if _, err := safety.Load(ctx, missing, safety.LoadOptions{AllowSyntheticForTests: true}); err == nil {
				t.Fatal("missing Persian L0 accepted")
			}
		} else if _, err := validator.New(ctx, missing, validator.Options{AllowSyntheticForTests: true}); err == nil {
			t.Fatal("missing Persian Validator accepted")
		}
	}
}
