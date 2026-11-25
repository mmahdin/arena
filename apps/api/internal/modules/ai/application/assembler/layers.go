package assembler

import (
	"context"
	"encoding/json"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"sort"
)

// RecentThrough can recover an early turn even when many later messages have
// been admitted. SQL must select by message_order BEFORE applying its TOP bound.
func RecentThrough(ctx context.Context, store ports.ConversationStore, user, conv, trigger c.ID, n int) ([]c.Message, error) {
	if bounded, ok := store.(interface {
		RecentThrough(context.Context, c.ID, c.ID, c.ID, int) ([]c.Message, error)
	}); ok {
		return bounded.RecentThrough(ctx, user, conv, trigger, n)
	}
	return store.Recent(ctx, user, conv, n)
}
func tokenCost(v any) int { b, _ := json.Marshal(v); return (len(b) + 3) / 4 }

// Layers reserves budgets independently. The original current message is never
// shortened; oversized mandatory state fails explicitly at model input validation.
func Layers(w c.WorkingContext, s contextstate.Snapshot, baseline c.PersistentContext, retrieved c.RetrievalResult, limits ...bundle.ContextTokenBudgets) c.WorkingContext {
	var budget bundle.ContextTokenBudgets
	if len(limits) > 0 {
		budget = limits[0]
	}
	w.SchemaVersion = "wctx.v3"
	w.InfoNeeds = nil
	w.FutureHints = append([]c.FutureHint(nil), s.FutureHints...)
	w.Persistent = baseline
	w.Persistent.Items = nil
	if baseline.Profile != nil {
		v := *baseline.Profile
		v.DisplayName = clipRunes(v.DisplayName, 120)
		v.Onboarding = map[string][]string{}
		for _, key := range []string{"focus", "concerns", "relationships", "strengths", "recharge"} {
			values := boundedStrings(baseline.Profile.Onboarding[key], 2, 100)
			if len(values) == 0 {
				continue
			}
			v.Onboarding[key] = values
			if budget.Profile > 0 && tokenCost(v) > budget.Profile {
				delete(v.Onboarding, key)
			}
		}
		w.Persistent.Profile = &v
	}
	seen := map[c.ID]bool{}
	for _, m := range baseline.Items {
		if seen[m.ID] || !c.ValidID(m.ID) {
			continue
		}
		m.Text = clipRunes(m.Text, 720)
		candidate := append(append([]c.MemoryView{}, w.Persistent.Items...), m)
		if budget.Persistent > 0 && tokenCost(c.PersistentContext{Items: candidate, Profile: w.Persistent.Profile, Degraded: baseline.Degraded, Reason: baseline.Reason}) > budget.Persistent {
			continue
		}
		w.Persistent.Items = candidate
		seen[m.ID] = true
	}
	if s.Formulation != nil {
		f := *s.Formulation
		f.Paragraph = clipRunes(f.Paragraph, 500)
		w.Formulation = &f
	}
	w.Formulations = nil
	w.ActiveThreads = nil
	threads := append([]c.ThreadRecord{}, s.Threads...)
	relevant := map[c.ID]bool{}
	for _, m := range retrieved.Items {
		if m.Source == "thread" {
			relevant[m.ID] = true
		}
		if m.ThreadID != nil {
			relevant[*m.ThreadID] = true
		}
	}
	sort.SliceStable(threads, func(i, j int) bool {
		a, b := threads[i], threads[j]
		af := w.FocusThread != nil && a.ID == w.FocusThread.ID
		bf := w.FocusThread != nil && b.ID == w.FocusThread.ID
		if af != bf {
			return af
		}
		if relevant[a.ID] != relevant[b.ID] {
			return relevant[a.ID]
		}
		if a.LastActiveAt.Equal(b.LastActiveAt) {
			return a.ID.String() < b.ID.String()
		}
		return a.LastActiveAt.After(b.LastActiveAt)
	})
	for _, t := range threads {
		if t.UserID != s.User.UserID || t.Phase == c.PhaseResolved {
			continue
		}
		v := c.ThreadView{ID: t.ID, Topic: clipRunes(t.Topic, 160), Phase: t.Phase, Goal: t.Goal, LastActive: t.LastActiveAt}
		for _, slot := range t.Slots {
			slot.Value = clipRunes(slot.Value, 180)
			v.Slots = append(v.Slots, slot)
		}
		// SQL map iteration is nondeterministic; sort before budget selection.
		sortSlots(v.Slots)
		if len(v.Slots) > 4 {
			v.Slots = v.Slots[:4]
		}
		if v.Goal != nil {
			value := clipRunes(*v.Goal, 240)
			v.Goal = &value
		}
		for _, m := range baseline.Items {
			for _, id := range t.HypothesisIDs {
				if m.ID == id && m.Kind == c.MemHypothesis && m.Epistemic == c.EpInferred {
					v.Hypotheses = append(v.Hypotheses, c.HypView{ID: id, SourceVersion: m.SourceVersion, Text: clipRunes(m.Text, 240), Status: m.Status, Confidence: c.ConfLow})
				}
			}
		}
		if w.FocusThread != nil && w.FocusThread.ID == t.ID {
			v.Slots = append([]c.SlotView{}, w.FocusThread.Slots...)
			for i := range v.Slots {
				v.Slots[i].Value = clipRunes(v.Slots[i].Value, 180)
				if len(v.Slots[i].Evidence) > 4 {
					v.Slots[i].Evidence = v.Slots[i].Evidence[:4]
				}
			}
			v.Hypotheses = boundedHypotheses(v.Hypotheses)
			w.FocusThread = &v
			continue
		}
		if len(w.ActiveThreads) < 4 {
			w.ActiveThreads = append(w.ActiveThreads, v)
		}
	}
	for _, f := range s.Formulations {
		f.Paragraph = clipRunes(f.Paragraph, 500)
		if len(w.Formulations) < 4 {
			w.Formulations = append(w.Formulations, f)
		}
	}
	w.Session.Summary = clipRunes(s.Session.Summary, 1800)
	if w.Session.PendingQuestion != nil {
		v := *w.Session.PendingQuestion
		v.Text = clipRunes(v.Text, 240)
		w.Session.PendingQuestion = &v
	}
	for i := range w.OpenLoops {
		w.OpenLoops[i].Text = clipRunes(w.OpenLoops[i].Text, 180)
	}
	w.Prefs.Avoid = boundedStrings(w.Prefs.Avoid, 12, 100)
	w.Prefs.Likes = boundedStrings(w.Prefs.Likes, 12, 100)
	w.Prefs.AddressForm = clipRunes(w.Prefs.AddressForm, 120)
	for budget.Recent > 0 && tokenCost(w.Recent) > budget.Recent && len(w.Recent) > 0 {
		w.Recent = w.Recent[1:]
	}
	// Collapse the same source identity across layers, not similar accounts.
	seen[w.Turn.MessageID] = true
	for _, m := range w.Recent {
		seen[m.ID] = true
	}
	if w.FocusThread != nil {
		seen[w.FocusThread.ID] = true
	}
	for _, t := range w.ActiveThreads {
		seen[t.ID] = true
	}
	if w.Protocol != nil {
		seen[w.Protocol.InstanceID] = true
	}
	for _, f := range w.Formulations {
		seen[f.ID] = true
	}
	if w.Formulation != nil {
		duplicate := false
		for _, f := range w.Formulations {
			if f.ID == w.Formulation.ID {
				duplicate = true
			}
		}
		if duplicate {
			w.Formulation = nil
		} else {
			seen[w.Formulation.ID] = true
		}
	}
	briefs := w.OtherThreads[:0]
	for _, t := range w.OtherThreads {
		if !seen[t.ID] {
			briefs = append(briefs, t)
		}
	}
	w.OtherThreads = briefs
	// A hypothesis already represented by a persistent memory keeps its original
	// evidence and epistemic label there instead of repeating it in thread state.
	if w.FocusThread != nil {
		v := *w.FocusThread
		v.Hypotheses = nil
		for _, h := range w.FocusThread.Hypotheses {
			if !seen[h.ID] {
				v.Hypotheses = append(v.Hypotheses, h)
			} else {
				v.HypothesisRefs = append(v.HypothesisRefs, h.ID)
			}
		}
		w.FocusThread = &v
	}
	for i := range w.ActiveThreads {
		keep := []c.HypView{}
		for _, h := range w.ActiveThreads[i].Hypotheses {
			if !seen[h.ID] {
				keep = append(keep, h)
			} else {
				w.ActiveThreads[i].HypothesisRefs = append(w.ActiveThreads[i].HypothesisRefs, h.ID)
			}
		}
		w.ActiveThreads[i].Hypotheses = keep
	}
	w.Retrieved = nil
	w.RetrievalMeta = retrieved.Meta
	w.RetrievalMeta.Queries = nil // query text is traced separately, not duplicated in model context
	for _, m := range retrieved.Items {
		if m.Source == "session" && m.ID == w.Session.SessionID && w.Session.Summary != "" {
			continue // This exact session source is already represented in active state.
		}
		if seen[m.ID] {
			continue
		}
		m.Text = clipRunes(m.Text, 900)
		if budget.Retrieved > 0 && tokenCost(w.Retrieved)+tokenCost(m) > budget.Retrieved {
			continue
		}
		w.Retrieved = append(w.Retrieved, m)
		seen[m.ID] = true
	}
	w.Eligibility.MemoryAvailable = len(w.Retrieved) > 0 || !retrieved.Meta.Degraded
	w.LayerTokens = c.ContextLayerTokens{Persistent: tokenCost(w.Persistent), Retrieved: tokenCost(w.Retrieved), Recent: tokenCost(w.Recent)}
	active := w
	active.Persistent = c.PersistentContext{}
	active.Retrieved = nil
	active.Recent = nil
	active.Turn.Text = ""
	active.TokenEstimate = 0
	active.LayerTokens = c.ContextLayerTokens{}
	w.LayerTokens.Active = tokenCost(active)
	w.TokenEstimate = estimatedTokens(w)
	return w
}
func sortSlots(s []c.SlotView) { sort.Slice(s, func(i, j int) bool { return s[i].Slot < s[j].Slot }) }
func boundedHypotheses(h []c.HypView) []c.HypView {
	if len(h) > 3 {
		return h[:3]
	}
	return h
}

func boundedStrings(in []string, n, size int) []string {
	out := []string{}
	for _, v := range in {
		if len(out) >= n {
			break
		}
		out = append(out, clipRunes(v, size))
	}
	return out
}

// BoundActive bounds assembled active state. Optional redundant
// material yields first; mandatory active state has an explicit failure budget.
func BoundActive(w c.WorkingContext, limits ...int) c.WorkingContext {
	var limit int
	if len(limits) > 0 {
		limit = limits[0]
	}
	cost := func() int {
		v := w
		v.Persistent = c.PersistentContext{}
		v.Retrieved = nil
		v.Recent = nil
		v.Turn.Text = ""
		v.TokenEstimate = 0
		v.LayerTokens = c.ContextLayerTokens{}
		return tokenCost(v)
	}
	for limit > 0 && cost() > limit && len(w.OtherThreads) > 0 {
		w.OtherThreads = w.OtherThreads[:len(w.OtherThreads)-1]
	}
	for limit > 0 && cost() > limit && len(w.Formulations) > 0 {
		w.Formulations = w.Formulations[:len(w.Formulations)-1]
	}
	for limit > 0 && cost() > limit && len(w.ActiveThreads) > 0 {
		w.ActiveThreads = w.ActiveThreads[:len(w.ActiveThreads)-1]
	}
	w.LayerTokens.Active = cost()
	w.TokenEstimate = estimatedTokens(w)
	return w
}
