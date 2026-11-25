package retrieval

import (
	"context"
	"fmt"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"os"
	"testing"
)

func BenchmarkHybridCompactCorpus(b *testing.B) {
	for _, n := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			raw, e := os.ReadFile("testdata/retrieval.yaml")
			if e != nil {
				b.Fatal(e)
			}
			owner := c.NewID()
			repo := &linkedRepo{fixtureRepo: &fixtureRepo{}, items: map[c.ID]Candidate{}}
			for i := 0; i < n; i++ {
				item := memory(owner, rankTestID(i+100), fmt.Sprintf("fictional corpus record %d", i), i%90)
				item.Vector = []float32{0, 1}
				if i == 0 {
					item.Text = "unique interview evidence"
					item.Vector = []float32{1, 0}
				}
				repo.items[item.ID] = item
				item.Lexical = BuildLexical(item.Text)
				item.Text = ""
				repo.candidates = append(repo.candidates, item)
			}
			s, e := New(repo, &fakeEmbed{vector: []float32{1, 0}, model: "synthetic"}, fixtureRegistry{fixtureHandle{raw}}, fixtureUse{search: true})
			if e != nil {
				b.Fatal(e)
			}
			req := c.RetrievalRequest{UserID: owner, Turn: c.TurnView{TurnID: c.NewID(), MessageID: c.NewID(), Text: "unique interview"}, MaxItems: 5}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				out, e := s.Retrieve(context.Background(), req)
				if e != nil || len(out.Items) == 0 {
					b.Fatal(e)
				}
			}
		})
	}
}
