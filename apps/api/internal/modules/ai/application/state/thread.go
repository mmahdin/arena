package state

import (
	"context"
	"fmt"
	"sort"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/contextstate"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

type rejectFunc func(path, rule, detail string)

func (s *Service) validateThread(ctx context.Context, snap contextstate.Snapshot, pol contextstate.Policy, d *c.StateDelta, reject rejectFunc) error {
	known := map[string]bool{}
	for _, def := range pol.Problem.Slots {
		known[def.ID] = true
	}
	t := d.Thread
	if t != nil {
		previous, exists := contextstate.FindThread(snap, t.ThreadID)
		if !c.ValidID(t.ThreadID) || t.Create != nil && exists || t.Create == nil && !exists {
			reject("thread", "ST-001", "thread must be owned and creation must use a new ID")
			d.Thread = nil
		} else if t.Create != nil && d.ExpectedVersions.Thread != 0 || t.Create == nil && d.ExpectedVersions.Thread != previous.Version {
			return stateErr(c.ErrStateConflict, fmt.Errorf("ST-001: thread version changed"))
		} else {
			if t.Create != nil && (t.Create.Topic == "" || len([]rune(t.Create.Topic)) > 400) {
				reject("thread.create", "ST-002", "thread topic invalid")
				d.Thread = nil
			} else {
				if t.Create != nil {
					previous = c.ThreadRecord{ID: t.ThreadID, UserID: d.UserID, Phase: c.PhaseExploring, Slots: map[string]c.SlotView{}}
				}
				allIDs := []c.ID{}
				for _, v := range t.Slots {
					allIDs = append(allIDs, v.Evidence...)
				}
				for _, v := range t.HypothesisSignals {
					allIDs = append(allIDs, v.Evidence...)
				}
				ids, err := s.reader.UserEvidence(ctx, d.UserID, uniqueIDs(allIDs))
				if err != nil {
					return err
				}
				validSlots := make([]c.SlotUpdate, 0, len(t.Slots))
				used := map[string]bool{}
				for i, v := range t.Slots {
					path := fmt.Sprintf("thread.slots[%d]", i)
					if !known[v.Slot] || used[v.Slot] {
						reject(path, "BR-007", "unknown or duplicate slot")
						continue
					}
					used[v.Slot] = true
					if v.Status != c.EpReported && v.Status != c.EpInferred || v.Value == "" {
						reject(path, "ST-004", "slot must contain reported/inferred value")
						continue
					}
					if wordCount(v.Value) > 30 {
						v.Value = firstWords(v.Value, 30)
						reject(path, "BR-020", "slot capped at 30 words")
					}
					filtered := make([]c.ID, 0, len(v.Evidence))
					for _, id := range uniqueIDs(v.Evidence) {
						if ids[id] {
							filtered = append(filtered, id)
						} else {
							reject(path, "ST-004", "evidence not an owned user message")
						}
					}
					if len(filtered) == 0 {
						reject(path, "ST-004", "missing trustworthy evidence")
						continue
					}
					// No semantic support verifier exists before WP-10. Never
					// upgrade unsupported inferred text to user-reported.
					v.Evidence = filtered
					validSlots = append(validSlots, v)
				}
				t.Slots = validSlots
				validSignals := make([]c.HypothesisSignal, 0, len(t.HypothesisSignals))
				// Thread-local references alone do not prove database ownership.
				// Soft-deleted or rejected hypotheses must never be reused.
				ownedHypotheses, err := s.reader.ExistingHypotheses(ctx, d.UserID, previous.HypothesisIDs)
				if err != nil {
					return err
				}
				hypIDs := make(map[c.ID]bool, len(previous.HypothesisIDs))
				for _, id := range previous.HypothesisIDs {
					hypIDs[id] = ownedHypotheses[id] && !snap.User.MemoryPaused
				}
				for i, v := range t.HypothesisSignals {
					if !v.Op.Valid() || v.Op == c.HypPropose || v.ID == nil || !hypIDs[*v.ID] {
						// Propose/cooldown handling is owned by WP-12; the
						// state pipeline may only record existing owned IDs.
						reject(fmt.Sprintf("thread.hypothesis_signals[%d]", i), "ST-007", "unknown or unverified hypothesis")
						continue
					}
					good := true
					for _, id := range v.Evidence {
						if !ids[id] {
							good = false
							break
						}
					}
					if !good || len(v.Evidence) == 0 {
						reject("thread.hypothesis_signals", "ST-007", "hypothesis evidence missing or not user-owned")
						continue
					}
					validSignals = append(validSignals, v)
				}
				t.HypothesisSignals = validSignals
				if t.Phase != nil && !validPhaseChange(previous.Phase, *t.Phase, previous.PrevPhase) {
					reject("thread.phase", "ST-002", "disallowed phase transition")
					t.Phase = nil
				}
				if t.Phase != nil && !isActiveThread(*t.Phase) &&
					snap.Session.FocusThreadID != nil && *snap.Session.FocusThreadID == t.ThreadID &&
					(d.Session.FocusThread == nil || *d.Session.FocusThread == t.ThreadID) {
					// Clearing focus requires a new WP-02 SQL contract. Do not
					// leave a focused thread non-active in the meantime.
					reject("thread.phase", "ST-006", "focused thread cannot pause/resolve without a valid new focus")
					t.Phase = nil
				}
				if t.Phase != nil && *t.Phase == c.PhaseUnderstanding && !readyForUnderstanding(snap, previous, *t, pol) {
					reject("thread.phase", "ST-003", "readiness conditions not met")
					t.Phase = nil
				}
				for _, id := range t.RelatedThreadIDs {
					if id == t.ThreadID {
						reject("thread.related_thread_ids", "ST-006", "self link not allowed")
						continue
					}
				}
				// WP-02 always writes related_thread_ids_json, even for a
				// slots-only or phase-only patch. Preserve existing links.
				if t.Create == nil && len(t.RelatedThreadIDs) == 0 {
					t.RelatedThreadIDs = append([]c.ID{}, previous.RelatedThreadIDs...)
				}
				linked := []c.ID{}
				for _, id := range uniqueIDs(t.RelatedThreadIDs) {
					if id == t.ThreadID {
						continue
					}
					if _, ok := contextstate.FindThread(snap, id); ok {
						linked = append(linked, id)
					} else {
						reject("thread.related_thread_ids", "ST-006", "related thread not owned")
					}
				}
				t.RelatedThreadIDs = linked
			}
		}
	}
	// Secondary operations must target existing owner-scoped threads and obey
	// exactly the same phase graph. No hard-delete operation is supported.
	ops := make([]c.ThreadOp, 0, len(d.ThreadOps))
	seen := map[c.ID]bool{}
	for i, op := range d.ThreadOps {
		rec, ok := contextstate.FindThread(snap, op.ThreadID)
		if !ok || seen[op.ThreadID] {
			reject(fmt.Sprintf("thread_ops[%d]", i), "ST-002", "unowned or duplicate thread")
			continue
		}
		seen[op.ThreadID] = true
		target := rec.Phase
		switch op.Op {
		case "pause":
			target = c.PhasePaused
		case "dormant":
			target = c.PhaseDormant
		case "resolve":
			target = c.PhaseResolved
		case "resume":
			target = rec.PrevPhase
		default:
			reject(fmt.Sprintf("thread_ops[%d]", i), "ST-002", "unknown thread operation")
			continue
		}
		if !isActiveThread(target) && snap.Session.FocusThreadID != nil && *snap.Session.FocusThreadID == op.ThreadID &&
			(d.Session.FocusThread == nil || *d.Session.FocusThread == op.ThreadID) {
			reject(fmt.Sprintf("thread_ops[%d]", i), "ST-006", "cannot pause/resolve focused thread without refocusing")
			continue
		}
		if !validPhaseChange(rec.Phase, target, rec.PrevPhase) {
			reject(fmt.Sprintf("thread_ops[%d]", i), "ST-002", "invalid phase transition")
			continue
		}
		ops = append(ops, op)
	}
	d.ThreadOps = ops
	// ST-005. Compute projected active state *after* accepted secondary
	// operations. A user-proposed pause already satisfying the cap must not
	// cause a second, unrelated thread to be paused. Newly-created threads
	// require capacity in the same atomic WP-02 finalization.
	max := pol.Thresholds.Thread.MaxActive
	if max <= 0 || max > 3 {
		max = 3
	}
	projected := []c.ThreadRecord{}
	for _, rec := range snap.Threads {
		if rec.UserID != d.UserID {
			continue
		}
		phase := rec.Phase
		if d.Thread != nil && d.Thread.Create == nil && d.Thread.ThreadID == rec.ID && d.Thread.Phase != nil {
			phase = *d.Thread.Phase
		}
		for _, op := range d.ThreadOps {
			if op.ThreadID == rec.ID {
				switch op.Op {
				case "pause":
					phase = c.PhasePaused
				case "resolve":
					phase = c.PhaseResolved
				case "dormant":
					phase = c.PhaseDormant
				case "resume":
					phase = rec.PrevPhase
				}
			}
		}
		if isActiveThread(phase) {
			projected = append(projected, rec)
		}
	}
	incoming := 0
	if d.Thread != nil && d.Thread.Create != nil && (d.Thread.Phase == nil || isActiveThread(*d.Thread.Phase)) {
		incoming = 1
	}
	sort.Slice(projected, func(i, j int) bool {
		if projected[i].Priority != projected[j].Priority {
			return projected[i].Priority < projected[j].Priority
		}
		if !projected[i].LastActiveAt.Equal(projected[j].LastActiveAt) {
			return projected[i].LastActiveAt.Before(projected[j].LastActiveAt)
		}
		return projected[i].ID.String() < projected[j].ID.String()
	})
	for len(projected)+incoming > max && len(projected) > 0 {
		idx := 0
		for idx < len(projected) && snap.Session.FocusThreadID != nil && projected[idx].ID == *snap.Session.FocusThreadID &&
			(d.Session.FocusThread == nil || *d.Session.FocusThread == *snap.Session.FocusThreadID) {
			idx++
		}
		if idx == len(projected) {
			// Do not violate the focus invariant to satisfy the active cap.
			// Refuse the newly created thread if there is no safe victim.
			if incoming != 0 {
				reject("thread.create", "ST-005", "no safe thread available for automatic pause")
				d.Thread = nil
				incoming = 0
			}
			break
		}
		paused := projected[idx]
		d.ThreadOps = append(d.ThreadOps, c.ThreadOp{ThreadID: paused.ID, Op: "pause", Reason: "max_active_threads"})
		reject("thread_ops", "ST-005", "auto-paused lowest-priority eligible active thread")
		projected = append(projected[:idx], projected[idx+1:]...)
	}

	return nil
}

func uniqueIDs(ids []c.ID) []c.ID {
	out := make([]c.ID, 0, len(ids))
	m := map[c.ID]bool{}
	for _, id := range ids {
		if !m[id] {
			out = append(out, id)
			m[id] = true
		}
	}
	return out
}
func isActiveThread(p c.ThreadPhase) bool {
	return p == c.PhaseExploring || p == c.PhaseUnderstanding || p == c.PhaseWorking || p == c.PhaseMaintaining
}
func validPhaseChange(from, to, previous c.ThreadPhase) bool {
	if from == to {
		return true
	}
	if to == c.PhasePaused || to == c.PhaseDormant {
		return from != c.PhaseResolved
	}
	if from == c.PhasePaused || from == c.PhaseDormant {
		return to == previous && isActiveThread(to)
	}
	switch from {
	case c.PhaseExploring:
		return to == c.PhaseUnderstanding
	case c.PhaseUnderstanding:
		return to == c.PhaseWorking
	case c.PhaseWorking:
		return to == c.PhaseMaintaining
	case c.PhaseMaintaining:
		return to == c.PhaseResolved
	default:
		return false
	}
}
func readyForUnderstanding(s contextstate.Snapshot, rec c.ThreadRecord, p c.ThreadPatch, pol contextstate.Policy) bool {
	slots := map[string]c.SlotView{}
	for k, v := range rec.Slots {
		slots[k] = v
	}
	for _, update := range p.Slots {
		slots[update.Slot] = c.SlotView{Slot: update.Slot, Status: update.Status, Value: update.Value, Evidence: update.Evidence}
	}
	filled := func(id string) bool {
		v, ok := slots[id]
		return ok && (v.Status == c.EpReported || v.Status == c.EpInferred) && v.Value != "" && len(v.Evidence) > 0 && !v.Declined
	}
	for _, id := range pol.Problem.ReadinessRule.RequireFilled {
		if !filled(id) {
			return false
		}
	}
	any := len(pol.Problem.ReadinessRule.AndAnyOf) == 0
	for _, id := range pol.Problem.ReadinessRule.AndAnyOf {
		if filled(id) {
			any = true
			break
		}
	}
	if !any {
		return false
	}
	return s.ThreadUserTurns[rec.ID] >= pol.Problem.ReadinessRule.MinUserTurnsInThread
}
