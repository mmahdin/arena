package brain

import (
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"testing"
)

func TestDecisionGateRules(t *testing.T) {
	a := testAssets(t)
	tests := []struct {
		name, rule string
		change     func(*c.BrainDecision, *ports.BrainInput, *policyFacts)
		check      func(c.BrainDecision) bool
	}{
		{"invalid objective", "BR-002", func(d *c.BrainDecision, _ *ports.BrainInput, _ *policyFacts) { d.Decision.Primary = "invented" }, func(d c.BrainDecision) bool { return d.Decision.Primary == c.ObjReflect }},
		{"router safety only", "BR-012", func(d *c.BrainDecision, _ *ports.BrainInput, _ *policyFacts) { d.Decision.Primary = c.ObjSafetySupport }, func(d c.BrainDecision) bool { return d.Decision.Primary != c.ObjSafetySupport }},
		{"skill conflict", "BR-003", func(d *c.BrainDecision, _ *ports.BrainInput, _ *policyFacts) {
			d.Plan.Skills.Secondary = []string{"test_conflict"}
		}, func(d c.BrainDecision) bool { return len(d.Plan.Skills.Secondary) == 0 }},
		{"unknown primary skill", "BR-003", func(d *c.BrainDecision, _ *ports.BrainInput, _ *policyFacts) { d.Plan.Skills.Primary = "not_real" }, func(d c.BrainDecision) bool { return d.Plan.Skills.Primary == "reflect_emotion" }},
		{"hallucinated context", "BR-004", func(d *c.BrainDecision, _ *ports.BrainInput, _ *policyFacts) {
			d.Plan.PermittedContext = []c.ID{c.NewID()}
		}, func(d c.BrainDecision) bool { return len(d.Plan.PermittedContext) == 0 }},
		{"missing evidence", "BR-005", func(d *c.BrainDecision, _ *ports.BrainInput, _ *policyFacts) {
			d.Understanding.Slots = []c.SlotUpdate{{Slot: "situation", Value: "event", Status: c.EpReported, Evidence: []c.ID{c.NewID()}}}
		}, func(d c.BrainDecision) bool { return len(d.Understanding.Slots) == 0 }},
		{"unsupported reported", "BR-006", func(d *c.BrainDecision, in *ports.BrainInput, f *policyFacts) {
			id := in.WCtx.Turn.MessageID
			f.owned = map[c.ID]bool{id: true}
			d.Understanding.Slots = []c.SlotUpdate{{Slot: "situation", Value: "unrelated invented explanation", Status: c.EpReported, Evidence: []c.ID{id}}}
		}, func(d c.BrainDecision) bool {
			return len(d.Understanding.Slots) == 1 && d.Understanding.Slots[0].Status == c.EpInferred
		}},
		{"unknown slot", "BR-007", func(d *c.BrainDecision, _ *ports.BrainInput, _ *policyFacts) {
			d.Understanding.Slots = []c.SlotUpdate{{Slot: "diagnosis", Value: "invented", Status: c.EpReported}}
		}, func(d c.BrainDecision) bool { return len(d.Understanding.Slots) == 0 }},
		{"offer without shortlist", "BR-008", func(d *c.BrainDecision, _ *ports.BrainInput, _ *policyFacts) {
			d.Decision.Primary = c.ObjOfferIntervention
		}, func(d c.BrainDecision) bool { return d.Decision.Primary == c.ObjExplore && d.Plan.Protocol == nil }},
		{"no active protocol", "BR-009", func(d *c.BrainDecision, _ *ports.BrainInput, _ *policyFacts) {
			d.Decision.Primary = c.ObjContinueIntervention
		}, func(d c.BrainDecision) bool { return d.Decision.Primary == c.ObjExplore }},
		{"close without active", "BR-010", func(d *c.BrainDecision, _ *ports.BrainInput, _ *policyFacts) { d.Decision.Primary = c.ObjCloseOrPause }, func(d c.BrainDecision) bool { return d.Decision.Primary == c.ObjExplore }},
		{"question budget", "BR-011", func(_ *c.BrainDecision, in *ports.BrainInput, _ *policyFacts) {
			in.WCtx.Session.ConsecutiveQuestions = 2
		}, func(d c.BrainDecision) bool { return !d.Plan.Question.Allowed && d.Plan.Question.MaxCount == 0 }},
		{"clarify without gap", "BR-013", func(d *c.BrainDecision, _ *ports.BrainInput, _ *policyFacts) { d.Decision.Primary = c.ObjClarify }, func(d c.BrainDecision) bool { return d.Decision.Primary == c.ObjExplore }},
		{"insight without permission", "BR-014", func(d *c.BrainDecision, _ *ports.BrainInput, _ *policyFacts) {
			d.Decision.Primary = c.ObjProvideInsight
			id := c.NewID()
			d.Plan.VoiceHypothesisID = &id
		}, func(d c.BrainDecision) bool {
			return d.Decision.Primary == c.ObjReflect && d.Plan.VoiceHypothesisID == nil
		}},
		{"reflection avoids advice", "BR-015", func(d *c.BrainDecision, _ *ports.BrainInput, _ *policyFacts) { d.Plan.MustAvoid = []string{} }, func(d c.BrainDecision) bool { return contains(d.Plan.MustAvoid, "advice") }},
		{"exact miss", "BR-016", func(d *c.BrainDecision, in *ports.BrainInput, _ *policyFacts) {
			d.Decision.Primary = c.ObjAnswerDirect
			d.TurnAnalysis.Intent = c.IntentAskRecall
			b := false
			in.WCtx.RetrievalMeta.ExactMatchFound = &b
		}, func(d c.BrainDecision) bool {
			return d.Decision.Primary == c.ObjClarify && contains(d.Plan.MustAvoid, "invented_recall")
		}},
		{"never lower risk", "BR-017", func(d *c.BrainDecision, _ *ports.BrainInput, _ *policyFacts) {
			d.RiskRead = &c.BrainRiskRead{Level: c.RiskNone}
		}, func(d c.BrainDecision) bool { return d.RiskRead == nil }},
		{"unknown return thread", "BR-018", func(d *c.BrainDecision, _ *ports.BrainInput, _ *policyFacts) {
			d.TurnAnalysis.ThreadRelation = c.RelReturns
			id := c.NewID()
			d.TurnAnalysis.TargetThreadID = &id
		}, func(d c.BrainDecision) bool { return d.TurnAnalysis.ThreadRelation == c.RelContinues }},
		{"second hop cap", "BR-019", func(d *c.BrainDecision, in *ports.BrainInput, _ *policyFacts) {
			in.Hop = 1
			d.Needs.Context = &c.NeedsContext{Kind: c.ContextEntity, Query: "boss", Why: c.ContextMissingBackground}
		}, func(d c.BrainDecision) bool { return d.Needs.Context == nil }},
		{"word cap", "BR-020", func(d *c.BrainDecision, _ *ports.BrainInput, _ *policyFacts) {
			s := "one two three four five six seven eight nine ten"
			d.TurnAnalysis.NewThreadTopic = &s
		}, func(d c.BrainDecision) bool {
			return *d.TurnAnalysis.NewThreadTopic == "one two three four five six seven eight"
		}},
		{"cooldown unknown denies", "BR-021", func(d *c.BrainDecision, in *ports.BrainInput, f *policyFacts) {
			tag := "synthetic_pattern"
			f.owned = map[c.ID]bool{in.WCtx.Turn.MessageID: true}
			d.Understanding.Hypotheses = []c.HypothesisSignal{{Op: c.HypPropose, Tag: &tag, Evidence: []c.ID{in.WCtx.Turn.MessageID}}}
		}, func(d c.BrainDecision) bool { return len(d.Understanding.Hypotheses) == 0 }},
		{"correction repair", "BR-022", func(d *c.BrainDecision, _ *ports.BrainInput, _ *policyFacts) {
			d.TurnAnalysis.Intent = c.IntentDisagreeCorrect
		}, func(d c.BrainDecision) bool {
			return d.Decision.Primary == c.ObjRepair && d.Plan.Question.Purpose == "confirm"
		}},
		{"degraded memory", "BR-023", func(d *c.BrainDecision, in *ports.BrainInput, _ *policyFacts) {
			id := c.NewID()
			in.WCtx.Retrieved = []c.MemoryView{{ID: id}}
			in.WCtx.RetrievalMeta.Degraded = true
			d.Plan.PermittedContext = []c.ID{id}
		}, func(d c.BrainDecision) bool {
			return len(d.Plan.PermittedContext) == 0 && contains(d.Plan.MustAvoid, "claiming_recall")
		}},
		{"start forbidden", "BR-024", func(d *c.BrainDecision, _ *ports.BrainInput, _ *policyFacts) {
			d.Plan.Protocol = &c.ProtocolDirective{Action: c.PAStart}
		}, func(d c.BrainDecision) bool { return d.Plan.Protocol == nil }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := input()
			d := decision(a, in)
			f := policyFacts{}
			tt.change(&d, &in, &f)
			before := string(encoded(t, d))
			got, r := a.validate(d, in, f)
			if !hasRule(r, tt.rule) || !tt.check(got) {
				t.Fatalf("rule %s not enforced: %+v %+v", tt.rule, got, r)
			}
			if string(encoded(t, d)) != before {
				t.Fatal("mutated input")
			}
		})
	}
}

func TestEvidenceAndProtocolPositivePaths(t *testing.T) {
	a := testAssets(t)
	in := input()
	d := decision(a, in)
	id := in.WCtx.Turn.MessageID
	d.Understanding.Slots = []c.SlotUpdate{{Slot: "situation", Value: "Synthetic event at work", Status: c.EpReported, Evidence: []c.ID{id}}}
	out, r := a.validate(d, in, policyFacts{owned: map[c.ID]bool{id: true}})
	if len(r) != 0 || out.Understanding.Slots[0].Status != c.EpReported {
		t.Fatal(out, r)
	}
	// A correctly shortlisted offer remains an offer; consent cannot start it.
	pid := "synthetic_protocol"
	d.Decision.Primary = c.ObjOfferIntervention
	d.Plan.Protocol = &c.ProtocolDirective{Action: c.PAOffer, ProtocolID: &pid}
	in.WCtx.Eligibility.ProtocolsPossible = true
	in.WCtx.Shortlist = []c.ProtocolBrief{{ProtocolID: pid}}
	out, r = a.validate(d, in, policyFacts{owned: map[c.ID]bool{id: true}, offer: true})
	if out.Decision.Primary != c.ObjOfferIntervention || out.Plan.Protocol.Action != c.PAOffer {
		t.Fatal(out, r)
	}
	in.WCtx.Risk.FollowUpActive = true
	out, _ = a.validate(d, in, policyFacts{offer: true})
	if out.Plan.Protocol != nil || out.Decision.Primary == c.ObjOfferIntervention {
		t.Fatal("follow-up allowed intervention")
	}
}
func TestRiskRaiseReroutesAndReferenceAmbiguity(t *testing.T) {
	a := testAssets(t)
	in := input()
	d := decision(a, in)
	d.RiskRead = &c.BrainRiskRead{Level: c.RiskHigh, Categories: []c.RiskCategory{c.RiskOther}, ReasonTag: "synthetic"}
	_, _, override := a.Gate(ports.GateInput{Decision: d, WCtx: in.WCtx, Safety: in.Safety})
	if override == nil || override.Route != c.RouteCrisis {
		t.Fatal(override)
	}
	id := c.NewID()
	in.WCtx.Retrieved = []c.MemoryView{{ID: id, Kind: c.MemEpisode}}
	in.WCtx.RetrievalMeta.AmbiguousReference = true
	d.TurnAnalysis.PastRefs = []c.PastRef{{Phrase: "that time", ResolvedTo: []c.ID{id}}}
	out, _ := a.validate(d, in, policyFacts{})
	if !out.TurnAnalysis.PastRefs[0].Ambiguous || len(out.TurnAnalysis.PastRefs[0].ResolvedTo) != 0 {
		t.Fatal(out)
	}
}
