package validator

import (
	"context"
	"errors"
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

var (
	idLeakRE = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b|\b(?:brain|wctx|safety|memwriter|reflection)\.v\d+\b|\b(?:schema_version|permitted_context_ids|client_turn_id|turn_id|session_id|thread_id|user_id|message_id|risk_fsm)\b|<\/?(?:plan|memory|system|state|user_message|regen_note)(?:\s|>)`)
	urlRE    = regexp.MustCompile(`(?i)\b(?:https?://|www\.)[^\s<>()\[\]"']+`)
	emailRE  = regexp.MustCompile(`(?i)\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.-]+\.[a-z]{2,}\b`)
	handleRE = regexp.MustCompile(`(?:^|\s)@[a-zA-Z][a-zA-Z0-9._]{2,}\b`)
	// Avoid treating short counts/dates as phone numbers. Curated short-code
	// resources are still recognized by contacts in the crisis plan.
	phoneRE      = regexp.MustCompile(`(?:\+?[0-9][0-9\-(). ]{6,}[0-9])`)
	recallRE     = regexp.MustCompile(`(?i)\b(?:i remember you|you told me (?:before|last time)|last time you (?:told|said)|from our previous conversation)\b`)
	splitWordsRE = regexp.MustCompile(`[\pL\pN]+`)
)

type finishKey struct{}
type requirementsKey struct{}
type requiredContent struct {
	ids      []string
	resolved map[string]string
}

// WithFinishReason supplies provider completion metadata without modifying
// the versioned ports.Validator contract. A length-stop with an unfinished
// sentence is OV-031; no missing metadata is fabricated.
func WithFinishReason(ctx context.Context, reason string) context.Context {
	return context.WithValue(ctx, finishKey{}, reason)
}

// WithProtocolRequirements binds reviewed must_say identifiers to their
// actual approved text. Unresolvable identifiers are rejected (OV-014).
// A future Protocol Engine must supply these on the same turn context.
func WithProtocolRequirements(ctx context.Context, spec *contract.ProtocolStepSpec, approvedText map[string]string) context.Context {
	req := requiredContent{resolved: make(map[string]string, len(approvedText))}
	if spec != nil {
		req.ids = append(req.ids, spec.MustSay...)
	}
	for id, text := range approvedText {
		req.resolved[id] = text
	}
	return context.WithValue(ctx, requirementsKey{}, req)
}

// StreamGuard is run against the INTERNAL buffer, never the HTTP response.
// It is a fast backstop, not a substitute for Finalize's contextual checks.
func (s *Service) StreamGuard(plan contract.ResponsePlan, risk contract.RiskLevel) ports.StreamGuard {
	return &guard{service: s, plan: plan, risk: risk}
}

type guard struct {
	service *Service
	plan    contract.ResponsePlan
	risk    contract.RiskLevel
	text    strings.Builder
	blocked bool
}

var _ ports.StreamGuard = (*guard)(nil)

func (g *guard) Push(token string) (bool, *contract.Violation) {
	if g.blocked {
		return false, &contract.Violation{Code: "OV-031", Severity: "block"}
	}
	if !utf8.ValidString(token) || g.text.Len()+len(token) > 80000 {
		g.blocked = true
		return false, &contract.Violation{Code: "OV-031", Severity: "block"}
	}
	_, _ = g.text.WriteString(token)
	text := g.text.String()
	if g.service == nil {
		g.blocked = true
		return false, &contract.Violation{Code: "OV-031", Severity: "block"}
	}
	// Locale is absent from StreamGuard's frozen port. Check all reviewed
	// locale rule sets; Finalize selects the correct locale using WCtx.
	if strings.Contains(text, "<<PLAN_UNSAFE>>") {
		g.blocked = true
		return false, &contract.Violation{Code: "OV-030", Severity: "block"}
	}
	for _, r := range g.service.locales {
		for _, kind := range []string{"diagnosis", "medication", "therapist_identity", "guarantee", "crisis_forbidden", "internal_leak"} {
			if (kind == "crisis_forbidden" && !riskAtLeast(g.risk, contract.RiskHigh)) || !matches(r.patterns[kind], text) {
				continue
			}
			g.blocked = true
			return false, &contract.Violation{Code: supportedPatternKeys[kind], Severity: "block"}
		}
	}
	if idLeakRE.MatchString(text) {
		g.blocked = true
		return false, &contract.Violation{Code: "OV-010", Severity: "block"}
	}
	if g.service.hasUnauthorizedContact(text, true) {
		g.blocked = true
		return false, &contract.Violation{Code: "OV-011", Severity: "block"}
	}
	// Approved crisis templates may ask a single immediate-safety
	// question even when the non-crisis restricted plan forbids questions.
	if count, _ := questions(text, rules{}); count > 1 || (count != 0 && !g.plan.Question.Allowed && !riskAtLeast(g.risk, contract.RiskHigh)) {
		g.blocked = true
		code := "OV-001"
		if count == 1 {
			code = "OV-002"
		}
		return false, &contract.Violation{Code: code, Severity: "repair"}
	}
	return true, nil
}

// Finalize always produces a bounded report with stable, non-content error
// codes. On a classifier failure it returns an error rather than silently
// accepting dangerous text. A failed/blocked draft must never be committed.
func (s *Service) Finalize(ctx context.Context, text string, plan contract.ResponsePlan, wc contract.WorkingContext, risk contract.RiskLevel, crisis *contract.CrisisPlan) (contract.ValidationReport, error) {
	report := contract.ValidationReport{Violations: []contract.Violation{}, Extracted: contract.Extracted{QuestionCount: 0}, Action: contract.VARegenerate}
	if err := ctx.Err(); err != nil {
		return report, err
	}
	if s == nil || !risk.Valid() {
		return report, ErrAssets
	}
	lang := strings.TrimSpace(wc.Turn.Lang)
	if wc.Prefs.Language != "" {
		lang = strings.TrimSpace(wc.Prefs.Language)
	}
	r, ok := s.forLocale(lang)
	if !ok {
		return report, ErrCoverage
	}
	seen := map[string]bool{}
	add := func(code, severity string) {
		if !seen[code] {
			report.Violations = append(report.Violations, contract.Violation{Code: code, Severity: severity})
			seen[code] = true
		}
	}
	normalized := strings.TrimSpace(text)
	if !utf8.ValidString(text) || len(text) > 80000 || normalized == "" {
		add("OV-031", "repair")
	}
	if finish, _ := ctx.Value(finishKey{}).(string); (finish == "length" || finish == "max_tokens") && !hasSentenceEnd(normalized) {
		add("OV-031", "repair")
	}
	if strings.Contains(normalized, "<<PLAN_UNSAFE>>") {
		report.Extracted.Sentinel = "PLAN_UNSAFE"
		add("OV-030", "block")
	}
	count, question := questions(normalized, r)
	report.Extracted.QuestionCount = count
	if question != "" {
		report.Extracted.QuestionText = &question
	}
	if count > 1 {
		add("OV-001", "repair")
	}
	if count > 0 && crisis == nil && (!plan.Question.Allowed || plan.Question.MaxCount == 0 || !wc.Eligibility.QuestionAllowed) {
		add("OV-002", "repair")
	}
	if count > 0 && crisis == nil && plan.Question.MaxCount > 0 && count > plan.Question.MaxCount {
		add("OV-001", "repair")
	}
	for _, pair := range []struct{ kind, code, severity string }{
		{"diagnosis", "OV-003", "block"}, {"medication", "OV-004", "block"},
		{"guarantee", "OV-005", "repair"}, {"therapist_identity", "OV-006", "block"},
		{"internal_leak", "OV-010", "block"}, {"dependency", "OV-016", "repair"},
		{"invalidating", "OV-017", "repair"},
	} {
		if matches(r.patterns[pair.kind], normalized) {
			add(pair.code, pair.severity)
		}
	}
	if idLeakRE.MatchString(normalized) {
		add("OV-010", "block")
	}
	if s.hasUnauthorizedContact(normalized, false) {
		add("OV-011", "block")
	}
	avoidAdvice := false
	for _, v := range plan.MustAvoid {
		if v == "advice" {
			avoidAdvice = true
		}
	}
	if avoidAdvice && matches(r.patterns["directive_advice"], normalized) {
		add("OV-007", "repair")
	}
	if plan.Protocol == nil && matches(r.patterns["technique"], normalized) {
		add("OV-015", "repair")
	}
	if plan.Protocol != nil {
		req, ok := ctx.Value(requirementsKey{}).(requiredContent)
		if !ok || len(req.ids) == 0 {
			add("OV-014", "repair")
		} else {
			for _, id := range req.ids {
				phrase := strings.TrimSpace(req.resolved[id])
				if phrase == "" || !strings.Contains(strings.ToLower(normalized), strings.ToLower(phrase)) {
					add("OV-014", "repair")
				}
			}
		}
	}
	// Enforce knowledge provenance on retrieved items (no hidden/extra IDs)
	// and inference hedging within the same sentence. Do not guess new facts.
	permitted := map[contract.ID]bool{}
	for _, id := range plan.PermittedContext {
		permitted[id] = true
	}
	referenced := map[contract.ID]bool{}
	for _, mem := range wc.Retrieved {
		if !contract.ValidID(mem.ID) || !claimMatches(normalized, mem.Text) {
			continue
		}
		if !permitted[mem.ID] || wc.RetrievalMeta.Degraded || !wc.Eligibility.MemoryAvailable {
			add("OV-009", "repair")
			continue
		}
		referenced[mem.ID] = true
		inferred := mem.Epistemic == contract.EpInferred
		hedged := hedgeNear(normalized, mem.Text, r.hedges)
		if inferred && !hedged {
			add("OV-008", "repair")
		}
		id := mem.ID
		report.Extracted.ClaimsAboutUser = append(report.Extracted.ClaimsAboutUser,
			contract.UserClaim{Text: mem.Text, Source: &id, Hedged: hedged, Epistemic: mem.Epistemic})
	}
	if wc.RetrievalMeta.Degraded && recallRE.MatchString(normalized) {
		add("OV-009", "repair")
	}
	// Focus-thread inferred data is not a statement of user fact either.
	if wc.FocusThread != nil {
		for _, slot := range wc.FocusThread.Slots {
			if slot.Status == contract.EpInferred && claimMatches(normalized, slot.Value) && !hedgeNear(normalized, slot.Value, r.hedges) {
				add("OV-008", "repair")
			}
		}
		for _, hyp := range wc.FocusThread.Hypotheses {
			if claimMatches(normalized, hyp.Text) && !hedgeNear(normalized, hyp.Text, r.hedges) {
				add("OV-008", "repair")
			}
		}
	}
	for id := range referenced {
		report.Extracted.ReferencedContext = append(report.Extracted.ReferencedContext, id)
	}
	sort.Slice(report.Extracted.ReferencedContext, func(i, j int) bool {
		return report.Extracted.ReferencedContext[i].String() < report.Extracted.ReferencedContext[j].String()
	})
	target := lengthTarget(plan.Length)
	if target > 0 {
		n := len(splitWordsRE.FindAllString(normalized, -1))
		if n > target*2 {
			add("OV-012", "repair")
		} else if float64(n) > float64(target)*1.4 || float64(n) < float64(target)*0.6 {
			add("OV-012", "warn")
		}
	}
	if !languageMatches(normalized, lang) || (wc.Turn.Lang != "" && wc.Prefs.Language != "" && !languageMatches(normalized, wc.Turn.Lang)) {
		add("OV-013", "repair")
	}
	if riskAtLeast(risk, contract.RiskHigh) || crisis != nil {
		if crisis == nil || !crisis.Buffered || strings.TrimSpace(crisis.TemplateText) == "" || len(crisis.Resources) == 0 ||
			!containsRequiredElements(crisis.RequiredElements) || strings.TrimSpace(crisis.TemplateText) != normalized {
			// Until clinically reviewed per-element matchers exist, a crisis
			// response MUST equal the vetted, rendered template. This is an
			// intentional conservative rule, not a generative clinical policy.
			add("OV-020", "repair")
		}
		if crisis != nil {
			for _, item := range crisis.Resources {
				if _, ok := s.contacts[normalizeContact(item.Contact)]; !ok || !contactPresent(normalized, item.Contact) {
					add("OV-020", "repair")
				}
			}
		}
		if matches(r.patterns["crisis_forbidden"], normalized) {
			add("OV-021", "block")
		}
	}
	// Do not spend another model call on text already deterministically blocked.
	if s.classifier != nil && (len(report.Violations) == 0 || allWarnings(report.Violations)) {
		classifyCtx, cancel := context.WithTimeout(ctx, s.classifierTimeout)
		scores, err := s.classifier.Scores(classifyCtx, normalized, wc.Recent, lang)
		cancel()
		if err != nil {
			return report, errors.Join(ErrClassification, err)
		}
		report.Scores = map[string]float32{}
		for key, rule := range scoreToRule {
			value, ok := scores[key]
			if !ok || math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) || value < 0 || value > 1 {
				return report, ErrClassification
			}
			report.Scores[key] = value
			if value >= s.thresholds[key] {
				if key != "directive_advice" || avoidAdvice {
					add(rule, "repair")
				}
			}
		}
		if len(scores) != len(scoreToRule) {
			return report, ErrClassification
		}
	}
	if seen["OV-030"] {
		report.Action = contract.VACrisis
	}
	if len(report.Violations) == 0 || allWarnings(report.Violations) {
		report.Passed, report.Action = true, contract.VAAccept
	}
	return report, nil
}

func matches(patterns []*regexp.Regexp, text string) bool {
	for _, p := range patterns {
		if p.MatchString(text) {
			return true
		}
	}
	return false
}
func allWarnings(items []contract.Violation) bool {
	for _, v := range items {
		if v.Severity != "warn" {
			return false
		}
	}
	return true
}
func lengthTarget(size contract.Length) int {
	switch size {
	case contract.LengthMinimal:
		return 25
	case contract.LengthShort:
		return 60
	case contract.LengthMedium:
		return 120
	case contract.LengthLong:
		return 220
	default:
		return 0
	}
}
func riskAtLeast(a, b contract.RiskLevel) bool {
	rank := map[contract.RiskLevel]int{contract.RiskNone: 0, contract.RiskLow: 1, contract.RiskElevated: 2, contract.RiskHigh: 3, contract.RiskImminent: 4}
	return rank[a] >= rank[b]
}
func containsRequiredElements(have []string) bool {
	set := map[string]bool{}
	for _, h := range have {
		set[h] = true
	}
	for _, h := range []string{"acknowledge", "care", "ask_safety", "resources", "invite_continue"} {
		if !set[h] {
			return false
		}
	}
	return true
}
func hasSentenceEnd(s string) bool {
	s = strings.TrimRight(s, " \r\n\t\"”»")
	if s == "" {
		return false
	}
	r := []rune(s)
	return strings.ContainsRune(".!?؟", r[len(r)-1])
}

// A provider may split a whitelisted contact into arbitrary tokens. Do not
// reject an unfinished prefix on the stream: Finalize rechecks the full text.
// Completed unapproved contacts are blocked immediately on the private stream.
func (s *Service) hasUnauthorizedContact(text string, streaming bool) bool {
	isAllowed := func(candidate string, end int) bool {
		key := normalizeContact(candidate)
		if _, ok := s.contacts[key]; ok {
			return true
		}
		if streaming && end == len(text) {
			// Only defer a true contact prefix; unmatched, completed-looking
			// names are rejected when a word delimiter arrives.
			for approved := range s.contacts {
				if strings.HasPrefix(approved, key) {
					return true
				}
			}
		}
		return false
	}
	for _, indexes := range urlRE.FindAllStringIndex(text, -1) {
		candidate := strings.TrimRight(text[indexes[0]:indexes[1]], ".,;!?؟")
		if !isAllowed(candidate, indexes[1]) {
			return true
		}
	}
	for _, indexes := range emailRE.FindAllStringIndex(text, -1) {
		if !isAllowed(text[indexes[0]:indexes[1]], indexes[1]) {
			return true
		}
	}
	for _, indexes := range handleRE.FindAllStringIndex(text, -1) {
		if !isAllowed(strings.TrimSpace(text[indexes[0]:indexes[1]]), indexes[1]) {
			return true
		}
	}
	for _, indexes := range phoneRE.FindAllStringIndex(text, -1) {
		candidate := text[indexes[0]:indexes[1]]
		if len(normalizeContact(candidate)) >= 7 && !isAllowed(candidate, indexes[1]) {
			return true
		}
	}
	return false
}
func contactPresent(text, contact string) bool {
	normalized := normalizeContact(contact)
	if normalized == "" {
		return false
	}
	if strings.Contains(contact, "://") || strings.Contains(contact, "@") {
		return strings.Contains(strings.ToLower(text), strings.ToLower(contact))
	}
	// Numeric short codes are checked as bounded digit groups; longer
	// formatted phone numbers may have whitespace/punctuation in the reply.
	for _, candidate := range regexp.MustCompile(`[+0-9][+0-9\-(). ]*[0-9]`).FindAllString(text, -1) {
		if normalizeContact(candidate) == normalized {
			return true
		}
	}
	return false
}

func languageMatches(text, lang string) bool {
	var latin, arabic int
	for _, c := range text {
		if unicode.Is(unicode.Latin, c) {
			latin++
		}
		if unicode.Is(unicode.Arabic, c) {
			arabic++
		}
	}
	base := strings.Split(lang, "-")[0]
	// Script checks only when confident. Distinguishing en/de/tr requires an
	// approved language classifier; no false certainty is invented here.
	switch base {
	case "fa", "ar":
		return arabic > 0 || latin < 4
	case "en", "de", "tr", "fr", "es":
		return latin > 0 || arabic < 4
	default:
		return true
	}
}

// questions excludes quoted examples and collapses adjacent ?? punctuation.
func questions(text string, r rules) (int, string) {
	var count int
	var last string
	quoted := false
	previous := rune(0)
	beginning := 0
	for i, c := range text {
		if c == '"' || c == '“' || c == '”' || c == '«' || c == '»' {
			quoted = !quoted
		}
		if (c == '?' || c == '؟') && !quoted && previous != '?' && previous != '؟' {
			count++
			last = strings.TrimSpace(text[beginning : i+utf8.RuneLen(c)])
		}
		if c == '.' || c == '!' || c == '?' || c == '؟' || c == '\n' {
			beginning = i + utf8.RuneLen(c)
		}
		previous = c
	}
	// Locale interrogatives without a question mark are included only when
	// explicitly provided by the reviewed locale lexicon.
	if count == 0 {
		for _, line := range regexp.MustCompile(`[.!\n؟?]+`).Split(text, -1) {
			if matches(r.questions, strings.TrimSpace(line)) {
				count++
				last = strings.TrimSpace(line)
			}
		}
	}
	if count > 0 && len([]rune(last)) > 160 {
		last = string([]rune(last)[:160])
	}
	return count, last
}

func claimMatches(reply, item string) bool {
	words := distinctWords(item)
	if len(words) < 2 {
		return false
	}
	target := distinctWords(reply)
	hits := 0
	for w := range words {
		if target[w] {
			hits++
		}
	}
	return hits >= 2 && float64(hits)/float64(len(words)) >= 0.6
}
func distinctWords(text string) map[string]bool {
	set := map[string]bool{}
	for _, w := range splitWordsRE.FindAllString(strings.ToLower(text), -1) {
		// English suffix folding is deliberately limited; do not turn a
		// weak lexical overlap into an asserted psychological inference.
		if len(w) > 5 && strings.HasSuffix(w, "ing") {
			w = strings.TrimSuffix(w, "ing")
		}
		if len(w) > 4 && strings.HasSuffix(w, "ed") {
			w = strings.TrimSuffix(w, "ed")
		}
		if len(w) > 3 && strings.HasSuffix(w, "s") {
			w = strings.TrimSuffix(w, "s")
		}
		if len([]rune(w)) >= 3 {
			set[w] = true
		}
	}
	return set
}
func hedgeNear(reply, item string, markers []string) bool {
	words := distinctWords(item)
	if len(words) == 0 {
		return false
	}
	for _, sentence := range regexp.MustCompile(`[.!?؟\n]+`).Split(strings.ToLower(reply), -1) {
		set := distinctWords(sentence)
		hits := 0
		for w := range words {
			if set[w] {
				hits++
			}
		}
		if hits < 2 || float64(hits)/float64(len(words)) < 0.6 {
			continue
		}
		for _, hedge := range markers {
			if strings.Contains(sentence, hedge) {
				return true
			}
		}
	}
	return false
}
