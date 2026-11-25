package protocol

import (
	"context"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// Engine owns ONLY deterministic decisions. Commit the returned patch inside
// WP-02's atomic TurnFinalizer with WP-08's StateCommitter validation; never
// mutate a protocol instance or emit user-visible guidance directly.
type Engine struct {
	assets *Assets
	reader contextstate.Reader
	now    func() time.Time
}

var _ ports.ProtocolEngine = (*Engine)(nil)

func New(assets *Assets, reader contextstate.Reader, now func() time.Time) (*Engine, error) {
	if assets == nil || reader == nil {
		return nil, ErrInvalid
	}
	if now == nil {
		now = time.Now
	}
	return &Engine{assets: assets, reader: reader, now: now}, nil
}
func (e *Engine) Reference() c.BundleRef { return e.assets.Reference() }

type ResolveInput = c.ProtocolResolutionInput

// ResolveTurn is the authenticated protocol boundary. Read snapshots and
// consent message role from WP-08; verify owner and same-turn IDs. This is
// deliberately read-only; the finalizer enforces versions and foreign keys.
func (e *Engine) ResolveTurn(ctx context.Context, in ResolveInput) (stepResult c.ProtocolStepSpec, patchResult c.ProtocolPatch, resolveErr error) {
	ctx, inspectionSpan := trace.Start(ctx, "application/protocol/engine.ResolveTurn", func() any { return map[string]any{"in": in} })
	defer func() {
		inspectionSpan.Complete(func() any { return map[string]any{"stepResult": stepResult, "patchResult": patchResult} }, resolveErr, recover())
	}()

	var zeroSpec c.ProtocolStepSpec
	var zeroPatch c.ProtocolPatch
	if err := ctx.Err(); err != nil {
		return zeroSpec, zeroPatch, err
	}
	if !c.ValidID(in.UserID) || !c.ValidID(in.TurnID) || !c.ValidID(in.UserMessageID) || in.TurnID != in.WCtx.Turn.TurnID || in.UserMessageID != in.WCtx.Turn.MessageID || !reflect.DeepEqual(in.Bundle, e.assets.Reference()) {
		return zeroSpec, zeroPatch, ErrInvalid
	}
	snapshot, err := e.reader.ForTurn(ctx, in.UserID, in.TurnID)
	if err != nil {
		return zeroSpec, zeroPatch, err
	}
	if snapshot.User.UserID != in.UserID || snapshot.Session.UserID != in.UserID || snapshot.Session.ClosedAt != nil || snapshot.Session.ID != in.WCtx.Session.SessionID || snapshot.Protocol == nil && in.Active != nil || snapshot.Protocol != nil && in.Active == nil {
		return zeroSpec, zeroPatch, ErrInvalid
	}
	if in.Active != nil && (snapshot.Protocol.ID != in.Active.ID || snapshot.Protocol.Version != in.Active.Version || snapshot.Protocol.UserID != in.UserID || snapshot.Protocol.Status != in.Active.Status || snapshot.Protocol.StepID != in.Active.StepID || snapshot.Protocol.ProtocolVersion != in.Active.ProtocolVersion) {
		return zeroSpec, zeroPatch, ErrInvalid
	}
	effective := c.MaxRisk(in.Risk, snapshot.User.Risk.Level)
	effective = c.MaxRisk(effective, in.WCtx.Risk.Current)
	directive := in.Plan.Protocol
	if directive == nil {
		return zeroSpec, zeroPatch, ErrInvalid
	}
	if !e.assets.manifest.ProtocolLanguage(c.ResponseLanguage(snapshot.User.Prefs.Language, in.WCtx.Turn.Lang)) ||
		!e.assets.manifest.ProtocolLanguage(c.ResponseLanguage(in.WCtx.Prefs.Language, in.WCtx.Turn.Lang)) {
		if in.Active == nil {
			return zeroSpec, zeroPatch, ErrUnreviewed
		}
		status := c.PIStopped
		return zeroSpec, c.ProtocolPatch{InstanceID: in.Active.ID, NewStatus: &status, Reason: "PR-STOP:language_resources_unavailable"}, nil
	}
	if directive.Action == c.PAOffer {
		return e.offer(ctx, in, snapshot, effective)
	}
	if in.Active == nil {
		return zeroSpec, zeroPatch, ErrInvalid
	}
	p, ok := e.assets.Protocol(in.Active.ProtocolID)
	if !ok || p.Version != in.Active.ProtocolVersion {
		return zeroSpec, zeroPatch, ErrUnreviewed
	}
	if directive.ProtocolID != nil && *directive.ProtocolID != p.ID {
		return zeroSpec, zeroPatch, ErrInvalid
	}
	var step *bundle.ProtocolStep
	for i := range p.Steps {
		if p.Steps[i].ID == in.Active.StepID {
			step = &p.Steps[i]
			break
		}
	}
	if step == nil {
		return zeroSpec, zeroPatch, ErrInvalid
	}
	patch := c.ProtocolPatch{InstanceID: in.Active.ID, Inputs: map[string]string{}}
	spec := e.stepSpec(p, *step, in.Active.ID, "")
	if in.Active.Status == c.PIOffered {
		if effective.Rank() >= c.RiskElevated.Rank() || in.WCtx.Risk.FollowUpActive || directive.Outcome != nil && *directive.Outcome == c.OutDistress {
			v := c.PIStopped
			patch.NewStatus = &v
			patch.Reason = "PR-SAFETY:offer_interrupted"
			return zeroSpec, patch, nil
		}
		// PR-003: the Brain can only say advance/success; the engine derives start.
		if directive.Action == c.PADecline || directive.Action == c.PAStop || directive.Outcome != nil && *directive.Outcome == c.OutDecline {
			v := c.PIDeclined
			patch.NewStatus = &v
			patch.Reason = "PR-003:user_declined"
			return zeroSpec, patch, nil
		}
		if directive.Action != c.PAAdvance || directive.Outcome == nil || *directive.Outcome != c.OutSuccess || !in.UserAffirmed || effective.Rank() > c.RiskLow.Rank() || in.WCtx.Risk.FollowUpActive {
			return zeroSpec, zeroPatch, ErrConsent
		}
		evidence, err := e.reader.UserEvidence(ctx, in.UserID, []c.ID{in.UserMessageID})
		if err != nil {
			return zeroSpec, zeroPatch, err
		}
		if !evidence[in.UserMessageID] {
			return zeroSpec, zeroPatch, ErrConsent
		}
		now := e.now().UTC()
		v := c.PIActive
		patch.NewStatus = &v
		patch.ConsentAt = &now
		patch.ConsentMessageID = &in.UserMessageID
		patch.Reason = "PR-003:explicit_user_consent"
		return spec, patch, nil
	}
	if in.Active.Status != c.PIActive && in.Active.Status != c.PIPaused {
		return zeroSpec, zeroPatch, ErrInvalid
	}
	if in.Active.Status == c.PIPaused {
		if directive.Action == c.PAStop || directive.Action == c.PADecline {
			v := c.PIStopped
			patch.NewStatus = &v
			patch.Reason = "PR-STOP:paused_user_stop"
			return zeroSpec, patch, nil
		}
		// The Brain is forbidden to issue PAStart. A paused instance only
		// resumes through a fresh user-affirmative advance/success; the
		// engine derives the resume transition after verifying its role.
		if directive.Action != c.PAAdvance || directive.Outcome == nil || *directive.Outcome != c.OutSuccess || !in.UserAffirmed || effective.Rank() > c.RiskLow.Rank() || in.WCtx.Risk.FollowUpActive {
			return zeroSpec, zeroPatch, ErrConsent
		}
		evidence, err := e.reader.UserEvidence(ctx, in.UserID, []c.ID{in.UserMessageID})
		if err != nil {
			return zeroSpec, zeroPatch, err
		}
		if !evidence[in.UserMessageID] {
			return zeroSpec, zeroPatch, ErrConsent
		}
		v := c.PIActive
		patch.NewStatus = &v
		patch.Reason = "PR-RESUME:explicit_user_request"
		return spec, patch, nil
	}
	if effective.Rank() >= c.RiskElevated.Rank() || in.WCtx.Risk.FollowUpActive {
		// Never continue or re-present a step after a safety escalation.
		action := stopRule(p, "risk_at_least_elevated", step.OnDistress)
		if action == "pause" {
			v := c.PIPaused
			patch.NewStatus = &v
		} else {
			v := c.PIStopped
			patch.NewStatus = &v
		}
		patch.Reason = "PR-SAFETY:risk_at_least_elevated"
		e.setFollowUp(p, "risk_at_least_elevated", snapshot.User, &patch)
		return zeroSpec, patch, nil
	}
	if directive.Outcome != nil && *directive.Outcome == c.OutDistress {
		if step.OnDistress == "pause" {
			v := c.PIPaused
			patch.NewStatus = &v
		} else {
			v := c.PIStopped
			patch.NewStatus = &v
		}
		patch.Reason = "PR-SAFETY:step_distress"
		return zeroSpec, patch, nil
	}
	if directive.Outcome != nil && *directive.Outcome == c.OutDecline {
		if step.OnDecline == "pause" {
			v := c.PIPaused
			patch.NewStatus = &v
		} else {
			v := c.PIStopped
			patch.NewStatus = &v
		}
		patch.Reason = "PR-STOP:user_declined"
		return zeroSpec, patch, nil
	}
	switch directive.Action {
	case c.PAStop, c.PADecline:
		v := c.PIStopped
		patch.NewStatus = &v
		patch.Reason = "PR-STOP:user_request"
		return zeroSpec, patch, nil
	case c.PAPause:
		v := c.PIPaused
		patch.NewStatus = &v
		patch.Reason = "PR-PAUSE:user_request"
		return zeroSpec, patch, nil
	case c.PARepeat:
		if directive.Outcome != nil && *directive.Outcome == c.OutStuck {
			patch.StuckInc = 1
		}
		if directive.Outcome != nil && *directive.Outcome == c.OutOffTopic && directive.Note != nil && *directive.Note == "topic_shift" {
			v := c.PIPaused
			patch.NewStatus = &v
			patch.Reason = "PR-PAUSE:topic_shift_later"
			e.setFollowUp(p, "topic_shift_later", snapshot.User, &patch)
			return zeroSpec, patch, nil
		}
		patch.AttemptsInc = 1
		patch.Reason = "PR-REPEAT:requested"
		return spec, patch, nil
	case c.PAAdapt:
		if directive.Outcome == nil || (*directive.Outcome != c.OutStuck && *directive.Outcome != c.OutPartial) {
			return zeroSpec, zeroPatch, ErrInvalid
		}
		if step.OnStuck.Adapt == "" {
			patch.AttemptsInc = 1
			patch.Reason = "PR-REPEAT:no_reviewed_adaptation"
			return spec, patch, nil
		}
		patch.AttemptsInc = 1
		patch.StuckInc = 1
		spec = e.stepSpec(p, *step, in.Active.ID, step.OnStuck.Adapt)
		patch.Reason = "PR-ADAPT:reviewed_adaptation"
		return spec, patch, nil
	case c.PAAdvance, c.PAComplete:
		if directive.Action == c.PAComplete && step.OnSuccess != "complete" {
			return zeroSpec, zeroPatch, ErrInvalid
		}
		if directive.Outcome == nil || *directive.Outcome != c.OutSuccess {
			patch.AttemptsInc = 1
			if directive.Outcome != nil && *directive.Outcome == c.OutStuck {
				patch.StuckInc = 1
			}
			patch.Reason = "PR-REPEAT:outcome_not_success"
			return spec, patch, nil
		}
		for key, value := range directive.InputsCaptured {
			if !has(step.ExpectedInputs, key) || strings.TrimSpace(value) == "" || len(strings.Fields(value)) > 30 || len(key) > 100 {
				return zeroSpec, zeroPatch, ErrInvalid
			}
			patch.Inputs[key] = value
		}
		patch.AttemptsInc = 1
		patch.StuckReset = true
		if step.OnSuccess == "complete" {
			v := c.PICompleted
			patch.NewStatus = &v
			outcome := "completed"
			patch.Outcome = &outcome
			patch.Reason = "PR-COMPLETE:reviewed_terminal_edge"
			if snapshot.User.ProactiveOptIn && p.Completion.FollowUpDays > 0 {
				due := e.now().UTC().AddDate(0, 0, p.Completion.FollowUpDays)
				patch.FollowUpAt = &due
			}
			return zeroSpec, patch, nil
		}
		if directive.Action == c.PAComplete {
			return zeroSpec, zeroPatch, ErrInvalid
		}
		var next *bundle.ProtocolStep
		for i := range p.Steps {
			if p.Steps[i].ID == step.OnSuccess {
				next = &p.Steps[i]
				break
			}
		}
		if next == nil {
			return zeroSpec, zeroPatch, ErrInvalid
		}
		patch.NewStep = &next.ID
		patch.Reason = "PR-ADVANCE:reviewed_success_edge"
		return e.stepSpec(p, *next, in.Active.ID, ""), patch, nil
	default:
		return zeroSpec, zeroPatch, ErrInvalid
	}
}

func (e *Engine) offer(ctx context.Context, in ResolveInput, s contextstate.Snapshot, risk c.RiskLevel) (stepResult c.ProtocolStepSpec, patchResult c.ProtocolPatch, resolveErr error) {
	ctx, inspectionSpan := trace.Start(ctx, "application/protocol/engine.offer", func() any { return map[string]any{"in": in, "s": s, "risk": risk} })
	defer func() {
		inspectionSpan.Complete(func() any { return map[string]any{"stepResult": stepResult, "patchResult": patchResult} }, resolveErr, recover())
	}()

	var none c.ProtocolStepSpec
	var noPatch c.ProtocolPatch
	d := in.Plan.Protocol
	if in.Active != nil || s.Protocol != nil || in.Objective != c.ObjOfferIntervention || d == nil || d.Action != c.PAOffer || d.ProtocolID == nil || len(d.InputsCaptured) > 0 || risk.Rank() > c.RiskLow.Rank() || in.WCtx.Risk.FollowUpActive || !in.WCtx.Eligibility.ProtocolsPossible {
		return none, noPatch, ErrInvalid
	}
	// JEV selection alone is never authority to offer. Recheck eligibility against
	// persisted state before accepting an offer for a selected protocol definition.
	var focus *c.ThreadRecord
	for i := range s.Threads {
		if in.WCtx.FocusThread != nil && s.Threads[i].ID == in.WCtx.FocusThread.ID && s.Threads[i].UserID == in.UserID {
			focus = &s.Threads[i]
			break
		}
	}
	actual, err := e.ShortlistWithContext(ctx, ShortlistContext{WCtx: in.WCtx, Thread: focus, User: s.User, Now: e.now()})
	if err != nil {
		return none, noPatch, err
	}
	authorized := false
	for _, v := range actual {
		if v.ProtocolID == *d.ProtocolID {
			authorized = true
		}
	}
	if !authorized {
		return none, noPatch, ErrInvalid
	}
	included := false
	for _, contentID := range in.SelectedContentIDs {
		if contentID == "protocols/"+*d.ProtocolID+".yaml" {
			included = true
			break
		}
	}
	if !included {
		return none, noPatch, ErrInvalid
	}
	p, ok := e.assets.Protocol(*d.ProtocolID)
	if !ok {
		return none, noPatch, ErrUnreviewed
	}
	if len(p.Steps) == 0 || len(p.Offer.WhyTemplates) == 0 {
		return none, noPatch, ErrUnreviewed
	}
	var tid *c.ID
	if in.WCtx.FocusThread != nil {
		id := in.WCtx.FocusThread.ID
		if _, owned := contextstate.FindThread(s, id); !owned {
			return none, noPatch, ErrInvalid
		}
		tid = &id
	}

	why := []string{}
	for _, id := range p.Offer.WhyTemplates {
		v := e.assets.offers.WhyTemplates[id]
		if strings.TrimSpace(v.Text) == "" {
			return none, noPatch, ErrUnreviewed
		}
		why = append(why, v.Text)
	}
	instanceID := c.NewID()
	now := e.now().UTC()
	patch := c.ProtocolPatch{InstanceID: instanceID, Create: &c.ProtocolInstanceInit{ProtocolID: p.ID, Version: p.Version, ThreadID: tid, OfferedAt: now}, NewStep: &p.Steps[0].ID, Inputs: map[string]string{}, Reason: "PR-OFFER:eligible_shortlist"}
	spec := c.ProtocolStepSpec{InstanceID: instanceID, ProtocolID: p.ID, MustSay: append([]string(nil), p.Offer.MustSay...), Offer: &c.OfferSpec{ProtocolID: p.ID, Name: p.Name, Why: why, EstMinutes: p.EstMinutes, OptOutLine: true}}

	// A clear request for this kind of exercise is already opt-in. Record the
	// same owned message as consent and deliver the first step without re-asking.
	if in.UserAffirmed {
		owned, err := e.reader.UserEvidence(ctx, in.UserID, []c.ID{in.UserMessageID})
		if err != nil {
			return none, noPatch, err
		}
		if !owned[in.UserMessageID] {
			return none, noPatch, ErrConsent
		}
		status := c.PIActive
		patch.NewStatus, patch.ConsentAt, patch.ConsentMessageID = &status, &now, &in.UserMessageID
		patch.Reason = "PR-START:user_requested_exercise"
		spec = e.stepSpec(p, p.Steps[0], instanceID, "")
	}
	return spec, patch, nil
}

func (e *Engine) stepSpec(p bundle.Protocol, step bundle.ProtocolStep, instance c.ID, adapt string) c.ProtocolStepSpec {
	spec := c.ProtocolStepSpec{InstanceID: instance, ProtocolID: p.ID, StepID: step.ID, Instruction: step.Instruction, ExpectedInputs: append([]string(nil), step.ExpectedInputs...), MustSay: append([]string(nil), p.Offer.MustSay...)}
	for _, a := range step.Adaptations {
		spec.AllowedAdaptations = append(spec.AllowedAdaptations, a.ID)
		if a.ID == adapt {
			spec.Instruction = a.Instruction
		}
	}
	if step.UI.Widget != "" && step.UI.Widget != "none" {
		spec.UI = &c.ProtocolUI{ProtocolInstanceID: instance, StepID: step.ID, Widget: step.UI.Widget}
	}
	return spec
}
func stopRule(p bundle.Protocol, when, fallback string) string {
	for _, r := range p.StopRules {
		if r.When == when && (r.Action == "stop" || r.Action == "pause") {
			return r.Action
		}
	}
	return fallback
}

// setFollowUp writes only the protocol due timestamp. WP-14 owns dispatching
// and is not activated here. No proactive work is scheduled without opt-in.
func (e *Engine) setFollowUp(p bundle.Protocol, when string, user c.UserState, patch *c.ProtocolPatch) {
	if !user.ProactiveOptIn {
		return
	}
	for _, r := range p.StopRules {
		if r.When == when && r.FollowUpDays > 0 {
			due := e.now().UTC().AddDate(0, 0, r.FollowUpDays)
			patch.FollowUpAt = &due
			return
		}
	}
}

// RequiredText resolves only reviewed identifiers from this pinned graph. The
// orchestrator binds them to WP-07 on the same turn context as generation.
func (e *Engine) RequiredText(spec c.ProtocolStepSpec) (map[string]string, error) {
	p, ok := e.assets.Protocol(spec.ProtocolID)
	if !ok {
		return nil, ErrInvalid
	}
	out := map[string]string{}
	for _, id := range spec.MustSay {
		if !slices.Contains(p.Offer.MustSay, id) {
			return nil, ErrInvalid
		}
		text := e.assets.offers.MustSay[id].Text
		if text == "" {
			return nil, ErrInvalid
		}
		out[id] = text
	}
	return out, nil
}
