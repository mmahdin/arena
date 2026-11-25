package composition

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/llmhttp"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/memorycrypto"
	aidb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/background"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/embedding"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/metrics"
	"gopkg.in/yaml.v3"
)

var ErrRuntime = errors.New("AI Core deployment configuration unavailable")

// RuntimeSetupError contains only an approved, non-sensitive startup stage.
type RuntimeSetupError struct{ Step string }

func (e *RuntimeSetupError) Error() string { return ErrRuntime.Error() }
func (e *RuntimeSetupError) Unwrap() error { return ErrRuntime }
func runtimeStep(step string) error        { return &RuntimeSetupError{Step: step} }

type runtimeComponent struct {
	Chain       []llm.Model `json:"chain"`
	TimeoutMS   int         `json:"timeout_ms"`
	MaxTokens   int         `json:"max_tokens"`
	Temperature float32     `json:"temperature"`
}

type runtimeLLM struct {
	Components         map[ports.ModelTier]runtimeComponent `json:"components"`
	MaxRetriesPerModel int                                  `json:"max_retries_per_model"`
	DefaultTimeoutMS   int                                  `json:"default_timeout_ms"`
	RetryBaseMS        int                                  `json:"retry_base_ms"`
	RetryMaxMS         int                                  `json:"retry_max_ms"`
	BreakerWindowMS    int                                  `json:"breaker_window_ms"`
	BreakerCooldownMS  int                                  `json:"breaker_cooldown_ms"`
	BreakerErrorRate   float64                              `json:"breaker_error_rate"`
	BreakerMinSamples  int                                  `json:"breaker_min_samples"`
	DailyQuota         llm.DailyQuota                       `json:"daily_quota"`
}

// Component keys, model identities, and per-operation deadlines are specified
// exactly once in the operator-owned runtime file; Knowledge never selects a model.
func (r runtimeLLM) clientConfig() (llm.Config, error) {
	keys := []ports.ModelTier{ports.ComponentJEV, ports.ComponentUnified, ports.ComponentMemoryWriter, ports.ComponentReflection, ports.ComponentEmbedding}
	if len(r.Components) != len(keys) || r.DefaultTimeoutMS < 1 {
		return llm.Config{}, ErrRuntime
	}
	cfg := llm.Config{Chains: make(map[ports.ModelTier][]llm.Model), MaxRetriesPerModel: r.MaxRetriesPerModel,
		DefaultTimeout: time.Duration(r.DefaultTimeoutMS) * time.Millisecond,
		RetryBase:      time.Duration(r.RetryBaseMS) * time.Millisecond, RetryMax: time.Duration(r.RetryMaxMS) * time.Millisecond,
		BreakerWindow: time.Duration(r.BreakerWindowMS) * time.Millisecond, BreakerCooldown: time.Duration(r.BreakerCooldownMS) * time.Millisecond,
		BreakerErrorRate: r.BreakerErrorRate, BreakerMinSamples: r.BreakerMinSamples, DailyQuota: r.DailyQuota}
	for _, component := range keys {
		c, ok := r.Components[component]
		if !ok || c.TimeoutMS < 1 || c.Temperature < 0 || c.Temperature > 2 {
			return llm.Config{}, ErrRuntime
		}
		if component == ports.ComponentEmbedding {
			if c.MaxTokens != 0 || len(c.Chain) != 1 {
				return llm.Config{}, ErrRuntime
			}
		} else if c.MaxTokens < 1 {
			return llm.Config{}, ErrRuntime
		}
		if component == ports.ComponentJEV {
			if len(c.Chain) != 1 || c.Chain[0].Provider != "OpenCode" || c.Chain[0].Name != "jev-1.13-free" || c.Chain[0].BillingSource != "free" || c.Chain[0].NativeJSONSchema {
				return llm.Config{}, ErrRuntime
			}
		} else {
			for _, model := range c.Chain {
				if model.Provider == "OpenCode" {
					return llm.Config{}, ErrRuntime
				}
			}
		}
		cfg.Chains[component] = append([]llm.Model(nil), c.Chain...)
	}
	if err := cfg.Validate(); err != nil {
		return llm.Config{}, err
	}
	return cfg, nil
}

type deploymentSettings struct {
	Background        background.Config        `json:"background"`
	RetrievalDefaults bundle.RetrievalDefaults `json:"retrievalDefaults"`
	EmbeddingCache    embedding.Config         `json:"embeddingCache"`
	Schema            string                   `json:"schema"`
	BundleDirectory   string                   `json:"bundleDirectory"`
	BundleCandidates  []string                 `json:"bundleCandidates"` // optional, read-only, validated at startup
	TrustedKeys       map[string]string        `json:"trustedKeys"`
	Providers         map[string]struct {
		BaseURL string `json:"baseUrl"`
		KeyEnv  string `json:"keyEnv"`
	} `json:"providers"`
	LLM        runtimeLLM `json:"llm"`
	Encryption struct {
		Active string            `json:"active"`
		KeyEnv map[string]string `json:"keyEnv"`
	} `json:"encryption"`
	Policy   aidb.ProductPolicySettings `json:"policy"`
	WorkerID string                     `json:"workerId"`
	Feedback mem.FeedbackPolicy         `json:"feedback"`
	// Budgets is decoded separately with yaml tags; JSON strict decoding rejects
	// unknown keys everywhere else without weakening privacy-policy parsing.
	Budgets           json.RawMessage `json:"budgets"`
	RetrievalTimeouts struct {
		TotalMS       int `json:"total_ms"`
		PerStrategyMS int `json:"per_strategy_ms"`
	} `json:"retrievalTimeouts"`
}

func readRuntime(path string) (deploymentSettings, bundle.Budgets, error) {
	var cfg deploymentSettings
	var budgets bundle.Budgets
	raw, err := os.ReadFile(path)
	if err != nil || len(raw) > 65536 {
		return cfg, budgets, runtimeStep("read_config")
	}
	// YAML v3 rejects duplicate keys; converting its scalar/map tree to JSON
	// lets us reuse all existing strict json tags on product/security policy structs.
	var tree any
	yamlDecoder := yaml.NewDecoder(bytes.NewReader(raw))
	if yamlDecoder.Decode(&tree) != nil || yamlDecoder.Decode(new(any)) != io.EOF {
		return cfg, budgets, runtimeStep("config_schema")
	}
	normalized, err := json.Marshal(tree)
	if err != nil {
		return cfg, budgets, runtimeStep("config_schema")
	}
	decoder := json.NewDecoder(bytes.NewReader(normalized))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&cfg) != nil || decoder.Decode(new(any)) != io.EOF || cfg.Schema != "ai_runtime.v4" || cfg.BundleDirectory == "" || cfg.WorkerID == "" {
		return cfg, budgets, runtimeStep("config_schema")
	}
	if cfg.EmbeddingCache.Validate() != nil {
		return cfg, budgets, runtimeStep("embedding_cache")
	}
	budgetDecoder := yaml.NewDecoder(bytes.NewReader(cfg.Budgets))
	budgetDecoder.KnownFields(true)
	if budgetDecoder.Decode(&budgets) != nil || budgetDecoder.Decode(new(any)) != io.EOF {
		return cfg, budgets, runtimeStep("budget_schema")
	}
	// Candidate roots are operator-authored, never accepted from Admin HTTP.
	// No scan of a parent folder or filesystem outside these fixed roots.
	if len(cfg.BundleCandidates) > 5 {
		return cfg, budgets, runtimeStep("candidate_config")
	}
	roots := map[string]bool{filepath.Clean(cfg.BundleDirectory): true}
	for _, root := range cfg.BundleCandidates {
		clean := filepath.Clean(root)
		if !filepath.IsAbs(root) || roots[clean] {
			return cfg, budgets, runtimeStep("candidate_config")
		}
		roots[clean] = true
	}
	if !cfg.Background.Valid() || cfg.RetrievalDefaults.MaxItems < 1 || cfg.RetrievalDefaults.PerItemTokens < 1 || cfg.RetrievalDefaults.TotalTokens < cfg.RetrievalDefaults.PerItemTokens || cfg.RetrievalDefaults.HardCapWithHop < cfg.RetrievalDefaults.MaxItems {
		return cfg, budgets, runtimeStep("operational_config")
	}
	if !validRuntimeBudgets(budgets) {
		return cfg, budgets, runtimeStep("budget_validation")
	}
	if cfg.RetrievalTimeouts.TotalMS < 1 || cfg.RetrievalTimeouts.PerStrategyMS < 1 ||
		cfg.RetrievalTimeouts.PerStrategyMS > cfg.RetrievalTimeouts.TotalMS {
		return cfg, budgets, runtimeStep("retrieval_budget")
	}
	if _, err := cfg.LLM.clientConfig(); err != nil {
		return cfg, budgets, runtimeStep("model_config")
	}
	return cfg, budgets, nil
}

func validRuntimeBudgets(b bundle.Budgets) bool {
	t := b.Turn
	if b.Schema != "budgets.v1" || t.TotalMS <= 0 || t.RetrievalMS <= 0 || t.JEVMS <= 0 || t.UnifiedMS <= 0 || t.UnifiedMS > t.TotalMS || t.JEVMS > t.TotalMS || t.RetrievalMS > t.TotalMS || b.Limits.MaxSyncLLMCalls < 1 || b.Limits.MaxRegenerations < 0 || b.Limits.MaxBrainHops < 0 || b.Quotas.UserMsgsPerMinute < 1 || b.Quotas.ReflectionPerUserPerDay < 1 || b.Quotas.MemoryWriterPerUserPerHour < 1 {
		return false
	}
	for _, value := range []int{b.Operations.SchedulerIntervalMS, b.Operations.SchedulerRetryBaseMS, b.Operations.SchedulerRetryMaxMS, b.Operations.SchedulerMaxFailures, b.Operations.HintReadMS, b.Operations.AttemptWriteMS, b.Operations.StageWriteMS, b.Operations.TraceReferenceMS, b.Operations.TraceJobsMS, b.Context.Active, b.Context.Persistent, b.Context.Profile, b.Context.Retrieved, b.Context.Recent, b.Tokens.WCtxTarget, b.Tokens.UnifiedIn, b.Tokens.UnifiedOut, b.Tokens.JEVIn, b.Tokens.KnowledgeContent, b.Tokens.MemoryWriterIn, b.Tokens.ReflectionIn,
		b.Operations.TurnLeaseMS, b.Operations.TurnRenewMS, b.Operations.TurnCleanupMS, b.Operations.WorkerLeaseMS, b.Operations.WorkerTimeoutMS, b.Operations.WorkerCleanupMS, b.Operations.HintVerificationMS, b.Operations.HeartbeatMS, b.Operations.HeartbeatIntervalMS, b.Operations.RecentMessages} {
		if value <= 0 {
			return false
		}
	}
	return b.Operations.SchedulerRetryMaxMS >= b.Operations.SchedulerRetryBaseMS && b.Operations.TurnRenewMS < b.Operations.TurnLeaseMS && b.Operations.WorkerTimeoutMS < b.Operations.WorkerLeaseMS

}

// runtimeBundle supplies operational budgets to legacy asset consumers through
// the existing Bundle port. It never adds them to the signed knowledge content,
// its hash, approvals, or release manifest. The runtime file is their sole source.
type runtimeBundle struct {
	*bundle.Snapshot
	budgets    []byte
	retrieval  []byte
	background []byte
}

func (b runtimeBundle) Artifact(ctx context.Context, name string) ([]byte, error) {
	if name == "config/budgets.yaml" {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return append([]byte(nil), b.budgets...), nil
	}
	if name == "config/background.yaml" {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return append([]byte(nil), b.background...), nil
	}
	if name == "config/retrieval.yaml" {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return append([]byte(nil), b.retrieval...), nil
	}
	return b.Snapshot.Artifact(ctx, name)
}
func (b runtimeBundle) ArtifactNames() []string {
	return append(b.Snapshot.ArtifactNames(), "config/budgets.yaml")
}

func loadProduct(ctx context.Context, db *sql.DB, path string, localDraft bool) (ProductOptions, error) {
	var out ProductOptions
	cfg, budgets, err := readRuntime(path)
	if err != nil {
		return out, err
	}
	if cfg.Schema != "ai_runtime.v4" {
		return out, runtimeStep("unified_runtime_required")
	}
	trusted := map[string]ed25519.PublicKey{}
	for id, v := range cfg.TrustedKeys {
		b, err := base64.StdEncoding.DecodeString(v)
		if err != nil || len(b) != ed25519.PublicKeySize {
			return out, runtimeStep("trusted_key")
		}
		trusted[id] = b
	}
	pinned, activeReport, err := bundle.Read(ctx, cfg.BundleDirectory, bundle.Options{CoreVersion: "1.0.0", Release: !localDraft, TrustedKeys: trusted})
	if err != nil || (localDraft && !pinned.SyntheticOnly()) {
		return out, runtimeStep("bundle_read")
	}
	// Metadata is computed only from validated immutable snapshots. Candidate
	// verification failures do not affect the active runtime or expose paths.
	out.BundleInventory = inspectDeploymentBundles(ctx, pinned, activeReport, cfg.BundleCandidates, localDraft, trusted)
	providers := map[string]llm.Provider{}
	for name, p := range cfg.Providers {
		if !strings.HasPrefix(p.KeyEnv, "MODEL_") && !(name == "OpenCode" && p.KeyEnv == "OPENCODE_API_KEY") {
			return out, runtimeStep("provider_key_env")
		}
		if inspector := trace.InspectorFromContext(ctx); inspector != nil {
			inspector.RegisterSecret(os.Getenv(p.KeyEnv))
		}
		constructor := llmhttp.New
		if name == "OpenCode" {
			if p.KeyEnv != "OPENCODE_API_KEY" {
				return out, runtimeStep("provider_key_env")
			}
			constructor = llmhttp.NewSystemOne
		}
		provider, err := constructor(p.BaseURL, os.Getenv(p.KeyEnv), nil)
		if err != nil {
			return out, runtimeStep("provider_init")
		}
		providers[name] = provider
	}
	modelConfig, err := cfg.LLM.clientConfig()
	if err != nil {
		return out, runtimeStep("model_config")
	}
	telemetryStore, err := aidb.NewTelemetryStore(db)
	if err != nil {
		return out, runtimeStep("telemetry_init")
	}
	var journal llm.AttemptJournal = telemetryStore
	if meter := metrics.FromContext(ctx); meter != nil {
		journal = meteredAttemptJournal{next: telemetryStore, meter: meter}
	}
	modelConfig.AttemptWriteTimeout = time.Duration(budgets.Operations.AttemptWriteMS) * time.Millisecond
	rawClient, err := llm.New(modelConfig, providers, llm.Options{Attempts: journal})
	if err != nil {
		return out, runtimeStep("model_client")
	}
	client := configuredComponentClient{next: rawClient, components: cfg.LLM.Components}
	keys := map[string][]byte{}
	for name, keyEnv := range cfg.Encryption.KeyEnv {
		raw, err := base64.StdEncoding.DecodeString(os.Getenv(keyEnv))
		if err != nil || !strings.HasPrefix(keyEnv, "AI_") {
			return out, runtimeStep("encryption_key")
		}
		keys[name] = raw
	}
	cipher, err := memorycrypto.New(cfg.Encryption.Active, keys)
	if err != nil {
		return out, runtimeStep("encryption_init")
	}
	policy, err := aidb.NewProductUsePolicy(db, cfg.Policy)
	if err != nil {
		return out, runtimeStep("use_policy")
	}
	embed := cfg.LLM.Components[ports.ComponentEmbedding].Chain[0]
	// Supply runtime budgets via a read-only virtual artifact. No Knowledge
	// version/hash/checksum changes when operational deadlines are adjusted.
	budgetRaw, err := yaml.Marshal(budgets)
	if err != nil {
		return out, runtimeStep("budget_encode")
	}
	rawRetrieval, err := pinned.Artifact(ctx, "config/retrieval.yaml")
	if err != nil {
		return out, runtimeStep("retrieval_read")
	}
	var retrieval bundle.Retrieval
	if err := yaml.Unmarshal(rawRetrieval, &retrieval); err != nil {
		return out, runtimeStep("retrieval_decode")
	}
	retrieval.Defaults = cfg.RetrievalDefaults
	retrieval.Timeouts.TotalMS = cfg.RetrievalTimeouts.TotalMS
	retrieval.Timeouts.PerStrategyMS = cfg.RetrievalTimeouts.PerStrategyMS
	retrievalRaw, err := yaml.Marshal(retrieval)
	if err != nil {
		return out, runtimeStep("retrieval_encode")
	}
	backgroundRaw, err := yaml.Marshal(struct {
		Schema            string `yaml:"schema"`
		background.Config `yaml:",inline"`
	}{background.OutputSchema, cfg.Background})
	if err != nil {
		return out, runtimeStep("background_encode")
	}
	activeBundle := runtimeBundle{Snapshot: pinned, budgets: budgetRaw, retrieval: retrievalRaw, background: backgroundRaw}
	core, err := NewAIInternal(ctx, AIInternalOptions{Inspector: trace.InspectorFromContext(ctx), DB: db, Bundle: activeBundle,
		Client: client, Embeddings: client, EmbeddingCache: cfg.EmbeddingCache, UsePolicy: policy, MemoryPolicy: policy, Cipher: cipher,
		EmbeddingModel: embed.Name, Dimension: embed.Dimensions, WorkerID: cfg.WorkerID,
		Feedback: cfg.Feedback, Tracer: loggingTracer{logger: slog.Default(), stages: telemetryStore, writeTimeout: time.Duration(budgets.Operations.StageWriteMS) * time.Millisecond}, WorkerPulse: telemetryStore, DisableProactive: true,
		AdmissionLimit: budgets.Quotas.UserMsgsPerMinute, AllowSyntheticForTests: localDraft,
		TurnTimeout: time.Duration(budgets.Turn.TotalMS) * time.Millisecond,
		Budgets:     budgets,
		JEVTimeout:  time.Duration(cfg.LLM.Components[ports.ComponentJEV].TimeoutMS) * time.Millisecond, UnifiedTimeout: time.Duration(cfg.LLM.Components[ports.ComponentUnified].TimeoutMS) * time.Millisecond, UnifiedInputTokens: budgets.Tokens.UnifiedIn, UnifiedOutputTokens: budgets.Tokens.UnifiedOut})
	if err != nil {
		return out, runtimeStep("core_init")
	}
	out.Core = core
	out.ConsentKeys = append([]string(nil), cfg.Policy.ConsentKeys...)
	sort.Strings(out.ConsentKeys)
	out.ChatConsentKey = cfg.Policy.ChatConsentKey
	return out, nil
}
