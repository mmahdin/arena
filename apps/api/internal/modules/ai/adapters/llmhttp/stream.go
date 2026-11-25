package llmhttp

import (
	"bufio"
	"encoding/json"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"
	"io"
	"strings"
	"sync"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// SSE is bounded by max line/event/body sizes. A missing [DONE] or a
// truncated/filtered completion fails rather than accepting partial output.
type sseStream struct {
	body   io.ReadCloser
	scan   *bufio.Scanner
	mu     sync.Mutex
	seen   bool
	done   bool
	bytes  int
	events int
}

func newSSEStream(body io.ReadCloser) *sseStream {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 4096), 128*1024)
	return &sseStream{body: body, scan: scanner}
}
func (s *sseStream) Recv() (ports.Chunk, error) {
	if s.done {
		return ports.Chunk{}, io.EOF
	}
	for s.scan.Scan() {
		line := s.scan.Text()
		s.bytes += len(line)
		s.events++
		if s.bytes > 2*maxHTTPBody || s.events > 4096 {
			return ports.Chunk{}, providerFailure(llm.FailureInvalid)
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			if !s.seen {
				return ports.Chunk{}, providerFailure(llm.FailureInvalid)
			}
			s.done = true
			return ports.Chunk{}, io.EOF
		}
		var event struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
			Usage *usageWire `json:"usage"`
		}
		if json.Unmarshal([]byte(data), &event) != nil || len(event.Choices) > 1 {
			return ports.Chunk{}, providerFailure(llm.FailureInvalid)
		}
		var chunk ports.Chunk
		if len(event.Choices) == 1 {
			chunk.Text = event.Choices[0].Delta.Content
			if event.Choices[0].FinishReason != nil {
				chunk.FinishReason = *event.Choices[0].FinishReason
				if chunk.FinishReason != "stop" {
					return ports.Chunk{}, providerFailure(llm.FailureInvalid)
				}
				s.seen = true
			}
		}
		if event.Usage != nil {
			usage := event.Usage.value(chunk.FinishReason)
			chunk.Usage = &usage
		}
		if chunk.Text != "" || chunk.Usage != nil || chunk.FinishReason != "" {
			return chunk, nil
		}
	}
	if s.scan.Err() != nil {
		return ports.Chunk{}, &llm.Failure{Kind: llm.FailureTransport, Cause: s.scan.Err(), Origin: observability.Source(0)}
	}
	return ports.Chunk{}, &llm.Failure{Kind: llm.FailureTransport, Reason: "stream_incomplete", Origin: observability.Source(0)}
}
func (s *sseStream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.body.Close()
}

var _ ports.Stream = (*sseStream)(nil)
