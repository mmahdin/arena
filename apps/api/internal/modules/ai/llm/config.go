// Package llm implements WP-04's in-process provider-neutral model client.
// It does not import or invoke the experimental AI application or its gateway.
package llm

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// Model is one configured rung in a per-component primary -> secondary -> fallback
// chain. Costs are micro-USD per million tokens (not per token). Cached input
// tokens are part of InTokens, and charged at CachedMicroUSDPerM instead.
// These prices are configuration inputs, never assumed or fetched from a model.
type Model struct {
	BillingSource    string `json:"billing_source,omitempty" yaml:"billing_source,omitempty"`
	Provider         string `json:"provider" yaml:"provider"`
	Name             string `json:"model" yaml:"model"`
	NativeJSONSchema bool   `json:"native_json_schema" yaml:"native_json_schema"`
	// Nil preserves strict output for existing native-schema configurations.
	JSONSchemaStrict    *bool  `json:"json_schema_strict,omitempty" yaml:"json_schema_strict,omitempty"`
	SupportsTemperature *bool  `json:"supports_temperature,omitempty" yaml:"supports_temperature,omitempty"`
	ReasoningEffort     string `json:"reasoning_effort,omitempty" yaml:"reasoning_effort,omitempty"`
	InputMicroUSDPerM   int64  `json:"input_micro_usd_per_m" yaml:"input_micro_usd_per_m"`
	OutputMicroUSDPerM  int64  `json:"output_micro_usd_per_m" yaml:"output_micro_usd_per_m"`
	CachedMicroUSDPerM  int64  `json:"cached_micro_usd_per_m" yaml:"cached_micro_usd_per_m"`
	Dimensions          int    `json:"dimensions,omitempty" yaml:"dimensions,omitempty"`
}

type Config struct {
	AttemptWriteTimeout time.Duration `json:"attempt_write_timeout"`
	// Every execution component has an explicit ordered chain, including fallback.
	// An absent chain is an error; an absent provider is never silently replaced.
	Chains             map[ports.ModelTier][]Model `json:"chains"`
	MaxRetriesPerModel int                         `json:"max_retries_per_model"`
	DefaultTimeout     time.Duration               `json:"default_timeout"`
	RetryBase          time.Duration               `json:"retry_base"`
	RetryMax           time.Duration               `json:"retry_max"`
	BreakerWindow      time.Duration               `json:"breaker_window"`
	BreakerCooldown    time.Duration               `json:"breaker_cooldown"`
	BreakerErrorRate   float64                     `json:"breaker_error_rate"`
	BreakerMinSamples  int                         `json:"breaker_min_samples"`
	DailyQuota         DailyQuota                  `json:"daily_quota"`
}

type DailyQuota struct {
	Calls        int   `json:"calls"` // 0 means unlimited
	InTokens     int64 `json:"in_tokens"`
	OutTokens    int64 `json:"out_tokens"`
	CostMicroUSD int64 `json:"cost_micro_usd"`
}

var ErrConfig = errors.New("llm configuration invalid")

func (c Config) Validate() error {
	if c.AttemptWriteTimeout < 0 || c.DefaultTimeout <= 0 || c.MaxRetriesPerModel < 0 || c.RetryBase < 0 || c.RetryMax < c.RetryBase ||
		c.BreakerWindow <= 0 || c.BreakerCooldown <= 0 || c.BreakerErrorRate <= 0 || c.BreakerErrorRate > 1 || c.BreakerMinSamples < 1 ||
		c.DailyQuota.Calls < 0 || c.DailyQuota.InTokens < 0 || c.DailyQuota.OutTokens < 0 || c.DailyQuota.CostMicroUSD < 0 {
		return ErrConfig
	}
	componentKeys := []ports.ModelTier{ports.ComponentSafety, ports.ComponentRouter, ports.ComponentBrain, ports.ComponentGenerator,
		ports.ComponentValidator, ports.ComponentMemoryWriter, ports.ComponentReflection, ports.ComponentEmbedding}
	compatKeys := []ports.ModelTier{ports.ComponentSafety, ports.ComponentBrain, ports.ComponentGenerator,
		ports.ComponentValidator, ports.ComponentMemoryWriter, ports.ComponentReflection, ports.ComponentEmbedding}
	legacyKeys := []ports.ModelTier{ports.TierSmall, ports.TierStrong, ports.TierClassifier, ports.TierEmbedding}
	validShape := func(keys []ports.ModelTier) bool {
		if len(c.Chains) != len(keys) {
			return false
		}
		for _, key := range keys {
			if len(c.Chains[key]) == 0 {
				return false
			}
		}
		return true
	}
	unifiedKeys := []ports.ModelTier{ports.ComponentUnified, ports.ComponentMemoryWriter, ports.ComponentReflection, ports.ComponentEmbedding}
	jevKeys := append([]ports.ModelTier{ports.ComponentJEV}, unifiedKeys...)
	if !validShape(jevKeys) && !validShape(unifiedKeys) && !validShape(componentKeys) && !validShape(compatKeys) && !validShape(legacyKeys) {
		return ErrConfig
	}
	for tier, chain := range c.Chains {
		if !tier.Valid() || len(chain) == 0 || len(chain) > 8 {
			return ErrConfig
		}
		seen := make(map[string]bool, len(chain))
		for _, model := range chain {
			if model.BillingSource != "" && (model.BillingSource != "free" || tier != ports.ComponentJEV || model.Provider != "OpenCode" || model.Name != "jev-1.13-free" || model.InputMicroUSDPerM != 0 || model.OutputMicroUSDPerM != 0 || model.CachedMicroUSDPerM != 0) {
				return ErrConfig
			}
			if strings.TrimSpace(model.Provider) != model.Provider || model.Provider == "" || strings.TrimSpace(model.Name) != model.Name || model.Name == "" ||
				strings.ContainsAny(model.Provider+model.Name, "\r\n") || model.InputMicroUSDPerM < 0 || model.OutputMicroUSDPerM < 0 || model.CachedMicroUSDPerM < 0 ||
				model.Dimensions < 0 || model.Dimensions > 8192 || ((tier == ports.ComponentEmbedding || tier == ports.TierEmbedding) && model.Dimensions == 0) || ((tier != ports.ComponentEmbedding && tier != ports.TierEmbedding) && model.Dimensions != 0) {
				return fmt.Errorf("%w: invalid model tier entry", ErrConfig)
			}
			key := model.Provider + "\x00" + model.Name
			if seen[key] {
				return ErrConfig
			}
			seen[key] = true
		}
	}
	return nil
}

// CostMicroUSD rounds up to the nearest micro-dollar so tiny billable calls
// cannot disappear from accounting. Invalid/overflowed provider counters fail.
func CostMicroUSD(model Model, in, out, cached int) (int64, error) {
	if in < 0 || out < 0 || cached < 0 || cached > in {
		return 0, ErrUsage
	}
	items := [][2]int64{{int64(in - cached), model.InputMicroUSDPerM}, {int64(cached), model.CachedMicroUSDPerM}, {int64(out), model.OutputMicroUSDPerM}}
	var total int64
	for _, item := range items {
		if item[1] < 0 || (item[0] != 0 && item[1] > math.MaxInt64/item[0]) {
			return 0, ErrUsage
		}
		x := item[0] * item[1]
		amount := x / 1_000_000
		if x%1_000_000 > 0 {
			amount++
		}
		if amount > math.MaxInt64-total {
			return 0, ErrUsage
		}
		total += amount
	}
	return total, nil
}
