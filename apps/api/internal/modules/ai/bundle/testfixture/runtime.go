// Package testfixture gives isolated module tests the same runtime-owned
// operational budgets as the real composition root, without copying them into
// the Psychology Knowledge Bundle or relying on live providers.
package testfixture

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"gopkg.in/yaml.v3"
)

type Overlay struct {
	ports.Bundle
	budgets, retrieval []byte
}

func (o Overlay) Artifact(ctx context.Context, name string) ([]byte, error) {
	if name == "config/budgets.yaml" {
		return append([]byte(nil), o.budgets...), nil
	}
	if name == "config/retrieval.yaml" {
		return append([]byte(nil), o.retrieval...), nil
	}
	return o.Bundle.Artifact(ctx, name)
}
func (o Overlay) SyntheticOnly() bool {
	source, ok := o.Bundle.(interface{ SyntheticOnly() bool })
	return ok && source.SyntheticOnly()
}

func (o Overlay) ArtifactNames() []string {
	if source, ok := o.Bundle.(interface{ ArtifactNames() []string }); ok {
		return append(append([]string(nil), source.ArtifactNames()...), "config/budgets.yaml")
	}
	return []string{"config/budgets.yaml"}
}

func WithRuntimeSettings(t *testing.T, snapshot ports.Bundle) Overlay {
	t.Helper()
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime fixture path unavailable")
	}
	// From .../apps/api/internal/modules/ai/bundle/testfixture/ to repo root.
	root := filepath.Clean(filepath.Join(filepath.Dir(self), "..", "..", "..", "..", "..", "..", ".."))
	data, err := os.ReadFile(filepath.Join(root, "config", "ai-runtime.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Budgets           bundle.Budgets `yaml:"budgets"`
		RetrievalTimeouts struct {
			TotalMS       int `yaml:"total_ms"`
			PerStrategyMS int `yaml:"per_strategy_ms"`
		} `yaml:"retrievalTimeouts"`
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	// Historical Brain/Generator asset regressions use the old budget slots.
	// They are test fixtures only; active v3 configuration has no such stages.
	if config.Budgets.Turn.UnifiedMS > 0 {
		config.Budgets.Turn.SafetyMS = 5000
		config.Budgets.Turn.RouterMS = 1800
		config.Budgets.Turn.BrainMS = 12000
		config.Budgets.Turn.GenFirstTokenMS = 10000
		config.Budgets.Turn.GenTotalMS = 30000
		config.Budgets.Tokens.BrainIn = 3500
		config.Budgets.Tokens.BrainOut = 700
		config.Budgets.Tokens.GenIn = 4500
		config.Budgets.Tokens.GenOut = 600
		config.Budgets.Tokens.WCtxTarget = 1500
	}
	budgets, err := yaml.Marshal(config.Budgets)
	if err != nil {
		t.Fatal(err)
	}
	retrievalSource, err := snapshot.Artifact(context.Background(), "config/retrieval.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var retrieval bundle.Retrieval
	if err := yaml.Unmarshal(retrievalSource, &retrieval); err != nil {
		t.Fatal(err)
	}
	retrieval.Timeouts.TotalMS = config.RetrievalTimeouts.TotalMS
	retrieval.Timeouts.PerStrategyMS = config.RetrievalTimeouts.PerStrategyMS
	retrievalData, err := yaml.Marshal(retrieval)
	if err != nil {
		t.Fatal(err)
	}
	return Overlay{Bundle: snapshot, budgets: budgets, retrieval: retrievalData}
}
