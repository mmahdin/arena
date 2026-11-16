package application

import (
	"context"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/policy"
	conv "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/conversation/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/conversation/domain"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type ModelGateway interface {
	Generate(ctx context.Context, messages []ChatMessage) (string, error)
}
type Service struct {
	gateway         ModelGateway
	maxContextChars int
}

// ErrProvider is deliberately generic: neither the caller nor HTTP responses
// should expose SDK errors, raw upstream JSON, tokens, or the user's prompt.
var ErrProvider = errors.New("model provider unavailable")

func New(gateway ModelGateway, maxContextChars int) *Service {
	if maxContextChars < 1 {
		maxContextChars = 16000
	}
	return &Service{gateway: gateway, maxContextChars: maxContextChars}
}

// contextTail selects a contiguous, chronological suffix. Limit both the
// number of messages (upstream repository) and the total Unicode characters.
// Never truncate a message and accidentally change what the user said.
func contextTail(history []domain.Message, budget int) []ChatMessage {
	start := len(history)
	for start > 0 {
		m := history[start-1]
		if m.Role != "user" && m.Role != "assistant" {
			break
		}
		length := utf8.RuneCountInString(m.Content)
		if length > budget {
			break
		}
		budget -= length
		start--
	}
	// A clipped history must never start with an orphan assistant answer.
	if start < len(history) && history[start].Role == "assistant" {
		start++
	}
	out := make([]ChatMessage, 0, len(history)-start)
	for _, m := range history[start:] {
		out = append(out, ChatMessage{Role: m.Role, Content: m.Content})
	}
	return out
}

func (s *Service) Reply(ctx context.Context, p conv.Profile, history []domain.Message, userMessage string) (string, string, error) {
	// Treat cancellation as a failed turn even if a mock/third-party gateway
	// ignores its context. Never persist a late model reply.
	if ctx.Err() != nil {
		return "", policy.Version, ErrProvider
	}
	if policy.CrisisSignal(userMessage) {
		return policy.SupportiveFallback, policy.Version, nil
	}
	messages := []ChatMessage{{Role: "system", Content: policy.Build(p)}}
	messages = append(messages, contextTail(history, s.maxContextChars)...)
	messages = append(messages, ChatMessage{Role: "user", Content: userMessage})
	reply, err := s.gateway.Generate(ctx, messages)
	if err != nil || ctx.Err() != nil {
		return "", policy.Version, ErrProvider
	}
	reply = strings.TrimSpace(reply)
	if reply == "" || !utf8.ValidString(reply) || utf8.RuneCountInString(reply) > 4000 || strings.IndexFunc(reply, func(r rune) bool {
		return unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t'
	}) >= 0 {
		return "", policy.Version, ErrProvider
	}
	if policy.UnsafeOutput(reply) {
		return policy.SafetyFailure, policy.Version, nil
	}
	return reply, policy.Version, nil
}
