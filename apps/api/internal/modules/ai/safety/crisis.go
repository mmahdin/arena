package safety

import (
	"context"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"sort"
	"strings"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// Planner reads only pinned, reviewed WP-03 assets; it never asks a model for
// crisis resources, telephone numbers, a response template, or escalation.
type Planner struct {
	assets *Assets
	now    func() time.Time
}

var _ ports.CrisisPlanner = (*Planner)(nil)

func NewPlanner(assets *Assets, clock func() time.Time) (*Planner, error) {
	if assets == nil {
		return nil, ErrAssets
	}
	if clock == nil {
		clock = time.Now
	}
	return &Planner{assets: assets, now: clock}, nil
}

func (p *Planner) Plan(ctx context.Context, sig contract.SafetySignal, locale string, state contract.RiskState) (result0 contract.CrisisPlan, result1 error) {
	ctx, inspectionSpan := trace.Start(ctx, "safety/crisis.Plan", func() any { return map[string]any{"sig": sig, "locale": locale, "state": state} })
	defer func() {
		inspectionSpan.Complete(func() any { return map[string]any{"result0": result0} }, result1, recover())
	}()

	if err := ctx.Err(); err != nil {
		return contract.CrisisPlan{}, err
	}
	if p == nil || p.assets == nil || (!sig.Level.Valid() && !state.Level.Valid()) {
		return contract.CrisisPlan{}, ErrCrisis
	}
	level := maxValid(sig.Level, state.Level)
	if level.Rank() < contract.RiskHigh.Rank() {
		return contract.CrisisPlan{}, ErrCrisis
	}
	// No family can be selected without an approved category: specifically
	// never invent a generic script for unknown or failed risk assessments.
	categories := append([]contract.RiskCategory(nil), sig.Categories...)
	sort.Slice(categories, func(i, j int) bool { return categories[i] < categories[j] })
	family := ""
	highest := -1
	for _, category := range categories {
		cfg, ok := p.assets.policy.Categories[string(category)]
		if !ok {
			continue
		}
		floor := contract.RiskLevel(cfg.EscalateTo).Rank()
		if floor > highest {
			family = cfg.TemplateFamily
			highest = floor
		}
	}
	if family == "" {
		return contract.CrisisPlan{}, ErrCrisis
	}
	cfg, ok := p.assets.policy.TemplateFamilies[family]
	if !ok || len(cfg.RequiredElements) == 0 || len(cfg.Forbidden) == 0 {
		return contract.CrisisPlan{}, ErrCrisis
	}
	script, ok := p.assets.templates[family]
	if !ok {
		return contract.CrisisPlan{}, ErrCrisis
	}
	text, ok := p.textForLocale(script, locale)
	if !ok {
		return contract.CrisisPlan{}, ErrCrisis
	}
	resources, general := p.resourceLookup(locale)
	if len(resources) == 0 {
		return contract.CrisisPlan{}, ErrCrisis
	}
	if general {
		// The warning is a separate REVIEWED, versioned, non-interpolated
		// template. Never synthesize locale/legal wording in code.
		notice, ok := p.assets.templates["resources_general_notice"]
		if !ok {
			return contract.CrisisPlan{}, ErrCrisis
		}
		warning, ok := p.textForLocale(notice, locale)
		if !ok {
			return contract.CrisisPlan{}, ErrCrisis
		}
		text = strings.TrimSpace(text) + "\n" + strings.TrimSpace(warning)
	}
	if err := ctx.Err(); err != nil {
		return contract.CrisisPlan{}, err
	}
	return contract.CrisisPlan{TemplateFamily: family,
		RequiredElements: append([]string(nil), cfg.RequiredElements...),
		Forbidden:        append([]string(nil), cfg.Forbidden...),
		Resources:        resources, TemplateText: text, Buffered: true}, nil
}

func (p *Planner) textForLocale(t template, locale string) (string, bool) {
	if loc := strings.TrimSpace(t.locales[locale]); loc != "" {
		return loc, true
	}
	if idx := strings.IndexByte(locale, '-'); idx > 0 {
		if text := strings.TrimSpace(t.locales[locale[:idx]]); text != "" {
			return text, true
		}
	}
	if locale == "en" {
		return strings.TrimSpace(t.text), strings.TrimSpace(t.text) != ""
	}
	return "", false
}

func (p *Planner) resourceLookup(locale string) ([]contract.CrisisResource, bool) {
	now := p.now().UTC()
	if p.assets.fullCheck.IsZero() || p.assets.fullCheck.After(now.Add(24*time.Hour)) ||
		now.Sub(p.assets.fullCheck) > time.Duration(p.assets.staleDays)*24*time.Hour {
		return nil, false
	}
	selectSet := func(loc string) []contract.CrisisResource {
		set, ok := p.assets.resources[loc]
		if !ok {
			return nil
		}
		valid := make([]contract.CrisisResource, 0, len(set.resources))
		for _, r := range set.resources {
			verified, err := time.Parse("2006-01-02", r.VerifiedOn)
			if err != nil || verified.After(now.Add(24*time.Hour)) || now.Sub(verified) > time.Duration(p.assets.staleDays)*24*time.Hour {
				continue
			}
			valid = append(valid, r)
		}
		return valid
	}
	if result := selectSet(locale); len(result) > 0 {
		return result, false
	}
	// No cross-region assumption; the curated international_directory or
	// fallback locale is selected exactly as specified by the bundle.
	if result := selectSet(p.assets.fallback); len(result) > 0 {
		return result, true
	}
	return nil, false
}

// StaticCrisisFallback is for the *buffered*, validated crisis route only. It
// does not bypass WP-07 Validator and must never release an unvalidated draft.
// Resource contacts live in CrisisPlan.Resources/SafetyBanner, not model text.
func StaticCrisisFallback(plan contract.CrisisPlan) (string, bool) {
	if !plan.Buffered || len(plan.Resources) == 0 || len(plan.RequiredElements) == 0 || strings.TrimSpace(plan.TemplateText) == "" ||
		hasInterpolation(plan.TemplateText) || strings.Contains(plan.TemplateText, "SYNTHETIC PLACEHOLDER") {
		return "", false
	}
	for _, k := range []string{"acknowledge", "care", "ask_safety", "resources", "invite_continue"} {
		if !hasString(plan.RequiredElements, k) {
			return "", false
		}
	}
	return plan.TemplateText, true
}

// FallbackFunc adapts the existing WP-05 signature for normal/fast messages.
// Crisis fallback requires its specific CrisisPlan, so this adapter declines
// CRISIS rather than presenting the wrong category or unverified contact.
func (a *Assets) FallbackFunc(route contract.Route, objective contract.Objective, locale string) (string, bool) {
	if a == nil || route == contract.RouteCrisis || objective == contract.ObjSafetySupport || (route != contract.RouteNormal && route != contract.RouteFast) {
		return "", false
	}
	t, ok := a.fallbacks[objective]
	if !ok || strings.Contains(t.text, "SYNTHETIC PLACEHOLDER") {
		return "", false
	}
	p := Planner{assets: a}
	return p.textForLocale(t, locale)
}
