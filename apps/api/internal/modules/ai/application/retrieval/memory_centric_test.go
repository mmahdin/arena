package retrieval

import (
	"context"
	"errors"
	"fmt"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"sort"
	"strings"
	"testing"
	"time"
)

type linkedRepo struct {
	*fixtureRepo
	links    map[c.ID][]c.EvidenceRef
	items    map[c.ID]Candidate
	baseline []Candidate
	lookups  int
}

func (r *linkedRepo) Evidence(_ context.Context, _ Query, item Candidate) ([]c.EvidenceRef, *c.ID, error) {
	return r.links[item.ID], nil, nil
}
func (r *linkedRepo) Lookup(_ context.Context, q Query, _ SourceKind, id c.ID) (Candidate, bool, error) {
	r.lookups++
	v, ok := r.items[id]
	return v, ok && v.UserID == q.UserID, nil
}
func (r *linkedRepo) Baseline(context.Context, c.ID, c.ID) ([]Candidate, error) {
	return r.baseline, nil
}
func TestEmbeddingFailurePersianAndEnglishHasNoLexicalFallback(t *testing.T) {
	for _, text := range []string{"fictional interview", "مصاحبه کاری ساختگی"} {
		s, r, q := setup(t)
		q.Turn.Text = text
		r.candidates = []Candidate{memory(q.UserID, c.NewID(), text, 100)}
		s.embed.(*fakeEmbed).err = errors.New("synthetic embedding outage")
		out, e := s.Retrieve(context.Background(), q)
		if e != nil || len(out.Items) != 0 || !out.Meta.Degraded || len(out.Meta.FailedStrategies) != 1 || len(out.Meta.Strategies) != 0 {
			t.Fatal(out, e)
		}
	}
}
func TestBoundedEvidenceExpansionPreservesContradictionAndIdentity(t *testing.T) {
	s, base, q := setup(t)
	parent := memory(q.UserID, c.NewID(), "fictional interview concern", 10)
	parent.Vector = []float32{1, 0}
	evidence := memory(q.UserID, c.NewID(), "I corrected the fictional interview date", 10)
	evidence.Source = SourceMessage
	repo := &linkedRepo{fixtureRepo: base, links: map[c.ID][]c.EvidenceRef{parent.ID: {{Kind: "message", Ref: evidence.ID, Role: "contradicts"}}}, items: map[c.ID]Candidate{evidence.ID: evidence}}
	base.candidates = []Candidate{parent}
	s.store = repo
	q.Turn.Text = "fictional interview"
	out, e := s.Retrieve(context.Background(), q)
	if e != nil || len(out.Items) != 2 || out.Items[1].ExpandedFrom == nil || *out.Items[1].ExpandedFrom != parent.ID || out.Items[1].Evidence[0].Role != "contradicts" || out.Items[1].Source != "message" || repo.lookups != 1 {
		t.Fatal(out, e, repo.lookups)
	}
	repo.items[evidence.ID] = memory(c.NewID(), evidence.ID, "foreign text", 10)
	out, e = s.Retrieve(context.Background(), q)
	if e != nil || len(out.Items) != 1 {
		t.Fatal("foreign evidence leaked", out, e)
	}
}
func TestBaselineDoesNotCallEmbedderAndRejectsForeignOwner(t *testing.T) {
	s, base, q := setup(t)
	item := memory(q.UserID, c.NewID(), "confirmed preference", 1)
	repo := &linkedRepo{fixtureRepo: base, baseline: []Candidate{item}}
	s.store = repo
	s.embed.(*fakeEmbed).err = errors.New("embedding unavailable")
	out, e := s.Persistent(context.Background(), q.UserID, q.Turn.MessageID)
	if e != nil || len(out.Items) != 1 || s.embed.(*fakeEmbed).calls != 0 {
		t.Fatal(out, e)
	}
	repo.baseline[0].UserID = c.NewID()
	out, e = s.Persistent(context.Background(), q.UserID, q.Turn.MessageID)
	if e == nil || len(out.Items) > 0 {
		t.Fatal("foreign baseline leaked")
	}
}
func TestTemporalHints(t *testing.T) {
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	for _, text := range []string{"yesterday", "دیروز", "2026-11-22", "۲۰۲۶-۱۱-۲۲"} {
		refs := timeWindows(text, "UTC", now)
		if len(refs) != 1 || refs[0].From.Day() != 22 {
			t.Fatal(text, refs)
		}
	}
}
func TestThreadSourceRanking(t *testing.T) {
	focus := c.NewID()
	docs := []Candidate{{ID: focus, Source: SourceThread, Text: "fictional meeting"}, {ID: c.NewID(), Source: SourceMessage, Text: "fictional meeting"}, {ID: c.NewID(), Source: SourceMemory, Text: "independent interview evidence"}}
	if hits := threadRanks(docs, &focus); len(hits) != 1 || hits[0].index != 0 {
		t.Fatal(hits)
	}
}

// These timings include mocked, immediate embeddings and in-memory sources.
// They measure local ranking/selection CPU, never SQL or provider performance.
func TestSyntheticRetrievalLatencyAndRecall(t *testing.T) {
	for _, n := range []int{100, 1000, 10000} {
		s, r, q := setup(t)
		// This is a CPU/relevance measurement, not a wall-clock release gate.
		// Race instrumentation or concurrent CI work must not redefine relevance.
		raw := s.bundles.(fixtureRegistry).handle.(fixtureHandle).raw
		s.bundles = fixtureRegistry{fixtureHandle{[]byte(strings.Replace(string(raw), "total_ms: 1000", "total_ms: 5000", 1))}}
		q.Turn.Text = "unique synthetic interview"
		target := memory(q.UserID, c.NewID(), q.Turn.Text, 2000)
		target.Vector = []float32{1, 0}
		r.candidates = []Candidate{target}
		for i := 1; i < n; i++ {
			d := memory(q.UserID, rankTestID(i+100), fmt.Sprintf("unrelated synthetic cooking record %d", i), i%60)
			d.Vector = []float32{0, 1}
			r.candidates = append(r.candidates, d)
		}
		samples := []time.Duration{}
		hits := 0
		for i := 0; i < 20; i++ {
			start := time.Now()
			out, e := s.Retrieve(context.Background(), q)
			samples = append(samples, time.Since(start))
			if e != nil {
				t.Fatal(e)
			}
			for _, v := range out.Items {
				if v.ID == target.ID {
					hits++
					break
				}
			}
		}
		sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
		t.Logf("synthetic n=%d p50=%s p95=%s Recall@5=%d/20", n, samples[10], samples[18], hits)
		if hits != 20 {
			t.Fatal("old relevant evidence lost")
		}
	}
}

func TestLexicalProjectionPreservesExactBM25Scores(t *testing.T) {
	raw := []Candidate{{Text: "fictional interview interview tomorrow"}, {Text: "unrelated cooking story"}, {Text: "مصاحبه کاری ساختگی"}}
	compact := make([]Candidate, len(raw))
	copy(compact, raw)
	for i := range compact {
		compact[i].Lexical = BuildLexical(compact[i].Text)
		compact[i].Text = ""
	}
	for _, query := range []string{"interview", "مصاحبه کاری"} {
		a, b := bm25(query, raw), bm25(query, compact)
		if len(a) != len(b) {
			t.Fatal(a, b)
		}
		for i := range a {
			if a[i].index != b[i].index || a[i].score != b[i].score {
				t.Fatal("cached BM25 changed rankings", a, b)
			}
		}
	}
}
func TestCompactCorpusHydratesOnlySelectedSources(t *testing.T) {
	s, base, q := setup(t)
	q.Turn.Text = "unique interview"
	repo := &linkedRepo{fixtureRepo: base, items: map[c.ID]Candidate{}}
	s.store = repo
	target := memory(q.UserID, c.NewID(), q.Turn.Text, 2000)
	target.Vector = []float32{1, 0}
	base.candidates = []Candidate{target}
	for i := 1; i < 2000; i++ {
		v := memory(q.UserID, rankTestID(i+100), "unrelated fictional cooking", i%50)
		v.Vector = []float32{0, 1}
		base.candidates = append(base.candidates, v)
	}
	for i := range base.candidates {
		v := base.candidates[i]
		repo.items[v.ID] = v
		base.candidates[i].Lexical = BuildLexical(v.Text)
		base.candidates[i].Text = ""
	}
	out, e := s.Retrieve(context.Background(), q)
	if e != nil || len(out.Items) == 0 || out.Items[0].ID != target.ID || repo.lookups > q.MaxItems || out.Items[0].Text == "" {
		t.Fatal("compact ranking lost evidence or hydrated corpus", out, e, repo.lookups)
	}
}
