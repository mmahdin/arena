package retrieval

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
)

// Research executes an explicit, bounded request after its turn is finalized.
// It uses the same source policies and trigger fence as ordinary retrieval.
func (s *Service) PrepareResearch(ctx context.Context, user, turn, trigger c.ID, request c.BackgroundRequest) (out ports.BackgroundResearch, err error) {
	ctx, span := trace.Start(ctx, "background.PrepareResearch", func() any { return request })
	defer func() { span.Complete(func() any { return out }, err, recover()) }()
	if len(request.Queries) < 1 || len(request.Queries) > 3 {
		return out, errors.New("invalid background query count")
	}
	out.Request = request
	seen := map[string]bool{}
	for _, query := range request.Queries {
		if len(query) == 0 || len(query) > 1200 {
			return out, errors.New("invalid background query budget")
		}
		result, e := s.Retrieve(ctx, c.RetrievalRequest{UserID: user, Turn: c.TurnView{TurnID: turn, MessageID: trigger, Text: query}, Background: true})
		if e != nil && ctx.Err() != nil {
			return out, e
		}
		out.Meta = append(out.Meta, result.Meta)
		for _, item := range result.Items {
			key := item.Source + ":" + item.ID.String() + ":" + item.SourceVersion
			if !seen[key] {
				seen[key] = true
				out.Items = append(out.Items, item)
			}
		}
	}
	sort.SliceStable(out.Items, func(i, j int) bool { return out.Items[i].Score > out.Items[j].Score })
	if len(out.Items) > 6 {
		out.Items = out.Items[:6]
	}
	for {
		raw, _ := json.Marshal(out)
		if len(raw) <= 3600 || len(out.Items) == 0 {
			break
		}
		out.Items = out.Items[:len(out.Items)-1]
	}
	err = s.VerifyResearch(ctx, user, trigger, out)
	return out, err
}
func (s *Service) VerifyResearch(ctx context.Context, user, trigger c.ID, in ports.BackgroundResearch) (err error) {
	ctx, span := trace.Start(ctx, "background.VerifyResearch", func() any { return in })
	defer func() { span.Complete(nil, err, recover()) }()
	repo, ok := s.store.(Relationships)
	if !ok {
		return errors.New("research requires versioned sources")
	}
	for _, item := range in.Items {
		actual, found, err := repo.Lookup(ctx, Query{UserID: user, TriggerID: trigger}, SourceKind(item.Source), item.ID)
		if err != nil {
			return err
		}
		if !found || actual.UserID != user || actual.SourceVersion != item.SourceVersion || actual.Kind != item.Kind || actual.Epistemic != item.Epistemic || actual.Status != item.Status || !strings.HasPrefix(actual.Text, strings.TrimSuffix(item.Text, "…")) {
			return errors.New("research source changed or denied")
		}
	}
	return nil
}
