package safety

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

func planFor(t *testing.T, a *Assets, locale string) (contract.CrisisPlan, error) {
	t.Helper()
	p, err := NewPlanner(a, fixedTime)
	if err != nil {
		t.Fatal(err)
	}
	return p.Plan(context.Background(), contract.SafetySignal{Level: contract.RiskHigh, Categories: []contract.RiskCategory{contract.RiskOther}}, locale, contract.RiskState{Level: contract.RiskElevated})
}
func TestCrisisPlannerLocaleResourcesAndStaticFallback(t *testing.T) {
	a := loadFixture(t)
	p, err := planFor(t, a, "en")
	if err != nil || !p.Buffered || p.TemplateFamily != "synthetic_case" || len(p.RequiredElements) != 5 || len(p.Forbidden) != 2 || len(p.Resources) != 1 || p.Resources[0].Contact != "TEST-ONLY-CONTACT" || p.TemplateText != "SYNTHETIC TEST CRISIS SCRIPT (not for people)" {
		t.Fatalf("crisis plan: %+v %v", p, err)
	}
	if text, ok := StaticCrisisFallback(p); !ok || text != p.TemplateText {
		t.Fatal("static crisis fallback not available for verified plan")
	}
	if text, ok := a.FallbackFunc(contract.RouteCrisis, contract.ObjSafetySupport, "en"); ok || text != "" {
		t.Fatal("crisis improperly used generic fallback")
	}
	if text, ok := a.FallbackFunc(contract.RouteNormal, contract.ObjReflect, "fa"); !ok || text != "SYNTHETIC FA FALLBACK" {
		t.Fatal("normal fallback locale mismatch")
	}
}
func TestCrisisPlannerMissingLocaleUsesCuratedFallbackAndReviewedNotice(t *testing.T) {
	p, err := planFor(t, loadFixture(t), "fa")
	if err != nil || !p.Buffered || len(p.Resources) != 1 || p.Resources[0].Locale != "en" ||
		!strings.Contains(p.TemplateText, "SYNTHETIC FA GENERAL NOTICE") {
		t.Fatalf("fallback locale: %+v %v", p, err)
	}
}
func TestCrisisPlannerFailsClosedWhenAssetsUnavailableOrStale(t *testing.T) {
	for _, mod := range []struct {
		name string
		edit func(*Assets)
	}{
		{"missing resources", func(a *Assets) { a.resources = map[string]resourceSet{} }},
		{"stale verification", func(a *Assets) { a.fullCheck = fixedTime().Add(-190 * 24 * time.Hour) }},
		{"future verification", func(a *Assets) { a.fullCheck = fixedTime().Add(3 * 24 * time.Hour) }},
		{"no templates", func(a *Assets) { a.templates = map[string]template{} }},
		{"missing general notice", func(a *Assets) { delete(a.templates, "resources_general_notice") }},
	} {
		t.Run(mod.name, func(t *testing.T) {
			a := loadFixture(t)
			mod.edit(a)
			locale := "en"
			if mod.name == "missing general notice" {
				locale = "fa"
			}
			p, err := planFor(t, a, locale)
			if !errors.Is(err, ErrCrisis) || p.TemplateText != "" || len(p.Resources) != 0 {
				t.Fatalf("unsafe success: %+v %v", p, err)
			}
		})
	}
	p, err := planFor(t, loadFixture(t), "unknown")
	if !errors.Is(err, ErrCrisis) || len(p.Resources) != 0 {
		t.Fatalf("unreviewed locale allowed: %+v %v", p, err)
	}
}
func TestCrisisPlannerRejectsUnknownCategoryAndUnmatchedRisk(t *testing.T) {
	a := loadFixture(t)
	p, _ := NewPlanner(a, fixedTime)
	for _, sig := range []contract.SafetySignal{
		{Level: contract.RiskHigh},
		{Level: contract.RiskHigh, Categories: []contract.RiskCategory{contract.RiskSelfHarm}},
		{Level: contract.RiskLow, Categories: []contract.RiskCategory{contract.RiskOther}},
	} {
		got, err := p.Plan(context.Background(), sig, "en", contract.RiskState{Level: contract.RiskLow})
		if !errors.Is(err, ErrCrisis) || len(got.Resources) > 0 {
			t.Fatalf("accepted unsafe plan: %+v %v", got, err)
		}
	}
}
func TestStaticCrisisFallbackNeverBypassesPlanChecks(t *testing.T) {
	for _, plan := range []contract.CrisisPlan{
		{Buffered: false, TemplateText: "test", Resources: []contract.CrisisResource{{ID: "test"}}, RequiredElements: []string{"acknowledge"}},
		{Buffered: true, TemplateText: "test", Resources: nil, RequiredElements: []string{"acknowledge"}},
		{Buffered: true, TemplateText: "{{user.text}}", Resources: []contract.CrisisResource{{ID: "test"}}, RequiredElements: []string{"acknowledge", "care", "ask_safety", "resources", "invite_continue"}},
	} {
		if _, ok := StaticCrisisFallback(plan); ok {
			t.Fatal("unvalidated fallback accepted")
		}
	}
}
