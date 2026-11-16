package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application"
)

// Chat Completions-compatible adapter. Network access and secrets remain on the
// backend. No request or response bodies are logged or used in errors.
type Gateway struct {
	baseURL, key, model string
	maxOutputTokens     int
	client              *http.Client
}

var errProvider = errors.New("upstream model request failed")

const maxResponseBytes = 256 * 1024

func New(baseURL, key, model string, timeout time.Duration, maxOutputTokens int) *Gateway {
	return &Gateway{
		baseURL: strings.TrimSuffix(baseURL, "/"), key: key, model: model,
		maxOutputTokens: maxOutputTokens,
		client: &http.Client{
			Timeout: timeout,
			// Prevent accidentally sending API keys to redirected destinations.
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

func (g *Gateway) Generate(ctx context.Context, messages []application.ChatMessage) (string, error) {
	request := struct {
		Model               string                    `json:"model"`
		Messages            []application.ChatMessage `json:"messages"`
		Temperature         *float64                  `json:"temperature,omitempty"`
		MaxTokens           int                       `json:"max_tokens,omitempty"`
		MaxCompletionTokens int                       `json:"max_completion_tokens,omitempty"`
		ReasoningEffort     string                    `json:"reasoning_effort,omitempty"`
	}{Model: g.model, Messages: messages}
	if strings.HasPrefix(g.model, "gpt-5") || strings.HasPrefix(g.model, "gpt-6") || strings.HasPrefix(g.model, "o1") || strings.HasPrefix(g.model, "o3") || strings.HasPrefix(g.model, "o4") {
		request.MaxCompletionTokens = g.maxOutputTokens
		// This chat has a small bounded output budget. Disable optional reasoning
		// for GPT-5.6 so it cannot consume that budget before any visible answer.
		if g.model == "gpt-5.6" || strings.HasPrefix(g.model, "gpt-5.6-") {
			request.ReasoningEffort = "none"
		}
	} else {
		temperature := 0.5
		request.Temperature = &temperature
		request.MaxTokens = g.maxOutputTokens
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return "", errProvider
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", errProvider
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.key)
	resp, err := g.client.Do(req)
	if err != nil {
		return "", errProvider
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errProvider
	}
	// An unexpectedly large body can exhaust resources or contain error details.
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil || len(data) > maxResponseBytes {
		return "", errProvider
	}
	var result struct {
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if json.Unmarshal(data, &result) != nil || len(result.Choices) == 0 {
		return "", errProvider
	}
	choice := result.Choices[0]
	// Reject truncated or filtered replies; neither is a complete safe response.
	if choice.FinishReason != "" && choice.FinishReason != "stop" {
		return "", errProvider
	}
	if strings.TrimSpace(choice.Message.Content) == "" {
		return "", errProvider
	}
	return choice.Message.Content, nil
}
