package retrieval

import (
	"context"
	"errors"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"math"
	"os"
	"strings"
	"testing"
	"time"
)

type fixtureHandle struct{ raw []byte }

func (b fixtureHandle) Reference() c.BundleRef                           { return c.BundleRef{Version: "kb@test"} }
func (b fixtureHandle) Artifact(context.Context, string) ([]byte, error) { return b.raw, nil }

type fixtureRegistry struct{ handle ports.Bundle }

func (b fixtureRegistry) Active(context.Context, c.ID) (ports.Bundle, error) { return b.handle, nil }

type fixtureUse struct {
	search bool
	deny   map[c.ID]bool
	fail   bool
}

func (u fixtureUse) CanSearch(context.Context, c.ID, c.ID) (bool, error) {
	if u.fail {
		return false, errors.New("policy unavailable")
	}
	return u.search, nil
}
func (u fixtureUse) CanUse(_ context.Context, _ c.ID, _ SourceKind, id c.ID) (bool, error) {
	if u.fail {
		return false, errors.New("policy unavailable")
	}
	return !u.deny[id], nil
}

type fixtureRepo struct {
	state      State
	candidates []Candidate
	fail       error
	reads      int
	queries    []Query
}

func (r *fixtureRepo) UserState(context.Context, c.ID) (State, error) {
	r.reads++
	return r.state, r.fail
}
func (r *fixtureRepo) Scan(ctx context.Context, q Query, visit func(Candidate) error) error {
	r.reads++
	r.queries = append(r.queries, q)
	if r.fail != nil {
		return r.fail
	}
	for _, item := range r.candidates {
		if e := ctx.Err(); e != nil {
			return e
		}
		if e := visit(item); e != nil {
			return e
		}
	}
	return nil
}

type fakeEmbed struct {
	vector []float32
	model  string
	err    error
	calls  int
	input  string
	meta   c.CallMeta
}

func (f *fakeEmbed) Embed(_ context.Context, q ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
	f.calls++
	f.meta = q.Meta
	f.input = q.Input
	return ports.EmbeddingResponse{Vector: f.vector, Model: f.model}, f.err
}
func setup(t *testing.T) (*Service, *fixtureRepo, c.RetrievalRequest) {
	t.Helper()
	raw, e := os.ReadFile("testdata/retrieval.yaml")
	if e != nil {
		t.Fatal(e)
	}
	repo := &fixtureRepo{state: State{TZ: "Asia/Tehran"}}
	embed := &fakeEmbed{vector: []float32{1, 0}, model: "synthetic-multilingual"}
	s, e := New(repo, embed, fixtureRegistry{fixtureHandle{raw}}, fixtureUse{search: true})
	if e != nil {
		t.Fatal(e)
	}
	s.WithClock(func() time.Time { return time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC) })
	return s, repo, c.RetrievalRequest{UserID: c.NewID(), Turn: c.TurnView{TurnID: c.NewID(), MessageID: c.NewID(), Text: "situation at work", Lang: "en"}, MaxItems: 5}
}
func memory(owner, id c.ID, text string, days int) Candidate {
	return Candidate{ID: id, UserID: owner, Source: SourceMemory, SourceVersion: "0000000000000001", Kind: c.MemEpisode, Text: text, Epistemic: c.EpReported, Status: c.MemActive, Sensitivity: c.SensNormal, EvidenceN: 1, CreatedAt: time.Now().AddDate(0, 0, -days)}
}
func TestMandatoryDependencies(t *testing.T) {
	s, r, _ := setup(t)
	if _, e := New(r, nil, s.bundles, s.use); e == nil {
		t.Fatal("vector search must be mandatory")
	}
	if _, e := New(r, s.embed, s.bundles, nil); e == nil {
		t.Fatal("policy required")
	}
}
func TestOnePipelineUsesUnmodifiedMessageForAllLegacyHints(t *testing.T) {
	s, r, q := setup(t)
	q.Turn.Text = "درباره مدیرم چه گفتم؟"
	q.Extra = &c.NeedsContext{Query: "rewritten words"}
	q.Strategies = []c.RetrievalStrategy{c.StratTemporal, c.StratEntity}
	item := memory(q.UserID, c.NewID(), "گفتگو درباره مدیرم بود", 200)
	item.Vector = []float32{1, 0}
	r.candidates = []Candidate{item}
	out, e := s.Retrieve(context.Background(), q)
	if e != nil || len(out.Items) != 1 {
		t.Fatal(out, e)
	}
	f := s.embed.(*fakeEmbed)
	if f.calls != 1 || f.input != q.Turn.Text || len(r.queries) != 1 || strings.Join(out.Meta.Strategies, ",") != "vector" || out.Meta.ExactMatchFound != nil {
		t.Fatal("legacy route/query selection survived", out)
	}
}
func TestAllPersistentSourceKindsAndMemoryTypes(t *testing.T) {
	for _, source := range []SourceKind{SourceMemory, SourceMessage, SourceSession, SourceThread, SourceFormulation, SourceProtocol, SourceOpenLoop} {
		t.Run(string(source), func(t *testing.T) {
			s, r, q := setup(t)
			item := memory(q.UserID, c.NewID(), "work situation", 1)
			item.Source = source
			item.Vector = []float32{1, 0}
			r.candidates = []Candidate{item}
			out, e := s.Retrieve(context.Background(), q)
			if e != nil || len(out.Items) != 1 || out.Items[0].Source != string(source) || out.Items[0].SourceVersion != item.SourceVersion {
				t.Fatal(out, e)
			}
		})
	}
	for _, kind := range []c.MemoryKind{c.MemFact, c.MemPreference, c.MemEpisode, c.MemHypothesis, c.MemGoal, c.MemEvent, c.MemSafetyEvt} {
		s, r, q := setup(t)
		item := memory(q.UserID, c.NewID(), "work situation", 1)
		item.Kind = kind
		item.Vector = []float32{1, 0}
		if kind == c.MemHypothesis {
			item.Epistemic = c.EpInferred
			item.Status = c.MemSupported
		}
		r.candidates = []Candidate{item}
		out, e := s.Retrieve(context.Background(), q)
		if e != nil || len(out.Items) != 1 {
			t.Fatalf("excluded memory type %s: %+v %v", kind, out, e)
		}
	}
}
func TestSemanticOnlyExcludesUnindexedKeywordSources(t *testing.T) {
	s, r, q := setup(t)
	q.Turn.Text = "I feel overwhelmed by deadlines"
	semantic := memory(q.UserID, c.NewID(), "The workload has become too much", 300)
	semantic.Vector = []float32{1, 0}
	keyword := memory(q.UserID, c.NewID(), "deadlines", 1)
	r.candidates = []Candidate{semantic, keyword}
	out, e := s.Retrieve(context.Background(), q)
	if e != nil || len(out.Items) != 1 || out.Items[0].ID != semantic.ID {
		t.Fatal(out, e)
	}
}
func TestFullCorpusBeyondOldPreselectionCaps(t *testing.T) {
	s, r, q := setup(t)
	q.MaxItems = 1
	for i := 0; i < 2000; i++ {
		v := memory(q.UserID, c.NewID(), "unrelated recipe", 0)
		v.Vector = []float32{0, 1}
		r.candidates = append(r.candidates, v)
	}
	target := memory(q.UserID, c.NewID(), "work situation", 1000)
	target.Vector = []float32{1, 0}
	r.candidates = append(r.candidates, target)
	out, e := s.Retrieve(context.Background(), q)
	if e != nil || len(out.Items) != 1 || out.Items[0].ID != target.ID {
		t.Fatal("old relevant memory lost", out, e)
	}
}
func TestIsolationPermissionSensitivityAndFailure(t *testing.T) {
	for _, mode := range []string{"wrong_owner", "denied", "sensitive", "provider", "storage", "excluded"} {
		t.Run(mode, func(t *testing.T) {
			s, r, q := setup(t)
			v := memory(q.UserID, c.NewID(), "work situation", 1)
			r.candidates = []Candidate{v}
			switch mode {
			case "wrong_owner":
				r.candidates[0].UserID = c.NewID()
			case "denied":
				s.use = fixtureUse{search: true, deny: map[c.ID]bool{v.ID: true}}
			case "sensitive":
				r.candidates[0].Sensitivity = c.SensHighly
			case "provider":
				s.embed.(*fakeEmbed).err = errors.New("private failure")
			case "storage":
				r.fail = errors.New("private SQL")
			case "excluded":
				q.Exclude = []c.ID{v.ID}
			}
			out, _ := s.Retrieve(context.Background(), q)
			if len(out.Items) != 0 {
				t.Fatal("source leaked", out)
			}
			if mode == "provider" && len(out.Meta.Strategies) != 0 {
				t.Fatal("failed embedding fabricated an active strategy", out)
			}
			if mode == "provider" && !out.Meta.Degraded {
				t.Fatal("embedding failure was not reported as degraded")
			}
		})
	}
}
func TestIndependentSourcesWithIdenticalTextArePreserved(t *testing.T) {
	s, r, q := setup(t)
	a := memory(q.UserID, c.NewID(), "work situation", 0)
	a.Vector = []float32{1, 0}
	b := a
	b.ID = c.NewID()
	b.Source = SourceMessage
	r.candidates = []Candidate{a, b}
	out, e := s.Retrieve(context.Background(), q)
	if e != nil || len(out.Items) != 2 {
		t.Fatal(out, e)
	}
	if !strings.Contains(truncate(strings.Repeat("فارسی ", 200), 10), "…") {
		t.Fatal("UTF-8 truncation failed")
	}
}
func TestBM25FormulaAndNormalization(t *testing.T) {
	docs := []Candidate{{Text: "cat cat"}, {Text: "dog"}}
	got := bm25("cat", docs)
	want := math.Log(2) * 2 * 2.2 / (2 + 1.2*(.25+.75*2/1.5))
	if len(got) != 1 || math.Abs(got[0].score-want) > 1e-10 {
		t.Fatalf("not BM25: %+v want %f", got, want)
	}
	a := strings.Join(Terms("مديرِ كاري ۱۲۳ نیم‌فاصله"), " ")
	b := strings.Join(Terms("مدیر کاری 123 نیم فاصله"), " ")
	if a != b {
		t.Fatal(a, b)
	}
	long := strings.Repeat("unrelated ", 80) + "cat"
	r := bm25("cat", []Candidate{{Text: long}, {Text: "cat"}})
	if r[0].index != 1 {
		t.Fatal("length normalization missing", r)
	}
}
func TestRRFUsesRanksAndEqualVotes(t *testing.T) {
	docs := []Candidate{{ID: c.NewID()}, {ID: c.NewID()}, {ID: c.NewID()}}
	got := fuse(docs, []ranked{{0, .999}, {1, .8}}, []ranked{{1, 900}, {2, 300}})
	if got[0].index != 1 || math.Abs(got[0].score-(1.0/62+1.0/61)) > 1e-10 {
		t.Fatal(got)
	}
}

// These are complete retrieval runs over representative synthetic conversations,
// with deterministic semantic vectors to make regressions reproducible offline.
func TestPersianConversationRelevance(t *testing.T) {
	cases := []struct {
		query, target string
		vector        []float32
	}{
		{"درباره جلسه با مدیرم چه گفته بودم؟", "دیروز جلسه با مدیرم داشتم و درباره حجم کار صحبت کردیم", []float32{1, 0}},
		{"خوابم هنوز به هم ریخته", "چند شب است تا صبح بیدار می‌مانم و خسته‌ام", []float32{0, 1}},
		{"از امتحان زبانم نگرانم", "برای امتحان زبان هفته بعد آماده می‌شوم", []float32{1, 0}},
		{"What did we discuss about my interview?", "درباره مصاحبه شغلی فردا و سوال‌های مدیر صحبت کردیم", []float32{1, 0}},
	}
	hits := 0
	for _, tc := range cases {
		s, r, q := setup(t)
		q.Turn.Text = tc.query
		q.MaxItems = 1
		s.embed.(*fakeEmbed).vector = tc.vector
		target := memory(q.UserID, c.NewID(), tc.target, 40)
		target.Source = SourceMessage
		target.Vector = tc.vector
		other := memory(q.UserID, c.NewID(), "امروز یک غذای تازه پختم", 0)
		other.Vector = []float32{0, 0}
		r.candidates = []Candidate{other, target}
		out, e := s.Retrieve(context.Background(), q)
		if e != nil || len(out.Items) != 1 || out.Items[0].ID != target.ID {
			t.Errorf("query %q: %+v %v", tc.query, out, e)
		} else {
			hits++
		}
	}
	t.Logf("Persian/bilingual synthetic conversation Recall@1=%d/%d", hits, len(cases))
}

func TestRetrievalIgnoresLegacyPause(t *testing.T) {
	s, repo, req := setup(t)
	repo.state.MemoryPaused = true
	item := memory(req.UserID, c.NewID(), "work situation", 1)
	item.Vector = []float32{1, 0}
	repo.candidates = []Candidate{item}
	out, err := s.Retrieve(context.Background(), req)
	if err != nil || out.Meta.Degraded || len(out.Items) != 1 || out.Items[0].ID != item.ID {
		t.Fatalf("legacy pause blocked retrieval: %+v %v", out, err)
	}
}
