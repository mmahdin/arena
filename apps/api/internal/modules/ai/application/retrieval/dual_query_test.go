package retrieval

import (
	"context"
	"errors"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"strings"
	"sync"
	"testing"
)

type contextualRepo struct {
	*fixtureRepo
	hint Hints
}

func (r contextualRepo) RetrievalHints(context.Context, c.ID, c.ID) (Hints, error) {
	return r.hint, nil
}

type dualEmbed struct {
	mu           sync.Mutex
	inputs       []string
	contextFails bool
}

func (e *dualEmbed) Embed(ctx context.Context, r ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
	e.mu.Lock()
	e.inputs = append(e.inputs, r.Input)
	e.mu.Unlock()
	if strings.Contains(r.Input, "assistant:") {
		if e.contextFails {
			return ports.EmbeddingResponse{}, errors.New("synthetic timeout")
		}
		return ports.EmbeddingResponse{Model: "synthetic", Vector: []float32{0, 1}}, nil
	}
	return ports.EmbeddingResponse{Model: "synthetic", Vector: []float32{1, 0}}, nil
}
func TestDualSemanticQueriesPreserveSuccessfulCurrentBranch(t *testing.T) {
	for _, fail := range []bool{false, true} {
		s, repo, req := setup(t)
		req.Turn.Text = "The same thing happened again."
		s.store = contextualRepo{repo, Hints{RecentContext: "user: The fictional argument with my sibling hurt.\nassistant: What happened next?"}}
		embed := &dualEmbed{contextFails: fail}
		s.embed = embed
		one := memory(req.UserID, c.NewID(), "unrelated surface phrase", 0)
		one.Vector = []float32{1, 0}
		two := memory(req.UserID, c.NewID(), "The fictional argument with my sibling", 0)
		two.Vector = []float32{0, 1}
		repo.candidates = []Candidate{one, two, one}
		out, e := s.Retrieve(context.Background(), req)
		if e != nil || len(embed.inputs) != 2 || len(out.Meta.Queries) != 2 || len(out.Items) != map[bool]int{false: 2, true: 1}[fail] || out.Meta.Degraded != fail {
			t.Fatal(out, e, embed.inputs)
		}
		if out.Meta.Queries[0].Text != req.Turn.Text || !strings.HasSuffix(out.Meta.Queries[1].Text, "user: "+req.Turn.Text) {
			t.Fatal("queries replaced current input")
		}
		if len(repo.queries) != 1 {
			t.Fatal("queries must share one authorized corpus scan")
		}
		if fail && len(out.Meta.Queries[1].FailedStrategies) != 1 {
			t.Fatal("contextual semantic failure concealed")
		}
	}
}
