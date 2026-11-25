package retrieval

import (
	"context"
	"time"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/telemetry"
)

// RAG instrumentation describes only *authorized* candidates after the scoped scan and policy checks.
// In particular, it does not export the raw embedding vectors, user UUID,
// excluded document text or policy-denied records. The source version and
// content used for ranking are retained for debugging synthetic test cases.
func recordCandidates(ctx context.Context, docs []Candidate) {
	if !telemetry.ContentEnabled(ctx) {
		return
	}
	items := make([]map[string]any, 0, len(docs))
	for _, item := range docs {
		items = append(items, map[string]any{
			"id":                  item.ID.String(),
			"source":              item.Source,
			"source_version":      item.SourceVersion,
			"conversation_id":     item.ConversationID,
			"kind":                item.Kind,
			"status":              item.Status,
			"sensitivity":         item.Sensitivity,
			"epistemic":           item.Epistemic,
			"text":                item.Text,
			"evidence_count":      item.EvidenceN,
			"created_at":          item.CreatedAt.UTC().Format(time.RFC3339Nano),
			"embedding_available": len(item.Vector) > 0,
			"embedding_dimension": len(item.Vector),
		})
	}
	telemetry.RAGEvent(ctx, "authorized_candidates", map[string]any{
		"count": len(docs), "items": items,
	})
}

type ragRanking struct {
	name string
	hits []ranked
}

func recordRankings(ctx context.Context, docs []Candidate, rankings []ragRanking, fused []ranked) {
	if !telemetry.ContentEnabled(ctx) {
		return
	}
	// Report only branches that actually voted. Full lists are bounded by the
	// event size limit, independent of the offline Inspector's Top-5 preview.
	positions := make(map[int]map[string]int, len(fused))
	for _, ranking := range rankings {
		telemetry.RAGEvent(ctx, ranking.name+"_ranking", fullRankings(docs, ranking.hits))
		for index, hit := range ranking.hits {
			if positions[hit.index] == nil {
				positions[hit.index] = make(map[string]int, len(rankings))
			}
			positions[hit.index][ranking.name] = index + 1
		}
	}
	rows := make([]map[string]any, 0, len(fused))
	for position, hit := range fused {
		rows = append(rows, map[string]any{
			"id": docs[hit.index].ID.String(), "source": docs[hit.index].Source,
			"rank": position + 1, "rrf_score": hit.score, "branch_ranks": positions[hit.index],
		})
	}
	telemetry.RAGEvent(ctx, "rrf_ranking", map[string]any{
		"method": "rrf", "k": 60, "ranked_count": len(rows), "items": rows,
	})
}

func rankDecision(item Candidate, position int, hit ranked, reason string) map[string]any {
	return map[string]any{
		"id": item.ID.String(), "source": item.Source,
		"rank": position + 1, "rrf_score": hit.score,
		"reason": reason, "selected": reason == "selected",
	}
}

func fullRankings(docs []Candidate, hits []ranked) map[string]any {
	rows := make([]map[string]any, 0, len(hits))
	for i, hit := range hits {
		item := docs[hit.index]
		rows = append(rows, map[string]any{
			"source": item.Source, "id": item.ID.String(), "rank": i + 1, "score": hit.score,
		})
	}
	return map[string]any{"count": len(hits), "items": rows}
}

func recordSelectedContext(ctx context.Context, out c.RetrievalResult) {
	if !telemetry.ContentEnabled(ctx) {
		return
	}
	// "selected_context" is the result returned by Retrieval; it is NOT a
	// claim that every item was later placed into a final JEV or Unified prompt.
	telemetry.RAGEvent(ctx, "selected_context", map[string]any{
		"count": len(out.Items), "items": out.Items,
		"degraded": out.Meta.Degraded, "degraded_reason": out.Meta.DegradedReason,
	})
}
