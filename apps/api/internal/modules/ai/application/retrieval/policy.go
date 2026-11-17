// Package retrieval implements the WP-09 deterministic Retrieval Service from
// Contract Pack v1.2 §4.4 and the Final Architecture Document §3.4.
//
// It is speculative and deterministic: it runs in parallel with the Safety
// Pre-Gate before the Brain, uses only cheap lexical/temporal signals, and the
// embedding model is its single model dependency. It never calls a reasoning
// LLM and never imports the experimental prototype pipeline.
package retrieval

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// Config is the validated, immutable view of `config/retrieval.yaml` plus the
// two WP-03 budgets retrieval is allowed to observe. All behavioural limits
// come from the pinned bundle; none are hard-coded clinical policy.
type Config struct {
	Ref        c.BundleRef
	Synthetic  bool
	MaxItems   int
	MinScore   float64
	PerItemTok int
	HardCap    int
	Weights    Weights
	HalfLife   time.Duration
	Anaphora   Anaphora
	Temporal   Temporal
	Hypotheses Hypotheses
	Novelty    Novelty
	Timeout    time.Duration
	Strategy   time.Duration
	// WeekStart is used for weekday-relative temporal phrases; it comes from
	// the bundle's temporal configuration, not from Go's default week.
	WeekStart time.Weekday
	// ContextBudget is the WorkingContext token target ("wctx_target"). It is
	// an upper bound only: retrieval never pads to reach it.
	ContextBudget int
}

// Weights are the v1.2 relevance weights and must sum to one.
type Weights struct {
	Similarity, Recency, Importance, ThreadMatch float64
}

type Anaphora struct {
	RecentWindowDays int
	EmotionBoost     float64
	TopicBoost       float64
}

type Temporal struct {
	MaxMessages  int
	DefaultRoles []string
	WeekStartsOn time.Weekday
}

type Hypotheses struct {
	IncludeStatuses    []c.MemoryStatus
	RequireThreadMatch bool
}

type Novelty struct {
	SkipIfInRecent int
	SimilarityGTE  float64
}

// LoadConfig pins one Knowledge Bundle for the whole operation and decodes the
// retrieval/budget artifacts strictly. Unknown fields, wrong schema versions or
// unsafe values are errors rather than silently invented defaults.
func LoadConfig(ctx context.Context, reg ports.BundleRegistry, userID c.ID) (Config, error) {
	var out Config
	if reg == nil {
		return out, errors.New("WP-09 retrieval requires a Knowledge Bundle registry")
	}
	if !c.ValidID(userID) {
		return out, errors.New("retrieval owner identity required")
	}
	pinned, err := reg.Active(ctx, userID)
	if err != nil {
		return out, fmt.Errorf("pin retrieval bundle: %w", err)
	}
	if pinned == nil || pinned.Reference().Version == "" {
		return out, errors.New("no pinned retrieval bundle")
	}
	out.Ref = pinned.Reference()
	if s, ok := pinned.(interface{ SyntheticOnly() bool }); ok {
		out.Synthetic = s.SyntheticOnly()
	}
	var rc bundle.Retrieval
	raw, err := pinned.Artifact(ctx, "config/retrieval.yaml")
	if err != nil {
		return Config{}, fmt.Errorf("load retrieval config: %w", err)
	}
	if err = decodeStrict(raw, &rc); err != nil {
		return Config{}, fmt.Errorf("decode retrieval config: %w", err)
	}
	var budgets bundle.Budgets
	raw, err = pinned.Artifact(ctx, "config/budgets.yaml")
	if err != nil {
		return Config{}, fmt.Errorf("load budgets: %w", err)
	}
	if err = decodeStrict(raw, &budgets); err != nil {
		return Config{}, fmt.Errorf("decode budgets: %w", err)
	}
	out.MaxItems = rc.Defaults.MaxItems
	out.MinScore = rc.Defaults.MinScore
	out.PerItemTok = rc.Defaults.PerItemTokens
	out.HardCap = rc.Defaults.HardCapWithHop
	out.Weights = Weights{Similarity: rc.Weights.Similarity, Recency: rc.Weights.Recency,
		Importance: rc.Weights.Importance, ThreadMatch: rc.Weights.ThreadMatch}
	out.HalfLife = time.Duration(rc.Recency.HalfLifeDays) * 24 * time.Hour
	out.Anaphora = Anaphora{RecentWindowDays: rc.Anaphora.RecentWindowDays,
		EmotionBoost: rc.Anaphora.EmotionBoost, TopicBoost: rc.Anaphora.TopicBoost}
	out.Temporal = Temporal{MaxMessages: rc.Temporal.MaxMessages, DefaultRoles: append([]string(nil), rc.Temporal.DefaultRoles...),
		WeekStartsOn: parseWeekday(rc.Temporal.WeekStartsOn)}
	out.WeekStart = out.Temporal.WeekStartsOn
	for _, s := range rc.Hypotheses.IncludeStatuses {
		st := c.MemoryStatus(s)
		if !st.Valid() {
			return Config{}, fmt.Errorf("unknown hypothesis status %q", s)
		}
		out.Hypotheses.IncludeStatuses = append(out.Hypotheses.IncludeStatuses, st)
	}
	out.Hypotheses.RequireThreadMatch = rc.Hypotheses.RequireThreadMatch
	out.Novelty = Novelty{SkipIfInRecent: rc.NoveltyFilter.SkipIfInRecentMessages, SimilarityGTE: rc.NoveltyFilter.SimilarityGTE}
	out.Timeout = time.Duration(rc.Timeouts.TotalMS) * time.Millisecond
	out.Strategy = time.Duration(rc.Timeouts.PerStrategyMS) * time.Millisecond
	out.ContextBudget = budgets.Tokens.WCtxTarget
	if err = out.validate(rc.Schema, budgets.Schema, rc.Temporal.TZSource); err != nil {
		return Config{}, err
	}
	return out, nil
}

// validate enforces the contract's anti-excess invariants. A misconfigured
// bundle fails closed; it may not silently fall back to larger budgets.
func (p Config) validate(retrievalSchema, budgetsSchema, tzSource string) error {
	if retrievalSchema != "retrieval.v1" || budgetsSchema != "budgets.v1" {
		return errors.New("unsupported retrieval/budget bundle schema")
	}
	if p.MaxItems < 1 || p.MaxItems > 5 {
		return fmt.Errorf("retrieval max_items %d outside contract hard top-K", p.MaxItems)
	}
	if p.HardCap < p.MaxItems || p.HardCap > 7 {
		return fmt.Errorf("retrieval hard_cap_with_hop %d outside contract bound", p.HardCap)
	}
	if math.IsNaN(p.MinScore) || p.MinScore < 0 || p.MinScore > 1 {
		return errors.New("retrieval min_score outside [0,1]")
	}
	if p.PerItemTok < 1 || p.PerItemTok > 200 {
		return errors.New("retrieval per_item_tokens outside contract range")
	}
	if p.ContextBudget < 1 {
		return errors.New("retrieval requires a positive WorkingContext token target")
	}
	w := p.Weights
	if math.Abs(w.Similarity+w.Recency+w.Importance+w.ThreadMatch-1) > 0.0001 ||
		w.Similarity < 0 || w.Recency < 0 || w.Importance < 0 || w.ThreadMatch < 0 {
		return errors.New("retrieval weights must be nonnegative and sum to one")
	}
	if p.HalfLife <= 0 || p.Anaphora.RecentWindowDays < 1 || p.Anaphora.RecentWindowDays > 90 {
		return errors.New("invalid recency or anaphora window")
	}
	if p.Anaphora.EmotionBoost < 0 || p.Anaphora.EmotionBoost > 0.5 || p.Anaphora.TopicBoost < 0 || p.Anaphora.TopicBoost > 0.5 {
		return errors.New("anaphora boosts outside contract range")
	}
	if p.Temporal.MaxMessages < 1 || p.Temporal.MaxMessages > 50 {
		return errors.New("temporal max_messages outside contract range")
	}
	if len(p.Temporal.DefaultRoles) == 0 {
		return errors.New("temporal default_roles required")
	}
	for _, role := range p.Temporal.DefaultRoles {
		if role != string(c.RoleUser) && role != string(c.RoleAssistant) {
			return fmt.Errorf("temporal role %q is not a chat role", role)
		}
	}
	// The contract pins tz_source to user_state; accepting another source would
	// silently shift every temporal window.
	if tzSource != "user_state" {
		return fmt.Errorf("unsupported temporal tz_source %q", tzSource)
	}
	if len(p.Hypotheses.IncludeStatuses) == 0 {
		return errors.New("hypothesis include_statuses required")
	}
	for _, st := range p.Hypotheses.IncludeStatuses {
		if st != c.MemProposed && st != c.MemSupported {
			return fmt.Errorf("hypothesis status %q may not be retrieved", st)
		}
	}
	if !p.Hypotheses.RequireThreadMatch {
		return errors.New("hypotheses must require an active-thread match")
	}
	if p.Novelty.SkipIfInRecent < 0 || p.Novelty.SkipIfInRecent > 32 ||
		p.Novelty.SimilarityGTE <= 0 || p.Novelty.SimilarityGTE > 1 {
		return errors.New("invalid novelty filter configuration")
	}
	if p.Timeout <= 0 || p.Strategy <= 0 || p.Strategy > p.Timeout {
		return errors.New("invalid retrieval timeouts")
	}
	return nil
}

func parseWeekday(s string) time.Weekday {
	switch s {
	case "sunday":
		return time.Sunday
	case "tuesday":
		return time.Tuesday
	case "wednesday":
		return time.Wednesday
	case "thursday":
		return time.Thursday
	case "friday":
		return time.Friday
	case "saturday":
		return time.Saturday
	default:
		return time.Monday
	}
}

func decodeStrict(raw []byte, out any) error {
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	if err := dec.Decode(out); err != nil {
		return err
	}
	var extra yaml.Node
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("more than one YAML document")
		}
		return err
	}
	return nil
}
