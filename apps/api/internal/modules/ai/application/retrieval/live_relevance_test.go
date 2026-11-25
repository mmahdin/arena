package retrieval

import (
	"context"
	"encoding/json"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/llmhttp"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"os"
	"testing"
	"time"
)

type liveEmbedding struct {
	provider *llmhttp.Provider
	model    llm.Model
	cache    map[string]ports.EmbeddingResponse
}

func (e liveEmbedding) Embed(ctx context.Context, req ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
	if v, ok := e.cache[req.Input]; ok {
		return v, nil
	}
	v, err := e.provider.Embed(ctx, e.model, req)
	response := ports.EmbeddingResponse{Vector: v.Vector, Model: e.model.Name}
	if err == nil {
		e.cache[req.Input] = response
	}
	return response, err
}

// Explicit opt-in: sends only these fabricated conversations to the configured
// embedding provider, never application records. No generator/query LLM is used.
func TestLivePersianConversationRelevance(t *testing.T) {
	if os.Getenv("THEIA_LIVE_EMBEDDINGS") != "1" {
		t.Skip("explicit synthetic embedding-provider evaluation opt-in required")
	}
	base := os.Getenv("MODEL_API_BASE_URL")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	provider, err := llmhttp.New(base, os.Getenv("MODEL_API_KEY"), nil)
	if err != nil {
		t.Fatal("live embedding configuration unavailable")
	}
	embed := liveEmbedding{provider: provider, model: llm.Model{Name: "text-embedding-3-small", Dimensions: 1024}, cache: map[string]ports.EmbeddingResponse{}}
	if cachePath := os.Getenv("THEIA_RELEVANCE_CACHE"); cachePath != "" {
		if raw, e := os.ReadFile(cachePath); e == nil {
			_ = json.Unmarshal(raw, &embed.cache)
		}
		defer func() { raw, _ := json.Marshal(embed.cache); _ = os.WriteFile(cachePath, raw, 0600) }()
	}

	docs := []string{
		"دیروز در جلسه شرکت، مدیرم دو بار حرفم را قطع کرد و از حجم کار ناراحت شدم.",
		"این هفته تا سحر بیدار می‌مانم و صبح‌ها خیلی خسته هستم.",
		"امتحان زبان انگلیسی من سه‌شنبه هفته آینده است و هنوز برای بخش شنیداری آماده نیستم.",
		"با خواهرم سر تقسیم کارهای خانه بحث کردم، ولی بعد با هم آشتی کردیم.",
		"پدرم چند روز است سرما خورده و برایش سوپ درست کردم.",
		"دارم برای نقل مکان از شیراز به تبریز و پیدا کردن خانه برنامه می‌ریزم.",
		"هر وقت کارم زیاد می‌شود، نقاشی با آبرنگ کمک می‌کند ذهنم آرام‌تر شود.",
		"برای پنجشنبه ساعت ده صبح وقت دندانپزشکی گرفته‌ام.",
		"دوشنبه و چهارشنبه بعد از کار به باشگاه می‌روم و شنا می‌کنم.",
		"آخر هفته برای دوستانم کیک شکلاتی پختم و دستورش را در دفترم نوشتم.",
		"قرار است جمعه با دوستم در پارک قدم بزنیم و درباره سفرش حرف بزنیم.",
		"گلدان روی میز کارم را هر سه روز یک بار آب می‌دهم.",
	}
	cases := []struct {
		query  string
		target int
	}{
		{"جلسه با مديرم چطور گذشت؟", 0},
		{"درباره مشکل خوابم چه گفته بودم؟", 1},
		{"امتحان زبان من کی است؟", 2},
		{"درباره دعوای من و خواهرم چی یادت هست؟", 3},
		{"Who was ill in my family?", 4},
		{"What did I say about changing cities?", 5},
		{"وقتی فشار کاری دارم چطور خودم را آرام می‌کنم؟", 6},
		{"وقت دندانپزشکی‌ام چه روزی است؟", 7},
		{"چند روز در هفته ورزش می‌کنم؟", 8},
		{"برای دوستانم چه شیرینی درست کردم؟", 9},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	s, repo, req := setup(t)
	s.embed = embed
	s.bundles = fixtureRegistry{fixtureHandle{[]byte("schema: retrieval.v2\ndefaults:\n  max_items: 5\n  hard_cap_with_hop: 7\n  per_item_tokens: 100\n  total_tokens: 500\ntimeouts:\n  total_ms: 5000\n  per_strategy_ms: 4500\n")}}
	sources := []SourceKind{SourceMemory, SourceMessage, SourceSession, SourceThread, SourceFormulation, SourceProtocol, SourceOpenLoop}
	for i, text := range docs {
		response, e := embed.Embed(ctx, ports.EmbeddingRequest{Input: text})
		if e != nil {
			t.Fatalf("embedding fixture %d unavailable: %T", i, e)
		}
		item := memory(req.UserID, c.NewID(), text, 100-i)
		item.Source = sources[i%len(sources)]
		item.Vector = response.Vector
		repo.candidates = append(repo.candidates, item)
	}
	top1, top3, top5, semanticTop5 := 0, 0, 0, 0
	rr := 0.
	for i, tc := range cases {
		req.Turn.Text = tc.query
		out, e := s.Retrieve(ctx, req)
		if e != nil {
			t.Fatalf("query %d failed: %T", i, e)
		}
		rank := 0
		for j, item := range out.Items {
			if item.ID == repo.candidates[tc.target].ID {
				rank = j + 1
				break
			}
		}
		if rank == 1 {
			top1++
		}
		if rank > 0 && rank <= 3 {
			top3++
		}
		if rank > 0 {
			top5++
			rr += 1 / float64(rank)
		}
		semanticRank, keywordRank := 0, 0
		for j, hit := range vectorSearch(embed.cache[tc.query].Vector, repo.candidates) {
			if hit.index == tc.target {
				semanticRank = j + 1
			}
		}
		for j, hit := range bm25(tc.query, repo.candidates) {
			if hit.index == tc.target {
				keywordRank = j + 1
			}
		}
		if semanticRank > 0 && semanticRank <= 5 {
			semanticTop5++
		}
		t.Logf("synthetic query %02d selected_semantic=%d cosine=%d offline_BM25_reference=%d", i+1, rank, semanticRank, keywordRank)
	}
	t.Logf("model=%s dim=%d corpus=%d queries=%d Recall@1=%.3f Recall@3=%.3f Recall@5=%.3f MRR@5=%.3f", embed.model.Name, embed.model.Dimensions, len(docs), len(cases), float64(top1)/float64(len(cases)), float64(top3)/float64(len(cases)), float64(top5)/float64(len(cases)), rr/float64(len(cases)))
	t.Logf("semantic-only Recall@5=%.3f", float64(semanticTop5)/float64(len(cases)))
	if top5 < semanticTop5 {
		t.Fatalf("semantic selection lost corpus coverage: Recall@5=%d/%d, vector baseline=%d/%d", top5, len(cases), semanticTop5, len(cases))
	}
}
