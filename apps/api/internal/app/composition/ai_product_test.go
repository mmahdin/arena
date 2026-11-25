package composition

import (
	"context"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/config"
	"os"
	"path/filepath"
	"testing"
)

var runtimeExamplePath = filepath.Join("..", "..", "..", "..", "..", "config", "ai-runtime.yaml")

func TestRuntimeFailsClosedWithoutDeploymentAssets(t *testing.T) {
	for _, text := range []string{`{}`, `{"unknown":true}`, `{} {}`, `{"bundleDirectory":"/missing","workerId":"synthetic"}`} {
		p := filepath.Join(t.TempDir(), "runtime.json")
		if e := os.WriteFile(p, []byte(text), 0600); e != nil {
			t.Fatal(e)
		}
		if _, e := loadProduct(context.Background(), nil, p, false); !errors.Is(e, ErrRuntime) {
			t.Fatal("invalid deployment activated", e)
		}
	}
}
func TestV4ExampleLoadsAndDoesNotDependOnKnowledgeModelPins(t *testing.T) {
	cfg, budgets, err := readRuntime(runtimeExamplePath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Schema != "ai_runtime.v4" || budgets.Turn.UnifiedMS != 40000 || len(cfg.LLM.Components) != 5 || budgets.Limits.MaxSyncLLMCalls != 2 {
		t.Fatal(cfg.Schema, budgets.Turn)
	}
	if _, err := cfg.LLM.clientConfig(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join("..", "..", "..", "..", "..", "knowledge", "config", "models.yaml")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("knowledge still owns model selection: %v", err)
	}
}

func TestAPIWorkerLifetimeJoinsAndCannotRunTwice(t *testing.T) {
	a := &API{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- a.Run(ctx) }()
	cancel()
	if e := <-done; !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	if e := a.Run(context.Background()); !errors.Is(e, ErrRuntime) {
		t.Fatal("double worker activation")
	}
}

func TestRuntimeComponentConfigurationUsesDedicatedChains(t *testing.T) {
	keys := []ports.ModelTier{ports.ComponentJEV, ports.ComponentUnified, ports.ComponentMemoryWriter, ports.ComponentReflection, ports.ComponentEmbedding}
	config := runtimeLLM{Components: map[ports.ModelTier]runtimeComponent{}, DefaultTimeoutMS: 60000,
		BreakerWindowMS: 60000, BreakerCooldownMS: 30000, BreakerErrorRate: 0.5, BreakerMinSamples: 5}
	for _, component := range keys {
		m := llm.Model{Provider: "test", Name: string(component)}
		if component == ports.ComponentJEV {
			m = llm.Model{Provider: "OpenCode", Name: "jev-1.13-free", BillingSource: "free"}
		}
		c := runtimeComponent{Chain: []llm.Model{m}, TimeoutMS: 5000, MaxTokens: 256}
		if component == ports.ComponentEmbedding {
			m.Dimensions = 3
			c.Chain = []llm.Model{m}
			c.MaxTokens = 0
		}
		config.Components[component] = c
	}
	if _, err := config.clientConfig(); err != nil {
		t.Fatal(err)
	}
	delete(config.Components, ports.ComponentJEV)
	if _, err := config.clientConfig(); err == nil {
		t.Fatal("missing JEV component accepted")
	}
	config.Components[ports.TierClassifier] = runtimeComponent{Chain: []llm.Model{{Provider: "test", Name: "legacy"}}, TimeoutMS: 5000, MaxTokens: 256}
	if _, err := config.clientConfig(); err == nil {
		t.Fatal("legacy tier accepted by runtime")
	}
}
func TestProductRejectsIncompleteCoreInsteadOfPanicking(t *testing.T) {
	if _, e := NewAPI(config.Config{}, nil, ProductOptions{Core: &AIInternal{}}); !errors.Is(e, ErrRuntime) {
		t.Fatal(e)
	}
}

// Config.Load guards the environment path. This independently protects API
// composition when a caller constructs config.Config without using Load.
func TestProductRejectsManuallyConstructedNonlocalDraft(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  config.Config
	}{
		{"production", config.Config{LocalKBDraft: true, Environment: "production", Address: "127.0.0.1:8080", AICoreConfigFile: "/dev/null"}},
		{"staging", config.Config{LocalKBDraft: true, Environment: "staging", Address: "127.0.0.1:8080", AICoreConfigFile: "/dev/null"}},
		{"public", config.Config{LocalKBDraft: true, Environment: "local", Address: "0.0.0.0:8080", AICoreConfigFile: "/dev/null"}},
		{"no-runtime-config", config.Config{LocalKBDraft: true, Environment: "local", Address: "127.0.0.1:8080"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewAPI(tc.cfg, nil); err != ErrRuntime {
				t.Fatalf("expected local knowledge draft validation to return ErrRuntime, got %T (%v)", err, err)
			}
		})
	}
}

// A safe local draft must reach the real bundle loader with Dev Login off.
// Missing artifacts still fail safely, and do not make a synthetic session mandatory.
func TestLocalKnowledgeDraftDoesNotRequireDevLogin(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-runtime.json")
	_, err := NewAPI(config.Config{
		Environment: "local", Address: "127.0.0.1:8080", LocalKBDraft: true,
		AICoreConfigFile: path,
	}, nil)
	var setup *RuntimeSetupError
	if !errors.As(err, &setup) || setup.Step != "read_config" {
		t.Fatalf("expected safe runtime loader failure (not dev-login gate), got: %v", err)
	}
}

func TestConfiguredCoreLoadsWithoutDevLogin(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-runtime.json")
	_, err := NewAPI(config.Config{Environment: "staging", AICoreConfigFile: path}, nil)
	var setup *RuntimeSetupError
	if !errors.As(err, &setup) || setup.Step != "read_config" {
		t.Fatalf("configured core must be loaded regardless of Dev Login: %v", err)
	}
}

func TestLocalInspectorStillRequiresLocalLoopbackWithoutDevLogin(t *testing.T) {
	for _, tc := range []struct {
		name, environment, address string
		allowed                    bool
	}{
		{"local-loopback", "local", "127.0.0.1:8080", true},
		{"local-public", "local", "0.0.0.0:8080", false},
		{"staging-loopback", "staging", "127.0.0.1:8080", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewAPI(config.Config{
				LocalInspector: true, LocalInspectorDir: filepath.Join(t.TempDir(), "inspector"),
				Environment: tc.environment, Address: tc.address,
			}, nil)
			if (err == nil) != tc.allowed {
				t.Fatalf("inspector gate: %v", err)
			}
		})
	}
}

func TestRuntimeRejectsInvalidCandidateConfiguration(t *testing.T) {
	original, err := os.ReadFile(runtimeExamplePath)
	if err != nil {
		t.Fatal(err)
	}
	scenarios := []string{
		"bundleCandidates: [relative/candidate]",
		"bundleCandidates: [/tmp/a,/tmp/a]",
		"bundleCandidates: [/tmp/a,/tmp/b,/tmp/c,/tmp/d,/tmp/e,/tmp/f]",
	}
	for _, extra := range scenarios {
		path := filepath.Join(t.TempDir(), "runtime.yaml")
		if err := os.WriteFile(path, append(append([]byte{}, original...), []byte("\n"+extra+"\n")...), 0600); err != nil {
			t.Fatal(err)
		}
		_, _, readErr := readRuntime(path)
		var failure *RuntimeSetupError
		if !errors.As(readErr, &failure) || failure.Step != "candidate_config" {
			t.Fatalf("invalid config %s accepted: %v", extra, readErr)
		}
	}
}
