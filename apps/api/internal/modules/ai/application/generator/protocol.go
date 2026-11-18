package generator

import (
	"reflect"
	"strings"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// This only renders a resolved specification. It performs no eligibility,
// consent, selection, transitions, or persistence (all belong to WP-13).
func (a *Assets) protocol(in ports.GenInput, locale string) (string, error) {
	if in.Spec == nil && in.Plan.Protocol == nil {
		return tag("protocol_step", ""), nil
	}
	if in.Spec == nil || in.Plan.Protocol == nil {
		return "", invalid()
	}
	spec := in.Spec
	directive := in.Plan.Protocol
	p, ok := a.protocols[spec.ProtocolID]
	if !ok || !directive.Action.Valid() || directive.ProtocolID != nil && *directive.ProtocolID != spec.ProtocolID {
		return "", invalid()
	}
	// Protocols currently carry English instructions only. Do not silently
	// translate reviewed clinical instructions into an unreviewed locale.
	if locale != "en" {
		return "", &c.Error{Code: c.ErrBundle, Step: "generator_protocol_locale"}
	}
	if directive.Action == c.PAOffer {
		if in.Objective != c.ObjOfferIntervention || spec.Offer == nil || directive.ProtocolID == nil || spec.Offer.ProtocolID != p.ID || spec.Offer.Name != p.Name || spec.Offer.EstMinutes != p.EstMinutes || !spec.Offer.OptOutLine || spec.Instruction != "" || spec.StepID != "" || len(spec.ExpectedInputs) > 0 || len(spec.AllowedAdaptations) > 0 || len(directive.InputsCaptured) > 0 {
			return "", invalid()
		}
		if len(spec.Offer.Why) == 0 {
			return "", invalid()
		}
		for _, why := range spec.Offer.Why {
			found := false
			for _, id := range p.Offer.WhyTemplates {
				if a.offers.WhyTemplates[id].Text == why && why != "" {
					found = true
				}
			}
			if !found {
				return "", invalid()
			}
		}
	} else {
		if !c.ValidID(spec.InstanceID) || spec.Offer != nil {
			return "", invalid()
		}
		found := false
		for _, step := range p.Steps {
			if step.ID != spec.StepID || !reflect.DeepEqual(step.ExpectedInputs, spec.ExpectedInputs) {
				continue
			}
			allowed := []string{}
			instruction := spec.Instruction == step.Instruction
			for _, adapt := range step.Adaptations {
				allowed = append(allowed, adapt.ID)
				if directive.Action == c.PAAdapt && spec.Instruction == adapt.Instruction {
					instruction = true
				}
			}
			if !instruction || strings.TrimSpace(spec.Instruction) == "" {
				continue
			}
			for _, id := range spec.AllowedAdaptations {
				if !includes(allowed, id) {
					return "", invalid()
				}
			}
			found = true
		}
		if !found {
			return "", invalid()
		}
	}
	for _, id := range p.Offer.MustSay {
		if !includes(spec.MustSay, id) {
			return "", invalid()
		}
	}
	if len(spec.MustSay) == 0 {
		return "", invalid()
	}
	text := "<protocol_step protocol_id=\"" + escape(spec.ProtocolID) + "\" step_id=\"" + escape(spec.StepID) + "\""
	if c.ValidID(spec.InstanceID) {
		text += " instance_id=\"" + spec.InstanceID.String() + "\""
	}
	text += ">"
	text += data("instruction", spec.Instruction) + data("expected_inputs", spec.ExpectedInputs) + data("allowed_adaptations", spec.AllowedAdaptations)
	for key := range directive.InputsCaptured {
		if !includes(spec.ExpectedInputs, key) {
			return "", invalid()
		}
	}
	text += data("protocol_inputs", directive.InputsCaptured)
	for _, id := range spec.MustSay {
		entry, ok := a.offers.MustSay[id]
		if !ok || entry.Text == "" {
			return "", invalid()
		}
		text += "<must_say id=\"" + escape(id) + "\">" + escape(entry.Text) + "</must_say>"
	}
	if spec.Offer != nil {
		text += data("offer", spec.Offer)
	}
	return text + "</protocol_step>\n", nil
}
