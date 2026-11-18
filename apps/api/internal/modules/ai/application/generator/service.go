package generator

import (
	"context"
	"errors"
	"io"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// SourceVerifier is consumer-owned. The MSSQL implementation reuses WP-10's
// ownership/evidence checks and WP-09's explicit AI-use policy. It is mandatory
// in production, before any private content is sent to WP-04.
type SourceVerifier interface {
	VerifyGeneration(context.Context, c.ID, ports.GenInput) error
}
type Options struct{ Verifier SourceVerifier }
type Service struct {
	client   ports.Client
	assets   *Assets
	verifier SourceVerifier
}

var _ ports.Generator = (*Service)(nil)

func New(client ports.Client, a *Assets, opt Options) (*Service, error) {
	if client == nil || a == nil || opt.Verifier == nil && !a.synthetic {
		return nil, ErrAssets
	}
	return &Service{client, a, opt.Verifier}, nil
}
func (*Service) WP11()                    {}
func (s *Service) Reference() c.BundleRef { return s.assets.Reference() }
func (s *Service) GenerationTimeout() time.Duration {
	return time.Duration(s.assets.budgets.Turn.GenTotalMS) * time.Millisecond
}
func (s *Service) MaxRegenerations() int { return s.assets.budgets.Limits.MaxRegenerations }
func (s *Service) MaxSyncCalls() int     { return s.assets.budgets.Limits.MaxSyncLLMCalls }
func (s *Service) TurnTimeout() time.Duration {
	return time.Duration(s.assets.budgets.Turn.TotalMS) * time.Millisecond
}

// OutagePlan implements FB-006 using the bundle's existing generic ("other")
// crisis family. High here is the Planner's template-selection input, never a
// new user risk assessment or a StateDelta. Missing approved content fails
// closed; no generic clinical script or contact is synthesized in code.
func (s *Service) OutagePlan(ctx context.Context, locale string) (c.CrisisPlan, error) {
	return s.assets.planner.Plan(ctx, c.SafetySignal{Level: c.RiskHigh, Categories: []c.RiskCategory{c.RiskOther}}, locale, c.RiskState{Level: c.RiskHigh})
}

type metaKey struct{}

// WithCallMeta is for the authenticated, admitted Orchestrator only. GenInput
// has deliberately no caller-supplied user id. A different bundle/turn is fatal.
func WithCallMeta(ctx context.Context, meta c.CallMeta) context.Context {
	ref := c.BundleRef{Version: meta.Bundle.Version, PromptSHAs: map[string]string{}}
	for k, v := range meta.Bundle.PromptSHAs {
		ref.PromptSHAs[k] = v
	}
	meta.Bundle = ref
	return context.WithValue(ctx, metaKey{}, meta)
}

var headingRE = regexp.MustCompile(`(?m)^\s{0,3}#{1,6}\s`)

// Generate performs ONE attempt into an INTERNAL sink. WP-04 owns pre-token
// provider failover; the Orchestrator owns one regeneration, WP-07 Finalize,
// reviewed fallback validation and atomic delivery. A sink is never HTTP.
// Elevated/crisis drafts remain local until the provider has finished cleanly;
// the Orchestrator additionally buffers all routes until validation succeeds.
func (s *Service) Generate(ctx context.Context, in ports.GenInput, sink ports.TokenSink) (out ports.GenOutput, err error) {
	if err = ctx.Err(); err != nil {
		return
	}
	meta, ok := ctx.Value(metaKey{}).(c.CallMeta)
	if !ok || !c.ValidID(meta.UserID) || meta.TurnID != in.Turn.TurnID || !c.ValidID(meta.TurnID) || !meta.Route.Valid() {
		return out, &c.Error{Code: c.ErrForbidden, Step: "generator_identity"}
	}
	if !reflect.DeepEqual(meta.Bundle, s.assets.Reference()) || !reflect.DeepEqual(in.Bundle, meta.Bundle) {
		return out, &c.Error{Code: c.ErrBundle, Step: "generator_bundle"}
	}
	p, err := s.assets.render(ctx, in, meta.Route)
	if err != nil {
		return out, err
	}
	if s.verifier != nil {
		if err = s.verifier.VerifyGeneration(ctx, meta.UserID, in); err != nil {
			return out, &c.Error{Code: c.ErrForbidden, Step: "generator_sources"}
		}
	}
	maxOut := s.assets.budgets.Tokens.GenOut
	first := time.Duration(s.assets.budgets.Turn.GenFirstTokenMS) * time.Millisecond
	if b, ok := c.BudgetFromContext(ctx); ok {
		if b.MaxTokensOut > 0 && b.MaxTokensOut < maxOut {
			maxOut = b.MaxTokensOut
		}
		if b.GenFirstToken > 0 && b.GenFirstToken < first {
			first = b.GenFirstToken
		}
	}
	stream, err := s.client.Stream(ctx, ports.StreamRequest{Tier: ports.TierStrong, System: p.system, Messages: p.messages, MaxTokens: maxOut, Temperature: .7, Timeout: s.GenerationTimeout(), FirstToken: first, Meta: meta})
	if err != nil {
		var ce *c.Error
		if errors.As(err, &ce) && ce.Code == c.ErrBudget {
			return out, &c.Error{Code: c.ErrBudget, Step: "generator_stream", Degrades: true}
		}
		return out, &c.Error{Code: c.ErrProvider, Step: "generator_stream", Degrades: true}
	}
	if stream == nil {
		return out, &c.Error{Code: c.ErrProvider, Step: "generator_stream", Degrades: true}
	}
	defer func() { _ = stream.Close() }()
	fail := func() (ports.GenOutput, error) {
		out.Text = ""
		out.Aborted = true
		return out, &c.Error{Code: c.ErrGenFailed, Step: "generator_stream", Degrades: true}
	}
	var draft strings.Builder
	buffered := in.Risk.Rank() >= c.RiskElevated.Rank()
	forwarded := 0
	finished, billed := false, false
	finishReason := ""
	for {
		chunk, e := stream.Recv()
		if chunk.Usage != nil {
			if billed {
				return fail()
			}
			billed = true
			out.Usage = *chunk.Usage
		}
		if e == io.EOF {
			break
		}
		if ctx.Err() != nil {
			out.Aborted = true
			return out, ctx.Err()
		}
		if e != nil && draft.Len() == 0 {
			out.Aborted = true
			return out, &c.Error{Code: c.ErrProvider, Step: "generator_stream", Degrades: true}
		}
		if e != nil {
			return fail()
		}
		if !utf8.ValidString(chunk.Text) || draft.Len()+len(chunk.Text) > 80000 || finished && chunk.Text != "" {
			return fail()
		}
		draft.WriteString(chunk.Text)
		if chunk.FinishReason != "" {
			if finished || chunk.FinishReason != "stop" && chunk.FinishReason != "end_turn" {
				return fail()
			}
			finished = true
			finishReason = chunk.FinishReason
		}
		text := strings.TrimSpace(draft.String())
		// Do not leak even a partial sentinel to a sink. Once the exact marker
		// arrives, abort promptly; extra surrounding prose also fails closed.
		if strings.Contains(text, "<<PLAN_UNSAFE>>") {
			out.Sentinel = "PLAN_UNSAFE"
			out.Aborted = true
			return out, nil
		}
		pending := text == "" || strings.HasPrefix("<<PLAN_UNSAFE>>", text)
		if !buffered && !pending && sink != nil && draft.Len() > forwarded {
			if e := sink.Send(draft.String()[forwarded:]); e != nil {
				out.Aborted = true
				return out, &c.Error{Code: c.ErrValidation, Step: "generator_guard", Degrades: true}
			}
			forwarded = draft.Len()
		}
	}
	text := strings.TrimSpace(draft.String())
	if ctx.Err() != nil {
		return out, ctx.Err()
	}
	if !finished || !billed || text == "" || len(utf16.Encode([]rune(text))) > 20000 {
		return fail()
	}
	if headingRE.MatchString(text) || strings.HasPrefix(text, "{") || strings.HasPrefix(text, "[") || strings.HasPrefix(text, "```") {
		out.Aborted = true
		return out, &c.Error{Code: c.ErrValidation, Step: "generator_format", Degrades: true}
	}
	if sink != nil && draft.Len() > forwarded {
		if err = sink.Send(draft.String()[forwarded:]); err != nil {
			out.Aborted = true
			return out, &c.Error{Code: c.ErrValidation, Step: "generator_guard", Degrades: true}
		}
	}
	out.Usage.FinishReason = finishReason
	out.Text = text
	return out, nil
}

// Fatal errors must never be turned into a successful fallback turn.
func Fatal(err error) bool {
	var ce *c.Error
	return errors.As(err, &ce) && (ce.Code == c.ErrForbidden || ce.Code == c.ErrBundle || ce.Code == c.ErrStateConflict)
}
