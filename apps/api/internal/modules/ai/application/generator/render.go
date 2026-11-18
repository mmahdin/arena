package generator

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

func escape(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}
func tag(name, text string) string { return "<" + name + ">" + text + "</" + name + ">\n" }
func data(name string, v any) string {
	var b strings.Builder
	encoder := json.NewEncoder(&b)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(v)
	return tag(name, escape(strings.TrimSuffix(b.String(), "\n")))
}

// Same conservative, provider-independent approximation as WP-08/10. A
// deployment must calibrate this against its chosen provider tokenizers.
func estimate(s string) int { return (utf8.RuneCountInString(s) + 2) / 3 }
func invalid() error        { return &c.Error{Code: c.ErrValidation, Step: "generator_input"} }
func budgetError() error {
	return &c.Error{Code: c.ErrBudget, Step: "generator_render", Degrades: true}
}

var regenRE = regexp.MustCompile(`^(OV-(?:001|002|003|004|005|006|007|008|009|010|011|012|013|014|015|016|017|020|021|030|031))(,OV-(?:001|002|003|004|005|006|007|008|009|010|011|012|013|014|015|016|017|020|021|030|031)){0,3}(?:: [A-Za-z ,.-]{1,150})?$`)

type prompt struct {
	system   []ports.Block
	messages []ports.ChatMessage
}

func (a *Assets) render(ctx context.Context, in ports.GenInput, route c.Route) (prompt, error) {
	if !in.Objective.Valid() || !in.Risk.Valid() || !c.ValidID(in.Turn.MessageID) || !c.ValidID(in.Turn.TurnID) || !utf8.ValidString(in.Turn.Text) || strings.TrimSpace(in.Turn.Text) == "" {
		return prompt{}, invalid()
	}
	lang := in.Turn.Lang
	if in.Prefs.Language != "" {
		lang = in.Prefs.Language
	}
	locale, ok := a.locale(lang)
	if !ok {
		return prompt{}, &c.Error{Code: c.ErrBundle, Step: "generator_locale"}
	}
	if in.RegenNote != nil && (!regenRE.MatchString(*in.RegenNote) || estimate(*in.RegenNote) > 60) {
		return prompt{}, invalid()
	}
	p := prompt{}
	add := func(name, text string, cache bool, cap int) error {
		if !utf8.ValidString(text) || estimate(text) > cap {
			return budgetError()
		}
		p.system = append(p.system, ports.Block{Name: name, Text: text, Cache: cache})
		return nil
	}
	if e := add("persona_and_format", tag("persona_and_format", a.system), true, 400); e != nil {
		return p, e
	}
	crisis := in.Crisis != nil
	if crisis != (route == c.RouteCrisis) || (!crisis && in.Risk.Rank() >= c.RiskHigh.Rank()) || crisis && in.Objective != c.ObjSafetySupport {
		return p, invalid()
	}
	if crisis {
		// Re-derive from the pinned WP-06 assets. Arbitrary resource contacts,
		// stale resources and caller-supplied crisis instructions are rejected.
		matched := false
		keys := make([]string, 0, len(a.policy.Categories))
		for key := range a.policy.Categories {
			keys = append(keys, key)
		}
		sortStrings(keys)
		for _, key := range keys {
			if a.policy.Categories[key].TemplateFamily != in.Crisis.TemplateFamily {
				continue
			}
			approved, e := a.planner.Plan(ctx, c.SafetySignal{Level: c.RiskHigh, Categories: []c.RiskCategory{c.RiskCategory(key)}}, lang, c.RiskState{Level: c.RiskHigh})
			if e == nil && reflect.DeepEqual(approved, *in.Crisis) {
				matched = true
				break
			}
		}
		if !matched {
			return p, &c.Error{Code: c.ErrBundle, Step: "generator_crisis"}
		}
		if e := add("crisis_policy", a.crisis, true, 600); e != nil {
			return p, e
		}
	} else if e := add("communication_policy", tag("communication_policy", a.communication), true, 600); e != nil {
		return p, e
	}
	if e := add("prohibitions", tag("prohibitions", escape(a.prohibitions)), true, 250); e != nil {
		return p, e
	}
	if crisis {
		// WP-07 currently requires exact reviewed crisis text. Supply that text
		// as escaped data, so the model can realize the existing validator
		// contract without guessing a clinical script from element names.
		text := "<crisis_template_family id=\"" + escape(in.Crisis.TemplateFamily) + "\" copy_verbatim=\"true\">" + escape(in.Crisis.TemplateText) + "</crisis_template_family>\n" + data("required_elements", in.Crisis.RequiredElements) + data("forbidden", in.Crisis.Forbidden)
		text += "<resources copy_verbatim=\"true\">\n"
		for _, r := range in.Crisis.Resources {
			text += "<resource id=\"" + escape(r.ID) + "\" locale=\"" + escape(r.Locale) + "\">" + data("name", r.Name) + data("kind", r.Kind) + data("contact", r.Contact) + data("hours", r.Hours) + "</resource>\n"
		}
		text += "</resources>\n"
		if e := add("crisis_plan", text, false, 1000); e != nil {
			return p, e
		}
		// Crisis has no permission to expose unrelated recalled content or skills.
		if len(in.Permitted) > 0 || len(in.Plan.PermittedContext) > 0 || in.Spec != nil || in.Plan.Protocol != nil || in.Plan.VoiceHypothesisID != nil {
			return p, invalid()
		}
	} else {
		if e := a.plan(in, route, locale, &p); e != nil {
			return p, e
		}
	}
	contextText, err := permitted(in)
	if err != nil {
		return p, err
	}
	if e := add("permitted_context", tag("permitted_context", contextText), false, 350); e != nil {
		return p, e
	}
	if e := add("prefs", data("prefs", in.Prefs), false, 80); e != nil {
		return p, e
	}
	recent := append([]c.MsgView(nil), in.Recent...)
	seen := map[c.ID]bool{in.Turn.MessageID: true}
	for _, m := range recent {
		if !c.ValidID(m.ID) || seen[m.ID] || m.Role != c.RoleUser && m.Role != c.RoleAssistant || !utf8.ValidString(m.Text) {
			return p, invalid()
		}
		seen[m.ID] = true
	}
	if len(recent) > 6 {
		recent = recent[len(recent)-6:]
	}
	recentText := func() string {
		var s strings.Builder
		for _, m := range recent {
			s.WriteString("<message id=\"" + m.ID.String() + "\" role=\"" + escape(string(m.Role)) + "\" age=\"" + escape(m.Age) + "\">" + escape(m.Text) + "</message>\n")
		}
		return tag("recent", s.String())
	}
	current := "<user_message id=\"" + in.Turn.MessageID.String() + "\" lang=\"" + escape(in.Turn.Lang) + "\" local_time=\"" + escape(in.Turn.LocalTime) + "\">" + escape(in.Turn.Text) + "</user_message>\n"
	if estimate(current) > 400 {
		return p, budgetError()
	}
	if in.RegenNote != nil {
		// The caller's hint is never forwarded. Only recognized OV codes and
		// this fixed technical correction can reach the provider.
		codes := strings.SplitN(*in.RegenNote, ":", 2)[0]
		current += tag("regen_note", escape(codes+": revise to satisfy the approved plan and safety rules"))
	}
	cap := a.budgets.Tokens.GenIn
	if b, ok := c.BudgetFromContext(ctx); ok && b.MaxTokensIn > 0 && b.MaxTokensIn < cap {
		cap = b.MaxTokensIn
	}
	for {
		r := recentText()
		size := estimate(current) + estimate(r)
		for _, b := range p.system {
			size += estimate(b.Text)
		}
		if estimate(r) <= 700 && size <= cap {
			p.system = append(p.system, ports.Block{Name: "recent", Text: r})
			p.messages = []ports.ChatMessage{{Role: c.RoleUser, Content: current}}
			return p, nil
		}
		if len(recent) == 0 {
			return p, budgetError()
		}
		recent = recent[1:]
	}
}

func (a *Assets) plan(in ports.GenInput, route c.Route, locale string, p *prompt) error {
	plan := in.Plan
	fast := route == c.RouteFast && in.Objective == c.ObjAnswerDirect && reflect.DeepEqual(plan, c.ResponsePlan{Question: c.QuestionPlan{Allowed: false, MaxCount: 0}})
	if !fast && (!plan.Length.Valid() || !plan.Tone.Valid() || plan.Skills.Primary == "") {
		return invalid()
	}
	if in.Objective == c.ObjSafetySupport || len(plan.Skills.Secondary) > 2 || plan.Question.MaxCount < 0 || plan.Question.MaxCount > 1 || plan.Question.Allowed != (plan.Question.MaxCount == 1) {
		return invalid()
	}
	if (in.Objective == c.ObjOfferIntervention || in.Objective == c.ObjContinueIntervention) && plan.Protocol == nil {
		return invalid()
	}
	for _, v := range plan.MustAvoid {
		if !includes(a.avoids, v) {
			return invalid()
		}
	}
	if plan.Question.Purpose != "" && !includes([]string{"close_gap", "invite_detail", "check_readiness", "confirm", "choose", "none"}, plan.Question.Purpose) {
		return invalid()
	}
	words := map[c.Length]int{c.LengthMinimal: 25, c.LengthShort: 60, c.LengthMedium: 120, c.LengthLong: 220}[plan.Length]
	if fast {
		words = 25
	}
	attrs := ""
	if plan.VoiceHypothesisID != nil {
		if in.Objective != c.ObjProvideInsight || !includes(plan.PermittedContext, *plan.VoiceHypothesisID) {
			return invalid()
		}
		found := false
		for _, m := range in.Permitted {
			if m.ID == *plan.VoiceHypothesisID && m.Kind == c.MemHypothesis && m.Status == c.MemSupported && m.Epistemic == c.EpInferred {
				found = true
			}
		}
		if !found {
			return invalid()
		}
		attrs = " voice_hypothesis_id=\"" + plan.VoiceHypothesisID.String() + "\""
	}
	text := "<plan" + attrs + ">" + data("objective", in.Objective) + data("tone", plan.Tone) + data("length", plan.Length) + data("word_target", words) + data("question", plan.Question) + data("must_avoid", plan.MustAvoid)
	for _, id := range plan.PermittedContext {
		text += "<context_ref id=\"" + id.String() + "\"/>"
	}
	text += "</plan>\n"
	if estimate(text) > 250 {
		return budgetError()
	}
	p.system = append(p.system, ports.Block{Name: "plan", Text: text}, ports.Block{Name: "skills_selected", Text: "<skills_selected>\n"})
	ids := []string{}
	if !fast {
		ids = append(ids, plan.Skills.Primary)
		ids = append(ids, plan.Skills.Secondary...)
	}
	cardSize := 0
	for i, id := range ids {
		skill, ok := a.skills[id]
		if !ok || includes(ids[:i], id) {
			return invalid()
		}
		for _, other := range ids {
			if includes(skill.ConflictsWith, other) {
				return invalid()
			}
		}
		card := skill.CardText
		if locale != "en" {
			localized, ok := skill.Locales[locale]
			if !ok || localized.CardText == "" {
				return &c.Error{Code: c.ErrBundle, Step: "generator_skill_locale"}
			}
			card = localized.CardText
		}
		text := "<skill id=\"" + escape(id) + "\">" + escape(card) + "</skill>\n"
		if estimate(text) > 150 {
			return budgetError()
		}
		cardSize += estimate(text)
		p.system = append(p.system, ports.Block{Name: "skill." + id, Text: text, Cache: true})
	}
	if cardSize > 450 {
		return budgetError()
	}
	p.system = append(p.system, ports.Block{Name: "skills_selected_end", Text: "</skills_selected>\n"})
	text, err := a.protocol(in, locale)
	if err != nil {
		return err
	}
	if estimate(text) > 300 {
		return budgetError()
	}
	p.system = append(p.system, ports.Block{Name: "protocol_step", Text: text})
	return nil
}

func permitted(in ports.GenInput) (string, error) {
	if len(in.Plan.PermittedContext) != len(in.Permitted) {
		return "", invalid()
	}
	items := map[c.ID]c.MemoryView{}
	for _, m := range in.Permitted {
		if !c.ValidID(m.ID) || !utf8.ValidString(m.Text) || m.Text == "" || m.Epistemic == c.EpUnknown || !m.Epistemic.Valid() || m.Sensitivity == c.SensHighly {
			return "", invalid()
		}
		if _, exists := items[m.ID]; exists {
			return "", invalid()
		}
		items[m.ID] = m
		if !m.Verbatim && (!m.Kind.Valid() || !includes([]c.MemoryStatus{c.MemActive, c.MemProposed, c.MemSupported}, m.Status) || m.EvidenceN < 1) {
			return "", invalid()
		}
		if m.Kind == c.MemHypothesis && (m.Epistemic != c.EpInferred || m.Status != c.MemSupported || in.Plan.VoiceHypothesisID == nil || *in.Plan.VoiceHypothesisID != m.ID) {
			return "", invalid()
		}
	}
	var s strings.Builder
	seen := map[c.ID]bool{}
	for _, id := range in.Plan.PermittedContext {
		m, ok := items[id]
		if !ok || seen[id] {
			return "", invalid()
		}
		seen[id] = true
		s.WriteString("<memory id=\"" + id.String() + "\" kind=\"" + escape(string(m.Kind)) + "\" epistemic=\"" + escape(string(m.Epistemic)) + "\" status=\"" + escape(string(m.Status)) + "\" age=\"" + escape(m.Age) + "\" verbatim=\"" + boolString(m.Verbatim) + "\">" + escape(m.Text) + "</memory>\n")
	}
	return s.String(), nil
}
func boolString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
func sortStrings(v []string) { sort.Strings(v) }
