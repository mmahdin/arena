package retrieval

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math"

	"gopkg.in/yaml.v3"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

func loadConfig(ctx context.Context, handle ports.Bundle) (bundle.Retrieval, error) {
	var cfg bundle.Retrieval
	if handle == nil || handle.Reference().Version == "" {
		return cfg, errors.New("WP-09 needs a pinned knowledge bundle")
	}
	raw, err := handle.Artifact(ctx, "config/retrieval.yaml")
	if err != nil {
		return cfg, err
	}
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return cfg, err
	}
	var trailing yaml.Node
	if err := dec.Decode(&trailing); err != io.EOF {
		return cfg, errors.New("retrieval configuration must be one YAML document")
	}
	w := cfg.Weights
	if cfg.Schema != "retrieval.v1" ||
		cfg.Defaults.MaxItems < 1 || cfg.Defaults.MaxItems > 7 ||
		cfg.Defaults.HardCapWithHop < cfg.Defaults.MaxItems || cfg.Defaults.HardCapWithHop > 7 ||
		cfg.Defaults.PerItemTokens < 1 || cfg.Defaults.PerItemTokens > 400 ||
		!between(cfg.Defaults.MinScore, 0, 1) ||
		!between(w.Similarity, 0, 1) || !between(w.Recency, 0, 1) ||
		!between(w.Importance, 0, 1) || !between(w.ThreadMatch, 0, 1) ||
		math.Abs(w.Similarity+w.Recency+w.Importance+w.ThreadMatch-1) > 0.0001 ||
		cfg.Recency.HalfLifeDays < 1 || cfg.Recency.HalfLifeDays > 3650 ||
		cfg.Temporal.MaxMessages < 1 || cfg.Temporal.MaxMessages > 100 ||
		cfg.Temporal.WeekStartsOn != "monday" || cfg.Temporal.TZSource != "user_state" ||
		cfg.Anaphora.RecentWindowDays < 1 || cfg.Anaphora.RecentWindowDays > 365 ||
		!between(cfg.Anaphora.EmotionBoost, 0, 1) || !between(cfg.Anaphora.TopicBoost, 0, 1) ||
		cfg.NoveltyFilter.SkipIfInRecentMessages < 0 || cfg.NoveltyFilter.SkipIfInRecentMessages > 40 ||
		!between(cfg.NoveltyFilter.SimilarityGTE, 0, 1) ||
		cfg.Timeouts.TotalMS < 1 || cfg.Timeouts.TotalMS > 5000 ||
		cfg.Timeouts.PerStrategyMS < 1 || cfg.Timeouts.PerStrategyMS > cfg.Timeouts.TotalMS {
		return cfg, errors.New("invalid WP-09 retrieval configuration")
	}
	return cfg, nil
}

func between(v, low, high float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= low && v <= high
}
