// Package scheduler owns internal timers. It never contacts a user or a push
// service. Due returns durable, authorized wakeup metadata; executing a
// synthetic AI turn requires a separate, approved admission contract.
package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type Repository interface {
	ApplyTimers(context.Context, c.ID, []c.FollowUpOp, time.Time) error
	DueTimers(context.Context, time.Time, int) ([]c.FollowUp, error)
	Sweep(context.Context, c.BundleRef, time.Time, int) error
}

// RunOptions are infrastructure limits, not a clinical or notification policy.
// No user identifiers, SQL errors, or message content enter health telemetry.
type RunOptions struct {
	Interval               time.Duration
	RetryBase              time.Duration
	RetryMax               time.Duration
	MaxConsecutiveFailures int
}

type Health struct {
	Cycles              uint64
	Failures            uint64
	EventsDispatched    uint64 // internal outbox only; NOT notifications delivered
	ConsecutiveFailures int
	LastFailureStage    string
	LastFailureCode     c.Code
	LastAttemptAt       time.Time
	LastSuccessAt       time.Time
	LastFailureAt       time.Time
}

var ErrUnhealthy = errors.New("scheduler: persistent internal failure")

type Service struct {
	repo   Repository
	bundle c.BundleRef
	now    func() time.Time
	opts   RunOptions
	mu     sync.Mutex
	health Health
}

var _ ports.Scheduler = (*Service)(nil)

func New(r Repository, b c.BundleRef) (*Service, error) {
	return NewWithOptions(r, b, RunOptions{})
}

func NewWithOptions(r Repository, b c.BundleRef, opt RunOptions) (*Service, error) {
	if r == nil || b.Version == "" {
		return nil, mem.ErrAssets
	}
	if opt.Interval == 0 {
		opt.Interval = time.Minute
	}
	if opt.RetryBase == 0 {
		opt.RetryBase = 2 * time.Second
	}
	if opt.RetryMax == 0 {
		opt.RetryMax = time.Minute
	}
	if opt.MaxConsecutiveFailures == 0 {
		opt.MaxConsecutiveFailures = 6
	}
	if opt.Interval < time.Millisecond || opt.RetryBase < time.Millisecond || opt.RetryMax < opt.RetryBase || opt.MaxConsecutiveFailures < 1 || opt.MaxConsecutiveFailures > 100 {
		return nil, mem.ErrAssets
	}
	return &Service{repo: r, bundle: b, now: time.Now, opts: opt}, nil
}

func (s *Service) Apply(ctx context.Context, ops []c.FollowUpOp) error {
	m, ok := mem.Metadata(ctx)
	if !ok || !c.ValidID(m.UserID) || len(ops) > 10 {
		return mem.ErrDenied
	}
	return s.repo.ApplyTimers(ctx, m.UserID, ops, s.now().UTC())
}
func (s *Service) Due(ctx context.Context, now time.Time, limit int) ([]c.FollowUp, error) {
	if now.IsZero() || limit < 1 || limit > 100 {
		return nil, mem.ErrDenied
	}
	return s.repo.DueTimers(ctx, now.UTC(), limit)
}

// Health returns bounded, safe-to-export counters. It contains no raw error,
// identifiers, payloads, or prompt text. Hosts should monitor it alongside
// the error returned by Run; neither implies user-facing delivery.
func (s *Service) Health() Health {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.health
}

func failureCode(err error) c.Code {
	var ce *c.Error
	if errors.As(err, &ce) && ce != nil && ce.Code.Valid() {
		return ce.Code
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return c.ErrBudget
	}
	if errors.Is(err, mem.ErrDenied) {
		return c.ErrForbidden
	}
	if errors.Is(err, mem.ErrConflict) {
		return c.ErrStateConflict
	}
	return c.ErrInternal
}

func (s *Service) failed(at time.Time, stage string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.health.Cycles++
	s.health.Failures++
	s.health.ConsecutiveFailures++
	s.health.LastAttemptAt = at
	s.health.LastFailureAt = at
	s.health.LastFailureStage = stage // fixed internal enum: sweep | due
	s.health.LastFailureCode = failureCode(err)
}
func (s *Service) succeeded(at time.Time, dispatched int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.health.Cycles++
	s.health.ConsecutiveFailures = 0
	s.health.LastAttemptAt = at
	s.health.LastSuccessAt = at
	s.health.EventsDispatched += uint64(dispatched)
}

// Pulse performs a bounded, recoverable maintenance cycle and dispatches only
// authorized internal wakeup.v1 outbox metadata. It does NOT admit an AI turn,
// manufacture a user message, or deliver a notification. The repository owns
// atomic per-user authorization, idempotency and durable outbox writes.
func (s *Service) Pulse(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	at := s.now().UTC()
	if err := s.repo.Sweep(ctx, s.bundle, at, 100); err != nil {
		if ctx.Err() == nil {
			s.failed(at, "sweep", err)
		}
		return 0, err
	}
	due, err := s.repo.DueTimers(ctx, at, 100)
	if err != nil {
		if ctx.Err() == nil {
			s.failed(at, "due", err)
		}
		return 0, err
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	s.succeeded(at, len(due))
	return len(due), nil
}

func (s *Service) retryDelay(failures int) time.Duration {
	d := s.opts.RetryBase
	for i := 1; i < failures && d < s.opts.RetryMax; i++ {
		if d >= s.opts.RetryMax/2 {
			return s.opts.RetryMax
		}
		d *= 2
	}
	return d
}

// Run reports persistent failures to the supervising composition, rather than
// silently retrying forever. Temporary failures back off with bounded retries;
// a successful cycle resets the failure streak. Raw repository error messages
// must never reach health telemetry or the surfaced unhealthy error.
func (s *Service) Run(ctx context.Context) error {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			_, err := s.Pulse(ctx)
			if ctx.Err() != nil {
				return ctx.Err()
			}
			delay := s.opts.Interval
			if err != nil {
				h := s.Health()
				if h.ConsecutiveFailures >= s.opts.MaxConsecutiveFailures {
					return fmt.Errorf("%w: stage=%s code=%s failures=%d", ErrUnhealthy, h.LastFailureStage, h.LastFailureCode, h.ConsecutiveFailures)
				}
				delay = s.retryDelay(h.ConsecutiveFailures)
			}
			timer.Reset(delay)
		}
	}
}
func Quiet(now time.Time, q *c.QuietHours) bool {
	if q == nil {
		return false
	}
	parse := func(v string) (int, bool) { t, e := time.Parse("15:04", v); return t.Hour()*60 + t.Minute(), e == nil }
	start, ok := parse(q.Start)
	end, ok2 := parse(q.End)
	if !ok || !ok2 || start == end {
		return true
	}
	n := now.Hour()*60 + now.Minute()
	if start < end {
		return n >= start && n < end
	}
	return n >= start || n < end
}
func Authorized(ctx context.Context, p mem.Policy, u c.UserState, now time.Time) bool {
	if p == nil || !u.ProactiveOptIn || !u.Risk.Level.Valid() || u.Risk.Level.Rank() >= c.RiskElevated.Rank() || u.Risk.FollowUpUntil != nil && now.Before(*u.Risk.FollowUpUntil) {
		return false
	}
	loc, e := time.LoadLocation(u.TZ)
	if e != nil || Quiet(now.In(loc), u.QuietHours) {
		return false
	}
	ok, e := p.Allow(ctx, mem.Permission{Action: mem.Proactive, User: u, SourceKind: "follow_up", Origin: c.ProvenanceSrc{Origin: "scheduler"}})
	return e == nil && ok
}
func Reason(s string) bool {
	switch s {
	case "check_in", "protocol_review", "goal_review", "risk_follow_up":
		return true
	}
	return false
}
func Validate(op c.FollowUpOp, now time.Time) error {
	if !c.ValidID(op.FollowUpID) || op.Op != "schedule" && op.Op != "cancel" || op.Op == "schedule" && (!Reason(op.Reason) || !op.DueAt.After(now)) {
		return fmt.Errorf("scheduler: invalid timer")
	}
	return nil
}
