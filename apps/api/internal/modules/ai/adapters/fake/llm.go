// Package fake supplies an independent scripted WP-04 LLM provider, not the
// synthetic one-shot chat responder under adapters/mock.
package fake

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/llm"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type Call struct {
	Operation string
	Model     string
	Mode      llm.Mode
}

type StructuredReply struct {
	JSON  []byte
	Usage contract.LLMUsage
	Err   error
	Delay time.Duration
}

type EmbeddingReply struct {
	Vector []float32
	Usage  contract.LLMUsage
	Err    error
}

type StreamReply struct {
	Chunks []ports.Chunk
	// FailAt is the zero-based Recv index at which Err is returned. Set -1
	// to return all chunks and then io.EOF. Delay applies to each Recv.
	FailAt int
	Err    error
	Delay  time.Duration
}

// Provider queues responses by model ID. Only sanitized Call records are kept;
// no user messages, schemas, vectors or model text are retained in call logs.
type Provider struct {
	mu         sync.Mutex
	Calls      []Call
	structured map[string][]StructuredReply
	streaming  map[string][]StreamReply
	embedding  map[string][]EmbeddingReply
}

func New() *Provider {
	return &Provider{structured: make(map[string][]StructuredReply), streaming: make(map[string][]StreamReply), embedding: make(map[string][]EmbeddingReply)}
}
func (p *Provider) QueueStructured(model string, replies ...StructuredReply) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.structured[model] = append(p.structured[model], replies...)
}
func (p *Provider) QueueStream(model string, replies ...StreamReply) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.streaming[model] = append(p.streaming[model], replies...)
}
func (p *Provider) QueueEmbedding(model string, replies ...EmbeddingReply) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.embedding[model] = append(p.embedding[model], replies...)
}
func (p *Provider) History() []Call {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]Call(nil), p.Calls...)
}
func wait(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return ctx.Err()
	}
}
func (p *Provider) Structured(ctx context.Context, model llm.Model, _ ports.StructuredRequest, mode llm.Mode) (llm.StructuredResult, error) {
	p.mu.Lock()
	p.Calls = append(p.Calls, Call{Operation: "structured", Model: model.Name, Mode: mode})
	queue := p.structured[model.Name]
	if len(queue) == 0 {
		p.mu.Unlock()
		return llm.StructuredResult{}, &llm.Failure{Kind: llm.FailurePermanent}
	}
	reply := queue[0]
	p.structured[model.Name] = queue[1:]
	p.mu.Unlock()
	if err := wait(ctx, reply.Delay); err != nil {
		return llm.StructuredResult{}, err
	}
	return llm.StructuredResult{JSON: append([]byte(nil), reply.JSON...), Usage: reply.Usage}, reply.Err
}
func (p *Provider) Stream(ctx context.Context, model llm.Model, _ ports.StreamRequest) (ports.Stream, error) {
	p.mu.Lock()
	p.Calls = append(p.Calls, Call{Operation: "stream", Model: model.Name})
	queue := p.streaming[model.Name]
	if len(queue) == 0 {
		p.mu.Unlock()
		return nil, &llm.Failure{Kind: llm.FailurePermanent}
	}
	reply := queue[0]
	p.streaming[model.Name] = queue[1:]
	p.mu.Unlock()
	return &stream{ctx: ctx, script: reply}, nil
}
func (p *Provider) Embed(ctx context.Context, model llm.Model, _ ports.EmbeddingRequest) (llm.EmbeddingResult, error) {
	p.mu.Lock()
	p.Calls = append(p.Calls, Call{Operation: "embedding", Model: model.Name})
	queue := p.embedding[model.Name]
	if len(queue) == 0 {
		p.mu.Unlock()
		return llm.EmbeddingResult{}, &llm.Failure{Kind: llm.FailurePermanent}
	}
	reply := queue[0]
	p.embedding[model.Name] = queue[1:]
	p.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return llm.EmbeddingResult{}, err
	}
	return llm.EmbeddingResult{Vector: append([]float32(nil), reply.Vector...), Usage: reply.Usage}, reply.Err
}

type stream struct {
	ctx    context.Context
	script StreamReply
	index  int
	mu     sync.Mutex
	closed bool
}

func (s *stream) Recv() (ports.Chunk, error) {
	if err := wait(s.ctx, s.script.Delay); err != nil {
		return ports.Chunk{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ports.Chunk{}, io.EOF
	}
	if s.script.Err != nil && s.index == s.script.FailAt {
		return ports.Chunk{}, s.script.Err
	}
	if s.index >= len(s.script.Chunks) {
		return ports.Chunk{}, io.EOF
	}
	chunk := s.script.Chunks[s.index]
	s.index++
	return chunk, nil
}
func (s *stream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

var _ llm.Provider = (*Provider)(nil)
