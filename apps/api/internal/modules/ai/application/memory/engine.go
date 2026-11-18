package memory

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// Reconcile is pure with respect to persistence. Caller-supplied vectors must
// have been computed before the short optimistic SQL transaction.
func (a *Assets) Reconcile(ctx context.Context, s Snapshot, m Meta, ops []c.MemoryOpProposal, origin c.ProvenanceSrc, vectors map[string][]float32, policy Policy, now time.Time) (Plan, error) {
	plan := Plan{Report: ports.MemoryApplyReport{Applied: []c.ID{}, Rejected: []c.DeltaRejection{}}}
	if policy == nil || s.User.UserID != m.UserID || len(ops) > 8 || origin.Origin != "memory_writer" && origin.Origin != "reflection" && origin.Origin != "user_edit" {
		return plan, ErrDenied
	}
	lang := s.User.Locale
	lex, ok := a.lex(lang)
	if !ok {
		return plan, ErrAssets
	}
	items := map[c.ID]Record{}
	before := map[c.ID]Record{}
	for _, r := range s.Records {
		if r.Item.UserID != m.UserID || !c.ValidID(r.Item.ID) {
			return plan, ErrDenied
		}
		items[r.Item.ID] = cloneRecord(r)
		before[r.Item.ID] = cloneRecord(r)
	}
	ledger := map[string]Receipt{}
	for k, v := range s.Ledger {
		ledger[k] = v
	}
	for index, raw := range ops {
		proposalOrigin := origin
		if origin.Origin == "reflection" && m.SourceModels[Hash(raw)] != "" {
			proposalOrigin.Model = m.SourceModels[Hash(raw)]
		}
		op := raw
		op.EvidenceMsg = uniqueIDs(op.EvidenceMsg)
		hash := Hash(op)
		if receipt, exists := ledger[hash]; exists {
			plan.Report.Applied = append(plan.Report.Applied, receipt.Applied...)
			plan.Report.Rejected = append(plan.Report.Rejected, receipt.Rejected...)
			continue
		}
		receipt := Receipt{Hash: hash, Applied: []c.ID{}, Rejected: []c.DeltaRejection{}}
		reject := func(rule string) {
			receipt.Rejected = append(receipt.Rejected, c.DeltaRejection{Path: "ops[" + strconv.Itoa(index) + "]", RuleCode: rule})
		}
		apply := func(id c.ID) { receipt.Applied = append(receipt.Applied, id) }
		func() {
			if !op.Op.Valid() || len(op.EvidenceMsg) < 1 || len(op.EvidenceMsg) > 6 || op.ReasonTag == "" || len(op.ReasonTag) > 40 || !utf8.ValidString(op.Content) || len([]rune(op.Content)) > 320 || len(strings.Fields(op.Content)) > 40 || len(op.Entities) > 4 || len(op.Tags) > 6 {
				reject("MC-001")
				return
			}
			userEvidence := false
			for _, id := range op.EvidenceMsg {
				source, exists := s.Sources[id]
				if !exists || !c.ValidID(id) || source.Message.ID != id || (source.Message.Role != c.RoleUser && source.Message.Role != c.RoleAssistant) || source.Future || source.Hidden && !allowed(ctx, policy, s, m, UseRetainedSource, "message", id, &op, origin) || source.Message.UserID != m.UserID || !allowed(ctx, policy, s, m, UseSource, "message", id, &op, origin) {
					reject("MC-001")
					return
				}
				if source.Message.Role == c.RoleUser {
					userEvidence = true
				}
			}
			if op.Op == c.OpNoop {
				return
			}
			if m.Consolidate && op.Kind != "" && op.Kind != c.MemEpisode {
				reject("MC-001")
				return
			}
			var target Record
			if op.TargetID != nil {
				var exists bool
				target, exists = items[*op.TargetID]
				if !exists || target.DeletedAt != nil || target.Item.UserID != m.UserID || !allowed(ctx, policy, s, m, UseSource, "memory", *op.TargetID, &op, origin) {
					reject("MC-001")
					return
				}
			}
			if op.Op != c.OpAdd && op.TargetID == nil {
				reject("MC-001")
				return
			}
			if !allowed(ctx, policy, s, m, Store, "memory", target.Item.ID, &op, origin) {
				reject("MC-008")
				return
			}
			if op.TargetID != nil && (target.Item.Status == c.MemRejected || target.Item.Status == c.MemSuperseded) {
				reject("MC-010")
				return
			}
			if op.Op == c.OpConfirm || op.Op == c.OpContradict || op.Op == c.OpReject {
				if op.Op == c.OpReject && (!userEvidence || !allowed(ctx, policy, s, m, ExplicitChange, "memory", target.Item.ID, &op, origin)) {
					reject("MC-007")
					return
				}
				role := "supports"
				if op.Op == c.OpContradict || op.Op == c.OpReject {
					role = "contradicts"
				}
				attach(&target.Item, op.EvidenceMsg, s.Sources, role)
				if op.Op == c.OpReject {
					target.Item.Status = c.MemRejected
					cooldowns(a, &plan, target, now)
				} else if op.Op == c.OpContradict && target.Item.Kind != c.MemHypothesis {
					target.Item.Status = c.MemContested
					plan.Loops = append(plan.Loops, target.Item.ID)
				}
				counts(a, &target.Item, s.Sources, s.Events, now)
				touch(&target, now)
				items[target.Item.ID] = target
				apply(target.Item.ID)
				return
			}
			if target.Item.Source.Origin == "user_edit" && origin.Origin != "user_edit" {
				reject("MC-013")
				return
			}
			if s.User.MemoryPaused && origin.Origin != "user_edit" {
				reject("MC-008")
				return
			}
			if !allowed(ctx, policy, s, m, Store, "memory", target.Item.ID, &op, origin) {
				reject("MC-008")
				return
			}
			if op.Kind == "" {
				op.Kind = target.Item.Kind
			}
			if op.Epistemic == "" {
				op.Epistemic = target.Item.Epistemic
			}
			if op.Sensitivity == "" {
				op.Sensitivity = target.Item.Sensitivity
			}
			if op.Sensitivity == "" {
				op.Sensitivity = c.SensNormal
			}
			if !op.Kind.Valid() || op.Kind == c.MemSafetyEvt || !contains(a.config.AllowedKinds, string(op.Kind)) || !op.Sensitivity.Valid() || strings.TrimSpace(op.Content) == "" {
				reject("MC-001")
				return
			}
			if target.Item.Kind == c.MemHypothesis && op.Kind != c.MemHypothesis {
				if op.Op != c.OpAdd || origin.Origin != "user_edit" || !userEvidence || op.Kind != c.MemFact || op.Epistemic != c.EpReported || !allowed(ctx, policy, s, m, Endorse, "memory", target.Item.ID, &op, origin) {
					reject("MC-003")
					return
				}
			}
			if op.TargetID != nil && target.Item.Kind != c.MemHypothesis && ((op.Op == c.OpAdd) || op.Kind != target.Item.Kind) {
				reject("MC-001")
				return
			}
			if op.Kind == c.MemHypothesis {
				op.Epistemic = c.EpInferred
				hedged := false
				for _, h := range lex.hedges {
					if h.MatchString(op.Content) {
						hedged = true
					}
				}
				if !hedged {
					reject("MC-004")
					return
				}
				for _, label := range lex.labels {
					if strings.Contains(Canonical(op.Content), Canonical(label)) {
						reject("MC-005")
						return
					}
				}
				known := false
				for _, tag := range op.Tags {
					if contains(a.hypTags, tag) {
						known = true
						if until := s.Cooldowns[tag]; until.After(now) {
							reject("MC-010")
							return
						}
					}
				}
				if !known {
					reject("MC-010")
					return
				}
			}
			if op.Epistemic != c.EpReported && op.Epistemic != c.EpInferred {
				reject("MC-002")
				return
			}
			if op.Epistemic == c.EpReported && !userEvidence {
				if op.Kind == c.MemFact || op.Kind == c.MemPreference {
					reject("MC-002")
					return
				}
				op.Epistemic = c.EpInferred
			}
			if (op.Kind == c.MemFact || op.Kind == c.MemPreference) && op.Epistemic != c.EpReported {
				reject("MC-002")
				return
			}
			if op.Sensitivity == c.SensHighly && (!userEvidence || !allowed(ctx, policy, s, m, Sensitive, "memory", target.Item.ID, &op, origin)) {
				reject("MC-011")
				return
			}
			if s.Suppressed[ClaimHash(op.Content)] {
				reject("MC-009")
				return
			}
			for _, tag := range op.Tags {
				if !a.validTag(tag) {
					reject("MC-001")
					return
				}
			}
			for _, entity := range op.Entities {
				if entity.Role == "" || len([]rune(entity.Role)) > 30 || entity.Label != nil && len([]rune(*entity.Label)) > 40 {
					reject("MC-001")
					return
				}
			}
			if op.Importance != nil && (math.IsNaN(float64(*op.Importance)) || *op.Importance < 0 || *op.Importance > 1) {
				reject("MC-001")
				return
			}
			var validFrom *time.Time
			if op.ValidFrom != nil {
				t, e := time.Parse("2006-01-02", *op.ValidFrom)
				if e != nil {
					reject("MC-001")
					return
				}
				validFrom = &t
			}
			// Confirm duplicates without changing content, epistemics, sensitivity
			// or user-edited source authority. Stable ordering removes map races.
			if op.Op == c.OpAdd && op.TargetID == nil {
				// A changed value of an identified attribute must not disappear
				// into a high cosine score (e.g. two different preferred names).
				attribute := a.attribute(op.Tags)
				if attribute != "" && (op.Kind == c.MemFact || op.Kind == c.MemPreference) {
					for _, id := range sortedItems(items) {
						r := items[id]
						if r.DeletedAt == nil && r.Item.Status == c.MemActive && r.Item.Kind == op.Kind && entitiesEqual(r.Item.Entities, op.Entities) && a.attribute(r.Item.Tags) == attribute && Canonical(r.Item.Content) != Canonical(op.Content) {
							op.TargetID = &id
							target = r
							op.Op = c.OpSupersede
							break
						}
					}
				}
				for _, id := range sortedItems(items) {
					if op.Op != c.OpAdd {
						break
					}
					r := items[id]
					if r.DeletedAt != nil || !active(r.Item.Status) || r.Item.Kind != op.Kind || r.Item.Epistemic != op.Epistemic || r.Item.Sensitivity != op.Sensitivity || !entitiesEqual(r.Item.Entities, op.Entities) {
						continue
					}
					if equivalent(op.Content, r.Item.Content, vectors, a.thresholds.Memory.DedupeCosineGTE) {
						attach(&r.Item, op.EvidenceMsg, s.Sources, "supports")
						counts(a, &r.Item, s.Sources, s.Events, now)
						touch(&r, now)
						items[id] = r
						apply(id)
						return
					}
				}
				attribute = a.attribute(op.Tags)
				if attribute != "" && (op.Kind == c.MemFact || op.Kind == c.MemPreference) {
					for _, id := range sortedItems(items) {
						r := items[id]
						if r.DeletedAt == nil && r.Item.Status == c.MemActive && r.Item.Kind == op.Kind && entitiesEqual(r.Item.Entities, op.Entities) && a.attribute(r.Item.Tags) == attribute {
							op.TargetID = &id
							target = r
							op.Op = c.OpSupersede
							break
						}
					}
				}
			}
			if target.Item.Source.Origin == "user_edit" && origin.Origin != "user_edit" {
				reject("MC-013")
				return
			}
			if op.TargetID != nil && sensitivityRank(target.Item.Sensitivity) > sensitivityRank(op.Sensitivity) {
				reject("MC-011")
				return
			}
			importance := float32(.3)
			if op.Importance != nil {
				importance = *op.Importance
			}
			r := Record{Item: c.MemoryItem{ID: c.NewID(), UserID: m.UserID, Kind: op.Kind, Content: op.Content, Lang: lang, Epistemic: op.Epistemic, Status: c.MemActive, Confidence: c.ConfLow, Entities: op.Entities, Tags: op.Tags, Sensitivity: op.Sensitivity, Importance: importance, CreatedAt: now, UpdatedAt: now, ValidFrom: validFrom, Source: proposalOrigin}, Version: 1}
			r.Item.ThreadIDs = append([]c.ID(nil), target.Item.ThreadIDs...)
			if s.ThreadID != nil {
				r.Item.ThreadIDs = uniqueIDs(append(r.Item.ThreadIDs, *s.ThreadID))
			}
			if r.Item.Kind == c.MemHypothesis {
				r.Item.Status = c.MemProposed
			}
			if origin.Origin == "user_edit" {
				r.Pinned = true
			}
			evidenceRole := "supports"
			if origin.Origin == "user_edit" {
				evidenceRole = "context"
			}
			attach(&r.Item, op.EvidenceMsg, s.Sources, evidenceRole)
			if origin.Origin == "user_edit" {
				r.Item.Evidence = append(r.Item.Evidence, c.EvidenceRef{Kind: "event", Ref: m.OperationID, Role: "supports", At: now})
			}
			if op.Op == c.OpAdd && target.Item.Kind == c.MemHypothesis {
				r.Item.Evidence = append(r.Item.Evidence, c.EvidenceRef{Kind: "item", Ref: target.Item.ID, Role: "context", At: now})
			}
			if op.Op == c.OpUpdate || op.Op == c.OpSupersede {
				if origin.Origin == "memory_writer" && target.Item.LastConfirmedAt != nil {
					latest := time.Time{}
					for _, id := range op.EvidenceMsg {
						source := s.Sources[id]
						if source.Message.Role == c.RoleUser && source.Message.CreatedAt.After(latest) {
							latest = source.Message.CreatedAt
						}
					}
					if latest.Before(*target.Item.LastConfirmedAt) {
						reject("MC-007")
						return
					}
				}
				if !userEvidence {
					reject("MC-007")
					return
				}
				if allowed(ctx, policy, s, m, ExplicitChange, "memory", target.Item.ID, &op, origin) {
					target.Item.Status = c.MemSuperseded
					target.Item.ValidTo = &now
					target.Item.SupersededBy = &r.Item.ID
					r.Item.Supersedes = &target.Item.ID
				} else {
					target.Item.Status = c.MemContested
					r.Item.Status = c.MemContested
					plan.Loops = append(plan.Loops, target.Item.ID, r.Item.ID)
				}
				r.Item.Evidence = append(r.Item.Evidence, c.EvidenceRef{Kind: "item", Ref: target.Item.ID, Role: "context", At: now})
				touch(&target, now)
				items[target.Item.ID] = target
			}
			events := s.Events
			if origin.Origin == "user_edit" {
				events = map[c.ID]EventSource{}
				for id, event := range s.Events {
					events[id] = event
				}
				events[m.OperationID] = EventSource{Actor: "user", Type: "explicit_confirmation", At: now}
			}
			counts(a, &r.Item, s.Sources, events, now)
			if r.Item.Kind == c.MemHypothesis {
				r.Item.Status = c.MemProposed
			}
			items[r.Item.ID] = r
			apply(r.Item.ID)
		}()
		ledger[hash] = receipt
		plan.Receipts = append(plan.Receipts, receipt)
		plan.Report.Applied = append(plan.Report.Applied, receipt.Applied...)
		plan.Report.Rejected = append(plan.Report.Rejected, receipt.Rejected...)
	}
	capHot(a, items, now)
	for _, id := range sortedItems(items) {
		r := items[id]
		old, exists := before[id]
		if exists && reflect.DeepEqual(old, r) {
			continue
		}
		var prev *Record
		if exists {
			x := old
			prev = &x
		}
		if v, ok := vectors[ClaimHash(r.Item.Content)]; ok {
			r.Vector = append([]float32(nil), v...)
		}
		plan.Changes = append(plan.Changes, Mutation{Before: prev, After: r})
	}
	plan.Report.Applied = uniqueIDs(plan.Report.Applied)
	plan.Loops = uniqueIDs(plan.Loops)
	return plan, nil
}
func sensitivityRank(s c.Sensitivity) int {
	switch s {
	case c.SensHighly:
		return 3
	case c.SensSensitive:
		return 2
	case c.SensNormal:
		return 1
	}
	return 0
}
func cloneRecord(r Record) Record {
	raw, _ := json.Marshal(r)
	var out Record
	_ = json.Unmarshal(raw, &out)
	return out
}
func active(s c.MemoryStatus) bool {
	return s == c.MemActive || s == c.MemProposed || s == c.MemSupported
}
func touch(r *Record, now time.Time) { r.Version++; r.Item.UpdatedAt = now }
func sortedItems(items map[c.ID]Record) []c.ID {
	ids := []c.ID{}
	for id := range items {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	return ids
}
func entitiesEqual(a, b []c.EntityRef) bool {
	canonical := func(xs []c.EntityRef) []string {
		out := []string{}
		for _, x := range xs {
			v := Canonical(x.Role) + ":"
			if x.Label != nil {
				v += Canonical(*x.Label)
			}
			out = append(out, v)
		}
		sort.Strings(out)
		return out
	}
	return reflect.DeepEqual(canonical(a), canonical(b))
}
func equivalent(a, b string, vectors map[string][]float32, threshold float64) bool {
	if Canonical(a) == Canonical(b) {
		return true
	}
	x, y := vectors[ClaimHash(a)], vectors[ClaimHash(b)]
	if len(x) == 0 || len(x) != len(y) {
		return false
	}
	dot, xx, yy := 0., 0., 0.
	for i := range x {
		dot += float64(x[i]) * float64(y[i])
		xx += float64(x[i]) * float64(x[i])
		yy += float64(y[i]) * float64(y[i])
	}
	return xx > 0 && yy > 0 && dot/math.Sqrt(xx*yy) >= threshold
}
func attach(item *c.MemoryItem, ids []c.ID, sources map[c.ID]Source, role string) {
	for _, id := range ids {
		source := sources[id]
		actual := role
		if source.Message.Role != c.RoleUser {
			actual = "context"
		}
		found := false
		for _, e := range item.Evidence {
			if e.Kind == "message" && e.Ref == id && e.Role == actual {
				found = true
			}
		}
		if !found {
			item.Evidence = append(item.Evidence, c.EvidenceRef{Kind: "message", Ref: id, Role: actual, At: source.Message.CreatedAt})
		}
	}
}
func counts(a *Assets, item *c.MemoryItem, sources map[c.ID]Source, events map[c.ID]EventSource, now time.Time) {
	support, contradict := map[c.ID]bool{}, map[c.ID]bool{}
	sessions := map[c.ID]bool{}
	for _, e := range item.Evidence {
		if e.Kind == "event" {
			event, ok := events[e.Ref]
			if ok && event.Actor == "user" {
				if e.Role == "supports" && event.Type == "explicit_confirmation" {
					support[e.Ref] = true
					if c.ValidID(event.SessionID) {
						sessions[event.SessionID] = true
					}
				}
				if e.Role == "contradicts" && event.Type == "memory_corrected" {
					contradict[e.Ref] = true
				}
			}
			continue
		}
		if e.Kind != "message" {
			continue
		}
		source, ok := sources[e.Ref]
		if !ok || source.Message.Role != c.RoleUser {
			continue
		}
		if e.Role == "supports" {
			support[e.Ref] = true
			if c.ValidID(source.SessionID) {
				sessions[source.SessionID] = true
			}
		}
		if e.Role == "contradicts" {
			contradict[e.Ref] = true
		}
	}
	item.SupportN = len(support)
	item.ContradictN = len(contradict)
	item.SessionsN = len(sessions)
	item.Confidence = c.ConfLow
	if item.SupportN >= a.thresholds.Hypothesis.SupportedMinSupport && item.SessionsN >= a.thresholds.Hypothesis.SupportedMinSessions && item.ContradictN == 0 {
		item.Confidence = c.ConfHigh
	} else if item.SupportN >= 2 && item.ContradictN == 0 {
		item.Confidence = c.ConfMedium
	}
	if len(support) > 0 {
		latest := time.Time{}
		for _, e := range item.Evidence {
			if e.Role == "supports" && e.At.After(latest) {
				latest = e.At
			}
		}
		item.LastConfirmedAt = &latest
	}
	if item.Kind == c.MemHypothesis && (item.Status == c.MemProposed || item.Status == c.MemSupported || item.Status == c.MemContested) {
		t := a.thresholds.Hypothesis
		if item.ContradictN >= t.ContestedWhen.ContradictGTE && (!t.ContestedWhen.AndContradictGTESupport || item.ContradictN >= item.SupportN) {
			item.Status = c.MemContested
		} else if item.Confidence == c.ConfHigh {
			item.Status = c.MemSupported
		}
	}
}
func cooldowns(a *Assets, plan *Plan, r Record, now time.Time) {
	if r.Item.Kind != c.MemHypothesis {
		return
	}
	for _, tag := range r.Item.Tags {
		if contains(a.hypTags, tag) {
			plan.Cooldowns = append(plan.Cooldowns, Cooldown{Tag: tag, Item: r.Item.ID, Until: now.AddDate(0, 0, a.thresholds.Hypothesis.RejectionCooldownDays)})
		}
	}
}
func capHot(a *Assets, items map[c.ID]Record, now time.Time) {
	ids := []c.ID{}
	for id, r := range items {
		if r.DeletedAt == nil && active(r.Item.Status) && !r.Pinned && r.Item.Source.Origin != "user_edit" {
			ids = append(ids, id)
		}
	}
	if len(ids) <= a.thresholds.Memory.HotSetMaxItems {
		return
	}
	sort.Slice(ids, func(i, j int) bool {
		score := func(id c.ID) float64 {
			r := items[id]
			days := math.Max(0, now.Sub(r.Item.UpdatedAt).Hours()/24)
			return float64(r.Item.Importance) / (1 + days)
		}
		if score(ids[i]) == score(ids[j]) {
			return ids[i].String() < ids[j].String()
		}
		return score(ids[i]) < score(ids[j])
	})
	for _, id := range ids[:len(ids)-a.thresholds.Memory.HotSetMaxItems] {
		r := items[id]
		r.Item.Status = c.MemStale
		touch(&r, now)
		items[id] = r
	}
}
