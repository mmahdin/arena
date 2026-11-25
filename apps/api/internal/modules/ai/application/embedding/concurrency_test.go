package embedding

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

func TestHundredConcurrentIdenticalRequestsComputeOnce(t *testing.T) {
	var calls atomic.Int32
	entered := make(chan struct{})
	release := make(chan struct{})
	s := service(t, embeddingFunc(func(ctx context.Context, r ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
		if calls.Add(1) == 1 {
			close(entered)
		}
		select {
		case <-release:
			return good(), nil
		case <-ctx.Done():
			return ports.EmbeddingResponse{}, ctx.Err()
		}
	}), Options{})
	req := request()
	const count = 100
	start := make(chan struct{})
	ready := sync.WaitGroup{}
	done := sync.WaitGroup{}
	var usage atomic.Int32
	for n := 0; n < count; n++ {
		ready.Add(1)
		done.Add(1)
		go func() {
			defer done.Done()
			ready.Done()
			<-start
			v, e := s.Embed(context.Background(), req)
			if e != nil {
				t.Error(e)
				return
			}
			usage.Add(int32(v.Usage.InTokens))
			v.Vector[0] = 123
		}()
	}
	ready.Wait()
	close(start)
	<-entered
	close(release)
	done.Wait()
	if calls.Load() != 1 || usage.Load() != 4 {
		t.Fatal("duplicate execution or accounting", calls.Load(), usage.Load())
	}
	if r := embedOK(t, s, req); r.Vector[0] != 1 {
		t.Fatal("consumer mutated shared vector")
	}
}

func TestCallerTimeoutAndCancellationDoNotOwnExecution(t *testing.T) {
	for _, first := range []string{"retrieval.current", "memory_worker"} {
		for _, cancelled := range []bool{false, true} {
			t.Run(first+map[bool]string{false: "/timeout", true: "/cancel"}[cancelled], func(t *testing.T) {
				entered := make(chan context.Context, 1)
				release := make(chan struct{})
				var calls atomic.Int32
				s := service(t, embeddingFunc(func(ctx context.Context, r ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
					calls.Add(1)
					entered <- ctx
					select {
					case <-release:
						return good(), nil
					case <-ctx.Done():
						return ports.EmbeddingResponse{}, ctx.Err()
					}
				}), Options{})
				req := request()
				req.Consumer = first
				caller, cancel := context.WithCancel(context.Background())
				defer cancel()
				if !cancelled {
					var stop context.CancelFunc
					caller, stop = context.WithTimeout(caller, 20*time.Millisecond)
					defer stop()
				}
				caller = c.WithBudget(caller, c.Budget{TotalDeadline: time.Now().Add(20 * time.Millisecond), MaxLLMCalls: 2})
				firstDone := make(chan error, 1)
				go func() { _, e := s.Embed(caller, req); firstDone <- e }()
				execution := <-entered
				other := req
				other.Consumer = "other_consumer"
				other.Timeout = time.Second
				otherDone := make(chan error, 1)
				go func() {
					r, e := s.Embed(context.Background(), other)
					if e == nil && len(r.Vector) != 2 {
						e = ErrInvalid
					}
					otherDone <- e
				}()
				if cancelled {
					cancel()
				}
				if e := <-firstDone; !errors.Is(e, context.Canceled) && !errors.Is(e, context.DeadlineExceeded) {
					t.Fatal("caller could not stop", e)
				}
				if execution.Err() != nil {
					t.Fatal("first caller cancelled shared execution")
				}
				if b, _ := c.BudgetFromContext(execution); time.Until(b.TotalDeadline) < 100*time.Millisecond {
					t.Fatal("stale Turn budget inherited")
				}
				close(release)
				if e := <-otherDone; e != nil {
					t.Fatal("other consumer lost successful result", e)
				}
				embedOK(t, s, req)
				if calls.Load() != 1 {
					t.Fatal("redundant execution", calls.Load())
				}
			})
		}
	}
}

func TestAllWaitersMayLeaveAndComputationStillPopulatesCache(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	s := service(t, embeddingFunc(func(ctx context.Context, _ ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
		calls.Add(1)
		close(entered)
		select {
		case <-release:
			return good(), nil
		case <-ctx.Done():
			return ports.EmbeddingResponse{}, ctx.Err()
		}
	}), Options{})
	req := request()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, e := s.Embed(ctx, req); done <- e }()
	<-entered
	cancel()
	if e := <-done; !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	close(release)
	waitFor(t, func() bool { return idle(s) })
	embedOK(t, s, req)
	if calls.Load() != 1 {
		t.Fatal("abandoned computation discarded")
	}
}

func TestCapacityExecutionDeadlineAndShutdownAreBounded(t *testing.T) {
	entered := make(chan struct{}, 4)
	var calls atomic.Int32
	p := profile()
	p.ExecutionTimeout = 60 * time.Millisecond
	s := service(t, embeddingFunc(func(ctx context.Context, _ ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
		calls.Add(1)
		entered <- struct{}{}
		<-ctx.Done()
		return ports.EmbeddingResponse{}, ctx.Err()
	}), Options{Profile: p, Config: Config{MaxInFlight: 1}})
	req := request()
	done := make(chan error, 1)
	go func() { _, e := s.Embed(context.Background(), req); done <- e }()
	<-entered
	other := req
	other.Input = "different"
	if _, e := s.Embed(context.Background(), other); !errors.Is(e, ErrCapacity) {
		t.Fatal("capacity not enforced", e)
	}
	if e := <-done; !errors.Is(e, context.DeadlineExceeded) {
		t.Fatal("execution not bounded", e)
	}
	go func() { _, e := s.Embed(context.Background(), req); done <- e }()
	<-entered
	s.Close()
	if e := <-done; !errors.Is(e, ErrClosed) {
		t.Fatal("shutdown did not cancel computation", e)
	}
	if _, e := s.Embed(context.Background(), req); !errors.Is(e, ErrClosed) {
		t.Fatal("stopped service accepted work", e)
	}
	if !idle(s) || calls.Load() != 2 {
		t.Fatal("resources not released")
	}
}

func TestInvalidationFencesActivePublicationAndIsUserScoped(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	req := request()
	s := service(t, embeddingFunc(func(ctx context.Context, r ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
		calls.Add(1)
		if r.Meta.UserID == req.Meta.UserID && calls.Load() == 2 {
			close(entered)
			<-release
		}
		return good(), nil
	}), Options{})
	other := req
	other.Meta.UserID = c.NewID()
	embedOK(t, s, other)
	done := make(chan error, 1)
	go func() { _, e := s.Embed(context.Background(), req); done <- e }()
	<-entered
	s.InvalidateEmbeddingUser(req.Meta.UserID)
	close(release)
	if e := <-done; !errors.Is(e, ErrInvalidated) {
		t.Fatal("invalidated data delivered", e)
	}
	embedOK(t, s, other)
	if calls.Load() != 2 {
		t.Fatal("different user was invalidated")
	}
	embedOK(t, s, req)
	if calls.Load() != 3 {
		t.Fatal("invalidated computation repopulated cache")
	}
	s.InvalidateEmbeddingUser(req.Meta.UserID)
	embedOK(t, s, req)
	if calls.Load() != 4 {
		t.Fatal("published cache not invalidated")
	}
}

func TestDifferentUsersNeverJoinActiveComputation(t *testing.T) {
	entered := make(chan c.ID, 2)
	release := make(chan struct{})
	var calls atomic.Int32
	s := service(t, embeddingFunc(func(ctx context.Context, r ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
		calls.Add(1)
		entered <- r.Meta.UserID
		select {
		case <-release:
			return good(), nil
		case <-ctx.Done():
			return ports.EmbeddingResponse{}, ctx.Err()
		}
	}), Options{})
	one := request()
	two := one
	two.Meta.UserID = c.NewID()
	done := make(chan error, 2)
	for _, req := range []ports.EmbeddingRequest{one, two} {
		go func(req ports.EmbeddingRequest) { _, e := s.Embed(context.Background(), req); done <- e }(req)
	}
	for n := 0; n < 2; n++ {
		select {
		case <-entered:
		case <-time.After(time.Second):
			t.Fatal("different users incorrectly joined one flight")
		}
	}
	close(release)
	for n := 0; n < 2; n++ {
		if e := <-done; e != nil {
			t.Fatal(e)
		}
	}
	if calls.Load() != 2 {
		t.Fatal("cross-user computation reuse", calls.Load())
	}
}

func TestApplicationLifecycleCancelsDetachedExecutions(t *testing.T) {
	life, cancel := context.WithCancel(context.Background())
	entered := make(chan struct{})
	s, e := New(life, embeddingFunc(func(ctx context.Context, _ ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
		close(entered)
		<-ctx.Done()
		return ports.EmbeddingResponse{}, ctx.Err()
	}), Options{Profile: profile()})
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	done := make(chan error, 1)
	go func() { _, e := s.Embed(context.Background(), request()); done <- e }()
	<-entered
	cancel()
	select {
	case e := <-done:
		if !errors.Is(e, ErrClosed) {
			t.Fatal(e)
		}
	case <-time.After(time.Second):
		t.Fatal("application shutdown did not cancel independent execution")
	}
}

func TestRestartHasEmptyCacheAndCanRecompute(t *testing.T) {
	var calls atomic.Int32
	provider := embeddingFunc(func(context.Context, ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
		calls.Add(1)
		return good(), nil
	})
	req := request()
	first := service(t, provider, Options{})
	embedOK(t, first, req)
	first.Close()
	restarted := service(t, provider, Options{})
	embedOK(t, restarted, req)
	if calls.Load() != 2 {
		t.Fatal("new process-local service did not recover independently", calls.Load())
	}
}
