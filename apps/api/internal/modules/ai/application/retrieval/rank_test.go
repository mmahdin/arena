package retrieval

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"strings"
	"testing"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

func rankTestID(n int) c.ID { return c.ID{6: 0x40, 8: 0x80, 14: byte(n >> 8), 15: byte(n)} }

// Each branch has five exclusive leaders. The shared sixth-ranked source must
// beat them after fusion: 2/66 > 1/61, despite missing both branch Top 5 lists.
func sharedSixthCandidates(owner c.ID) ([]Candidate, c.ID) {
	var docs []Candidate
	for i := 0; i < 5; i++ {
		item := memory(owner, rankTestID(i+1), fmt.Sprintf("semantic leader %d", i), 0)
		item.Vector = []float32{float32(10 - i), 0, 1}
		docs = append(docs, item)
	}
	for i := 0; i < 5; i++ {
		item := memory(owner, rankTestID(i+6), strings.Repeat("anchor ", i+3)+fmt.Sprintf("keyword%d", i), 0)
		item.Vector = []float32{0, float32(10 - i), 1}
		docs = append(docs, item)
	}
	target := memory(owner, rankTestID(11), "anchor shared", 0)
	target.Vector = []float32{1, 1, 1}
	return append(docs, target), target.ID
}

func TestRankFusesCandidatesBeyondBothTopFive(t *testing.T) {
	s, repo, req := setup(t)
	req.Turn.Text = "anchor"
	docs, target := sharedSixthCandidates(req.UserID)
	for name, hits := range map[string][]ranked{"current vector": vectorSearch([]float32{1, 0, 0}, docs), "contextual vector": vectorSearch([]float32{0, 1, 0}, docs)} {
		if len(hits) != 6 || docs[hits[5].index].ID != target {
			t.Fatalf("fixture must put target sixth in %s: %+v", name, hits)
		}
	}
	repo.candidates = docs
	s.store = contextualRepo{repo, Hints{ContextQuery: "contextual anchor"}}
	s.embed = queryEmbedFunc(func(_ context.Context, r ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
		v := []float32{1, 0, 0}
		if r.Input == "contextual anchor" {
			v = []float32{0, 1, 0}
		}
		return ports.EmbeddingResponse{Model: "synthetic", Vector: v}, nil
	})
	for _, maxItems := range []int{1, 5} {
		t.Run(fmt.Sprintf("MaxItems=%d", maxItems), func(t *testing.T) {
			req.MaxItems = maxItems
			out, err := s.Retrieve(context.Background(), req)
			if err != nil {
				t.Fatal(err)
			}
			if len(out.Items) != maxItems || out.Items[0].ID != target {
				t.Fatalf("shared sixth-ranked source must lead final Top %d: %+v", maxItems, out.Items)
			}
			if math.Abs(float64(out.Items[0].Score)-2.0/66) > 1e-8 {
				t.Fatalf("expected both rank-six votes, got %v", out.Items[0].Score)
			}
		})
	}
}

func TestRankAppliesContextBudgetsAfterFusion(t *testing.T) {
	s, _, req := setup(t)
	docs, target := sharedSixthCandidates(req.UserID)
	for _, tc := range []struct {
		name                  string
		perItem, total, count int
	}{
		{"total budget", 60, 4, 1},
		{"per-item and total budgets", 2, 4, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := s.rankWithHints(context.Background(), c.RetrievalResult{}, docs, "anchor", []float32{1, 0, 0}, 5, tc.perItem, tc.total, Hints{ContextQuery: "contextual anchor", ContextVector: []float32{0, 1, 0}})
			if len(out.Items) != tc.count || out.Items[0].ID != target {
				t.Fatalf("budgets changed fusion winner: %+v", out.Items)
			}
			used := 0
			for _, item := range out.Items {
				n := approximateTokens(item.Text)
				if n > tc.perItem {
					t.Fatalf("per-item budget exceeded: %d", n)
				}
				used += n
			}
			if used > tc.total {
				t.Fatalf("context budget exceeded: %d", used)
			}
		})
	}
}

func TestRankDeduplicatesAndFillsFinalSelection(t *testing.T) {
	s, _, req := setup(t)
	var docs []Candidate
	for i := 1; i <= 14; i++ {
		text := "anchor duplicate"
		if i >= 10 {
			text = fmt.Sprintf("anchor unique%d", i)
		}
		item := memory(req.UserID, rankTestID(i), text, 0)
		item.Vector = []float32{1, 0}
		docs = append(docs, item)
	}
	out := s.rank(context.Background(), c.RetrievalResult{}, docs, "anchor", []float32{1, 0}, 5, 60, 400)
	want := []c.ID{rankTestID(1), rankTestID(10), rankTestID(11), rankTestID(12), rankTestID(13)}
	if len(out.Items) != len(want) {
		t.Fatalf("duplicate leaders must not starve final selection: %+v", out.Items)
	}
	for i, item := range out.Items {
		if item.ID != want[i] {
			t.Fatalf("unexpected deduplicated order: %+v", out.Items)
		}
	}
}

func TestRankTieOrderIsStableAcrossInputPermutations(t *testing.T) {
	for _, mixedSources := range []bool{false, true} {
		t.Run(fmt.Sprintf("mixedSources=%t", mixedSources), func(t *testing.T) {
			s, _, req := setup(t)
			docs := []Candidate{
				memory(req.UserID, rankTestID(4), "semantic fourth", 0),
				memory(req.UserID, rankTestID(1), "anchor first", 0),
				memory(req.UserID, rankTestID(2), "semantic second", 0),
				memory(req.UserID, rankTestID(3), "anchor third", 0),
			}
			for i := range docs {
				docs[i].Vector = []float32{1, 0}
				docs[i].Text = fmt.Sprintf("record%d", i)
			}
			want := []c.ID{rankTestID(1), rankTestID(2), rankTestID(3), rankTestID(4)}
			if mixedSources {
				docs[1].Source, docs[3].Source = SourceMessage, SourceMessage
				want = []c.ID{rankTestID(2), rankTestID(1), rankTestID(4), rankTestID(3)}
			}
			random := rand.New(rand.NewSource(17))
			var baseline []c.MemoryView
			for attempt := 0; attempt < 20; attempt++ {
				random.Shuffle(len(docs), func(i, j int) { docs[i], docs[j] = docs[j], docs[i] })
				out := s.rank(context.Background(), c.RetrievalResult{}, docs, "anchor", []float32{1, 0}, 4, 60, 400)
				if len(out.Items) != len(want) {
					t.Fatal(out.Items)
				}
				for i, item := range out.Items {
					if item.ID != want[i] {
						t.Fatalf("source/ID tie order changed: %+v", out.Items)
					}
				}
				if attempt == 0 {
					baseline = out.Items
				} else if !reflect.DeepEqual(baseline, out.Items) {
					t.Fatal("permutation changed ranked output")
				}
			}
		})
	}
}
