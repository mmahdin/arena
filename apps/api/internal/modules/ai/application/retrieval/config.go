package retrieval

import (
	"bytes"
	"context"
	"errors"
	"io"

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
	if cfg.Schema != "retrieval.v2" || cfg.Defaults.MaxItems < 1 || cfg.Defaults.PerItemTokens < 1 || cfg.Defaults.TotalTokens < cfg.Defaults.PerItemTokens || cfg.Timeouts.TotalMS < 1 || cfg.Timeouts.PerStrategyMS < 1 || cfg.Timeouts.PerStrategyMS > cfg.Timeouts.TotalMS {
		return cfg, errors.New("invalid hybrid retrieval configuration")
	}
	return cfg, nil
}
