package retrieval

import (
	"context"
	"errors"
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// ---- Temporal / exact recall ----

func TestTemporalExactRecallReturnsVerbatimMessages(t *testing.T) {
	f := newFixture(t)
	// Two messages in the window, one before it, one in another user's account.
	inRange := f.store.addMessage(f.user, c.RoleUser, "I spoke to my boss about the deadline", f.now.AddDate(0, 0, -7).Add(time.Hour))
	inRange2 := f.store.addMessage(f.user, c.RoleUser, "my boss said the deadline moves", f.now.AddDate(0, 0, -7).Add(2*time.Hour))
	f.store.addMessage(f.user, c.RoleUser, "unrelated earlier message", f.now.AddDate(0, 0, -20))
	other := c.NewID()
	f.store.addMessage(other, c.RoleUser, "another user's message in the window", f.now.AddDate(0, 0, -7).Add(time.Hour))

	req := f.request("What did I tell you last Tuesday about my boss?")
	req.Turn.TemporalRefs = []c.TemporalRef{{
		Phrase: "last tuesday",
		From:   time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC),
		To:     time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC),
		RecallVerb: true,
	}}
	result, err := f.svc.Retrieve(context.Background(), req)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if !containsID(result.Items, inRange) || !containsID(result.Items, inRange2) {
		t.Fatalf("missing verbatim in-window messages: %+v", idsOf(result.Items))
	}
	if containsID(result.Items, other) {
		t.Fatal("cross-user message leaked into another user's retrieval")
	}
	for _, item := range result.Items {
		if !item.Verbatim {
			t.Fatalf("temporal item must be verbatim, got %+v", item)
		}
		if item.Status != c.MemActive || item.Epistemic != c.EpReported {
			t.Fatalf("raw message must be reported/active: %+v", item)
		}
	}
	if len(f.store.messages) != 4 {
		t.Fatal("retrieval mutated stored messages")
	}
	if f.store.lastWindow.From.Location() != time.UTC {
		t.Fatal("window bounds must be sent as UTC")
	}
}

func TestTemporalEmptyWindowIsSuccessfulNotFound(t *testing.T) {
	f := newFixture(t)
	f.store.addMessage(f.user, c.RoleUser, "message on a different day", f.now.AddDate(0, 0, -30))
	req := f.request("what did I tell you last tuesday")
	req.Turn.TemporalRefs = []c.TemporalRef{{Phrase: "last tuesday",
		From: time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC), RecallVerb: true}}
	result, err := f.svc.Retrieve(context.Background(), req)
	if err != nil {
		t.Fatalf("an empty window is not an error: %v", err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("semantic substitution occurred: %+v", result.Items)
	}
	if result.Meta.Degraded {
		t.Fatal("empty window incorrectly degraded")
	}
	if result.Meta.ExactMatchFound == nil || *result.Meta.ExactMatchFound {
		t.Fatalf("exact_match_found must be present and false: %+v", result.Meta)
	}
}

func TestTemporalUsesPersistedUserZone(t *testing.T) {
	f := newFixture(t)
	// A user asking about "yesterday" on the 4th at 10:00 Tehran means the
	// Tehran day starting 3 March 00:00 (+03:30) = 2026-03-02T20:30Z.
	tehran, err := time.LoadLocation("Asia/Tehran")
	if err != nil {
		t.Fatalf("zone: %v", err)
	}
	f.store.tz = "Asia/Tehran"
	f.now = time.Date(2026, 3, 4, 10, 0, 0, 0, tehran)
	svc, err := New(f.store, f.registry, f.embedder, WithClock(func() time.Time { return f.now }))
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	// 19:30Z on 3 March is 23:00 on 3 March in Tehran: inside "yesterday" for
	// that user, but outside the UTC day the server would otherwise use.
	lateNight := f.store.addMessage(f.user, c.RoleUser, "I couldn't sleep again", time.Date(2026, 3, 3, 19, 30, 0, 0, time.UTC))
	req := f.request("what did I tell you yesterday")
	// The Gateway normally supplies the parsed window; here the service must
	// parse it itself in the user's persisted zone.
	result, err := svc.Retrieve(context.Background(), req)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if !containsID(result.Items, lateNight) {
		t.Fatalf("local-time window missed the message: strategies=%v", result.Meta.Strategies)
	}
}

func TestTemporalUnparseableQueryReportsAmbiguity(t *testing.T) {
	f := newFixture(t)
	req := f.request("what about that time")
	req.Extra = &c.NeedsContext{Kind: c.ContextTimeWindow, Query: "one of those days", Why: c.ContextReferenceUnresolved}
	result, err := f.svc.Retrieve(context.Background(), req)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if !result.Meta.AmbiguousReference {
		t.Fatal("unparseable second-hop window must be flagged ambiguous")
	}
	if len(result.Items) != 0 {
		t.Fatal("invented a window result")
	}
}

func TestTemporalInvalidUserZoneDegrades(t *testing.T) {
	f := newFixture(t)
	f.store.tz = "Mars/Olympus_Mons"
	req := f.request("What did I tell you last Tuesday?")
	req.Turn.TemporalRefs = []c.TemporalRef{{Phrase: "last tuesday",
		From: time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC), RecallVerb: true}}
	result, err := f.svc.Retrieve(context.Background(), req)
	if err != nil {
		t.Fatalf("a bad zone must degrade, not fail the turn: %v", err)
	}
	if !result.Meta.Degraded {
		t.Fatal("an unusable stored zone must mark retrieval degraded")
	}
}

// ---- Thread / hypothesis / entity ----

func TestThreadStrategyLinksAndBoundsHypotheses(t *testing.T) {
	f := newFixture(t)
	thread := c.ThreadRecord{ID: c.NewID(), UserID: f.user, Topic: "social comparison",
		Phase: c.PhaseExploring, Priority: 0.9, CreatedAt: f.now.AddDate(0, 0, -3), LastActiveAt: f.now}
	f.store.threads = []c.ThreadRecord{thread}

	episode := f.memoryItem("felt worthless after a coworker's promotion", c.MemEpisode, c.MemActive, f.now.AddDate(0, 0, -9))
	episode.ThreadIDs = []c.ID{thread.ID}
	hypothesis := f.memoryItem("self-worth may be tied to comparison with others", c.MemHypothesis, c.MemProposed, f.now.AddDate(0, 0, -9))
	hypothesis.Epistemic = c.EpInferred
	hypothesis.ThreadIDs = []c.ID{thread.ID}
	unlinkedHypothesis := f.memoryItem("unrelated hypothesis", c.MemHypothesis, c.MemProposed, f.now)
	unlinkedHypothesis.Epistemic = c.EpInferred
	rejected := f.memoryItem("rejected hypothesis text", c.MemHypothesis, c.MemRejected, f.now)
	rejected.Epistemic = c.EpInferred
	rejected.ThreadIDs = []c.ID{thread.ID}
	f.store.memory = []c.MemoryItem{episode, hypothesis, unlinkedHypothesis, rejected}

	req := f.request("I feel the same way as before")
	req.FocusThread = &thread.ID
	req.Strategies = []c.RetrievalStrategy{c.StratThread}
	req.Exclude = nil
	result, err := f.svc.Retrieve(context.Background(), req)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if !containsID(result.Items, episode.ID) {
		t.Fatalf("thread episode missing: %+v", idsOf(result.Items))
	}
	if !containsID(result.Items, hypothesis.ID) {
		t.Fatalf("linked proposed hypothesis must be retrievable: %+v", idsOf(result.Items))
	}
	if containsID(result.Items, unlinkedHypothesis.ID) {
		t.Fatal("hypothesis without an active-thread link was retrieved")
	}
	if containsID(result.Items, rejected.ID) {
		t.Fatal("rejected status was retrieved")
	}
}

func TestThreadRejectsForeignFocusThread(t *testing.T) {
	f := newFixture(t)
	stranger := c.NewID()
	thread := c.ThreadRecord{ID: c.NewID(), UserID: stranger, Topic: "not mine", Phase: c.PhaseExploring}
	f.store.threads = []c.ThreadRecord{thread}
	req := f.request("continue where we left off")
	req.FocusThread = &thread.ID
	if _, err := f.svc.Retrieve(context.Background(), req); err == nil {
		t.Fatal("retrieved against another user's thread")
	}
}

func TestEntityStrategyMatchesStoredRoleOnly(t *testing.T) {
	f := newFixture(t)
	sister := f.memoryItem("my sister Lena moved to Tehran", c.MemFact, c.MemActive, f.now.AddDate(0, 0, -5))
	sister.Entities = []c.EntityRef{{Role: "sister", Label: ptr("Lena")}}
	boss := f.memoryItem("my boss criticized my work", c.MemFact, c.MemActive, f.now.AddDate(0, 0, -5))
	boss.Entities = []c.EntityRef{{Role: "boss"}}
	elsewhere := c.NewID()
	foreign := c.MemoryItem{ID: c.NewID(), UserID: elsewhere, Kind: c.MemFact, Content: "my sister in another account",
		Epistemic: c.EpReported, Status: c.MemActive, Sensitivity: c.SensNormal, Importance: 0.9,
		Entities: []c.EntityRef{{Role: "sister"}}, CreatedAt: f.now, UpdatedAt: f.now}
	f.store.memory = []c.MemoryItem{sister, boss, foreign}

	req := f.request("did I tell you about my sister Lena?")
	req.Strategies = []c.RetrievalStrategy{c.StratEntity}
	result, err := f.svc.Retrieve(context.Background(), req)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if !containsID(result.Items, sister.ID) {
		t.Fatalf("entity role match failed: %+v", idsOf(result.Items))
	}
	if containsID(result.Items, boss.ID) || containsID(result.Items, foreign.ID) {
		t.Fatal("entity strategy returned a non-matching or foreign item")
	}
}

func TestEntitySecondHopWithoutKnownRoleIsAmbiguous(t *testing.T) {
	f := newFixture(t)
	f.store.memory = []c.MemoryItem{f.memoryItem("some fact", c.MemFact, c.MemActive, f.now)}
	req := f.request("what about her mother-in-law")
	req.Extra = &c.NeedsContext{Kind: c.ContextEntity, Query: "Lena", Why: c.ContextReferenceUnresolved}
	result, err := f.svc.Retrieve(context.Background(), req)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if !result.Meta.AmbiguousReference {
		t.Fatal("an entity that maps to no stored role must be ambiguous")
	}
	if len(result.Items) != 0 {
		t.Fatal("guessed an entity match")
	}
}

// ---- Anaphora ----

func TestAnaphoraBoostAndAmbiguity(t *testing.T) {
	f := newFixture(t)
	recent := f.memoryItem("felt anxious before my sister's visit", c.MemEpisode, c.MemActive, f.now.AddDate(0, 0, -3))
	recent.Tags = []string{"anxiety"}
	old := f.memoryItem("felt anxious last year about the same thing", c.MemEpisode, c.MemActive, f.now.AddDate(0, 0, -60))
	old.Tags = []string{"anxiety"}
	f.store.memory = []c.MemoryItem{recent, old}

	// The reference marker plus the emotion word "anxious" is what lets the
	// anaphora strategy resolve the referent; without an emotion or topic
	// signal the reference stays ambiguous (asserted below).
	req := f.request("I feel anxious again, same as before")
	req.Strategies = []c.RetrievalStrategy{c.StratAnaphora}
	result, err := f.svc.Retrieve(context.Background(), req)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if !containsID(result.Items, recent.ID) {
		t.Fatalf("recent matching item missing: %+v", idsOf(result.Items))
	}
	for _, item := range result.Items {
		if item.ID == old.ID {
			t.Fatal("anaphora escaped the configured recent window")
		}
	}
	// A reference with no matching history must be flagged, not guessed.
	f2 := newFixture(t)
	f2.store.memory = []c.MemoryItem{f2.memoryItem("something about a birthday", c.MemEpisode, c.MemActive, f2.now)}
	req2 := f2.request("that thing I told you about")
	req2.Strategies = []c.RetrievalStrategy{c.StratAnaphora}
	result2, err := f2.svc.Retrieve(context.Background(), req2)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if !result2.Meta.AmbiguousReference {
		t.Fatal("unresolved reference must set ambiguous_reference")
	}
}

func TestAnaphoraAutoRouterAddsRelatedStrategies(t *testing.T) {
	f := newFixture(t)
	req := f.request("I feel the same thing again with my sister")
	result, err := f.svc.Retrieve(context.Background(), req)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	joined := strings.Join(result.Meta.Strategies, ",")
	for _, want := range []string{"thread", "anaphora", "entity", "hybrid"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("router omitted %s: %v", want, result.Meta.Strategies)
		}
	}
}

// ---- Hybrid semantics ----

func TestHybridRankingPrefersSimilarityThenRecency(t *testing.T) {
	f := newFixture(t)
	relevant := f.memoryItem("my boss at work keeps moving the deadline", c.MemEpisode, c.MemActive, f.now.AddDate(0, 0, -2))
	relevant.Importance = 0.4
	stale := f.memoryItem("my boss at work was calm last year", c.MemEpisode, c.MemActive, f.now.AddDate(0, 0, -200))
	stale.Importance = 0.9
	unrelated := f.memoryItem("I slept well and felt calm", c.MemEpisode, c.MemActive, f.now.AddDate(0, 0, -1))
	unrelated.Importance = 0.9
	f.store.memory = []c.MemoryItem{relevant, stale, unrelated}
	f.store.vectors[relevant.ID] = retrievalVector{vector: []float32{0, 1, 0}, model: "test-embed", codec: 1}
	f.store.vectors[stale.ID] = retrievalVector{vector: []float32{0, 1, 0}, model: "test-embed", codec: 1}
	f.store.vectors[unrelated.ID] = retrievalVector{vector: []float32{0, 0, 1}, model: "test-embed", codec: 1}

	req := f.request("my boss moved the deadline at work again")
	result, err := f.svc.Retrieve(context.Background(), req)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if len(result.Items) < 2 {
		t.Fatalf("expected ranked items: %+v", idsOf(result.Items))
	}
	if result.Items[0].ID != relevant.ID {
		t.Fatalf("expected the recent similar item first, got %v", idsOf(result.Items))
	}
	// Scores must be strictly descending and within [0,1].
	for i := 1; i < len(result.Items); i++ {
		if result.Items[i].Score > result.Items[i-1].Score {
			t.Fatalf("items are not ranked: %+v", result.Items)
		}
		if result.Items[i].Score < 0 || result.Items[i].Score > 1 {
			t.Fatalf("score outside [0,1]: %v", result.Items[i].Score)
		}
	}
}

func TestMissingEmbeddingsDegradeQualityNotTruth(t *testing.T) {
	f := newFixture(t)
	withVector := f.memoryItem("my boss at work", c.MemEpisode, c.MemActive, f.now.Add(-time.Hour))
	noVector := f.memoryItem("my boss at work last week", c.MemEpisode, c.MemActive, f.now.AddDate(0, 0, -3))
	withVector.Importance = 1
	noVector.Importance = 1
	f.store.memory = []c.MemoryItem{withVector, noVector}
	f.store.vectors[withVector.ID] = retrievalVector{vector: []float32{0, 1, 0}, model: "test-embed", codec: 1}

	req := f.request("my boss at work")
	req.Strategies = []c.RetrievalStrategy{c.StratHybrid}
	result, err := f.svc.Retrieve(context.Background(), req)
	if err != nil {
		t.Fatalf("a missing embedding is not an error: %v", err)
	}
	if !containsID(result.Items, withVector.ID) {
		t.Fatalf("embedded item missing: %+v", idsOf(result.Items))
	}
	if result.Meta.Degraded {
		t.Fatal("missing embeddings must not be reported as a degraded retrieval")
	}
	// The item without a vector still competes on recency/importance; what it
	// must never do is inherit a similarity it does not have, so its score is
	// strictly bounded by the deterministic components.
	noVectorScore := float64(-1)
	for _, item := range result.Items {
		if item.Score > 1 {
			t.Fatalf("invented similarity: %+v", item)
		}
		if item.ID == noVector.ID {
			noVectorScore = float64(item.Score)
		}
	}
	cfg := loadTestConfig(t)
	ceiling := cfg.Weights.Recency + cfg.Weights.Importance
	if noVectorScore >= 0 && noVectorScore > ceiling+1e-6 {
		t.Fatalf("unembedded item scored above its available evidence: %v > %v", noVectorScore, ceiling)
	}
}

func TestDimensionMismatchIsIgnoredNotScored(t *testing.T) {
	f := newFixture(t)
	item := f.memoryItem("my boss and the deadline", c.MemEpisode, c.MemActive, f.now.Add(-time.Hour))
	item.Importance = 1
	f.store.memory = []c.MemoryItem{item}
	f.store.vectors[item.ID] = retrievalVector{vector: []float32{1, 0}, model: "test-embed", codec: 1} // 2 dims, provider 3
	req := f.request("my boss and the deadline")
	req.Strategies = []c.RetrievalStrategy{c.StratHybrid}
	result, err := f.svc.Retrieve(context.Background(), req)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if !containsID(result.Items, item.ID) {
		t.Fatal("item dropped because of an unusable vector")
	}
	// Similarity contribution must be zero: the score equals the deterministic
	// part of the formula derived from the same candidate fields.
	cfg := loadTestConfig(t)
	expected := cfg.Weights.Recency*recencyDecay(f.now, f.now.Add(-time.Hour), cfg.HalfLife) +
		cfg.Weights.Importance*float64(item.Importance)
	if diff := math.Abs(float64(result.Items[0].Score) - expected); diff > 0.0001 {
		t.Fatalf("mismatched dimension contributed to the score: got %v want %v", result.Items[0].Score, expected)
	}
}

func TestOtherModelVectorIsNotCompared(t *testing.T) {
	f := newFixture(t)
	item := f.memoryItem("my boss and the deadline", c.MemEpisode, c.MemActive, f.now.Add(-time.Hour))
	item.Importance = 1
	f.store.memory = []c.MemoryItem{item}
	f.store.vectors[item.ID] = retrievalVector{vector: []float32{0, 1, 0}, model: "different-model", codec: 1}
	req := f.request("my boss and the deadline")
	req.Strategies = []c.RetrievalStrategy{c.StratHybrid}
	result, err := f.svc.Retrieve(context.Background(), req)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	cfg := loadTestConfig(t)
	expected := cfg.Weights.Recency*recencyDecay(f.now, f.now.Add(-time.Hour), cfg.HalfLife) +
		cfg.Weights.Importance*float64(item.Importance)
	if diff := math.Abs(float64(result.Items[0].Score) - expected); diff > 0.0001 {
		t.Fatalf("foreign model vector was used: got %v want %v", result.Items[0].Score, expected)
	}
}

func TestNilEmbedderStillReturnsDeterministicResults(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	user := c.NewID()
	store := newFakeStore(user)
	svc, err := New(store, testRegistry(), nil, WithClock(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	item := c.MemoryItem{ID: c.NewID(), UserID: user, Kind: c.MemEpisode, Content: "my boss again",
		Epistemic: c.EpReported, Status: c.MemActive, Sensitivity: c.SensNormal, Importance: 0.9,
		CreatedAt: now.AddDate(0, 0, -1), UpdatedAt: now.AddDate(0, 0, -1)}
	store.memory = []c.MemoryItem{item}
	result, err := svc.Retrieve(context.Background(), c.RetrievalRequest{UserID: user, MaxItems: 5,
		Turn: c.TurnView{TurnID: c.NewID(), MessageID: c.NewID(), Text: "my boss again", Lang: "en"}})
	if err != nil {
		t.Fatalf("retrieve without an embedding model: %v", err)
	}
	if !containsID(result.Items, item.ID) {
		t.Fatal("deterministic retrieval must work without a provider")
	}
}

// ---- Filters, dedupe, budget ----

func TestExcludedStatusesAndSensitivityAreNeverRetrieved(t *testing.T) {
	f := newFixture(t)
	active := f.memoryItem("my boss moved the deadline", c.MemEpisode, c.MemActive, f.now)
	rejected := f.memoryItem("my boss moved the deadline", c.MemEpisode, c.MemRejected, f.now)
	superseded := f.memoryItem("my boss moved the deadline v1", c.MemEpisode, c.MemSuperseded, f.now)
	stale := f.memoryItem("my boss moved the deadline long ago", c.MemEpisode, c.MemStale, f.now)
	highly := f.memoryItem("my boss detail that must stay excluded", c.MemEpisode, c.MemActive, f.now)
	highly.Sensitivity = c.SensHighly
	f.store.memory = []c.MemoryItem{active, rejected, superseded, stale, highly}
	// Force every strategy so the filters are exercised on all paths.
	req := f.request("my boss moved the deadline")
	req.Strategies = []c.RetrievalStrategy{c.StratHybrid, c.StratThread, c.StratAnaphora, c.StratEntity}
	result, err := f.svc.Retrieve(context.Background(), req)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	for _, banned := range []c.ID{rejected.ID, superseded.ID, stale.ID, highly.ID} {
		if containsID(result.Items, banned) {
			t.Fatalf("ineligible item retrieved: %v", banned)
		}
	}
	if !containsID(result.Items, active.ID) {
		t.Fatal("eligible item was dropped")
	}
}

func TestExcludeListAndClusterDedupe(t *testing.T) {
	f := newFixture(t)
	cluster := c.NewID()
	first := f.memoryItem("my boss moved the deadline", c.MemEpisode, c.MemActive, f.now.AddDate(0, 0, -1))
	first.ClusterID = &cluster
	first.Importance = 0.9
	second := f.memoryItem("my boss moved the deadline again", c.MemEpisode, c.MemActive, f.now.AddDate(0, 0, -2))
	second.ClusterID = &cluster
	second.Importance = 0.8
	excluded := f.memoryItem("already in context", c.MemEpisode, c.MemActive, f.now)
	f.store.memory = []c.MemoryItem{first, second, excluded}
	req := f.request("my boss moved the deadline")
	req.Exclude = []c.ID{excluded.ID}
	result, err := f.svc.Retrieve(context.Background(), req)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if containsID(result.Items, excluded.ID) {
		t.Fatal("excluded item was returned")
	}
	count := 0
	for _, item := range result.Items {
		if item.ID == first.ID || item.ID == second.ID {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("cluster dedupe failed, got %d of a twinned cluster: %+v", count, idsOf(result.Items))
	}
}

func TestIdenticalTextIsDeduped(t *testing.T) {
	f := newFixture(t)
	a := f.memoryItem("my boss moved the deadline", c.MemEpisode, c.MemActive, f.now)
	b := f.memoryItem("my boss moved the deadline", c.MemEpisode, c.MemActive, f.now)
	f.store.memory = []c.MemoryItem{a, b}
	result, err := f.svc.Retrieve(context.Background(), f.request("my boss moved the deadline"))
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("identical content was not deduped: %+v", idsOf(result.Items))
	}
}

func TestTopKAndTokenBudgetAreEnforced(t *testing.T) {
	f := newFixture(t)
	for i := 0; i < 12; i++ {
		item := f.memoryItem("my boss moved the deadline number "+strings.Repeat("x", i+1), c.MemEpisode, c.MemActive, f.now.Add(-time.Duration(i)*time.Hour))
		item.Importance = 0.9
		f.store.memory = append(f.store.memory, item)
	}
	result, err := f.svc.Retrieve(context.Background(), f.request("my boss moved the deadline"))
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if len(result.Items) > 5 {
		t.Fatalf("hard top-K violated: %d items", len(result.Items))
	}
	// Per-item token cap: no view may exceed the configured words plus ellipsis.
	for _, item := range result.Items {
		words := len(strings.Fields(strings.TrimSuffix(item.Text, "…")))
		if words > 60 {
			t.Fatalf("per-item token cap violated: %d words", words)
		}
	}
}

func TestRequestedMaxItemsCannotExceedHardCap(t *testing.T) {
	f := newFixture(t)
	for i := 0; i < 10; i++ {
		item := f.memoryItem("my boss moved the deadline "+strings.Repeat("y", i+1), c.MemEpisode, c.MemActive, f.now.Add(-time.Duration(i)*time.Minute))
		item.Importance = 0.9
		f.store.memory = append(f.store.memory, item)
	}
	req := f.request("my boss moved the deadline")
	req.MaxItems = 50
	req.Extra = &c.NeedsContext{Kind: c.ContextTopic, Query: "boss", Why: c.ContextMissingBackground}
	result, err := f.svc.Retrieve(context.Background(), req)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if len(result.Items) > 7 {
		t.Fatalf("hard cap with hop violated: %d", len(result.Items))
	}
	if result.Meta.Hops != 1 {
		t.Fatalf("hop count must be reported: %+v", result.Meta)
	}
}

func TestMinimumScoreThresholdFiltersLowRelevance(t *testing.T) {
	f := newFixture(t)
	// Low importance, old, no lexical overlap, no thread/match => below 0.32.
	weak := f.memoryItem("a completely different topic about bicycles", c.MemEpisode, c.MemActive, f.now.AddDate(0, 0, -300))
	weak.Importance = 0.01
	f.store.memory = []c.MemoryItem{weak}
	result, err := f.svc.Retrieve(context.Background(), f.request("my boss moved the deadline"))
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if containsID(result.Items, weak.ID) {
		t.Fatalf("item below min_score was retrieved: %+v", result.Items)
	}
	// The double returns high-importance items even without lexical overlap;
	// this item is not high-importance, so it never entered the pool at all.
	if f.store.calls["candidates"] == 0 {
		t.Fatal("hybrid strategy did not query the candidate pool")
	}
}

// ---- Isolation and errors ----

func TestCrossUserIsolationAcrossEveryStrategy(t *testing.T) {
	f := newFixture(t)
	other := c.NewID()
	threadID := c.NewID()
	own := f.memoryItem("my boss moved the deadline", c.MemEpisode, c.MemActive, f.now)
	foreign := c.MemoryItem{ID: c.NewID(), UserID: other, Kind: c.MemEpisode, Content: "my boss moved the deadline",
		Epistemic: c.EpReported, Status: c.MemActive, Sensitivity: c.SensNormal, Importance: 1.0,
		ThreadIDs: []c.ID{threadID}, Entities: []c.EntityRef{{Role: "boss"}},
		CreatedAt: f.now, UpdatedAt: f.now}
	f.store.memory = []c.MemoryItem{foreign, own}
	f.store.threads = []c.ThreadRecord{{ID: threadID, UserID: other, Topic: "foreign thread", Phase: c.PhaseExploring}}
	f.store.vectors[foreign.ID] = retrievalVector{vector: []float32{0, 1, 0}, model: "test-embed", codec: 1}

	req := f.request("my boss moved the deadline")
	req.Strategies = []c.RetrievalStrategy{c.StratHybrid, c.StratEntity, c.StratAnaphora}
	result, err := f.svc.Retrieve(context.Background(), req)
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if containsID(result.Items, foreign.ID) {
		t.Fatal("cross-user memory leaked through a strategy")
	}
	if !containsID(result.Items, own.ID) {
		t.Fatal("own memory was dropped")
	}
}

func TestStrategyFailureIsolatesAndDegrades(t *testing.T) {
	f := newFixture(t)
	item := f.memoryItem("my boss moved the deadline", c.MemEpisode, c.MemActive, f.now)
	item.Importance = 0.9
	f.store.memory = []c.MemoryItem{item}
	f.store.errThread = errors.New("simulated thread read failure")
	req := f.request("my boss moved the deadline")
	req.FocusThread = &c.ID{} // invalid identity is rejected before any SQL
	if _, err := f.svc.Retrieve(context.Background(), req); err == nil {
		t.Fatal("an invalid focus thread identity must be rejected")
	}
	threadID := c.NewID()
	f.store.threads = []c.ThreadRecord{{ID: threadID, UserID: f.user, Phase: c.PhaseWorking, Topic: "boss"}}
	req.FocusThread = &threadID
	result, err := f.svc.Retrieve(context.Background(), req)
	if err != nil {
		t.Fatalf("one strategy failing must not fail the turn: %v", err)
	}
	if !result.Meta.Degraded {
		t.Fatal("a failed strategy must be reported as degraded")
	}
	// Hybrid still supplied the item, so the degradation is partial.
	if !containsID(result.Items, item.ID) {
		t.Fatal("a partial failure lost the remaining strategies' results")
	}
}

func TestAllStrategiesFailingReturnsError(t *testing.T) {
	f := newFixture(t)
	f.store.errCand = errors.New("simulated database outage")
	req := f.request("my boss moved the deadline")
	req.Strategies = []c.RetrievalStrategy{c.StratHybrid}
	result, err := f.svc.Retrieve(context.Background(), req)
	if err == nil {
		t.Fatal("total failure must return an error so the caller marks Degraded")
	}
	if !result.Meta.Degraded {
		t.Fatal("total failure must report degraded metadata")
	}
	if len(result.Items) != 0 {
		t.Fatal("total failure returned fabricated items")
	}
	contractErr := AsContractError(err)
	if contractErr.Code != c.ErrRetrievalDegraded {
		t.Fatalf("expected retrieval_degraded, got %s", contractErr.Code)
	}
	if !contractErr.Degrades || !contractErr.Retryable {
		t.Fatalf("degraded retrieval error must be retryable/degrading: %+v", contractErr)
	}
}

func TestEmbeddingProviderErrorDoesNotFailRetrieval(t *testing.T) {
	f := newFixture(t)
	f.embedder.err = errors.New("provider unavailable")
	item := f.memoryItem("my boss moved the deadline", c.MemEpisode, c.MemActive, f.now)
	item.Importance = 0.9
	f.store.memory = []c.MemoryItem{item}
	f.store.vectors[item.ID] = retrievalVector{vector: []float32{0, 1, 0}, model: "test-embed", codec: 1}
	result, err := f.svc.Retrieve(context.Background(), f.request("my boss moved the deadline"))
	if err != nil {
		t.Fatalf("embedding failure must not fail retrieval: %v", err)
	}
	if !containsID(result.Items, item.ID) {
		t.Fatal("lexical retrieval was lost when embeddings failed")
	}
}

func TestEmbeddingStorageErrorDoesNotFailRetrieval(t *testing.T) {
	f := newFixture(t)
	f.store.errEmbed = errors.New("simulated vector table failure")
	item := f.memoryItem("my boss moved the deadline", c.MemEpisode, c.MemActive, f.now)
	item.Importance = 0.9
	f.store.memory = []c.MemoryItem{item}
	result, err := f.svc.Retrieve(context.Background(), f.request("my boss moved the deadline"))
	if err != nil {
		t.Fatalf("vector read failure must not fail retrieval: %v", err)
	}
	if !containsID(result.Items, item.ID) {
		t.Fatal("candidate was lost when vectors could not be read")
	}
}

func TestCancelledContextReturnsImmediately(t *testing.T) {
	f := newFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := f.svc.Retrieve(ctx, f.request("my boss"))
	if err == nil {
		t.Fatal("a cancelled context must abort retrieval")
	}
	if !result.Meta.Degraded {
		t.Fatal("cancelled retrieval must be degraded")
	}
}

func TestValidationAndIdentityRejections(t *testing.T) {
	f := newFixture(t)
	cases := []struct {
		name string
		req  c.RetrievalRequest
	}{
		{"no owner", func() c.RetrievalRequest {
			r := f.request("hello")
			r.UserID = c.ID{}
			return r
		}()},
		{"no message id", func() c.RetrievalRequest { r := f.request("hello"); r.Turn.MessageID = c.ID{}; return r }()},
		{"empty text", func() c.RetrievalRequest { r := f.request("   "); return r }()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := f.svc.Retrieve(context.Background(), tc.req); err == nil {
				t.Fatal("invalid request accepted")
			}
		})
	}
}

func TestNilDependenciesRejected(t *testing.T) {
	if _, err := New(nil, testRegistry(), nil); err == nil {
		t.Fatal("nil store accepted")
	}
	if _, err := New(newFakeStore(c.NewID()), nil, nil); err == nil {
		t.Fatal("nil bundle registry accepted")
	}
}

func TestBundleFailureFailsClosed(t *testing.T) {
	f := newFixture(t)
	f.registry.mu.Lock()
	f.registry.err = errors.New("no active bundle")
	f.registry.mu.Unlock()
	if _, err := f.svc.Retrieve(context.Background(), f.request("hello")); err == nil {
		t.Fatal("retrieval ran without a pinned bundle")
	}
	if len(f.store.calls) != 0 {
		t.Fatalf("storage was read before the policy was known: %v", f.store.calls)
	}
}

func TestUnknownBundleArtifactFailsClosed(t *testing.T) {
	reg := &fakeRegistry{bundle: fakeBundle{files: map[string][]byte{
		"config/retrieval.yaml": []byte(retrievalYAML),
	}}}
	svc, err := New(newFakeStore(c.NewID()), reg, nil)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if _, err := svc.Retrieve(context.Background(), c.RetrievalRequest{UserID: c.NewID(), MaxItems: 5,
		Turn: c.TurnView{TurnID: c.NewID(), MessageID: c.NewID(), Text: "hi", Lang: "en"}}); err == nil {
		t.Fatal("missing budgets artifact must fail closed")
	}
}

// ---- Concurrency ----

func TestConcurrentRetrievalHasNoSharedMutableState(t *testing.T) {
	f := newFixture(t)
	threadID := c.NewID()
	f.store.threads = []c.ThreadRecord{{ID: threadID, UserID: f.user, Phase: c.PhaseWorking, Topic: "boss"}}
	for i := 0; i < 8; i++ {
		item := f.memoryItem("my boss moved the deadline", c.MemEpisode, c.MemActive, f.now.Add(-time.Duration(i)*time.Hour))
		item.Importance = 0.9
		item.ThreadIDs = []c.ID{threadID}
		f.store.memory = append(f.store.memory, item)
		f.store.vectors[item.ID] = retrievalVector{vector: []float32{0, 1, 0}, model: "test-embed", codec: 1}
	}
	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req := f.request("my boss moved the deadline")
			req.FocusThread = &threadID
			if i%2 == 0 {
				req.Strategies = []c.RetrievalStrategy{c.StratTemporal}
			}
			result, err := f.svc.Retrieve(context.Background(), req)
			if err != nil {
				errs <- err
				return
			}
			for _, item := range result.Items {
				if item.ID == (c.ID{}) || item.Score < 0 || item.Score > 1 {
					errs <- errors.New("corrupt item from concurrent retrieval")
					return
				}
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}

// loadTestConfig decodes the fixture bundle directly for score assertions.
func loadTestConfig(t *testing.T) Config {
	t.Helper()
	svc := &Service{policy: testRegistry(), now: func() time.Time { return time.Now() }}
	cfg, err := LoadConfig(context.Background(), svc.policy, c.NewID())
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	return cfg
}
