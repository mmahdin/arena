// Package llmhttp is a stand-alone OpenAI Chat Completions/Embeddings-compatible
// WP-04 transport. It does not import the previous one-shot OpenAI gateway.
package llmhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

const maxHTTPBody = 1024 * 1024

// Provider can be used as the transport for any configured OpenAI-compatible
// host. Credentials never reach errors, metrics, logs or returned data.
type Provider struct {
	baseURL string
	key     string
	http    *http.Client
}

// client may be an httptest TLS client. Even an injected client's redirects
// are disabled to prevent sending authorization to a second host.
func New(baseURL, apiKey string, client *http.Client) (*Provider, error) {
	endpoint, err := url.Parse(baseURL)
	if err != nil || endpoint.Scheme != "https" || endpoint.Hostname() == "" || endpoint.User != nil ||
		endpoint.RawQuery != "" || endpoint.Fragment != "" || endpoint.Opaque != "" || strings.TrimSpace(apiKey) == "" || strings.ContainsAny(apiKey, "\r\n") {
		return nil, llm.ErrConfig
	}
	if client == nil {
		client = &http.Client{}
	}
	copyClient := *client
	copyClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	copyClient.Timeout = 0 // context deadline is authoritative for each call
	return &Provider{baseURL: strings.TrimRight(baseURL, "/"), key: apiKey, http: &copyClient}, nil
}

type chatMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type requestJSON struct {
	Model               string    `json:"model"`
	Messages            []chatMsg `json:"messages"`
	ResponseFormat      any       `json:"response_format,omitempty"`
	Stream              bool      `json:"stream,omitempty"`
	StreamOptions       any       `json:"stream_options,omitempty"`
	Temperature         *float32  `json:"temperature,omitempty"`
	MaxTokens           int       `json:"max_tokens,omitempty"`
	MaxCompletionTokens int       `json:"max_completion_tokens,omitempty"`
	ReasoningEffort     string    `json:"reasoning_effort,omitempty"`
	Stop                []string  `json:"stop,omitempty"`
}

func makeRequest(model llm.Model, system []ports.Block, messages []ports.ChatMessage, temperature float32, maxTokens int) requestJSON {
	payload := requestJSON{Model: model.Name}
	for _, block := range system {
		payload.Messages = append(payload.Messages, chatMsg{Role: "system", Content: block.Text})
	}
	for _, message := range messages {
		payload.Messages = append(payload.Messages, chatMsg{Role: string(message.Role), Content: message.Content})
	}
	// Reasoning-first models need completion-token limits and may reject
	// temperature. The orchestration layer still controls the requested tier.
	if strings.HasPrefix(model.Name, "gpt-5") || strings.HasPrefix(model.Name, "gpt-6") ||
		strings.HasPrefix(model.Name, "o1") || strings.HasPrefix(model.Name, "o3") || strings.HasPrefix(model.Name, "o4") {
		payload.MaxCompletionTokens = maxTokens
		if model.Name == "gpt-5.6" || strings.HasPrefix(model.Name, "gpt-5.6-") {
			payload.ReasoningEffort = "none"
		}
	} else {
		payload.MaxTokens = maxTokens
		payload.Temperature = &temperature
	}
	return payload
}

func (p *Provider) send(ctx context.Context, resource string, payload any) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil || len(body) > maxHTTPBody {
		return nil, &llm.Failure{Kind: llm.FailurePermanent}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+resource, bytes.NewReader(body))
	if err != nil {
		return nil, &llm.Failure{Kind: llm.FailurePermanent}
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+p.key)
	response, err := p.http.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, &llm.Failure{Kind: llm.FailureTransport}
	}
	if response.StatusCode != http.StatusOK {
		failure := classifyStatus(response)
		_ = response.Body.Close() // never inspect or expose sensitive vendor bodies
		return nil, failure
	}
	return response, nil
}
func classifyStatus(response *http.Response) error {
	switch {
	case response.StatusCode == 429:
		return &llm.Failure{Kind: llm.FailureRateLimit, RetryAfter: parseRetryAfter(response.Header.Get("Retry-After"))}
	case response.StatusCode >= 500:
		return &llm.Failure{Kind: llm.FailureServer}
	default:
		return &llm.Failure{Kind: llm.FailurePermanent}
	}
}
func parseRetryAfter(value string) time.Duration {
	if n, err := strconv.Atoi(value); err == nil && n >= 0 && n <= 30 {
		return time.Duration(n) * time.Second
	}
	if at, err := http.ParseTime(value); err == nil {
		wait := time.Until(at)
		if wait > 0 && wait <= 30*time.Second {
			return wait
		}
	}
	return 0
}

func readJSON(response *http.Response, output any) error {
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxHTTPBody+1))
	if err != nil || len(body) > maxHTTPBody || json.Unmarshal(body, output) != nil {
		return &llm.Failure{Kind: llm.FailureInvalid}
	}
	return nil
}

type usageWire struct {
	PromptTokens        int `json:"prompt_tokens"`
	CompletionTokens    int `json:"completion_tokens"`
	PromptTokensDetails struct {
		CachedTokens int `json:"cached_tokens"`
	} `json:"prompt_tokens_details"`
}

func (u usageWire) value(reason string) contract.LLMUsage {
	return contract.LLMUsage{InTokens: u.PromptTokens, OutTokens: u.CompletionTokens,
		CachedTokens: u.PromptTokensDetails.CachedTokens, FinishReason: reason}
}

type completion struct {
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage *usageWire `json:"usage"`
}

func schemaName(name string) string {
	name = strings.ReplaceAll(name, ".", "_")
	if len(name) < 1 || len(name) > 64 {
		return ""
	}
	for _, r := range name {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			return ""
		}
	}
	return name
}

func (p *Provider) Structured(ctx context.Context, model llm.Model, req ports.StructuredRequest, mode llm.Mode) (llm.StructuredResult, error) {
	var out llm.StructuredResult
	payload := makeRequest(model, req.System, req.Messages, req.Temperature, req.MaxTokens)
	switch mode {
	case llm.ModeNativeSchema:
		name := schemaName(req.SchemaName)
		if name == "" || !json.Valid(req.Schema) {
			return out, &llm.Failure{Kind: llm.FailurePermanent}
		}
		payload.ResponseFormat = map[string]any{"type": "json_schema", "json_schema": map[string]any{
			"name": name, "strict": true, "schema": json.RawMessage(req.Schema),
		}}
	case llm.ModeJSONObject:
		if !json.Valid(req.Schema) || len(req.Schema) == 0 {
			return out, &llm.Failure{Kind: llm.FailurePermanent}
		}
		payload.ResponseFormat = map[string]any{"type": "json_object"}
		// JSON-object mode does not transmit response_format.json_schema.
		// The provider MUST receive the actual pinned schema in its messages,
		// including for the single schema-repair request. Without this, a
		// schema-conforming response would be left to guesswork.
		payload.Messages = append([]chatMsg{{Role: "system", Content: "Return only a JSON object satisfying this JSON Schema (no markdown or explanation):\n" + string(req.Schema)}}, payload.Messages...)
	default:
		return out, &llm.Failure{Kind: llm.FailurePermanent}
	}
	// The caller owns idempotency. Repeating keys are safe only when the
	// provider explicitly supports them, so we deliberately do not inject
	// nonstandard idempotency HTTP headers into arbitrary compatible hosts.
	response, err := p.send(ctx, "/chat/completions", payload)
	if err != nil {
		return out, err
	}
	var decoded completion
	if err := readJSON(response, &decoded); err != nil || len(decoded.Choices) != 1 || decoded.Choices[0].FinishReason != "stop" ||
		strings.TrimSpace(decoded.Choices[0].Message.Content) == "" || decoded.Usage == nil {
		return out, &llm.Failure{Kind: llm.FailureInvalid}
	}
	return llm.StructuredResult{JSON: []byte(decoded.Choices[0].Message.Content), Usage: decoded.Usage.value("stop")}, nil
}

func (p *Provider) Stream(ctx context.Context, model llm.Model, req ports.StreamRequest) (ports.Stream, error) {
	payload := makeRequest(model, req.System, req.Messages, req.Temperature, req.MaxTokens)
	payload.Stream = true
	payload.StreamOptions = map[string]any{"include_usage": true}
	payload.Stop = req.Stop
	response, err := p.send(ctx, "/chat/completions", payload)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(strings.ToLower(response.Header.Get("Content-Type")), "text/event-stream") {
		_ = response.Body.Close()
		return nil, &llm.Failure{Kind: llm.FailureInvalid}
	}
	return newSSEStream(response.Body), nil
}

func (p *Provider) Embed(ctx context.Context, model llm.Model, req ports.EmbeddingRequest) (llm.EmbeddingResult, error) {
	var out llm.EmbeddingResult
	response, err := p.send(ctx, "/embeddings", map[string]any{"model": model.Name, "input": req.Input, "dimensions": model.Dimensions})
	if err != nil {
		return out, err
	}
	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
		Usage *usageWire `json:"usage"`
	}
	if err := readJSON(response, &result); err != nil || len(result.Data) != 1 || result.Data[0].Index != 0 || result.Usage == nil {
		return out, &llm.Failure{Kind: llm.FailureInvalid}
	}
	return llm.EmbeddingResult{Vector: result.Data[0].Embedding, Usage: result.Usage.value("stop")}, nil
}

var _ llm.Provider = (*Provider)(nil)
