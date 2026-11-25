// Package safety implements the version-pinned WP-06 safety pre-gate, crisis
// planning and risk transition rules. It never imports the prototype AI path.
package safety

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"gopkg.in/yaml.v3"
)

var (
	ErrAssets     = errors.New("safety: bundle assets unavailable or unreviewed")
	ErrLocale     = errors.New("safety: locale has no approved safety coverage")
	ErrClassifier = errors.New("safety: classifier unavailable or invalid")
	ErrCrisis     = errors.New("safety: reviewed crisis plan unavailable")
)

// LoadOptions.AllowSyntheticForTests is ONLY for synthetic fixtures, never
// production requests or real mental-health disclosures. A production caller
// must pass a non-synthetic, release-verified WP-03 snapshot.
type LoadOptions struct{ AllowSyntheticForTests bool }

type lexRule struct {
	id         string
	category   contract.RiskCategory
	level      contract.RiskLevel
	pattern    *regexp.Regexp
	literal    string
	secondLook bool
}

type resourceSet struct {
	resources []contract.CrisisResource
}

type template struct {
	text    string
	locales map[string]string
}

// Assets is immutable after Load and bound to one pinned WP-03 BundleRef.
// Its fields cannot be mutated by port consumers.
type Assets struct {
	ref                  contract.BundleRef
	locales              map[string][]lexRule
	lexVersion           map[string]string
	policy               bundle.SafetyPolicy
	resources            map[string]resourceSet
	fallback             string
	staleDays            int
	fullCheck            time.Time
	templates            map[string]template
	fallbacks            map[contract.Objective]template
	prompt               string
	requireSafeStatement bool
}

func (a *Assets) Reference() contract.BundleRef {
	if a == nil {
		return contract.BundleRef{}
	}
	ref := contract.BundleRef{Version: a.ref.Version, PromptSHAs: map[string]string{}}
	for k, v := range a.ref.PromptSHAs {
		ref.PromptSHAs[k] = v
	}
	return ref
}

func readYAML(ctx context.Context, b ports.Bundle, name string, dest any) error {
	data, err := b.Artifact(ctx, name)
	if err != nil || len(data) == 0 || len(data) > 256<<10 || !utf8.Valid(data) {
		return ErrAssets
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(dest); err != nil {
		return fmt.Errorf("%w: invalid %s", ErrAssets, name)
	}
	var rest any
	if err := decoder.Decode(&rest); err != io.EOF {
		return fmt.Errorf("%w: multiple documents in %s", ErrAssets, name)
	}
	return nil
}

// Load pins reviewed assets. It intentionally does NOT substitute default
// lexicons, clinical rules, contact details, templates, or other locale text.
// WP-03's release-verified registry is the authority for clinical sign-off;
// this loader adds runtime sanity checks and rejects synthetics by default.
func Load(ctx context.Context, source ports.Bundle, opts LoadOptions) (*Assets, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if source == nil {
		return nil, ErrAssets
	}
	review, ok := source.(interface{ SyntheticOnly() bool })
	if (!ok || review.SyntheticOnly()) && !opts.AllowSyntheticForTests {
		return nil, ErrAssets
	}
	ref := source.Reference()
	if !strings.HasPrefix(ref.Version, "kb@") {
		return nil, ErrAssets
	}
	var manifest bundle.Manifest
	if err := readYAML(ctx, source, "bundle.yaml", &manifest); err != nil || manifest.ValidateLanguages() != nil {
		return nil, ErrAssets
	}
	if manifest.SyntheticOnly && !opts.AllowSyntheticForTests {
		return nil, ErrAssets
	}

	a := &Assets{ref: ref, locales: map[string][]lexRule{}, lexVersion: map[string]string{}, resources: map[string]resourceSet{},
		templates: map[string]template{}, fallbacks: map[contract.Objective]template{}}
	if err := readYAML(ctx, source, "policy/safety_policy.yaml", &a.policy); err != nil || a.policy.Schema != "safety_policy.v1" {
		return nil, ErrAssets
	}
	if !validPolicy(a.policy) {
		return nil, ErrAssets
	}
	var thresholds bundle.Thresholds
	if err := readYAML(ctx, source, "config/thresholds.yaml", &thresholds); err != nil || thresholds.Schema != "thresholds.v1" || !thresholds.Safety.Deescalate.RequireUserStatement {
		return nil, ErrAssets
	}
	a.requireSafeStatement = thresholds.Safety.Deescalate.RequireUserStatement
	var resources bundle.CrisisResources
	if err := readYAML(ctx, source, "resources/crisis_resources.yaml", &resources); err != nil || resources.Schema != "crisis_resources.v1" || resources.StaleAfterDays <= 0 {
		return nil, ErrAssets
	}
	a.fallback, a.staleDays = resources.FallbackLocale, resources.StaleAfterDays
	if resources.LastFullVerification != "" {
		var err error
		a.fullCheck, err = time.Parse("2006-01-02", resources.LastFullVerification)
		if err != nil {
			return nil, ErrAssets
		}
	}
	for locale, items := range resources.Locales {
		if locale == "" {
			return nil, ErrAssets
		}
		set := resourceSet{}
		ids := map[string]bool{}
		for _, item := range items {
			if item.ID == "" || ids[item.ID] || item.Name == "" || item.Contact == "" || !validResourceKind(item.Kind) {
				return nil, ErrAssets
			}
			ids[item.ID] = true
			if _, err := time.Parse("2006-01-02", item.VerifiedOn); err != nil {
				return nil, ErrAssets
			}
			set.resources = append(set.resources, contract.CrisisResource{ID: item.ID, Name: item.Name, Kind: item.Kind, Contact: item.Contact,
				Locale: locale, Hours: item.Hours, VerifiedOn: item.VerifiedOn})
		}
		a.resources[locale] = set
	}
	if a.fallback == "" {
		return nil, ErrAssets
	}

	var scripts bundle.CrisisTemplates
	if err := readYAML(ctx, source, "templates/crisis_templates.yaml", &scripts); err != nil || scripts.Schema != "crisis_templates.v1" {
		return nil, ErrAssets
	}
	for name, entry := range scripts.Templates {
		if name == "" || strings.TrimSpace(entry.Text) == "" || hasInterpolation(entry.Text) {
			return nil, ErrAssets
		}
		t := template{text: entry.Text, locales: map[string]string{}}
		for locale, text := range entry.Locales {
			if strings.TrimSpace(text) == "" || hasInterpolation(text) {
				return nil, ErrAssets
			}
			t.locales[locale] = text
		}
		a.templates[name] = t
	}
	var fallback bundle.Fallback
	if err := readYAML(ctx, source, "templates/fallback_by_objective.yaml", &fallback); err != nil || fallback.Schema != "fallback.v1" {
		return nil, ErrAssets
	}
	for objective, entry := range fallback.Templates {
		if strings.TrimSpace(entry.Text) == "" || hasInterpolation(entry.Text) {
			return nil, ErrAssets
		}
		t := template{text: entry.Text, locales: map[string]string{}}
		for loc, text := range entry.Locales {
			if strings.TrimSpace(text) == "" || hasInterpolation(text) {
				return nil, ErrAssets
			}
			t.locales[loc] = text
		}
		a.fallbacks[contract.Objective(objective)] = t
	}

	seenLocale := map[string]bool{}
	for _, locale := range manifest.Conversations() {
		if locale == "" || seenLocale[locale] {
			return nil, ErrAssets
		}
		seenLocale[locale] = true
		var lex bundle.Lexicon
		name := "lexicons/safety_l0." + locale + ".yaml"
		if err := readYAML(ctx, source, name, &lex); err != nil || lex.Schema != "lexicon.v1" || lex.Locale != locale || lex.Version == "" {
			return nil, ErrAssets
		}
		a.lexVersion[locale] = lex.Version
		seen := map[string]bool{}
		for _, item := range lex.Rules {
			c, level := contract.RiskCategory(item.Category), contract.RiskLevel(item.Level)
			if item.ID == "" || seen[item.ID] || !c.Valid() || !level.Valid() || level == contract.RiskNone {
				return nil, ErrAssets
			}
			seen[item.ID] = true
			rule := lexRule{id: item.ID, category: c, level: level, secondLook: item.SecondLook}
			switch item.Match.Type {
			case "regex":
				if item.Match.Pattern == "" || len(item.Match.Pattern) > 512 {
					return nil, ErrAssets
				}
				var err error
				rule.pattern, err = regexp.Compile(item.Match.Pattern)
				if err != nil {
					return nil, ErrAssets
				}
			case "literal":
				if item.Match.Pattern == "" {
					return nil, ErrAssets
				}
				rule.literal = item.Match.Pattern
			default:
				return nil, ErrAssets
			}
			a.locales[locale] = append(a.locales[locale], rule)
		}
		// An empty WP-03 lexicon is deliberately NOT interpreted as a negative
		// L0 test result; Gate.Assess will report a degraded elevated signal.
	}
	if raw, err := source.Artifact(ctx, "prompts/safety_l1.system.tmpl"); err == nil && len(raw) > 0 && utf8.Valid(raw) {
		a.prompt = string(raw)
	}
	return a, nil
}

func validResourceKind(kind string) bool {
	return kind == "hotline" || kind == "text_line" || kind == "emergency" || kind == "chat"
}
func hasInterpolation(text string) bool {
	return strings.Contains(text, "{{") || strings.Contains(text, "${") || strings.Contains(text, "<user")
}
func validPolicy(p bundle.SafetyPolicy) bool {
	if p.RiskFSM.FollowUpHours < 1 || p.RiskFSM.Deescalate.FromHighToElevated.MinHoursSinceSignal < 1 ||
		p.RiskFSM.Deescalate.FromElevatedToLow.MinHoursSinceSignal < 1 {
		return false
	}
	if !hasString(p.RiskFSM.Deescalate.FromHighToElevated.Require, "no_new_signals") ||
		!hasString(p.RiskFSM.Deescalate.FromElevatedToLow.Require, "no_new_signals") {
		return false
	}
	for category, rule := range p.Categories {
		if !contract.RiskCategory(category).Valid() || rule.TemplateFamily == "" || !contract.RiskLevel(rule.EscalateTo).Valid() || p.TemplateFamilies[rule.TemplateFamily].RequiredElements == nil {
			return false
		}
	}
	for _, family := range p.TemplateFamilies {
		if len(family.RequiredElements) == 0 || len(family.Forbidden) == 0 {
			return false
		}
		for _, k := range []string{"acknowledge", "care", "ask_safety", "resources", "invite_continue"} {
			if !hasString(family.RequiredElements, k) {
				return false
			}
		}
	}
	if p.SecondLook.Downgrade.From != "high" || p.SecondLook.Downgrade.To != "elevated" ||
		!hasString(p.SecondLook.Downgrade.NeverWhen, "first_person_present_tense") {
		return false
	}
	return true
}
func hasString(all []string, target string) bool {
	for _, s := range all {
		if s == target {
			return true
		}
	}
	return false
}

func (a *Assets) lexicon(locale string) ([]lexRule, string, bool) {
	if a == nil {
		return nil, "", false
	}
	if canonical, ok := contract.ConversationLanguage(locale); ok {
		locale = canonical
	}
	if rules, ok := a.locales[locale]; ok {
		return rules, a.lexVersion[locale], true
	}
	if idx := strings.IndexByte(locale, '-'); idx > 0 {
		base := locale[:idx]
		if rules, ok := a.locales[base]; ok {
			return rules, a.lexVersion[base], true
		}
	}
	return nil, "", false
}
