package validator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"io"
	"strings"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

const classifierSchema = `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":false,"required":["directive_advice","certainty_about_inference","dependency_encouraging","invalidating"],"properties":{"directive_advice":{"type":"number","minimum":0,"maximum":1},"certainty_about_inference":{"type":"number","minimum":0,"maximum":1},"dependency_encouraging":{"type":"number","minimum":0,"maximum":1},"invalidating":{"type":"number","minimum":0,"maximum":1}}}`

type classifierMetaKey struct{}

// WithCallMeta propagates authenticated IDs and the pinned Knowledge Bundle;
// no metadata is accepted from user content. Called by the Orchestrator after
// admission and bundle pinning, analogously to WP-06's L1 classifier.
func WithCallMeta(ctx context.Context, meta contract.CallMeta) context.Context {
	return context.WithValue(ctx, classifierMetaKey{}, meta)
}

type LLMClassifier struct {
	client  ports.Client
	prompt  string
	ref     contract.BundleRef
	timeout time.Duration
}

var _ Classifier = (*LLMClassifier)(nil)

// NewLLMClassifier is optional and only uses the reviewed, bundle-pinned
// classifier prompt. No placeholder prompt may reach a provider.
func NewLLMClassifier(ctx context.Context, client ports.Client, source ports.Bundle, timeout time.Duration) (*LLMClassifier, error) {
	if client == nil || source == nil || timeout <= 0 {
		return nil, ErrAssets
	}
	reviewed, ok := source.(interface{ SyntheticOnly() bool })
	if !ok || reviewed.SyntheticOnly() {
		return nil, ErrAssets
	}
	raw, err := source.Artifact(ctx, "prompts/validator_classifier.system.tmpl")
	if err != nil || len(raw) == 0 || len(raw) > 16384 || strings.Contains(string(raw), "SYNTHETIC PLACEHOLDER") {
		return nil, ErrAssets
	}
	return &LLMClassifier{client: client, prompt: string(raw), ref: source.Reference(), timeout: timeout}, nil
}

func (c *LLMClassifier) Scores(ctx context.Context, text string, recent []contract.MsgView, lang string) (result0 map[string]float32, result1 error) {
	ctx, inspectionSpan := trace.Start(ctx, "validator/classifier.Scores", func() any { return map[string]any{"text": text, "recent": recent, "lang": lang} })
	defer func() {
		inspectionSpan.Complete(func() any { return map[string]any{"result0": result0} }, result1, recover())
	}()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrClassification
	}
	meta, ok := ctx.Value(classifierMetaKey{}).(contract.CallMeta)
	if !ok || !contract.ValidID(meta.UserID) || !contract.ValidID(meta.TurnID) || meta.Bundle.Version != c.ref.Version {
		return nil, ErrClassification
	}
	meta.Bundle = c.ref
	messages := make([]ports.ChatMessage, 0, 5)
	start := len(recent) - 4
	if start < 0 {
		start = 0
	}
	for _, m := range recent[start:] {
		if m.Role == contract.RoleUser || m.Role == contract.RoleAssistant {
			messages = append(messages, ports.ChatMessage{Role: m.Role, Content: m.Text})
		}
	}
	messages = append(messages, ports.ChatMessage{Role: contract.RoleUser, Content: text})
	trace.Capture(ctx, "classifier_template", "prompt_template", "prompts/validator_classifier.system.tmpl", c.prompt)
	resp, err := c.client.Structured(ctx, ports.StructuredRequest{
		Tier: ports.ComponentValidator, System: []ports.Block{{Name: "validator_classifier.system", Text: c.prompt, Cache: true}},
		Messages: messages, Schema: []byte(classifierSchema), SchemaName: "validator.classifier.v1",
		MaxTokens: 128, Temperature: 0, Timeout: c.timeout, Meta: meta,
		IdempotencyKey: meta.TurnID.String() + "/validator_classifier",
	})
	if err != nil {
		return nil, errors.Join(ErrClassification, err)
	}
	// A plain map decode silently accepts duplicate JSON keys. A duplicate
	// score can overwrite a harmful classification; reject it before any
	// threshold decision. WP-04 still applies the structured-output schema.
	dec := json.NewDecoder(bytes.NewReader(resp.JSON))
	begin, err := dec.Token()
	if err != nil || begin != json.Delim('{') {
		return nil, ErrClassification
	}
	scores := make(map[string]float32, len(scoreToRule))
	for dec.More() {
		key, err := dec.Token()
		name, ok := key.(string)
		if err != nil || !ok {
			return nil, ErrClassification
		}
		if _, allowed := scoreToRule[name]; !allowed {
			return nil, ErrClassification
		}
		if _, duplicate := scores[name]; duplicate {
			return nil, ErrClassification
		}
		var score float64
		if err := dec.Decode(&score); err != nil || score < 0 || score > 1 {
			return nil, ErrClassification
		}
		scores[name] = float32(score)
	}
	end, err := dec.Token()
	if err != nil || end != json.Delim('}') {
		return nil, ErrClassification
	}
	if len(scores) != len(scoreToRule) {
		return nil, ErrClassification
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return nil, ErrClassification
	}
	_ = lang // language/locale is constrained by the caller's pinned bundle.
	return scores, nil
}
