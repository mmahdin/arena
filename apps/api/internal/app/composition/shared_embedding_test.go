package composition

import (
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"testing"
)

func TestSharedEmbeddingRuntimeDefaultsValidationAndResolvedIdentity(t *testing.T) {
	raw, e := os.ReadFile(runtimeExamplePath)
	if e != nil {
		t.Fatal(e)
	}
	var original map[string]any
	if e = yaml.Unmarshal(raw, &original); e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct {
		name     string
		settings any
		valid    bool
	}{
		{"absent", nil, true}, {"explicit", map[string]any{"ttl_seconds": 900, "max_entries": 4096, "max_in_flight": 64}, true},
		{"negative_ttl", map[string]any{"ttl_seconds": -1}, false}, {"unbounded_capacity", map[string]any{"max_entries": 65537}, false},
		{"unbounded_flights", map[string]any{"max_in_flight": 1025}, false}, {"unknown", map[string]any{"raw_text_keys": true}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.settings == nil {
				delete(original, "embeddingCache")
			} else {
				original["embeddingCache"] = tc.settings
			}
			data, _ := yaml.Marshal(original)
			path := filepath.Join(t.TempDir(), "runtime.yaml")
			if e = os.WriteFile(path, data, 0600); e != nil {
				t.Fatal(e)
			}
			cfg, _, e := readRuntime(path)
			if (e == nil) != tc.valid {
				t.Fatal("runtime validation mismatch", e)
			}
			if tc.valid {
				if cfg.EmbeddingCache.TTLSeconds != 900 || cfg.EmbeddingCache.MaxEntries != 4096 || cfg.EmbeddingCache.MaxInFlight != 64 {
					t.Fatal(cfg.EmbeddingCache)
				}
				profile, e := (configuredComponentClient{components: cfg.LLM.Components}).EmbeddingProfile(ports.ComponentEmbedding)
				model := cfg.LLM.Components[ports.ComponentEmbedding].Chain[0]
				if e != nil || len(profile.Models) != 1 || profile.Models[0].Provider != model.Provider || profile.Models[0].Model != model.Name || profile.Models[0].Dimensions != model.Dimensions || profile.ExecutionTimeout.Milliseconds() != int64(cfg.LLM.Components[ports.ComponentEmbedding].TimeoutMS) {
					t.Fatal("effective identity diverged from runtime", profile, e)
				}
			}
		})
	}
}
