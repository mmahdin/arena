package llm

import (
	"context"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// Stream buffers are INTERNAL. This layer does not push tokens to HTTP and does
// not make clinical validation decisions (WP-07 and WP-11 own those decisions).
func (c *Client) Stream(ctx context.Context, req ports.StreamRequest) (result ports.Stream, resultErr error) {
	if contract.ValidID(req.Meta.TurnID) {
		ctx = observability.WithTurnRef(ctx, observability.TurnRef(req.Meta.TurnID))
	}
	started := time.Now()
	defer func() { callOutcome(ctx, "stream_open", req.Tier, started, resultErr) }()
	if !c.configured(req.Tier, false) || !validMessages(req.System, req.Messages, req.MaxTokens) || !validTemperature(req.Temperature) || len(req.Stop) > 8 {
		return nil, ErrConfig
	}
	for _, stop := range req.Stop {
		if stop == "" || len(stop) > 100 || !utf8.ValidString(stop) {
			return nil, ErrConfig
		}
	}
	callCtx, cancel, err := c.begin(ctx, req.Meta, req.Timeout)
	if err != nil {
		return nil, err
	}
	req.Temperature = setTemperature(req.Tier, req.Temperature)
	if req.FirstToken <= 0 {
		req.FirstToken = req.Timeout
		if req.FirstToken <= 0 {
			req.FirstToken = c.config.DefaultTimeout
		}
	}
	s := &managedStream{client: c, ctx: callCtx, cancel: cancel, req: req}
	if err := s.open(); err != nil {
		s.Close()
		return nil, err
	}
	return s, nil
}

// A per-attempt breaker record is emitted once, even if Close races Recv.
// In particular, closing a half-open probe never leaves the circuit locked.
type streamBreakerRecord struct {
	once   sync.Once
	client *Client
	key    string
	probe  bool
}

func (r *streamBreakerRecord) record(failed bool) {
	if r != nil {
		r.once.Do(func() { r.client.breakers.record(r.key, r.client.now(), r.probe, failed) })
	}
}

type managedStream struct {
	client                *Client
	ctx                   context.Context
	cancel                context.CancelFunc
	req                   ports.StreamRequest
	mu                    sync.Mutex // protects active resource against concurrent Close
	active                ports.Stream
	breaker               *streamBreakerRecord
	stop                  context.CancelFunc
	timer                 *time.Timer
	firstExpired          *atomic.Bool
	closed                atomic.Bool
	rung, retry, attempts int
	model                 Model
	key                   string
	probe                 bool
	emitted               bool
	finished              bool
	done                  bool
	billed                bool
	bytes                 int
	cost                  int64
	usage                 contract.LLMUsage
	lastFailure           error
	started               time.Time
}

func (s *managedStream) next(retryable bool, failure *Failure) error {
	if retryable && s.retry < s.client.config.MaxRetriesPerModel {
		prior := s.retry
		s.retry++
		return s.client.delay(s.ctx, prior, failure)
	}
	s.rung++
	s.retry = 0
	return nil
}
func (s *managedStream) open() error {
	c := s.client
	chain := c.config.Chains[s.req.Tier]
	for s.rung < len(chain) {
		if err := s.ctx.Err(); err != nil {
			return err
		}
		if s.closed.Load() {
			return ErrCancelled
		}
		model := chain[s.rung]
		key := model.Provider + "\x00" + model.Name
		if err := c.preflight(s.ctx, s.req.Meta.UserID, s.attempts, s.req.Tier); err != nil {
			return err
		}
		allowed, probe := c.breakers.allow(key, c.now())
		if !allowed {
			c.decision(s.ctx, s.req.Tier, model, s.attempts, "circuit_open", false, s.rung+1 < len(chain))
			s.rung++
			s.retry = 0
			continue
		}
		s.attempts++
		if s.rung > 0 && s.retry == 0 {
			c.decision(s.ctx, s.req.Tier, model, s.attempts, "fallback_started", false, true)
		}
		if s.retry > 0 {
			slog.DebugContext(s.ctx, "llm.retry.started", append([]any{"component", "llm", "attempt", s.attempts,
				"retry_index", s.retry}, observability.CorrelationFields(s.ctx)...)...)
		}
		attemptCtx, stop := context.WithCancel(s.ctx)
		expired := &atomic.Bool{}
		timer := time.AfterFunc(s.req.FirstToken, func() {
			expired.Store(true)
			stop()
		})
		start := c.now()
		providerStream, err := c.providers[model.Provider].Stream(attemptCtx, model, s.req)
		if err == nil && s.closed.Load() {
			_ = providerStream.Close()
			err = ErrCancelled
		}
		if err != nil {
			timer.Stop()
			stop()
			failure, retryable := classify(err)
			if expired.Load() && s.ctx.Err() == nil {
				failure, retryable = &Failure{Kind: FailureTransport, Reason: "first_token_timeout", Timeout: "first_token", Origin: observability.Source(0)}, true
			}
			s.lastFailure = failure
			c.breakers.record(key, c.now(), probe, retryable)
			c.observe(s.ctx, s.req.Tier, model, s.attempts, c.now().Sub(start), failure.Kind, false, s.rung > 0, failure)
			if err := s.next(retryable, failure); err != nil {
				return err
			}
			continue
		}
		s.mu.Lock()
		s.active = providerStream
		s.breaker = &streamBreakerRecord{client: c, key: key, probe: probe}
		s.firstExpired = expired
		s.stop = stop
		s.timer = timer
		s.mu.Unlock()
		s.model, s.key, s.probe, s.started = model, key, probe, start
		s.emitted, s.billed, s.finished = false, false, false
		return nil
	}
	return exhausted(s.ctx, s.attempts, s.lastFailure)
}
func (s *managedStream) release() {
	s.mu.Lock()
	stream, stop, timer, record := s.active, s.stop, s.timer, s.breaker
	s.active, s.stop, s.timer, s.breaker = nil, nil, nil, nil
	s.mu.Unlock()
	if record != nil {
		record.record(false)
	} // early Close frees a half-open probe
	if timer != nil {
		timer.Stop()
	}
	if stop != nil {
		stop()
	}
	if stream != nil {
		_ = stream.Close()
	}
}

// Recv is single-consumer. Close is safe concurrently and cancels the provider
// HTTP request to unblock a pending Recv; no retry can occur after ANY token was
// returned to the sink. When failure occurs before the first token, a new
// provider/model may be opened within the original turn deadline.
func (s *managedStream) Recv() (ports.Chunk, error) {
	if s.closed.Load() {
		return ports.Chunk{}, ErrCancelled
	}
	if s.done {
		return ports.Chunk{}, io.EOF
	}
	for {
		if err := s.ctx.Err(); err != nil {
			s.release()
			return ports.Chunk{}, err
		}
		s.mu.Lock()
		stream := s.active
		s.mu.Unlock()
		if stream == nil {
			return ports.Chunk{}, ErrUnavailable
		}
		chunk, err := stream.Recv()
		if err != nil {
			failed := err != io.EOF || !s.emitted || !s.finished || !s.billed
			c := s.client
			failure, retryable := classify(err)
			if failed {
				if s.firstExpired != nil && s.firstExpired.Load() && s.ctx.Err() == nil && !s.emitted {
					failure, retryable = &Failure{Kind: FailureTransport, Reason: "first_token_timeout", Timeout: "first_token", Origin: observability.Source(0)}, true
				}
				s.mu.Lock()
				record := s.breaker
				s.mu.Unlock()
				s.lastFailure = failure
				record.record(retryable)
				c.observe(s.ctx, s.req.Tier, s.model, s.attempts, c.now().Sub(s.started), failure.Kind, s.emitted, s.rung > 0, failure)
			} else {
				s.mu.Lock()
				record := s.breaker
				s.mu.Unlock()
				record.record(false)
				c.observe(s.ctx, s.req.Tier, s.model, s.attempts, c.now().Sub(s.started), "", true, s.rung > 0)
			}
			s.release()
			if !failed {
				s.done = true
				s.cancel()
				return ports.Chunk{}, io.EOF
			}
			if s.emitted || s.closed.Load() || s.ctx.Err() != nil {
				if err := s.ctx.Err(); err != nil {
					return ports.Chunk{}, err
				}
				if err == io.EOF {
					return ports.Chunk{}, observability.Detail(ErrInvalid, "invalid_response", "", "")
				}
				return ports.Chunk{}, errors.Join(ErrUnavailable, failure)
			}
			if err := s.next(retryable, failure); err != nil {
				return ports.Chunk{}, err
			}
			if err := s.open(); err != nil {
				return ports.Chunk{}, err
			}
			continue
		}
		if (chunk.FinishReason != "" && chunk.FinishReason != "stop" && chunk.FinishReason != "end_turn") || !utf8.ValidString(chunk.Text) || len(chunk.Text)+s.bytes > 1024*1024 {
			s.release()
			return ports.Chunk{}, observability.Detail(ErrInvalid, "invalid_response", "", "")
		}
		s.bytes += len(chunk.Text)
		if chunk.FinishReason != "" {
			s.finished = true
		}
		if chunk.Usage != nil {
			if s.billed {
				s.release()
				return ports.Chunk{}, localFailure(ErrUsage)
			}
			charged, chargeErr := s.client.charge(s.ctx, s.req.Meta, s.req.Tier, s.model, *chunk.Usage, s.client.now().Sub(s.started))
			if chargeErr != nil {
				s.release()
				return ports.Chunk{}, chargeErr
			}
			if chargeErr = accumulate(&s.usage, charged); chargeErr != nil {
				s.release()
				return ports.Chunk{}, chargeErr
			}
			s.cost = s.usage.CostMicroUSD
			if chargeErr = s.client.checkCost(s.ctx, s.cost); chargeErr != nil {
				s.release()
				return ports.Chunk{}, chargeErr
			}
			total := s.usage
			total.Attempts = s.attempts
			chunk.Usage = &total
			s.billed = true
		}
		if err := s.ctx.Err(); err != nil {
			s.release()
			return ports.Chunk{}, err
		}
		if s.closed.Load() {
			s.release()
			return ports.Chunk{}, ErrCancelled
		}
		if chunk.Text != "" && !s.emitted {
			s.emitted = true
			s.mu.Lock()
			if s.timer != nil {
				s.timer.Stop()
				s.timer = nil
			}
			s.mu.Unlock()
		}
		return chunk, nil
	}
}
func (s *managedStream) Close() error {
	if s.closed.CompareAndSwap(false, true) {
		s.cancel()
		s.release()
	}
	return nil
}
