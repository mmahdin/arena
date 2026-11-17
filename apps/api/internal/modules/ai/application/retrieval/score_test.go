package retrieval

import (
	"math"
	"testing"
	"time"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

func scoreFixture(t *testing.T) (Config, time.Time, c.ID) {
	t.Helper()
	cfg := loadTestConfig(t)
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	return cfg, now, c.NewID()
}

func itemAt(user c.ID, at time.Time, importance float32) c.MemoryItem {
	return c.MemoryItem{
		ID: c.NewID(), UserID: user, Kind: c.MemEpisode, Content: "content", Epistemic: c.EpReported,
		Status: c.MemActive, Sensitivity: c.SensNormal, Importance: importance, CreatedAt: at, UpdatedAt: at,
	}
}

func TestScoreComponentsAreBoundedAndWeighted(t *testing.T) {
	cfg, now, user := scoreFixture(t)
	item := itemAt(user, now, 1)
	item.Importance = 1

	// Fresh item, importance 1, thread match, similarity 1 => every component at
	// its maximum, so the weighted score must be exactly 1.
	full := score(cfg, candidate{Item: &item, Similarity: 1, ThreadMatch: true}, now)
	if math.Abs(full-1) > 1e-9 {
		t.Fatalf("maximum candidate should score 1, got %v", full)
	}
	// Nothing available at all => 0, never NaN.
	zero := score(cfg, candidate{Item: nil}, now)
	if zero != 0 {
		t.Fatalf("empty candidate scored %v", zero)
	}
	// A stale item with low importance and no similarity is strictly below a
	// fresh, important one.
	staleOld := itemAt(user, now.AddDate(-1, 0, 0), 0.05)
	if got := score(cfg, candidate{Item: &staleOld}, now); got >= full {
		t.Fatalf("stale item did not score lower: %v", got)
	}
}

// A missing component must contribute exactly zero, never be renormalized away:
// otherwise every recent memory would clear the relevance floor on recency alone.
func TestScoreMissingComponentsContributeZero(t *testing.T) {
	cfg, now, user := scoreFixture(t)
	item := itemAt(user, now, 1)
	// Fresh + importance 1, no similarity, no thread match.
	got := score(cfg, candidate{Item: &item}, now)
	want := cfg.Weights.Recency + cfg.Weights.Importance
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("missing similarity/thread components must score zero: got %v want %v", got, want)
	}
	if got >= 1 {
		t.Fatal("a candidate without similarity evidence must not reach the maximum")
	}
	// A stale, unimportant item with no similarity signal falls through the
	// configured floor, which is what prevents over-retrieval.
	stale := itemAt(user, now.AddDate(0, 0, -400), 0.01)
	if s := score(cfg, candidate{Item: &stale}, now); s >= cfg.MinScore {
		t.Fatalf("an irrelevant item cleared min_score: %v", s)
	}
}

func TestRecencyDecayHalfLife(t *testing.T) {
	halfLife := 21 * 24 * time.Hour
	now := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	if v := recencyDecay(now, now, halfLife); v != 1 {
		t.Fatalf("age zero should decay to 1, got %v", v)
	}
	if v := recencyDecay(now, now.Add(-halfLife), halfLife); math.Abs(v-0.5) > 1e-9 {
		t.Fatalf("one half-life should decay to 0.5, got %v", v)
	}
	if v := recencyDecay(now, now.Add(-2*halfLife), halfLife); math.Abs(v-0.25) > 1e-9 {
		t.Fatalf("two half-lives should decay to 0.25, got %v", v)
	}
	if v := recencyDecay(now, now.Add(time.Hour), halfLife); v != 1 {
		t.Fatalf("future timestamps must clamp to 1, got %v", v)
	}
	if v := recencyDecay(now, time.Time{}, halfLife); v != 0 {
		t.Fatalf("unknown time must score 0, got %v", v)
	}
	if v := recencyDecay(now, now, 0); v != 0 {
		t.Fatalf("invalid half-life must score 0, got %v", v)
	}
}

func TestCosine(t *testing.T) {
	cases := []struct {
		a, b []float32
		want float64
		ok   bool
	}{
		{[]float32{1, 0}, []float32{1, 0}, 1, true},
		{[]float32{1, 0}, []float32{0, 1}, 0, true},
		{[]float32{1, 0}, []float32{-1, 0}, 0, true}, // negative similarity clamps to 0
		{[]float32{3, 4}, []float32{6, 8}, 1, true},
		{[]float32{1, 0}, []float32{1, 0, 0}, 0, false},
		{[]float32{}, []float32{}, 0, false},
		{[]float32{0, 0}, []float32{1, 0}, 0, false},
		{[]float32{float32(math.NaN()), 0}, []float32{1, 0}, 0, false},
		{[]float32{float32(math.Inf(1)), 0}, []float32{1, 0}, 0, false},
	}
	for i, tc := range cases {
		got, ok := cosine(tc.a, tc.b)
		if ok != tc.ok {
			t.Fatalf("case %d ok=%v want %v", i, ok, tc.ok)
		}
		if ok && math.Abs(got-tc.want) > 1e-6 {
			t.Fatalf("case %d got %v want %v", i, got, tc.want)
		}
	}
}

func TestTruncateAndEstimateTokens(t *testing.T) {
	if got := truncateWords("one two three", 5); got != "one two three" {
		t.Fatalf("unexpected truncation: %q", got)
	}
	if got := truncateWords("one two three", 2); got != "one two…" {
		t.Fatalf("expected ellipsis marker: %q", got)
	}
	if got := truncateWords("single", 0); got != "single" {
		t.Fatalf("zero budget must not drop content silently: %q", got)
	}
	if got := estimateTokens(""); got != 0 {
		t.Fatalf("empty text should be zero tokens, got %d", got)
	}
	// The estimate is the larger of word count and bytes/4.
	if got := estimateTokens("abcdefgh"); got != 2 {
		t.Fatalf("expected ceil(bytes/4)=2, got %d", got)
	}
	if got := estimateTokens("a b c d e"); got != 5 {
		t.Fatalf("expected word count to dominate, got %d", got)
	}
}

func TestAgeLabel(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		at   time.Time
		want string
	}{
		{now.Add(-30 * time.Second), "0m"},
		{now.Add(-5 * time.Minute), "5m"},
		{now.Add(-3 * time.Hour), "3h"},
		{now.Add(-50 * time.Hour), "2d"},
		{now.Add(time.Hour), "0m"},
		{time.Time{}, "unknown"},
	} {
		if got := ageLabel(now, tc.at); got != tc.want {
			t.Fatalf("ageLabel(%s)=%s want %s", tc.at, got, tc.want)
		}
	}
}

func TestRankOrderingIsDeterministicOnTies(t *testing.T) {
	cfg, now, user := scoreFixture(t)
	cfg.MinScore = 0
	var cands []candidate
	for i := 0; i < 5; i++ {
		item := itemAt(user, now, 0.5)
		cands = append(cands, candidate{Item: &item})
	}
	first := rank(cfg, cands, nil, now, 5, 100000)
	second := rank(cfg, cands, nil, now, 5, 100000)
	if len(first) != len(second) {
		t.Fatal("ranking is not stable")
	}
	for i := range first {
		if first[i].ID != second[i].ID {
			t.Fatalf("tie-break is not deterministic at %d", i)
		}
	}
}

func TestRankRespectsTokenBudget(t *testing.T) {
	cfg, now, user := scoreFixture(t)
	cfg.MinScore = 0
	item := itemAt(user, now, 0.9)
	item.Content = "one two three four five six seven eight nine ten"
	cands := []candidate{{Item: &item}, {Item: ptr(itemAt(user, now, 0.8))}}
	// A budget smaller than the first item still returns one item: an empty
	// result would be worse than a bounded one, so the first is always kept.
	out := rank(cfg, cands, nil, now, 5, 1)
	if len(out) != 1 {
		t.Fatalf("expected exactly one item under a tiny budget, got %d", len(out))
	}
	// With enough budget for one item only, the second is skipped.
	one := estimateTokens(truncateWords("one two three four five six seven eight nine ten", cfg.PerItemTok))
	out = rank(cfg, cands, nil, now, 5, one)
	if len(out) != 1 {
		t.Fatalf("budget did not limit the result to one item: %d", len(out))
	}
}

func TestMemoryViewDropsUnusableRecords(t *testing.T) {
	cfg, now, user := scoreFixture(t)
	empty := itemAt(user, now, 0.9)
	empty.Content = "   "
	if _, ok := memoryView(cfg, candidate{Item: &empty}, now, 0.5); ok {
		t.Fatal("empty content must not become a context item")
	}
	invalidStatus := itemAt(user, now, 0.9)
	invalidStatus.Status = "not_a_status"
	if _, ok := memoryView(cfg, candidate{Item: &invalidStatus}, now, 0.5); ok {
		t.Fatal("invalid status must not become a context item")
	}
	msg := messageRecord{}
	if _, ok := memoryView(cfg, candidate{Message: &msg}, now, 0.5); ok {
		t.Fatal("a message without an id must be dropped")
	}
}

func TestValidFromIsExposed(t *testing.T) {
	cfg, now, user := scoreFixture(t)
	item := itemAt(user, now, 0.9)
	from := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)
	item.ValidFrom = &from
	view, ok := memoryView(cfg, candidate{Item: &item}, now, 0.5)
	if !ok || view.ValidAt == nil || *view.ValidAt != "2025-01-02" {
		t.Fatalf("valid_from must be rendered as an ISO date: %+v", view)
	}
}

// ---- Lexical extraction ----

func TestLexicalQueryExtraction(t *testing.T) {
	q := buildLexicalQuery("What did I tell you last Tuesday about my boss and my sister Lena?")
	if !q.RecallVerb {
		t.Fatal("recall verb not detected")
	}
	if !contains(q.Tokens, "boss") || !contains(q.Tokens, "sister") {
		t.Fatalf("keyword tokens missing: %v", q.Tokens)
	}
	if contains(q.Tokens, "the") || contains(q.Tokens, "about") {
		t.Fatalf("stop words leaked into the query: %v", q.Tokens)
	}
	if len(q.Roles) != 2 || q.Roles[0] != "boss" || q.Roles[1] != "sister" {
		t.Fatalf("roles not extracted in sorted order: %v", q.Roles)
	}
	if q.Anaphora {
		t.Fatal("a plain recall question is not an anaphora")
	}
	q2 := buildLexicalQuery("I feel anxious again, the same feeling as before")
	if !q2.Anaphora {
		t.Fatal("anaphora marker missed")
	}
	if q2.Emotion != "anxiety" {
		t.Fatalf("emotion label not recognized: %q", q2.Emotion)
	}
	if len(q2.Tokens) > 12 {
		t.Fatalf("query is not bounded: %d tokens", len(q2.Tokens))
	}
}

func TestLexicalQueryDoesNotGuessPersonNames(t *testing.T) {
	// v1.2 leaves person alias/name matching [OPEN]; a capitalized proper noun
	// must never become an entity role.
	q := buildLexicalQuery("I argued with Lena yesterday")
	if len(q.Roles) != 0 {
		t.Fatalf("invented an entity role from a name: %v", q.Roles)
	}
}

func TestOverlappingTerms(t *testing.T) {
	hits := overlappingTerms("I felt anxious about my sister's visit", []string{"anxious", "sister", "boss"})
	if len(hits) != 2 || !contains(hits, "anxious") || !contains(hits, "sister") {
		t.Fatalf("unexpected hits: %v", hits)
	}
	if len(overlappingTerms("", []string{"anything"})) != 0 {
		t.Fatal("empty text matched")
	}
	// Two-way four-character prefix matching only; short tokens never fuzzily
	// match arbitrary words.
	if len(overlappingTerms("catastrophe", []string{"cat"})) != 0 {
		t.Fatal("a three-character token must not fuzzy match")
	}
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

// ---- Configuration validation ----

func TestLoadConfigMatchesSyntheticBundle(t *testing.T) {
	cfg := loadTestConfig(t)
	if cfg.MaxItems != 5 || cfg.HardCap != 7 || cfg.PerItemTok != 60 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if math.Abs(cfg.Weights.Similarity-0.5) > 1e-9 || math.Abs(cfg.HalfLife.Hours()-21*24) > 1e-9 {
		t.Fatalf("unexpected weights or half-life: %+v", cfg)
	}
	if len(cfg.Hypotheses.IncludeStatuses) != 2 || !cfg.Hypotheses.RequireThreadMatch {
		t.Fatalf("hypothesis policy not decoded: %+v", cfg.Hypotheses)
	}
	if cfg.ContextBudget != 1500 || cfg.Synthetic != true {
		t.Fatalf("budget/synthetic flag not decoded: %+v", cfg)
	}
	if cfg.Ref.Version != "kb@test" {
		t.Fatalf("bundle reference not pinned: %+v", cfg.Ref)
	}
}

func TestLoadConfigRejectsUnsafeDocuments(t *testing.T) {
	base := map[string][]byte{
		"config/retrieval.yaml": []byte(retrievalYAML),
		"config/budgets.yaml":   []byte(budgetsYAML),
	}
	mutations := map[string]func(map[string][]byte){
		"weights do not sum to one": func(files map[string][]byte) {
			files["config/retrieval.yaml"] = replaceOnce(files["config/retrieval.yaml"], "similarity: 0.5", "similarity: 0.9")
		},
		"negative weight": func(files map[string][]byte) {
			files["config/retrieval.yaml"] = replaceOnce(files["config/retrieval.yaml"], "similarity: 0.5", "similarity: -0.5")
			files["config/retrieval.yaml"] = replaceOnce(files["config/retrieval.yaml"], "recency: 0.2", "recency: 1.2")
		},
		"hard cap below max items": func(files map[string][]byte) {
			files["config/retrieval.yaml"] = replaceOnce(files["config/retrieval.yaml"], "hard_cap_with_hop: 7", "hard_cap_with_hop: 3")
		},
		"max items above contract cap": func(files map[string][]byte) {
			files["config/retrieval.yaml"] = replaceOnce(files["config/retrieval.yaml"], "max_items: 5", "max_items: 9")
		},
		"no thread match for hypotheses": func(files map[string][]byte) {
			files["config/retrieval.yaml"] = replaceOnce(files["config/retrieval.yaml"], "require_thread_match: true", "require_thread_match: false")
		},
		"active status allowed for hypotheses": func(files map[string][]byte) {
			files["config/retrieval.yaml"] = replaceOnce(files["config/retrieval.yaml"], "- proposed", "- active")
		},
		"unknown tz source": func(files map[string][]byte) {
			files["config/retrieval.yaml"] = replaceOnce(files["config/retrieval.yaml"], "tz_source: user_state", "tz_source: server_utc")
		},
		"unknown yaml field": func(files map[string][]byte) {
			files["config/retrieval.yaml"] = append(files["config/retrieval.yaml"], []byte("\nregenerate_scores: true\n")...)
		},
		"wrong schema": func(files map[string][]byte) {
			files["config/retrieval.yaml"] = replaceOnce(files["config/retrieval.yaml"], "schema: retrieval.v1", "schema: retrieval.v0")
		},
		"zero context budget": func(files map[string][]byte) {
			files["config/budgets.yaml"] = replaceOnce(files["config/budgets.yaml"], "wctx_target: 1500", "wctx_target: 0")
		},
		"timeout below strategy": func(files map[string][]byte) {
			files["config/retrieval.yaml"] = replaceOnce(files["config/retrieval.yaml"], "total_ms: 300", "total_ms: 100")
		},
		"invalid role": func(files map[string][]byte) {
			files["config/retrieval.yaml"] = replaceOnce(files["config/retrieval.yaml"], "- user", "- spectator")
		},
		"two yaml documents": func(files map[string][]byte) {
			files["config/retrieval.yaml"] = append(files["config/retrieval.yaml"], []byte("\n---\nschema: retrieval.v1\n")...)
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			files := map[string][]byte{}
			for k, v := range base {
				files[k] = append([]byte(nil), v...)
			}
			mutate(files)
			reg := &fakeRegistry{bundle: fakeBundle{files: files}}
			if _, err := LoadConfig(t.Context(), reg, c.NewID()); err == nil {
				t.Fatal("unsafe configuration accepted")
			}
		})
	}
}

func TestLoadConfigRequiresRegistryAndOwner(t *testing.T) {
	if _, err := LoadConfig(t.Context(), nil, c.NewID()); err == nil {
		t.Fatal("nil registry accepted")
	}
	if _, err := LoadConfig(t.Context(), testRegistry(), c.ID{}); err == nil {
		t.Fatal("missing owner accepted")
	}
}

func replaceOnce(in []byte, old, new string) []byte {
	s := string(in)
	idx := len(s)
	for i := 0; i+len(old) <= len(s); i++ {
		if s[i:i+len(old)] == old {
			idx = i
			break
		}
	}
	if idx == len(s) && (len(old) == 0 || idx+len(old) > len(s)) {
		return in
	}
	return []byte(s[:idx] + new + s[idx+len(old):])
}
