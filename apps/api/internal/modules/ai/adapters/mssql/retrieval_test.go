package mssql

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
)

// These tests are pure Go: they prove the adapter's query construction, owner
// scoping guards and decoding without claiming any SQL Server execution. Live
// execution is covered by tests/integration/ai_retrieval_test.go.

func TestNewRetrievalStoreRequiresDatabase(t *testing.T) {
	if _, err := NewRetrievalStore(nil); err == nil {
		t.Fatal("nil database accepted")
	}
	store := &RetrievalStore{}
	if err := ownerRequired(c.ID{}); err == nil {
		t.Fatal("zero owner accepted")
	}
	_ = store
}

func TestOwnerGuardsRejectBeforeSQL(t *testing.T) {
	store := &RetrievalStore{}
	ctx := context.Background()
	if _, err := store.MessagesInWindow(ctx, retrieval.MessageWindowQuery{From: time.Now(), To: time.Now().Add(time.Hour), Limit: 5}); err == nil {
		t.Fatal("missing owner accepted by the window read")
	}
	if _, err := store.MemoryByThread(ctx, c.ID{}, c.NewID(), nil, 5); err == nil {
		t.Fatal("missing owner accepted by the thread read")
	}
	if _, err := store.MemoryByEntity(ctx, retrieval.EntityQuery{Role: "sister", Limit: 5}); err == nil {
		t.Fatal("missing owner accepted by the entity read")
	}
	if _, err := store.MemoryCandidates(ctx, retrieval.CandidateQuery{Limit: 5}); err == nil {
		t.Fatal("missing owner accepted by the candidate read")
	}
	if _, err := store.MemoryByIDs(ctx, c.ID{}, []c.ID{c.NewID()}); err == nil {
		t.Fatal("missing owner accepted by the id read")
	}
	if _, err := store.Embeddings(ctx, c.ID{}, []retrieval.EmbeddingRef{{ID: c.NewID()}}); err == nil {
		t.Fatal("missing owner accepted by the embedding read")
	}
	if _, err := store.Thread(ctx, c.ID{}, c.NewID()); err == nil {
		t.Fatal("missing owner accepted by the thread lookup")
	}
	if _, err := store.ActiveThreads(ctx, c.ID{}, 3); err == nil {
		t.Fatal("missing owner accepted by the active-thread read")
	}
	if _, err := store.UserTimeZone(ctx, c.ID{}); err == nil {
		t.Fatal("missing owner accepted by the time-zone read")
	}
}

func TestInvalidBoundsRejected(t *testing.T) {
	store := &RetrievalStore{}
	user := c.NewID()
	if _, err := store.MessagesInWindow(context.Background(), retrieval.MessageWindowQuery{
		UserID: user, To: time.Now(), Limit: 5}); err == nil {
		t.Fatal("zero lower bound accepted")
	}
	if _, err := store.MessagesInWindow(context.Background(), retrieval.MessageWindowQuery{
		UserID: user, From: time.Now(), To: time.Now().Add(-time.Hour), Limit: 5}); err == nil {
		t.Fatal("inverted window accepted")
	}
	if _, err := store.MessagesInWindow(context.Background(), retrieval.MessageWindowQuery{
		UserID: user, From: time.Now().Add(-time.Hour), To: time.Now(), Limit: 1000}); err == nil {
		t.Fatal("unbounded limit accepted")
	}
	if _, err := store.MessagesInWindow(context.Background(), retrieval.MessageWindowQuery{
		UserID: user, From: time.Now().Add(-time.Hour), To: time.Now(), Roles: []c.Role{"admin"}, Limit: 5}); err == nil {
		t.Fatal("unknown role accepted")
	}
	if _, err := store.MemoryCandidates(context.Background(), retrieval.CandidateQuery{
		UserID: user, Limit: 5, Terms: []string{"x"}}); err == nil {
		t.Fatal("out-of-range term length accepted")
	}
	if _, err := store.MemoryCandidates(context.Background(), retrieval.CandidateQuery{
		UserID: user, Limit: 5, Terms: []string{strings.Repeat("a", 41)}}); err == nil {
		t.Fatal("oversized term accepted")
	}
	if _, err := store.MemoryByEntity(context.Background(), retrieval.EntityQuery{
		UserID: user, Limit: 5}); err == nil {
		t.Fatal("empty entity role accepted")
	}
	if _, err := store.MemoryByIDs(context.Background(), user, nil); err == nil {
		t.Fatal("empty id set accepted")
	}
	if _, err := store.MemoryByIDs(context.Background(), user, make([]c.ID, 21)); err == nil {
		t.Fatal("unbounded id set accepted")
	}
	if _, err := store.Embeddings(context.Background(), user, make([]retrieval.EmbeddingRef, 101)); err == nil {
		t.Fatal("unbounded embedding ref set accepted")
	}
	if _, err := store.ActiveThreads(context.Background(), user, 0); err == nil {
		t.Fatal("zero active-thread limit accepted")
	}
}

func TestInvalidFiltersRejected(t *testing.T) {
	query := "SELECT 1 FROM dbo.ai_memory_items m WHERE 1=1"
	if _, _, err := withKinds(query, nil, []c.MemoryKind{"not_a_kind"}); err == nil {
		t.Fatal("unknown memory kind accepted into SQL")
	}
	if _, _, err := withStatuses(query, nil, []c.MemoryStatus{"not_a_status"}); err == nil {
		t.Fatal("unknown memory status accepted into SQL")
	}
	got, args, err := withKinds(query, []any{"a"}, []c.MemoryKind{c.MemFact, c.MemEpisode})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "m.kind IN (N'fact',N'episode')") || len(args) != 1 {
		t.Fatalf("kind filter not appended as literals: %s %v", got, args)
	}
	got, _, err = withStatuses(query, nil, []c.MemoryStatus{c.MemActive})
	if err != nil || !strings.Contains(got, "m.status IN (N'active')") {
		t.Fatalf("status filter not appended: %s %v", got, err)
	}
	if got, _, err = withKinds(query, nil, nil); err != nil || got != query {
		t.Fatal("empty kind filter must not change the query")
	}
}

func TestEscapeLikeNeutralizesWildcards(t *testing.T) {
	cases := map[string]string{
		"boss":      "boss",
		"100%":      `100\%`,
		"a_b":       `a\_b`,
		"[abc]":     `\[abc\]`,
		`back\slash`: `back\\slash`,
	}
	for in, want := range cases {
		if got := escapeLike(in); got != want {
			t.Fatalf("escapeLike(%q)=%q want %q", in, got, want)
		}
	}
	// Apostrophes are not wildcards and must survive for the parameter binding.
	if got := escapeLike("rock'n'roll"); got != "rock'n'roll" {
		t.Fatalf("unexpected apostrophe handling: %q", got)
	}
}

func TestPlaceholderUsesDriverNumbering(t *testing.T) {
	for n := 1; n <= 12; n++ {
		got := placeholder(n)
		if n < 10 && got != string(rune('0'+n)) {
			t.Fatalf("placeholder(%d)=%q", n, got)
		}
		if n >= 10 && got != "10" && got != "11" && got != "12" {
			t.Fatalf("placeholder(%d)=%q", n, got)
		}
	}
}

func TestDecodeIDListAndStrings(t *testing.T) {
	var ids []c.ID
	if err := decodeIDList("", &ids); err != nil || ids != nil {
		t.Fatal("empty input should be a no-op")
	}
	first, second := c.NewID(), c.NewID()
	if err := decodeIDList(`["`+first.String()+`","`+second.String()+`"]`, &ids); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(ids) != 2 || ids[0] != first || ids[1] != second {
		t.Fatalf("unexpected ids: %v", ids)
	}
	// Unknown or malformed entries are skipped, never fabricated.
	ids = nil
	if err := decodeIDList(`["`+first.String()+`","not-a-uuid"]`, &ids); err != nil || len(ids) != 1 {
		t.Fatalf("malformed entries not skipped: %v %v", ids, err)
	}
	ids = nil
	if err := decodeIDList(`[{"id":"`+first.String()+`"}]`, &ids); err != nil || len(ids) != 1 {
		t.Fatalf("object form not supported: %v %v", ids, err)
	}
	if err := decodeIDList(`{"not":"an array"}`, &ids); err == nil {
		t.Fatal("malformed JSON accepted silently")
	}
	var tags []string
	if err := decodeStrings(`["anxiety","work"]`, &tags); err != nil || len(tags) != 2 {
		t.Fatalf("tags not decoded: %v %v", tags, err)
	}
}

func TestParseIDRejectsMalformedUUID(t *testing.T) {
	if _, err := parseID("nope"); err == nil {
		t.Fatal("malformed identifier accepted")
	}
	if _, err := parseID(""); err == nil {
		t.Fatal("empty identifier accepted")
	}
}

func TestRankUsesOnlyAvailableVectors(t *testing.T) {
	// The codec rejects a mismatched dimension before cosine ranking, so the
	// adapter surfaces the mismatch as "no usable vector" rather than a score.
	raw, err := EncodeEmbedding("m", []float32{3, 4})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if _, err = DecodeEmbedding(1, "m", "m", 2, 3, raw); err == nil {
		t.Fatal("dimension mismatch accepted by the storage codec")
	}
	if _, err = DecodeEmbedding(2, "m", "m", 2, 2, raw); err == nil {
		t.Fatal("unknown codec version accepted")
	}
	if _, err = DecodeEmbedding(1, "other", "m", 2, 2, raw); err == nil {
		t.Fatal("foreign model accepted")
	}
}

func TestStoreImplementsRetrievalPort(t *testing.T) {
	var _ retrieval.Store = (*RetrievalStore)(nil)
	if errors.Is(retrieval.ErrNotVisible, nil) {
		t.Fatal("ErrNotVisible must be a non-nil sentinel")
	}
}
