package retrieval

import (
	"context"
	"testing"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

func TestOTelFullRankingsDescribeEntireListsWithoutChangingRetrieval(t *testing.T) {
	s, _, req := setup(t)
	docs, target := sharedSixthCandidates(req.UserID)
	vector := []float32{1, 0, 0}
	semantic := vectorSearch(vector, docs)
	contextual := vectorSearch([]float32{0, 1, 0}, docs)
	combined := fuse(docs, semantic, contextual)

	// The local Inspector intentionally keeps its previous Top-5 preview,
	// whereas OBS-04B exports the complete authorized list (within the
	// explicitly reported payload size budget).
	got := fullRankings(docs, combined)
	items, ok := got["items"].([]map[string]any)
	if !ok || got["count"] != len(combined) || len(items) != len(combined) || len(combined) <= rankTracePreviewLimit {
		t.Fatalf("RRF trace was limited to the Inspector preview: %v", got)
	}
	for i, item := range items {
		if item["rank"] != i+1 || item["score"] != combined[i].score || item["id"] != docs[combined[i].index].ID.String() {
			t.Fatalf("OTel ranking misrepresented RRF at index %d: %v", i, item)
		}
	}
	if items[0]["id"] != target.String() {
		t.Fatal("the full RRF trace lost its shared-sixth winner")
	}
	out := s.rankWithHints(context.Background(), c.RetrievalResult{}, docs, "anchor", vector, 5, 60, 400,
		Hints{ContextQuery: "prior context", ContextVector: []float32{0, 1, 0}})
	if len(out.Items) != 5 {
		t.Fatalf("expected five selected items, got %d", len(out.Items))
	}
	for i, item := range out.Items {
		if item.ID != docs[combined[i].index].ID || item.Score != float32(combined[i].score) {
			t.Fatalf("selected context disagrees with dual-vector RRF at position %d", i)
		}
	}
	if len(traceRanks(semantic, docs)) > rankTracePreviewLimit || len(traceRanks(contextual, docs)) > rankTracePreviewLimit {
		t.Fatal("local Inspector preview unexpectedly changed")
	}
}

func TestOTelEmptyRankingIsExplicitNotInvented(t *testing.T) {
	rows := fullRankings(nil, nil)
	if rows["count"] != 0 || len(rows["items"].([]map[string]any)) != 0 {
		t.Fatal("no candidates should produce zero ranking observations")
	}
}
