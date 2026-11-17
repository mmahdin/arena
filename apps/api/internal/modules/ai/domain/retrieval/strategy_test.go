package retrieval

import (
	"math"
	"testing"
	"time"
)

func TestPlanSelectsHybridByDefault(t *testing.T) {
	p := Plan(PlanInput{Text: "I feel tired after work"})
	if len(p.Strategies) != 1 || p.Strategies[0] != StrategyHybrid {
		t.Fatalf("default plan must be hybrid-only, got %v", p.Strategies)
	}
	if p.Ambiguous || p.Temporal || p.Anaphora {
		t.Fatalf("no special signals expected: %+v", p)
	}
}

func TestPlanTemporalRequiresWindowAndRecallVerb(t *testing.T) {
	w := &Window{Phrase: "last tuesday", From: time.Now().Add(-8 * 24 * time.Hour), To: time.Now()}
	p := Plan(PlanInput{Text: "what did I tell you last Tuesday", Window: w})
	if !p.Temporal {
		t.Fatal("temporal + recall verb must select temporal_exact")
	}
	if p.Strategies[0] != StrategyTemporal {
		t.Fatalf("temporal must be evaluated first, got %v", p.Strategies)
	}
	// A window without a recall verb is not exact recall.
	p = Plan(PlanInput{Text: "yesterday was rough", Window: w})
	if p.Temporal {
		t.Fatal("a window alone must not force exact recall")
	}
	if p.Strategies[0] != StrategyHybrid {
		t.Fatalf("expected hybrid, got %v", p.Strategies)
	}
	// Forced temporal (explicit request strategy) always wins.
	p = Plan(PlanInput{Text: "yesterday", Window: w, ForceTemporal: true})
	if !p.Temporal {
		t.Fatal("ForceTemporal must set the temporal flag")
	}
}

func TestPlanAnaphoraWithoutThreadIsAmbiguous(t *testing.T) {
	p := Plan(PlanInput{Text: "it's the same feeling I told you about before"})
	if !p.Anaphora {
		t.Fatal("expected anaphora detection")
	}
	if !p.Ambiguous {
		t.Fatal("a reference with no thread and no theme is ambiguous: the Brain must clarify")
	}
	if p.Strategies[0] != StrategyAnaphora {
		t.Fatalf("anaphora first, got %v", p.Strategies)
	}
	p = Plan(PlanInput{Text: "same feeling as before", FocusThread: "thr-1"})
	if p.Ambiguous {
		t.Fatal("a focus thread resolves the reference")
	}
	found := false
	for _, s := range p.Strategies {
		if s == StrategyThread {
			found = true
		}
	}
	if !found {
		t.Fatalf("anaphora with a focus thread must also run thread retrieval: %v", p.Strategies)
	}
}

func TestPlanSecondHopAddsExactlyOneTargetedStrategy(t *testing.T) {
	p := Plan(PlanInput{Text: "what about her", Hop: 1, NeedsKind: "entity"})
	found := false
	for _, s := range p.Strategies {
		if s == StrategyEntity {
			found = true
		}
	}
	if !found {
		t.Fatalf("entity hop must add the entity strategy: %v", p.Strategies)
	}
	if len(p.Strategies) > 3 {
		t.Fatalf("hop must stay bounded, got %v", p.Strategies)
	}
}

func TestPlanHonoursRequestedStrategies(t *testing.T) {
	p := Plan(PlanInput{Requested: []string{"entity", "thread"}, Text: "x", FocusThread: "t"})
	if len(p.Strategies) != 2 {
		t.Fatalf("forced strategies only, got %v", p.Strategies)
	}
	if p.Strategies[0] != StrategyThread || p.Strategies[1] != StrategyEntity {
		t.Fatalf("forced strategies must follow canonical order, got %v", p.Strategies)
	}
	if IsKnownStrategy("pgvector_semantic") {
		t.Fatal("unknown strategies must be rejected")
	}
}

func TestAnaphoraDetectionMultilingual(t *testing.T) {
	positives := []string{
		"it's the same feeling again",
		"like last time, I froze",
		"that thing I told you about came back",
		"یادت هست چه حسی داشتم",
		"همان موضوع قبلی",
		"بار قبل هم همینطور بود",
	}
	for _, text := range positives {
		if !HasAnaphora(text) {
			t.Fatalf("expected anaphora in %q", text)
		}
	}
	negatives := []string{"I went to the gym", "my week was calm", ""}
	for _, text := range negatives {
		if HasAnaphora(text) {
			t.Fatalf("unexpected anaphora in %q", text)
		}
	}
}

func TestEntityMentions(t *testing.T) {
	cases := map[string][2]string{
		"my sister Lena called me":      {"sister", "Lena"},
		"my boss was rude":              {"boss", ""},
		"خواهرم امروز زنگ زد":           {"خواهرم", ""},
		"my partner named Sam is upset": {"partner", "Sam"},
	}
	for text, want := range cases {
		got := EntityMentions(text)
		if len(got) == 0 {
			t.Fatalf("%q: expected an entity mention", text)
		}
		if got[0][0] != want[0] {
			t.Fatalf("%q: role=%q want %q", text, got[0][0], want[0])
		}
		if want[1] != "" && got[0][1] != want[1] {
			t.Fatalf("%q: label=%q want %q", text, got[0][1], want[1])
		}
	}
	if len(EntityMentions("the weather is grey")) != 0 {
		t.Fatal("no entity expected")
	}
}

func TestEntityQueryFromHop(t *testing.T) {
	q := EntityQueryFrom("unrelated text", "my sister")
	if q.Role != "sister" {
		t.Fatalf("hop query should resolve the role, got %+v", q)
	}
	q = EntityQueryFrom("my boss is harsh", "")
	if q.Role != "boss" {
		t.Fatalf("turn text should resolve the role, got %+v", q)
	}
}

func TestCosineSimilarity(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{1, 0, 0}
	if s, err := CosineSimilarity(a, b); err != nil || math.Abs(float64(s)-1) > 1e-6 {
		t.Fatalf("identical vectors: s=%v err=%v", s, err)
	}
	c := []float32{0, 1, 0}
	if s, err := CosineSimilarity(a, c); err != nil || s != 0 {
		t.Fatalf("orthogonal vectors: s=%v err=%v", s, err)
	}
	if s, err := CosineSimilarity(a, []float32{-1, 0, 0}); err != nil || s != 0 {
		t.Fatalf("opposite vectors must clamp to 0, got %v err=%v", s, err)
	}
	if _, err := CosineSimilarity(a, []float32{1, 0}); err == nil {
		t.Fatal("dimension mismatch must error, never be scored")
	}
	if _, err := CosineSimilarity(nil, nil); err == nil {
		t.Fatal("empty vector must error")
	}
	if _, err := CosineSimilarity([]float32{0, 0}, []float32{0, 0}); err == nil {
		t.Fatal("zero-magnitude vectors must error")
	}
}

func TestLexicalSimilarityIsCoverageOfQuery(t *testing.T) {
	if got := LexicalSimilarity("work stress", "my work makes me stressed"); got <= 0 {
		t.Fatalf("expected partial lexical overlap, got %v", got)
	}
	if got := LexicalSimilarity("work stress", "the garden is green"); got != 0 {
		t.Fatalf("unrelated text must score 0, got %v", got)
	}
	if got := LexicalSimilarity("", "anything"); got != 0 {
		t.Fatalf("empty query scores 0, got %v", got)
	}
}

func TestHopNeedsContextKindIsClosed(t *testing.T) {
	for _, kind := range []string{"entity", "topic", "time_window", "item"} {
		if !HopNeedsContextKind(kind) {
			t.Fatalf("%q must be an allowed hop kind", kind)
		}
	}
	for _, kind := range []string{"", "everything", "sql", "ENTITY"} {
		if HopNeedsContextKind(kind) {
			t.Fatalf("%q must not be an allowed hop kind", kind)
		}
	}
}

func TestKeywordMatchAndEqualityHelpers(t *testing.T) {
	if !KeywordMatch([]string{"boss"}, "My BOSS was rude") {
		t.Fatal("keyword match is case-insensitive")
	}
	if KeywordMatch([]string{"boss"}, "nothing here") {
		t.Fatal("unexpected keyword match")
	}
	if !KeywordMatch(nil, "anything") {
		t.Fatal("empty terms match everything")
	}
	// The zero-width non-joiner is not a space: these stay distinct labels while
	// the helper stays stable across calls.
	first := EqualFoldTolerant("سه‌شنبه", "سه شنبه")
	if second := EqualFoldTolerant("سه‌شنبه", "سه شنبه"); first != second {
		t.Fatal("comparison must be deterministic")
	}
	if first {
		t.Fatal("ZWNJ removal must not equate two different spellings by accident")
	}
	if !EqualFoldTolerant("سه‌شنبه", "سه‌شنبه ") {
		t.Fatal("zero-width non-joiner and trailing space must compare equal")
	}
	if !EqualFoldTolerant("Work", " work ") {
		t.Fatal("trim + fold expected")
	}
}
