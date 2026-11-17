package mssql

import (
	"math"
	"testing"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	domain "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/domain/retrieval"
)

func TestNewPanicsOnNilDatabase(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("a nil pool must fail at construction, not at the first query")
		}
	}()
	New(nil)
}

func TestDecodeVectorRejectsBadInput(t *testing.T) {
	cases := map[string]struct {
		blob []byte
		dim  int
	}{
		"zero dim":     {make([]byte, 4), 0},
		"dim too big":  {make([]byte, 4), 9000},
		"length short": {make([]byte, 7), 2},
		"length long":  {make([]byte, 12), 2},
		"empty blob":   {nil, 2},
	}
	for name, tc := range cases {
		if _, err := decodeVector(tc.blob, tc.dim); err == nil {
			t.Fatalf("%s: expected a decode error", name)
		}
	}
}

func TestDecodeVectorRejectsNonFinite(t *testing.T) {
	blob, err := EncodeVector([]float32{1, 2, 3, 4})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := decodeVector(blob, 4); err != nil {
		t.Fatalf("valid vector rejected: %v", err)
	}
	// Corrupt one lane into NaN.
	blob[0], blob[1], blob[2], blob[3] = 0, 0, 0xC0, 0x7F
	if _, err := decodeVector(blob, 4); err == nil {
		t.Fatal("NaN must be rejected")
	}
	inf := float32(math.Inf(1))
	if _, err := EncodeVector([]float32{inf}); err == nil {
		t.Fatal("infinite values must be rejected when encoding")
	}
}

func TestVectorRoundTrip(t *testing.T) {
	in := []float32{0.5, -0.25, 0.125}
	blob, err := EncodeVector(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(blob) != len(in)*4 {
		t.Fatalf("float32 storage must be 4 bytes per dimension, got %d", len(blob))
	}
	out, err := decodeVector(blob, len(in))
	if err != nil {
		t.Fatal(err)
	}
	for i := range in {
		if in[i] != out[i] {
			t.Fatalf("lane %d: %v != %v", i, in[i], out[i])
		}
	}
}

func TestParseUUIDStrict(t *testing.T) {
	if _, err := parseUUIDStrict("not-a-uuid"); err == nil {
		t.Fatal("malformed identifiers must error")
	}
	id, err := parseUUIDStrict("aaaaaaaa-0000-0000-0000-000000000001")
	if err != nil || id == (contract.ID{}) {
		t.Fatalf("valid uuid rejected: %v", err)
	}
}

func TestJSONColumnParsingToleratesMalformedEntries(t *testing.T) {
	if _, err := parseStringArray(`["a","b"]`); err != nil {
		t.Fatal(err)
	}
	if _, err := parseStringArray(`not json`); err == nil {
		t.Fatal("invalid JSON must surface as an error, not silent data loss")
	}
	ids, err := parseUUIDArray(`["aaaaaaaa-0000-0000-0000-000000000001","broken"]`)
	if err != nil {
		t.Fatalf("one malformed entry must not fail retrieval: %v", err)
	}
	if len(ids) != 1 {
		t.Fatalf("expected the 1 valid thread id, got %d", len(ids))
	}
}

func TestClampLimit(t *testing.T) {
	if clampLimit(0) != 25 || clampLimit(-5) != 25 {
		t.Fatal("a missing limit must default to a bounded value")
	}
	if clampLimit(10000) != 500 {
		t.Fatal("an unbounded limit must be capped")
	}
	if clampLimit(8) != 8 {
		t.Fatal("a reasonable limit must pass through")
	}
}

func TestJoinLikeBuildsOnlyContentPredicates(t *testing.T) {
	got := joinLike([]string{"@q1", "@q2"})
	if got != "m.content LIKE @q1 OR m.content LIKE @q2" {
		t.Fatalf("unexpected predicate: %s", got)
	}
}

func TestReverseMessages(t *testing.T) {
	in := []contract.MessageRecord{{Content: "1"}, {Content: "2"}, {Content: "3"}}
	reverseMessages(in)
	if in[0].Content != "3" || in[2].Content != "1" {
		t.Fatal("window results must be returned oldest-last")
	}
}

func TestStrategyListMatchesDomain(t *testing.T) {
	// Guards against a divergence between the storage/adapter layer and the
	// domain strategy registry.
	for _, s := range []string{"temporal_exact", "thread", "hybrid", "anaphora", "entity"} {
		if !domain.IsKnownStrategy(s) {
			t.Fatalf("adapter references unknown strategy %q", s)
		}
	}
}
