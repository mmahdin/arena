package llm

import (
	"context"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"
	"log/slog"
	"math"
	"math/rand"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract/schema"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// Client implements the v1.2 Structured/Stream contract and the additive
// EmbeddingClient. It is not wired to the prototype turn runtime (WP-05).
type Client struct {
	config    Config
	providers map[string]Provider
	ledger    UsageLedger
	metrics   Metrics
	breakers  *breakers
	now       func() time.Time
	jitter    func(time.Duration) time.Duration
}

// Options are test seams, not global overrides. A nil ledger gets the
// non-durable MemoryLedger. Never construct a production client with the
// in-memory ledger when hard cross-instance quotas are required.
type Options struct {
	Ledger   UsageLedger
	Metrics  Metrics
	Attempts AttemptJournal
	Now      func() time.Time
	Jitter   func(time.Duration) time.Duration
}

var _ ports.Client = (*Client)(nil)
var _ ports.EmbeddingClient = (*Client)(nil)

func New(config Config, providers map[string]Provider, opts Options) (*Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	copyProviders := make(map[string]Provider, len(providers))
	for name, provider := range providers {
		if name == "" || provider == nil {
			return nil, ErrConfig
		}
		copyProviders[name] = inspectedProvider{next: provider}
	}
	chains := make(map[ports.ModelTier][]Model, len(config.Chains))
	for tier, chain := range config.Chains {
		chains[tier] = append([]Model(nil), chain...)
		for _, model := range chain {
			if copyProviders[model.Provider] == nil {
				return nil, ErrConfig
			}
		}
	}
	config.Chains = chains
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Jitter == nil {
		opts.Jitter = func(max time.Duration) time.Duration {
			if max <= 0 {
				return 0
			}
			return time.Duration(rand.Int63n(int64(max) + 1))
		}
	}
	if opts.Ledger == nil {
		opts.Ledger = NewMemoryLedger(opts.Now)
	}
	if opts.Attempts != nil {
		for name, provider := range copyProviders {
			copyProviders[name] = meteredProvider{next: provider, journal: opts.Attempts, now: opts.Now, writeTimeout: config.AttemptWriteTimeout}
		}
	}
	return &Client{config: config, providers: copyProviders, ledger: opts.Ledger, metrics: opts.Metrics,
		breakers: newBreakers(config), now: opts.Now, jitter: opts.Jitter}, nil
}

func (c *Client) begin(ctx context.Context, meta contract.CallMeta, timeout time.Duration) (context.Context, context.CancelFunc, error) {
	if ctx == nil || !contract.ValidID(meta.UserID) {
		return nil, nil, ErrConfig
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if contract.ValidID(meta.TurnID) {
		ctx = observability.WithTurnRef(ctx, observability.TurnRef(meta.TurnID))
	}
	if err := c.ledger.Check(ctx, meta.UserID, c.config.DailyQuota); err != nil {
		return nil, nil, localFailure(err)
	}
	if timeout <= 0 {
		timeout = c.config.DefaultTimeout
	}
	if budget, ok := contract.BudgetFromContext(ctx); ok && !budget.TotalDeadline.IsZero() {
		if deadline, exists := ctx.Deadline(); !exists || budget.TotalDeadline.Before(deadline) {
			var cancel context.CancelFunc
			ctx, cancel = context.WithDeadline(ctx, budget.TotalDeadline)
			bounded, cancelTimeout := context.WithTimeout(ctx, timeout)
			return bounded, func() { cancelTimeout(); cancel() }, nil
		}
	}
	bounded, cancel := context.WithTimeout(ctx, timeout)
	return bounded, cancel, nil
}

func (c *Client) preflight(ctx context.Context, user contract.ID, attempts int, tier ports.ModelTier) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if budget, ok := contract.BudgetFromContext(ctx); ok && budget.MaxLLMCalls > 0 && attempts >= budget.MaxLLMCalls {
		return &contract.Error{Code: contract.ErrBudget, Reason: "budget_exceeded", Origin: observability.Source(0)}
	}
	if err := c.ledger.Check(ctx, user, c.config.DailyQuota); err != nil {
		return err
	}
	if (tier == ports.ComponentJEV || tier == ports.ComponentUnified || tier == ports.TierSmall || tier == ports.TierStrong || tier == ports.ComponentBrain ||
		tier == ports.ComponentGenerator || tier == ports.ComponentRouter || tier == ports.ComponentReflection || tier == ports.ComponentMemoryWriter) && !contract.ReserveLLMCall(ctx) {
		return &contract.Error{Code: contract.ErrBudget, Reason: "budget_exceeded", Origin: observability.Source(0)}
	}
	return nil
}

func (c *Client) configured(tier ports.ModelTier, embedding bool) bool {
	return tier.Valid() && (tier == ports.ComponentEmbedding || tier == ports.TierEmbedding) == embedding
}
func validMessages(system []ports.Block, messages []ports.ChatMessage, maxTokens int) bool {
	if maxTokens < 1 || len(system) > 32 || len(messages) > 128 {
		return false
	}
	var size int
	for _, block := range system {
		if block.Name == "" || !utf8.ValidString(block.Text) || len(block.Text) > 65536 {
			return false
		}
		size += len(block.Text)
	}
	for _, msg := range messages {
		if msg.Role != contract.RoleUser && msg.Role != contract.RoleAssistant && msg.Role != contract.RoleSystem || !utf8.ValidString(msg.Content) || len(msg.Content) > 65536 {
			return false
		}
		size += len(msg.Content)
	}
	return size < 512*1024
}
func setTemperature(tier ports.ModelTier, value float32) float32 {
	// In runtime v2 a temperature of 0 is intentional (e.g. Validator).
	if tier != ports.TierSmall && tier != ports.TierStrong && tier != ports.TierClassifier {
		return value
	}
	if value != 0 {
		return value
	}
	switch tier {
	case ports.TierStrong:
		return 0.7 // Generator; Reflection overrides with 0.3
	case ports.TierSmall:
		return 0.2 // Brain; Memory Writer overrides with 0.1
	default:
		return 0.1
	}
}
func validTemperature(value float32) bool {
	return !math.IsNaN(float64(value)) && !math.IsInf(float64(value), 0) && value >= 0 && value <= 2
}

// An attempt records billable usage before any subsequent schema validation or
// fallback. The response aggregates usage across repairs and provider changes.
// Upstreams with unknown usage cannot be accurately billed; those failures are
// metered as events, not as invented token counts.
func (c *Client) charge(ctx context.Context, meta contract.CallMeta, tier ports.ModelTier, model Model, usage contract.LLMUsage, elapsed time.Duration) (contract.LLMUsage, error) {
	usage.Provider, usage.Model = model.Provider, model.Name
	usage.Latency = elapsed
	usage.Attempts = 1
	cost, err := CostMicroUSD(model, usage.InTokens, usage.OutTokens, usage.CachedTokens)
	if err != nil {
		return contract.LLMUsage{}, err
	}
	usage.CostMicroUSD = cost
	if err := c.ledger.Record(ctx, meta.UserID, usage); err != nil {
		return contract.LLMUsage{}, err
	}
	if c.metrics != nil {
		c.metrics.Observe(Event{Tier: tier, Provider: model.Provider, Model: model.Name, Elapsed: elapsed, Charged: true, InTokens: usage.InTokens, OutTokens: usage.OutTokens, CachedTokens: usage.CachedTokens, CostMicroUSD: cost})
	}
	return usage, nil
}
func accumulate(dst *contract.LLMUsage, src contract.LLMUsage) error {
	if src.InTokens > math.MaxInt-dst.InTokens || src.OutTokens > math.MaxInt-dst.OutTokens ||
		src.CachedTokens > math.MaxInt-dst.CachedTokens || src.CostMicroUSD > math.MaxInt64-dst.CostMicroUSD {
		return ErrUsage
	}
	dst.InTokens += src.InTokens
	dst.OutTokens += src.OutTokens
	dst.CachedTokens += src.CachedTokens
	dst.CostMicroUSD += src.CostMicroUSD
	dst.Latency += src.Latency
	dst.Provider, dst.Model = src.Provider, src.Model
	dst.FinishReason = src.FinishReason
	return nil
}
func (c *Client) observe(ctx context.Context, tier ports.ModelTier, model Model, attempt int, elapsed time.Duration, failure FailureKind, token bool, fallback bool, details ...error) {
	if trace.Enabled(ctx) {
		trace.Capture(ctx, "attempt_result", "attempt", "llm.Client", map[string]any{"tier": tier, "model": model, "attempt": attempt, "elapsed_ns": elapsed, "failure": failure, "first_token": token, "fallback": fallback})
	}
	if c.metrics != nil {
		c.metrics.Observe(Event{Tier: tier, Provider: model.Provider, Model: model.Name, Attempt: attempt, Elapsed: elapsed,
			Failure: failure, FirstToken: token, Fallback: fallback})
	}
	if elapsed < 0 {
		elapsed = 0
	}
	args := append([]any{"component", "llm", "operation", "provider_request", "attempt", attempt,
		"tier", string(tier), "duration_ms", elapsed.Milliseconds(), "fallback_used", fallback, "first_token", token}, observability.CorrelationFields(ctx)...)
	if provider := observability.SafeLabel(model.Provider); provider != "" {
		args = append(args, "provider", provider)
	}
	if name := observability.SafeLabel(model.Name); name != "" {
		args = append(args, "model", name)
	}
	if failure != "" {
		err := error(&Failure{Kind: failure})
		if len(details) > 0 && details[0] != nil {
			err = details[0]
		}
		args = append(args, observability.DiagnosticFields(err, observability.Source(0))...)
		slog.Log(ctx, slog.LevelWarn, "llm.attempt.failed", args...)
	} else {
		slog.Log(ctx, slog.LevelDebug, "llm.attempt.completed", args...)
	}
}
func classify(err error) (*Failure, bool) {
	var failure *Failure
	if errors.As(err, &failure) {
		return failure, failure.Retryable()
	}
	kind := FailureUnknown
	reason := observability.ErrorReason(err)
	if reason == "unknown" {
		reason = "unclassified_error"
	}
	switch {
	case errors.Is(err, context.Canceled):
		kind, reason = FailureCanceled, "request_canceled"
	case errors.Is(err, context.DeadlineExceeded):
		kind, reason = FailureTimeout, "request_timeout"
	}
	timeout := ""
	if kind == FailureTimeout {
		timeout = "request"
	}
	return &Failure{Kind: kind, Reason: reason, Timeout: timeout, Cause: err, Origin: observability.Source(0)}, false
}
func (c *Client) delay(ctx context.Context, attempt int, failure *Failure) error {
	max := c.config.RetryBase
	for i := 0; i < attempt && max < c.config.RetryMax; i++ {
		if max > c.config.RetryMax/2 {
			max = c.config.RetryMax
		} else {
			max *= 2
		}
	}
	if max > c.config.RetryMax {
		max = c.config.RetryMax
	}
	delay := c.jitter(max)
	if failure != nil && failure.RetryAfter > delay {
		delay = failure.RetryAfter
	}
	if trace.Enabled(ctx) {
		trace.Capture(ctx, "retry_delay", "retry", "llm.Client.delay", map[string]any{"retry_index": attempt, "delay_ns": delay})
	}
	if delay <= 0 {
		return ctx.Err()
	}
	// Never sleep past the caller's total deadline (including Retry-After).
	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < delay {
		return context.DeadlineExceeded
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return ctx.Err()
	}
}
func (c *Client) checkCost(ctx context.Context, cost int64) error {
	if budget, ok := contract.BudgetFromContext(ctx); ok && budget.MaxCostMicroUSD > 0 && cost >= budget.MaxCostMicroUSD {
		return &contract.Error{Code: contract.ErrBudget, Reason: "budget_exceeded", Origin: observability.Source(0)} // safe fixed message
	}
	return nil
}

func (c *Client) Structured(ctx context.Context, req ports.StructuredRequest) (out ports.StructuredResponse, err error) {
	if contract.ValidID(req.Meta.TurnID) {
		ctx = observability.WithTurnRef(ctx, observability.TurnRef(req.Meta.TurnID))
	}
	callStarted := time.Now()
	defer func() { callOutcome(ctx, "structured", req.Tier, callStarted, err) }()

	ctx, inspectionSpan := trace.Start(ctx, "llm/client.Structured", func() any { return map[string]any{"req": req} })
	defer func() { inspectionSpan.Complete(func() any { return map[string]any{"out": out} }, err, recover()) }()

	if !c.configured(req.Tier, false) || !validMessages(req.System, req.Messages, req.MaxTokens) || !validTemperature(req.Temperature) || strings.TrimSpace(req.SchemaName) == "" || req.MaxAttempts < 0 {
		return out, ErrConfig
	}
	validate, err := schema.CompileInline(req.Schema)
	if err != nil {
		return out, ErrConfig
	}
	callCtx, cancel, err := c.begin(ctx, req.Meta, req.Timeout)
	if err != nil {
		return out, err
	}
	defer cancel()
	req.Temperature = setTemperature(req.Tier, req.Temperature)
	var usage contract.LLMUsage
	var attempts int
	var lastFailure error
	// WP-10 must account for failed schema repairs/provider attempts before
	// deciding whether another reasoning hop fits the turn's call budget.
	defer func() {
		if err != nil {
			out.Usage = usage
			out.Usage.Attempts = attempts
			out.Attempts = attempts
		}
	}()
	for rung, model := range c.config.Chains[req.Tier] {
		key := model.Provider + "\x00" + model.Name
		for retry := 0; retry <= c.config.MaxRetriesPerModel; retry++ {
			if req.MaxAttempts > 0 && attempts >= req.MaxAttempts {
				return out, exhausted(callCtx, attempts, lastFailure)
			}
			if err := callCtx.Err(); err != nil {
				return out, err
			}
			if err := c.preflight(callCtx, req.Meta.UserID, attempts, req.Tier); err != nil {
				return out, err
			}
			allowed, probe := c.breakers.allow(key, c.now())
			if !allowed {
				trace.Skip(callCtx, "provider.Structured", "circuit_breaker_open")
				c.decision(callCtx, req.Tier, model, attempts, "circuit_open", false, rung+1 < len(c.config.Chains[req.Tier]))
				break
			}
			attempts++
			if rung > 0 && retry == 0 {
				c.decision(callCtx, req.Tier, model, attempts, "fallback_started", false, true)
			}
			if retry > 0 {
				slog.DebugContext(callCtx, "llm.retry.started", append([]any{"component", "llm", "attempt", attempts,
					"retry_index", retry}, observability.CorrelationFields(callCtx)...)...)
			}
			mode := ModeJSONObject
			if model.NativeJSONSchema {
				mode = ModeNativeSchema
			}
			if req.SystemOne != nil {
				mode = ModeSystemOne
			}
			start := c.now()
			response, callErr := c.providers[model.Provider].Structured(callCtx, model, req, mode)
			elapsed := c.now().Sub(start)
			if elapsed < 0 {
				elapsed = 0
			}
			if callErr != nil {
				failure, retryable := classify(callErr)
				lastFailure = failure
				c.breakers.record(key, c.now(), probe, retryable)
				c.observe(callCtx, req.Tier, model, attempts, elapsed, failure.Kind, false, rung > 0, failure)
				if !retryable || retry == c.config.MaxRetriesPerModel {
					c.decision(callCtx, req.Tier, model, attempts, "model_exhausted", false, rung+1 < len(c.config.Chains[req.Tier]))
					break
				}
				if err := c.delay(callCtx, retry, failure); err != nil {
					return out, err
				}
				continue
			}
			c.breakers.record(key, c.now(), probe, false)
			charged, err := c.charge(callCtx, req.Meta, req.Tier, model, response.Usage, elapsed)
			if err != nil {
				return out, err
			}
			if err := accumulate(&usage, charged); err != nil {
				return out, err
			}
			if err := callCtx.Err(); err != nil {
				return out, err
			}
			c.observe(callCtx, req.Tier, model, attempts, elapsed, "", false, rung > 0)
			if err := c.checkCost(ctx, usage.CostMicroUSD); err != nil {
				return out, err
			}
			if err := validate(response.JSON); err == nil {
				usage.Attempts = attempts
				return ports.StructuredResponse{JSON: append([]byte(nil), response.JSON...), Usage: usage,
					Model: model.Name, Provider: model.Provider, Attempts: attempts}, nil
			}
			c.decision(callCtx, req.Tier, model, attempts, "schema_invalid", mode == ModeJSONObject, false)
			if mode == ModeNativeSchema || mode == ModeSystemOne {
				return out, observability.Detail(ErrInvalid, "schema_invalid", "", "")
			}
			if err := callCtx.Err(); err != nil {
				return out, err
			}
			if len(response.JSON) > 64*1024 {
				return out, observability.Detail(ErrInvalid, "schema_invalid", "", "")
			}
			// Exactly one repair, ONLY for a JSON-object mode schema mismatch.
			// The original request and schema are unchanged; the repair instruction
			// forbids new interpretation or reasoning, and output is revalidated.
			if req.MaxAttempts > 0 && attempts >= req.MaxAttempts {
				return out, observability.Detail(ErrInvalid, "schema_invalid", "", "")
			}
			if err := c.preflight(callCtx, req.Meta.UserID, attempts, req.Tier); err != nil {
				return out, err
			}
			repair := req
			if repair.IdempotencyKey != "" {
				repair.IdempotencyKey += ".repair"
			}
			repair.Messages = append(append([]ports.ChatMessage(nil), req.Messages...),
				ports.ChatMessage{Role: contract.RoleAssistant, Content: string(response.JSON)},
				ports.ChatMessage{Role: contract.RoleUser, Content: "Fix ONLY the preceding JSON to match the supplied JSON schema. Preserve its meaning; do not re-reason, add facts or commentary. Return the corrected JSON object only."})
			attempts++
			start = c.now()
			fixed, fixErr := c.providers[model.Provider].Structured(callCtx, model, repair, ModeJSONObject)
			elapsed = c.now().Sub(start)
			if elapsed < 0 {
				elapsed = 0
			}
			if fixErr != nil {
				failure, _ := classify(fixErr)
				c.observe(callCtx, req.Tier, model, attempts, elapsed, failure.Kind, false, rung > 0, failure)
				return out, errors.Join(ErrInvalid, failure)
			}
			charged, err = c.charge(callCtx, req.Meta, req.Tier, model, fixed.Usage, elapsed)
			if err != nil {
				return out, err
			}
			if err := accumulate(&usage, charged); err != nil {
				return out, err
			}
			if err := callCtx.Err(); err != nil {
				return out, err
			}
			c.observe(callCtx, req.Tier, model, attempts, elapsed, "", false, rung > 0)
			if err := c.checkCost(ctx, usage.CostMicroUSD); err != nil {
				return out, err
			}
			if err := validate(fixed.JSON); err != nil {
				return out, observability.Detail(ErrInvalid, "schema_invalid", "", "")
			}
			usage.Attempts = attempts
			return ports.StructuredResponse{JSON: append([]byte(nil), fixed.JSON...), Usage: usage,
				Model: model.Name, Provider: model.Provider, Attempts: attempts}, nil
		}
	}
	return out, exhausted(callCtx, attempts, lastFailure)
}

func (c *Client) Embed(ctx context.Context, req ports.EmbeddingRequest) (result0 ports.EmbeddingResponse, result1 error) {
	if contract.ValidID(req.Meta.TurnID) {
		ctx = observability.WithTurnRef(ctx, observability.TurnRef(req.Meta.TurnID))
	}
	callStarted := time.Now()
	defer func() { callOutcome(ctx, "embed", req.Tier, callStarted, result1) }()

	ctx, inspectionSpan := trace.Start(ctx, "llm/client.Embed", func() any { return map[string]any{"req": req} })
	defer func() {
		inspectionSpan.Complete(func() any { return map[string]any{"result0": result0} }, result1, recover())
	}()

	var out ports.EmbeddingResponse
	if req.Tier != ports.ComponentEmbedding && req.Tier != ports.TierEmbedding || !utf8.ValidString(req.Input) || strings.TrimSpace(req.Input) == "" || len(req.Input) > 128*1024 {
		return out, ErrConfig
	}
	callCtx, cancel, err := c.begin(ctx, req.Meta, req.Timeout)
	if err != nil {
		return out, err
	}
	defer cancel()
	var attempts int
	var lastFailure error
	for rung, model := range c.config.Chains[req.Tier] {
		key := model.Provider + "\x00" + model.Name
		for retry := 0; retry <= c.config.MaxRetriesPerModel; retry++ {
			if err := callCtx.Err(); err != nil {
				return out, err
			}
			if err := c.preflight(callCtx, req.Meta.UserID, attempts, req.Tier); err != nil {
				return out, err
			}
			allowed, probe := c.breakers.allow(key, c.now())
			if !allowed {
				c.decision(callCtx, req.Tier, model, attempts, "circuit_open", false, rung+1 < len(c.config.Chains[req.Tier]))
				break
			}
			attempts++
			if rung > 0 && retry == 0 {
				c.decision(callCtx, req.Tier, model, attempts, "fallback_started", false, true)
			}
			if retry > 0 {
				slog.DebugContext(callCtx, "llm.retry.started", append([]any{"component", "llm", "attempt", attempts,
					"retry_index", retry}, observability.CorrelationFields(callCtx)...)...)
			}
			start := c.now()
			result, callErr := c.providers[model.Provider].Embed(callCtx, model, req)
			elapsed := c.now().Sub(start)
			if elapsed < 0 {
				elapsed = 0
			}
			if callErr != nil {
				failure, retryable := classify(callErr)
				lastFailure = failure
				c.breakers.record(key, c.now(), probe, retryable)
				c.observe(callCtx, req.Tier, model, retry+1, elapsed, failure.Kind, false, rung > 0, failure)
				if !retryable || retry == c.config.MaxRetriesPerModel {
					c.decision(callCtx, req.Tier, model, attempts, "model_exhausted", false, rung+1 < len(c.config.Chains[req.Tier]))
					break
				}
				if err := c.delay(callCtx, retry, failure); err != nil {
					return out, err
				}
				continue
			}
			c.breakers.record(key, c.now(), probe, false)
			charged, err := c.charge(callCtx, req.Meta, req.Tier, model, result.Usage, elapsed)
			if err != nil {
				return out, err
			}
			if err := callCtx.Err(); err != nil {
				return out, err
			}
			c.observe(callCtx, req.Tier, model, retry+1, elapsed, "", false, rung > 0)
			if len(result.Vector) != model.Dimensions {
				return out, observability.Detail(ErrInvalid, "schema_invalid", "", "")
			}
			for _, element := range result.Vector {
				if math.IsNaN(float64(element)) || math.IsInf(float64(element), 0) {
					return out, observability.Detail(ErrInvalid, "schema_invalid", "", "")
				}
			}
			if err := c.checkCost(ctx, charged.CostMicroUSD); err != nil {
				return out, err
			}
			charged.Attempts = attempts
			return ports.EmbeddingResponse{Vector: append([]float32(nil), result.Vector...), Model: model.Name, Usage: charged}, nil
		}
	}
	return out, exhausted(callCtx, attempts, lastFailure)
}
