package brain

import (
	"encoding/json"
	"golang.org/x/text/unicode/norm"
	"strings"
	"unicode"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type policyFacts struct {
	owned, hypotheses map[c.ID]bool
	tags              map[string]bool
	offer, voice      bool
}

var _ ports.PolicyGate = (*Assets)(nil)

// Gate is a pure deterministic boundary for typed callers. Database-dependent
// permissions default to deny here; Service.Decide supplies verified facts.
func (a *Assets) Gate(in ports.GateInput) (c.BrainDecision, []c.DeltaRejection, *ports.RouteOverride) {
	d, r := a.validate(in.Decision, ports.BrainInput{WCtx: in.WCtx, Safety: in.Safety}, policyFacts{})
	if d.RiskRead != nil && d.RiskRead.Level.Rank() >= c.RiskHigh.Rank() {
		return d, r, &ports.RouteOverride{Route: c.RouteCrisis, Reason: "SF-005", ForceBufferedOutput: true}
	}
	return d, r, nil
}
func (a *Assets) fallback(w c.WorkingContext) c.BrainDecision {
	allowed := w.Eligibility.QuestionAllowed && w.Session.ConsecutiveQuestions < a.gate.Questions.MaxConsecutiveQuestionTurns && a.gate.Questions.MaxPerReply == 1
	count := 0
	if allowed {
		count = 1
	}
	return c.BrainDecision{SchemaVersion: c.BrainSchemaVersion,
		TurnAnalysis:  c.TurnAnalysis{Intent: c.IntentOther, MessageType: c.MessageDisclosure, ThreadRelation: c.RelContinues, Emotions: []c.EmotionRead{}, PastRefs: []c.PastRef{}, Ambiguities: []c.Ambiguity{}, UserReadiness: c.ConfLow},
		Understanding: c.UnderstandingUpdate{Slots: []c.SlotUpdate{}, Hypotheses: []c.HypothesisSignal{}, Unknowns: []string{}},
		Decision:      c.Decision{Primary: c.ObjReflect, Rationale: []string{"brain_fallback"}, Alternatives: []c.Objective{}, Confidence: c.ConfLow},
		Plan:          c.ResponsePlan{Skills: c.SkillSelection{Primary: "reflect_emotion", Secondary: []string{}}, Question: c.QuestionPlan{Allowed: allowed, MaxCount: count, Purpose: "invite_detail"}, Tone: c.ToneWarmSteady, Length: c.LengthShort, PermittedContext: []c.ID{}, MustAvoid: []string{"advice", "claiming_recall", "intervention"}}}
}
func (a *Assets) validate(proposed c.BrainDecision, in ports.BrainInput, f policyFacts) (c.BrainDecision, []c.DeltaRejection) {
	// Copy all nested slices/maps; neither gates nor prompt trimming mutate inputs.
	raw, _ := json.Marshal(proposed)
	var d c.BrainDecision
	_ = json.Unmarshal(raw, &d)
	w := in.WCtx
	r := []c.DeltaRejection{}
	reject := func(path, code string) { r = append(r, c.DeltaRejection{Path: path, RuleCode: code}) }
	clamp := func(o c.Objective, code string) {
		if d.Decision.Primary != o {
			d.Decision.Primary = o
			d.Plan.Skills = c.SkillSelection{Primary: "reflect_emotion", Secondary: []string{}}
			reject("decision.primary_objective", code)
		}
	}
	avoid := func(tag string) {
		if !contains(d.Plan.MustAvoid, tag) {
			d.Plan.MustAvoid = append(d.Plan.MustAvoid, tag)
		}
	}
	ids := contextIDs(w)
	msgs := contextMessages(w)
	hyps := hypotheses(w)
	if !d.Decision.Primary.Valid() || d.Decision.Primary == c.ObjSafetySupport {
		clamp(c.ObjReflect, "BR-002")
		reject("decision.primary_objective", "BR-012")
	}
	if _, ok := a.skills[d.Plan.Skills.Primary]; !ok {
		d.Plan.Skills.Primary = "reflect_emotion"
		reject("response_plan.skills", "BR-003")
	}
	secondary := []string{}
	for _, id := range d.Plan.Skills.Secondary {
		sk, ok := a.skills[id]
		primary := a.skills[d.Plan.Skills.Primary]
		if !ok || id == d.Plan.Skills.Primary || contains(secondary, id) || contains(primary.ConflictsWith, id) || contains(sk.ConflictsWith, primary.ID) {
			reject("response_plan.skills.secondary", "BR-003")
			continue
		}
		if len(secondary) < 2 {
			secondary = append(secondary, id)
		}
	}
	d.Plan.Skills.Secondary = secondary
	permitted := []c.ID{}
	for _, id := range d.Plan.PermittedContext {
		if !ids[id] {
			reject("response_plan.permitted_context_ids", "BR-004")
			continue
		}
		if !contains(permitted, id) {
			permitted = append(permitted, id)
		}
	}
	d.Plan.PermittedContext = permitted
	slots := []c.SlotUpdate{}
	seen := map[string]bool{}
	for _, s := range d.Understanding.Slots {
		if !contains(a.vocab.Slots, s.Slot) || seen[s.Slot] {
			reject("understanding_update.slots", "BR-007")
			continue
		}
		seen[s.Slot] = true
		evidence := []c.ID{}
		all := len(s.Evidence) > 0
		support := false
		for _, id := range s.Evidence {
			m, ok := msgs[id]
			if !ok {
				all = false
				reject("understanding_update.slots.evidence", "BR-005")
				continue
			}
			if f.owned[id] && m.Role == c.RoleUser {
				support = support || supports(m.Text, s.Value)
			} else {
				all = false
			}
			if !contains(evidence, id) {
				evidence = append(evidence, id)
			}
		}
		if len(evidence) == 0 {
			reject("understanding_update.slots", "BR-005")
			continue
		}
		if s.Status == c.EpReported && (!all || !support) {
			s.Status = c.EpInferred
			reject("understanding_update.slots.status", "BR-006")
		}
		// Evidence from assistant text may inform reasoning but cannot be proposed as
		// a user slot, even as inferred. WP-08 independently checks this again.
		valid := []c.ID{}
		for _, id := range evidence {
			if f.owned[id] && msgs[id].Role == c.RoleUser {
				valid = append(valid, id)
			}
		}
		if len(valid) == 0 {
			reject("understanding_update.slots", "BR-005")
			continue
		}
		if len(valid) != len(evidence) {
			s.Status = c.EpInferred
		}
		s.Evidence = valid
		slots = append(slots, s)
	}
	d.Understanding.Slots = slots
	signals := []c.HypothesisSignal{}
	for _, h := range d.Understanding.Hypotheses {
		good := len(h.Evidence) > 0
		for _, id := range h.Evidence {
			if m, ok := msgs[id]; !ok || m.Role != c.RoleUser || !f.owned[id] {
				good = false
			}
		}
		if !good {
			reject("understanding_update.hypotheses.evidence", "BR-005")
			continue
		}
		if h.Op == c.HypPropose {
			if h.Tag == nil || !contains(a.vocab.HypothesisTags, *h.Tag) || !f.tags[*h.Tag] {
				reject("understanding_update.hypotheses", "BR-021")
				continue
			}
		}
		if h.Op != c.HypPropose && (h.ID == nil || !hyps[*h.ID] || !f.hypotheses[*h.ID]) {
			reject("understanding_update.hypotheses", "ST-007")
			continue
		}
		signals = append(signals, h)
	}
	d.Understanding.Hypotheses = signals
	// References and correction targets may only identify sources actually shown.
	for i := range d.TurnAnalysis.PastRefs {
		p := &d.TurnAnalysis.PastRefs[i]
		good := []c.ID{}
		for _, id := range p.ResolvedTo {
			if ids[id] && !w.RetrievalMeta.Degraded {
				good = append(good, id)
			} else {
				p.Ambiguous = true
				reject("turn_analysis.past_references", "BR-004")
			}
		}
		p.ResolvedTo = good
		if w.RetrievalMeta.AmbiguousReference {
			p.Ambiguous = true
			p.ResolvedTo = []c.ID{}
		}
	}
	if d.TurnAnalysis.Intent == c.IntentDisagreeCorrect {
		clamp(c.ObjRepair, "BR-022")
		if d.TurnAnalysis.CorrectionTarget == nil || !memoryID(w, *d.TurnAnalysis.CorrectionTarget) {
			d.TurnAnalysis.CorrectionTarget = nil
			d.Plan.PermittedContext = []c.ID{}
			avoid("claiming_recall")
			d.Plan.Question = c.QuestionPlan{Allowed: true, Purpose: "confirm", MaxCount: 1}
			reject("turn_analysis.correction_target", "BR-022")
		} else {
			target := *d.TurnAnalysis.CorrectionTarget
			out := []c.ID{}
			for _, id := range d.Plan.PermittedContext {
				if id != target {
					out = append(out, id)
				}
			}
			d.Plan.PermittedContext = out
		}
	} else if d.TurnAnalysis.CorrectionTarget != nil {
		d.TurnAnalysis.CorrectionTarget = nil
		reject("turn_analysis.correction_target", "BR-022")
	}
	if !knownThread(w, d.TurnAnalysis.TargetThreadID) {
		if d.TurnAnalysis.TargetThreadID != nil {
			reject("turn_analysis.target_thread_id", "BR-018")
		}
		d.TurnAnalysis.TargetThreadID = nil
		if d.TurnAnalysis.ThreadRelation == c.RelReturns || d.TurnAnalysis.ThreadRelation == c.RelShifts {
			d.TurnAnalysis.ThreadRelation = c.RelContinues
		}
	}
	if d.TurnAnalysis.ThreadRelation == c.RelReturns && d.TurnAnalysis.TargetThreadID == nil {
		d.TurnAnalysis.ThreadRelation = c.RelContinues
		reject("turn_analysis.thread_relation", "BR-018")
	}
	if d.TurnAnalysis.ThreadRelation == c.RelNew && (d.TurnAnalysis.NewThreadTopic == nil || strings.TrimSpace(*d.TurnAnalysis.NewThreadTopic) == "") {
		topic := words(w.Turn.Text, 8)
		if topic == "" {
			d.TurnAnalysis.ThreadRelation = c.RelContinues
		} else {
			d.TurnAnalysis.NewThreadTopic = &topic
		}
		reject("turn_analysis.new_thread_topic", "BR-018")
	}
	if d.Understanding.ThreadID != nil && !knownThread(w, d.Understanding.ThreadID) {
		d.Understanding.ThreadID = nil
		d.Understanding.Slots = []c.SlotUpdate{}
		d.Understanding.Hypotheses = []c.HypothesisSignal{}
		d.Understanding.ThreadPhaseProposal = nil
		reject("understanding_update.thread_id", "BR-018")
	}
	if in.Hop != 0 || w.RetrievalMeta.Hops != 0 {
		if d.Needs.Context != nil {
			d.Needs.Context = nil
			reject("needs.needs_context", "BR-019")
		}
	}
	// Honor raises only. Policy runs against the raised level BEFORE a hop or
	// generator is allowed; a high raise is routed by the orchestrator.
	risk := c.MaxRisk(w.Risk.Current, in.Safety.Level)
	if d.RiskRead != nil {
		if !d.RiskRead.Level.Valid() || d.RiskRead.Level.Rank() <= risk.Rank() {
			d.RiskRead = nil
			reject("risk_read", "BR-017")
		} else {
			risk = d.RiskRead.Level
		}
	}
	restricted := risk.Rank() >= c.RiskElevated.Rank() || w.Risk.FollowUpActive
	if restricted {
		d.Needs.Context = nil
		d.Plan.VoiceHypothesisID = nil
		if d.Plan.Protocol != nil {
			d.Plan.Protocol = nil
			clamp(c.ObjReflect, "SF-004")
		}
		avoid("intervention")
	}
	switch d.Decision.Primary {
	case c.ObjOfferIntervention:
		p := d.Plan.Protocol
		valid := !restricted && w.Eligibility.ProtocolsPossible && w.Protocol == nil && f.offer && p != nil && p.Action == c.PAOffer && p.ProtocolID != nil
		found := false
		if valid {
			for _, v := range w.Shortlist {
				if v.ProtocolID == *p.ProtocolID {
					_, found = a.protocols[v.ProtocolID]
				}
			}
		}
		if !valid || !found {
			if d.TurnAnalysis.ActionCue {
				clamp(c.ObjProvideGuidance, "BR-008")
			} else {
				clamp(c.ObjExplore, "BR-008")
			}
			d.Plan.Protocol = nil
		}
	case c.ObjContinueIntervention:
		active := w.Protocol != nil && (w.Protocol.Status == c.PIActive || w.Protocol.Status == c.PIOffered || w.Protocol.Status == c.PIPaused)
		if !active || restricted {
			clamp(c.ObjExplore, "BR-009")
			d.Plan.Protocol = nil
			break
		}
		p := d.Plan.Protocol
		if p == nil {
			out := c.OutPartial
			p = &c.ProtocolDirective{Action: c.PARepeat, Outcome: &out}
			d.Plan.Protocol = p
			reject("response_plan.protocol", "BR-009")
		}
		if !contains([]c.ProtocolAction{c.PAAdvance, c.PARepeat, c.PAAdapt, c.PAPause, c.PAStop, c.PAComplete}, p.Action) || !contains(w.Protocol.AllowedActions, p.Action) {
			p.Action = c.PARepeat
			reject("response_plan.protocol.action", "BR-009")
		}
		if p.Outcome == nil || !p.Outcome.Valid() {
			v := c.OutPartial
			p.Outcome = &v
			reject("response_plan.protocol.step_outcome", "BR-009")
		}
	case c.ObjCloseOrPause:
		if w.Protocol == nil || (w.Protocol.Status != c.PIActive && w.Protocol.Status != c.PIOffered && w.Protocol.Status != c.PIPaused) {
			clamp(c.ObjExplore, "BR-010")
			d.Plan.Protocol = nil
		}
	}
	if p := d.Plan.Protocol; p != nil {
		if p.Action == c.PAStart {
			d.Plan.Protocol = nil
			reject("response_plan.protocol.action", "BR-024")
		} else if d.Decision.Primary != c.ObjOfferIntervention && d.Decision.Primary != c.ObjContinueIntervention && d.Decision.Primary != c.ObjCloseOrPause {
			d.Plan.Protocol = nil
			reject("response_plan.protocol", "BR-008")
		} else if w.Protocol != nil {
			graph, ok := a.protocols[w.Protocol.ProtocolID]
			if !ok || graph.Version != w.Protocol.Version {
				d.Plan.Protocol = nil
				clamp(c.ObjExplore, "BR-009")
			} else {
				keys := []string{}
				for _, step := range graph.Steps {
					if step.ID == w.Protocol.StepID {
						keys = step.ExpectedInputs
					}
				}
				for key := range p.InputsCaptured {
					if !contains(keys, key) {
						delete(p.InputsCaptured, key)
						reject("response_plan.protocol.inputs_captured", "PR-004")
					}
				}
				if p.ProtocolID != nil && *p.ProtocolID != w.Protocol.ProtocolID {
					p.ProtocolID = nil
					reject("response_plan.protocol.protocol_id", "BR-009")
				}
				if d.TurnAnalysis.ThreadRelation != c.RelContinues {
					p.Action = c.PAPause
					note := "topic_shift"
					p.Note = &note
					clamp(c.ObjCloseOrPause, "BR-009")
				}
			}
		}
	}
	if d.Decision.Primary == c.ObjProvideInsight {
		id := d.Plan.VoiceHypothesisID
		if restricted || !w.Eligibility.MayVoiceHypothesis || !f.voice || id == nil || !supportedHypothesis(w, *id) || !f.hypotheses[*id] {
			clamp(c.ObjReflect, "BR-014")
			d.Plan.VoiceHypothesisID = nil
		}
	}
	if d.Decision.Primary != c.ObjProvideInsight && d.Plan.VoiceHypothesisID != nil {
		d.Plan.VoiceHypothesisID = nil
		reject("response_plan.voice_hypothesis_id", "BR-014")
	}
	if w.RetrievalMeta.Degraded || !w.Eligibility.MemoryAvailable {
		out := []c.ID{}
		for _, id := range d.Plan.PermittedContext {
			if !memoryID(w, id) {
				out = append(out, id)
			} else {
				reject("response_plan.permitted_context_ids", "BR-023")
			}
		}
		d.Plan.PermittedContext = out
		avoid("claiming_recall")
	}
	// Hypotheses cannot leak through generic permitted-context use. Only the
	// explicitly gated insight id can be passed to the later Generator.
	safe := []c.ID{}
	for _, id := range d.Plan.PermittedContext {
		if hyps[id] && (d.Plan.VoiceHypothesisID == nil || id != *d.Plan.VoiceHypothesisID) {
			reject("response_plan.permitted_context_ids", "BR-014")
			continue
		}
		safe = append(safe, id)
	}
	d.Plan.PermittedContext = safe
	if d.TurnAnalysis.Intent == c.IntentAskRecall {
		avoid("invented_recall")
		avoid("claiming_recall")
		exact := w.RetrievalMeta.ExactMatchFound != nil && *w.RetrievalMeta.ExactMatchFound && !w.RetrievalMeta.Degraded
		if !exact {
			clamp(c.ObjClarify, "BR-016")
			d.Plan.PermittedContext = []c.ID{}
			d.Plan.Question = c.QuestionPlan{Allowed: true, Purpose: "confirm", MaxCount: 1}
		} else {
			safe := []c.ID{}
			for _, id := range d.Plan.PermittedContext {
				for _, m := range w.Retrieved {
					if id == m.ID && m.Verbatim {
						safe = append(safe, id)
					}
				}
			}
			d.Plan.PermittedContext = safe
		}
	}
	allowed := w.Eligibility.QuestionAllowed && w.Session.ConsecutiveQuestions < a.gate.Questions.MaxConsecutiveQuestionTurns && a.gate.Questions.MaxPerReply == 1
	if w.FocusThread != nil && d.Plan.Question.Slot != "" {
		for _, slot := range w.FocusThread.Slots {
			if slot.Slot == d.Plan.Question.Slot && slot.Declined {
				allowed = false
			}
		}
	}
	if !allowed && d.Plan.Question.Allowed {
		d.Plan.Question.Allowed = false
		reject("response_plan.question", "BR-011")
	}
	if !d.Plan.Question.Allowed {
		d.Plan.Question.MaxCount = 0
		d.Plan.Question.Purpose = "none"
		d.Plan.Question.Slot = ""
	} else {
		d.Plan.Question.MaxCount = 1
		if d.Plan.Question.Purpose == "" || d.Plan.Question.Purpose == "none" {
			d.Plan.Question.Purpose = "invite_detail"
		}
	}
	if d.Decision.Primary == c.ObjClarify {
		need := false
		for _, v := range w.InfoNeeds {
			if v.Slot == d.Plan.Question.Slot && v.Slot != "" {
				need = true
			}
		}
		// BR-016's not-found clarification is an explicit rule, not an invented gap.
		recallMissing := d.TurnAnalysis.Intent == c.IntentAskRecall && (w.RetrievalMeta.ExactMatchFound == nil || !*w.RetrievalMeta.ExactMatchFound)
		if !d.Plan.Question.Allowed || len(d.TurnAnalysis.Ambiguities) == 0 && !need && !recallMissing {
			clamp(c.ObjExplore, "BR-013")
		}
	}
	if d.Decision.Primary == c.ObjReflect || d.Decision.Primary == c.ObjValidateSupport {
		if !contains(d.Plan.MustAvoid, "advice") && !contains(d.Plan.MustAvoid, "directive_advice") {
			avoid("advice")
			reject("response_plan.must_avoid", "BR-015")
		}
	}
	// Free text is bounded by words in addition to JSON Schema character caps.
	truncate := func(p *string, max int) {
		v := words(*p, max)
		if v != *p {
			*p = v
			reject("text", "BR-020")
		}
	}
	if d.TurnAnalysis.NewThreadTopic != nil {
		truncate(d.TurnAnalysis.NewThreadTopic, 8)
	}
	for i := range d.TurnAnalysis.Ambiguities {
		truncate(&d.TurnAnalysis.Ambiguities[i].Note, 12)
	}
	for i := range d.Understanding.Slots {
		truncate(&d.Understanding.Slots[i].Value, 30)
	}
	if d.Understanding.SessionSummaryDelta != nil {
		truncate(d.Understanding.SessionSummaryDelta, 40)
	}
	if d.Needs.Context != nil {
		truncate(&d.Needs.Context.Query, 12)
		if strings.TrimSpace(d.Needs.Context.Query) == "" {
			d.Needs.Context = nil
			reject("needs.needs_context", "BR-020")
		}
	}
	if d.Plan.Protocol != nil {
		for key, val := range d.Plan.Protocol.InputsCaptured {
			truncate(&val, 30)
			d.Plan.Protocol.InputsCaptured[key] = val
		}
	}
	return d, r
}
func words(s string, n int) string {
	v := strings.Fields(s)
	if len(v) <= n {
		return s
	}
	return strings.Join(v[:n], " ")
}
func contextMessages(w c.WorkingContext) map[c.ID]c.MsgView {
	m := map[c.ID]c.MsgView{w.Turn.MessageID: {ID: w.Turn.MessageID, Role: c.RoleUser, Text: w.Turn.Text}}
	for _, v := range w.Recent {
		if v.ID != w.Turn.MessageID {
			m[v.ID] = v
		}
	}
	for _, v := range w.Retrieved {
		if v.Verbatim {
			if _, ok := m[v.ID]; !ok {
				m[v.ID] = c.MsgView{ID: v.ID, Role: c.RoleUser, Text: v.Text}
			}
		}
	}
	return m
}
func contextIDs(w c.WorkingContext) map[c.ID]bool {
	m := map[c.ID]bool{}
	for id := range contextMessages(w) {
		m[id] = true
	}
	for _, v := range w.Retrieved {
		m[v.ID] = true
	}
	if w.FocusThread != nil {
		m[w.FocusThread.ID] = true
		for _, v := range w.FocusThread.Hypotheses {
			m[v.ID] = true
		}
	}
	return m
}
func hypotheses(w c.WorkingContext) map[c.ID]bool {
	m := map[c.ID]bool{}
	if w.FocusThread != nil {
		for _, v := range w.FocusThread.Hypotheses {
			m[v.ID] = true
		}
	}
	for _, v := range w.Retrieved {
		if v.Kind == c.MemHypothesis {
			m[v.ID] = true
		}
	}
	return m
}
func supportedHypothesis(w c.WorkingContext, id c.ID) bool {
	if w.FocusThread != nil {
		for _, h := range w.FocusThread.Hypotheses {
			if h.ID == id {
				return h.Status == c.MemSupported
			}
		}
	}
	for _, m := range w.Retrieved {
		if m.ID == id {
			return m.Kind == c.MemHypothesis && m.Status == c.MemSupported
		}
	}
	return false
}
func memoryID(w c.WorkingContext, id c.ID) bool {
	for _, v := range w.Retrieved {
		if v.ID == id {
			return true
		}
	}
	return hypotheses(w)[id]
}
func knownThread(w c.WorkingContext, id *c.ID) bool {
	if id == nil {
		return false
	}
	if w.FocusThread != nil && w.FocusThread.ID == *id {
		return true
	}
	for _, v := range w.OtherThreads {
		if v.ID == *id && v.Phase != c.PhaseResolved {
			return true
		}
	}
	return false
}

// Conservative lexical branch of §10.6. No semantic model/clinical vocabulary
// is invented: exact normalized content tokens must meet the specified 50%.
// Inflection/stopword variants without a reviewed tokenizer downgrade to inferred.
func supports(msg, value string) bool {
	tokens := func(s string) []string {
		return strings.FieldsFunc(strings.ToLower(norm.NFC.String(s)), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
	}
	source := map[string]bool{}
	for _, v := range tokens(msg) {
		source[v] = true
	}
	v := tokens(value)
	if len(v) == 0 {
		return false
	}
	hits := 0
	for _, t := range v {
		if source[t] {
			hits++
		}
	}
	return hits*2 >= len(v)
}
