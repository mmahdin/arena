package retrieval

import (
	"math"
	"sort"
	"strings"
	"time"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// candidate is one retrieved record before scoring. It carries provenance back
// to the stored record so the ranked MemoryView can cite what it came from.
type candidate struct {
	// Memory-backed candidate (memory item / episode / hypothesis).
	Item *c.MemoryItem
	// Message-backed candidate (exact/temporal recall or an in-thread quote).
	Message *messageRecord
	// Similarity is the cosine similarity in [0,1], or zero when no comparable
	// stored vector exists. A zero contribution is deliberate: an absent
	// embedding is not evidence of relevance, and treating it as such would
	// retrieve every recent memory on every turn.
	Similarity float64
	// Match is the structural match strength in [0,1] produced by an exact
	// lookup: an entity role match, an exact temporal window, a thread link, or
	// an anaphora emotion/tag hit. Only strategies that matched structurally set
	// it, and it is never derived from user-visible text heuristically. It acts
	// as the deterministic equivalent of similarity when no vector exists —
	// which is exactly the "right method for the query type" the architecture
	// requires — and it can never exceed a real cosine match.
	Match float64
	ThreadMatch bool
	// Strategy records which retrieval path produced the candidate.
	Strategy c.RetrievalStrategy
	// Boost is the bounded anaphora adjustment applied before ranking.
	Boost float64
	// Exact marks a verbatim message returned by exact/temporal recall. Such a
	// record IS the answer to the question and must not be discarded by the
	// semantic relevance floor (§4.4 scenario 2: no semantic substitution).
	Exact bool
}

// messageRecord is an owner-scoped raw message read from Conversation storage.
// Content is not logged or traced by this package.
type messageRecord struct {
	ID        c.ID
	TurnID    c.ID
	Role      c.Role
	Text      string
	CreatedAt time.Time
	// Embedding is optional; a missing embedding is a normal degraded signal.
	Similarity float64
}

// score is the v1.2 ranking formula, applied literally:
//
//	score = w1·similarity + w2·recency_decay + w3·importance + w4·thread_match
//
// Every component is normalized to [0,1] before weighting, so the final score is
// directly comparable to `defaults.min_score` from the bundle. Each component
// therefore contributes exactly its configured weight and never more.
//
// A component with no data contributes zero and the weights are NOT
// renormalized. That is the anti-excess behaviour the architecture requires: if
// stored embeddings are missing (or no embedding model is configured), similarity
// contributes nothing, so the same `min_score` floor admits fewer items and
// retrieval leans on the temporal/thread/entity/lexical paths instead of
// returning every recent memory.
func score(p Config, cand candidate, now time.Time) float64 {
	w := p.Weights
	// The similarity component is the stronger of the semantic cosine and the
	// structural match. An exact entity/temporal/thread match therefore ranks
	// high without an embedding, while an unrelated recent memory still gains
	// nothing from the absence of a vector.
	similarity := clamp01(cand.Similarity)
	if cand.Match > similarity {
		similarity = clamp01(cand.Match)
	}
	var total float64
	total += w.Similarity * similarity
	total += w.Recency * recencyDecay(now, candidateTime(cand), p.HalfLife)
	if v, ok := importance(cand); ok {
		total += w.Importance * float64(v)
	}
	if cand.ThreadMatch {
		total += w.ThreadMatch
	}
	return clamp01(total + cand.Boost)
}

// candidateTime is the record's relevance time: last confirmation when the user
// re-affirmed an item, otherwise its update (or creation) time. Raw messages use
// their own creation time.
func candidateTime(cand candidate) time.Time {
	if cand.Item != nil {
		item := cand.Item
		if item.LastConfirmedAt != nil && item.LastConfirmedAt.After(item.UpdatedAt) {
			return *item.LastConfirmedAt
		}
		if !item.UpdatedAt.IsZero() {
			return item.UpdatedAt
		}
		return item.CreatedAt
	}
	if cand.Message != nil {
		return cand.Message.CreatedAt
	}
	return time.Time{}
}

// importance returns the stored importance, clamped. Non-finite values are
// treated as an absent component rather than as a perfect match.
func importance(cand candidate) (float32, bool) {
	if cand.Item == nil {
		return 0, false
	}
	v := cand.Item.Importance
	if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
		return 0, false
	}
	return clamp01f(v), true
}

func clamp01f(v float32) float32 {
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}

// recencyDecay is an exponential half-life decay bounded to [0,1]. Future
// timestamps (clock skew) clamp to 1 rather than exceeding the weight budget.
func recencyDecay(now, at time.Time, halfLife time.Duration) float64 {
	if at.IsZero() || halfLife <= 0 {
		return 0
	}
	age := now.Sub(at)
	if age <= 0 {
		return 1
	}
	return math.Pow(0.5, float64(age)/float64(halfLife))
}

func clamp01(v float64) float64 {
	switch {
	case math.IsNaN(v), v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}

// cosine returns the cosine similarity of two already-decoded vectors in
// [0,1] (negative similarity is clamped to 0: an inverted vector is not a
// match). Mismatched or empty dimensions return ok=false, never a fake score.
func cosine(a, b []float32) (float64, bool) {
	if len(a) == 0 || len(a) != len(b) {
		return 0, false
	}
	var dot, na, nb float64
	for i := range a {
		av, bv := float64(a[i]), float64(b[i])
		if math.IsNaN(av) || math.IsNaN(bv) || math.IsInf(av, 0) || math.IsInf(bv, 0) {
			return 0, false
		}
		dot += av * bv
		na += av * av
		nb += bv * bv
	}
	if na == 0 || nb == 0 {
		return 0, false
	}
	return clamp01(dot / (math.Sqrt(na) * math.Sqrt(nb))), true
}

// rank filters, de-duplicates, sorts and budget-caps the candidate pool.
//
// Order of operations is contractual:
//  1. eligibility filters already applied by the caller (status, sensitivity,
//     hypothesis thread match, visibility);
//  2. minimum score threshold;
//  3. de-duplication by cluster_id, then by item id, then by text;
//  4. deterministic ordering (score desc, then recency desc, then id);
//  5. per-item token cap and total WorkingContext token budget.
func rank(p Config, candidates []candidate, exclude map[c.ID]bool, now time.Time, limit, budgetTokens int) []c.MemoryView {
	scored := make([]struct {
		score float64
		cand  candidate
	}, 0, len(candidates))
	for _, cand := range candidates {
		if cand.Item != nil && exclude[cand.Item.ID] {
			continue
		}
		if cand.Message != nil && exclude[cand.Message.ID] {
			continue
		}
		s := score(p, cand, now)
		// Exact recall is not a relevance ranking: the messages in the requested
		// window are the answer, so the semantic floor cannot discard them.
		if s < p.MinScore && !cand.Exact {
			continue
		}
		scored = append(scored, struct {
			score float64
			cand  candidate
		}{score: s, cand: cand})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		ti, tj := candidateTime(scored[i].cand), candidateTime(scored[j].cand)
		if !ti.Equal(tj) {
			return ti.After(tj)
		}
		return candidateID(scored[i].cand).String() < candidateID(scored[j].cand).String()
	})
	seenCluster := map[c.ID]bool{}
	seenText := map[string]bool{}
	out := make([]c.MemoryView, 0, len(scored))
	used := 0
	for _, s := range scored {
		cand := s.cand
		if cand.Item != nil {
			hasCluster := cand.Item.ClusterID != nil && *cand.Item.ClusterID != (c.ID{})
			if hasCluster && seenCluster[*cand.Item.ClusterID] {
				continue // de-duplicate by cluster_id (v1.2 §3.4)
			}
			key := normalizeText(cand.Item.Content)
			if key != "" && seenText[key] {
				continue
			}
			if key != "" {
				seenText[key] = true
			}
			if hasCluster {
				seenCluster[*cand.Item.ClusterID] = true
			}
		}
		view, ok := memoryView(p, cand, now, s.score)
		if !ok {
			continue
		}
		tokens := estimateTokens(view.Text)
		if used+tokens > budgetTokens && len(out) > 0 {
			continue
		}
		used += tokens
		out = append(out, view)
		if len(out) == limit {
			break
		}
	}
	return out
}

func candidateID(cand candidate) c.ID {
	if cand.Item != nil {
		return cand.Item.ID
	}
	if cand.Message != nil {
		return cand.Message.ID
	}
	return c.ID{}
}

// memoryView renders a stored record into the bounded WorkingContext item.
// Text is word-capped (per_item_tokens) and marked with an ellipsis when cut so
// the Brain can tell a truncated memory from a complete one.
func memoryView(p Config, cand candidate, now time.Time, s float64) (c.MemoryView, bool) {
	if cand.Item != nil {
		item := cand.Item
		if !c.ValidID(item.ID) || !item.Kind.Valid() || !item.Epistemic.Valid() || !item.Status.Valid() {
			return c.MemoryView{}, false
		}
		text := item.Content
		if strings.TrimSpace(text) == "" {
			// Encrypted-only or empty content is not usable context. It is
			// dropped rather than replaced with an invented summary.
			return c.MemoryView{}, false
		}
		view := c.MemoryView{
			ID: item.ID, Kind: item.Kind, Epistemic: item.Epistemic, Status: item.Status,
			Text:      truncateWords(text, p.PerItemTok),
			Age:       ageLabel(now, candidateTime(cand)),
			EvidenceN: len(item.Evidence), Sensitivity: item.Sensitivity,
			Verbatim: false, Score: float32(s),
		}
		if item.ValidFrom != nil {
			v := item.ValidFrom.UTC().Format("2006-01-02")
			view.ValidAt = &v
		}
		return view, true
	}
	if cand.Message != nil {
		msg := cand.Message
		if !c.ValidID(msg.ID) || !msg.Role.Valid() || strings.TrimSpace(msg.Text) == "" {
			return c.MemoryView{}, false
		}
		// Raw messages are returned verbatim (never summarized) but still
		// bounded, and flagged Verbatim so the Brain must quote, not infer.
		return c.MemoryView{
			ID: msg.ID, Kind: c.MemEpisode, Epistemic: c.EpReported, Status: c.MemActive,
			Text: truncateWords(msg.Text, p.PerItemTok), Age: ageLabel(now, msg.CreatedAt),
			Sensitivity: c.SensNormal, Verbatim: true, Score: float32(s),
		}, true
	}
	return c.MemoryView{}, false
}

func ageLabel(now, at time.Time) string {
	if at.IsZero() {
		return "unknown"
	}
	d := now.Sub(at)
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Hour:
		return itoa(int(d.Minutes())) + "m"
	case d < 24*time.Hour:
		return itoa(int(d.Hours())) + "h"
	default:
		return itoa(int(d.Hours()/24)) + "d"
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// truncateWords applies the per-item budget. The ellipsis is appended only
// when content was actually removed.
func truncateWords(s string, maxWords int) string {
	words := strings.Fields(s)
	if maxWords < 1 || len(words) <= maxWords {
		return strings.TrimSpace(s)
	}
	return strings.Join(words[:maxWords], " ") + "…"
}

// estimateTokens is a deliberately conservative approximation shared with the
// assembler's budget check: max(words, ceil(bytes/4)). It is NOT tokenizer
// certification; the WorkingContext target is enforced with the same estimate
// so retrieval and assembly cannot disagree.
func estimateTokens(s string) int {
	if s == "" {
		return 0
	}
	words := len(strings.Fields(s))
	b := (len([]byte(s)) + 3) / 4
	if words > b {
		return words
	}
	return b
}

func normalizeText(s string) string {
	low := strings.ToLower(strings.TrimSpace(s))
	return strings.Join(strings.Fields(low), " ")
}
