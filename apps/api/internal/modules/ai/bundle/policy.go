package bundle

import (
	"regexp"
	"strings"
	"time"
)

func validatePolicyAssets(files map[string][]byte, m *Manifest, vocab map[string]map[string]bool, protocols map[string]Protocol, r *Report, opt Options) {
	var safety SafetyPolicy
	if parse(files, "policy/safety_policy.yaml", "safety_policy.v1", &safety, r) {
		if safety.RiskFSM.FollowUpHours < 1 || safety.RiskFSM.Deescalate.FromHighToElevated.MinHoursSinceSignal < 1 || safety.RiskFSM.Deescalate.FromElevatedToLow.MinHoursSinceSignal < 1 {
			r.fail("policy/safety_policy.yaml", "risk_fsm", "invalid risk follow-up/deescalation threshold")
		}
		if len(safety.Categories) == 0 || len(safety.TemplateFamilies) == 0 {
			coverage(r, opt, "policy/safety_policy.yaml", "safety_not_reviewed", "no reviewed risk categories/template families")
		}
		for cat, v := range safety.Categories {
			if !oneOf(cat, "suicidal_ideation", "self_harm", "harm_to_others", "abuse_disclosure", "medical_emergency", "substance_crisis", "other") {
				r.fail("policy/safety_policy.yaml", "risk_category", "unknown risk category")
			}
			if _, ok := safety.TemplateFamilies[v.TemplateFamily]; !ok {
				r.fail("policy/safety_policy.yaml", "template_reference", "risk category refers to missing template family")
			}
			if !oneOf(v.EscalateTo, "low", "elevated", "high", "imminent") {
				r.fail("policy/safety_policy.yaml", "risk_level", "invalid escalation level")
			}
		}
	}
	var boundary BoundaryPolicy
	if parse(files, "policy/boundary_policy.yaml", "boundary.v1", &boundary, r) {
		ids := map[string]bool{}
		for _, mode := range boundary.Modes {
			if !validID(mode.ID) || ids[mode.ID] || mode.ResponseScriptID == "" {
				r.fail("policy/boundary_policy.yaml", "boundary_id", "missing/duplicate mode id or script")
			}
			ids[mode.ID] = true
			for _, tag := range mode.Detect {
				requireTag(vocab, "rationale_tags", tag, "policy/boundary_policy.yaml", r)
			}
		}
		if len(boundary.Modes) == 0 {
			coverage(r, opt, "policy/boundary_policy.yaml", "boundary_not_reviewed", "no reviewed boundary modes")
		}
	}
	var continuity ContinuityPolicy
	if parse(files, "policy/continuity_policy.yaml", "continuity.v1", &continuity, r) {
		if continuity.StaleAfterDays < 0 || continuity.ResumptionBands.SameMin < 0 || continuity.ResumptionBands.ShortHours < 0 || continuity.ResumptionBands.DayDays < 0 {
			r.fail("policy/continuity_policy.yaml", "continuity_ranges", "negative resumption/continuity threshold")
		}
	}
	var fallback Fallback
	if parse(files, "templates/fallback_by_objective.yaml", "fallback.v1", &fallback, r) {
		objectives := []string{"reflect", "validate_and_support", "clarify", "explore", "provide_insight", "provide_guidance", "offer_intervention", "continue_intervention", "close_or_pause_intervention", "review_progress", "summarize", "repair", "answer_direct", "boundary", "safety_support"}
		for _, objective := range objectives {
			t, ok := fallback.Templates[objective]
			if !ok || t.ID == "" || strings.TrimSpace(t.Text) == "" {
				r.fail("templates/fallback_by_objective.yaml", "fallback_coverage", "missing fallback for "+objective)
				continue
			}
			if hasInterpolation(t.Text) {
				r.fail("templates/fallback_by_objective.yaml", "fallback_interpolation", "fallback must not interpolate user content")
			}
			for _, text := range t.Locales {
				if strings.TrimSpace(text) == "" || hasInterpolation(text) {
					r.fail("templates/fallback_by_objective.yaml", "fallback_locale", "unsafe localized fallback")
				}
			}
			if objective == "safety_support" && !t.IncludesResources {
				r.fail("templates/fallback_by_objective.yaml", "fallback_safety", "safety fallback must declare resource support")
			}
		}
		for objective := range fallback.Templates {
			if !oneOf(objective, objectives...) {
				r.fail("templates/fallback_by_objective.yaml", "fallback_objective", "unknown objective")
			}
		}
	}
	var crisis CrisisTemplates
	if parse(files, "templates/crisis_templates.yaml", "crisis_templates.v1", &crisis, r) {
		if len(crisis.Templates) == 0 {
			coverage(r, opt, "templates/crisis_templates.yaml", "crisis_templates_not_reviewed", "no reviewed static crisis templates")
		}
		for _, template := range crisis.Templates {
			if hasInterpolation(template.Text) {
				r.fail("templates/crisis_templates.yaml", "crisis_interpolation", "crisis template must not interpolate user text")
			}
		}
	}
	validatorLexicons := 0
	for _, p := range filesWithPrefix(files, "lexicons/") {
		if strings.HasPrefix(p, "lexicons/validator.") {
			validatorLexicons++
			var lex ValidatorLexicon
			if !parse(files, p, "validator_lexicon.v1", &lex, r) {
				continue
			}
			loc := strings.TrimSuffix(strings.TrimPrefix(p, "lexicons/validator."), ".yaml")
			if lex.Locale != loc || !localeRE.MatchString(loc) {
				r.fail(p, "validator_locale", "invalid WP-07 locale")
			}
			if len(lex.Hedges) == 0 || len(lex.Questions) == 0 || len(lex.ClinicalLabels) == 0 {
				r.fail(p, "validator_coverage", "missing WP-07 question/hedge/clinical markers")
			}
			for _, key := range []string{"diagnosis", "medication", "guarantee", "therapist_identity", "directive_advice", "internal_leak", "technique", "dependency", "invalidating", "crisis_forbidden"} {
				if len(lex.Patterns[key]) == 0 {
					r.fail(p, "validator_coverage", "WP-07 missing mandatory matcher "+key)
				}
			}
			for kind, values := range lex.Patterns {
				if !oneOf(kind, "diagnosis", "medication", "guarantee", "therapist_identity", "directive_advice", "internal_leak", "technique", "dependency", "invalidating", "crisis_forbidden") {
					r.fail(p, "validator_rule", "unknown WP-07 matcher group")
				}
				for _, pattern := range values {
					if len(pattern) == 0 || len(pattern) > 512 {
						r.fail(p, "validator_rule", "empty/oversized WP-07 pattern")
					} else if _, err := regexp.Compile(pattern); err != nil {
						r.fail(p, "validator_regex", "WP-07 regex must compile under RE2")
					}
				}
			}
			for _, pattern := range lex.Questions {
				if len(pattern) == 0 || len(pattern) > 512 {
					r.fail(p, "validator_rule", "empty/oversized WP-07 question pattern")
				} else if _, err := regexp.Compile(pattern); err != nil {
					r.fail(p, "validator_regex", "WP-07 question regex must compile under RE2")
				}
			}
			continue
		}
		if !strings.HasPrefix(p, "lexicons/safety_l0.") {
			r.fail(p, "lexicon_name", "lexicon must be safety_l0.<locale>.yaml or validator.<locale>.yaml")
			continue
		}
		var lex Lexicon
		if !parse(files, p, "lexicon.v1", &lex, r) {
			continue
		}
		loc := strings.TrimSuffix(strings.TrimPrefix(p, "lexicons/safety_l0."), ".yaml")
		if lex.Locale != loc || !localeRE.MatchString(loc) || lex.Version == "" {
			r.fail(p, "lexicon_locale", "invalid lexicon locale/version")
		}
		if len(lex.Rules) == 0 {
			coverage(r, opt, p, "lexicon_not_reviewed", "no reviewed L0 rules")
		}
		ids := map[string]bool{}
		for _, rule := range lex.Rules {
			if ids[rule.ID] || !regexp.MustCompile(`^[a-z0-9_.-]+$`).MatchString(rule.ID) {
				r.fail(p, "lexicon_id", "invalid/duplicate L0 rule id")
			}
			ids[rule.ID] = true
			if !oneOf(rule.Category, "suicidal_ideation", "self_harm", "harm_to_others", "abuse_disclosure", "medical_emergency", "substance_crisis", "other") || !oneOf(rule.Level, "low", "elevated", "high", "imminent") {
				r.fail(p, "lexicon_risk", "invalid risk category/level")
			}
			switch rule.Match.Type {
			case "regex":
				if len(rule.Match.Pattern) > 512 {
					r.fail(p, "lexicon_pattern", "pattern too long")
				}
				if _, err := regexp.Compile(rule.Match.Pattern); err != nil {
					r.fail(p, "lexicon_regex", "pattern must compile under Go RE2")
				}
			case "literal":
				if rule.Match.Pattern == "" {
					r.fail(p, "lexicon_pattern", "empty literal")
				}
			default:
				r.fail(p, "lexicon_match", "unsupported rule matcher")
			}
		}
	}
	if validatorLexicons > 0 || len(m.ConversationLanguages) > 0 {
		for _, loc := range m.Conversations() {
			if _, ok := files["lexicons/validator."+loc+".yaml"]; !ok {
				r.fail("lexicons/validator."+loc+".yaml", "validator_coverage", "missing WP-07 lexicon for manifest locale")
			}
		}
	}
	var resources CrisisResources
	if parse(files, "resources/crisis_resources.yaml", "crisis_resources.v1", &resources, r) {
		if resources.StaleAfterDays < 1 {
			r.fail("resources/crisis_resources.yaml", "freshness", "stale_after_days must be positive")
		}
		if resources.FallbackLocale == "" {
			r.fail("resources/crisis_resources.yaml", "fallback_locale", "fallback locale required")
		}
		at := opt.At
		if at.IsZero() {
			at = time.Now().UTC()
		}
		for loc, entries := range resources.Locales {
			if !localeRE.MatchString(loc) {
				r.fail("resources/crisis_resources.yaml", "locale", "invalid resource locale")
			}
			ids := map[string]bool{}
			if len(entries) == 0 {
				coverage(r, opt, "resources/crisis_resources.yaml", "resource_not_reviewed", "no verified resources for "+loc)
			}
			for _, entry := range entries {
				if !validID(entry.ID) || ids[entry.ID] || entry.Name == "" || entry.Contact == "" || entry.Kind == "" {
					r.fail("resources/crisis_resources.yaml", "resource_item", "invalid or duplicate resource metadata")
				}
				ids[entry.ID] = true
				date, err := time.Parse("2006-01-02", entry.VerifiedOn)
				if err != nil {
					r.fail("resources/crisis_resources.yaml", "resource_date", "verified_on must be YYYY-MM-DD")
					continue
				}
				if date.After(at.Add(24 * time.Hour)) {
					r.fail("resources/crisis_resources.yaml", "resource_date", "verification date lies in future")
				}
				if at.Sub(date) > time.Duration(resources.StaleAfterDays)*24*time.Hour {
					coverage(r, opt, "resources/crisis_resources.yaml", "resource_stale", "verified resource is past stale_after_days")
				}
			}
		}
		if resources.LastFullVerification == "" {
			coverage(r, opt, "resources/crisis_resources.yaml", "resource_verification_pending", "no complete verification date")
		} else if verified, err := time.Parse("2006-01-02", resources.LastFullVerification); err != nil || verified.After(at.Add(24*time.Hour)) {
			r.fail("resources/crisis_resources.yaml", "resource_verification_date", "invalid future/full verification date")
		}
		if len(resources.Locales) == 0 {
			r.fail("resources/crisis_resources.yaml", "resource_locales", "resources locale map is empty")
		}
	}
	for _, loc := range m.Conversations() {
		p := "lexicons/safety_l0." + loc + ".yaml"
		if _, ok := files[p]; !ok {
			r.fail(p, "locale_coverage", "manifest locale lacks L0 lexicon")
		}
		if _, ok := resources.Locales[loc]; !ok {
			r.fail("resources/crisis_resources.yaml", "locale_coverage", "manifest locale lacks crisis resources: "+loc)
		}
		for objective, t := range fallback.Templates {
			if loc != "en" && t.Locales[loc] == "" {
				r.fail("templates/fallback_by_objective.yaml", "locale_coverage", "missing fallback locale "+loc+" for "+objective)
			}
		}
		if loc != "en" {
			for name, t := range crisis.Templates {
				if t.Locales[loc] == "" {
					r.fail("templates/crisis_templates.yaml", "locale_coverage", "missing crisis template locale "+loc+" for "+name)
				}
			}
		}
	}
	var scripts BoundaryScripts
	if parse(files, "templates/boundary_scripts.yaml", "boundary_scripts.v1", &scripts, r) {
		for _, mode := range boundary.Modes {
			if script, ok := scripts.Scripts[mode.ResponseScriptID]; !ok || script.Text == "" {
				r.fail("policy/boundary_policy.yaml", "boundary_script_reference", "boundary mode references missing script")
			}
		}
		for id, script := range scripts.Scripts {
			if !validID(id) || script.Text == "" || hasInterpolation(script.Text) {
				r.fail("templates/boundary_scripts.yaml", "boundary_script", "invalid/interpolating script")
			}
			for _, loc := range m.Conversations() {
				if loc != "en" && script.Locales[loc] == "" {
					r.fail("templates/boundary_scripts.yaml", "locale_coverage", "boundary script missing locale")
				}
			}
		}
	}
	var offers ProtocolOfferTexts
	if parse(files, "templates/protocol_offer.yaml", "protocol_offer.v1", &offers, r) {
		for id, p := range protocols {
			for _, reason := range p.Offer.WhyTemplates {
				if entry, ok := offers.WhyTemplates[reason]; !ok || entry.Text == "" {
					r.fail("protocols/"+id+".yaml", "offer_reference", "offer references missing why_template")
				}
			}
			for _, must := range p.Offer.MustSay {
				if entry, ok := offers.MustSay[must]; !ok || entry.Text == "" {
					r.fail("protocols/"+id+".yaml", "offer_reference", "offer references missing must_say")
				}
			}
		}
		for _, texts := range []map[string]struct {
			Text    string            `yaml:"text"`
			Locales map[string]string `yaml:"locales"`
		}{offers.WhyTemplates, offers.MustSay} {
			for id, entry := range texts {
				if !validID(id) || entry.Text == "" || hasInterpolation(entry.Text) {
					r.fail("templates/protocol_offer.yaml", "offer_template", "invalid/interpolating text")
				}
			}
		}
	}
	var forbidden ForbiddenClaims
	if parse(files, "config/vocab/forbidden_claims.yaml", "forbidden_claims.v1", &forbidden, r) {
		for _, loc := range m.Conversations() {
			if len(forbidden.Locales[loc]) == 0 {
				coverage(r, opt, "config/vocab/forbidden_claims.yaml", "forbidden_claims_not_reviewed", "no reviewed forbidden-claim patterns for "+loc)
			}
		}
		for loc, patterns := range forbidden.Locales {
			if !localeRE.MatchString(loc) {
				r.fail("config/vocab/forbidden_claims.yaml", "claim_locale", "invalid locale")
			}
			for _, pattern := range patterns {
				if len(pattern) > 512 || pattern == "" {
					r.fail("config/vocab/forbidden_claims.yaml", "claim_pattern", "empty/oversized forbidden-claim pattern")
					continue
				}
				compiled, err := regexp.Compile(pattern)
				if err != nil {
					r.fail("config/vocab/forbidden_claims.yaml", "claim_regex", "pattern does not compile with RE2")
					continue
				}
				for p, b := range files {
					if strings.HasPrefix(p, "prompts/generator") || strings.HasPrefix(p, "skills/") || strings.HasPrefix(p, "templates/") || strings.HasPrefix(p, "policy/communication_policy") {
						if compiled.Match(b) {
							r.fail(p, "forbidden_claim", "generator-facing asset contains forbidden claim pattern")
						}
					}
				}
			}
		}
	}

}
func hasInterpolation(s string) bool {
	return strings.Contains(s, "{{") || strings.Contains(s, "${") || strings.Contains(s, "%s") || strings.Contains(s, "%v")
}
func coverage(r *Report, opt Options, path, code, description string) {
	if opt.Release {
		r.fail(path, code, description)
	} else {
		r.warn(path, code, description)
	}
}
