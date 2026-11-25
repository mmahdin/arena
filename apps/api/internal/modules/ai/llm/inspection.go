package llm

import (
	"context"
	"encoding/json"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/telemetry"
	"io"
	"sync"
	"time"
)

// Instrument the shared provider boundary so new callers inherit capture.
type inspectedProvider struct{ next Provider }

func (p inspectedProvider) Structured(ctx context.Context, m Model, r ports.StructuredRequest, mode Mode) (out StructuredResult, err error) {
	if !trace.Enabled(ctx) && !telemetry.ContentEnabled(ctx) {
		return p.next.Structured(ctx, m, r, mode)
	}
	ctx, span := trace.Start(ctx, "provider.Structured", func() any { return map[string]any{"model": m, "request": r, "mode": mode} })
	trace.BindInputs(ctx)
	started := time.Now()
	defer func() {
		if span != nil && err == nil {
			recordInspectionUsage(ctx, m, out.Usage, time.Since(started))
		}
		span.Complete(func() any { return map[string]any{"json": string(out.JSON), "usage": out.Usage} }, err, recover())
	}()
	if r.SystemOne != nil {
		raw, _ := json.Marshal(r.SystemOne)
		trace.EstimateText(ctx, "systemone/state_and_questions", string(raw))
	} else {
		estimateRequest(ctx, r.System, r.Messages)
	}
	out, err = p.next.Structured(ctx, m, r, mode)
	return
}
func (p inspectedProvider) Embed(ctx context.Context, m Model, r ports.EmbeddingRequest) (out EmbeddingResult, err error) {
	if !trace.Enabled(ctx) && !telemetry.ContentEnabled(ctx) {
		return p.next.Embed(ctx, m, r)
	}
	ctx, span := trace.Start(ctx, "provider.Embed", func() any { return map[string]any{"model": m, "request": r} })
	started := time.Now()
	defer func() {
		if span != nil && err == nil {
			recordInspectionUsage(ctx, m, out.Usage, time.Since(started))
		}
		span.Complete(func() any { return out }, err, recover())
	}()
	trace.EstimateText(ctx, "embedding_input", r.Input)
	out, err = p.next.Embed(ctx, m, r)
	return
}
func (p inspectedProvider) Stream(ctx context.Context, m Model, r ports.StreamRequest) (out ports.Stream, err error) {
	if !trace.Enabled(ctx) && !telemetry.ContentEnabled(ctx) {
		return p.next.Stream(ctx, m, r)
	}
	ctx, span := trace.Start(ctx, "provider.Stream", func() any { return map[string]any{"model": m, "request": r} })
	if span == nil {
		return p.next.Stream(ctx, m, r)
	}
	trace.BindInputs(ctx)
	defer func() {
		if v := recover(); v != nil {
			span.Complete(nil, err, v)
		}
	}()
	estimateRequest(ctx, r.System, r.Messages)
	started := time.Now()
	next, err := p.next.Stream(ctx, m, r)
	if err != nil {
		span.End(nil, err)
		return nil, err
	}
	if next == nil {
		span.End(nil, ErrInvalid)
		return next, nil
	}
	return &inspectionStream{next: next, ctx: ctx, span: span, model: m, started: started}, nil
}
func estimateRequest(ctx context.Context, system []ports.Block, messages []ports.ChatMessage) {
	if !trace.Enabled(ctx) {
		return
	}
	for _, b := range system {
		trace.EstimateText(ctx, "system/"+b.Name, b.Text)
		trace.EstimateSections(ctx, "system/"+b.Name, b.Text)
	}
	for _, m := range messages {
		trace.EstimateText(ctx, "message/"+string(m.Role), m.Content)
		trace.EstimateSections(ctx, "message/"+string(m.Role), m.Content)
	}
}
func recordInspectionUsage(ctx context.Context, m Model, u contract.LLMUsage, elapsed time.Duration) {
	u.Provider = m.Provider
	u.Model = m.Name
	u.Latency = elapsed
	var cost *int64
	if m.BillingSource == "free" {
		zero := int64(0)
		cost = &zero
	}
	if m.InputMicroUSDPerM != 0 || m.OutputMicroUSDPerM != 0 || m.CachedMicroUSDPerM != 0 {
		if v, err := CostMicroUSD(m, u.InTokens, u.OutTokens, u.CachedTokens); err == nil {
			cost = &v
		}
	}
	trace.Usage(ctx, u, cost)
}

type inspectionStream struct {
	next     ports.Stream
	ctx      context.Context
	span     *trace.Span
	model    Model
	started  time.Time
	mu       sync.Mutex
	err      error
	terminal bool
}

func (s *inspectionStream) Recv() (ports.Chunk, error) {
	chunk, err := s.next.Recv()
	s.mu.Lock()
	defer s.mu.Unlock()
	trace.Capture(s.ctx, "chunk", "stream_chunk", "provider", map[string]any{"at": time.Now().UTC(), "chunk": chunk})
	if chunk.Usage != nil {
		recordInspectionUsage(s.ctx, s.model, *chunk.Usage, time.Since(s.started))
	}
	if err != nil {
		if err != io.EOF {
			s.err = err
		}
		s.terminal = true
		s.span.End(func() any { return map[string]any{"eof": err == io.EOF} }, s.err)
	}
	return chunk, err
}
func (s *inspectionStream) Close() error {
	err := s.next.Close()
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.terminal && s.err == nil {
		s.err = context.Canceled
	}
	if err != nil {
		s.err = err
	}
	s.span.End(func() any { return map[string]any{"closed": true, "fully_consumed": s.terminal} }, s.err)
	return err
}
