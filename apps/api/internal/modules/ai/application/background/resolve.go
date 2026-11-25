package background

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"sort"
	"strings"
	"time"
)

func Clone[T any](in T) T {
	var out T
	raw, _ := json.Marshal(in)
	_ = json.Unmarshal(raw, &out)
	return out
}
func ClaimID(user c.ID, kind c.MemoryKind, content string, from *time.Time) c.ID {
	return uuid.NewSHA1(user, []byte("intelligence-memory:"+string(kind)+":"+mem.Canonical(content)+":"+mem.Hash(from)))
}

// Normalize only equivalent labels for supplied knowledge and reported memory.
// Evidence, ownership, scope and authored framework requirements are still checked
// by resolve; normalization cannot authorize a new source or strengthen a claim.
func normalizeOutput(out *Output, in Input) {
	kinds := map[c.ID]c.MemoryKind{}
	for _, record := range in.Memories {
		kinds[record.Item.ID] = record.Item.Kind
	}
	for i := range out.Operations {
		op := &out.Operations[i]
		kind := op.Kind
		if op.TargetID != nil && kinds[*op.TargetID] != "" {
			kind = kinds[*op.TargetID]
		}
		if kind.Valid() && kind != c.MemHypothesis && (op.Status == c.MemProposed || op.Status == c.MemSupported) {
			op.Status = c.MemActive
		}
		if kind == c.MemHypothesis && op.Status == c.MemActive {
			op.Status = c.MemProposed
		}
	}
	supplied := map[string]bool{}
	for _, resource := range in.Knowledge {
		supplied[resource.ID] = true
	}
	for i, id := range out.KnowledgeIDs {
		if !supplied[id] && supplied["framework:"+id] {
			out.KnowledgeIDs[i] = "framework:" + id
		}
	}
	for i := range out.Formulations {
		framework := out.Formulations[i].Doc.Framework
		if strings.HasPrefix(framework, "framework:") && supplied[framework] {
			out.Formulations[i].Doc.Framework = strings.TrimPrefix(framework, "framework:")
		}
	}
}

func (s *Service) resolve(ctx context.Context, l mem.Lease, p Prepared, out Output, usage c.LLMUsage) (plan mem.Plan, hints []Hint, err error) {
	ctx, span := trace.Start(ctx, "background.Resolve", func() any {
		return map[string]any{"lease": l, "prepared": p, "output": out, "usage": usage}
	})
	defer func() {
		span.Complete(func() any { return map[string]any{"plan": plan, "hints": hints} }, err, recover())
	}()
	plan = mem.Plan{Changes: []mem.Mutation{}, Report: ports.MemoryApplyReport{Applied: []c.ID{}, Rejected: []c.DeltaRejection{}}}
	if out.Schema != OutputSchema || len(out.Operations) > 16 || usage.Model == "" {
		return plan, nil, mem.ErrDenied
	}
	input := p.Input
	raw := map[c.ID]c.Message{}
	for _, m := range input.Messages {
		if m.UserID == l.UserID && m.Role == c.RoleUser {
			raw[m.ID] = m
		}
	}
	records := map[c.ID]mem.Record{}
	for _, r := range input.Memories {
		records[r.Item.ID] = r
	}
	seen := map[string]bool{}
	touched := map[c.ID]bool{}
	now := time.Now().UTC()
	origin := c.ProvenanceSrc{Origin: "memory_writer", Model: usage.Model, PromptVersion: s.Reference().Version}
	if input.Request.Kind == "reflection" {
		origin.Origin = "reflection"
	}
	reject := func(i int, why string) {
		plan.Report.Rejected = append(plan.Report.Rejected, c.DeltaRejection{Path: fmt.Sprintf("operations.%d", i), RuleCode: why, Detail: "independent analytical operation rejected"})
	}
	for i, op := range out.Operations {
		if op.ID == "" || seen[op.ID] {
			reject(i, "BI-ID")
			continue
		}
		seen[op.ID] = true
		if op.Action == "noop" {
			continue
		}
		target, exists := mem.Record{}, false
		if op.TargetID != nil {
			target, exists = records[*op.TargetID]
		}
		if op.Action != "add" && !exists {
			reject(i, "BI-TARGET")
			continue
		}
		if exists && (target.Item.UserID != l.UserID || target.DeletedAt != nil || touched[target.Item.ID]) {
			reject(i, "BI-TARGET")
			continue
		}
		kind := op.Kind
		if exists {
			kind = target.Item.Kind
		}
		if !kind.Valid() || kind == c.MemSafetyEvt || input.Request.Kind == "reflection" && kind != c.MemHypothesis || input.Request.Kind != "reflection" && kind == c.MemHypothesis {
			reject(i, "BI-RESPONSIBILITY")
			continue
		}
		if op.Status != "" && (kind == c.MemHypothesis && op.Status != c.MemProposed && op.Status != c.MemSupported || kind != c.MemHypothesis && op.Status != c.MemActive) {
			reject(i, "BI-STATUS")
			continue
		}
		if kind == c.MemHypothesis && (op.Action == "add" || op.Action == "revise" || op.Action == "supersede" || op.Action == "consolidate") && strings.TrimSpace(op.Uncertainty) == "" {
			reject(i, "BI-UNCERTAINTY")
			continue
		}
		evidence := []c.EvidenceRef{}
		valid := len(op.Evidence) > 0
		support := false
		for _, ev := range op.Evidence {
			m, ok := raw[ev.ID]
			if !ok || ev.Role != "supports" && ev.Role != "contradicts" && ev.Role != "context" {
				valid = false
				break
			}
			if ev.Role == "supports" {
				support = true
			}
			evidence = append(evidence, c.EvidenceRef{Kind: "message", Ref: m.ID, Role: ev.Role, At: m.CreatedAt})
		}
		for _, id := range op.ContextIDs {
			r, ok := records[id]
			if !ok {
				valid = false
				break
			}
			evidence = append(evidence, c.EvidenceRef{Kind: "item", Ref: id, Role: "context", At: r.Item.UpdatedAt})
		}
		if !valid || (op.Action == "add" || op.Action == "confirm" || op.Action == "revise" || op.Action == "supersede" || op.Action == "consolidate") && !support {
			reject(i, "BI-PRIMARY-EVIDENCE")
			continue
		}
		contentAction := op.Action == "add" || op.Action == "revise" || op.Action == "supersede" || op.Action == "consolidate"
		if contentAction && (strings.TrimSpace(op.Content) == "" || len([]rune(op.Content)) > 600) {
			reject(i, "BI-CONTENT")
			continue
		}
		if contentAction && p.Snapshot.Suppressed[mem.ClaimHash(op.Content)] {
			reject(i, "BI-SUPPRESSED")
			continue
		}
		if kind == c.MemHypothesis && contentAction {
			for _, tag := range op.Tags {
				if until := p.Snapshot.Cooldowns[tag]; until.After(now) {
					valid = false
				}
			}
			if !valid {
				reject(i, "BI-COOLDOWN")
				continue
			}
		}
		perm := mem.Permission{Action: mem.Store, User: p.Snapshot.User, TurnID: l.TurnID, Origin: origin}
		if exists {
			perm.SourceID = target.Item.ID
			perm.Memory = &target.Item
		}
		allowed, e := s.Policy.Allow(ctx, perm)
		if e != nil {
			return plan, nil, e
		}
		if !allowed {
			reject(i, "BI-POLICY")
			continue
		}
		if op.Sensitivity == "" {
			op.Sensitivity = c.SensNormal
		}
		if !op.Sensitivity.Valid() {
			reject(i, "BI-SENSITIVITY")
			continue
		}
		if op.Sensitivity == c.SensHighly {
			perm.Action = mem.Sensitive
			allowed, e = s.Policy.Allow(ctx, perm)
			if e != nil {
				return plan, nil, e
			}
			if !allowed {
				reject(i, "BI-SENSITIVE")
				continue
			}
		}
		if op.ValidFrom != nil && op.ValidTo != nil && !op.ValidTo.After(*op.ValidFrom) {
			reject(i, "BI-TIME")
			continue
		}
		changes := []mem.Mutation{}
		next := Clone(target)
		create := op.Action == "add" || op.Action == "supersede" || op.Action == "consolidate"
		coalesce := -1
		if create {
			// Exact normalized identity is operational deduplication, not a
			// semantic similarity decision. It also recognizes legacy UUIDs.
			identities := []string{}
			byIdentity := map[string]mem.Record{}
			for id, record := range records {
				identities = append(identities, id.String())
				byIdentity[id.String()] = record
			}
			sort.Strings(identities)
			for _, identity := range identities {
				existing := byIdentity[identity]
				if existing.Item.Kind == kind && mem.Canonical(existing.Item.Content) == mem.Canonical(op.Content) && mem.Hash(existing.Item.ValidFrom) == mem.Hash(op.ValidFrom) && existing.DeletedAt == nil && sameClaimContext(existing.Item, op, input.Request.Scope) {
					target = existing
					exists = true
					create = false
					next = Clone(existing)
					if touched[existing.Item.ID] {
						if op.Action != "add" {
							valid = false
							break
						}
						for index, change := range plan.Changes {
							if change.After.Item.ID == existing.Item.ID {
								coalesce = index
								break
							}
						}
					} else {
						next.Version++
					}
					next.Item.UpdatedAt = now
					break
				}
			}
		}
		if !valid {
			reject(i, "BI-DUPLICATE")
			continue
		}
		if create {
			identity := ClaimID(l.UserID, kind, op.Content, op.ValidFrom)
			if kind == c.MemHypothesis || kind == c.MemEvent || kind == c.MemEpisode || len(op.Entities) > 0 {
				ids := []string{}
				for _, ev := range op.Evidence {
					if ev.Role == "supports" {
						ids = append(ids, ev.ID.String())
					}
				}
				sort.Strings(ids)
				threads := []string{}
				for _, id := range input.Request.Scope.ThreadIDs {
					threads = append(threads, id.String())
				}
				sort.Strings(threads)
				if kind != c.MemHypothesis && kind != c.MemEvent && kind != c.MemEpisode {
					ids = nil
					threads = nil
				}
				identity = uuid.NewSHA1(identity, []byte(mem.Hash(struct {
					Primary, Threads []string
					Entities         []c.EntityRef
				}{ids, threads, op.Entities})))
			}
			item := c.MemoryItem{ID: identity, UserID: l.UserID, Kind: kind, Content: op.Content, Lang: p.Snapshot.User.Locale, Epistemic: c.EpReported, Status: c.MemActive, Confidence: c.ConfLow, ThreadIDs: append([]c.ID{}, input.Request.Scope.ThreadIDs...), Evidence: []c.EvidenceRef{}, Entities: op.Entities, Tags: op.Tags, Sensitivity: op.Sensitivity, Importance: .3, ValidFrom: op.ValidFrom, ValidTo: op.ValidTo, CreatedAt: now, UpdatedAt: now, Source: origin}
			if kind == c.MemHypothesis {
				item.Epistemic = c.EpInferred
				item.Status = c.MemProposed
			}
			if existing, ok := records[item.ID]; ok && existing.Item.Kind == item.Kind && mem.Canonical(existing.Item.Content) == mem.Canonical(item.Content) {
				next = Clone(existing)
				create = false
			} else {
				next = mem.Record{Item: item, Version: 1}
			}
		} else {
			next.Version++
			next.Item.UpdatedAt = now
			next.Item.Source = origin
		}
		if op.Action == "revise" {
			next.Item.Content = op.Content
			next.Item.ValidFrom = op.ValidFrom
			next.Item.ValidTo = op.ValidTo
			if kind == c.MemHypothesis {
				next.Item.Status = c.MemProposed
				next.Item.Confidence = c.ConfLow
			}
		}
		if kind == c.MemHypothesis && op.Status != "" {
			next.Item.Status = op.Status
		}
		if op.Action == "contradict" {
			next.Item.Status = c.MemContested
		}
		if op.Action == "invalidate" {
			next.Item.Status = c.MemRejected
		}
		switch op.Action {
		case "add", "confirm", "revise", "contradict", "supersede", "consolidate", "invalidate":
		default:
			reject(i, "BI-ACTION")
			continue
		}
		for _, ev := range evidence {
			duplicate := false
			for _, old := range next.Item.Evidence {
				if old.Kind == ev.Kind && old.Ref == ev.Ref && old.Role == ev.Role {
					duplicate = true
					break
				}
			}
			if !duplicate {
				next.Item.Evidence = append(next.Item.Evidence, ev)
			}
		}
		supports, contradicts := map[c.ID]bool{}, map[c.ID]bool{}
		sessions := map[c.ID]bool{}
		for _, ev := range next.Item.Evidence {
			if ev.Kind != "message" {
				continue
			}
			if source, ok := p.Snapshot.Sources[ev.Ref]; !ok || source.Message.Role != c.RoleUser {
				continue
			}
			if ev.Role == "supports" {
				supports[ev.Ref] = true
			}
			if ev.Role == "contradicts" {
				contradicts[ev.Ref] = true
			}
			if src, ok := p.Snapshot.Sources[ev.Ref]; ok {
				sessions[src.SessionID] = true
			}
		}
		next.Item.SupportN = len(supports)
		next.Item.ContradictN = len(contradicts)
		next.Item.SessionsN = max(next.Item.SessionsN, len(sessions))
		// Counts describe actual distinct primary evidence, never psychological confidence.
		if kind == c.MemHypothesis {
			next.Item.Epistemic = c.EpInferred
			next.Item.Confidence = c.ConfLow
		}
		if !create && next.Item.ID != target.Item.ID {
			existing := records[next.Item.ID]
			next.Version = existing.Version + 1
			next.Item.UpdatedAt = now
			changes = append(changes, mem.Mutation{Before: &existing, After: next})
		} else if create {
			changes = append(changes, mem.Mutation{After: next})
		} else {
			before := Clone(target)
			changes = append(changes, mem.Mutation{Before: &before, After: next})
		}
		if op.Action == "supersede" || op.Action == "consolidate" {
			ids := []c.ID{target.Item.ID}
			ids = append(ids, op.MergeIDs...)
			if op.Action == "consolidate" && len(ids) < 2 {
				valid = false
			}
			mergeSeen := map[c.ID]bool{}
			for _, id := range ids {
				old, ok := records[id]
				if !ok || touched[id] || mergeSeen[id] || id == next.Item.ID || old.Item.Kind != kind {
					valid = false
					break
				}
				mergeSeen[id] = true
				after := Clone(old)
				after.Version++
				after.Item.Status = c.MemSuperseded
				after.Item.SupersededBy = &next.Item.ID
				until := now
				if op.ValidFrom != nil {
					until = *op.ValidFrom
				}
				after.Item.ValidTo = &until
				after.Item.UpdatedAt = now
				changes = append(changes, mem.Mutation{Before: &old, After: after})
			}
			if !valid {
				reject(i, "BI-CONSOLIDATE")
				continue
			}
			changes[0].After.Item.Supersedes = op.TargetID
		}
		if coalesce >= 0 {
			plan.Changes[coalesce].After = next
			records[next.Item.ID] = next
			continue
		}
		for _, change := range changes {
			touched[change.After.Item.ID] = true
			records[change.After.Item.ID] = change.After
			plan.Report.Applied = append(plan.Report.Applied, change.After.Item.ID)
		}
		plan.Changes = append(plan.Changes, changes...)
	}
	// Validate authored knowledge and primary provenance before accepting formulations/hints.
	known := map[string]bool{}
	for _, card := range s.Cards {
		known[card.ID] = true
	}
	selected := map[string]bool{}
	supplied := map[string]bool{}
	for _, resource := range input.Knowledge {
		supplied[resource.ID] = true
	}
	for _, id := range out.KnowledgeIDs {
		if !supplied[id] {
			return plan, nil, mem.ErrDenied
		}
		selected[id] = true
	}
	for _, form := range out.Formulations {
		if input.Request.Kind != "reflection" || !selected["framework:"+form.Doc.Framework] || len(form.Doc.Fields) == 0 {
			return plan, nil, mem.ErrDenied
		}
		if form.ThreadID != nil {
			ok := false
			for _, t := range input.Threads {
				if t.ID == *form.ThreadID {
					ok = true
				}
			}
			if !ok {
				return plan, nil, mem.ErrDenied
			}
		}
		framework, exists := s.Frameworks[form.Doc.Framework]
		if !exists || framework.Paragraph.MaxWords > 0 && len(strings.Fields(form.Doc.Paragraph)) > framework.Paragraph.MaxWords {
			return plan, nil, mem.ErrDenied
		}
		rules := map[string]FrameworkField{}
		for _, rule := range framework.Fields {
			rules[rule.ID] = rule
		}
		for name, field := range form.Doc.Fields {
			rule, exists := rules[name]
			if !exists {
				return plan, nil, mem.ErrDenied
			}
			if field.Status != c.EpUnknown {
				allowed := false
				for _, status := range rule.Allowed {
					if field.Status == status {
						allowed = true
					}
				}
				evidenceSet, sessions := map[c.ID]bool{}, map[c.ID]bool{}
				for _, id := range field.Evidence {
					evidenceSet[id] = true
					if source, ok := p.Snapshot.Sources[id]; ok && c.ValidID(source.SessionID) {
						sessions[source.SessionID] = true
					}
				}
				if !allowed || len(evidenceSet) < rule.Evidence || len(sessions) < rule.Sessions {
					return plan, nil, mem.ErrDenied
				}
			}

			if field.Status == c.EpUnknown {
				continue
			}
			if len(field.Evidence) == 0 {
				return plan, nil, mem.ErrDenied
			}
			for _, id := range field.Evidence {
				if _, ok := raw[id]; !ok {
					return plan, nil, mem.ErrDenied
				}
			}
		}
	}
	hints = []Hint{}
	for _, hint := range out.Hints {
		valid := hint.TTLTurns >= 1 && hint.TTLTurns <= 3 && len([]rune(hint.Topic)) <= 160 && len(hint.EvidenceIDs) > 0
		for _, id := range hint.ResourceIDs {
			if !known[id] {
				valid = false
			}
		}
		for _, id := range hint.MemoryIDs {
			if _, ok := records[id]; !ok {
				valid = false
			}
		}
		for _, id := range hint.EvidenceIDs {
			if _, ok := raw[id]; !ok {
				valid = false
			}
		}
		if valid {
			hints = append(hints, hint)
		}
	}
	return plan, hints, nil
}

// Identical wording is not enough to merge independent episodes or hypotheses.
// Their scope and primary lineage remain distinct unless the model targets an
// existing item explicitly. Simple reported facts may accumulate primary proof.
func sameClaimContext(item c.MemoryItem, op Operation, scope Scope) bool {
	if len(item.Entities) > 0 || len(op.Entities) > 0 {
		if mem.Hash(item.Entities) != mem.Hash(op.Entities) {
			return false
		}
	}
	if item.Kind != c.MemHypothesis && item.Kind != c.MemEvent && item.Kind != c.MemEpisode {
		return true
	}
	if len(item.ThreadIDs) != len(scope.ThreadIDs) {
		return false
	}
	for _, id := range item.ThreadIDs {
		found := false
		for _, allowed := range scope.ThreadIDs {
			if id == allowed {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	for _, old := range item.Evidence {
		if old.Kind != "message" || old.Role != "supports" {
			continue
		}
		for _, ev := range op.Evidence {
			if ev.Role == "supports" && ev.ID == old.Ref {
				return true
			}
		}
	}
	return false
}
