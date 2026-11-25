package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/protocol"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/unified"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"gopkg.in/yaml.v3"
)

func copyIntent(to *c.UnifiedTurn, from c.UnifiedTurn, kind string) {
	switch kind {
	case "session_intent":
		to.Session = from.Session
	case "thread_intent":
		to.Thread = from.Thread
	case "protocol_intent":
		to.Protocol = from.Protocol
	case "open_loops":
		to.OpenLoops = from.OpenLoops
	case "preferences":
		to.Preferences = from.Preferences
	case "risk_read":
		to.RiskRead = from.RiskRead
	}
}
func proposalList(u c.UnifiedTurn) []c.StateProposal {
	if u.Proposals != nil {
		return u.Proposals
	}
	raw, _ := json.Marshal(u)
	var envelope c.UnifiedEnvelope
	_ = json.Unmarshal(raw, &envelope)
	var list []c.StateProposal
	for _, raw := range envelope.StateProposals {
		var p c.StateProposal
		if json.Unmarshal(raw, &p) == nil {
			list = append(list, p)
		}
	}
	return list
}
func (o *Service) resolveProposals(ctx context.Context, a c.Admission, u c.UnifiedTurn, w c.WorkingContext, s ports.StateSnapshot, resources []c.KnowledgeResource) (result c.UnifiedTurn, stateDelta c.StateDelta, protocolUI *c.ProtocolUI, acceptedResult []string, rejections []c.DeltaRejection, err error) {
	selectedContentIDs := make([]string, 0, len(resources))
	for _, resource := range resources {
		selectedContentIDs = append(selectedContentIDs, resource.ContentID)
	}
	ctx, span := trace.Start(ctx, "orchestrator.resolveProposals", func() any {
		return map[string]any{"admission": a, "turn": u, "context": w, "state": s, "selected_content_ids": selectedContentIDs}
	})
	defer func() {
		span.Complete(func() any {
			return map[string]any{"turn": result, "state": stateDelta, "ui": protocolUI, "accepted_ids": acceptedResult, "rejected": rejections}
		}, err, recover())
	}()
	base := u
	base.Session = nil
	base.Thread = nil
	base.Protocol = nil
	base.OpenLoops = nil
	base.Preferences = nil
	base.RiskRead = nil
	accepted := base
	delta := unifiedDelta(a, base, w, s, o.integration.now())
	var ui *c.ProtocolUI
	rejected := append([]c.DeltaRejection{}, u.Rejected...)
	acceptedIDs := []string{}
	done := map[string]bool{}
	pending := proposalList(u)
	for len(pending) > 0 {
		progress := false
		next := []c.StateProposal{}
		for _, p := range pending {
			ready := true
			for _, id := range p.DependsOn {
				if !done[id] {
					ready = false
				}
			}
			if !ready {
				next = append(next, p)
				continue
			}
			progress = true
			candidate := accepted
			copyIntent(&candidate, u, p.Kind)
			deny := func(rule string) {
				rejected = append(rejected, c.DeltaRejection{Path: "state_proposals." + p.Kind, RuleCode: rule, Detail: "independent proposal rejected"})
			}
			if planner, ok := o.deps.UnifiedMemory.(interface{ BackgroundOnly() bool }); ok && planner.BackgroundOnly() && candidate.Thread != nil && len(candidate.Thread.Hypotheses) > 0 {
				deny("OUT-REFLECTION-OWNED")
				continue
			}
			if unified.ValidateReferences(candidate, w) != nil {
				deny("OUT-REF")
				continue
			}
			proposed := unifiedDelta(a, candidate, w, s, o.integration.now())
			if err := o.prepareRisk(ctx, a, &proposed); err != nil {
				return accepted, delta, ui, acceptedIDs, rejected, err
			}
			var candidateUI *c.ProtocolUI
			if candidate.Protocol != nil {
				engine := o.deps.Protocol
				if engine == nil {
					deny("OUT-PROTOCOL")
					continue
				}
				spec, patch, e := engine.ResolveTurn(ctx, protocol.ResolveInput{UserID: a.UserID, TurnID: a.TurnID, UserMessageID: a.UserMessageID, Bundle: o.bundleReference(ctx), Plan: c.ResponsePlan{Protocol: &candidate.Protocol.Directive}, Objective: candidate.Conversation.Objective, Active: s.Protocol, WCtx: w, SelectedContentIDs: selectedContentIDs, UserAffirmed: candidate.Protocol.UserAffirmed, Risk: effectiveRisk(u, w)})
				if e != nil {
					if ctx.Err() != nil {
						return accepted, delta, ui, acceptedIDs, rejected, ctx.Err()
					}
					if !errors.Is(e, protocol.ErrInvalid) && !errors.Is(e, protocol.ErrConsent) && !errors.Is(e, protocol.ErrUnreviewed) {
						return accepted, delta, ui, acceptedIDs, rejected, e
					}
					deny("OUT-PROTOCOL")
					continue
				}
				status := c.PIOffered
				if s.Protocol != nil {
					status = s.Protocol.Status
				}
				if patch.NewStatus != nil {
					status = *patch.NewStatus
				}
				if status != candidate.Protocol.ExpectedStatus || spec.StepID != candidate.Protocol.ExpectedStep {
					deny("OUT-PROTOCOL-MISMATCH")
					continue
				}
				proposed.Protocol = &patch
				candidateUI = spec.UI
			}
			checked, failures, e := o.integration.committer.ValidateDelta(ctx, proposed)
			if e != nil {
				var ce *c.Error
				if errors.As(e, &ce) && ce.Code == c.ErrValidation {
					deny("OUT-STATE")
					continue
				}
				return accepted, delta, ui, acceptedIDs, rejected, e
			}
			if len(failures) > 0 {
				deny("OUT-STATE")
				rejected = append(rejected, failures...)
				continue
			}
			accepted = candidate
			delta = checked
			ui = candidateUI
			done[p.ID] = true
			acceptedIDs = append(acceptedIDs, p.ID)
		}
		if !progress {
			for i := range next {
				rejected = append(rejected, c.DeltaRejection{Path: fmt.Sprintf("state_proposals.dependencies.%d", i), RuleCode: "OUT-DEPENDENCY", Detail: "missing, rejected or cyclic dependency"})
			}
			break
		}
		pending = next
	}
	// Delivering an exercise must always match an actual accepted transition,
	// including a model that omitted an explicit reply dependency by mistake.
	if u.Protocol != nil && accepted.Protocol == nil {
		return accepted, delta, ui, acceptedIDs, rejected, coreError(c.ErrValidation, "protocol_response_mismatch", nil)
	}
	trace.Capture(ctx, "state_resolution", "state", "independent_proposals", map[string]any{"accepted_ids": acceptedIDs, "accepted": delta, "rejected": rejected})
	return accepted, delta, ui, acceptedIDs, rejected, nil
}

// The existing risk transition policy and duration are unchanged. Safety stays
// within Unified; selection and optional analyses have no risk authority.
func (o *Service) prepareRisk(ctx context.Context, a c.Admission, delta *c.StateDelta) error {
	if delta.User.Risk == nil {
		return nil
	}
	pinned, _ := (pinnedRegistry{}).Active(ctx, a.UserID)
	raw, e := pinned.Artifact(ctx, "policy/safety_policy.yaml")
	if e != nil {
		return e
	}
	var policy struct {
		Risk struct {
			FollowUpHours int `yaml:"follow_up_hours"`
		} `yaml:"risk_fsm"`
	}
	if yaml.Unmarshal(raw, &policy) != nil || policy.Risk.FollowUpHours <= 0 {
		return coreError(c.ErrBundle, "risk_policy", nil)
	}
	until := o.integration.now().UTC().Add(time.Duration(policy.Risk.FollowUpHours) * time.Hour)
	delta.Events = []c.DomainEvent{{Type: c.EvRiskLevelChanged, Payload: map[string]any{"from": delta.User.Risk.From, "to": delta.User.Risk.To, "turn_id": a.TurnID, "follow_up_until": until}}}
	return nil
}
