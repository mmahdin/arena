package llm

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/telemetry"
)

// AttemptJournal is an optional durable, privacy-minimal provider boundary.
// Begin is persisted BEFORE the network request; Finish is a one-time terminal
// transition. An unfinished row means interrupted/unknown, not zero usage.
// No prompts, responses, error text, user IDs or secrets cross this port.
type AttemptJournal interface {
	BeginAttempt(context.Context, AttemptStart) error
	FinishAttempt(context.Context, AttemptFinish) error
}

type AttemptStart struct {
	ID                                contract.ID
	TurnID                            contract.ID // Optional, never used to infer an owner.
	Kind                              string      // structured, stream, embed
	Tier                              ports.ModelTier
	Provider, Model                   string
	StartedAt                         time.Time
	InputRate, OutputRate, CachedRate int64 // configured micro-USD per million
}

type AttemptFinish struct {
	ID                                contract.ID
	FinishedAt                        time.Time
	Status                            string      // succeeded, failed, incomplete
	Failure                           FailureKind // enumerated, never the provider error body
	Duration                          time.Duration
	UsageKnown                        bool
	InTokens, OutTokens, CachedTokens int
	// Cost is an estimate from configured model rates, not invoice truth.
	CostMicroUSD int64
	CostStatus   string // estimated, unpriced, unknown
}

var ErrTelemetry = errors.New("llm telemetry unavailable")

// meteredProvider observes each physical provider request, including retries,
// schema repairs, embedding calls and streamed calls. It does not alter prompts.
type meteredProvider struct {
	writeTimeout time.Duration
	next         Provider
	journal      AttemptJournal
	now          func() time.Time
}

func (p meteredProvider) begin(ctx context.Context, model Model, tier ports.ModelTier, turn contract.ID, kind string) (AttemptStart, error) {
	provider, name := observability.SafeLabel(model.Provider), observability.SafeLabel(model.Name)
	if provider == "" || name == "" {
		return AttemptStart{}, ErrTelemetry
	}
	a := AttemptStart{ID: contract.NewID(), Kind: kind, Tier: tier, Provider: provider, Model: name,
		StartedAt: p.now().UTC(), InputRate: model.InputMicroUSDPerM, OutputRate: model.OutputMicroUSDPerM,
		CachedRate: model.CachedMicroUSDPerM}
	if contract.ValidID(turn) {
		a.TurnID = turn
	}
	if err := p.journal.BeginAttempt(ctx, a); err != nil {
		return AttemptStart{}, ErrTelemetry
	}
	return a, nil
}

func (p meteredProvider) finish(ctx context.Context, start AttemptStart, model Model, usage contract.LLMUsage, status string, cause error, span *telemetry.Span) error {
	finished := p.now().UTC()
	if finished.Before(start.StartedAt) {
		finished = start.StartedAt
	}
	duration := finished.Sub(start.StartedAt)
	if duration < 0 {
		duration = 0
	}
	e := AttemptFinish{ID: start.ID, FinishedAt: finished, Status: status, Duration: duration, CostStatus: "unknown"}
	if cause != nil {
		failure, _ := classify(cause)
		e.Failure = failure.Kind
	}
	// A zero-value usage structure cannot prove that the provider reported
	// tokens. Unknown provider usage MUST NOT become a zero-dollar invoice.
	if usage.InTokens > 0 || usage.OutTokens > 0 || usage.CachedTokens > 0 {
		if usage.InTokens >= 0 && usage.OutTokens >= 0 && usage.CachedTokens >= 0 && usage.CachedTokens <= usage.InTokens {
			e.UsageKnown, e.InTokens, e.OutTokens, e.CachedTokens = true, usage.InTokens, usage.OutTokens, usage.CachedTokens
			// Any consumed token category without a configured rate makes
			// the TOTAL cost unpriced; do not report a misleading partial sum.
			priced := (usage.InTokens == usage.CachedTokens || model.InputMicroUSDPerM > 0) &&
				(usage.CachedTokens == 0 || model.CachedMicroUSDPerM > 0) &&
				(usage.OutTokens == 0 || model.OutputMicroUSDPerM > 0)
			if priced {
				if cost, err := CostMicroUSD(model, usage.InTokens, usage.OutTokens, usage.CachedTokens); err == nil {
					e.CostMicroUSD, e.CostStatus = cost, "estimated"
				} else {
					e.CostStatus = "unpriced"
				}
			} else {
				e.CostStatus = "unpriced"
			}
		}
	}
	if span != nil {
		// Use the exact durable journal's classification, never provider body.
		span.Set(telemetry.String("theia.llm.status", e.Status),
			telemetry.String("theia.llm.cost_status", e.CostStatus))
		if e.UsageKnown {
			span.Set(telemetry.Int("gen_ai.usage.input_tokens", int64(e.InTokens)),
				telemetry.Int("gen_ai.usage.output_tokens", int64(e.OutTokens)))
		}
		if e.CostStatus == "estimated" {
			span.Set(telemetry.Int("theia.llm.cost_micro_usd", e.CostMicroUSD))
		}
	}
	// A cancelled request context must not suppress the final accounting write.
	persistCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	if p.writeTimeout > 0 {
		cancel()
		persistCtx, cancel = context.WithTimeout(context.WithoutCancel(ctx), p.writeTimeout)
	}
	defer cancel()
	if err := p.journal.FinishAttempt(persistCtx, e); err != nil {
		return ErrTelemetry
	}
	return nil
}

// One span represents one physical provider attempt. The span begins only
// after durable BeginAttempt succeeds; an aborted SQL begin never calls model.
func startAttemptSpan(ctx context.Context, a AttemptStart) (context.Context, *telemetry.Span) {
	op := "chat"
	if a.Kind == "embed" {
		op = "embeddings"
	}
	return telemetry.Start(ctx, "gen_ai.client",
		telemetry.String("theia.llm_attempt_id", a.ID.String()),
		telemetry.String("theia.turn_id", a.TurnID.String()),
		telemetry.String("theia.llm.kind", a.Kind),
		telemetry.String("theia.llm.tier", string(a.Tier)),
		telemetry.String("gen_ai.operation.name", op),
		telemetry.String("gen_ai.provider.name", a.Provider),
		telemetry.String("gen_ai.request.model", a.Model))
}

func (p meteredProvider) Structured(ctx context.Context, m Model, r ports.StructuredRequest, mode Mode) (StructuredResult, error) {
	a, err := p.begin(ctx, m, r.Tier, r.Meta.TurnID, "structured")
	if err != nil {
		return StructuredResult{}, err
	}
	callCtx, span := startAttemptSpan(ctx, a)
	result, err := p.next.Structured(callCtx, m, r, mode)
	status := "succeeded"
	if err != nil {
		status = "failed"
	}
	saveErr := p.finish(callCtx, a, m, result.Usage, status, err, span)
	span.End(errors.Join(err, saveErr))
	if saveErr != nil {
		return StructuredResult{}, saveErr
	}
	return result, err
}

func (p meteredProvider) Embed(ctx context.Context, m Model, r ports.EmbeddingRequest) (EmbeddingResult, error) {
	a, err := p.begin(ctx, m, r.Tier, r.Meta.TurnID, "embed")
	if err != nil {
		return EmbeddingResult{}, err
	}
	callCtx, span := startAttemptSpan(ctx, a)
	result, err := p.next.Embed(callCtx, m, r)
	status := "succeeded"
	if err != nil {
		status = "failed"
	}
	saveErr := p.finish(callCtx, a, m, result.Usage, status, err, span)
	span.End(errors.Join(err, saveErr))
	if saveErr != nil {
		return EmbeddingResult{}, saveErr
	}
	return result, err
}

func (p meteredProvider) Stream(ctx context.Context, m Model, r ports.StreamRequest) (ports.Stream, error) {
	a, err := p.begin(ctx, m, r.Tier, r.Meta.TurnID, "stream")
	if err != nil {
		return nil, err
	}
	callCtx, span := startAttemptSpan(ctx, a)
	stream, err := p.next.Stream(callCtx, m, r)
	if err != nil {
		saveErr := p.finish(callCtx, a, m, contract.LLMUsage{}, "failed", err, span)
		span.End(errors.Join(err, saveErr))
		if saveErr != nil {
			return nil, saveErr
		}
		return nil, err
	}
	if stream == nil {
		saveErr := p.finish(callCtx, a, m, contract.LLMUsage{}, "failed", ErrUnavailable, span)
		span.End(errors.Join(ErrUnavailable, saveErr))
		if saveErr != nil {
			return nil, saveErr
		}
		return nil, ErrUnavailable
	}
	return &meteredStream{next: stream, p: p, start: a, model: m, ctx: callCtx, span: span, done: make(chan struct{})}, nil
}

type meteredStream struct {
	next        ports.Stream
	p           meteredProvider
	start       AttemptStart
	model       Model
	ctx         context.Context
	span        *telemetry.Span
	usage       contract.LLMUsage
	mu          sync.Mutex
	once        sync.Once
	done        chan struct{}
	terminalErr error
}

// Recv is single-consumer; Close may run concurrently with Recv.
func (s *meteredStream) Recv() (ports.Chunk, error) {
	chunk, err := s.next.Recv()
	if chunk.Usage != nil {
		s.mu.Lock()
		s.usage = *chunk.Usage
		s.mu.Unlock()
	}
	if err != nil {
		status := "failed"
		if errors.Is(err, io.EOF) {
			status = "succeeded"
			err = nil
		}
		if status == "succeeded" {
			if saveErr := s.complete(status, nil); saveErr != nil {
				return ports.Chunk{}, saveErr
			}
			return chunk, io.EOF
		}
		if saveErr := s.complete(status, err); saveErr != nil {
			return ports.Chunk{}, saveErr
		}
	}
	return chunk, err
}

func (s *meteredStream) Close() error {
	err := s.next.Close()
	if saveErr := s.complete("incomplete", err); saveErr != nil {
		return saveErr
	}
	return err
}

func (s *meteredStream) complete(status string, err error) error {
	s.once.Do(func() {
		s.mu.Lock()
		usage := s.usage
		s.mu.Unlock()
		savedErr := s.p.finish(s.ctx, s.start, s.model, usage, status, err, s.span)
		s.span.End(errors.Join(err, savedErr))
		s.terminalErr = savedErr
		close(s.done)
	})
	<-s.done
	return s.terminalErr
}
