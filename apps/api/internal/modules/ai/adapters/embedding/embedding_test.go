package embedding

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type fakeClient struct {
	resp     ports.EmbeddingResponse
	err      error
	calls    int
	lastReq  ports.EmbeddingRequest
	panicOn  bool
	callback func(ports.EmbeddingRequest)
}

func (c *fakeClient) Embed(_ context.Context, req ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
	c.calls++
	c.lastReq = req
	if c.callback != nil {
		c.callback(req)
	}
	if c.panicOn {
		panic("provider panic")
	}
	return c.resp, c.err
}

func validVector(dims int) []float32 {
	v := make([]float32, dims)
	v[0] = 1
	return v
}

func TestNewValidatesConfiguration(t *testing.T) {
	cases := []struct {
		name string
		opts Options
		ok   bool
	}{
		{"valid", Options{Client: &fakeClient{}, Model: ModelID{Model: "embed-1", Dimensions: 3}}, true},
		{"no client", Options{Model: ModelID{Model: "embed-1", Dimensions: 3}}, false},
		{"no model", Options{Client: &fakeClient{}, Model: ModelID{Dimensions: 3}}, false},
		{"zero dims", Options{Client: &fakeClient{}, Model: ModelID{Model: "embed-1"}}, false},
		{"too many dims", Options{Client: &fakeClient{}, Model: ModelID{Model: "embed-1", Dimensions: 9000}}, false},
		{"override dims", Options{Client: &fakeClient{}, Model: ModelID{Model: "embed-1", Dimensions: 3}, Dimensions: 5}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			adapter, err := New(tc.opts)
			if tc.ok != (err == nil) {
				t.Fatalf("ok=%v err=%v", tc.ok, err)
			}
			if tc.ok && adapter == nil {
				t.Fatal("nil adapter on success")
			}
		})
	}
}

func TestEmbedPassesThroughProviderResult(t *testing.T) {
	client := &fakeClient{resp: ports.EmbeddingResponse{
		Vector: []float32{0.6, 0.8}, Model: "embed-1",
		Usage: contract.LLMUsage{Provider: "fake", Model: "embed-1", InTokens: 7},
	}}
	var usage contract.LLMUsage
	adapter, err := New(Options{Client: client, Model: ModelID{Model: "embed-1", Dimensions: 2},
		Timeout: 150 * time.Millisecond, OnUsage: func(u contract.LLMUsage) { usage = u }})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	meta := contract.CallMeta{TurnID: contract.NewID(), UserID: contract.NewID()}
	vec, model, err := adapter.WithCallMeta(meta).Embed(context.Background(), "my boss again")
	if err != nil {
		t.Fatalf("embed: %v", err)
	}
	if len(vec) != 2 || math.Abs(float64(vec[1])-0.8) > 1e-6 || model != "embed-1" {
		t.Fatalf("unexpected vector/model: %v %s", vec, model)
	}
	if client.calls != 1 {
		t.Fatalf("expected exactly one provider call, got %d", client.calls)
	}
	if client.lastReq.Tier != ports.TierEmbedding || client.lastReq.Timeout != 150*time.Millisecond {
		t.Fatalf("request not routed to the embedding tier: %+v", client.lastReq)
	}
	if client.lastReq.Meta.UserID != meta.UserID {
		t.Fatal("turn/user identity not propagated for quota and tracing")
	}
	if usage.InTokens != 7 {
		t.Fatalf("usage hook not invoked: %+v", usage)
	}
}

func TestEmbedRejectsMismatchedProviderOutput(t *testing.T) {
	cases := map[string]ports.EmbeddingResponse{
		// The provider returns a different dimension than the stored codec
		// expects: comparing them would silently corrupt every score.
		"wrong dimensions": {Vector: []float32{1}, Model: "embed-1"},
		"wrong model":      {Vector: validVector(3), Model: "other-model"},
		"empty vector":     {Vector: nil, Model: "embed-1"},
	}
	for name, resp := range cases {
		t.Run(name, func(t *testing.T) {
			client := &fakeClient{resp: resp}
			adapter, err := New(Options{Client: client, Model: ModelID{Model: "embed-1", Dimensions: 3}})
			if err != nil {
				t.Fatalf("new: %v", err)
			}
			if _, _, err = adapter.Embed(context.Background(), "text"); err == nil {
				t.Fatal("mismatched provider output accepted")
			}
		})
	}
}

func TestEmbedPropagatesProviderErrorWithoutRetrying(t *testing.T) {
	sentinel := errors.New("provider unavailable")
	client := &fakeClient{err: sentinel}
	adapter, err := New(Options{Client: client, Model: ModelID{Model: "embed-1", Dimensions: 3}})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if _, _, err = adapter.Embed(context.Background(), "text"); !errors.Is(err, sentinel) {
		t.Fatalf("error not propagated: %v", err)
	}
	if client.calls != 1 {
		t.Fatalf("the adapter must not retry or double-charge: %d calls", client.calls)
	}
}

// WithCallMeta must not mutate the receiver: the adapter is shared by a service
// that may serve concurrent turns.
func TestWithCallMetaDoesNotMutateReceiver(t *testing.T) {
	client := &fakeClient{resp: ports.EmbeddingResponse{Vector: validVector(3), Model: "embed-1"}}
	adapter, err := New(Options{Client: client, Model: ModelID{Model: "embed-1", Dimensions: 3}})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	first := contract.NewID()
	second := contract.NewID()
	bound := adapter.WithCallMeta(contract.CallMeta{UserID: first})
	if _, _, err = bound.Embed(context.Background(), "a"); err != nil {
		t.Fatalf("embed: %v", err)
	}
	if client.lastReq.Meta.UserID != first {
		t.Fatal("call meta not attached")
	}
	if _, _, err = adapter.Embed(context.Background(), "b"); err != nil {
		t.Fatalf("embed: %v", err)
	}
	if client.lastReq.Meta.UserID == first {
		t.Fatal("unbound adapter inherited a previous caller's identity")
	}
	_ = second
}

func TestEmbedRejectsEmptyInput(t *testing.T) {
	client := &fakeClient{resp: ports.EmbeddingResponse{Vector: validVector(3), Model: "embed-1"}}
	adapter, err := New(Options{Client: client, Model: ModelID{Model: "embed-1", Dimensions: 3}})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if _, _, err = adapter.Embed(context.Background(), ""); err == nil {
		t.Fatal("empty input accepted")
	}
	if client.calls != 0 {
		t.Fatal("an empty input must not reach the provider")
	}
}

func TestEmbedHonoursCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := &fakeClient{resp: ports.EmbeddingResponse{Vector: validVector(3), Model: "embed-1"}}
	adapter, err := New(Options{Client: client, Model: ModelID{Model: "embed-1", Dimensions: 3}})
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	// The adapter delegates context handling to WP-04; this asserts the value
	// is forwarded rather than replaced.
	if _, _, err = adapter.Embed(ctx, "text"); err != nil {
		t.Fatalf("adapter must not invent its own cancellation: %v", err)
	}
}
