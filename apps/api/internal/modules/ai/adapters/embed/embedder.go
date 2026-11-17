// Package embed implements WP-09's embedding integration.
//
// Contract Pack §6.3 makes the embedding tier part of the WP-04 LLM client
// layer, and §7.3 forbids implying a pgvector deployment: vectors are stored as
// float32 blobs in SQL Server and compared in Go. This package therefore exposes
// a small `Embedder` surface plus two implementations:
//
//   - Deterministic: a hash-based, dependency-free embedder used by tests. It is
//     NOT a semantic model. It exists so ranking, dedupe and budget logic can be
//     exercised end to end without a network call, and it is clearly labelled.
//   - OpenAI-compatible: POST {base}/embeddings, the same provider shape the
//     existing backend already uses for chat completions. No SDK, no logging of
//     inputs, bounded response size.
package embed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"hash/fnv"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// ErrProvider is deliberately generic: no upstream body, credential or prompt is
// ever returned to the caller or written to a log.
var ErrProvider = errors.New("embedding provider unavailable")

// ErrEmptyInput rejects an empty batch instead of issuing a billable request.
var ErrEmptyInput = errors.New("embedding input is empty")

const maxResponseBytes = 8 * 1024 * 1024

// Deterministic is a stable, offline embedder for tests and for local synthetic
// runs where no provider is configured. Identical text always produces identical
// vectors; lexically similar text tends to share dimensions (bag-of-words hash),
// which is enough to test the ranking pipeline, and it is explicitly *not* a
// semantic model.
type Deterministic struct {
	Dimensions int
	Model      string
}

// NewDeterministic builds the offline embedder. Dimensions are clamped to a
// sane range so a typo cannot allocate an enormous vector.
func NewDeterministic(dimensions int) *Deterministic {
	if dimensions < 8 || dimensions > 4096 {
		dimensions = 128
	}
	return &Deterministic{Dimensions: dimensions, Model: "deterministic-test-embed"}
}

func (d *Deterministic) ModelID() string { return d.Model }

func (d *Deterministic) Embed(_ context.Context, texts []string) ([]contract.Vector, error) {
	if len(texts) == 0 {
		return nil, ErrEmptyInput
	}
	out := make([]contract.Vector, 0, len(texts))
	for _, text := range texts {
		if strings.TrimSpace(text) == "" {
			return nil, ErrEmptyInput
		}
		values := make([]float32, d.Dimensions)
		for _, token := range strings.Fields(strings.ToLower(text)) {
			h := fnv.New32a()
			_, _ = h.Write([]byte(token))
			sum := h.Sum32()
			idx := int(sum % uint32(d.Dimensions))
			sign := 1.0
			if sum&0x80000000 != 0 {
				sign = -1.0
			}
			values[idx] += float32(sign)
		}
		// L2-normalize so cosine similarity is a plain dot product.
		var norm float64
		for _, v := range values {
			norm += float64(v) * float64(v)
		}
		if norm == 0 {
			values[0] = 1
			norm = 1
		}
		scale := float32(1 / math.Sqrt(norm))
		for i := range values {
			values[i] *= scale
		}
		out = append(out, contract.Vector{ModelID: d.Model, Dim: d.Dimensions, Values: values})
	}
	return out, nil
}

// OpenAICompatible calls a Chat-Completions-style provider's /embeddings route.
type OpenAICompatible struct {
	baseURL, key, model string
	dimensions          int
	client              *http.Client
}

// NewOpenAICompatible builds the adapter. Redirects are refused so credentials
// cannot be forwarded to an unexpected host; the caller supplies the timeout.
func NewOpenAICompatible(baseURL, key, model string, dimensions int, timeout time.Duration) *OpenAICompatible {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &OpenAICompatible{
		baseURL: strings.TrimSuffix(baseURL, "/"), key: key, model: model, dimensions: dimensions,
		client: &http.Client{
			Timeout:       timeout,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

func (o *OpenAICompatible) ModelID() string { return o.model }

type embeddingRequest struct {
	Model          string   `json:"model"`
	Input          []string `json:"input"`
	Dimensions     int      `json:"dimensions,omitempty"`
	EncodingFormat string   `json:"encoding_format"`
}

type embeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
}

// Embed issues one bounded request per call. The batch order is preserved by the
// response `index` field, not by array order.
func (o *OpenAICompatible) Embed(ctx context.Context, texts []string) ([]contract.Vector, error) {
	if len(texts) == 0 {
		return nil, ErrEmptyInput
	}
	for _, t := range texts {
		if strings.TrimSpace(t) == "" {
			return nil, ErrEmptyInput
		}
	}
	if o.key == "" || o.model == "" {
		return nil, ErrProvider
	}
	body := embeddingRequest{Model: o.model, Input: texts, EncodingFormat: "float"}
	if o.dimensions > 0 {
		body.Dimensions = o.dimensions
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, ErrProvider
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/embeddings", bytes.NewReader(payload))
	if err != nil {
		return nil, ErrProvider
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.key)
	resp, err := o.client.Do(req)
	if err != nil {
		return nil, ErrProvider
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, ErrProvider
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil || len(data) > maxResponseBytes {
		return nil, ErrProvider
	}
	var parsed embeddingResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, ErrProvider
	}
	if len(parsed.Data) != len(texts) {
		return nil, ErrProvider
	}
	model := parsed.Model
	if model == "" {
		model = o.model
	}
	out := make([]contract.Vector, len(texts))
	seen := make([]bool, len(texts))
	for _, item := range parsed.Data {
		if item.Index < 0 || item.Index >= len(texts) || seen[item.Index] {
			return nil, ErrProvider
		}
		if len(item.Embedding) == 0 || len(item.Embedding) > 8192 {
			return nil, ErrProvider
		}
		for _, v := range item.Embedding {
			if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
				return nil, ErrProvider
			}
		}
		seen[item.Index] = true
		out[item.Index] = contract.Vector{ModelID: model, Dim: len(item.Embedding), Values: item.Embedding}
	}
	return out, nil
}
