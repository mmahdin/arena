package retrieval

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type fixtureHandle struct{ raw []byte }

func (b fixtureHandle) Reference() c.BundleRef { return c.BundleRef{Version: "kb@test"} }
func (b fixtureHandle) Artifact(_ context.Context, path string) ([]byte, error) {
	if path != "config/retrieval.yaml" {
		return nil, errors.New("missing bundle artifact")
	}
	return b.raw, nil
}

type fixtureRegistry struct{ handle ports.Bundle }

func (b fixtureRegistry) Active(context.Context, c.ID) (ports.Bundle, error) { return b.handle, nil }

type fixtureUse struct {
	search bool
	deny   map[c.ID]bool
	fail   bool
}

func (u fixtureUse) CanSearch(context.Context, c.ID, c.ID) (bool, error) {
	if u.fail {
		return false, errors.New("eligibility offline")
	}
	return u.search, nil
}
func (u fixtureUse) CanUse(_ context.Context, _ c.ID, _ SourceKind, id c.ID) (bool, error) {
	if u.fail {
		return false, errors.New("eligibility offline")
	}
	return !u.deny[id], nil
}

type fixtureRepo struct {
	state      State
	candidates []Candidate
	messages   []Candidate
	recent     []string
	fail       error
	reads      int
	queries    []MemoryQuery
	windows    []MessageQuery
}

func (r *fixtureRepo) UserState(_ context.Context, _ c.ID) (State, error) {
	r.reads++
	if r.fail != nil {
		return State{}, r.fail
	}
	return r.state, nil
}
func (r *fixtureRepo) RecentBefore(_ context.Context, _, _ c.ID, n int) ([]string, error) {
	r.reads++
	if r.fail != nil {
		return nil, r.fail
	}
	if n > len(r.recent) {
		n = len(r.recent)
	}
	return r.recent[:n], nil
}
func (r *fixtureRepo) Memories(_ context.Context, q MemoryQuery) ([]Candidate, error) {
	r.reads++
	r.queries = append(r.queries, q)
	if r.fail != nil {
		return nil, r.fail
	}
	var out []Candidate
	for _, v := range r.candidates {
		if q.ThreadOnly && !threadContains(v.ThreadIDs, q.FocusThread) {
			continue
		}
		if q.EntityRole != "" || q.EntityLabel != "" {
			match := false
			for _, e := range v.Entities {
				if q.EntityRole != "" && e.Role == q.EntityRole {
					match = true
				}
				if q.EntityLabel != "" && e.Label != nil && *e.Label == q.EntityLabel {
					match = true
				}
			}
			if !match {
				continue
			}
		}
		if q.Since != nil && v.CreatedAt.Before(*q.Since) {
			continue
		}
		if q.LexicalTerm != "" && !strings.Contains(strings.ToLower(v.Text), q.LexicalTerm) {
			continue
		}
		out = append(out, v)
	}
	return out, nil
}
func (r *fixtureRepo) Messages(_ context.Context, q MessageQuery) ([]Candidate, error) {
	r.reads++
	r.windows = append(r.windows, q)
	if r.fail != nil {
		return nil, r.fail
	}
	var out []Candidate
	for _, v := range r.messages {
		if !v.CreatedAt.Before(q.From) && v.CreatedAt.Before(q.To) {
			out = append(out, v)
		}
	}
	return out, nil
}

type fakeEmbed struct {
	vector []float32
	model  string
	err    error
	calls  int
	meta   c.CallMeta
}

func (f *fakeEmbed) Embed(_ context.Context, req ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
	f.calls++
	f.meta = req.Meta
	if f.err != nil {
		return ports.EmbeddingResponse{}, f.err
	}
	return ports.EmbeddingResponse{Vector: f.vector, Model: f.model}, nil
}

func setup(t *testing.T) (*Service, *fixtureRepo, c.RetrievalRequest) {
	t.Helper()
	raw, err := os.ReadFile("testdata/retrieval.yaml")
	if err != nil {
		t.Fatal(err)
	}
	repo := &fixtureRepo{state: State{TZ: "America/New_York"}}
	service, err := New(repo, nil, fixtureRegistry{fixtureHandle{raw}}, fixtureUse{search: true})
	if err != nil {
		t.Fatal(err)
	}
	service.WithClock(func() time.Time { return time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC) })
	req := c.RetrievalRequest{UserID: c.NewID(), Turn: c.TurnView{TurnID: c.NewID(), MessageID: c.NewID(), Text: "situation at work", Lang: "en"}, MaxItems: 5}
	return service, repo, req
}
func memory(owner c.ID, id c.ID, text string, daysAgo int) Candidate {
	return Candidate{ID: id, UserID: owner, Source: SourceMemory, Kind: c.MemEpisode, Text: text, Epistemic: c.EpReported,
		Status: c.MemActive, Sensitivity: c.SensNormal, Importance: 0.8, EvidenceN: 2,
		CreatedAt: time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC).AddDate(0, 0, -daysAgo)}
}

func TestNewRequiresExplicitUsePolicy(t *testing.T) {
	s, r, _ := setup(t)
	if _, e := New(r, s.embed, s.bundles, nil); e == nil {
		t.Fatal("missing policy accepted")
	}
	if _, e := New(nil, nil, s.bundles, s.use); e == nil {
		t.Fatal("missing repository accepted")
	}
}
func TestTemporalExactWithTimezoneAndSubjectNoSemanticSubstitution(t *testing.T) {
	s, r, req := setup(t)
	req.Turn.Text = "What did I tell you last Tuesday about my boss?"
	// 2026-11-17 is Tuesday; 'last Tuesday' means the previous local Tuesday.
	from := time.Date(2026, 11, 10, 5, 0, 0, 0, time.UTC)
	r.messages = []Candidate{
		{ID: c.NewID(), UserID: req.UserID, Source: SourceMessage, Kind: c.MemEpisode, Text: "My boss yelled again", CreatedAt: from.Add(3 * time.Hour), Status: c.MemActive, Sensitivity: c.SensNormal, Epistemic: c.EpReported, EvidenceN: 1, Importance: 1},
		{ID: c.NewID(), UserID: req.UserID, Source: SourceMessage, Kind: c.MemEpisode, Text: "I cooked pasta", CreatedAt: from.Add(5 * time.Hour), Status: c.MemActive, Sensitivity: c.SensNormal, Epistemic: c.EpReported, EvidenceN: 1, Importance: 1},
	}
	r.candidates = []Candidate{memory(req.UserID, c.NewID(), "boss was unkind last month", 1)}
	result, e := s.Retrieve(context.Background(), req)
	if e != nil {
		t.Fatal(e)
	}
	if len(result.Items) != 1 || !result.Items[0].Verbatim || result.Items[0].ID != r.messages[0].ID || !*result.Meta.ExactMatchFound {
		t.Fatalf("not exact %+v", result)
	}
	if len(r.queries) != 0 || len(r.windows) != 1 || !r.windows[0].From.Equal(from) || !r.windows[0].To.Equal(from.Add(24*time.Hour)) {
		t.Fatalf("unexpected SQL temporal window: %+v", r.windows)
	}
	r.messages = nil
	result, e = s.Retrieve(context.Background(), req)
	if e != nil || len(result.Items) != 0 || result.Meta.ExactMatchFound == nil || *result.Meta.ExactMatchFound {
		t.Fatalf("fabricated exact recall: %+v %v", result, e)
	}
}
func TestHybridRankingDedupBudgetAndModelMatch(t *testing.T) {
	s, r, req := setup(t)
	emb := &fakeEmbed{vector: []float32{0, 3, 4}, model: "synthetic-v1"}
	s.embed = emb
	req.Turn.Text = "work stress"
	thread := c.NewID()
	req.FocusThread = &thread
	best := memory(req.UserID, c.NewID(), "work stress worries me every day", 1)
	best.Vector = []float32{0, 0.6, 0.8}
	best.ThreadIDs = []c.ID{thread}
	dup := memory(req.UserID, c.NewID(), "work stress worries me every day", 1)
	dup.Vector = []float32{0, 0.6, 0.8}
	older := memory(req.UserID, c.NewID(), "work stress from months ago", 160)
	older.Vector = []float32{0, 0.6, 0.8}
	wrong := memory(req.UserID, c.NewID(), "work unrelated unrelated", 1)
	wrong.Vector = []float32{1, 0} // dimension mismatch must not get cosine credit
	low := memory(req.UserID, c.NewID(), "unrelated text", 300)
	low.Vector = []float32{1, 0, 0}
	low.Importance = 0
	r.candidates = []Candidate{low, wrong, older, dup, best}
	result, e := s.Retrieve(context.Background(), req)
	if e != nil {
		t.Fatal(e)
	}
	if len(result.Items) < 1 || result.Items[0].ID != best.ID {
		t.Fatalf("incorrect ranking: %+v", result.Items)
	}
	seen := map[string]bool{}
	for _, v := range result.Items {
		if seen[v.Text] {
			t.Fatal("duplicate")
		}
		seen[v.Text] = true
		if approximateTokens(v.Text) > 60 {
			t.Fatalf("per-item token cap exceeded: %s", v.Text)
		}
	}
	if emb.calls != 1 || emb.meta.UserID != req.UserID || emb.meta.TurnID != req.Turn.TurnID || len(r.queries) == 0 || r.queries[0].Model != "synthetic-v1" || r.queries[0].Dimension != 3 {
		t.Fatalf("embedding integration absent: %+v %+v", emb, r.queries)
	}
}
func TestMissingEmbeddingUsesLexicalFallback(t *testing.T) {
	s, r, req := setup(t)
	m := memory(req.UserID, c.NewID(), "I have trouble with sleep lately", 0)
	m.Vector = nil
	r.candidates = []Candidate{m}
	req.Turn.Text = "trouble sleep lately"
	got, e := s.Retrieve(context.Background(), req)
	if e != nil || got.Meta.Degraded || len(got.Items) != 1 {
		t.Fatalf("missing embedding must not invent vector: %+v %v", got, e)
	}
}
func TestSensitiveHypothesisAndDeletionFilters(t *testing.T) {
	s, r, req := setup(t)
	req.Turn.Text = "worry at work"
	normal := memory(req.UserID, c.NewID(), "worry at work", 0)
	secret := memory(req.UserID, c.NewID(), "worry at work and more", 0)
	secret.Sensitivity = c.SensHighly
	stale := memory(req.UserID, c.NewID(), "worry at work", 0)
	stale.Status = c.MemStale
	naked := memory(req.UserID, c.NewID(), "worry at work", 0)
	naked.EvidenceN = 0
	deleted := memory(req.UserID, c.NewID(), "worry at work", 0)
	at := time.Now()
	deleted.DeletedAt = &at
	hyp := memory(req.UserID, c.NewID(), "worry at work", 0)
	hyp.Kind = c.MemHypothesis
	hyp.Status = c.MemSupported
	hyp.Epistemic = c.EpInferred
	r.candidates = []Candidate{normal, secret, stale, naked, deleted, hyp}
	got, e := s.Retrieve(context.Background(), req)
	if e != nil || len(got.Items) != 1 || got.Items[0].ID != normal.ID {
		t.Fatalf("unsafe candidate returned: %+v %v", got.Items, e)
	}
	req.IncludeSensitivity = []c.Sensitivity{c.SensHighly}
	got, e = s.Retrieve(context.Background(), req)
	if e != nil || len(got.Items) != 1 || got.Items[0].ID != secret.ID {
		t.Fatalf("explicit sensitivity scope failed: %+v %v", got.Items, e)
	}
}
func TestAnaphoraThreadAndAmbiguity(t *testing.T) {
	s, r, req := setup(t)
	req.Turn.Text = "it happened again, like before"
	focus := c.NewID()
	req.FocusThread = &focus
	other := memory(req.UserID, c.NewID(), "unrelated family trouble", 1)
	linked := memory(req.UserID, c.NewID(), "similar trouble last week", 3)
	linked.ThreadIDs = []c.ID{focus}
	r.candidates = []Candidate{other, linked}
	got, e := s.Retrieve(context.Background(), req)
	if e != nil || len(got.Items) == 0 || got.Items[0].ID != linked.ID {
		t.Fatalf("anaphora retrieval failed: %+v %v", got, e)
	}
	if len(got.Meta.Strategies) == 0 || got.Meta.Strategies[0] != "anaphora" {
		t.Fatalf("strategy selection: %+v", got.Meta)
	}
	req.FocusThread = nil
	got, e = s.Retrieve(context.Background(), req)
	if e != nil || !got.Meta.AmbiguousReference {
		t.Fatalf("missing anchor should be ambiguous: %+v %v", got.Meta, e)
	}
}
func TestEntityIsRoleScoped(t *testing.T) {
	s, r, req := setup(t)
	req.Turn.Text = "I talked about my boss"
	boss := memory(req.UserID, c.NewID(), "My boss raised his voice at work", 1)
	boss.Entities = []c.EntityRef{{Role: "boss"}}
	mom := memory(req.UserID, c.NewID(), "My mother visited", 1)
	mom.Entities = []c.EntityRef{{Role: "mother"}}
	r.candidates = []Candidate{boss, mom}
	got, e := s.Retrieve(context.Background(), req)
	if e != nil || len(got.Items) == 0 || got.Items[0].ID != boss.ID {
		t.Fatalf("entity mismatch: %+v %v", got, e)
	}
	selected := false
	for _, q := range r.queries {
		if q.EntityRole == "boss" {
			selected = true
		}
	}
	if !selected {
		t.Fatalf("entity query not owner-scoped: %+v", r.queries)
	}
}
func TestUsePolicyRejectsAndOwnerMismatch(t *testing.T) {
	s, r, req := setup(t)
	s.use = fixtureUse{search: false}
	got, e := s.Retrieve(context.Background(), req)
	if e != nil || !got.Meta.Degraded || r.reads != 0 {
		t.Fatalf("denied search queried private data: %+v %v", got, e)
	}
	s.use = fixtureUse{search: true}
	r.candidates = []Candidate{memory(c.NewID(), c.NewID(), "work stress", 0)}
	got, e = s.Retrieve(context.Background(), req)
	if e == nil || !got.Meta.Degraded || len(got.Items) != 0 {
		t.Fatalf("cross-user content leaked: %+v %v", got, e)
	}
	r.candidates = []Candidate{memory(req.UserID, c.NewID(), "work stress", 0)}
	s.use = fixtureUse{search: true, deny: map[c.ID]bool{r.candidates[0].ID: true}}
	got, e = s.Retrieve(context.Background(), req)
	if e == nil || !got.Meta.Degraded || len(got.Items) != 0 {
		t.Fatalf("forbidden source leaked: %+v %v", got, e)
	}
}
func TestMemoryPauseAndFailuresFailClosed(t *testing.T) {
	s, r, req := setup(t)
	r.state.MemoryPaused = true
	got, e := s.Retrieve(context.Background(), req)
	if e != nil || !got.Meta.Degraded || len(r.queries) > 0 {
		t.Fatalf("paused memory was searched: %+v %v", got, e)
	}
	r.state.MemoryPaused = false
	r.fail = errors.New("synthetic database failure")
	got, e = s.Retrieve(context.Background(), req)
	if e == nil || !got.Meta.Degraded || len(got.Items) > 0 {
		t.Fatalf("SQL outage leakage: %+v %v", got, e)
	}
	r.fail = nil
	ectx, cancel := context.WithCancel(context.Background())
	cancel()
	got, e = s.Retrieve(ectx, req)
	if e == nil || !got.Meta.Degraded {
		t.Fatalf("cancellation ignored: %+v %v", got, e)
	}
	s.embed = &fakeEmbed{err: errors.New("provider outage")}
	got, e = s.Retrieve(context.Background(), req)
	if e == nil || !got.Meta.Degraded {
		t.Fatalf("provider failure did not degrade: %+v %v", got, e)
	}
}
func TestNoveltyExcludeAndSecondHop(t *testing.T) {
	s, r, req := setup(t)
	req.Turn.Text = "longstanding work stress"
	text := "longstanding work stress was difficult yesterday"
	duplicate := memory(req.UserID, c.NewID(), text, 1)
	r.candidates = []Candidate{duplicate}
	r.recent = []string{text}
	got, e := s.Retrieve(context.Background(), req)
	if e != nil || len(got.Items) != 0 {
		t.Fatalf("novelty filter failed %+v %v", got, e)
	}
	r.recent = nil
	req.Exclude = []c.ID{duplicate.ID}
	got, e = s.Retrieve(context.Background(), req)
	if e != nil || len(got.Items) != 0 {
		t.Fatalf("exclude ids failed: %+v %v", got, e)
	}
	req.Exclude = nil
	req.Extra = &c.NeedsContext{Kind: c.ContextTopic, Query: "work stress", Why: c.ContextMissingBackground}
	got, e = s.Retrieve(context.Background(), req)
	if e != nil || len(got.Items) != 1 || got.Meta.Hops != 1 {
		t.Fatalf("second hop failed: %+v %v", got, e)
	}
	req.Extra.Query = strings.Repeat("word ", 13)
	got, e = s.Retrieve(context.Background(), req)
	if e == nil || !got.Meta.Degraded {
		t.Fatalf("oversized needs context accepted: %+v", got)
	}
}
func TestCorruptConfigIsRejected(t *testing.T) {
	s, _, req := setup(t)
	s.bundles = fixtureRegistry{fixtureHandle{[]byte("schema: retrieval.v1\nunknown: 1\n")}}
	got, e := s.Retrieve(context.Background(), req)
	if e == nil || !got.Meta.Degraded {
		t.Fatal("invalid pinned retrieval config accepted")
	}
}
func TestTemporalCalendarDST(t *testing.T) {
	// DST begins in New York on 2026-03-08; local Sunday lasts 23 hours.
	now := time.Date(2026, 3, 9, 16, 0, 0, 0, time.UTC)
	refs, e := TemporalReferences("What did I say yesterday?", now, "America/New_York")
	if e != nil || len(refs) != 1 || refs[0].To.Sub(refs[0].From) != 23*time.Hour {
		t.Fatalf("DST bounds wrong: %+v %v", refs, e)
	}
	_, e = TemporalReferences("yesterday", now, "Invalid/Timezone")
	if e == nil {
		t.Fatal("invalid timezone silently assumed")
	}
	refs, e = TemporalReferences("چه چیزی هفته پیش گفتم؟", now, "Asia/Tehran")
	if e != nil || len(refs) != 1 || refs[0].To.Sub(refs[0].From) != 7*24*time.Hour {
		t.Fatalf("Persian relative-week: %+v %v", refs, e)
	}
}
func TestPinnedOwnerBound(t *testing.T) {
	s, _, req := setup(t)
	handle := fixtureHandle{[]byte("invalid")}
	ctx := PinBundle(context.Background(), c.NewID(), handle)
	got, e := s.Retrieve(ctx, req)
	if e == nil || !got.Meta.Degraded {
		t.Fatal("cross-user pinned bundle accepted")
	}
}
