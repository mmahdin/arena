package embed

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	domain "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/domain/retrieval"
)

func TestDeterministicIsStableAndNormalized(t *testing.T) {
	d := NewDeterministic(64)
	ctx := context.Background()
	a, err := d.Embed(ctx, []string{"my manager criticized my work"})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := d.Embed(ctx, []string{"my manager criticized my work"})
	if len(a) != 1 || a[0].Dim != 64 || a[0].ModelID != d.ModelID() {
		t.Fatalf("unexpected vector shape: %+v", a)
	}
	for i := range a[0].Values {
		if a[0].Values[i] != b[0].Values[i] {
			t.Fatal("deterministic embedder must be stable")
		}
	}
	var norm float64
	for _, v := range a[0].Values {
		norm += float64(v) * float64(v)
	}
	if math.Abs(norm-1) > 1e-5 {
		t.Fatalf("vectors must be L2-normalized, got norm %v", norm)
	}
	// Self-similarity is maximal, and identical text scores 1.
	self, err := domain.CosineSimilarity(a[0].Values, a[0].Values)
	if err != nil || math.Abs(float64(self)-1) > 1e-6 {
		t.Fatalf("self similarity: %v err=%v", self, err)
	}
}

func TestDeterministicRejectsEmptyInput(t *testing.T) {
	d := NewDeterministic(32)
	if _, err := d.Embed(context.Background(), nil); !errors.Is(err, ErrEmptyInput) {
		t.Fatal("empty batch must be rejected before any work")
	}
	if _, err := d.Embed(context.Background(), []string{"  "}); !errors.Is(err, ErrEmptyInput) {
		t.Fatal("blank text must be rejected")
	}
}

func TestDeterministicDimensionsAreClamped(t *testing.T) {
	if got := NewDeterministic(1).Dimensions; got != 128 {
		t.Fatalf("out-of-range dimensions must fall back to a safe default, got %d", got)
	}
	if got := NewDeterministic(99999).Dimensions; got != 128 {
		t.Fatalf("huge dimensions must be clamped, got %d", got)
	}
}

func TestOpenAICompatibleRequestAndResponse(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/embeddings" {
			t.Errorf("unexpected route %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("backend auth header missing")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("invalid body: %v", err)
		}
		if body["model"] != "test-embed" || body["encoding_format"] != "float" {
			t.Errorf("unexpected request: %+v", body)
		}
		io.WriteString(w, `{"model":"test-embed","data":[
          {"index":0,"embedding":[0.1,0.2,0.3]},
          {"index":1,"embedding":[0.4,0.5,0.6]}]}`)
	}))
	defer server.Close()

	e := NewOpenAICompatible(server.URL+"/v1", "test-key", "test-embed", 3, 2*time.Second)
	e.client = server.Client()
	e.client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }

	out, err := e.Embed(context.Background(), []string{"first", "second"})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0].Dim != 3 || out[1].Values[2] != float32(0.6) {
		t.Fatalf("unexpected vectors: %+v", out)
	}
	if out[0].ModelID != "test-embed" {
		t.Fatalf("model id must travel with the vector, got %q", out[0].ModelID)
	}
}

func TestOpenAICompatibleRejectsMisorderedResponses(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"data":[{"index":5,"embedding":[0.1,0.2]}]}`)
	}))
	defer server.Close()
	e := NewOpenAICompatible(server.URL, "k", "m", 2, time.Second)
	e.client = server.Client()
	if _, err := e.Embed(context.Background(), []string{"only"}); !errors.Is(err, ErrProvider) {
		t.Fatalf("out-of-range index must be rejected, got %v", err)
	}
}

func TestOpenAICompatibleRejectsBadPayloads(t *testing.T) {
	cases := map[string]string{
		"wrong count":     `{"data":[{"index":0,"embedding":[0.1]}]}`,
		"empty vector":    `{"data":[{"index":0,"embedding":[]}]}`,
		"not json":        `nope`,
		"missing data":    `{}`,
		"numeric strings": `{"data":[{"index":0,"embedding":["x"]}]}`,
	}
	for name, payload := range cases {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				io.WriteString(w, payload)
			}))
			defer server.Close()
			e := NewOpenAICompatible(server.URL, "k", "m", 0, time.Second)
			e.client = server.Client()
			if _, err := e.Embed(context.Background(), []string{"a", "b"}); err == nil {
				t.Fatalf("%s: expected a provider error", name)
			}
		})
	}
}

func TestOpenAICompatibleNeverLeaksProviderDetail(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, `{"error":"secret internal detail"}`)
	}))
	defer server.Close()
	e := NewOpenAICompatible(server.URL, "k", "m", 0, time.Second)
	e.client = server.Client()
	_, err := e.Embed(context.Background(), []string{"text"})
	if !errors.Is(err, ErrProvider) {
		t.Fatalf("want ErrProvider, got %v", err)
	}
	if strings.Contains(err.Error(), "secret") {
		t.Fatal("provider payload leaked into the error")
	}
}

func TestOpenAICompatibleRequiresConfiguration(t *testing.T) {
	e := NewOpenAICompatible("https://example.invalid/v1", "", "m", 0, time.Second)
	if _, err := e.Embed(context.Background(), []string{"x"}); !errors.Is(err, ErrProvider) {
		t.Fatal("a missing key must fail before the request")
	}
	e = NewOpenAICompatible("https://example.invalid/v1", "k", "m", 0, time.Second)
	if _, err := e.Embed(context.Background(), nil); !errors.Is(err, ErrEmptyInput) {
		t.Fatal("empty input must be rejected before the request")
	}
}
