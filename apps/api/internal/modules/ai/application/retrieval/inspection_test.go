package retrieval

import (
	"context"
	"encoding/json"
	"errors"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestInspectorRetrievalCapturesCandidatesScoresQueriesAndSelection(t *testing.T) {
	s, repo, req := setup(t)
	candidate := memory(req.UserID, c.NewID(), "situation at work from a synthetic memory", 1)
	candidate.Vector = []float32{1, 0}
	repo.candidates = []Candidate{candidate}
	dir := filepath.Join(t.TempDir(), "private")
	ins, err := trace.NewInspector(trace.InspectorOptions{Directory: dir, Environment: "local", Address: "localhost:8080"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, root := ins.StartTurn(context.Background(), c.TurnRequest{Text: req.Turn.Text})
	out, err := s.Retrieve(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	root.End(nil, nil)
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	raw, _ := os.ReadFile(files[0])
	if err = trace.ValidateJSON(raw); err != nil {
		t.Fatal(err)
	}
	var d trace.Document
	_ = json.Unmarshal(raw, &d)
	scored, selected, queried := false, false, false
	for _, n := range d.Nodes {
		if n.Operation == "retrieval.repository.Scan" {
			queried = true
		}
		for _, a := range n.Artifacts {
			if a.Name == "score" {
				scored = true
			}
			if a.Name == "selected" {
				var view c.MemoryView
				_ = json.Unmarshal(a.Value, &view)
				for _, actual := range out.Items {
					if reflect.DeepEqual(view, actual) {
						selected = true
					}
				}
			}
		}
	}
	if !scored || !selected || !queried || !strings.Contains(string(raw), candidate.Text) {
		t.Fatal("retrieval details missing", scored, selected, queried)
	}
}

func TestInspectorRankPreviewsStayBoundedWithoutLimitingFusion(t *testing.T) {
	s, _, req := setup(t)
	docs, target := sharedSixthCandidates(req.UserID)
	// Both full rankings now exceed 100 entries. Extra shared hits rank below
	// the target, but must not turn the branch previews into full-list copies.
	for i := 12; i <= 112; i++ {
		item := memory(req.UserID, rankTestID(i), "anchor trace extra "+rankTestID(i).String(), 0)
		item.Vector = []float32{1, 1, 4}
		docs = append(docs, item)
	}
	dir := filepath.Join(t.TempDir(), "private")
	ins, err := trace.NewInspector(trace.InspectorOptions{Directory: dir, Environment: "local", Address: "localhost:8080"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, root := ins.StartTurn(context.Background(), c.TurnRequest{Text: "anchor"})
	out := s.rankWithHints(ctx, c.RetrievalResult{}, docs, "anchor", []float32{1, 0, 0}, 5, 60, 400, Hints{ContextQuery: "contextual anchor", ContextVector: []float32{0, 1, 0}})
	root.End(nil, nil)
	if len(out.Items) != 5 || out.Items[0].ID != target {
		t.Fatal("trace preview limited fusion", out.Items)
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil || len(files) != 1 {
		t.Fatal(files, err)
	}
	raw, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if err = trace.ValidateJSON(raw); err != nil {
		t.Fatal(err)
	}
	var document trace.Document
	if err = json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	previews, scores, selected := 0, 0, 0
	for _, node := range document.Nodes {
		if node.Operation != "retrieval.rank" {
			continue
		}
		for _, artifact := range node.Artifacts {
			if strings.Contains(artifact.Name, "bm25") || artifact.Source == "bm25" {
				t.Fatal("disabled BM25 emitted a rank artifact", artifact.Name)
			}
			switch artifact.Name {
			case "vector_ranks", "contextual_vector_ranks":
				var hits []struct {
					ID   c.ID `json:"id"`
					Rank int  `json:"rank"`
				}
				if err = json.Unmarshal(artifact.Value, &hits); err != nil {
					t.Fatal(err)
				}
				if len(hits) != 5 {
					t.Fatalf("rank preview grew with corpus: %d", len(hits))
				}
				for i, hit := range hits {
					if hit.Rank != i+1 || hit.ID == target {
						t.Fatal("preview no longer represents only the branch Top 5", hits)
					}
				}
				previews++
			case "score":
				scores++
			case "selected":
				selected++
			}
		}
	}
	if previews != 2 || scores != len(out.Items) || selected != len(out.Items) {
		t.Fatal("unexpected rank trace volume", previews, scores, selected)
	}
}

func TestInspectorSemanticRetrievalHasNoBM25Artifacts(t *testing.T) {
	for _, mode := range []string{"dual_success", "contextual_failure", "all_failed"} {
		t.Run(mode, func(t *testing.T) {
			s, repo, req := setup(t)
			s.store = contextualRepo{repo, Hints{ContextQuery: "assistant: synthetic contextual query"}}
			s.embed = queryEmbedFunc(func(_ context.Context, r ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
				if mode == "all_failed" || mode == "contextual_failure" && strings.Contains(r.Input, "assistant:") {
					return ports.EmbeddingResponse{}, errors.New("synthetic embedding failure")
				}
				v := []float32{1, 0}
				if strings.Contains(r.Input, "assistant:") {
					v = []float32{0, 1}
				}
				return ports.EmbeddingResponse{Model: "synthetic", Vector: v}, nil
			})
			one := memory(req.UserID, rankTestID(1), "synthetic semantic evidence", 1)
			one.Vector = []float32{1, 0}
			two := memory(req.UserID, rankTestID(2), "synthetic contextual evidence", 1)
			two.Vector = []float32{0, 1}
			repo.candidates = []Candidate{one, two, memory(req.UserID, rankTestID(3), req.Turn.Text, 1)}
			dir := filepath.Join(t.TempDir(), "private")
			ins, e := trace.NewInspector(trace.InspectorOptions{Directory: dir, Environment: "local", Address: "localhost:8080"})
			if e != nil {
				t.Fatal(e)
			}
			ctx, root := ins.StartTurn(context.Background(), c.TurnRequest{Text: req.Turn.Text})
			trace.Associate(ctx, req.Turn.TurnID)
			out, e := s.Retrieve(ctx, req)
			root.End(func() any { return out }, e)
			if e != nil {
				t.Fatal(e)
			}
			assertNoBM25(t, out)
			files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
			if len(files) != 1 {
				t.Fatal(files)
			}
			raw, e := os.ReadFile(files[0])
			if e != nil {
				t.Fatal(e)
			}
			if e = trace.ValidateJSON(raw); e != nil {
				t.Fatal(e)
			}
			var doc trace.Document
			if e = json.Unmarshal(raw, &doc); e != nil {
				t.Fatal(e)
			}
			ranks := map[string]bool{}
			for _, node := range doc.Nodes {
				if node.Operation == "application/retrieval/service.Retrieve" && mode != "dual_success" && node.Reason != "semantic_unavailable" {
					t.Fatal("Inspector hid degraded semantic retrieval", node.Reason)
				}
				for _, artifact := range node.Artifacts {
					if strings.Contains(artifact.Name, "bm25") || artifact.Source == "bm25" {
						t.Fatal("disabled rank branch captured", artifact.Name)
					}
					if artifact.Kind == "retrieval_rank" {
						ranks[artifact.Name] = true
					}
				}
			}
			if ranks["vector_ranks"] != (mode != "all_failed") || ranks["contextual_vector_ranks"] != (mode == "dual_success") {
				t.Fatal("trace advertises a failed semantic branch", ranks)
			}
			if target := os.Getenv("THEIA_WRITE_SEMANTIC_FIXTURE"); target != "" && mode == "contextual_failure" {
				if e = os.WriteFile(target, raw, 0600); e != nil {
					t.Fatal(e)
				}
			}
		})
	}
}
