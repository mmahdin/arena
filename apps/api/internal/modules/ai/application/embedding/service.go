package embedding

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
)

type Config struct {
	TTLSeconds  int `json:"ttl_seconds"`
	MaxEntries  int `json:"max_entries"`
	MaxInFlight int `json:"max_in_flight"`
}

func (c Config) Defaults() Config {
	return c
}
func (c Config) Validate() error {
	if c.TTLSeconds < 1 || c.MaxEntries < 1 || c.MaxInFlight < 1 {
		return ErrConfig
	}
	return nil
}

type Options struct {
	Config   Config
	Resolver ports.EmbeddingProfileResolver
	// Profile is only for explicitly injected clients without a resolver (for
	// example scripted clients). Runtime always resolves its actual component.
	Profile ports.EmbeddingProfile
	Now     func() time.Time
}

type Service struct {
	mu       sync.Mutex
	next     ports.EmbeddingClient
	resolver ports.EmbeddingProfileResolver
	profile  ports.EmbeddingProfile
	config   Config
	cache    cache
	flights  map[key]*flight
	life     context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	now      func() time.Time
}

type flight struct {
	done        chan struct{}
	cancel      context.CancelCauseFunc
	invalidated bool // Service.mu; no unbounded per-user generation registry
	info        trace.EmbeddingInfo
	result      ports.EmbeddingResponse
	err         error
}

var _ ports.EmbeddingClient = (*Service)(nil)
var _ ports.EmbeddingInvalidator = (*Service)(nil)

func New(life context.Context, next ports.EmbeddingClient, opt Options) (*Service, error) {
	cfg := opt.Config.Defaults()
	if life == nil || next == nil || cfg.Validate() != nil {
		return nil, ErrConfig
	}
	resolver := opt.Resolver
	if resolver == nil {
		resolver, _ = next.(ports.EmbeddingProfileResolver)
	}
	if resolver == nil && len(opt.Profile.Models) == 0 {
		return nil, ErrConfig
	}
	if opt.Now == nil {
		opt.Now = time.Now
	}
	life, cancel := context.WithCancel(life)
	opt.Profile.Models = append([]ports.EmbeddingModel(nil), opt.Profile.Models...)
	return &Service{next: next, resolver: resolver, profile: opt.Profile, config: cfg, cache: newCache(), flights: map[key]*flight{}, life: life, cancel: cancel, now: opt.Now}, nil
}

func (s *Service) Embed(ctx context.Context, req ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
	if err := ctx.Err(); err != nil {
		return ports.EmbeddingResponse{}, err
	}
	profile := s.profile
	if s.resolver != nil {
		var err error
		profile, err = s.resolver.EmbeddingProfile(req.Tier)
		if err != nil {
			return ports.EmbeddingResponse{}, err
		}
	}
	k, err := makeKey(req, profile)
	if err != nil {
		return ports.EmbeddingResponse{}, err
	}
	// Request.Timeout limits only this consumer's wait. The independent execution
	// uses the effective component deadline resolved above, including all retries.
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}
	started := s.now()
	ctx, waitSpan := trace.Start(ctx, "embedding.Embed", nil)
	consumer := req.Consumer
	s.mu.Lock()
	if s.life.Err() != nil {
		s.mu.Unlock()
		waitSpan.End(nil, ErrClosed)
		return ports.EmbeddingResponse{}, ErrClosed
	}
	if err := ctx.Err(); err != nil {
		s.mu.Unlock()
		waitSpan.End(nil, err)
		return ports.EmbeddingResponse{}, err
	}
	if result, info, ok := s.cache.get(k, s.now()); ok {
		s.mu.Unlock()
		info.Consumer = consumer
		info.CacheOutcome = "hit"
		info.WaitDurationNS = nonnegative(s.now().Sub(started))
		info.Outcome = "success"
		event(ctx, "embedding.cache_hit", info, nil)
		trace.RecordEmbedding(ctx, info)
		waitSpan.End(nil, nil)
		return result, nil
	}
	f := s.flights[k]
	joined := f != nil
	if f == nil {
		if len(s.flights) >= s.config.MaxInFlight {
			s.mu.Unlock()
			waitSpan.End(nil, ErrCapacity)
			return ports.EmbeddingResponse{}, ErrCapacity
		}
		detached := context.WithoutCancel(ctx)
		// WithoutCancel removes context deadlines, but the old absolute Turn budget
		// is also a context value. Rebind ONLY its deadline to this execution.
		if budget, ok := c.BudgetFromContext(detached); ok {
			budget.TotalDeadline = time.Now().Add(profile.ExecutionTimeout)
			detached = c.WithBudget(detached, budget)
		}
		execution, cancelCause := context.WithCancelCause(detached)
		execution, cancelTimeout := context.WithTimeout(execution, profile.ExecutionTimeout)
		stopLife := context.AfterFunc(s.life, func() { cancelCause(ErrClosed) })
		traceID, nodeID, turnID := trace.InspectorIdentity(ctx)
		info := trace.EmbeddingInfo{ComputationID: c.NewID().String(), Provider: k.provider, Model: k.model, Dimensions: k.dimensions, InputHash: fmt.Sprintf("%x", k.input), Consumer: consumer, CacheOutcome: "miss", OriginTraceID: traceID, OriginNodeID: nodeID, OriginTurnID: turnID}
		f = &flight{done: make(chan struct{}), cancel: cancelCause, info: info}
		s.flights[k] = f
		s.wg.Add(1)
		s.mu.Unlock()
		event(ctx, "embedding.cache_miss", info, nil)
		// Register while the caller span still holds the Turn inspection. This span
		// keeps the original checkpoint alive after every caller stops waiting.
		execution, computeSpan := trace.StartAsync(execution, "embedding.compute", func() any { return info })
		trace.RecordEmbedding(execution, info)
		req.Timeout = profile.ExecutionTimeout
		go func() {
			defer s.wg.Done()
			defer stopLife()
			defer cancelTimeout()
			defer cancelCause(nil)
			s.compute(execution, computeSpan, k, profile, req, f)
		}()
	} else {
		s.mu.Unlock()
	}
	s.mu.Lock()
	info := f.info
	s.mu.Unlock()
	info.Consumer = consumer
	if joined {
		info.CacheOutcome = "inflight"
		event(ctx, "embedding.inflight_join", info, nil)
	}
	select {
	case <-ctx.Done():
		info.WaitDurationNS = nonnegative(s.now().Sub(started))
		info.Outcome = "cancelled"
		info.Reason = "caller_cancelled"
		if ctx.Err() == context.DeadlineExceeded {
			info.Outcome = "timeout"
			info.Reason = "caller_deadline"
		}
		event(ctx, "embedding.wait_timeout", info, ctx.Err())
		trace.RecordEmbedding(ctx, info)
		waitSpan.End(nil, ctx.Err())
		return ports.EmbeddingResponse{}, ctx.Err()
	case <-f.done:
		s.mu.Lock()
		result, err := clone(f.result), f.err
		info = f.info
		invalid := f.invalidated
		s.mu.Unlock()
		if invalid {
			result = ports.EmbeddingResponse{}
			err = ErrInvalidated
		}
		if err == nil && ctx.Err() != nil {
			err = ctx.Err()
		}
		if joined {
			result.Usage = c.LLMUsage{}
			info.CacheOutcome = "inflight"
		}
		info.Consumer = consumer
		info.WaitDurationNS = nonnegative(s.now().Sub(started))
		trace.RecordEmbedding(ctx, info)
		waitSpan.End(nil, err)
		return result, err
	}
}

func (s *Service) compute(ctx context.Context, span *trace.Span, k key, profile ports.EmbeddingProfile, req ports.EmbeddingRequest, f *flight) {
	started := s.now()
	result, err := s.invoke(ctx, req)
	compatible := false
	if err == nil {
		compatible, err = validate(result, profile)
	}
	if ctx.Err() != nil {
		err = context.Cause(ctx)
	}
	s.mu.Lock()
	info := f.info
	if ctx.Err() != nil {
		err = context.Cause(ctx)
	}
	info.ComputeDurationNS = nonnegative(s.now().Sub(started))
	info.ProviderAttempts = max(result.Usage.Attempts, trace.EmbeddingAttempts(ctx))
	info.Outcome = "success"
	if f.invalidated {
		err = ErrInvalidated
	}
	if s.life.Err() != nil {
		err = ErrClosed
	}
	if err != nil {
		info.Outcome = "failed"
		info.Reason = reason(err)
	}
	if err == nil && compatible {
		s.cache.put(k, result, info, s.now(), s.config)
	} else if err == nil {
		info.Reason = "fallback_not_cached_under_primary"
	}
	// Keep the in-flight registration until the trace end is checkpointed; a
	// failure cannot be followed by an overlapping second physical execution.
	f.info = info
	s.mu.Unlock()
	if err != nil {
		event(ctx, "embedding.compute_failed", info, err)
	} else if compatible {
		event(ctx, "embedding.cache_store", info, nil)
	}
	trace.RecordEmbedding(ctx, info)
	span.End(func() any {
		return map[string]any{"model": result.Model, "provider_attempts": info.ProviderAttempts, "outcome": info.Outcome}
	}, err)
	s.mu.Lock()
	// Invalidation can race trace serialization after publication. It removes the
	// entry, cancels this flight and fences delivery as well as cache repopulation.
	if f.invalidated {
		err = ErrInvalidated
	}
	if s.life.Err() != nil {
		err = ErrClosed
	}
	if err != nil {
		result = ports.EmbeddingResponse{}
	}
	f.result = clone(result)
	f.err = err
	delete(s.flights, k)
	close(f.done)
	s.mu.Unlock()
}

func validate(r ports.EmbeddingResponse, p ports.EmbeddingProfile) (bool, error) {
	for i, m := range p.Models {
		if r.Model != m.Model || r.Usage.Provider != "" && r.Usage.Provider != m.Provider {
			continue
		}
		if len(r.Vector) != m.Dimensions {
			return false, ErrInvalid
		}
		var sum float64
		for _, v := range r.Vector {
			if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
				return false, ErrInvalid
			}
			sum += float64(v) * float64(v)
		}
		if sum == 0 {
			return false, ErrInvalid
		}
		// A fallback success may satisfy this invocation, but cannot poison the
		// primary model/provider key. An unidentified provider in a multi-provider
		// chain is likewise not safe to cache.
		return i == 0 && (r.Usage.Provider != "" || len(p.Models) == 1), nil
	}
	return false, ErrInvalid
}

func clone(r ports.EmbeddingResponse) ports.EmbeddingResponse {
	r.Vector = append([]float32(nil), r.Vector...)
	return r
}
func nonnegative(d time.Duration) int64 { return max(0, d.Nanoseconds()) }
func reason(err error) string {
	switch err {
	case context.DeadlineExceeded:
		return "execution_deadline"
	case context.Canceled:
		return "execution_cancelled"
	case ErrInvalidated:
		return "user_invalidated"
	case ErrClosed:
		return "shutdown"
	case ErrInvalid:
		return "invalid_vector"
	default:
		return "provider_error"
	}
}
func event(ctx context.Context, operation string, info trace.EmbeddingInfo, err error) {
	ctx, span := trace.Start(ctx, operation, nil)
	trace.RecordEmbedding(ctx, info)
	span.End(nil, err)
}

// Provider panics cannot escape the bounded service-owned execution goroutine.
func (s *Service) invoke(ctx context.Context, req ports.EmbeddingRequest) (result ports.EmbeddingResponse, err error) {
	defer func() {
		if recover() != nil {
			err = ErrInvalid
		}
	}()
	return s.next.Embed(ctx, req)
}
