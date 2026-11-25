package retrieval

import (
	"context"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"strings"
)

// Expand at most two selected derived sources, one hop, with at most one raw
// evidence excerpt and one related state record per source. Replaced lower-rank
// hits are restored when no permitted relationship can fill the reserved room.
func (s *Service) expand(ctx context.Context, q Query, out c.RetrievalResult, limit, total int) c.RetrievalResult {
	rel, ok := s.store.(Relationships)
	if !ok {
		return out
	}
	originals := append([]c.MemoryView{}, out.Items...)
	expanded := []c.MemoryView{}
	seen := map[c.ID]bool{}
	for _, v := range originals {
		seen[v.ID] = true
	}
	for i := 0; i < min(2, len(originals)); i++ {
		m := &originals[i]
		if m.Source != string(SourceMemory) {
			continue
		}
		parent := Candidate{ID: m.ID, UserID: q.UserID, Source: SourceMemory}
		refs, thread, err := rel.Evidence(ctx, q, parent)
		if err != nil {
			out.Meta.Degraded = true
			out.Meta.DegradedReason = "evidence_expansion_failed"
			out.Meta.FailedStrategies = append(out.Meta.FailedStrategies, "evidence")
			break
		}
		m.Evidence = refs
		m.ThreadID = thread
		parent.ThreadID = thread
		for _, ref := range refs {
			if seen[ref.Ref] {
				continue
			}
			kind := SourceMessage
			if ref.Kind == "item" {
				kind = SourceMemory
			} else if ref.Kind != "message" {
				continue
			}
			d, found, e := rel.Lookup(ctx, q, kind, ref.Ref)
			if e != nil {
				out.Meta.Degraded = true
				out.Meta.DegradedReason = "evidence_expansion_failed"
				continue
			}
			if !found || d.UserID != q.UserID || d.Sensitivity == c.SensHighly {
				continue
			}
			v := view(d, s.now())
			id := m.ID
			v.ExpandedFrom = &id
			v.Evidence = []c.EvidenceRef{ref}
			v.Text = truncate(v.Text, 90)
			expanded = append(expanded, v)
			seen[v.ID] = true
			break
		}
		if related, ok := s.store.(interface {
			Related(context.Context, Query, Candidate) ([]Candidate, error)
		}); ok && len(expanded) < 2 {
			docs, e := related.Related(ctx, q, parent)
			if e != nil {
				out.Meta.Degraded = true
				out.Meta.DegradedReason = "relationship_expansion_failed"
			}
			for _, d := range docs {
				if seen[d.ID] || d.UserID != q.UserID || d.Sensitivity == c.SensHighly {
					continue
				}
				v := view(d, s.now())
				id := m.ID
				v.ExpandedFrom = &id
				v.Text = truncate(v.Text, 90)
				expanded = append(expanded, v)
				seen[v.ID] = true
				if len(expanded) >= 2 {
					break
				}
			}
		}
		if len(expanded) >= 2 {
			break
		}
	}
	out.Items = nil
	seen = map[c.ID]bool{}
	used := 0
	base := max(1, limit-min(2, len(expanded)))
	appendView := func(v c.MemoryView) {
		if seen[v.ID] || len(out.Items) >= limit || used >= total {
			return
		}
		v.Text = truncate(v.Text, total-used)
		if strings.TrimSpace(v.Text) == "" {
			return
		}
		used += approximateTokens(v.Text)
		seen[v.ID] = true
		out.Items = append(out.Items, v)
	}
	for _, v := range originals[:min(base, len(originals))] {
		appendView(v)
	}
	for _, v := range expanded {
		appendView(v)
	}
	for _, v := range originals {
		appendView(v)
	}
	return out
}
