package retrieval

import (
	"errors"
	"math"
	"strings"
)

// ErrDimensionMismatch is returned when two vectors cannot be compared.
// A caller must treat it as "no embedding available" rather than as similarity 0
// silently mixed with real vectors — §7.3 requires rejecting mismatched
// dimensions before cosine ranking.
var ErrDimensionMismatch = errors.New("embedding dimension mismatch")

// ErrEmptyVector marks a stored vector with no values.
var ErrEmptyVector = errors.New("empty embedding vector")

// CosineSimilarity returns cosine similarity in [0,1] (negative values clamped to
// 0: opposite-meaning text is not "relevant context" and must not be pulled in
// by a mean-shifted vector).
func CosineSimilarity(a, b []float32) (float32, error) {
	if len(a) == 0 || len(b) == 0 {
		return 0, ErrEmptyVector
	}
	if len(a) != len(b) {
		return 0, ErrDimensionMismatch
	}
	var dot, na, nb float64
	for i := range a {
		x, y := float64(a[i]), float64(b[i])
		dot += x * y
		na += x * x
		nb += y * y
	}
	if na == 0 || nb == 0 {
		return 0, ErrEmptyVector
	}
	cos := dot / (math.Sqrt(na) * math.Sqrt(nb))
	if cos < 0 {
		return 0, nil
	}
	if cos > 1 {
		return 1, nil
	}
	return float32(cos), nil
}

// LexicalSimilarity is the deterministic fallback used when an item has no
// stored embedding (missing-embedding case) or when the embedding provider is
// unavailable. It is token-overlap based so it can never hallucinate a semantic
// relation: it only reports shared words.
func LexicalSimilarity(query, text string) float32 {
	q := tokenSet(normalizeText(query))
	t := tokenSet(normalizeText(text))
	if len(q) == 0 || len(t) == 0 {
		return 0
	}
	// Query coverage is the right measure for retrieval: how much of what the
	// user asked about is present in the item.
	hit := 0
	for tok := range q {
		if _, ok := t[tok]; ok {
			hit++
		}
	}
	return clamp01(float32(hit) / float32(len(q)))
}

// KeywordMatch reports whether any query term occurs in the text (LIKE-level
// candidate selection signal, used by the SQL adapter contract).
func KeywordMatch(terms []string, text string) bool {
	if len(terms) == 0 {
		return true
	}
	lower := strings.ToLower(text)
	for _, term := range terms {
		if term == "" {
			continue
		}
		if strings.Contains(lower, strings.ToLower(term)) {
			return true
		}
	}
	return false
}

// EqualFoldTolerant compares two labels ignoring case, whitespace and the
// Persian zero-width non-joiner, so tag matching is locale-safe without pulling
// in a normalization library.
func EqualFoldTolerant(a, b string) bool {
	norm := func(s string) string {
		s = strings.ReplaceAll(s, "\u200c", "")
		s = strings.ReplaceAll(s, "\u200f", "")
		s = strings.ReplaceAll(s, "\u200e", "")
		return strings.ToLower(strings.TrimSpace(s))
	}
	na, nb := norm(a), norm(b)
	return na != "" && na == nb
}
