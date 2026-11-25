package bundle

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

var semverRE = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
var localeRE = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)

func validateBundle(ctx context.Context, files map[string][]byte, m *Manifest, opt Options, report *Report) {
	checkRequired(files, report)
	validateManifest(files, m, opt, report)
	if m.EmitsSchemas["unified"] != "" {
		validateCards(files, report)
	}
	skills := map[string]Skill{}
	protocols := map[string]Protocol{}
	vocab := map[string]map[string]bool{}
	for _, name := range []string{"emotions", "rationale_tags", "avoid_tags", "hypothesis_tags", "memory_tags", "problem_tags", "skill_tags", "protocol_flags", "protocol_predicates"} {
		p := "config/vocab/" + name + ".yaml"
		var v TagVocabulary
		if parse(files, p, "vocab.v1", &v, report) {
			if len(v.Values) == 0 {
				report.fail(p, "vocab_empty", "controlled vocabulary must not be empty")
			}
			vocab[name] = makeSet(v.Values, p, report)
		}
	}
	var problem ProblemUnderstanding
	if parse(files, "schemas/problem_understanding.yaml", "problem.v1", &problem, report) {
		slots := map[string]bool{}
		for _, s := range problem.Slots {
			if !validID(s.ID) {
				report.fail("schemas/problem_understanding.yaml", "slot_id", "invalid slot id")
			}
			if slots[s.ID] {
				report.fail("schemas/problem_understanding.yaml", "duplicate_id", "duplicate slot")
			}
			slots[s.ID] = true
			if s.AskCost < 0 || s.AskCost > 1 {
				report.fail("schemas/problem_understanding.yaml", "slot_cost", "ask_cost outside 0..1")
			}
			for phase, n := range s.Importance {
				if !oneOf(phase, "exploring", "understanding_sufficient", "working", "maintaining", "resolved", "paused", "dormant") || n < 0 || n > 1 {
					report.fail("schemas/problem_understanding.yaml", "slot_importance", "invalid importance phase or weight")
				}
			}
		}
		if len(slots) == 0 {
			report.fail("schemas/problem_understanding.yaml", "slots_empty", "at least one slot required")
		}
		for _, id := range append(append([]string{}, problem.ReadinessRule.RequireFilled...), problem.ReadinessRule.AndAnyOf...) {
			if !slots[id] {
				report.fail("schemas/problem_understanding.yaml", "unknown_slot", "readiness references unknown slot")
			}
		}
		if problem.ReadinessRule.MinUserTurnsInThread < 0 || problem.InformationNeeds.VMin < 0 || problem.InformationNeeds.VMin > 1 {
			report.fail("schemas/problem_understanding.yaml", "ranges", "invalid readiness/information-needs values")
		}
		vocab["slots"] = slots
	}
	var user UserInformation
	if parse(files, "schemas/user_information.yaml", "userinfo.v1", &user, report) {
		ids := map[string]bool{}
		for _, field := range user.Fields {
			if !validID(field.ID) || ids[field.ID] {
				report.fail("schemas/user_information.yaml", "field_id", "invalid/duplicate information field")
			}
			ids[field.ID] = true
			if !oneOf(field.Sensitivity, "normal", "sensitive", "highly_sensitive") || field.AskCost < 0 || field.AskCost > 1 {
				report.fail("schemas/user_information.yaml", "field_schema", "invalid sensitivity or ask cost")
			}
			if field.Sensitivity == "highly_sensitive" && field.Collect.Proactive {
				report.fail("schemas/user_information.yaml", "sensitive_collection", "highly sensitive fields cannot be proactively collected")
			}
			if field.Collect.InferFromBehavior && !field.Collect.ConfirmBeforeStore {
				report.fail("schemas/user_information.yaml", "unconfirmed_inference", "inference needs user confirmation before storage")
			}
		}
	}
	formulations := map[string]bool{}
	for _, file := range []string{"schemas/formulation_cbt.yaml", "schemas/formulation_five_p.yaml"} {
		var f Formulation
		if !parse(files, file, "formulation.v1", &f, report) {
			continue
		}
		if !validID(f.Framework) || formulations[f.Framework] {
			report.fail(file, "framework_id", "invalid/duplicate framework")
		}
		formulations[f.Framework] = true
		ids := map[string]bool{}
		for _, field := range f.Fields {
			if !validID(field.ID) || ids[field.ID] || len(field.AllowedStatus) == 0 || field.MinEvidence < 1 || field.MinSessions < 0 {
				report.fail(file, "formulation_field", "invalid formulation field")
			}
			ids[field.ID] = true
			for _, status := range field.AllowedStatus {
				if !oneOf(status, "reported", "inferred", "unknown", "system_observed") {
					report.fail(file, "formulation_status", "unrecognized epistemic status")
				}
			}
		}
		for _, tag := range f.SelectWhen.ProblemTagsAny {
			requireTag(vocab, "problem_tags", tag, file, report)
		}
		if f.Paragraph.MaxWords <= 0 {
			report.fail(file, "paragraph_limit", "formulation paragraph needs a word limit")
		}
	}
	for _, p := range sortedKeys(files) {
		if err := ctx.Err(); err != nil {
			report.fail(p, "cancelled", "validation cancelled")
			return
		}
		switch {
		case strings.HasPrefix(p, "skills/") && strings.HasSuffix(p, ".yaml"):
			var s Skill
			if !parse(files, p, "skillcard.v1", &s, report) {
				continue
			}
			if !validID(s.ID) || s.ID != basenameID(p) {
				report.fail(p, "skill_id", "id must match filename and [a-z0-9_]+")
			}
			if _, ok := skills[s.ID]; ok {
				report.fail(p, "duplicate_id", "duplicate skill")
			}
			skills[s.ID] = s
			if !oneOf(s.Category, "listening", "questioning", "explaining", "structuring", "boundary", "repair") {
				report.fail(p, "skill_category", "invalid category")
			}
			if s.Name == "" || s.Descriptor == "" || s.Purpose == "" || utf8.RuneCountInString(s.Descriptor) > 160 || strings.Contains(s.Descriptor, "\n") {
				report.fail(p, "skill_descriptor", "descriptor must be one line, <=160 characters, and card metadata cannot be empty")
			}
			if s.CardText == "" {
				report.fail(p, "skill_token_budget", "card_text empty")
			}
			if s.Form.MaxSentences < 1 || s.Form.MaxSentences > 10 {
				report.fail(p, "skill_form", "invalid max_sentences")
			}
			if len(s.Examples.Good) == 0 || len(s.Examples.Poor) == 0 {
				report.fail(p, "skill_examples", "at least one good and one poor example required")
			}
			for _, ex := range s.Examples.Good {
				if ex.User == "" || ex.Reply == "" {
					report.fail(p, "skill_examples", "empty good example")
				}
			}
			for _, ex := range s.Examples.Poor {
				if ex.User == "" || ex.Reply == "" || ex.Why == "" {
					report.fail(p, "skill_examples", "empty poor example or explanation")
				}
			}
			for _, tag := range append(append([]string{}, s.UseWhen...), s.DoNotUseWhen...) {
				requireTag(vocab, "skill_tags", tag, p, report)
			}
			for _, tag := range s.Form.Avoid {
				requireTag(vocab, "avoid_tags", tag, p, report)
			}
			for loc, override := range s.Locales {
				if !localeRE.MatchString(loc) || override.CardText == "" || override.Descriptor == "" || utf8.RuneCountInString(override.Descriptor) > 160 {
					report.fail(p, "skill_locale", "invalid locale override")
				}
			}
		case strings.HasPrefix(p, "protocols/") && strings.HasSuffix(p, ".yaml"):
			var proto Protocol
			if !parse(files, p, "protocol.v1", &proto, report) {
				continue
			}
			if !validID(proto.ID) || proto.ID != basenameID(p) || !semverRE.MatchString(proto.Version) {
				report.fail(p, "protocol_id", "invalid id/version or filename mismatch")
			}
			if _, ok := protocols[proto.ID]; ok {
				report.fail(p, "duplicate_id", "duplicate protocol")
			}
			protocols[proto.ID] = proto
			validateProtocol(p, proto, vocab, report)
		}
	}
	if len(skills) > 20 {
		report.fail("skills/", "skill_catalog", "maximum 20 Skill Cards")
	}
	for id, s := range skills {
		p := "skills/" + id + ".yaml"
		for _, other := range append(append([]string{}, s.CompatibleWith...), s.ConflictsWith...) {
			if _, ok := skills[other]; !ok {
				report.fail(p, "unknown_skill", "referenced skill does not exist")
			}
			if id == other {
				report.fail(p, "skill_self_reference", "skill must not reference itself")
			}
		}
		for _, other := range s.CompatibleWith {
			for _, conflict := range s.ConflictsWith {
				if other == conflict {
					report.fail(p, "skill_conflict", "same skill compatible and conflicting")
				}
			}
		}
	}
	validatePolicyAssets(files, m, vocab, protocols, report, opt)
	validateConfig(files, vocab, report)
	validateIntegrity(files, m, opt, report)
	// Goldens are a packaging smoke check, not model evaluations (WP-15).
	covered := map[string]bool{}
	for _, p := range filesWithPrefix(files, "evals/golden/") {
		var g Golden
		if strictYAML(files[p], &g) == nil {
			for _, tag := range g.Tags {
				if strings.HasPrefix(tag, "objective_") {
					covered[strings.TrimPrefix(tag, "objective_")] = true
				}
			}
		}
	}
	missing := []string{}
	for _, name := range []string{"reflect", "validate_and_support", "clarify", "explore", "provide_insight", "provide_guidance", "offer_intervention", "continue_intervention", "close_or_pause_intervention", "review_progress", "summarize", "repair", "answer_direct", "boundary", "safety_support"} {
		if !covered[name] {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		report.warn("evals/golden/", "golden_coverage", fmt.Sprintf("%d objectives lack golden scenarios; WP-15 owns evaluation", len(missing)))
	}
	if len(protocols) == 0 || len(skills) == 0 {
		report.fail("knowledge/", "catalog_empty", "at least one synthetic/test catalog asset required")
	}
}
func parse[T any](files map[string][]byte, p, version string, out *T, r *Report) bool {
	b, ok := files[p]
	if !ok {
		return false
	}
	if err := strictYAML(b, out); err != nil {
		r.fail(p, "yaml_schema", err.Error())
		return false
	}
	// Schema discriminators are checked after strict shape decoding.
	schemaField := reflect.ValueOf(out).Elem().FieldByName("Schema")
	if !schemaField.IsValid() || schemaField.String() != version {
		r.fail(p, "schema_version", "unexpected schema version; expected "+version)
		return false
	}
	return true
}
func filesWithPrefix(files map[string][]byte, prefix string) []string {
	var out []string
	for p := range files {
		if strings.HasPrefix(p, prefix) {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}
func validID(id string) bool { return idPattern.MatchString(id) }
func oneOf(v string, options ...string) bool {
	for _, s := range options {
		if s == v {
			return true
		}
	}
	return false
}
func makeSet(values []string, p string, r *Report) map[string]bool {
	result := map[string]bool{}
	for _, v := range values {
		if !validID(v) {
			r.fail(p, "vocab_id", "invalid vocabulary identifier")
		}
		if result[v] {
			r.fail(p, "vocab_duplicate", "duplicate vocabulary identifier")
		}
		result[v] = true
	}
	return result
}
func requireTag(vocab map[string]map[string]bool, group, tag, file string, r *Report) {
	if !vocab[group][tag] {
		r.fail(file, "vocab_reference", "unknown "+group+" reference: "+tag)
	}
}
func estimateTokens(s string) int {
	// Provider-neutral conservative estimate for linting; not an exact tokenizer.
	n := len([]byte(s))
	words := len(strings.Fields(s))
	return int(math.Max(float64((n+3)/4), float64(words)))
}
func validateManifest(files map[string][]byte, m *Manifest, opt Options, r *Report) {
	if err := m.ValidateLanguages(); err != nil {
		r.fail("bundle.yaml", "language_contract", err.Error())
	}
	if m.Name != "theia-kb" || !semverRE.MatchString(m.Version) {
		r.fail("bundle.yaml", "manifest_identity", "expected name theia-kb and SemVer version")
	}
	core := opt.CoreVersion
	if core == "" {
		core = "1.0.0"
	}
	if !compatible(m.RequiresCore, core) {
		r.fail("bundle.yaml", "core_compatibility", "incompatible or malformed requires_core")
	}
	want := map[string]string{"memory_writer": contract.MemoryWriterSchemaVersion, "reflection": contract.ReflectionSchemaVersion, "safety_l1": contract.SafetySchemaVersion}
	if m.EmitsSchemas["unified"] != "" {
		want["unified"] = contract.UnifiedSchemaVersion
		if _, ok := files["config/background.yaml"]; ok {
			want["background"] = "background_intelligence.v2"
		}
	} else {
		want["brain"] = contract.BrainSchemaVersion
	}
	if len(m.EmitsSchemas) != len(want) {
		r.fail("bundle.yaml", "emits_schema", "unexpected emitted schema versions")
	}
	for k, v := range want {
		if m.EmitsSchemas[k] != v {
			r.fail("bundle.yaml", "emits_schema", "schema mismatch for "+k)
		}
	}
	if len(m.Conversations()) == 0 {
		r.fail("bundle.yaml", "locales", "at least one locale required")
	}
	seen := map[string]bool{}
	for _, loc := range m.Conversations() {
		if !localeRE.MatchString(loc) || seen[loc] {
			r.fail("bundle.yaml", "locales", "invalid/duplicate locale")
		}
		seen[loc] = true
	}
	if m.Changelog != "CHANGELOG.md" {
		r.fail("bundle.yaml", "changelog", "changelog must refer to CHANGELOG.md")
	}
	required := makeSet(m.ClinicalReview.RequiredFor, "bundle.yaml", r)
	// Once a WP-07 lexicon is shipped, its matcher rules, configured
	// thresholds and classifier prompt must carry independent clinical
	// approval hashes in the signed WP-03 release.
	if len(filesWithPrefix(files, "lexicons/validator.")) > 0 && !m.SyntheticOnly && !required["validator"] {
		r.fail("bundle.yaml", "validator_review", "WP-07 assets require clinical_review.required_for: validator")
	}
	for _, k := range []string{"protocols", "boundary_policy", "safety_policy", "communication_policy", "crisis_templates"} {
		if !required[k] {
			r.fail("bundle.yaml", "clinical_requirements", "mandatory review category missing: "+k)
		}
	}
	if opt.Release && m.SyntheticOnly {
		r.fail("bundle.yaml", "synthetic_release", "synthetic-only bundle cannot be released")
	}
	if !m.SyntheticOnly {
		for p, b := range files {
			if p == "bundle.yaml" || p == "bundle.sig" || p == "README.md" || p == "CHANGELOG.md" {
				continue
			}
			uppercase := strings.ToUpper(string(b))
			if strings.Contains(uppercase, "SYNTHETIC") || strings.Contains(uppercase, "PLACEHOLDER") || strings.Contains(uppercase, "NOT FOR CLINICAL USE") {
				r.fail(p, "unreviewed_placeholder", "placeholder text cannot be packaged as non-synthetic")
			}
		}
	}
	if m.SyntheticOnly {
		r.warn("bundle.yaml", "synthetic_only", "synthetic fixture bundle; cannot activate in production or sign for release")
	}
}
func compatible(expr, core string) bool {
	value, ok := parseVersion(core)
	if !ok {
		return false
	}
	fields := strings.Fields(expr)
	if len(fields) < 1 {
		return false
	}
	for _, f := range fields {
		op := ""
		for _, prefix := range []string{">=", "<=", ">", "<", "="} {
			if strings.HasPrefix(f, prefix) {
				op = prefix
				break
			}
		}
		if op == "" {
			return false
		}
		limit, ok := parseVersion(strings.TrimPrefix(f, op))
		if !ok {
			return false
		}
		cmp := 0
		for i := 0; i < 3; i++ {
			if value[i] < limit[i] {
				cmp = -1
				break
			}
			if value[i] > limit[i] {
				cmp = 1
				break
			}
		}
		switch op {
		case ">=":
			if cmp < 0 {
				return false
			}
		case "<=":
			if cmp > 0 {
				return false
			}
		case ">":
			if cmp <= 0 {
				return false
			}
		case "<":
			if cmp >= 0 {
				return false
			}
		case "=":
			if cmp != 0 {
				return false
			}
		}
	}
	return true
}
func parseVersion(s string) ([3]int, bool) {
	var n [3]int
	if !semverRE.MatchString(s) {
		return n, false
	}
	_, err := fmt.Sscanf(s, "%d.%d.%d", &n[0], &n[1], &n[2])
	return n, err == nil
}
func validateProtocol(p string, proto Protocol, vocab map[string]map[string]bool, r *Report) {
	if proto.Name == "" || proto.OneLiner == "" || utf8.RuneCountInString(proto.OneLiner) > 140 || proto.EstMinutes < 1 || !oneOf(proto.Family, "cbt", "act", "dbt", "grounding", "planning", "assessment") {
		r.fail(p, "protocol_format", "missing/invalid protocol metadata")
	}
	for _, tag := range proto.FitTags {
		requireTag(vocab, "problem_tags", tag, p, r)
	}
	for _, id := range proto.Eligibility.RequireSlotsFilled {
		requireTag(vocab, "slots", id, p, r)
	}
	for _, id := range proto.Eligibility.ForbidFlags {
		if !vocab["protocol_flags"][id] {
			r.fail(p, "PR-007", "unknown eligibility forbid_flag")
		}
	}
	if !oneOf(proto.Eligibility.RequireRiskAtMost, "none", "low", "elevated", "high") || !oneOf(proto.Eligibility.MinUserReadiness, "low", "medium", "high") {
		r.fail(p, "eligibility", "invalid readiness/risk eligibility")
	}
	for _, phase := range proto.Eligibility.RequirePhase {
		if !oneOf(phase, "exploring", "understanding_sufficient", "working", "maintaining", "resolved", "paused", "dormant") {
			r.fail(p, "eligibility", "unknown thread phase")
		}
	}
	if proto.Eligibility.DeclineCooldownDays < 0 || proto.Eligibility.MinDaysBetweenRuns < 0 || proto.Eligibility.MaxRunsPerThread < 0 {
		r.fail(p, "eligibility", "invalid cooldown/run limits")
	}
	for _, intent := range proto.Offer.Intents {
		if strings.TrimSpace(intent) == "" {
			r.fail(p, "protocol_intent", "empty/oversized therapeutic intent")
		}
	}
	stepIDs := map[string]bool{}
	if len(proto.Steps) == 0 {
		r.fail(p, "protocol_steps", "protocol needs steps")
	}
	for _, step := range proto.Steps {
		if !validID(step.ID) || stepIDs[step.ID] {
			r.fail(p, "protocol_step_id", "duplicate or invalid step id")
		}
		stepIDs[step.ID] = true
	}
	for _, step := range proto.Steps {
		if step.Goal == "" || step.Instruction == "" {
			r.fail(p, "protocol_step_tokens", "step goal/instruction missing")
		}
		if step.OnSuccess != "complete" && !stepIDs[step.OnSuccess] {
			r.fail(p, "PR-001", "on_success must reference a step or complete")
		}
		if !oneOf(step.OnDistress, "stop", "pause") {
			r.fail(p, "PR-002", "on_distress must be explicitly stop or pause")
		}
		if !oneOf(step.OnDecline, "stop", "pause") {
			r.fail(p, "protocol_decline", "on_decline must be stop or pause")
		}
		if step.SuccessCriteria == "" || !vocab["protocol_predicates"][step.SuccessCriteria] {
			r.fail(p, "PR-005", "unknown success criterion")
		}
		inputIDs := map[string]bool{}
		for _, slot := range step.ExpectedInputs {
			if !validID(slot) || inputIDs[slot] {
				r.fail(p, "PR-004", "invalid/duplicate expected input key")
			}
			inputIDs[slot] = true
		}
		adaptations := map[string]bool{}
		for _, adapt := range step.Adaptations {
			if !validID(adapt.ID) || adaptations[adapt.ID] || adapt.Instruction == "" {
				r.fail(p, "PR-006", "invalid adaptation")
			}
			adaptations[adapt.ID] = true
		}
		if step.OnStuck.Adapt != "" && !adaptations[step.OnStuck.Adapt] {
			r.fail(p, "PR-006", "on_stuck references missing adaptation")
		}
		if step.OnStuck.After < 0 || !oneOf(step.OnStuck.Then, "", "stop", "pause") {
			r.fail(p, "PR-006", "invalid on_stuck transition")
		}
	}
	// A finite-state graph must reach completion; detect nonterminating cycles
	// in the unconditional success path without altering runtime engine logic.
	for _, step := range proto.Steps {
		visited := map[string]bool{}
		next := step.ID
		for next != "complete" && !visited[next] {
			visited[next] = true
			found := false
			for _, s := range proto.Steps {
				if s.ID == next {
					next = s.OnSuccess
					found = true
					break
				}
			}
			if !found {
				break
			}
		}
		if next != "complete" && visited[next] {
			r.fail(p, "protocol_cycle", "success path has a cycle without completion")
		}
	}
	for _, tag := range proto.Completion.MemoryWrite.Tags {
		requireTag(vocab, "memory_tags", tag, p, r)
	}
}

// CommunicationPolicyTokenLimit preserves the Contract Pack v1.2 limit for
// release bundles while allowing longer synthetic knowledge during local tests.
func CommunicationPolicyTokenLimit(synthetic bool) int {
	if synthetic {
		return 1200
	}
	return 600
}
