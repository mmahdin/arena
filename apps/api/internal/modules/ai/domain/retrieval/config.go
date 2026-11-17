// Package retrieval contains the pure, deterministic retrieval rules of the AI
// Core (WP-09): relevance scoring, ranking, deduplication, context-budget
// computation, temporal-expression resolution, anaphora detection and entity
// extraction.
//
// Everything here is side-effect free: no SQL, no HTTP, no clock reads, no LLM
// calls. Time is always injected. That keeps every rule unit-testable and keeps
// the "SQL only in MSSQL adapters" boundary of the Modular Monolith intact.
package retrieval

import (
	"errors"
	"fmt"
	"time"
)

// Tunables mirror `config/retrieval.yaml` from Contract Pack §8.5. They are
// configuration, not clinical policy: values below are the contract defaults.
type Tunables struct {
	// Defaults section.
	MaxItems          int
	MinScore          float32
	PerItemTokens     int
	HardCapWithHop    int
	TotalTokensBudget int

	// Weights section. Must sum to ~1.0 (validated).
	WeightSimilarity float32
	WeightRecency    float32
	WeightImportance float32
	WeightThread     float32

	// Recency section.
	RecencyHalfLife time.Duration

	// Anaphora section.
	AnaphoraWindow time.Duration
	EmotionBoost   float32
	TopicBoost     float32

	// Temporal section.
	TemporalMaxMessages int
	TemporalRoles       []string
	WeekStartsOnMonday  bool

	// Hypothesis surfacing rules.
	HypothesisStatuses       []Status
	HypothesisRequireThread  bool
	ExcludeHighlySensitive   bool
	IncludeContestedInRecall bool

	// Novelty filter.
	NoveltyRecentMessages int
	NoveltySimilarity     float32

	// Timeouts.
	TotalTimeout        time.Duration
	PerStrategyTimeout  time.Duration
	CandidatePoolLimit  int
	EmbeddingDimensions int
}

// Status mirrors contract.MemoryStatus without importing the contract package,
// so the domain layer stays dependency-free (stdlib + UUID only rule, §2.3).
type Status string

const (
	StatusActive     Status = "active"
	StatusContested  Status = "contested"
	StatusSuperseded Status = "superseded"
	StatusRejected   Status = "rejected"
	StatusStale      Status = "stale"
	StatusProposed   Status = "proposed"
	StatusSupported  Status = "supported"
)

// Defaults returns the Contract Pack §8.5 `config/retrieval.yaml` values.
func Defaults() Tunables {
	return Tunables{
		MaxItems:          5,
		MinScore:          0.32,
		PerItemTokens:     60,
		HardCapWithHop:    7,
		TotalTokensBudget: 400,

		WeightSimilarity: 0.50,
		WeightRecency:    0.20,
		WeightImportance: 0.15,
		WeightThread:     0.15,

		RecencyHalfLife: 21 * 24 * time.Hour,

		AnaphoraWindow: 14 * 24 * time.Hour,
		EmotionBoost:   0.15,
		TopicBoost:     0.10,

		TemporalMaxMessages: 12,
		TemporalRoles:       []string{"user"},
		WeekStartsOnMonday:  true,

		HypothesisStatuses:      []Status{StatusProposed, StatusSupported},
		HypothesisRequireThread: true,

		ExcludeHighlySensitive: true,

		NoveltyRecentMessages: 8,
		NoveltySimilarity:     0.92,

		TotalTimeout:        300 * time.Millisecond,
		PerStrategyTimeout:  200 * time.Millisecond,
		CandidatePoolLimit:  300,
		EmbeddingDimensions: 0, // 0 = accept the model-reported dimension (OPEN-4)
	}
}

// ErrInvalidTunables marks a configuration that cannot be used safely.
var ErrInvalidTunables = errors.New("invalid retrieval configuration")

// Normalize fills zero values with defaults and rejects unsafe combinations.
// A misconfigured retriever fails at construction, not per request.
func (t Tunables) Normalize() (Tunables, error) {
	d := Defaults()
	if t.MaxItems <= 0 {
		t.MaxItems = d.MaxItems
	}
	if t.MaxItems > 20 {
		return t, fmt.Errorf("%w: max_items must be <= 20", ErrInvalidTunables)
	}
	if t.MinScore == 0 {
		t.MinScore = d.MinScore
	}
	if t.PerItemTokens <= 0 {
		t.PerItemTokens = d.PerItemTokens
	}
	if t.HardCapWithHop <= 0 {
		t.HardCapWithHop = d.HardCapWithHop
	}
	if t.HardCapWithHop < t.MaxItems {
		return t, fmt.Errorf("%w: hard_cap_with_hop must be >= max_items", ErrInvalidTunables)
	}
	if t.TotalTokensBudget <= 0 {
		t.TotalTokensBudget = d.TotalTokensBudget
	}
	if t.WeightSimilarity == 0 && t.WeightRecency == 0 && t.WeightImportance == 0 && t.WeightThread == 0 {
		t.WeightSimilarity, t.WeightRecency, t.WeightImportance, t.WeightThread =
			d.WeightSimilarity, d.WeightRecency, d.WeightImportance, d.WeightThread
	}
	sum := t.WeightSimilarity + t.WeightRecency + t.WeightImportance + t.WeightThread
	if sum < 0.95 || sum > 1.05 {
		return t, fmt.Errorf("%w: weights must sum to 1.0 (got %.3f)", ErrInvalidTunables, sum)
	}
	if t.WeightSimilarity < 0 || t.WeightRecency < 0 || t.WeightImportance < 0 || t.WeightThread < 0 {
		return t, fmt.Errorf("%w: weights must be non-negative", ErrInvalidTunables)
	}
	if t.RecencyHalfLife <= 0 {
		t.RecencyHalfLife = d.RecencyHalfLife
	}
	if t.AnaphoraWindow <= 0 {
		t.AnaphoraWindow = d.AnaphoraWindow
	}
	if t.EmotionBoost == 0 {
		t.EmotionBoost = d.EmotionBoost
	}
	if t.TopicBoost == 0 {
		t.TopicBoost = d.TopicBoost
	}
	if t.EmotionBoost < 0 || t.TopicBoost < 0 {
		return t, fmt.Errorf("%w: anaphora boosts must be non-negative", ErrInvalidTunables)
	}
	if t.TemporalMaxMessages <= 0 {
		t.TemporalMaxMessages = d.TemporalMaxMessages
	}
	if len(t.TemporalRoles) == 0 {
		t.TemporalRoles = d.TemporalRoles
	}
	for _, r := range t.TemporalRoles {
		if r != "user" && r != "assistant" {
			return t, fmt.Errorf("%w: temporal roles must be user/assistant", ErrInvalidTunables)
		}
	}
	if len(t.HypothesisStatuses) == 0 {
		t.HypothesisStatuses = d.HypothesisStatuses
	}
	for _, s := range t.HypothesisStatuses {
		if s != StatusProposed && s != StatusSupported {
			return t, fmt.Errorf("%w: hypotheses may only surface proposed/supported (§7.5)", ErrInvalidTunables)
		}
	}
	if t.NoveltyRecentMessages < 0 {
		return t, fmt.Errorf("%w: novelty window must be >= 0", ErrInvalidTunables)
	}
	if t.NoveltyRecentMessages == 0 {
		t.NoveltyRecentMessages = d.NoveltyRecentMessages
	}
	if t.NoveltySimilarity == 0 {
		t.NoveltySimilarity = d.NoveltySimilarity
	}
	if t.TotalTimeout <= 0 {
		t.TotalTimeout = d.TotalTimeout
	}
	if t.PerStrategyTimeout <= 0 {
		t.PerStrategyTimeout = d.PerStrategyTimeout
	}
	if t.PerStrategyTimeout > t.TotalTimeout {
		return t, fmt.Errorf("%w: per_strategy_ms must be <= total_ms", ErrInvalidTunables)
	}
	if t.CandidatePoolLimit <= 0 {
		t.CandidatePoolLimit = d.CandidatePoolLimit
	}
	if t.CandidatePoolLimit > 2000 {
		return t, fmt.Errorf("%w: candidate pool must stay bounded (<= 2000)", ErrInvalidTunables)
	}
	return t, nil
}

// ItemCap is the hard top-K for this request, including the optional extra hop.
func (t Tunables) ItemCap(hops int) int {
	cap := t.MaxItems
	if hops > 0 {
		cap = t.MaxItems + hops
	}
	if cap > t.HardCapWithHop {
		cap = t.HardCapWithHop
	}
	return cap
}
