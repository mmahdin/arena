package retrieval

import (
	"context"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type queryEmbedFunc func(context.Context, ports.EmbeddingRequest) (ports.EmbeddingResponse, error)

func (f queryEmbedFunc) Embed(ctx context.Context, r ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
	return f(ctx, r)
}

func assertNoBM25(t *testing.T, out c.RetrievalResult) {
	t.Helper()
	strategies := append(append([]string{}, out.Meta.Strategies...), out.Meta.FailedStrategies...)
	for _, q := range out.Meta.Queries {
		strategies = append(strategies, q.Strategies...)
		strategies = append(strategies, q.FailedStrategies...)
	}
	for _, strategy := range strategies {
		if strings.Contains(strings.ToLower(strategy), "bm25") || strings.Contains(strings.ToLower(strategy), "lexical") {
			t.Fatal("disabled strategy reported as executed", out.Meta)
		}
	}
}

func TestCurrentAndContextualLexicalMatchesHaveNoRankingVote(t *testing.T) {
	s, repo, req := setup(t)
	req.Turn.Text = "keyword"
	s.store = contextualRepo{repo, Hints{ContextQuery: "contextual keyword"}}
	s.embed = queryEmbedFunc(func(context.Context, ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
		return ports.EmbeddingResponse{Model: "synthetic", Vector: []float32{1, 0}}, nil
	})
	strong := memory(req.UserID, rankTestID(1), "different words", 0)
	strong.Vector = []float32{1, 0}
	weak := memory(req.UserID, rankTestID(2), strings.Repeat("keyword contextual ", 50), 0)
	weak.Vector = []float32{1, 1}
	pending := memory(req.UserID, rankTestID(3), strings.Repeat("keyword contextual ", 100), 0)
	pending.Lexical = BuildLexical(pending.Text)
	invalid := pending
	invalid.ID = rankTestID(4)
	invalid.Vector = []float32{float32(math.NaN()), 0}
	repo.candidates = []Candidate{weak, pending, strong, invalid}
	out, e := s.Retrieve(context.Background(), req)
	if e != nil || len(out.Items) != 2 || out.Items[0].ID != strong.ID || out.Items[1].ID != weak.ID {
		t.Fatal("lexical text influenced active rank membership/order", out, e)
	}
	if math.Abs(float64(out.Items[0].Score)-2.0/61) > 1e-8 || math.Abs(float64(out.Items[1].Score)-2.0/62) > 1e-8 {
		t.Fatal("RRF contains non-semantic query votes", out.Items)
	}
	assertNoBM25(t, out)
	if len(repo.queries) != 1 || len(repo.queries[0].TermHashes) != 0 {
		t.Fatal("active scan requested keyword frequencies", repo.queries)
	}
}

func TestQueryFailuresKeepOnlyValidatedSemanticBranches(t *testing.T) {
	for _, mode := range []string{"contextual_failure", "contextual_timeout", "current_failure", "both_failure", "incompatible_contextual", "invalid_current"} {
		t.Run(mode, func(t *testing.T) {
			s, repo, req := setup(t)
			req.Turn.Text = "current keyword"
			s.store = contextualRepo{repo, Hints{ContextQuery: "context keyword", ContextVector: []float32{0, 1}}}
			if mode == "contextual_timeout" {
				raw := s.bundles.(fixtureRegistry).handle.(fixtureHandle).raw
				s.bundles = fixtureRegistry{fixtureHandle{[]byte(strings.Replace(string(raw), "per_strategy_ms: 500", "per_strategy_ms: 20", 1))}}
			}
			s.embed = queryEmbedFunc(func(ctx context.Context, r ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
				current := r.Input == req.Turn.Text
				if mode == "both_failure" || current && mode == "current_failure" || !current && mode == "contextual_failure" {
					return ports.EmbeddingResponse{}, errors.New("synthetic embedding failure")
				}
				if !current && mode == "contextual_timeout" {
					<-ctx.Done()
					return ports.EmbeddingResponse{}, ctx.Err()
				}
				if current && mode == "invalid_current" {
					return ports.EmbeddingResponse{Model: "synthetic", Vector: []float32{0, 0}}, nil
				}
				model := "synthetic"
				v := []float32{1, 0}
				if !current {
					v = []float32{0, 1}
					if mode == "incompatible_contextual" {
						model = "other-model"
					}
				}
				return ports.EmbeddingResponse{Model: model, Vector: v}, nil
			})
			current := memory(req.UserID, rankTestID(1), "no lexical hit", 0)
			current.Vector = []float32{1, 0}
			contextual := memory(req.UserID, rankTestID(2), "different text", 0)
			contextual.Vector = []float32{0, 1}
			lexical := memory(req.UserID, rankTestID(3), "current context keyword", 0)
			repo.candidates = []Candidate{current, contextual, lexical}
			out, e := s.Retrieve(context.Background(), req)
			if e != nil || !out.Meta.Degraded || out.Meta.DegradedReason != "semantic_unavailable" {
				t.Fatal("semantic failure hidden", out, e)
			}
			wanted := current.ID
			if mode == "current_failure" || mode == "invalid_current" {
				wanted = contextual.ID
			}
			if mode == "both_failure" {
				if len(out.Items) != 0 || len(out.Meta.Strategies) != 0 || len(out.Meta.FailedStrategies) != 2 {
					t.Fatal("failed embeddings produced fake matches/strategies", out)
				}
			} else if len(out.Items) != 1 || out.Items[0].ID != wanted || len(out.Meta.FailedStrategies) != 1 {
				t.Fatal("validated surviving branch was lost or failed branch reused", out)
			}
			for _, query := range out.Meta.Queries {
				if len(query.FailedStrategies) > 0 && len(query.Strategies) != 0 {
					t.Fatal("failed query advertised success", query)
				}
			}
			assertNoBM25(t, out)
		})
	}
}

func TestNonBM25ThreadAndTemporalVotesRemainIndependent(t *testing.T) {
	for _, kind := range []string{"thread", "temporal"} {
		t.Run(kind, func(t *testing.T) {
			s, repo, req := setup(t)
			s.embed.(*fakeEmbed).err = errors.New("synthetic outage")
			signal := memory(req.UserID, rankTestID(1), "unrelated metadata-selected source", 0)
			lexical := memory(req.UserID, rankTestID(2), req.Turn.Text, 2000)
			if kind == "thread" {
				focus := signal.ID
				req.FocusThread = &focus
				signal.Source = SourceThread
			} else {
				signal.CreatedAt = time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
				req.Turn.TemporalRefs = []c.TemporalRef{{From: signal.CreatedAt.Add(-time.Hour), To: signal.CreatedAt.Add(time.Hour)}}
			}
			repo.candidates = []Candidate{signal, lexical}
			out, e := s.Retrieve(context.Background(), req)
			if e != nil || !out.Meta.Degraded || len(out.Items) != 1 || out.Items[0].ID != signal.ID || !reflect.DeepEqual(out.Meta.Strategies, []string{kind}) || math.Abs(float64(out.Items[0].Score)-1.0/61) > 1e-8 {
				t.Fatal("independent metadata signal changed or lexical match leaked", out, e)
			}
			assertNoBM25(t, out)
		})
	}
}
