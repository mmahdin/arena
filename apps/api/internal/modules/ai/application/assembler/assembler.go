// Package assembler builds a deterministic, bounded WorkingContext from
// WP-02 state, WP-03 pinned schemas, and a caller-supplied retrieval result.
// It never calls an LLM or imports the experimental AI context implementation.
package assembler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"
	"sort"
	"strings"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type Service struct {
	reader        contextstate.Reader
	conversations ports.ConversationStore
	bundles       ports.BundleRegistry
	now           func() time.Time
}

var _ ports.ContextAssembler = (*Service)(nil)

func New(reader contextstate.Reader, conversations ports.ConversationStore, bundles ports.BundleRegistry) (*Service, error) {
	if reader == nil || conversations == nil || bundles == nil {
		return nil, errors.New("WP-08 assembler requires state, conversation, and bundle readers")
	}
	return &Service{reader: reader, conversations: conversations, bundles: bundles, now: time.Now}, nil
}

// WithClock is intentionally a per-instance test hook, not a global time
// source. Set it before using the service concurrently.
func (a *Service) WithClock(now func() time.Time) *Service {
	if now != nil {
		a.now = now
	}
	return a
}

func (a *Service) Build(ctx context.Context, req c.TurnRequest, sig c.SafetySignal, retr c.RetrievalResult) (contextResult c.WorkingContext, stateResult ports.StateSnapshot, buildErr error) {
	ctx, inspectionSpan := trace.Start(ctx, "application/assembler/assembler.Build", func() any { return map[string]any{"req": req, "sig": sig, "retr": retr} })
	defer func() {
		inspectionSpan.Complete(func() any { return map[string]any{"contextResult": contextResult, "stateResult": stateResult} }, buildErr, recover())
	}()

	var empty ports.StateSnapshot
	if err := ctx.Err(); err != nil {
		return c.WorkingContext{}, empty, err
	}
	if !c.ValidID(req.UserID) || !c.ValidID(req.ConversationID) || !c.ValidID(req.ClientTurnID) {
		return c.WorkingContext{}, empty, observability.Detail(nil, "identity_mismatch", "call_identity", "assemble")
	}
	// The turn identifiers are server owned, never inferred from clientTurnID.
	turn, err := a.conversations.GetTurn(ctx, req.UserID, req.ClientTurnID)
	if err != nil {
		return c.WorkingContext{}, empty, err
	}
	if !c.ValidID(turn.TurnID) || !c.ValidID(turn.UserMessageID) || turn.Status != c.TurnRunning {
		return c.WorkingContext{}, empty, observability.Detail(nil, "context_state_mismatch", "turn_admission", "assemble")
	}
	snapshot, prefetched := contextstate.Prefetched(ctx, req.UserID, req.ConversationID)
	if !prefetched {
		snapshot, err = a.reader.Read(ctx, req.UserID, req.ConversationID)
	}
	if err != nil {
		return c.WorkingContext{}, empty, err
	}
	if snapshot.User.UserID != req.UserID || snapshot.Session.UserID != req.UserID ||
		snapshot.Session.ConversationID != req.ConversationID || !c.ValidID(snapshot.Session.ID) ||
		snapshot.Session.ClosedAt != nil || snapshot.Session.Version <= 0 || snapshot.User.Version <= 0 {
		return c.WorkingContext{}, empty, observability.Detail(nil, "context_state_mismatch", "context_state", "assemble")
	}
	policy, err := contextstate.LoadPolicy(ctx, a.bundles, req.UserID)
	if err != nil {
		return c.WorkingContext{}, empty, err
	}
	messages, err := RecentThrough(ctx, a.conversations, req.UserID, req.ConversationID, turn.UserMessageID, policy.Budgets.Operations.RecentMessages)
	if err != nil {
		return c.WorkingContext{}, empty, err
	}
	now := a.now().UTC()
	loc := timezone(snapshot.User.TZ, req.TZ)
	// User.Locale is historical request metadata, not a saved preference.
	responseLanguage := c.ResponseLanguage(snapshot.User.Prefs.Language, req.Locale)
	// WP-02's ConversationStore.Recent returns message_order ascending.
	// A queued later user turn can already have a durable raw message; never
	// leak that future turn into this one during recovery or replay.
	triggerIndex := -1
	for i, m := range messages {
		if m.UserID != req.UserID || m.ConvID != req.ConversationID {
			return c.WorkingContext{}, empty, observability.Detail(nil, "ownership_mismatch", "recent_message", "assemble")
		}
		if m.ID == turn.UserMessageID {
			if m.Role != c.RoleUser {
				return c.WorkingContext{}, empty, observability.Detail(nil, "source_role_mismatch", "current_message", "assemble")
			}
			triggerIndex = i
			break
		}
	}
	if triggerIndex == -1 {
		return c.WorkingContext{}, empty, observability.Detail(nil, "source_provenance_invalid", "current_message", "assemble")
	}
	past := make([]c.Message, 0, triggerIndex)
	var lastAt time.Time
	for _, m := range messages[:triggerIndex] {
		if m.UserID != req.UserID || m.ConvID != req.ConversationID {
			return c.WorkingContext{}, empty, observability.Detail(nil, "ownership_mismatch", "recent_message", "assemble")
		}
		if m.Role != c.RoleUser && m.Role != c.RoleAssistant {
			continue
		}
		past = append(past, m)
		if m.CreatedAt.After(lastAt) && !m.CreatedAt.After(now) {
			lastAt = m.CreatedAt
		}
	}
	recent := buildRecent(past, now, policy.Budgets.Operations.RecentMessages, policy.Budgets.Context.Recent)
	risk := maxRisk(snapshot.User.Risk.Level, sig.Level)
	if !risk.Valid() {
		risk = c.RiskElevated
	} // fail conservatively
	permittedRetr := retr
	// Partial lexical results retain valid provenance even when vector search failed.
	focus, others := threadViews(snapshot, permittedRetr, now)
	completeSlots(focus, policy)
	var protocol *c.ProtocolView
	if snapshot.Protocol != nil && (snapshot.Protocol.Status == c.PIOffered || snapshot.Protocol.Status == c.PIActive || snapshot.Protocol.Status == c.PIPaused) {
		p := snapshot.Protocol
		pv := &c.ProtocolView{InstanceID: p.ID, ProtocolID: p.ProtocolID, Version: p.ProtocolVersion, Status: p.Status, StepID: p.StepID,
			Attempts: p.Attempts, StuckCount: p.StuckCount, LastUserSummary: shortText(p.LastUserSummary, 45)}
		if graph, e := contextstate.ProtocolGraph(ctx, policy.Handle, p.ProtocolID); e == nil && graph.Version == p.ProtocolVersion {
			for _, step := range graph.Steps {
				if step.ID == p.StepID {
					pv.StepGoal = shortText(step.Goal, 40)
					pv.SuccessCriteria = step.SuccessCriteria
					pv.ExpectedInput = strings.Join(step.ExpectedInputs, ", ")
					break
				}
			}
			if p.Status == c.PIActive {
				pv.AllowedActions = []c.ProtocolAction{c.PAAdvance, c.PARepeat, c.PAAdapt, c.PAPause, c.PAStop, c.PAComplete}
			}
			if p.Status == c.PIOffered {
				pv.AllowedActions = []c.ProtocolAction{c.PAAdvance, c.PADecline, c.PAStop}
			}
			if p.Status == c.PIPaused {
				pv.AllowedActions = []c.ProtocolAction{c.PAAdvance, c.PAStop}
			}
		} // missing or unapproved definition is never guessed
		protocol = pv
	}
	var gap time.Duration
	if !lastAt.IsZero() {
		gap = now.Sub(lastAt)
		if gap < 0 {
			gap = 0
		}
	}
	if snapshot.Session.GapSincePrevious != nil && (lastAt.IsZero() || *snapshot.Session.GapSincePrevious > gap) {
		gap = *snapshot.Session.GapSincePrevious
	}
	resumed := gap >= time.Duration(policy.Thresholds.Session.IdleTimeoutMin)*time.Minute && !lastAt.IsZero()
	var resumption *c.ResumptionView
	if resumed {
		resumption = &c.ResumptionView{Gap: gap, Band: gapBand(gap, policy), RecentRisk: riskOrder(snapshot.User.Risk.Level) >= 2}
	}
	session := c.SessionView{SessionID: snapshot.Session.ID, Mode: snapshot.Session.Mode, TurnCount: snapshot.Session.TurnCount,
		LastObjective: snapshot.Session.LastObjective, ConsecutiveQuestions: snapshot.Session.ConsecutiveQuestions,
		PendingQuestion: snapshot.Session.PendingQuestion, Summary: shortText(snapshot.Session.Summary, 120), Resumed: resumed}
	if len(snapshot.Session.EmotionTrajectory) > 5 {
		session.EmotionTrajectory = append([]string(nil), snapshot.Session.EmotionTrajectory[len(snapshot.Session.EmotionTrajectory)-5:]...)
	} else {
		session.EmotionTrajectory = append([]string(nil), snapshot.Session.EmotionTrajectory...)
	}
	if !session.Mode.Valid() {
		session.Mode = c.ModeExploring
	}
	if session.ConsecutiveQuestions < 0 {
		session.ConsecutiveQuestions = 0
	}
	pref := snapshot.User.Prefs
	if !pref.Length.Valid() {
		pref.Length = c.LengthShort
	}
	if !pref.Tone.Valid() {
		pref.Tone = c.ToneWarmSteady
	}
	pref.Language = responseLanguage
	pref.Avoid = append([]string(nil), pref.Avoid...)
	pref.Likes = append([]string(nil), pref.Likes...)
	eligibility := computeEligibility(snapshot, focus, protocol, risk, policy, now)
	memoryAvailable := !retr.Meta.Degraded || len(retr.Items) > 0 && len(retr.Meta.Strategies) > 0
	eligibility.MemoryAvailable = memoryAvailable
	retrieved := []c.MemoryView{}
	if memoryAvailable {
		retrieved = boundedMemory(retr.Items, retr.Meta.Hops)
	}
	var formulation *c.FormulationBrief
	if focus != nil && snapshot.Formulation != nil && snapshot.Formulation.ThreadID == focus.ID {
		f := *snapshot.Formulation
		f.Paragraph = shortText(f.Paragraph, 90)
		f.Gaps = append([]string(nil), f.Gaps...)
		formulation = &f
	}
	local := now.In(loc).Format("Mon 15:04")
	turnView := c.TurnView{TurnID: turn.TurnID, MessageID: turn.UserMessageID, Text: req.Text, Lang: req.Locale,
		Origin: req.Origin, LocalTime: local, SinceLastMsg: gap}
	if req.WakeUp != nil {
		turnView.WakeUpReason = req.WakeUp.Reason
	}
	out := c.WorkingContext{SchemaVersion: "wctx.v2", Turn: turnView, Recent: recent, Session: session, FocusThread: focus, OtherThreads: others,
		Protocol: protocol, Prefs: pref, Risk: c.RiskView{Current: risk, DependencyWatch: snapshot.User.DependencyWatch,
			FollowUpActive: snapshot.User.Risk.FollowUpUntil != nil && snapshot.User.Risk.FollowUpUntil.After(now)},
		OpenLoops: boundedLoops(snapshot.OpenLoops), RetrievalMeta: retr.Meta, Retrieved: retrieved,
		Resumption: resumption, Formulation: formulation, Eligibility: eligibility}
	// Respect the configured context target without truncating source IDs,
	// ownership/version state or the triggering user message.
	target := policy.Budgets.Tokens.WCtxTarget
	out.TokenEstimate = estimatedTokens(out)
	if !prefetched {
		for out.TokenEstimate > target && len(out.Retrieved) > 0 {
			out.Retrieved = out.Retrieved[:len(out.Retrieved)-1]
			out.TokenEstimate = estimatedTokens(out)
		}
		for out.TokenEstimate > target && len(out.Recent) > 0 {
			out.Recent = out.Recent[1:]
			out.TokenEstimate = estimatedTokens(out)
		}
		if out.TokenEstimate > target {
			out.Session.Summary = ""
			out.TokenEstimate = estimatedTokens(out)
		}
	}
	var focusRecord *c.ThreadRecord
	if focus != nil {
		if t, ok := contextstate.FindThread(snapshot, focus.ID); ok {
			cp := t
			focusRecord = &cp
		}
	}
	ps := ports.StateSnapshot{Versions: contextstate.Versions(snapshot), Session: snapshot.Session, Focus: focusRecord,
		Threads: append([]c.ThreadRecord(nil), snapshot.Threads...), Protocol: snapshot.Protocol, User: snapshot.User}
	if err := ctx.Err(); err != nil {
		return c.WorkingContext{}, empty, err
	}
	return out, ps, nil
}

func estimatedTokens(w c.WorkingContext) int {
	w.TokenEstimate = 0
	b, _ := json.Marshal(w)
	return (len(b) + 3) / 4
}
func timezone(stateTZ, reqTZ string) *time.Location {
	for _, name := range []string{stateTZ, reqTZ, "UTC"} {
		if loc, e := time.LoadLocation(name); e == nil {
			return loc
		}
	}
	return time.UTC
}
func shortText(s string, maxWords int) string {
	words := strings.Fields(s)
	if len(words) <= maxWords {
		return s
	}
	return strings.Join(words[:maxWords], " ") + "…"
}
func ageString(now, at time.Time) string {
	if at.IsZero() {
		return "unknown"
	}
	d := now.Sub(at)
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
func buildRecent(in []c.Message, now time.Time, limits ...int) []c.MsgView {
	recentLimit, recentTokenBudget := len(in), 0
	if len(limits) > 0 {
		recentLimit = limits[0]
	}
	if len(limits) > 1 {
		recentTokenBudget = limits[1]
	}
	out := make([]c.MsgView, 0, recentLimit)
	used := 0
	for i := len(in) - 1; i >= 0 && len(out) < recentLimit; i-- {
		m := in[i]
		// Bound individual messages and the complete history using UTF-8-safe
		// rune cuts. Recent can contain old text, never the triggering message.
		text := clipRunes(m.Content, 900)
		cost := (len([]byte(text)) + 3) / 4
		if recentTokenBudget > 0 && cost+used > recentTokenBudget {
			room := (recentTokenBudget - used) * 4
			if room < 32 {
				break
			}
			text = clipRunes(text, room)
			cost = (len([]byte(text)) + 3) / 4
		}
		out = append(out, c.MsgView{ID: m.ID, Role: m.Role, Text: text, Age: ageString(now, m.CreatedAt), Objective: m.Meta.Objective})
		used += cost
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}
func clipRunes(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	if maxBytes < 6 {
		return "…"
	}
	for i, r := range s {
		if i+len(string(r)) > maxBytes-3 {
			return s[:i] + "…"
		}
	}
	return s
}
func boundedLoops(in []c.OpenLoop) []c.OpenLoop {
	if len(in) > 8 {
		in = in[:8]
	}
	return append([]c.OpenLoop(nil), in...)
}
func boundedMemory(in []c.MemoryView, hops int) []c.MemoryView {
	out := make([]c.MemoryView, 0, len(in))
	for _, v := range in {
		if !c.ValidID(v.ID) || !v.Kind.Valid() || !v.Epistemic.Valid() || !v.Status.Valid() {
			continue
		}
		v.Text = shortText(v.Text, 60)
		out = append(out, v)
	}
	return out
}
func threadViews(s contextstate.Snapshot, r c.RetrievalResult, now time.Time) (*c.ThreadView, []c.ThreadBrief) {
	threads := append([]c.ThreadRecord(nil), s.Threads...)
	sort.Slice(threads, func(i, j int) bool {
		if threads[i].LastActiveAt.Equal(threads[j].LastActiveAt) {
			return threads[i].ID.String() < threads[j].ID.String()
		}
		return threads[i].LastActiveAt.After(threads[j].LastActiveAt)
	})
	var focus *c.ThreadView
	others := []c.ThreadBrief{}
	for _, t := range threads {
		if t.UserID != s.User.UserID {
			continue
		}
		if s.Session.FocusThreadID != nil && t.ID == *s.Session.FocusThreadID && t.Phase != c.PhaseResolved && t.Phase != c.PhaseDormant && t.Phase != c.PhasePaused {
			slots := make([]c.SlotView, 0, len(t.Slots))
			for _, v := range t.Slots {
				v.Evidence = append([]c.ID(nil), v.Evidence...)
				slots = append(slots, v)
			}
			sort.Slice(slots, func(i, j int) bool { return slots[i].Slot < slots[j].Slot })
			focus = &c.ThreadView{ID: t.ID, Topic: t.Topic, Phase: t.Phase, Slots: slots, Goal: t.Goal, LastActive: t.LastActiveAt}
			// Hypotheses are never loaded straight from the memory table: only
			// the WP-09 retrieval result may grant access to memory-derived text.
			present := map[c.ID]bool{}
			for _, id := range t.HypothesisIDs {
				present[id] = true
			}
			for _, v := range r.Items {
				if len(focus.Hypotheses) >= 3 {
					break
				}
				if present[v.ID] && v.Kind == c.MemHypothesis && (v.Status == c.MemProposed || v.Status == c.MemSupported) {
					focus.Hypotheses = append(focus.Hypotheses, c.HypView{ID: v.ID, Text: shortText(v.Text, 40), Status: v.Status, Confidence: c.ConfLow})
				}
			}
			continue
		}
		if t.Phase != c.PhaseResolved && len(others) < 4 {
			others = append(others, c.ThreadBrief{ID: t.ID, Topic: t.Topic, Phase: t.Phase, LastActive: t.LastActiveAt})
		}
	}
	_ = now
	return focus, others
}
func riskOrder(v c.RiskLevel) int {
	switch v {
	case c.RiskNone:
		return 0
	case c.RiskLow:
		return 1
	case c.RiskElevated:
		return 2
	case c.RiskHigh:
		return 3
	case c.RiskImminent:
		return 4
	default:
		return -1
	}
}
func maxRisk(a, b c.RiskLevel) c.RiskLevel {
	if riskOrder(a) < 0 {
		a = c.RiskElevated
	}
	if riskOrder(b) < 0 {
		b = c.RiskElevated
	}
	if riskOrder(b) > riskOrder(a) {
		return b
	}
	return a
}
func gapBand(d time.Duration, p contextstate.Policy) string {
	switch {
	case d < time.Duration(p.Thresholds.ResumptionBands.SameMin)*time.Minute:
		return "same"
	case d < time.Duration(p.Thresholds.ResumptionBands.ShortHours)*time.Hour:
		return "short"
	case d < time.Duration(p.Thresholds.ResumptionBands.DayDays)*24*time.Hour:
		return "day"
	case d < 30*24*time.Hour:
		return "week"
	default:
		return "long"
	}
}
