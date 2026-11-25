package embedding

import (
	"context"
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type embeddingFunc func(context.Context, ports.EmbeddingRequest) (ports.EmbeddingResponse, error)

func (f embeddingFunc) Embed(ctx context.Context, r ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
	return f(ctx, r)
}
func profile() ports.EmbeddingProfile {
	return ports.EmbeddingProfile{Models: []ports.EmbeddingModel{{Provider: "synthetic", Model: "test", Dimensions: 2}}, ExecutionTimeout: time.Second}
}
func good() ports.EmbeddingResponse {
	return ports.EmbeddingResponse{Model: "test", Vector: []float32{1, 0}, Usage: c.LLMUsage{Provider: "synthetic", Model: "test", InTokens: 4, CostMicroUSD: 2, Attempts: 1}}
}
func request() ports.EmbeddingRequest {
	return ports.EmbeddingRequest{Tier: ports.ComponentEmbedding, Input: "سلام", Meta: c.CallMeta{UserID: c.NewID(), TurnID: c.NewID()}, Consumer: "retrieval.current"}
}
func service(t *testing.T, next ports.EmbeddingClient, opt Options) *Service {
	t.Helper()
	if len(opt.Profile.Models) == 0 {
		opt.Profile = profile()
	}
	s, e := New(context.Background(), next, opt)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(s.Close)
	return s
}
func embedOK(t *testing.T, s *Service, r ports.EmbeddingRequest) ports.EmbeddingResponse {
	t.Helper()
	v, e := s.Embed(context.Background(), r)
	if e != nil {
		t.Fatal(e)
	}
	return v
}
func waitFor(t *testing.T, fn func() bool) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for !fn() {
		select {
		case <-deadline:
			t.Fatal("condition did not complete")
		case <-time.After(time.Millisecond):
		}
	}
}
func idle(s *Service) bool { s.mu.Lock(); defer s.mu.Unlock(); return len(s.flights) == 0 }

func TestCacheIdentityUserIsolationAndDefensiveCopies(t *testing.T) {
	var calls atomic.Int32
	s := service(t, embeddingFunc(func(_ context.Context, r ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
		calls.Add(1)
		return good(), nil
	}), Options{})
	req := request()
	first := embedOK(t, s, req)
	first.Vector[0] = 99
	req.Meta.TurnID = c.NewID()
	req.Meta.Bundle = c.BundleRef{Version: "another-bundle"}
	req.Consumer = "memory_worker"
	cached := embedOK(t, s, req)
	if calls.Load() != 1 || cached.Vector[0] != 1 || cached.Usage.InTokens != 0 || cached.Usage.CostMicroUSD != 0 {
		t.Fatal("reuse mutated or double charged", calls.Load(), cached)
	}
	req.Input = "سلام "
	embedOK(t, s, req)
	req.Meta.UserID = c.NewID()
	embedOK(t, s, req)
	if calls.Load() != 3 {
		t.Fatal("exact text/user identities crossed", calls.Load())
	}
}

type changingResolver struct {
	mu sync.Mutex
	p  ports.EmbeddingProfile
}

func (r *changingResolver) EmbeddingProfile(ports.ModelTier) (ports.EmbeddingProfile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.p, nil
}
func TestProviderModelDimensionAndConfigurationIdentity(t *testing.T) {
	var calls atomic.Int32
	r := &changingResolver{p: profile()}
	s := service(t, embeddingFunc(func(_ context.Context, _ ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
		calls.Add(1)
		r.mu.Lock()
		m := r.p.Models[0]
		r.mu.Unlock()
		v := make([]float32, m.Dimensions)
		v[0] = 1
		return ports.EmbeddingResponse{Model: m.Model, Vector: v, Usage: c.LLMUsage{Provider: m.Provider}}, nil
	}), Options{Resolver: r})
	req := request()
	for _, change := range []func(*ports.EmbeddingProfile){func(*ports.EmbeddingProfile) {}, func(p *ports.EmbeddingProfile) { p.Models[0].Provider = "other" }, func(p *ports.EmbeddingProfile) { p.Models[0].Model = "other" }, func(p *ports.EmbeddingProfile) { p.Models[0].Dimensions = 3 }, func(p *ports.EmbeddingProfile) { p.Revision = "new endpoint settings" }} {
		r.mu.Lock()
		change(&r.p)
		r.mu.Unlock()
		embedOK(t, s, req)
		embedOK(t, s, req)
	}
	if calls.Load() != 5 {
		t.Fatal("incompatible configuration reused", calls.Load())
	}
}

func TestTTLAndLRU(t *testing.T) {
	var calls atomic.Int32
	var clockMu sync.Mutex
	now := time.Now()
	clock := func() time.Time { clockMu.Lock(); defer clockMu.Unlock(); return now }
	s := service(t, embeddingFunc(func(context.Context, ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
		calls.Add(1)
		return good(), nil
	}), Options{Config: Config{TTLSeconds: 1, MaxEntries: 2}, Now: clock})
	req := request()
	run := func(text string) { req.Input = text; embedOK(t, s, req) }
	run("A")
	run("B")
	run("A")
	run("C")
	run("B")
	if calls.Load() != 4 {
		t.Fatal("LRU did not evict least recent entry", calls.Load())
	}
	clockMu.Lock()
	now = now.Add(time.Second)
	clockMu.Unlock()
	run("B")
	if calls.Load() != 5 {
		t.Fatal("expired entry returned")
	}
}

func TestErrorsInvalidVectorsAndPanicsAreNeverCached(t *testing.T) {
	for name, bad := range map[string]embeddingFunc{
		"provider_error": func(context.Context, ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
			return ports.EmbeddingResponse{}, errors.New("synthetic failure")
		},
		"dimension": func(context.Context, ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
			r := good()
			r.Vector = []float32{1}
			return r, nil
		},
		"model": func(context.Context, ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
			r := good()
			r.Model = "incompatible"
			return r, nil
		},
		"provider": func(context.Context, ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
			r := good()
			r.Usage.Provider = "incompatible"
			return r, nil
		},
		"nan": func(context.Context, ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
			r := good()
			r.Vector[0] = float32(math.NaN())
			return r, nil
		},
		"infinity": func(context.Context, ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
			r := good()
			r.Vector[0] = float32(math.Inf(1))
			return r, nil
		},
		"zero": func(context.Context, ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
			r := good()
			r.Vector = []float32{0, 0}
			return r, nil
		},
		"panic": func(context.Context, ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
			panic("synthetic provider panic")
		},
	} {
		t.Run(name, func(t *testing.T) {
			var calls atomic.Int32
			s := service(t, embeddingFunc(func(ctx context.Context, r ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
				if calls.Add(1) == 1 {
					return bad(ctx, r)
				}
				return good(), nil
			}), Options{})
			req := request()
			if _, e := s.Embed(context.Background(), req); e == nil {
				t.Fatal("invalid computation succeeded")
			}
			embedOK(t, s, req)
			embedOK(t, s, req)
			if calls.Load() != 2 {
				t.Fatal("failed result cached")
			}
		})
	}
}

func TestFallbackCannotPoisonPrimaryIdentity(t *testing.T) {
	var calls atomic.Int32
	p := profile()
	p.Models = append(p.Models, ports.EmbeddingModel{Provider: "fallback", Model: "fallback", Dimensions: 3})
	s := service(t, embeddingFunc(func(context.Context, ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
		if calls.Add(1) == 1 {
			return ports.EmbeddingResponse{Model: "fallback", Vector: []float32{1, 0, 0}, Usage: c.LLMUsage{Provider: "fallback"}}, nil
		}
		return good(), nil
	}), Options{Profile: p})
	req := request()
	if r := embedOK(t, s, req); r.Model != "fallback" {
		t.Fatal(r)
	}
	if r := embedOK(t, s, req); r.Model != "test" {
		t.Fatal("fallback cached under primary", r)
	}
	embedOK(t, s, req)
	if calls.Load() != 2 {
		t.Fatal(calls.Load())
	}
}

func TestInvalidRequestsAndConfigurationFailBeforeProvider(t *testing.T) {
	var calls atomic.Int32
	s := service(t, embeddingFunc(func(context.Context, ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
		calls.Add(1)
		return good(), nil
	}), Options{})
	req := request()
	for _, bad := range []ports.EmbeddingRequest{{}, {Tier: ports.ComponentEmbedding, Input: " ", Meta: req.Meta}, {Tier: ports.ComponentUnified, Input: "valid", Meta: req.Meta}, {Tier: ports.ComponentEmbedding, Input: string([]byte{0xff}), Meta: req.Meta}} {
		if _, e := s.Embed(context.Background(), bad); e == nil {
			t.Fatal("invalid input accepted")
		}
	}
	if calls.Load() != 0 {
		t.Fatal("invalid inputs reached provider")
	}
	for _, cfg := range []Config{{TTLSeconds: -1}, {MaxEntries: -1}, {MaxInFlight: -1}, {MaxInFlight: 1025}, {MaxEntries: 65537}, {TTLSeconds: 86401}} {
		if _, e := New(context.Background(), s, Options{Config: cfg, Profile: profile()}); e == nil {
			t.Fatal("unbounded config accepted", cfg)
		}
	}
}
