// Package llmhttp is a stand-alone OpenAI Chat Completions/Embeddings-compatible
// WP-04 transport. It does not import the previous one-shot OpenAI gateway.
package llmhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/telemetry"
)

const maxHTTPBody = 1024 * 1024

// Provider can be used as the transport for any configured OpenAI-compatible
// host. Credentials never reach errors, metrics, logs or returned data.
type Provider struct {
	baseURL   string
	key       string
	http      *http.Client
	systemOne bool
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
	// Explicit deployment capabilities override the legacy model-family defaults.
	if model.ReasoningEffort != "" {
		payload.ReasoningEffort = model.ReasoningEffort
	}
	if model.SupportsTemperature != nil {
		payload.Temperature = nil
		if *model.SupportsTemperature {
			payload.Temperature = &temperature
		}
	}
	return payload
}

// providerFailure records its construction site, never a provider body or URL.
func providerFailure(kind llm.FailureKind) *llm.Failure {
	return &llm.Failure{Kind: kind, Origin: observability.Source(1)}
}

func (p *Provider) send(ctx context.Context, resource string, payload any) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil || len(body) > maxHTTPBody {
		return nil, requestFailure()
	}
	trace.Secret(ctx, p.key)
	// Physical HTTP request span is independent of the local Inspector.
	// It remains active through the response body (including SSE streaming).
	ctx, httpSpan := telemetry.StartHTTP(ctx, resource)
	// Use this physical HTTP context for both wire directions. A separate
	// optional Inspector child must not relocate the response payload event.
	httpCtx := ctx
	// Capture the final provider-bound JSON, not the prompt template or the
	// pre-schema request. Only an explicitly validated local synthetic private
	// OTLP runtime can export these values. No headers/API keys are captured.
	telemetry.CaptureRequest(ctx, body, p.key)
	// Keep the exact-wire local Inspector capture unchanged. A metadata-only
	// OTel session must not create a response-body wrapper or evaluate payloads.
	var wireSpan *trace.Span
	if trace.Enabled(ctx) {
		ctx, wireSpan = trace.Start(ctx, "http.request", func() any { return map[string]any{"method": "POST", "resource": resource, "body": string(body)} })
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+resource, bytes.NewReader(body))
	if err != nil {
		wireSpan.End(nil, requestFailure())
		httpSpan.End(requestFailure())
		return nil, requestFailure()
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+p.key)
	response, err := p.http.Do(request)
	if err != nil {
		failure := providerFailure(llm.FailureTransport)
		failure.Cause = err
		failure.Reason = observability.ErrorReason(err)
		if ctx.Err() != nil {
			failure.Cause = ctx.Err()
			if errors.Is(ctx.Err(), context.Canceled) {
				failure.Kind, failure.Reason = llm.FailureCanceled, "request_canceled"
			} else {
				failure.Kind, failure.Reason, failure.Timeout = llm.FailureTimeout, "request_timeout", "request"
			}
		} else {
			var network net.Error
			if errors.As(err, &network) && network.Timeout() {
				failure.Kind, failure.Reason, failure.Timeout = llm.FailureTimeout, "connection_timeout", "connection"
				// Preserve the adapter's existing timeout identity and retry policy.
				failure.Cause = errors.Join(context.DeadlineExceeded, err)
			}
		}
		wireSpan.End(nil, failure)
		httpSpan.End(failure)
		return nil, failure
	}
	telemetry.HTTPStatus(httpSpan, response.StatusCode)
	var inspectedBody *inspectionBody
	streaming := response.StatusCode == http.StatusOK && resource == "/chat/completions" && isStreamingRequest(payload)
	if wireSpan != nil {
		trace.Capture(ctx, "http_status", "http_metadata", "", response.StatusCode)
		inspectedBody = &inspectionBody{next: response.Body, ctx: ctx, span: wireSpan}
		response.Body = inspectedBody
	}
	if telemetry.ContentEnabled(ctx) {
		response.Body = newContentBody(response.Body, httpCtx, p.key, streaming)
	}
	if response.StatusCode != http.StatusOK {
		failure := classifyStatus(response).(*llm.Failure)
		// Bounded decoding extracts only allowlisted machine codes. The provider's
		// message/param fields and arbitrary values are never put into errors/logs.
		raw, _ := io.ReadAll(io.LimitReader(response.Body, maxHTTPBody+1))
		var envelope struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if len(raw) <= maxHTTPBody && json.Unmarshal(raw, &envelope) == nil {
			failure.ProviderCode = observability.SafeProviderCode(envelope.Error.Code)
			switch failure.ProviderCode {
			case "insufficient_quota":
				failure.Reason = "quota_exhausted"
			case "model_not_found":
				failure.Reason = "model_unavailable"
			case "unsupported_parameter":
				failure.Reason = "unsupported_parameter"
			case "invalid_json_schema":
				failure.Reason = "schema_invalid"
			case "invalid_parameter", "context_length_exceeded":
				failure.Reason = "invalid_request"
			}
		}
		if inspectedBody != nil {
			inspectedBody.err = failure
			trace.Capture(ctx, "rejection", "provider_error", "", failure)
		}
		// The HTTP layer knows the status; the client layer owns attempt/retry.
		fields := append([]any{"component", "llm_http", "operation", "provider_request"}, observability.CorrelationFields(ctx)...)
		fields = append(fields, observability.ErrorFields(failure)...)
		slog.WarnContext(ctx, "llm.http.rejected", fields...)
		_ = response.Body.Close() // ordinary logs receive only allowlisted machine metadata
		httpSpan.End(failure)
		return nil, failure
	}
	response.Body = observeHTTPBody(response.Body, httpSpan, streaming)
	return response, nil
}

func isStreamingRequest(payload any) bool {
	p, ok := payload.(requestJSON)
	return ok && p.Stream
}

func classifyStatus(response *http.Response) error {
	switch {
	case response.StatusCode == 429:
		return &llm.Failure{Kind: llm.FailureRateLimit, RetryAfter: parseRetryAfter(response.Header.Get("Retry-After")), HTTPStatus: response.StatusCode, Origin: observability.Source(0)}
	case response.StatusCode == 401 || response.StatusCode == 403:
		return &llm.Failure{Kind: llm.FailureAuth, Reason: authReason(response.StatusCode), HTTPStatus: response.StatusCode, Origin: observability.Source(0)}
	case response.StatusCode >= 500:
		return &llm.Failure{Kind: llm.FailureServer, HTTPStatus: response.StatusCode, Origin: observability.Source(0)}
	case response.StatusCode == 400 || response.StatusCode == 422:
		return &llm.Failure{Kind: llm.FailurePermanent, Reason: "invalid_request", HTTPStatus: response.StatusCode, Origin: observability.Source(0)}
	case response.StatusCode == 404:
		return &llm.Failure{Kind: llm.FailurePermanent, Reason: "unavailable", HTTPStatus: response.StatusCode, Origin: observability.Source(0)}
	default:
		return &llm.Failure{Kind: llm.FailurePermanent, Reason: "rejected", HTTPStatus: response.StatusCode, Origin: observability.Source(0)}
	}
}
func parseRetryAfter(value string) time.Duration {
	if n, err := strconv.Atoi(value); err == nil && n >= 0 && int64(n) <= int64(^uint64(0)>>1)/int64(time.Second) {
		return time.Duration(n) * time.Second
	}
	if at, err := http.ParseTime(value); err == nil {
		wait := time.Until(at)
		if wait > 0 {
			return wait
		}
	}
	return 0
}

func readJSON(response *http.Response, output any) error {
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxHTTPBody+1))
	if err != nil {
		return &llm.Failure{Kind: llm.FailureInvalid, Reason: "response_read_failed", Cause: err, Origin: observability.Source(0)}
	}
	if len(body) > maxHTTPBody {
		f := providerFailure(llm.FailureInvalid)
		f.Reason = "response_body_too_large"
		return f
	}
	if json.Unmarshal(body, output) != nil {
		f := providerFailure(llm.FailureInvalid)
		f.Reason = "malformed_provider_json"
		return f
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

func (p *Provider) Structured(ctx context.Context, model llm.Model, req ports.StructuredRequest, mode llm.Mode) (result0 llm.StructuredResult, result1 error) {
	ctx, inspectionSpan := trace.Start(ctx, "adapters/llmhttp/provider.Structured", func() any { return map[string]any{"model": model, "req": req, "mode": mode} })
	defer func() {
		inspectionSpan.Complete(func() any { return map[string]any{"result0": result0} }, result1, recover())
	}()

	var out llm.StructuredResult
	if p.systemOne {
		return p.systemOneSelection(ctx, model, req)
	}
	if req.SystemOne != nil {
		return out, requestFailure()
	}
	payload := makeRequest(model, req.System, req.Messages, req.Temperature, req.MaxTokens)
	switch mode {
	case llm.ModeNativeSchema:
		name := schemaName(req.SchemaName)
		if name == "" || !json.Valid(req.Schema) {
			return out, requestFailure()
		}
		strict := model.JSONSchemaStrict == nil || *model.JSONSchemaStrict
		payload.ResponseFormat = map[string]any{"type": "json_schema", "json_schema": map[string]any{
			"name": name, "strict": strict, "schema": json.RawMessage(req.Schema),
		}}
	case llm.ModeJSONObject:
		if !json.Valid(req.Schema) || len(req.Schema) == 0 {
			return out, requestFailure()
		}
		payload.ResponseFormat = map[string]any{"type": "json_object"}
		// JSON-object mode does not transmit response_format.json_schema.
		// The provider MUST receive the actual pinned schema in its messages,
		// including for the single schema-repair request. Without this, a
		// schema-conforming response would be left to guesswork.
		payload.Messages = append([]chatMsg{{Role: "system", Content: "Return only a JSON object satisfying this JSON Schema (no markdown or explanation):\n" + string(req.Schema)}}, payload.Messages...)
	default:
		return out, requestFailure()
	}
	// The caller owns idempotency. Repeating keys are safe only when the
	// provider explicitly supports them, so we deliberately do not inject
	// nonstandard idempotency HTTP headers into arbitrary compatible hosts.
	response, err := p.send(ctx, "/chat/completions", payload)
	if err != nil {
		return out, err
	}
	var decoded completion
	var decodeErr error
	defer func() {
		if trace.Enabled(ctx) && result1 != nil && decodeErr == nil && decoded.Usage != nil {
			usage := decoded.Usage.value("")
			usage.Provider = model.Provider
			usage.Model = model.Name
			var price *int64
			if model.InputMicroUSDPerM != 0 || model.OutputMicroUSDPerM != 0 || model.CachedMicroUSDPerM != 0 {
				if value, e := llm.CostMicroUSD(model, usage.InTokens, usage.OutTokens, usage.CachedTokens); e == nil {
					price = &value
				}
			}
			trace.Usage(ctx, usage, price)
		}
	}()
	if decodeErr = readJSON(response, &decoded); decodeErr != nil {
		// The payload was captured by the opt-in HTTP inspector; expose only
		// the safe structural failure code in operation diagnostics/logs.
		trace.Capture(ctx, "completion_validation", "diagnostic", "provider.Structured", map[string]any{
			"reason": observability.ErrorReason(decodeErr), "phase": "decode_provider_response",
		})
		return out, decodeErr
	}
	if len(decoded.Choices) != 1 || decoded.Choices[0].FinishReason != "stop" ||
		strings.TrimSpace(decoded.Choices[0].Message.Content) == "" || decoded.Usage == nil {
		reason := "invalid_response"
		switch {
		case len(decoded.Choices) != 1:
			reason = "unexpected_choice_count"
		case decoded.Choices[0].FinishReason != "stop":
			switch decoded.Choices[0].FinishReason {
			case "length":
				reason = "finish_reason_length"
			case "content_filter":
				reason = "finish_reason_content_filter"
			default:
				reason = "unexpected_finish_reason"
			}
		case strings.TrimSpace(decoded.Choices[0].Message.Content) == "":
			reason = "empty_completion_content"
		case decoded.Usage == nil:
			reason = "missing_provider_usage"
		}
		if len(decoded.Choices) == 1 && strings.TrimSpace(decoded.Choices[0].Message.Content) != "" {
			trace.Capture(ctx, "rejected_completion_content", "llm_text", "provider.Structured", decoded.Choices[0].Message.Content)
		}
		trace.Capture(ctx, "completion_validation", "diagnostic", "provider.Structured", map[string]any{
			"reason": reason, "choices": len(decoded.Choices), "usage_present": decoded.Usage != nil,
			"finish_reason": func() string {
				if len(decoded.Choices) == 1 {
					return decoded.Choices[0].FinishReason
				}
				return ""
			}(),
		})
		f := providerFailure(llm.FailureInvalid)
		f.Reason = reason
		return out, f
	}
	return llm.StructuredResult{JSON: []byte(decoded.Choices[0].Message.Content), Usage: decoded.Usage.value("stop")}, nil
}

func (p *Provider) Stream(ctx context.Context, model llm.Model, req ports.StreamRequest) (ports.Stream, error) {
	if p.systemOne {
		return nil, requestFailure()
	}
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
		return nil, providerFailure(llm.FailureInvalid)
	}
	stream := ports.Stream(newSSEStream(response.Body))
	if body := capturedContentBody(response.Body); body != nil {
		stream = &contentStream{next: stream, body: body}
	}
	return observeHTTPStream(stream, response.Body), nil
}

func (p *Provider) Embed(ctx context.Context, model llm.Model, req ports.EmbeddingRequest) (result0 llm.EmbeddingResult, result1 error) {
	if p.systemOne {
		return result0, requestFailure()
	}
	ctx, inspectionSpan := trace.Start(ctx, "adapters/llmhttp/provider.Embed", func() any { return map[string]any{"model": model, "req": req} })
	defer func() {
		inspectionSpan.Complete(func() any { return map[string]any{"result0": result0} }, result1, recover())
	}()

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
	if err := readJSON(response, &result); err != nil {
		return out, err
	}
	if len(result.Data) != 1 || result.Data[0].Index != 0 || result.Usage == nil {
		return out, providerFailure(llm.FailureInvalid)
	}
	return llm.EmbeddingResult{Vector: result.Data[0].Embedding, Usage: result.Usage.value("stop")}, nil
}

var _ llm.Provider = (*Provider)(nil)

func authReason(status int) string {
	if status == 403 {
		return "permission_denied"
	}
	return "authentication"
}

func requestFailure() *llm.Failure {
	return &llm.Failure{Kind: llm.FailurePermanent, Reason: "invalid_request", Origin: observability.Source(1)}
}
