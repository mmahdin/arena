package retrieval

import (
	"context"
	"errors"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"time"
)

func temporalCandidates(docs []Candidate, refs []c.TemporalRef) []Candidate {
	if len(refs) == 0 {
		return docs
	}
	out := []Candidate{}
	for _, d := range docs {
		at := d.CreatedAt
		if d.ValidFrom != nil {
			at = *d.ValidFrom
		}
		for _, ref := range refs {
			if !at.Before(ref.From) && at.Before(ref.To) {
				out = append(out, d)
				break
			}
		}
	}
	return out
}

func view(item Candidate, now time.Time) c.MemoryView {
	v := c.MemoryView{ID: item.ID, Source: string(item.Source), SourceVersion: item.SourceVersion, SourceCreatedAt: item.CreatedAt.UTC().Format(time.RFC3339Nano), SourceConversationID: item.ConversationID, Kind: item.Kind, Text: item.Text, Epistemic: item.Epistemic, Status: item.Status, Sensitivity: item.Sensitivity, EvidenceN: item.EvidenceN, Verbatim: item.Source == SourceMessage, Age: age(now, item.CreatedAt), ThreadID: item.ThreadID, Evidence: item.Evidence}
	if item.ValidFrom != nil {
		at := item.ValidFrom.Format("2006-01-02")
		v.ValidAt = &at
	}
	return v
}
func (s *Service) Persistent(ctx context.Context, user, trigger c.ID) (result c.PersistentContext, resultErr error) {
	ctx, span := trace.Start(ctx, "retrieval.Persistent", func() any { return map[string]any{"user": user, "trigger": trigger} })
	defer func() { span.Complete(func() any { return result }, resultErr, recover()) }()
	out := c.PersistentContext{Items: []c.MemoryView{}}
	rel, ok := s.store.(Relationships)
	if !ok {
		out.Degraded = true
		out.Reason = "baseline_unavailable"
		return out, nil
	}
	// Search authorization is also checked by the production adapter's baseline.
	docs, err := rel.Baseline(ctx, user, trigger)
	if err != nil {
		out.Degraded = true
		out.Reason = "baseline_read_failed"
		return out, err
	}
	for _, d := range docs {
		if d.UserID != user {
			out.Degraded = true
			out.Reason = "baseline_owner_mismatch"
			out.Items = nil
			return out, errors.New("baseline owner mismatch")
		}
		out.Items = append(out.Items, view(d, s.now()))
	}
	return out, nil
}
