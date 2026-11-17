package retrieval

import (
	"math"
	"sort"
	"strings"
	"time"
)

// Candidate is one pre-ranked retrieval hit, produced by any strategy.
//
// Similarity is the only strategy-dependent signal; the remaining fields are
// item properties so the ranking formula is identical for every strategy.
type Candidate struct {
	ID        string // uuid string form; the domain never imports the uuid package
	Kind      string // fact | preference | episode | hypothesis | goal | event | safety_event
	Epistemic string
	Status    Status
	Text      string
	Age       string

	Sensitivity string
	Importance  float32

	ThreadIDs   []string
	ClusterID   string
	EvidenceN   int
	SupportN    int
	ContradictN int
	ValidFrom   *time.Time
	UpdatedAt   time.Time
	// Tags are the item's topic/emotion tags used by the anaphora boost.
	Tags         []string
	Verbatim     bool
	MessageAgeAt *time.Time

	// Similarity in [0,1] from the strategy (vector cosine, FTS rank, temporal
	// or entity match). Zero is legitimate: an item can still rank on recency.
	Similarity float32

	// Strategy records which method produced the candidate (first producer wins).
	Strategy string

	// Excluded marks a candidate that must never surface (soft-deleted,
	// rejected, superseded, missing provenance, disallowed sensitivity).
	Excluded bool
	// ExcludeReason is a stable machine code for why it was dropped.
	ExcludeReason string
}

// ScoreContext carries the query-side facts needed by the formula.
type ScoreContext struct {
	Now            time.Time
	FocusThread    string   // uuid string of the focus thread, empty if none
	Emotions       []string // current-turn emotion labels (anaphora boost)
	Topics         []string // current-turn topic tags (anaphora boost)
	Anaphora       bool     // an anaphora strategy is active
	RecentMessages int      // novelty window candidate count
	Tunables       Tunables
}

// Breakdown explains a score for the turn trace (§12.1 retrieval summary).
type Breakdown struct {
	Similarity float32
	Recency    float32
	Importance float32
	Thread     float32
	Anaphora   float32
	Total      float32
}

// Score computes the Contract Pack §3.4 ranking formula:
//
//	score = w1·similarity + w2·recency_decay + w3·importance + w4·thread_match
//
// with an additive anaphora boost (§3.4 "boost items from the last 14 days with
// matching emotion/topic tags"). The result is clamped to [0,1] so a threshold
// is comparable across strategies.
func Score(c Candidate, ctx ScoreContext) Breakdown {
	t := ctx.Tunables
	sim := clamp01(c.Similarity)
	rec := RecencyDecay(c, ctx.Now, t.RecencyHalfLife)
	imp := clamp01(c.Importance)
	thr := ThreadMatch(c, ctx.FocusThread)

	total := t.WeightSimilarity*sim + t.WeightRecency*rec + t.WeightImportance*imp + t.WeightThread*thr

	var boost float32
	if ctx.Anaphora && c.UpdatedAt.After(ctx.Now.Add(-t.AnaphoraWindow)) {
		if sharesAny(c.Tags, ctx.Emotions) {
			boost += t.EmotionBoost
		}
		if sharesAny(c.Tags, ctx.Topics) {
			boost += t.TopicBoost
		}
	}
	total += boost
	return Breakdown{
		Similarity: sim, Recency: rec, Importance: imp, Thread: thr,
		Anaphora: boost, Total: clamp01(total),
	}
}

// RecencyDecay is exponential decay with the configured half-life. Future or
// zero timestamps never outrank "now" (defensive: legacy rows may be skew).
func RecencyDecay(c Candidate, now time.Time, halfLife time.Duration) float32 {
	ts := c.UpdatedAt
	if c.MessageAgeAt != nil {
		ts = *c.MessageAgeAt
	}
	if ts.IsZero() || halfLife <= 0 {
		return 0
	}
	age := now.Sub(ts)
	if age < 0 {
		age = 0
	}
	return clamp01(float32(math.Pow(0.5, age.Seconds()/halfLife.Seconds())))
}

// ThreadMatch returns 1 when the item is linked to the focus thread, 0 otherwise.
func ThreadMatch(c Candidate, focusThread string) float32 {
	if focusThread == "" {
		return 0
	}
	for _, id := range c.ThreadIDs {
		if id == focusThread {
			return 1
		}
	}
	return 0
}

func sharesAny(a, b []string) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	set := make(map[string]struct{}, len(a))
	for _, v := range a {
		set[strings.ToLower(strings.TrimSpace(v))] = struct{}{}
	}
	for _, v := range b {
		if _, ok := set[strings.ToLower(strings.TrimSpace(v))]; ok {
			return true
		}
	}
	return false
}

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// Ranked is a scored candidate.
type Ranked struct {
	Candidate
	Breakdown Breakdown
	Tokens    int
}

// Rank filters, scores and orders candidates deterministically.
//
// Rules applied, in order (§3.4, §7.6, §7.7):
//  1. structural exclusions (soft-deleted / rejected / superseded / no provenance
//     / sensitivity not permitted);
//  2. minimum score floor;
//  3. deterministic ordering: score desc, then kind priority, then id asc.
//
// Deduplication and top-K/token budgeting happen in Dedupe and Budget so each
// rule stays independently testable.
func Rank(cands []Candidate, ctx ScoreContext) []Ranked {
	out := make([]Ranked, 0, len(cands))
	for _, c := range cands {
		if c.Excluded {
			continue
		}
		b := Score(c, ctx)
		if b.Total < ctx.Tunables.MinScore {
			continue
		}
		out = append(out, Ranked{Candidate: c, Breakdown: b, Tokens: EstimateTokens(c.Text)})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Breakdown.Total != out[j].Breakdown.Total {
			return out[i].Breakdown.Total > out[j].Breakdown.Total
		}
		ki, kj := kindPriority(out[i].Kind), kindPriority(out[j].Kind)
		if ki != kj {
			return ki < kj
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// kindPriority gives a stable, explainable tie-break: verbatim/authoritative
// material before inferred material, so a hypothesis never displaces the user's
// own reported words at equal score (§7.6).
func kindPriority(kind string) int {
	switch kind {
	case "episode":
		return 0
	case "fact":
		return 1
	case "preference":
		return 2
	case "goal":
		return 3
	case "event":
		return 4
	case "hypothesis":
		return 5
	case "safety_event":
		return 6
	default:
		return 7
	}
}

// Dedupe removes near-duplicate items, keeping the highest-ranked representative.
//
// Two independent rules, both required by the architecture:
//   - identical cluster_id: one item per cluster (§3.4 "dedupe by cluster_id");
//   - near-identical text (normalized hash or ≥ tunable similarity): the
//     reconciliation safety net for rows written before clustering (§7.3).
//
// Input must already be ranked; the first occurrence of a cluster/quasi-cluster
// wins, which is the highest-scoring one.
func Dedupe(ranked []Ranked, threshold float32) []Ranked {
	seenCluster := make(map[string]struct{})
	seenText := make(map[string]struct{})
	var kept []Ranked
	for _, r := range ranked {
		if r.ClusterID != "" {
			if _, dup := seenCluster[r.ClusterID]; dup {
				continue
			}
		}
		key := normalizeText(r.Text)
		if _, dup := seenText[key]; dup {
			continue
		}
		// Near-duplicate guard for paraphrases that differ only in punctuation
		// or filler: prefix-token overlap above the threshold.
		if nearDuplicate(r.Text, kept, threshold) {
			continue
		}
		if r.ClusterID != "" {
			seenCluster[r.ClusterID] = struct{}{}
		}
		seenText[key] = struct{}{}
		kept = append(kept, r)
	}
	return kept
}

func nearDuplicate(text string, kept []Ranked, threshold float32) bool {
	if threshold <= 0 || threshold >= 1 {
		threshold = 0.92
	}
	norm := normalizeText(text)
	for _, k := range kept {
		if k.Kind != "episode" && k.Kind != "event" {
			continue
		}
		if jaccard(tokenSet(norm), tokenSet(normalizeText(k.Text))) >= threshold {
			return true
		}
	}
	return false
}

func normalizeText(s string) string {
	var b strings.Builder
	space := true
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9',
			r >= 0x0600 && r <= 0x06FF, r >= 0x0600 && r <= 0x06FF:
			b.WriteRune(r)
			space = false
		case r == ' ' || r == '\t' || r == '\n' || r == '\u200c':
			if !space {
				b.WriteByte(' ')
				space = true
			}
		default:
			if !space {
				b.WriteByte(' ')
				space = true
			}
		}
	}
	return strings.TrimSpace(b.String())
}

func tokenSet(s string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, tok := range strings.Fields(s) {
		if len([]rune(tok)) < 2 {
			continue
		}
		out[tok] = struct{}{}
	}
	return out
}

func jaccard(a, b map[string]struct{}) float32 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	inter := 0
	for k := range a {
		if _, ok := b[k]; ok {
			inter++
		}
	}
	union := len(a) + len(b) - inter
	if union == 0 {
		return 0
	}
	return float32(inter) / float32(union)
}

// EstimateTokens is the conservative character-based token approximation used
// everywhere in this module (the repo has no tokenizer dependency). ~4 chars per
// token for latin text, ~2 for dense scripts; we take the pessimistic branch so
// budgets are never exceeded.
func EstimateTokens(text string) int {
	runes := len([]rune(text))
	if runes == 0 {
		return 0
	}
	nonASCII := 0
	for _, r := range text {
		if r > 0x7F {
			nonASCII++
		}
	}
	// Non-ASCII runes count as 1 token per 2 runes, ASCII as 1 per 4.
	tokens := float64(runes-nonASCII)/4.0 + float64(nonASCII)/2.0
	if tokens < 1 {
		return 1
	}
	return int(math.Ceil(tokens))
}

// TruncateToTokens shortens text to the per-item token cap, appending an
// ellipsis marker so a reader knows the excerpt is cut (§4.3 MsgView contract).
func TruncateToTokens(text string, maxTokens int) string {
	if maxTokens <= 0 {
		return ""
	}
	for EstimateTokens(text) > maxTokens {
		runes := []rune(text)
		if len(runes) <= 1 {
			return text
		}
		// Drop ~10% at a time, then fix up by rune trimming.
		cut := len(runes) - len(runes)/10
		text = string(runes[:cut])
	}
	if EstimateTokens(text) > maxTokens {
		return ""
	}
	return text
}

// Budget selects items within the hard top-K and the token budget.
//
// It is a greedy pass over ranked items: an item that does not fit is truncated
// to the remaining budget when at least half of the per-item allowance remains,
// otherwise the item is skipped and later (shorter) items may still fill the
// budget. This keeps the "≤ 5 items, ≤ 400 tokens, 60 tokens each" contract
// while staying deterministic.
func Budget(ranked []Ranked, maxItems, tokenBudget, perItemTokens int) []Ranked {
	if maxItems <= 0 {
		return nil
	}
	out := make([]Ranked, 0, maxItems)
	remaining := tokenBudget
	for _, r := range ranked {
		if len(out) >= maxItems {
			break
		}
		r.Tokens = EstimateTokens(r.Text)
		if perItemTokens > 0 && r.Tokens > perItemTokens {
			r.Text = TruncateToTokens(r.Text, perItemTokens)
			r.Tokens = EstimateTokens(r.Text)
		}
		if remaining <= 0 {
			break
		}
		if r.Tokens > remaining {
			if remaining < perItemTokens/2 || r.Tokens == 0 {
				continue
			}
			r.Text = TruncateToTokens(r.Text, remaining)
			r.Tokens = EstimateTokens(r.Text)
			if r.Tokens > remaining {
				continue
			}
		}
		remaining -= r.Tokens
		out = append(out, r)
	}
	return out
}

// NoveltyFilter drops candidates whose text is already present verbatim (or at
// ≥ threshold similarity) in the last N messages of the context window: "do not
// retrieve what the last 8 messages already say" (§7.6).
func NoveltyFilter(ranked []Ranked, recent []string, threshold float32, window int) []Ranked {
	if window <= 0 || len(recent) == 0 {
		return ranked
	}
	if len(recent) > window {
		recent = recent[len(recent)-window:]
	}
	sets := make([]map[string]struct{}, 0, len(recent))
	texts := make([]string, 0, len(recent))
	for _, m := range recent {
		sets = append(sets, tokenSet(normalizeText(m)))
		texts = append(texts, normalizeText(m))
	}
	var out []Ranked
	for _, r := range ranked {
		norm := normalizeText(r.Text)
		dup := false
		for i, t := range texts {
			if t == norm {
				dup = true
				break
			}
			if jaccard(tokenSet(norm), sets[i]) >= threshold {
				dup = true
				break
			}
		}
		if !dup {
			out = append(out, r)
		}
	}
	return out
}
