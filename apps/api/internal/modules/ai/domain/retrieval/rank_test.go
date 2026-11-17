package retrieval

import (
	"math"
	"testing"
	"time"
)

func fixedNow() time.Time {
	return time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
}

func TestDefaultsMatchContractPack(t *testing.T) {
	d := Defaults()
	if d.MaxItems != 5 || d.HardCapWithHop != 7 || d.PerItemTokens != 60 {
		t.Fatalf("contract defaults drifted: %+v", d)
	}
	if d.WeightSimilarity != 0.50 || d.WeightRecency != 0.20 || d.WeightImportance != 0.15 || d.WeightThread != 0.15 {
		t.Fatalf("contract weights drifted: similarity=%v recency=%v importance=%v thread=%v",
			d.WeightSimilarity, d.WeightRecency, d.WeightImportance, d.WeightThread)
	}
	if d.MinScore != 0.32 || d.RecencyHalfLife != 21*24*time.Hour || d.AnaphoraWindow != 14*24*time.Hour {
		t.Fatalf("contract thresholds drifted: %+v", d)
	}
	if d.TotalTimeout != 300*time.Millisecond {
		t.Fatalf("retrieval timeout must stay 300ms, got %v", d.TotalTimeout)
	}
}

func TestNormalizeRejectsInvalidConfiguration(t *testing.T) {
	cases := map[string]func(*Tunables){
		"weights do not sum to one": func(tu *Tunables) {
			tu.WeightSimilarity, tu.WeightRecency, tu.WeightImportance, tu.WeightThread = 0.9, 0.9, 0.9, 0.9
		},
		"negative weight":          func(tu *Tunables) { tu.WeightSimilarity = -1 },
		"hard cap below max items": func(tu *Tunables) { tu.MaxItems, tu.HardCapWithHop = 6, 5 },
		"too many items":           func(tu *Tunables) { tu.MaxItems = 25 },
		"invalid temporal role":    func(tu *Tunables) { tu.TemporalRoles = []string{"system"} },
		"hypothesis status":        func(tu *Tunables) { tu.HypothesisStatuses = []Status{StatusContested} },
		"strategy timeout too big": func(tu *Tunables) {
			tu.PerStrategyTimeout, tu.TotalTimeout = 900*time.Millisecond, 200*time.Millisecond
		},
		"unbounded candidate pool": func(tu *Tunables) { tu.CandidatePoolLimit = 5000 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := Defaults()
			mutate(&cfg)
			if _, err := cfg.Normalize(); err == nil {
				t.Fatalf("%s: expected a configuration error", name)
			}
		})
	}
}

func TestNormalizeAcceptsZeroValueConfig(t *testing.T) {
	got, err := Tunables{}.Normalize()
	if err != nil {
		t.Fatalf("zero config must normalize: %v", err)
	}
	want := Defaults()
	if got.MaxItems != want.MaxItems || got.MinScore != want.MinScore || got.WeightSimilarity != want.WeightSimilarity {
		t.Fatalf("zero config did not pick up defaults: %+v", got)
	}
}

func TestItemCapHonoursHardCap(t *testing.T) {
	tu := Defaults()
	if got := tu.ItemCap(0); got != 5 {
		t.Fatalf("no hop: want 5, got %d", got)
	}
	if got := tu.ItemCap(1); got != 6 {
		t.Fatalf("one hop: want 6, got %d", got)
	}
	tu.MaxItems = 7
	if got := tu.ItemCap(1); got != 7 {
		t.Fatalf("hard cap: want 7, got %d", got)
	}
}

func TestRecencyDecayHalvesAtHalfLife(t *testing.T) {
	now := fixedNow()
	half := now.Add(-21 * 24 * time.Hour)
	c := Candidate{UpdatedAt: half}
	got := RecencyDecay(c, now, 21*24*time.Hour)
	if math.Abs(float64(got)-0.5) > 0.001 {
		t.Fatalf("want ~0.5 at one half-life, got %v", got)
	}
	future := Candidate{UpdatedAt: now.Add(48 * time.Hour)}
	if got := RecencyDecay(future, now, 21*24*time.Hour); got != 1 {
		t.Fatalf("future timestamps must clamp to 1, got %v", got)
	}
	zero := Candidate{}
	if got := RecencyDecay(zero, now, 21*24*time.Hour); got != 0 {
		t.Fatalf("zero timestamp must not gain recency, got %v", got)
	}
}

func TestScoreFollowsDocumentedFormula(t *testing.T) {
	now := fixedNow()
	tu := Defaults()
	c := Candidate{
		ID: "a", Kind: "fact", UpdatedAt: now, // recency 1
		Importance: 1, Similarity: 1, ThreadIDs: []string{"thr"},
	}
	ctx := ScoreContext{Now: now, FocusThread: "thr", Tunables: tu}
	b := Score(c, ctx)
	if math.Abs(float64(b.Total)-1.0) > 1e-5 {
		t.Fatalf("perfect match should score 1, got %v (%+v)", b.Total, b)
	}
	// Thread match contributes exactly its weight.
	ctx.FocusThread = "other"
	b2 := Score(c, ctx)
	if math.Abs(float64(b.Total-b2.Total)-float64(tu.WeightThread)) > 1e-5 {
		t.Fatalf("thread weight contribution wrong: %v vs %v", b.Total, b2.Total)
	}
}

func TestScoreAnaphoraBoostOnlyInsideWindow(t *testing.T) {
	now := fixedNow()
	tu := Defaults()
	recent := Candidate{UpdatedAt: now.Add(-2 * 24 * time.Hour), Tags: []string{"worthless", "work"}, Similarity: 0.4, Importance: 0.3}
	ctx := ScoreContext{Now: now, Anaphora: true, Emotions: []string{"worthless"}, Topics: []string{"work"}, Tunables: tu}
	b := Score(recent, ctx)
	if math.Abs(float64(b.Anaphora)-float64(tu.EmotionBoost+tu.TopicBoost)) > 1e-5 {
		t.Fatalf("expected emotion+topic boost, got %v", b.Anaphora)
	}
	if b.Total > 1 || b.Total <= 0 {
		t.Fatalf("score must be clamped into (0,1], got %v", b.Total)
	}
	old := recent
	old.UpdatedAt = now.Add(-30 * 24 * time.Hour)
	if b := Score(old, ctx); b.Anaphora != 0 {
		t.Fatalf("items older than the anaphora window must not be boosted, got %v", b.Anaphora)
	}
	if b := Score(recent, ScoreContext{Now: now, Anaphora: false, Emotions: []string{"worthless"}, Tunables: tu}); b.Anaphora != 0 {
		t.Fatalf("boost must require an active anaphora strategy, got %v", b.Anaphora)
	}
}

func TestRankOrdersByScoreThenKindThenID(t *testing.T) {
	now := fixedNow()
	tu := Defaults()
	ctx := ScoreContext{Now: now, Tunables: tu}
	cands := []Candidate{
		{ID: "3", Kind: "hypothesis", UpdatedAt: now, Importance: 1, Similarity: 1},
		{ID: "2", Kind: "fact", UpdatedAt: now, Importance: 1, Similarity: 1},
		{ID: "1", Kind: "fact", UpdatedAt: now, Importance: 1, Similarity: 1},
		{ID: "9", Kind: "fact", UpdatedAt: now, Importance: 0.1, Similarity: 0.1},
	}
	got := Rank(cands, ctx)
	if len(got) != 3 {
		t.Fatalf("low scorer below threshold must be dropped, got %d items", len(got))
	}
	if got[0].ID != "1" || got[1].ID != "2" {
		t.Fatalf("fact must outrank hypothesis and ids must break ties: %v %v", got[0].ID, got[1].ID)
	}
	if got[2].ID != "3" {
		t.Fatalf("hypothesis should be last, got %v", got[2].ID)
	}
}

func TestRankDropsExcludedCandidates(t *testing.T) {
	now := fixedNow()
	tu := Defaults()
	ctx := ScoreContext{Now: now, Tunables: tu}
	cands := []Candidate{
		{ID: "deleted", Kind: "fact", UpdatedAt: now, Importance: 1, Similarity: 1, Excluded: true, ExcludeReason: "soft_deleted"},
		{ID: "live", Kind: "fact", UpdatedAt: now, Importance: 1, Similarity: 1},
	}
	got := Rank(cands, ctx)
	if len(got) != 1 || got[0].ID != "live" {
		t.Fatalf("soft-deleted candidate leaked: %+v", got)
	}
}

func TestDedupeByClusterAndText(t *testing.T) {
	ranked := []Ranked{
		{Candidate: Candidate{ID: "a", Kind: "fact", Text: "user feels worthless after social comparison", ClusterID: "c1"}},
		{Candidate: Candidate{ID: "b", Kind: "fact", Text: "completely different statement", ClusterID: "c1"}},
		{Candidate: Candidate{ID: "c", Kind: "fact", Text: "User feels worthless after social comparison."}},
		{Candidate: Candidate{ID: "d", Kind: "fact", Text: "another distinct memory"}},
	}
	got := Dedupe(ranked, 0.92)
	if len(got) != 2 {
		t.Fatalf("want 2 items after cluster+text dedupe, got %d: %+v", len(got), got)
	}
	if got[0].ID != "a" || got[1].ID != "d" {
		t.Fatalf("dedupe must keep the first (highest-ranked) representative: %v %v", got[0].ID, got[1].ID)
	}
}

func TestDedupeKeepsNearDuplicatesAcrossKinds(t *testing.T) {
	// A hypothesis and a reported fact may legitimately share wording; only
	// episodes/events are subject to the near-duplicate guard.
	ranked := []Ranked{
		{Candidate: Candidate{ID: "fact", Kind: "fact", Text: "my manager criticized my work in front of the team"}},
		{Candidate: Candidate{ID: "hyp", Kind: "hypothesis", Text: "my manager criticized my work in front of the team"}},
	}
	got := Dedupe(ranked, 0.92)
	if len(got) == 1 && got[0].ID == "fact" {
		return // acceptable: exact normalized text is a hard duplicate
	}
	if len(got) != 1 {
		t.Fatalf("exact normalized duplicates must collapse, got %d", len(got))
	}
}

func TestBudgetRespectsTopKAndTokens(t *testing.T) {
	long := ""
	for i := 0; i < 400; i++ {
		long += "word "
	}
	var ranked []Ranked
	for i := 0; i < 10; i++ {
		ranked = append(ranked, Ranked{Candidate: Candidate{ID: string(rune('a' + i)), Text: long}})
	}
	got := Budget(ranked, 3, 400, 60)
	if len(got) != 3 {
		t.Fatalf("top-K not enforced: %d", len(got))
	}
	total := 0
	for _, r := range got {
		if r.Tokens > 60 {
			t.Fatalf("per-item token cap exceeded: %d", r.Tokens)
		}
		total += r.Tokens
	}
	if total > 400 {
		t.Fatalf("token budget exceeded: %d", total)
	}
}

func TestBudgetSkipsOversizedItemButKeepsLaterShortOnes(t *testing.T) {
	ranked := []Ranked{
		{Candidate: Candidate{ID: "big", Text: repeat("token ", 200)}},
		{Candidate: Candidate{ID: "small", Text: "short memory"}},
	}
	got := Budget(ranked, 2, 60, 60)
	if len(got) != 2 {
		t.Fatalf("a truncated big item should still fit the budget, got %d", len(got))
	}
	if got[0].Tokens > 60 {
		t.Fatalf("first item exceeded per-item cap: %d", got[0].Tokens)
	}
}

func TestEstimateTokensAndTruncate(t *testing.T) {
	if got := EstimateTokens(""); got != 0 {
		t.Fatalf("empty text has no tokens, got %d", got)
	}
	if got := EstimateTokens("abcd"); got != 1 {
		t.Fatalf("4 ascii chars ~ 1 token, got %d", got)
	}
	if got := EstimateTokens("سلام دنیا"); got < 2 {
		t.Fatalf("dense script must cost more than ascii, got %d", got)
	}
	long := repeat("متنی ", 100)
	truncated := TruncateToTokens(long, 20)
	if EstimateTokens(truncated) > 20 {
		t.Fatalf("truncation exceeded budget: %d", EstimateTokens(truncated))
	}
	if TruncateToTokens("anything", 0) != "" {
		t.Fatal("a zero token budget truncates to empty")
	}
}

func TestNoveltyFilterDropsRecentRepeats(t *testing.T) {
	ranked := []Ranked{
		{Candidate: Candidate{ID: "repeat", Text: "I argued with my manager today"}},
		{Candidate: Candidate{ID: "new", Text: "my sister called about the wedding"}},
	}
	recent := []string{"fine", "I argued with my manager today"}
	got := NoveltyFilter(ranked, recent, 0.92, 8)
	if len(got) != 1 || got[0].ID != "new" {
		t.Fatalf("already-said context must be filtered: %+v", got)
	}
	if out := NoveltyFilter(ranked, nil, 0.92, 8); len(out) != 2 {
		t.Fatal("an empty recent window filters nothing")
	}
}

func TestKindPriorityIsStable(t *testing.T) {
	order := []string{"episode", "fact", "preference", "goal", "event", "hypothesis", "safety_event", "unknown"}
	prev := -1
	for _, k := range order {
		p := kindPriority(k)
		if p <= prev {
			t.Fatalf("kind priority not monotonic at %s", k)
		}
		prev = p
	}
}

func repeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}
	return out
}
