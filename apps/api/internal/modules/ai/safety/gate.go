package safety

import (
	"context"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"
	"math"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// Assessment is a checked L1 safety.v1 response. The classifier provides only
// labels and calibrated confidence; it is never allowed to lower an L0 hit.
type Assessment struct {
	Output  contract.SafetyClassifierOutput
	Version string
}
type Classifier interface {
	Classify(ctx context.Context, in ports.SafetyInput) (Assessment, error)
}

// SecondLookReview must be supplied by a separate, clinically approved
// contextual verifier. A generic LLM's global flags are NOT sufficient proof
// that every triggered span was quoted/fictional and not first-person present.
type SecondLookReview struct {
	Flags              []contract.SafetyContextFlag
	ContextualTriggers []bool // one verified decision per trigger, same ordering
	FirstPersonChecked bool
	FirstPersonPresent bool
}
type SecondLook interface {
	Review(context.Context, ports.SafetyInput, []contract.SafetyTrigger, []contract.SafetyContextFlag) (SecondLookReview, error)
}

type Gate struct {
	assets     *Assets
	classifier Classifier
	secondLook SecondLook // nil disables SF-007 safely
}

var _ ports.SafetyGate = (*Gate)(nil)

func NewGate(assets *Assets, classifier Classifier, look SecondLook) (*Gate, error) {
	if assets == nil || classifier == nil {
		return nil, ErrAssets
	}
	return &Gate{assets: assets, classifier: classifier, secondLook: look}, nil
}

// Assess computes SF-001/SF-002 with versioned L0 (RE2) and strict L1.
// Invalid L1, timeout, missing locale, or incomplete L0 NEVER becomes "none".
// Recent messages have no text spans because the port has no per-message span
// reference; only rule IDs are exposed and no matched text is ever logged.
func (g *Gate) Assess(ctx context.Context, in ports.SafetyInput) (signalResult contract.SafetySignal, assessErr error) {
	ctx, inspectionSpan := trace.Start(ctx, "safety/gate.Assess", func() any { return map[string]any{"in": in} })
	defer func() {
		inspectionSpan.Complete(func() any { return map[string]any{"signalResult": signalResult} }, assessErr, recover())
	}()

	if err := ctx.Err(); err != nil {
		return degraded(contract.RiskElevated, nil, "", ""), err
	}
	if g == nil || g.assets == nil {
		return degraded(contract.RiskElevated, nil, "", ""), ErrAssets
	}
	rules, version, hasLocale := g.assets.lexicon(in.Lang)
	if !hasLocale || len(rules) == 0 {
		return degraded(contract.RiskElevated, nil, version, ""), observability.Detail(ErrLocale, "locale_unavailable", "safety_locale", "assess")
	}
	if !utf8.ValidString(in.Text) || len(in.Text) > 64<<10 {
		return degraded(contract.RiskElevated, nil, version, ""), ErrAssets
	}

	// Input tags are hints: mixed-language text must receive every loaded L0
	// backstop, including historical user messages in a different language.
	rules = nil
	languages := make([]string, 0, len(g.assets.locales))
	for lang := range g.assets.locales {
		languages = append(languages, lang)
	}
	slices.Sort(languages)
	for _, lang := range languages {
		rules = append(rules, g.assets.locales[lang]...)
	}
	var triggers []contract.SafetyTrigger
	categories := map[contract.RiskCategory]int{}
	l0 := contract.RiskNone
	secondEligible := map[string]bool{}
	// Inspect the current message and up to four recent user messages. In
	// history, span offsets would be ambiguous without a message ID.
	texts := []string{in.Text}
	start := len(in.Recent) - 4
	if start < 0 {
		start = 0
	}
	for _, msg := range in.Recent[start:] {
		if msg.Role == contract.RoleUser {
			texts = append(texts, msg.Text)
		}
	}
	for index, text := range texts {
		if !utf8.ValidString(text) || len(text) > 64<<10 {
			return degraded(maxValid(l0, contract.RiskElevated), triggers, version, ""), ErrAssets
		}
		text = contract.NormalizeSafetyText(text)
		for _, rule := range rules {
			begin, end := -1, -1
			if rule.pattern != nil {
				pos := rule.pattern.FindStringIndex(text)
				if pos != nil {
					begin, end = pos[0], pos[1]
				}
			} else if pos := strings.Index(text, rule.literal); pos >= 0 {
				begin, end = pos, pos+len(rule.literal)
			}
			if begin < 0 {
				continue
			}
			if trace.Enabled(ctx) {
				trace.Capture(ctx, "l0_match", "matched_rule", "lexicons/safety_l0.*.yaml", map[string]any{"rule_id": rule.id, "category": rule.category, "level": rule.level, "message_index": index, "normalized_text": text, "matched_text": text[begin:end], "byte_start": begin, "byte_end": end})
			}
			if rule.level.Rank() > l0.Rank() {
				l0 = rule.level
			}
			if rule.level.Rank() > categories[rule.category] {
				categories[rule.category] = rule.level.Rank()
			}
			// Rule IDs in legacy bundles can overlap across locales. One
			// non-downgradable match must remain a veto across all matches.
			allowed, exists := secondEligible[rule.id]
			secondEligible[rule.id] = rule.secondLook && (!exists || allowed)
			if len(triggers) < 32 {
				trigger := contract.SafetyTrigger{Layer: "l0", RuleID: rule.id}
				if index == 0 {
					trigger.Span = &contract.Span{Start: utf8.RuneCountInString(text[:begin]), End: utf8.RuneCountInString(text[:end])}
				}
				triggers = append(triggers, trigger)
			}
		}
	}
	// A truncated trigger set cannot be proven contextual on second look.
	if len(triggers) == 32 {
		secondEligible = nil
	}

	trace.Capture(ctx, "safety_policy", "knowledge", "policy/safety_policy.yaml", g.assets.policy)
	assessed, err := g.classifier.Classify(ctx, in)
	if err != nil || !validL1(assessed) {
		reason := "invalid_signal"
		if err != nil {
			reason = "dependency_unavailable"
		}
		return g.degradedL0(maxValid(l0, contract.RiskElevated), triggers, version, assessed.Version, categories), observability.Detail(errors.Join(ErrClassifier, err), reason, "safety_classifier", "assess")
	}
	l1 := assessed.Output.Level
	for _, category := range assessed.Output.Categories {
		if l1.Rank() > categories[category] {
			categories[category] = l1.Rank()
		}
		if len(triggers) < 32 {
			triggers = append(triggers, contract.SafetyTrigger{Layer: "l1", RuleID: "safety.v1/" + string(category)})
		}
	}
	level := maxValid(l0, l1)
	source := "max(l0,l1)"
	if l0.Rank() > l1.Rank() {
		source = "l0"
	}
	if l1.Rank() > l0.Rank() {
		source = "l1"
	}
	if level == contract.RiskHigh && g.secondLook != nil && secondEligible != nil && g.assets.policy.SecondLook.Downgrade.From == "high" {
		if review, e := g.secondLook.Review(ctx, in, triggers, assessed.Output.ContextFlags); e == nil &&
			eligibleSecondLook(review, triggers, secondEligible, g.assets.policy.SecondLook.Downgrade.WhenContextFlagsAny) {
			level = contract.RiskElevated // SF-007: never lower below elevated
			source = "max(l0,l1)"
		}
	}
	if err := ctx.Err(); err != nil {
		return degraded(maxValid(level, contract.RiskElevated), triggers, version, assessed.Version), err
	}
	return contract.SafetySignal{Level: level, Categories: orderedCategories(categories), Triggers: triggers, Source: source,
		Confidence: float32(assessed.Output.Confidence), ClassifierVersion: assessed.Version, LexiconVersion: version}, nil
}

func validL1(a Assessment) bool {
	v := a.Output
	if !v.Level.Valid() || math.IsNaN(v.Confidence) || math.IsInf(v.Confidence, 0) || v.Confidence < 0 || v.Confidence > 1 || len(v.Categories) > 3 || len(v.ContextFlags) > 16 {
		return false
	}
	seen := map[contract.RiskCategory]bool{}
	for _, category := range v.Categories {
		if !category.Valid() || seen[category] {
			return false
		}
		seen[category] = true
	}
	for _, flag := range v.ContextFlags {
		if !flag.Valid() {
			return false
		}
	}
	return true
}
func maxValid(a, b contract.RiskLevel) contract.RiskLevel {
	if !a.Valid() {
		a = contract.RiskElevated
	}
	if !b.Valid() {
		b = contract.RiskElevated
	}
	if a.Rank() >= b.Rank() {
		return a
	}
	return b
}
func degraded(level contract.RiskLevel, triggers []contract.SafetyTrigger, lexVersion, classifierVersion string) contract.SafetySignal {
	return contract.SafetySignal{Level: maxValid(level, contract.RiskElevated), Categories: []contract.RiskCategory{}, Triggers: triggers,
		Source: "timeout_default", LexiconVersion: lexVersion, ClassifierVersion: classifierVersion, Degraded: true}
}
func orderedCategories(cats map[contract.RiskCategory]int) []contract.RiskCategory {
	all := make([]contract.RiskCategory, 0, len(cats))
	for c := range cats {
		all = append(all, c)
	}
	sort.Slice(all, func(i, j int) bool {
		if cats[all[i]] != cats[all[j]] {
			return cats[all[i]] > cats[all[j]]
		}
		return all[i] < all[j]
	})
	if len(all) > 3 {
		all = all[:3]
	}
	return all
}
func eligibleSecondLook(r SecondLookReview, triggers []contract.SafetyTrigger, flags map[string]bool, permitted []string) bool {
	if !r.FirstPersonChecked || r.FirstPersonPresent || len(r.ContextualTriggers) != len(triggers) || len(triggers) == 0 || len(r.Flags) == 0 {
		return false
	}
	contextual := false
	for _, flag := range r.Flags {
		if !flag.Valid() {
			return false
		}
		for _, allowed := range permitted {
			if string(flag) == allowed {
				contextual = true
			}
		}
	}
	if !contextual {
		return false
	}
	for i, trigger := range triggers {
		if !r.ContextualTriggers[i] {
			return false
		}
		if trigger.Layer == "l0" && !flags[trigger.RuleID] {
			return false
		}
		if trigger.Layer != "l0" && trigger.Layer != "l1" {
			return false
		}
	}
	return true
}

// Preserve L0 category evidence even when L1 is unavailable. Without this a
// correctly routed high-risk turn could not select its reviewed crisis family.
func (g *Gate) degradedL0(level contract.RiskLevel, triggers []contract.SafetyTrigger, lex, classifier string, categories map[contract.RiskCategory]int) contract.SafetySignal {
	sig := degraded(level, triggers, lex, classifier)
	sig.Categories = orderedCategories(categories)
	return sig
}
