package memory

import (
	"context"
	"reflect"
	"time"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

func (a *Assets) maintain(ctx context.Context, s Snapshot, m Meta, p Policy, vectors map[string][]float32, now time.Time) (Plan, ports.MaintenanceReport) {
	plan := Plan{}
	report := ports.MaintenanceReport{}
	items := map[c.ID]Record{}
	before := map[c.ID]Record{}
	for _, r := range s.Records {
		items[r.Item.ID] = cloneRecord(r)
		before[r.Item.ID] = cloneRecord(r)
	}
	ids := sortedItems(items)
	// Merge compatible duplicates/episode clusters by uniting evidence. Keep
	// both original texts and links; no model-generated psychological summary.
	for i, id := range ids {
		r := items[id]
		if r.DeletedAt != nil || !active(r.Item.Status) || r.Pinned || r.Item.Source.Origin == "user_edit" || !allowed(ctx, p, s, m, UseSource, "memory", id, nil, c.ProvenanceSrc{Origin: "memory_writer"}) {
			continue
		}
		for _, other := range ids[i+1:] {
			n := items[other]
			if n.DeletedAt != nil || !active(n.Item.Status) || n.Pinned || n.Item.Source.Origin == "user_edit" || r.Item.Kind != n.Item.Kind || r.Item.Epistemic != n.Item.Epistemic || r.Item.Sensitivity != n.Item.Sensitivity || !entitiesEqual(r.Item.Entities, n.Item.Entities) || !allowed(ctx, p, s, m, UseSource, "memory", other, nil, c.ProvenanceSrc{Origin: "memory_writer"}) || !equivalent(r.Item.Content, n.Item.Content, vectors, a.thresholds.Memory.DedupeCosineGTE) {
				continue
			}
			for _, ev := range n.Item.Evidence {
				if !contains(r.Item.Evidence, ev) {
					r.Item.Evidence = append(r.Item.Evidence, ev)
				}
			}
			r.Item.Evidence = append(r.Item.Evidence, c.EvidenceRef{Kind: "item", Ref: other, Role: "context", At: now})
			r.Item.ThreadIDs = uniqueIDs(append(r.Item.ThreadIDs, n.Item.ThreadIDs...))
			r.Item.ClusterID = &id
			n.Item.ClusterID = &id
			n.Item.Status = c.MemSuperseded
			n.Item.SupersededBy = &id
			n.Item.ValidTo = &now
			touch(&n, now)
			items[other] = n
			touch(&r, now)
			counts(a, &r.Item, s.Sources, s.Events, now)
			report.Merged++
		}
		items[id] = r
	}
	for id, r := range items {
		if r.DeletedAt != nil || !active(r.Item.Status) || r.Pinned || r.Item.Source.Origin == "user_edit" {
			continue
		}
		last := r.Item.CreatedAt
		if r.Item.LastConfirmedAt != nil && r.Item.LastConfirmedAt.After(last) {
			last = *r.Item.LastConfirmedAt
		}
		if r.Item.LastUsedAt != nil && r.Item.LastUsedAt.After(last) {
			last = *r.Item.LastUsedAt
		}
		days := a.thresholds.Memory.StaleAfterDays
		low := float64(r.Item.Importance) < a.thresholds.Memory.StaleImportanceLT
		if r.Item.Kind == c.MemHypothesis {
			days = a.thresholds.Hypothesis.StaleAfterDays
			low = true
		}
		if low && now.Sub(last) > time.Duration(days)*24*time.Hour {
			r.Item.Status = c.MemStale
			touch(&r, now)
			items[id] = r
		}
	}
	capHot(a, items, now)
	for _, id := range sortedItems(items) {
		r := items[id]
		old := before[id]
		if !reflect.DeepEqual(r, old) {
			if r.Item.Status == c.MemStale && old.Item.Status != c.MemStale {
				report.Staled++
			}
			prev := old
			plan.Changes = append(plan.Changes, Mutation{Before: &prev, After: r})
		}
	}
	return plan, report
}
